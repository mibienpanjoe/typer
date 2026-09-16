package backend

import (
	"errors"
	"strings"
	"testing"

	"github.com/mibienpanjoe/typer/internal/domain"
)

func TestSendKittyMatchesOnlySnapshotID(t *testing.T) {
	kid := "3"
	to := "unix:/run/user/1000/kitty"
	var calls []string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil, nil
	}
	err := SendKitty(run, domain.Target{KittyID: &kid, ListenOn: &to}, "continue")
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

func TestSendKittyUsesListenOn(t *testing.T) {
	kid := "3"
	to := "unix:/run/user/1000/kitty"
	var calls []string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		return nil, nil
	}
	err := SendKitty(run, domain.Target{KittyID: &kid, ListenOn: &to}, "continue")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(calls[0], "--to unix:/run/user/1000/kitty") {
		t.Fatalf("%s", calls[0])
	}
}

func TestSendKittyRequiresSocketAndID(t *testing.T) {
	kid := "3"
	for _, target := range []domain.Target{{KittyID: &kid}, {WindowID: "46137358"}} {
		err := SendKitty(func(string, ...string) ([]byte, error) {
			t.Fatal("should not run")
			return nil, nil
		}, target, "x")
		if err == nil {
			t.Fatal("want error")
		}
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
		if len(args) > 0 && args[0] == "getactivewindow" {
			return []byte("2748\n"), nil
		}
		return nil, nil
	}
	if err := SendGnome(run, false, domain.Target{WindowID: "0xabc"}, "hello"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 5 {
		t.Fatalf("%v", calls)
	}
	if calls[0] != "windowactivate --sync 0xabc" || calls[1] != "getactivewindow" {
		t.Fatalf("%v", calls)
	}
	joined := strings.Join(calls, " ")
	if strings.Contains(joined, "type --clearmodifiers --window") {
		t.Fatal("GNOME ignores SendEvent typing to its top-level window")
	}
	if !strings.Contains(joined, "type --clearmodifiers -- hello") || !strings.Contains(joined, "key --clearmodifiers Return") {
		t.Fatalf("%v", calls)
	}
}

func TestSendGnomeRefusesWhenActivatedWindowDoesNotMatch(t *testing.T) {
	var calls []string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) > 0 && args[0] == "getactivewindow" {
			return []byte("999\n"), nil
		}
		return nil, nil
	}
	err := SendGnome(run, false, domain.Target{WindowID: "0xabc"}, "hello")
	if err == nil || !strings.Contains(err.Error(), "fenêtre active") {
		t.Fatalf("err=%v", err)
	}
	if strings.Contains(strings.Join(calls, " "), "type") {
		t.Fatalf("must not type after identity mismatch: %v", calls)
	}
}
