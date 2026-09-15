package backend

import (
	"errors"
	"strings"
	"testing"

	"github.com/mibienpanjoe/typer/internal/domain"
)

func TestSendKittyMatchesOnlySnapshotID(t *testing.T) {
	kid := "3"
	var calls []string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil, nil
	}
	err := SendKitty(run, domain.Target{KittyID: &kid}, "continue")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("%v", calls)
	}
	if !strings.Contains(calls[0], "--match id:3") {
		t.Fatalf("%s", calls[0])
	}
	if strings.Contains(calls[0], "id:1") && !strings.Contains(calls[0], "id:3") {
		t.Fatal("wrong id")
	}
}

func TestSendKittyRequiresID(t *testing.T) {
	err := SendKitty(func(string, ...string) ([]byte, error) {
		t.Fatal("should not run")
		return nil, nil
	}, domain.Target{}, "x")
	if err == nil {
		t.Fatal("want error")
	}
}

func TestSendGnomeLockedDoesNotType(t *testing.T) {
	ran := false
	err := SendGnome(func(string, ...string) ([]byte, error) {
		ran = true
		return nil, nil
	}, true, domain.Target{WindowID: "0x1"}, "continue")
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("%v", err)
	}
	if ran {
		t.Fatal("xdotool must not run when locked")
	}
}

func TestSendGnomeTargetsWindow(t *testing.T) {
	var calls []string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		return nil, nil
	}
	if err := SendGnome(run, false, domain.Target{WindowID: "0xabc"}, "hello"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("%v", calls)
	}
	if !strings.Contains(calls[0], "--window 0xabc") || !strings.Contains(calls[1], "--window 0xabc") {
		t.Fatalf("%v", calls)
	}
	joined := strings.Join(calls, " ")
	if strings.Contains(joined, "windowfocus") && !strings.Contains(joined, "--window") {
		t.Fatal("must not type into focused window only")
	}
}
