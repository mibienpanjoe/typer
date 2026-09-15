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
	ExitOK          = 0
	ExitUser        = 1
	ExitFire        = 2
)

type FormResult struct {
	Target  domain.Target
	At      string
	Message string
	Confirm bool
}

type Deps struct {
	Store     *store.Store
	Now       func() time.Time
	Runner    discover.Runner
	Stdout    io.Writer
	Stderr    io.Writer
	IsTTY     func() bool
	Form      func(targets []domain.Target, at, message string) (FormResult, error)
	Receipt   func(job domain.Job, warn string) string
	Exe       string
	Alive     func(int) bool
	Locked    func() bool
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
	if strings.TrimSpace(message) == "" {
		message = domain.DefaultMessage
	}
	msg, err := domain.ValidateMessage(message)
	if err != nil {
		fmt.Fprintln(d.Stderr, err)
		return ExitUser
	}

	targets, kittyErr := discover.All(d.run())
	if len(targets) == 0 {
		fmt.Fprintln(d.Stderr, "aucune fenêtre Kitty ou GNOME Terminal. Ouvrez l'agent dans Kitty ou GNOME Terminal.")
		if kittyErr != nil {
			fmt.Fprintf(d.Stderr, "Kitty: %v\nActivez allow_remote_control (socket-only) dans kitty.conf.\n", kittyErr)
		}
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
		res, err := d.Form(targets, atStr, msg)
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
		if res.Message != "" {
			msg = res.Message
			msg, err = domain.ValidateMessage(msg)
			if err != nil {
				fmt.Fprintln(d.Stderr, err)
				return ExitUser
			}
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
	}
	if target.Emulator == domain.EmulatorKitty && kittyErr != nil {
		fmt.Fprintf(d.Stderr, "Kitty remote control indisponible. Activez allow_remote_control (socket-only) dans kitty.conf.\n")
		return ExitUser
	}

	pending, err := d.Store.ListJobs()
	if err != nil {
		fmt.Fprintln(d.Stderr, err)
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
	if err := schedule.Start(d.run(), job.SystemdUnit, d.Exe, when, id); err != nil {
		_ = d.Store.DeleteJob(id)
		fmt.Fprintln(d.Stderr, err)
		fmt.Fprintln(d.Stderr, "systemd --user est requis (Pop!_OS / GNOME).")
		return ExitUser
	}

	warn := ""
	if backendName == domain.BackendGnome {
		warn = "⚠ GNOME : échouera si l'écran est verrouillé à cette heure."
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
	_ = schedule.Stop(d.run(), job.SystemdUnit)
	if err := d.Store.DeleteJob(id); err != nil {
		fmt.Fprintln(d.Stderr, err)
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
			locked := false
			if d.Locked != nil {
				locked = d.Locked()
			}
			return backend.SendGnome(d.run(), locked, job.Target, job.Message)
		default:
			return fmt.Errorf("backend inconnu %s", job.Backend)
		}
	}
	_, err := fire.Run(fire.Deps{
		Store: d.Store,
		Now:   d.now,
		Alive: d.Alive,
		List: func() ([]domain.Target, error) {
			t, _ := discover.All(d.run())
			return t, nil
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

func SessionLocked(run discover.Runner) bool {
	if run == nil {
		run = discover.DefaultRunner
	}
	out, err := run("loginctl", "show-session", "self", "-p", "LockedHint")
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "LockedHint=yes")
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
