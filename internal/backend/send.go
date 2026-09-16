package backend

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
)

var ErrLocked = errors.New("session verrouillée : impossible d'injecter au clavier")

func SendKitty(run discover.Runner, target domain.Target, message string) error {
	if run == nil {
		run = discover.DefaultRunner
	}
	if target.KittyID != nil && *target.KittyID != "" && target.ListenOn != nil && strings.HasPrefix(*target.ListenOn, "unix:") {
		match := "id:" + *target.KittyID
		args := []string{"@", "--to", *target.ListenOn}
		args = append(args, "send-text", "--match", match, "--", message+"\n")
		if _, err := run("kitty", args...); err != nil {
			return fmt.Errorf("kitty send-text (id %s): %w", *target.KittyID, err)
		}
		return nil
	}
	return fmt.Errorf("cible Kitty sans socket Unix remote-control et kitty id")
}

func SendGnome(run discover.Runner, locked bool, target domain.Target, message string) error {
	if locked {
		return ErrLocked
	}
	if run == nil {
		run = discover.DefaultRunner
	}
	if target.WindowID == "" {
		return fmt.Errorf("cible GNOME sans window id")
	}
	return typeIntoWindow(run, target.WindowID, message)
}

func typeIntoWindow(run discover.Runner, windowID, message string) error {
	if _, err := run("xdotool", "windowactivate", "--sync", windowID); err != nil {
		return fmt.Errorf("xdotool windowactivate %s: %w", windowID, err)
	}
	if err := requireActiveWindow(run, windowID); err != nil {
		return err
	}
	if _, err := run("xdotool", "type", "--clearmodifiers", "--", message); err != nil {
		return fmt.Errorf("xdotool type: %w — installez xdotool (écran déverrouillé)", err)
	}
	if err := requireActiveWindow(run, windowID); err != nil {
		return fmt.Errorf("refus d'envoyer Entrée: %w", err)
	}
	if _, err := run("xdotool", "key", "--clearmodifiers", "Return"); err != nil {
		return fmt.Errorf("xdotool key Return: %w", err)
	}
	return nil
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
