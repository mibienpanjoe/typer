package backend

import (
	"errors"
	"fmt"

	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
)

var ErrLocked = errors.New("session verrouillée : impossible d'injecter au clavier")

func SendKitty(run discover.Runner, target domain.Target, message string) error {
	if run == nil {
		run = discover.DefaultRunner
	}
	if target.KittyID != nil && *target.KittyID != "" {
		match := "id:" + *target.KittyID
		args := []string{"@"}
		if target.ListenOn != nil && *target.ListenOn != "" {
			args = append(args, "--to", *target.ListenOn)
		}
		args = append(args, "send-text", "--match", match, "--", message+"\n")
		if _, err := run("kitty", args...); err != nil {
			return fmt.Errorf("kitty send-text (id %s): %w", *target.KittyID, err)
		}
		return nil
	}
	if target.WindowID != "" {
		return typeIntoWindow(run, target.WindowID, message)
	}
	return fmt.Errorf("cible Kitty sans id de fenêtre")
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
	if _, err := run("xdotool", "type", "--clearmodifiers", "--window", windowID, "--", message); err != nil {
		return fmt.Errorf("xdotool type: %w — installez xdotool (écran déverrouillé)", err)
	}
	if _, err := run("xdotool", "key", "--window", windowID, "Return"); err != nil {
		return fmt.Errorf("xdotool key Return: %w", err)
	}
	return nil
}
