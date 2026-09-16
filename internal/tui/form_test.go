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

func TestSummaryShowsTomorrowAndGnomeWarning(t *testing.T) {
	now := time.Date(2026, 9, 15, 20, 0, 0, 0, time.Local)
	target := domain.Target{Emulator: domain.EmulatorGnome, Title: "Codex · typer", WindowID: "0xabc"}
	got := Summary(&target, "06:34", "continue", now)
	for _, want := range []string{"Demain à 06:34", "continue", "GNOME", "écran est verrouillé"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
}

func TestSummaryPromptsForIncompleteValues(t *testing.T) {
	got := Summary(nil, "", "", time.Now())
	if !strings.Contains(got, "Complétez") {
		t.Fatalf("%s", got)
	}
}

func TestInitialChoiceRequiresSelectionWhenAmbiguous(t *testing.T) {
	targets := []domain.Target{
		{Emulator: domain.EmulatorKitty, Title: "Codex a"},
		{Emulator: domain.EmulatorKitty, Title: "Codex b"},
	}
	if got := initialChoice(targets); got != "" {
		t.Fatalf("choice %q", got)
	}
	if got := initialChoice(targets[:1]); got != "0" {
		t.Fatalf("single choice %q", got)
	}
}

func TestTargetLabelsDisambiguateDuplicateWindowTitles(t *testing.T) {
	targets := []domain.Target{
		{Emulator: domain.EmulatorGnome, Title: "mj@host: ~", WindowID: "0x0480000a"},
		{Emulator: domain.EmulatorGnome, Title: "mj@host: ~", WindowID: "0x048125c9"},
	}
	labels := targetLabels(targets)
	if labels[0] == labels[1] {
		t.Fatalf("labels remain ambiguous: %q", labels[0])
	}
	if !strings.Contains(labels[0], "0x0480000a") || !strings.Contains(labels[1], "0x048125c9") {
		t.Fatalf("labels=%v", labels)
	}
}

func TestPanelListsExistingJobs(t *testing.T) {
	pending := []domain.Job{
		{
			ID:      "a",
			At:      time.Date(2026, 9, 15, 7, 10, 0, 0, time.Local),
			Message: "continue",
			Target:  domain.Target{Emulator: domain.EmulatorKitty, Title: "baraka launch video", WindowID: "2"},
		},
		{
			ID:      "b",
			At:      time.Date(2026, 9, 15, 8, 0, 0, 0, time.Local),
			Message: "continue",
			Target:  domain.Target{Emulator: domain.EmulatorKitty, Title: "autre", WindowID: "3"},
		},
	}
	sel := domain.Target{Emulator: domain.EmulatorKitty, Title: "hi | afrikopps", WindowID: "1"}
	got := Panel(&sel, pending, 80)
	if got == "" {
		t.Fatal("expected table")
	}
	if strings.Contains(got, ">") {
		t.Fatalf("draft row leaked:\n%s", got)
	}
	if strings.Contains(got, "06:34") {
		t.Fatalf("form time leaked:\n%s", got)
	}
	if !strings.Contains(got, "07:10") || !strings.Contains(got, "08:00") {
		t.Fatalf("times:\n%s", got)
	}
	if !strings.Contains(got, "baraka") || !strings.Contains(got, "autre") {
		t.Fatalf("labels:\n%s", got)
	}
	for _, clock := range []string{"07:10", "08:00"} {
		hits := 0
		for _, ln := range strings.Split(got, "\n") {
			if strings.Contains(ln, clock) {
				hits++
				if !strings.Contains(ln, "continue") {
					t.Fatalf("%s not on same line as message:\n%s", clock, got)
				}
			}
		}
		if hits != 1 {
			t.Fatalf("%s hits=%d\n%s", clock, hits, got)
		}
	}
}

func TestPanelEmptyWithoutJobs(t *testing.T) {
	if Panel(nil, nil, 80) != "" {
		t.Fatal("empty pending must hide the table")
	}
}

func TestPanelWarnsOnConflict(t *testing.T) {
	t0 := domain.Target{Emulator: domain.EmulatorKitty, Title: "hi", WindowID: "1"}
	got := Panel(
		&t0,
		[]domain.Job{{ID: "a", At: time.Date(2026, 9, 15, 6, 34, 0, 0, time.Local), Target: t0, Message: "continue"}},
		80,
	)
	if !strings.Contains(got, "!") {
		t.Fatalf("%s", got)
	}
}

func TestJobListShowsIDBackendAndTruncatedMessage(t *testing.T) {
	long := strings.Repeat("x", 80)
	got := JobList([]domain.Job{{
		ID:      "abc123",
		At:      time.Date(2026, 9, 16, 6, 34, 0, 0, time.Local),
		Backend: domain.BackendGnome,
		Message: long,
		Target:  domain.Target{Emulator: domain.EmulatorGnome, Title: "Codex · typer"},
	}})
	for _, want := range []string{"abc123", "16/09 06:34", "gnome", "Codex", "…"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, long) {
		t.Fatal("full message should not be displayed")
	}
}
