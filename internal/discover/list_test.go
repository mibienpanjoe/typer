package discover

import (
	"errors"
	"strings"
	"testing"

	"github.com/mibienpanjoe/typer/internal/domain"
)

func TestListGnomeParsesTerminalWindowsOnly(t *testing.T) {
	const raw = `0x02a00001  0 21501  gnome-terminal-server.Gnome-terminal  pop-os  Codex · afrikoopps
0x02a00002  0 99     firefox.Firefox  pop-os  Mozilla Firefox
0x02a00003  0 300    org.gnome.Nautilus.Nautilus  pop-os  Files
`
	run := func(name string, args ...string) ([]byte, error) {
		if name != "wmctrl" || strings.Join(args, " ") != "-lpx" {
			t.Fatalf("cmd %s %v", name, args)
		}
		return []byte(raw), nil
	}
	got, err := ListGnome(run)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len %d: %+v", len(got), got)
	}
	if got[0].Emulator != domain.EmulatorGnome || got[0].PID != 21501 {
		t.Fatalf("%+v", got[0])
	}
	if got[0].WindowID != "0x02a00001" {
		t.Fatalf("id %s", got[0].WindowID)
	}
	if got[0].Title != "Codex · afrikoopps" {
		t.Fatalf("title %q", got[0].Title)
	}
}

func TestListGnomeWmctrlMissing(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("executable file not found")
	}
	_, err := ListGnome(run)
	if err == nil {
		t.Fatal("want error")
	}
}

func TestLabelNeverPIDAlone(t *testing.T) {
	cwd := "/home/mj/projects/afrikoopps"
	got := Label(domain.Target{
		Emulator: domain.EmulatorKitty,
		PID:      4242,
		Title:    "gpt-5.6-sol medium",
		CWD:      &cwd,
	})
	if strings.Contains(got, "4242") {
		t.Fatalf("pid leaked: %s", got)
	}
	if !strings.Contains(got, "Kitty") || !strings.Contains(got, "afrikoopps") {
		t.Fatalf("got %s", got)
	}
}

func TestLabelTrimsKittyColonPrefix(t *testing.T) {
	got := Label(domain.Target{Emulator: domain.EmulatorKitty, Title: ": hi | afrikopps"})
	if got != "Kitty · hi | afrikopps" {
		t.Fatalf("got %q", got)
	}
}

func TestPickForYesSinglePlausible(t *testing.T) {
	targets := []domain.Target{
		{Emulator: domain.EmulatorGnome, Title: "Terminal"},
		{Emulator: domain.EmulatorKitty, Title: "Codex · proj"},
	}
	got, err := PickForYes(targets)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Codex · proj" {
		t.Fatalf("%+v", got)
	}
}

func TestPickForYesAmbiguous(t *testing.T) {
	_, err := PickForYes([]domain.Target{
		{Title: "Codex a"},
		{Title: "Codex b"},
	})
	if !IsAmbiguous(err) {
		t.Fatalf("got %v", err)
	}
}

func TestPickForYesNone(t *testing.T) {
	_, err := PickForYes(nil)
	if !IsNoTarget(err) {
		t.Fatalf("got %v", err)
	}
}

func TestAllMergesKittyAndGnome(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		switch name {
		case "kitty":
			return []byte(kittyLSFixture), nil
		case "wmctrl":
			return []byte("0x02a00001  0 21501  gnome-terminal-server.Gnome-terminal  pop-os  bash\n"), nil
		default:
			t.Fatalf("unexpected %s", name)
			return nil, nil
		}
	}
	got, kErr := All(run)
	if kErr != nil {
		t.Fatal(kErr)
	}
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
}
