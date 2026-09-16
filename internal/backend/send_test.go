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
	err := SendKitty(run, domain.Target{KittyID: &kid, ListenOn: &to}, "continue", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("%v", calls)
	}
	if !strings.Contains(calls[0], "send-text") || !strings.Contains(calls[0], "--match id:3") {
		t.Fatalf("%s", calls[0])
	}
	if strings.HasSuffix(calls[0], "\n") || strings.Contains(calls[0], "continue\n") {
		t.Fatalf("LF is not Enter: %q", calls[0])
	}
	if !strings.Contains(calls[1], "send-key") || !strings.Contains(calls[1], "Enter") {
		t.Fatalf("missing submit key: %v", calls)
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
	err := SendKitty(run, domain.Target{KittyID: &kid, ListenOn: &to}, "continue", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(calls[0], "--to unix:/run/user/1000/kitty") {
		t.Fatalf("%s", calls[0])
	}
	if len(calls) < 2 || !strings.Contains(calls[1], "send-key") || !strings.Contains(calls[1], "--to unix:/run/user/1000/kitty") {
		t.Fatalf("%v", calls)
	}
}

func TestSendKittyRequiresSocketAndID(t *testing.T) {
	kid := "3"
	for _, target := range []domain.Target{{KittyID: &kid}, {WindowID: "46137358"}} {
		err := SendKitty(func(string, ...string) ([]byte, error) {
			t.Fatal("should not run")
			return nil, nil
		}, target, "x", "")
		if err == nil {
			t.Fatal("want error")
		}
	}
}

func TestSendKittyCanUseCtrlJWhenConfigured(t *testing.T) {
	kid := "3"
	to := "unix:/run/user/1000/kitty"
	var calls []string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		return nil, nil
	}
	if err := SendKitty(run, domain.Target{KittyID: &kid, ListenOn: &to}, "continue", "ctrl+j"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || !strings.Contains(calls[1], "send-key") || !strings.Contains(calls[1], "ctrl+j") {
		t.Fatalf("%v", calls)
	}
}

func TestSendGnomeLockedDoesNotType(t *testing.T) {
	ran := false
	err := SendGnome(func(string, ...string) ([]byte, error) {
		ran = true
		return nil, nil
	}, true, domain.Target{WindowID: "0x1"}, "continue", "")
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
	if err := SendGnome(run, false, domain.Target{WindowID: "0xabc"}, "hello", ""); err != nil {
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

func TestSendGnomeCanUseCtrlJWhenConfigured(t *testing.T) {
	var calls []string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) > 0 && args[0] == "getactivewindow" {
			return []byte("2748\n"), nil
		}
		return nil, nil
	}
	if err := SendGnome(run, false, domain.Target{WindowID: "0xabc"}, "hello", "ctrl+j"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(calls, " "), "key --clearmodifiers ctrl+j") {
		t.Fatalf("%v", calls)
	}
}

func TestSendGnomeSendsReturnEvenIfFocusFlickersAfterType(t *testing.T) {
	var calls []string
	nActive := 0
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) > 0 && args[0] == "getactivewindow" {
			nActive++
			if nActive == 1 {
				return []byte("2748\n"), nil
			}
			return []byte("999\n"), nil
		}
		return nil, nil
	}
	if err := SendGnome(run, false, domain.Target{WindowID: "0xabc"}, "hi", ""); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(calls, " ")
	if !strings.Contains(joined, "type --clearmodifiers -- hi") {
		t.Fatalf("missing type: %v", calls)
	}
	if !strings.Contains(joined, "key --clearmodifiers Return") {
		t.Fatalf("Enter skipped after type: %v", calls)
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
	err := SendGnome(run, false, domain.Target{WindowID: "0xabc"}, "hello", "")
	if err == nil || !strings.Contains(err.Error(), "fenêtre active") {
		t.Fatalf("err=%v", err)
	}
	if strings.Contains(strings.Join(calls, " "), "type") {
		t.Fatalf("must not type after identity mismatch: %v", calls)
	}
}
