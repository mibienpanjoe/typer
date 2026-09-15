package discover

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/mibienpanjoe/typer/internal/domain"
)

var wmctrlLine = regexp.MustCompile(`^(0x[0-9a-fA-F]+)\s+\S+\s+(\d+)\s+(\S+)\s+\S+\s+(.*)$`)

func ListGnome(run Runner) ([]domain.Target, error) {
	if run == nil {
		run = DefaultRunner
	}
	out, err := run("wmctrl", "-lpx")
	if err != nil {
		return nil, fmt.Errorf("GNOME Terminal: wmctrl indisponible (%v). Installez wmctrl, ou utilisez Kitty", err)
	}
	var targets []domain.Target
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := wmctrlLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		class := strings.ToLower(m[3])
		if !strings.Contains(class, "gnome-terminal") {
			continue
		}
		pid, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		title := strings.TrimSpace(m[4])
		targets = append(targets, domain.Target{
			Emulator: domain.EmulatorGnome,
			PID:      pid,
			WindowID: m[1],
			Title:    title,
		})
	}
	return targets, nil
}
