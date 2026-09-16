package discover

import (
	"errors"
	"strings"
	"testing"
)

func TestPrepareGnomeFindsActiveLocalX11Session(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		switch name {
		case "xdotool":
			return []byte("xdotool version 3"), nil
		case "loginctl":
			joined := strings.Join(args, " ")
			if joined == "list-sessions --no-legend --no-pager" {
				return []byte("3 1000 mj seat0 tty2\n8 1001 other seat0 tty3\n"), nil
			}
			if strings.HasPrefix(joined, "show-session 3 ") {
				return []byte("Type=x11\nActive=yes\nRemote=no\n"), nil
			}
		}
		return nil, errors.New("unexpected command")
	}
	getenv := func(key string) string {
		if key == "XDG_SESSION_TYPE" {
			return "x11"
		}
		if key == "DISPLAY" {
			return ":1"
		}
		return ""
	}

	got, err := PrepareGnome(run, getenv, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if got != "3" {
		t.Fatalf("session %q", got)
	}
}

func TestPrepareGnomeRejectsWayland(t *testing.T) {
	called := false
	_, err := PrepareGnome(func(string, ...string) ([]byte, error) {
		called = true
		return nil, nil
	}, func(key string) string {
		if key == "XDG_SESSION_TYPE" {
			return "wayland"
		}
		return ":1"
	}, 1000)
	if err == nil || !strings.Contains(err.Error(), "X11") {
		t.Fatalf("err=%v", err)
	}
	if called {
		t.Fatal("commands must not run on Wayland")
	}
}

func TestPrepareGnomeRejectsMissingXdotool(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("executable file not found")
	}
	_, err := PrepareGnome(run, func(key string) string {
		if key == "XDG_SESSION_TYPE" {
			return "x11"
		}
		return ":1"
	}, 1000)
	if err == nil || !strings.Contains(err.Error(), "xdotool") {
		t.Fatalf("err=%v", err)
	}
}

func TestSessionLockedUsesExplicitSessionID(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		if name != "loginctl" || strings.Join(args, " ") != "show-session 3 -p LockedHint --value" {
			t.Fatalf("command %s %v", name, args)
		}
		return []byte("yes\n"), nil
	}
	locked, err := SessionLocked(run, "3")
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatal("expected locked")
	}
}

func TestSessionLockedFailsClosedOnUnknownState(t *testing.T) {
	_, err := SessionLocked(func(string, ...string) ([]byte, error) {
		return nil, errors.New("no session")
	}, "3")
	if err == nil {
		t.Fatal("unknown lock state must be an error")
	}
}
