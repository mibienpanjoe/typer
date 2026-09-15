package discover

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/mibienpanjoe/typer/internal/domain"
)

var ErrRemoteControl = errors.New("kitty remote control indisponible")

type Runner func(name string, args ...string) ([]byte, error)

func DefaultRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

type kittyOSWindow struct {
	ID                 int64         `json:"id"`
	PlatformWindowID   json.Number   `json:"platform_window_id"`
	Tabs               []kittyTab    `json:"tabs"`
}

type kittyTab struct {
	Windows []kittyWindow `json:"windows"`
}

type kittyWindow struct {
	ID    int64  `json:"id"`
	PID   int    `json:"pid"`
	CWD   string `json:"cwd"`
	Title string `json:"title"`
}

func ListKitty(run Runner) ([]domain.Target, error) {
	if run == nil {
		run = DefaultRunner
	}
	out, err := run("kitty", "@", "ls")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRemoteControl, err)
	}
	var osWindows []kittyOSWindow
	if err := json.Unmarshal(out, &osWindows); err != nil {
		return nil, fmt.Errorf("%w: JSON kitty ls: %v", ErrRemoteControl, err)
	}
	var targets []domain.Target
	for _, ow := range osWindows {
		wid := ow.PlatformWindowID.String()
		if wid == "" {
			wid = strconv.FormatInt(ow.ID, 10)
		}
		for _, tab := range ow.Tabs {
			for _, w := range tab.Windows {
				kid := strconv.FormatInt(w.ID, 10)
				t := domain.Target{
					Emulator: domain.EmulatorKitty,
					PID:      w.PID,
					WindowID: wid,
					KittyID:  &kid,
					Title:    w.Title,
				}
				if w.CWD != "" {
					cwd := w.CWD
					t.CWD = &cwd
				}
				targets = append(targets, t)
			}
		}
	}
	return targets, nil
}
