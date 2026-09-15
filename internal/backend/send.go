package backend

import (
	"errors"
	"fmt"

	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
)

var ErrLocked = errors.New("session verrouillée : GNOME Terminal ne peut pas recevoir de frappe")

func SendKitty(run discover.Runner, target domain.Target, message string) error {
	if run == nil {
		run = discover.DefaultRunner
	}
	if target.KittyID == nil || *target.KittyID == "" {
		return fmt.Errorf("cible Kitty sans id de fenêtre")
	}
	match := "id:" + *target.KittyID
	_, err := run("kitty", "@", "send-text", "--match", match, "--", message+"\n")
	if err != nil {
		return fmt.Errorf("kitty send-text (id %s): %w", *target.KittyID, err)
	}
	return nil
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
	if _, err := run("xdotool", "type", "--clearmodifiers", "--window", target.WindowID, "--", message); err != nil {
		return fmt.Errorf("xdotool type: %w — installez xdotool (X11, écran déverrouillé)", err)
	}
	if _, err := run("xdotool", "key", "--window", target.WindowID, "Return"); err != nil {
		return fmt.Errorf("xdotool key Return: %w", err)
	}
	return nil
}
