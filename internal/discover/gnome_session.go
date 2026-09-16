package discover

import (
	"fmt"
	"strconv"
	"strings"
)

func PrepareGnome(run Runner, getenv func(string) string, uid int) (string, error) {
	if run == nil {
		run = DefaultRunner
	}
	if getenv == nil {
		return "", fmt.Errorf("GNOME Terminal: environnement graphique inconnu")
	}
	if strings.ToLower(getenv("XDG_SESSION_TYPE")) != "x11" {
		return "", fmt.Errorf("GNOME Terminal: session X11 requise (Wayland non pris en charge)")
	}
	if strings.TrimSpace(getenv("DISPLAY")) == "" {
		return "", fmt.Errorf("GNOME Terminal: DISPLAY absent")
	}
	if _, err := run("xdotool", "version"); err != nil {
		return "", fmt.Errorf("GNOME Terminal: xdotool indisponible: %w", err)
	}
	return graphicalSession(run, uid)
}

func graphicalSession(run Runner, uid int) (string, error) {
	out, err := run("loginctl", "list-sessions", "--no-legend", "--no-pager")
	if err != nil {
		return "", fmt.Errorf("GNOME Terminal: sessions loginctl indisponibles: %w", err)
	}
	var matches []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != strconv.Itoa(uid) {
			continue
		}
		id := fields[0]
		properties, err := run(
			"loginctl", "show-session", id,
			"-p", "Type", "-p", "Active", "-p", "Remote", "--no-pager",
		)
		if err != nil {
			continue
		}
		values := parseProperties(properties)
		if values["Type"] == "x11" && values["Active"] == "yes" && values["Remote"] != "yes" {
			matches = append(matches, id)
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("GNOME Terminal: session graphique X11 active introuvable ou ambiguë")
	}
	return matches[0], nil
}

func SessionLocked(run Runner, sessionID string) (bool, error) {
	if run == nil {
		run = DefaultRunner
	}
	if strings.TrimSpace(sessionID) == "" {
		return false, fmt.Errorf("GNOME Terminal: session graphique non enregistrée")
	}
	out, err := run("loginctl", "show-session", sessionID, "-p", "LockedHint", "--value")
	if err != nil {
		return false, fmt.Errorf("GNOME Terminal: état de verrouillage inconnu: %w", err)
	}
	switch strings.TrimSpace(string(out)) {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	default:
		return false, fmt.Errorf("GNOME Terminal: état de verrouillage inconnu")
	}
}

func parseProperties(raw []byte) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			values[key] = value
		}
	}
	return values
}
