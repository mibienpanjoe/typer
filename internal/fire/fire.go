package fire

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/mibienpanjoe/typer/internal/backend"
	"github.com/mibienpanjoe/typer/internal/domain"
	"github.com/mibienpanjoe/typer/internal/store"
)

var (
	ErrMissingTarget = errors.New("cible absente ou titre incompatible")
	ErrBadJob        = errors.New("fichier job illisible, mauvais propriétaire ou permissions")
)

// MatchTitle: égalité, ou l'un est préfixe de l'autre (le titre Codex change souvent de suffixe).
func MatchTitle(snapshot, live string) bool {
	snapshot = stableTitle(snapshot)
	live = stableTitle(live)
	if snapshot == live {
		return true
	}
	if snapshot == "" || live == "" {
		return false
	}
	return strings.HasPrefix(live, snapshot) || strings.HasPrefix(snapshot, live)
}

func stableTitle(title string) string {
	title = strings.TrimSpace(title)
	r, size := utf8.DecodeRuneInString(title)
	if r >= '\u2800' && r <= '\u28ff' {
		return strings.TrimSpace(title[size:])
	}
	return title
}

func Revalidate(snap domain.Target, live []domain.Target, alive, gnomeLocked bool) error {
	if !alive {
		return ErrMissingTarget
	}
	var found *domain.Target
	for i := range live {
		if live[i].Identity() == snap.Identity() {
			found = &live[i]
			break
		}
	}
	if found == nil {
		return ErrMissingTarget
	}
	if !MatchTitle(snap.Title, found.Title) {
		return ErrMissingTarget
	}
	if snap.Emulator == domain.EmulatorGnome && gnomeLocked {
		return backend.ErrLocked
	}
	if snap.Emulator == domain.EmulatorKitty && snap.KittyID == nil && gnomeLocked {
		return backend.ErrLocked
	}
	return nil
}

type Deps struct {
	Store  *store.Store
	Now    func() time.Time
	Alive  func(pid int) bool
	List   func(domain.Target) ([]domain.Target, error)
	Locked func(domain.Target) (bool, error)
	Send   func(job domain.Job) error
	Stop   func(unit string) error
}

func AlivePID(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

func CheckJobFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBadJob, err)
	}
	if info.Mode().Perm() != 0o600 {
		return fmt.Errorf("%w: permissions %o (attendu 0600)", ErrBadJob, info.Mode().Perm())
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	if int(st.Uid) != os.Getuid() {
		return fmt.Errorf("%w: propriétaire", ErrBadJob)
	}
	return nil
}

func Run(d Deps, jobID string) (logResult string, err error) {
	if d.Now == nil {
		d.Now = time.Now
	}
	job, err := d.Store.LoadJob(jobID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBadJob, err)
	}
	if err := CheckJobFile(d.Store.JobPath(jobID)); err != nil {
		return "", err
	}

	finish := func(result string, cause error) (string, error) {
		var errs []error
		if cause != nil {
			errs = append(errs, cause)
		}
		if err := d.Store.AppendLog(store.LogEntry{ID: jobID, Timestamp: d.Now(), Result: result, Backend: job.Backend}); err != nil {
			errs = append(errs, fmt.Errorf("journal: %w", err))
		}
		if err := d.Store.DeleteJob(jobID); err != nil {
			errs = append(errs, fmt.Errorf("suppression du job: %w", err))
		}
		if d.Stop != nil && job.SystemdUnit != "" {
			if err := d.Stop(job.SystemdUnit); err != nil {
				errs = append(errs, fmt.Errorf("nettoyage de l'unité systemd: %w", err))
			}
		}
		return result, errors.Join(errs...)
	}

	live, err := d.List(job.Target)
	if err != nil {
		return finish(store.ResultAbortedBackend, fmt.Errorf("redécouverte de la cible: %w", err))
	}
	alive := true
	if d.Alive != nil {
		alive = d.Alive(job.Target.PID)
	}
	locked := false
	if job.Target.Emulator == domain.EmulatorGnome {
		if d.Locked == nil {
			res := store.ResultAbortedBackend
			err := fmt.Errorf("état de verrouillage GNOME inconnu")
			return finish(res, err)
		}
		locked, err = d.Locked(job.Target)
		if err != nil {
			res := store.ResultAbortedBackend
			return finish(res, err)
		}
	}
	if err := Revalidate(job.Target, live, alive, locked); err != nil {
		res := store.ResultAbortedMissingTarget
		if errors.Is(err, backend.ErrLocked) {
			res = store.ResultAbortedLocked
		}
		return finish(res, err)
	}
	if err := d.Send(job); err != nil {
		res := store.ResultAbortedBackend
		if errors.Is(err, backend.ErrLocked) {
			res = store.ResultAbortedLocked
		}
		return finish(res, err)
	}
	return finish(store.ResultSent, nil)
}
