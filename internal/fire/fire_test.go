package fire

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mibienpanjoe/typer/internal/backend"
	"github.com/mibienpanjoe/typer/internal/domain"
	"github.com/mibienpanjoe/typer/internal/store"
)

func sample(id string) domain.Job {
	kid := "3"
	return domain.Job{
		ID:          id,
		CreatedAt:   time.Now(),
		At:          time.Now().Add(time.Hour),
		Message:     "continue",
		Backend:     domain.BackendKitty,
		SystemdUnit: "typer-job-" + id,
		Target: domain.Target{
			Emulator: domain.EmulatorKitty,
			PID:      4242,
			WindowID: "44040192",
			KittyID:  &kid,
			Title:    "Codex · proj",
		},
	}
}

func TestMatchTitlePrefix(t *testing.T) {
	if !MatchTitle("Codex · proj", "Codex · proj — working") {
		t.Fatal("live suffix")
	}
	if MatchTitle("Codex", "Claude") {
		t.Fatal("unrelated")
	}
}

func TestRevalidateMissingDoesNotNeedSend(t *testing.T) {
	err := Revalidate(sample("x").Target, nil, true, false)
	if !errors.Is(err, ErrMissingTarget) {
		t.Fatalf("%v", err)
	}
}

func TestRunMissingTargetDoesNotSend(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir)
	job := sample("abc")
	if err := s.SaveJob(job); err != nil {
		t.Fatal(err)
	}
	sent := 0
	stopped := 0
	_, err := Run(Deps{
		Store: s,
		Now:   time.Now,
		Alive: func(int) bool { return true },
		List:  func() ([]domain.Target, error) { return nil, nil },
		Send: func(domain.Job) error {
			sent++
			return nil
		},
		Stop: func(string) error {
			stopped++
			return nil
		},
	}, "abc")
	if !errors.Is(err, ErrMissingTarget) {
		t.Fatalf("%v", err)
	}
	if sent != 0 {
		t.Fatalf("send called %d", sent)
	}
	if stopped != 1 {
		t.Fatalf("stop %d", stopped)
	}
	if _, err := s.LoadJob("abc"); err == nil {
		t.Fatal("job should be deleted")
	}
	raw, err := os.ReadFile(s.LogPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), store.ResultAbortedMissingTarget) {
		t.Fatalf("%s", raw)
	}
}

func TestRunSuccessDeletesJob(t *testing.T) {
	s := store.New(t.TempDir())
	job := sample("ok")
	if err := s.SaveJob(job); err != nil {
		t.Fatal(err)
	}
	res, err := Run(Deps{
		Store: s,
		Now:   time.Now,
		Alive: func(int) bool { return true },
		List:  func() ([]domain.Target, error) { return []domain.Target{job.Target}, nil },
		Send:  func(domain.Job) error { return nil },
		Stop:  func(string) error { return nil },
	}, "ok")
	if err != nil {
		t.Fatal(err)
	}
	if res != store.ResultSent {
		t.Fatalf("%s", res)
	}
	if _, err := s.LoadJob("ok"); err == nil {
		t.Fatal("job remains")
	}
}

func TestRunRejectsWorldReadableJob(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir)
	job := sample("perm")
	if err := s.SaveJob(job); err != nil {
		t.Fatal(err)
	}
	path := s.JobPath("perm")
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	sent := 0
	_, err := Run(Deps{
		Store: s,
		Now:   time.Now,
		Send: func(domain.Job) error {
			sent++
			return nil
		},
	}, "perm")
	if !errors.Is(err, ErrBadJob) {
		t.Fatalf("%v", err)
	}
	if sent != 0 {
		t.Fatal("send")
	}
}

func TestRunDeadPID(t *testing.T) {
	s := store.New(t.TempDir())
	job := sample("dead")
	if err := s.SaveJob(job); err != nil {
		t.Fatal(err)
	}
	sent := 0
	_, err := Run(Deps{
		Store: s,
		Now:   time.Now,
		Alive: func(int) bool { return false },
		List:  func() ([]domain.Target, error) { return []domain.Target{job.Target}, nil },
		Send: func(domain.Job) error {
			sent++
			return nil
		},
		Stop: func(string) error { return nil },
	}, "dead")
	if !errors.Is(err, ErrMissingTarget) {
		t.Fatalf("%v", err)
	}
	if sent != 0 {
		t.Fatal("send")
	}
}

func TestRunGnomeLocked(t *testing.T) {
	s := store.New(t.TempDir())
	job := sample("lock")
	job.Backend = domain.BackendGnome
	job.Target.Emulator = domain.EmulatorGnome
	job.Target.KittyID = nil
	job.Target.WindowID = "0xabc"
	if err := s.SaveJob(job); err != nil {
		t.Fatal(err)
	}
	sent := 0
	_, err := Run(Deps{
		Store:  s,
		Now:    time.Now,
		Alive:  func(int) bool { return true },
		Locked: func() bool { return true },
		List:   func() ([]domain.Target, error) { return []domain.Target{job.Target}, nil },
		Send: func(domain.Job) error {
			sent++
			return nil
		},
		Stop: func(string) error { return nil },
	}, "lock")
	if !errors.Is(err, backend.ErrLocked) {
		t.Fatalf("%v", err)
	}
	if sent != 0 {
		t.Fatal("send must not run when locked")
	}
}

