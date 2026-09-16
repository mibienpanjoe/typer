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
	"syscall"

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

	var last error
	var all []domain.Target
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
		all = append(all, targets...)
	}
	if len(all) > 0 {
		return all, nil
	}
	if last != nil {
		return nil, fmt.Errorf("%w: %v", ErrRemoteControl, last)
	}
	return nil, fmt.Errorf("%w: configurez listen_on unix:${XDG_RUNTIME_DIR}/kitty", ErrRemoteControl)
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
	if v := os.Getenv("KITTY_LISTEN_ON"); strings.HasPrefix(v, "unix:") {
		addrs = append(addrs, v)
	}
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = filepath.Join("/run/user", strconv.Itoa(os.Getuid()))
	}
	ents, err := os.ReadDir(runtime)
	if err != nil {
		return addrs
	}
	for _, e := range ents {
		name := e.Name()
		if !strings.Contains(strings.ToLower(name), "kitty") {
			continue
		}
		path := filepath.Join(runtime, name)
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSocket == 0 {
			continue
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || int(st.Uid) != os.Getuid() {
			continue
		}
		addr := "unix:" + path
		if !contains(addrs, addr) {
			addrs = append(addrs, addr)
		}
	}
	return addrs
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
