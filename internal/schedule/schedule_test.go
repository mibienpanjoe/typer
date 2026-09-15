package schedule

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mibienpanjoe/typer/internal/domain"
)

func TestCalendarLocalNaive(t *testing.T) {
	at := time.Date(2026, 9, 15, 6, 34, 0, 0, time.Local)
	got := Calendar(at)
	if got != "2026-09-15 06:34:00" {
		t.Fatalf("got %q", got)
	}
}

func TestStartInvokesSystemdRun(t *testing.T) {
	at := time.Date(2026, 9, 15, 6, 34, 0, 0, time.Local)
	var got []string
	run := func(name string, args ...string) ([]byte, error) {
		got = append([]string{name}, args...)
		return nil, nil
	}
	if err := Start(run, "typer-job-abc", "/usr/bin/typer", at, "abc"); err != nil {
		t.Fatal(err)
	}
	cmd := strings.Join(got, " ")
	if !strings.Contains(cmd, "systemd-run --user") || !strings.Contains(cmd, "--unit=typer-job-abc") {
		t.Fatalf("%s", cmd)
	}
	if !strings.Contains(cmd, "fire abc") {
		t.Fatalf("%s", cmd)
	}
	if !strings.Contains(cmd, "--on-calendar=2026-09-15 06:34:00") {
		t.Fatalf("%s", cmd)
	}
}

func TestStartWrapsSystemdError(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("not found")
	}
	err := Start(run, "u", "/bin/typer", time.Now(), "x")
	if !errors.Is(err, ErrSystemd) {
		t.Fatalf("%v", err)
	}
}

func TestStopStopsTimer(t *testing.T) {
	var got []string
	run := func(name string, args ...string) ([]byte, error) {
		got = append([]string{name}, args...)
		return nil, nil
	}
	if err := Stop(run, "typer-job-abc"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "systemctl --user stop typer-job-abc.timer" {
		t.Fatalf("%v", got)
	}
}

func TestConflictSameIdentity(t *testing.T) {
	kid := "3"
	t1 := domain.Target{Emulator: domain.EmulatorKitty, KittyID: &kid, WindowID: "1"}
	jobs := []domain.Job{{Target: t1}}
	if !Conflict(jobs, t1) {
		t.Fatal("want conflict")
	}
	other := "9"
	if Conflict(jobs, domain.Target{Emulator: domain.EmulatorKitty, KittyID: &other}) {
		t.Fatal("distinct kitty ids")
	}
}
