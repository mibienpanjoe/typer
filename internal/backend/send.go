package backend

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
)

var ErrLocked = errors.New("session verrouillée : impossible d'injecter au clavier")

func SendKitty(run discover.Runner, target domain.Target, message, submitKey string) error {
	if run == nil {
		run = discover.DefaultRunner
	}
	key, err := submitKeyForBackend(submitKey, domain.BackendKitty)
	if err != nil {
		return err
	}
	if target.KittyID != nil && *target.KittyID != "" && target.ListenOn != nil && strings.HasPrefix(*target.ListenOn, "unix:") {
		match := "id:" + *target.KittyID
		prefix := []string{"@", "--to", *target.ListenOn}
		if _, err := run("kitty", append(prefix, "send-text", "--match", match, "--", message)...); err != nil {
			return fmt.Errorf("kitty send-text (id %s): %w", *target.KittyID, err)
		}
		if _, err := run("kitty", append(prefix, "send-key", "--match", match, key)...); err != nil {
			return fmt.Errorf("kitty send-key %s (id %s): %w", key, *target.KittyID, err)
		}
		return nil
	}
	return fmt.Errorf("cible Kitty sans socket Unix remote-control et kitty id")
}

func SendGnome(run discover.Runner, locked bool, target domain.Target, message, submitKey string) error {
	if locked {
		return ErrLocked
	}
	if run == nil {
		run = discover.DefaultRunner
	}
	if target.WindowID == "" {
		return fmt.Errorf("cible GNOME sans window id")
	}
	key, err := submitKeyForBackend(submitKey, domain.BackendGnome)
	if err != nil {
		return err
	}
	return typeIntoWindow(run, target.WindowID, message, key)
}

func typeIntoWindow(run discover.Runner, windowID, message, submitKey string) error {
	if _, err := run("xdotool", "windowactivate", "--sync", windowID); err != nil {
		return fmt.Errorf("xdotool windowactivate %s: %w", windowID, err)
	}
	if err := requireActiveWindow(run, windowID); err != nil {
		return err
	}
	if _, err := run("xdotool", "type", "--clearmodifiers", "--", message); err != nil {
		return fmt.Errorf("xdotool type: %w — installez xdotool (écran déverrouillé)", err)
	}
	time.Sleep(50 * time.Millisecond)
	// Re-focus before sending the configured submission key. Skipping it after
	// typing leaves the message in the composer without starting the agent.
	if _, err := run("xdotool", "windowactivate", "--sync", windowID); err != nil {
		return fmt.Errorf("xdotool windowactivate (avant soumission) %s: %w", windowID, err)
	}
	if _, err := run("xdotool", "key", "--clearmodifiers", submitKey); err != nil {
		return fmt.Errorf("xdotool key %s: %w", submitKey, err)
	}
	return nil
}

func submitKeyForBackend(value, backendName string) (string, error) {
	key, err := domain.NormalizeSubmitKey(value)
	if err != nil {
		return "", err
	}
	switch backendName {
	case domain.BackendKitty:
		if key == "enter" {
			return "Enter", nil
		}
		return "ctrl+j", nil
	case domain.BackendGnome:
		if key == "enter" {
			return "Return", nil
		}
		return "ctrl+j", nil
	default:
		return "", fmt.Errorf("backend inconnu %s", backendName)
	}
}

func requireActiveWindow(run discover.Runner, windowID string) error {
	out, err := run("xdotool", "getactivewindow")
	if err != nil {
		return fmt.Errorf("xdotool getactivewindow: %w", err)
	}
	want, err := strconv.ParseUint(windowID, 0, 64)
	if err != nil {
		return fmt.Errorf("window id invalide %q", windowID)
	}
	got, err := strconv.ParseUint(strings.TrimSpace(string(out)), 0, 64)
	if err != nil || got != want {
		return fmt.Errorf("fenêtre active différente de la cible %s", windowID)
	}
	return nil
}
