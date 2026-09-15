package discover

import (
	"path/filepath"
	"strings"

	"github.com/mibienpanjoe/typer/internal/domain"
)

func Label(t domain.Target) string {
	emu := "Kitty"
	if t.Emulator == domain.EmulatorGnome {
		emu = "GNOME"
	}
	title := cleanTitle(t.Title)
	if title == "" {
		title = t.WindowID
	}
	if t.CWD != nil && *t.CWD != "" {
		return emu + " · " + title + " · " + filepath.Base(*t.CWD)
	}
	return emu + " · " + title
}

func cleanTitle(title string) string {
	title = strings.TrimSpace(title)
	for {
		next := strings.TrimLeft(title, ":：∶")
		next = strings.TrimSpace(next)
		if next == title {
			return title
		}
		title = next
	}
}

func IsPlausible(t domain.Target) bool {
	s := strings.ToLower(t.Title)
	if t.CWD != nil {
		s += " " + strings.ToLower(*t.CWD)
	}
	for _, n := range []string{"codex", "claude", "gpt-", "opencode", "aider"} {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func PickForYes(targets []domain.Target) (domain.Target, error) {
	if len(targets) == 0 {
		return domain.Target{}, errNoTarget{}
	}
	var plausible []domain.Target
	for _, t := range targets {
		if IsPlausible(t) {
			plausible = append(plausible, t)
		}
	}
	if len(plausible) == 1 {
		return plausible[0], nil
	}
	if len(plausible) == 0 && len(targets) == 1 {
		return targets[0], nil
	}
	return domain.Target{}, errAmbiguous{}
}

type errNoTarget struct{}

func (errNoTarget) Error() string {
	return "aucune fenêtre Kitty ou GNOME Terminal. Ouvrez l'agent dans Kitty ou GNOME Terminal"
}

type errAmbiguous struct{}

func (errAmbiguous) Error() string {
	return "plusieurs fenêtres possibles : relancez sans --yes et choisissez la cible"
}

func IsNoTarget(err error) bool {
	_, ok := err.(errNoTarget)
	return ok
}

func IsAmbiguous(err error) bool {
	_, ok := err.(errAmbiguous)
	return ok
}

func All(run Runner) (targets []domain.Target, kittyErr error) {
	if run == nil {
		run = DefaultRunner
	}
	kitty, kErr := ListKitty(run)
	if kErr != nil {
		kittyErr = kErr
	} else {
		targets = append(targets, kitty...)
	}
	gnome, gErr := ListGnome(run)
	if gErr == nil {
		targets = append(targets, gnome...)
	}
	return targets, kittyErr
}
