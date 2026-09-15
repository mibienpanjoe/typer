package discover

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mibienpanjoe/typer/internal/domain"
)

var ErrRemoteControl = errors.New("kitty remote control indisponible")

type Runner func(name string, args ...string) ([]byte, error)

func DefaultRunner(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return out, fmt.Errorf("%v: %s", err, msg)
		}
		return out, err
	}
	return out, nil
}

type kittyOSWindow struct {
	ID               int64       `json:"id"`
	PlatformWindowID json.Number `json:"platform_window_id"`
	Tabs             []kittyTab  `json:"tabs"`
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

	if out, err := run("kitty", "@", "ls"); err == nil {
		targets, perr := parseKittyLS(out, "")
		if perr == nil && len(targets) > 0 {
			return targets, nil
		}
		if perr != nil {
			return nil, perr
		}
	}

	var last error
	for _, to := range kittyListenAddrs() {
		out, err := run("kitty", "@", "--to", to, "ls")
		if err != nil {
			last = err
			continue
		}
		targets, err := parseKittyLS(out, to)
		if err != nil {
			last = err
			continue
		}
		if len(targets) > 0 {
			return targets, nil
		}
	}

	targets, err := listKittyX11(run)
	if err == nil && len(targets) > 0 {
		return targets, nil
	}
	if last != nil {
		return nil, fmt.Errorf("%w: %v", ErrRemoteControl, last)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRemoteControl, err)
	}
	return nil, fmt.Errorf("%w: aucune fenêtre Kitty (remote control + xdotool)", ErrRemoteControl)
}

func parseKittyLS(out []byte, listenOn string) ([]domain.Target, error) {
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
				if listenOn != "" {
					lo := listenOn
					t.ListenOn = &lo
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

func kittyListenAddrs() []string {
	var addrs []string
	if v := os.Getenv("KITTY_LISTEN_ON"); v != "" {
		addrs = append(addrs, v)
	}
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = filepath.Join("/run/user", strconv.Itoa(os.Getuid()))
	}
	for _, dir := range []string{runtime, os.TempDir()} {
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			name := e.Name()
			if !strings.Contains(strings.ToLower(name), "kitty") {
				continue
			}
			path := filepath.Join(dir, name)
			addrs = append(addrs, "unix:"+path)
		}
	}
	return addrs
}

func listKittyX11(run Runner) ([]domain.Target, error) {
	out, err := run("xdotool", "search", "--class", "kitty")
	if err != nil {
		return nil, err
	}
	var targets []domain.Target
	for _, id := range strings.Fields(string(out)) {
		if id == "" {
			continue
		}
		t := domain.Target{
			Emulator: domain.EmulatorKitty,
			WindowID: id,
		}
		if name, err := run("xdotool", "getwindowname", id); err == nil {
			t.Title = strings.TrimSpace(string(name))
		}
		if pidb, err := run("xdotool", "getwindowpid", id); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(pidb))); err == nil {
				t.PID = pid
			}
		}
		targets = append(targets, t)
	}
	return targets, nil
}
