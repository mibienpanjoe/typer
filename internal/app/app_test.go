package app

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
	"github.com/mibienpanjoe/typer/internal/store"
)

func desktopRun(t *testing.T) discover.Runner {
	t.Helper()
	t.Setenv("KITTY_LISTEN_ON", "unix:/tmp/typer-test-kitty")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	return func(name string, args ...string) ([]byte, error) {
		switch name {
		case "kitty":
			if len(args) >= 2 && args[0] == "@" && args[len(args)-1] == "ls" {
				return []byte(`[
  {"id":1,"platform_window_id":1,"tabs":[{"windows":[
    {"id":3,"pid":4242,"cwd":"/tmp/proj","title":"Codex · proj"}
  ]}]}
]`), nil
			}
			return nil, nil
		case "wmctrl":
			return []byte(""), nil
		case "systemd-run", "systemctl":
			return nil, nil
		default:
			return nil, nil
		}
	}
}

func TestScheduleYesOneTarget(t *testing.T) {
	var out, errb bytes.Buffer
	s := store.New(t.TempDir())
	code := Schedule(Deps{
		Store:  s,
		Now:    func() time.Time { return time.Date(2026, 9, 15, 3, 0, 0, 0, time.Local) },
		Runner: desktopRun(t),
		Stdout: &out,
		Stderr: &errb,
		IsTTY:  func() bool { return false },
		Exe:    "/usr/bin/typer",
	}, "06:34", "continue", true)
	if code != ExitOK {
		t.Fatalf("code %d err=%s", code, errb.String())
	}
	jobs, err := s.ListJobs()
	if err != nil || len(jobs) != 1 {
		t.Fatalf("jobs %v %v", jobs, err)
	}
	if !strings.Contains(out.String(), "typer cancel") {
		t.Fatalf("receipt %s", out.String())
	}
}

func TestScheduleConflict(t *testing.T) {
	s := store.New(t.TempDir())
	d := Deps{
		Store:  s,
		Now:    func() time.Time { return time.Date(2026, 9, 15, 3, 0, 0, 0, time.Local) },
		Runner: desktopRun(t),
		Stdout: ioDiscard{},
		Stderr: ioDiscard{},
		IsTTY:  func() bool { return false },
		Exe:    "/usr/bin/typer",
	}
	if Schedule(d, "06:34", "continue", true) != ExitOK {
		t.Fatal("first")
	}
	var errb bytes.Buffer
	d.Stderr = &errb
	if Schedule(d, "06:40", "continue", true) != ExitUser {
		t.Fatal("second should conflict")
	}
	if !strings.Contains(errb.String(), "déjà") {
		t.Fatalf("%s", errb.String())
	}
}

func TestListTruncatesMessage(t *testing.T) {
	s := store.New(t.TempDir())
	long := strings.Repeat("x", 80)
	if err := s.SaveJob(domain.Job{
		ID: "abc12345", At: time.Date(2026, 9, 15, 6, 34, 0, 0, time.Local),
		Message: long, Backend: domain.BackendKitty,
		Target: domain.Target{Emulator: domain.EmulatorKitty, Title: "Codex"},
	}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if List(Deps{Store: s, Stdout: &out, Stderr: os.Stderr}) != ExitOK {
		t.Fatal()
	}
	if strings.Contains(out.String(), long) {
		t.Fatal("full message dumped")
	}
	if !strings.Contains(out.String(), "…") {
		t.Fatalf("%s", out.String())
	}
}

func TestCancelRemovesJob(t *testing.T) {
	s := store.New(t.TempDir())
	d := Deps{
		Store: s, Runner: desktopRun(t),
		Now:    func() time.Time { return time.Date(2026, 9, 15, 3, 0, 0, 0, time.Local) },
		Stdout: ioDiscard{}, Stderr: ioDiscard{},
		IsTTY: func() bool { return false }, Exe: "/bin/typer",
	}
	if Schedule(d, "06:34", "continue", true) != ExitOK {
		t.Fatal("schedule")
	}
	jobs, _ := s.ListJobs()
	var out bytes.Buffer
	d.Stdout = &out
	if Cancel(d, jobs[0].ID) != ExitOK {
		t.Fatal("cancel")
	}
	left, _ := s.ListJobs()
	if len(left) != 0 {
		t.Fatalf("%v", left)
	}
}

func TestScheduleInvalidHour(t *testing.T) {
	var errb bytes.Buffer
	s := store.New(t.TempDir())
	code := Schedule(Deps{
		Store:  s,
		Now:    func() time.Time { return time.Date(2026, 9, 15, 3, 0, 0, 0, time.Local) },
		Runner: desktopRun(t),
		Stdout: ioDiscard{},
		Stderr: &errb,
		IsTTY:  func() bool { return false },
		Exe:    "/usr/bin/typer",
	}, "25:00", "continue", true)
	if code != ExitUser {
		t.Fatalf("code %d %s", code, errb.String())
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
