package app

import (
	"bytes"
	"errors"
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
	if jobs[0].SubmitKey != domain.DefaultSubmitKey {
		t.Fatalf("submit key %q", jobs[0].SubmitKey)
	}
}

func TestScheduleSnapshotsSubmitKeyFromEnv(t *testing.T) {
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
		Getenv: func(key string) string {
			if key == "TYPER_SUBMIT_KEY" {
				return "enter"
			}
			return ""
		},
	}, "06:34", "continue", true)
	if code != ExitOK {
		t.Fatalf("code %d err=%s", code, errb.String())
	}
	jobs, err := s.ListJobs()
	if err != nil || len(jobs) != 1 {
		t.Fatalf("jobs %v %v", jobs, err)
	}
	if jobs[0].SubmitKey != "enter" {
		t.Fatalf("submit key %q", jobs[0].SubmitKey)
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

func TestCancelReportsSystemdStopFailureButDeletesJob(t *testing.T) {
	s := store.New(t.TempDir())
	job := domain.Job{ID: "abc", SystemdUnit: "typer-job-abc"}
	if err := s.SaveJob(job); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := Cancel(Deps{
		Store: s,
		Runner: func(string, ...string) ([]byte, error) {
			return nil, errors.New("systemd unavailable")
		},
		Stdout: ioDiscard{},
		Stderr: &stderr,
	}, job.ID)
	if code != ExitUser || !strings.Contains(stderr.String(), "unité systemd") {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if _, err := s.LoadJob(job.ID); err == nil {
		t.Fatal("job must be removed so a stale timer cannot inject")
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

func TestScheduleGnomeSnapshotsGraphicalSession(t *testing.T) {
	t.Setenv("KITTY_LISTEN_ON", "unix:/tmp/typer-test-kitty")
	s := store.New(t.TempDir())
	var systemdArgs string
	run := func(name string, args ...string) ([]byte, error) {
		switch name {
		case "kitty":
			return nil, errors.New("remote unavailable")
		case "wmctrl":
			return []byte("0x02a00001  0 21501  gnome-terminal-server.Gnome-terminal  pop-os  Codex · typer\n"), nil
		case "systemd-run":
			systemdArgs = strings.Join(args, " ")
			return nil, nil
		default:
			return nil, errors.New("unexpected command")
		}
	}
	code := Schedule(Deps{
		Store: s,
		Now: func() time.Time {
			return time.Date(2026, 9, 15, 3, 0, 0, 0, time.Local)
		},
		Runner: run, Stdout: ioDiscard{}, Stderr: ioDiscard{},
		IsTTY: func() bool { return false }, Exe: "/usr/bin/typer",
		Getenv: func(key string) string {
			return map[string]string{"DISPLAY": ":1", "XAUTHORITY": "/tmp/xauth", "XDG_SESSION_TYPE": "x11", "PATH": "/home/mj/.local/bin:/usr/bin"}[key]
		},
		PrepareGnome: func() (string, error) { return "3", nil },
	}, "06:34", "continue", true)
	if code != ExitOK {
		t.Fatalf("code %d", code)
	}
	jobs, err := s.ListJobs()
	if err != nil || len(jobs) != 1 {
		t.Fatalf("jobs=%v err=%v", jobs, err)
	}
	if jobs[0].Target.SessionID == nil || *jobs[0].Target.SessionID != "3" {
		t.Fatalf("session %+v", jobs[0].Target.SessionID)
	}
	for _, want := range []string{"--setenv=TYPER_STATE=" + s.Root, "--setenv=DISPLAY=:1", "--setenv=XAUTHORITY=/tmp/xauth", "--setenv=XDG_SESSION_TYPE=x11", "--setenv=PATH=/home/mj/.local/bin:/usr/bin"} {
		if !strings.Contains(systemdArgs, want) {
			t.Fatalf("systemd args missing %q: %s", want, systemdArgs)
		}
	}
}

func TestScheduleGnomeRefusesUnavailableBackend(t *testing.T) {
	t.Setenv("KITTY_LISTEN_ON", "unix:/tmp/typer-test-kitty")
	s := store.New(t.TempDir())
	run := func(name string, args ...string) ([]byte, error) {
		if name == "wmctrl" {
			return []byte("0x02a00001  0 21501  gnome-terminal-server.Gnome-terminal  pop-os  Codex · typer\n"), nil
		}
		return nil, errors.New("unavailable")
	}
	var stderr bytes.Buffer
	code := Schedule(Deps{
		Store: s, Runner: run, Stdout: ioDiscard{}, Stderr: &stderr,
		IsTTY: func() bool { return false }, Exe: "/usr/bin/typer",
		PrepareGnome: func() (string, error) { return "", errors.New("xdotool indisponible") },
	}, "06:34", "continue", true)
	if code != ExitUser || !strings.Contains(stderr.String(), "xdotool") {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	jobs, err := s.ListJobs()
	if err != nil || len(jobs) != 0 {
		t.Fatalf("jobs=%v err=%v", jobs, err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
