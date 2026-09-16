package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mibienpanjoe/typer/internal/backend"
	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
	"github.com/mibienpanjoe/typer/internal/fire"
	"github.com/mibienpanjoe/typer/internal/schedule"
	"github.com/mibienpanjoe/typer/internal/store"
)

const (
	ExitOK   = 0
	ExitUser = 1
	ExitFire = 2
)

type FormResult struct {
	Target  domain.Target
	At      string
	Message string
	Confirm bool
}

type Deps struct {
	Store        *store.Store
	Now          func() time.Time
	Runner       discover.Runner
	Stdout       io.Writer
	Stderr       io.Writer
	IsTTY        func() bool
	Form         func(targets []domain.Target, pending []domain.Job, at, message string) (FormResult, error)
	Receipt      func(job domain.Job, warn string) string
	JobsView     func(jobs []domain.Job) string
	Exe          string
	Alive        func(int) bool
	Getenv       func(string) string
	PrepareGnome func() (string, error)
	Locked       func(domain.Target) (bool, error)
}

func (d Deps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d Deps) run() discover.Runner {
	if d.Runner != nil {
		return d.Runner
	}
	return discover.DefaultRunner
}

func Schedule(d Deps, atFlag, message string, yes bool) int {
	msg, err := domain.ValidateMessage(message)
	if err != nil {
		fmt.Fprintln(d.Stderr, err)
		return ExitUser
	}

	targets, kittyErr, gnomeErr := discover.All(d.run())
	if len(targets) == 0 {
		fmt.Fprintln(d.Stderr, "aucune fenêtre Kitty ou GNOME Terminal.")
		if kittyErr != nil {
			fmt.Fprintf(d.Stderr, "Kitty: %v\n", kittyErr)
			fmt.Fprintln(d.Stderr, "Dans kitty.conf, puis redémarrez Kitty :")
			fmt.Fprintln(d.Stderr, "  allow_remote_control socket-only")
			fmt.Fprintln(d.Stderr, "  listen_on unix:${XDG_RUNTIME_DIR}/kitty")
		}
		if gnomeErr != nil {
			fmt.Fprintf(d.Stderr, "GNOME: %v\n", gnomeErr)
			fmt.Fprintln(d.Stderr, "Installez wmctrl et xdotool, puis utilisez une session X11.")
		}
		return ExitUser
	}
	pending, err := d.Store.ListJobs()
	if err != nil {
		fmt.Fprintln(d.Stderr, err)
		return ExitUser
	}

	var (
		target domain.Target
		atStr  = atFlag
	)

	tty := d.IsTTY != nil && d.IsTTY()
	if yes || !tty {
		if atStr == "" {
			fmt.Fprintln(d.Stderr, "heure requise (--at HH:MM) hors TTY ou avec --yes")
			return ExitUser
		}
		t, err := discover.PickForYes(targets)
		if err != nil {
			fmt.Fprintln(d.Stderr, err)
			return ExitUser
		}
		target = t
	} else {
		if d.Form == nil {
			fmt.Fprintln(d.Stderr, "formulaire indisponible")
			return ExitUser
		}
		res, err := d.Form(targets, pending, atStr, msg)
		if err != nil {
			fmt.Fprintln(d.Stderr, err)
			return ExitUser
		}
		if !res.Confirm {
			fmt.Fprintln(d.Stderr, "annulé")
			return ExitUser
		}
		target = res.Target
		if res.At != "" {
			atStr = res.At
		}
		msg, err = domain.ValidateMessage(res.Message)
		if err != nil {
			fmt.Fprintln(d.Stderr, err)
			return ExitUser
		}
	}

	if atStr == "" {
		fmt.Fprintln(d.Stderr, "heure invalide \"\" (attendu HH:MM)")
		return ExitUser
	}
	when, err := domain.ParseClock(atStr, d.now())
	if err != nil {
		fmt.Fprintln(d.Stderr, err)
		return ExitUser
	}

	backendName := domain.BackendKitty
	if target.Emulator == domain.EmulatorGnome {
		backendName = domain.BackendGnome
		if d.PrepareGnome == nil {
			fmt.Fprintln(d.Stderr, "GNOME Terminal: vérification du backend indisponible")
			return ExitUser
		}
		sessionID, err := d.PrepareGnome()
		if err != nil {
			fmt.Fprintln(d.Stderr, err)
			return ExitUser
		}
		target.SessionID = &sessionID
	}
	if target.Emulator == domain.EmulatorKitty && (target.KittyID == nil || target.ListenOn == nil || !strings.HasPrefix(*target.ListenOn, "unix:")) {
		fmt.Fprintln(d.Stderr, "Kitty remote control indisponible. Configurez allow_remote_control socket-only et listen_on unix:${XDG_RUNTIME_DIR}/kitty.")
		return ExitUser
	}

	if schedule.Conflict(pending, target) {
		fmt.Fprintln(d.Stderr, "un job existe déjà pour cette fenêtre. typer cancel d'abord.")
		return ExitUser
	}

	id := domain.NewID()
	job := domain.Job{
		ID:          id,
		CreatedAt:   d.now(),
		At:          when,
		Message:     msg,
		Backend:     backendName,
		Target:      target,
		SystemdUnit: schedule.UnitName(id),
	}
	if err := d.Store.SaveJob(job); err != nil {
		fmt.Fprintln(d.Stderr, err)
		return ExitUser
	}
	environment := []string{"TYPER_STATE=" + d.Store.Root}
	if backendName == domain.BackendGnome && d.Getenv != nil {
		for _, key := range []string{"DISPLAY", "XAUTHORITY", "XDG_SESSION_TYPE", "PATH"} {
			if value := d.Getenv(key); value != "" {
				environment = append(environment, key+"="+value)
			}
		}
	}
	if err := schedule.Start(d.run(), job.SystemdUnit, d.Exe, when, id, environment...); err != nil {
		_ = d.Store.DeleteJob(id)
		fmt.Fprintln(d.Stderr, err)
		fmt.Fprintln(d.Stderr, "systemd --user est requis (Pop!_OS / GNOME).")
		return ExitUser
	}

	warn := ""
	if backendName == domain.BackendGnome || job.Target.KittyID == nil {
		warn = "⚠ Écran verrouillé à l'heure H : l'envoi clavier échouera."
	}
	if d.Receipt != nil {
		fmt.Fprint(d.Stdout, d.Receipt(job, warn))
	} else {
		fmt.Fprint(d.Stdout, Receipt(job, warn))
	}
	return ExitOK
}

func Receipt(job domain.Job, warn string) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  Typer · job enregistré\n")
	b.WriteString(fmt.Sprintf("  À %s  envoyer « %s » + Entrée\n", job.At.Format("15:04"), job.Message))
	b.WriteString(fmt.Sprintf("  → %s\n", discover.Label(job.Target)))
	b.WriteString(fmt.Sprintf("  id %s\n", job.ID))
	b.WriteString(fmt.Sprintf("  Annuler : typer cancel %s\n", job.ID))
	if warn != "" {
		b.WriteString("  " + warn + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

func List(d Deps) int {
	jobs, err := d.Store.ListJobs()
	if err != nil {
		fmt.Fprintln(d.Stderr, err)
		return ExitUser
	}
	if len(jobs) == 0 {
		fmt.Fprintln(d.Stdout, "aucun job en attente")
		return ExitOK
	}
	if d.JobsView != nil {
		fmt.Fprintln(d.Stdout, d.JobsView(jobs))
		return ExitOK
	}
	for _, j := range jobs {
		msg := j.Message
		if len(msg) > 40 {
			msg = msg[:40] + "…"
		}
		fmt.Fprintf(d.Stdout, "%s  %s  %s  %s  %s\n",
			j.ID, j.At.Format("15:04"), j.Backend, discover.Label(j.Target), msg)
	}
	return ExitOK
}

func Cancel(d Deps, id string) int {
	jobs, err := d.Store.ListJobs()
	if err != nil {
		fmt.Fprintln(d.Stderr, err)
		return ExitUser
	}
	if id == "" {
		if len(jobs) == 0 {
			fmt.Fprintln(d.Stderr, "aucun job à annuler")
			return ExitUser
		}
		if len(jobs) > 1 {
			fmt.Fprintln(d.Stderr, "plusieurs jobs : typer cancel <id>")
			return ExitUser
		}
		id = jobs[0].ID
	}
	job, err := d.Store.LoadJob(id)
	if err != nil {
		fmt.Fprintln(d.Stderr, "job introuvable")
		return ExitUser
	}
	stopErr := schedule.Stop(d.run(), job.SystemdUnit)
	if err := d.Store.DeleteJob(id); err != nil {
		fmt.Fprintln(d.Stderr, err)
		return ExitUser
	}
	if stopErr != nil {
		fmt.Fprintf(d.Stderr, "job supprimé, mais l'unité systemd n'a pas pu être arrêtée: %v\n", stopErr)
		fmt.Fprintln(d.Stderr, "Le job absent empêchera tout envoi si l'unité se déclenche.")
		return ExitUser
	}
	fmt.Fprintf(d.Stdout, "job %s annulé\n", id)
	return ExitOK
}

func Fire(d Deps, id string) int {
	if id == "" {
		fmt.Fprintln(d.Stderr, "typer fire <id>")
		return ExitUser
	}
	send := func(job domain.Job) error {
		switch job.Backend {
		case domain.BackendKitty:
			return backend.SendKitty(d.run(), job.Target, job.Message)
		case domain.BackendGnome:
			return backend.SendGnome(d.run(), false, job.Target, job.Message)
		default:
			return fmt.Errorf("backend inconnu %s", job.Backend)
		}
	}
	_, err := fire.Run(fire.Deps{
		Store: d.Store,
		Now:   d.now,
		Alive: d.Alive,
		List: func(target domain.Target) ([]domain.Target, error) {
			return discover.ListTarget(d.run(), target)
		},
		Locked: d.Locked,
		Send:   send,
		Stop: func(unit string) error {
			return schedule.Stop(d.run(), unit)
		},
	}, id)
	if err == nil {
		return ExitOK
	}
	fmt.Fprintln(d.Stderr, err)
	if errors.Is(err, fire.ErrMissingTarget) || errors.Is(err, backend.ErrLocked) || errors.Is(err, fire.ErrBadJob) {
		return ExitFire
	}
	return ExitFire
}

func SessionLocked(run discover.Runner, target domain.Target) (bool, error) {
	sessionID := ""
	if target.SessionID != nil {
		sessionID = *target.SessionID
	}
	return discover.SessionLocked(run, sessionID)
}

func DefaultAlive(pid int) bool {
	return fire.AlivePID(pid)
}

func MustExe() string {
	exe, err := os.Executable()
	if err != nil {
		return os.Args[0]
	}
	return exe
}
