package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultMessage   = "continue"
	MaxMessageBytes  = 4096
	BackendKitty     = "kitty"
	BackendGnome     = "gnome"
	EmulatorKitty    = "kitty"
	EmulatorGnome    = "gnome-terminal"
)

type Target struct {
	Emulator string  `json:"emulator"`
	PID      int     `json:"pid"`
	WindowID string  `json:"window_id"`
	KittyID  *string `json:"kitty_id"`
	TTY      *string `json:"tty"`
	Title    string  `json:"title"`
	CWD      *string `json:"cwd"`
}

type Job struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	At          time.Time `json:"at"`
	Message     string    `json:"message"`
	Backend     string    `json:"backend"`
	Target      Target    `json:"target"`
	SystemdUnit string    `json:"systemd_unit"`
}

func NewID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func (t Target) Identity() string {
	if t.KittyID != nil && *t.KittyID != "" {
		return "kitty:" + *t.KittyID
	}
	return t.Emulator + ":" + t.WindowID
}

func ParseClock(s string, now time.Time) (time.Time, error) {
	var hour, min int
	n, err := fmt.Sscanf(s, "%d:%d", &hour, &min)
	if err != nil || n != 2 {
		return time.Time{}, fmt.Errorf("heure invalide %q (attendu HH:MM)", s)
	}
	if strings.Count(s, ":") != 1 {
		return time.Time{}, fmt.Errorf("heure invalide %q (attendu HH:MM)", s)
	}
	if hour < 0 || hour > 23 || min < 0 || min > 59 {
		return time.Time{}, fmt.Errorf("heure invalide %q", s)
	}
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, loc)
	if today.After(now) {
		return today, nil
	}
	return today.AddDate(0, 0, 1), nil
}

func ValidateMessage(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("message vide")
	}
	if len(s) > MaxMessageBytes {
		return "", fmt.Errorf("message trop long (%d octets, max %d)", len(s), MaxMessageBytes)
	}
	return s, nil
}
