package discover

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mibienpanjoe/typer/internal/domain"
)

func kittySocket(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kitty.sock")
	t.Setenv("KITTY_LISTEN_ON", "unix:"+path)
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	return "unix:" + path
}

const kittyLSFixture = `[
  {
    "id": 1,
    "platform_window_id": 44040192,
    "tabs": [
      {
        "id": 1,
        "windows": [
          {
            "id": 3,
            "pid": 4242,
            "cwd": "/home/mj/projects/afrikoopps",
            "title": "gpt-5.6-sol medium  ~/projects/afrikoopps  hi"
          }
        ]
      }
    ]
  }
]`

func TestListKittyParsesWindows(t *testing.T) {
	to := kittySocket(t)
	run := func(name string, args ...string) ([]byte, error) {
		if name != "kitty" || strings.Join(args, " ") != "@ --to "+to+" ls" {
			t.Fatalf("cmd %s %v", name, args)
		}
		return []byte(kittyLSFixture), nil
	}
	targets, err := ListKitty(run)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("len %d", len(targets))
	}
	got := targets[0]
	if got.Emulator != domain.EmulatorKitty || got.PID != 4242 {
		t.Fatalf("%+v", got)
	}
	if got.KittyID == nil || *got.KittyID != "3" {
		t.Fatalf("kitty id %+v", got.KittyID)
	}
	if got.ListenOn == nil || *got.ListenOn != to {
		t.Fatalf("listen_on %+v", got.ListenOn)
	}
	if got.WindowID != "44040192" {
		t.Fatalf("window id %s", got.WindowID)
	}
	if got.CWD == nil || *got.CWD != "/home/mj/projects/afrikoopps" {
		t.Fatalf("cwd %+v", got.CWD)
	}
	if got.Title != "gpt-5.6-sol medium  ~/projects/afrikoopps  hi" {
		t.Fatalf("title %q", got.Title)
	}
}

func TestListKittyRemoteControlError(t *testing.T) {
	kittySocket(t)
	run := func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("Connection refused")
	}
	_, err := ListKitty(run)
	if !errors.Is(err, ErrRemoteControl) {
		t.Fatalf("got %v", err)
	}
}

func TestListKittyDoesNotFallBackToXdotool(t *testing.T) {
	kittySocket(t)
	xdotoolCalled := false
	run := func(name string, args ...string) ([]byte, error) {
		if name == "kitty" {
			return nil, errors.New("open /dev/tty: no such device")
		}
		if name == "xdotool" {
			xdotoolCalled = true
		}
		return nil, errors.New("unexpected " + name)
	}
	_, err := ListKitty(run)
	if !errors.Is(err, ErrRemoteControl) {
		t.Fatalf("got %v", err)
	}
	if xdotoolCalled {
		t.Fatal("Kitty discovery must not use keyboard injection")
	}
}
