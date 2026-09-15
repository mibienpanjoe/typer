package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mibienpanjoe/typer/internal/domain"
)

func sampleJob(id string) domain.Job {
	kitty := "1"
	cwd := "/tmp/proj"
	return domain.Job{
		ID:          id,
		CreatedAt:   time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC),
		At:          time.Date(2026, 9, 15, 6, 34, 0, 0, time.UTC),
		Message:     "continue",
		Backend:     domain.BackendKitty,
		SystemdUnit: "typer-job-" + id + ".service",
		Target: domain.Target{
			Emulator: domain.EmulatorKitty,
			PID:      42,
			WindowID: "0x123",
			KittyID:  &kitty,
			Title:    "Codex · proj",
			CWD:      &cwd,
		},
	}
}

func TestSaveLoadJobRoundTrip(t *testing.T) {
	s := New(t.TempDir())
	want := sampleJob("abc")
	if err := s.SaveJob(want); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadJob("abc")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.Message != want.Message || got.Backend != want.Backend {
		t.Fatalf("got %+v", got)
	}
	if got.Target.Title != want.Target.Title || got.Target.PID != want.Target.PID {
		t.Fatalf("target %+v", got.Target)
	}
}

func TestSaveJobMode0600(t *testing.T) {
	s := New(t.TempDir())
	if err := s.SaveJob(sampleJob("perm")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(s.jobPath("perm"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm %o", info.Mode().Perm())
	}
}

func TestJobsDirMode0700(t *testing.T) {
	s := New(t.TempDir())
	if err := s.SaveJob(sampleJob("dir")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(s.JobsDir())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("dir perm %o", info.Mode().Perm())
	}
}

func TestAppendLogOmitsMessage(t *testing.T) {
	s := New(t.TempDir())
	err := s.AppendLog(LogEntry{
		ID:        "abc",
		Timestamp: time.Date(2026, 9, 15, 6, 34, 0, 0, time.UTC),
		Result:    ResultSent,
		Backend:   domain.BackendKitty,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(s.LogPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "continue") || strings.Contains(strings.ToLower(string(raw)), `"message"`) {
		t.Fatalf("log leaked message: %s", raw)
	}
	var got LogEntry
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "abc" || got.Result != ResultSent || got.Backend != domain.BackendKitty {
		t.Fatalf("got %+v", got)
	}
}

func TestDeleteJobRemovesFile(t *testing.T) {
	s := New(t.TempDir())
	if err := s.SaveJob(sampleJob("gone")); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteJob("gone"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadJob("gone"); err == nil {
		t.Fatal("want missing")
	}
}

func TestListJobsReturnsPendingOnly(t *testing.T) {
	s := New(t.TempDir())
	if err := s.SaveJob(sampleJob("a")); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJob(sampleJob("b")); err != nil {
		t.Fatal(err)
	}
	jobs, err := s.ListJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("len %d", len(jobs))
	}
}

func TestDefaultRootUsesXDGStateHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
	got := DefaultRoot()
	want := filepath.Join("/tmp/xdg-state", "typer")
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
