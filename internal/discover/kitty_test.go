package discover

import (
	"errors"
	"strings"
	"testing"

	"github.com/mibienpanjoe/typer/internal/domain"
)

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
	run := func(name string, args ...string) ([]byte, error) {
		if name != "kitty" || strings.Join(args, " ") != "@ ls" {
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
	run := func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("Connection refused")
	}
	_, err := ListKitty(run)
	if !errors.Is(err, ErrRemoteControl) {
		t.Fatalf("got %v", err)
	}
}

func TestListKittyFallsBackToXdotool(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		if name == "kitty" {
			return nil, errors.New("open /dev/tty: no such device")
		}
		if name == "xdotool" && len(args) >= 2 && args[0] == "search" {
			return []byte("48234510\n46137358\n"), nil
		}
		if name == "xdotool" && len(args) >= 2 && args[0] == "getwindowname" && args[1] == "46137358" {
			return []byte("hi | afrikopps\n"), nil
		}
		if name == "xdotool" && len(args) >= 2 && args[0] == "getwindowname" {
			return []byte("other\n"), nil
		}
		if name == "xdotool" && len(args) >= 2 && args[0] == "getwindowpid" {
			return []byte("4242\n"), nil
		}
		return nil, errors.New("unexpected " + name)
	}
	got, err := ListKitty(run)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len %d %+v", len(got), got)
	}
	if got[1].WindowID != "46137358" || got[1].Title != "hi | afrikopps" {
		t.Fatalf("%+v", got[1])
	}
	if got[1].KittyID != nil {
		t.Fatal("x11 fallback must not invent a kitty id")
	}
}
