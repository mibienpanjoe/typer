package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/mibienpanjoe/typer/internal/domain"
)

func TestCardContainsCancel(t *testing.T) {
	got := Card(domain.Job{
		ID:      "abc",
		At:      time.Date(2026, 9, 15, 6, 34, 0, 0, time.Local),
		Message: "continue",
		Target:  domain.Target{Emulator: domain.EmulatorKitty, Title: "Codex"},
	}, "⚠ GNOME : lock")
	if !strings.Contains(got, "typer cancel abc") {
		t.Fatalf("%s", got)
	}
	if !strings.Contains(got, "06:34") || !strings.Contains(got, "⚠") {
		t.Fatalf("%s", got)
	}
}
