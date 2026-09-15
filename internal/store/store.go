package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mibienpanjoe/typer/internal/domain"
)

const (
	ResultSent                 = "sent"
	ResultAbortedMissingTarget = "aborted_missing_target"
	ResultAbortedLocked        = "aborted_locked"
	ResultAbortedBackend       = "aborted_backend"
)

type Store struct {
	Root string
}

type LogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Result    string    `json:"result"`
	Backend   string    `json:"backend"`
}

func New(root string) *Store {
	return &Store{Root: root}
}

func DefaultRoot() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "typer")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "typer-state")
	}
	return filepath.Join(home, ".local", "state", "typer")
}

func (s *Store) JobsDir() string {
	return filepath.Join(s.Root, "jobs")
}

func (s *Store) LogPath() string {
	return filepath.Join(s.Root, "log.jsonl")
}

func (s *Store) JobPath(id string) string {
	return s.jobPath(id)
}

func (s *Store) jobPath(id string) string {
	return filepath.Join(s.JobsDir(), id+".json")
}

func (s *Store) SaveJob(job domain.Job) error {
	if err := validID(job.ID); err != nil {
		return err
	}
	if err := os.MkdirAll(s.JobsDir(), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(s.JobsDir(), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	path := s.jobPath(job.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) LoadJob(id string) (domain.Job, error) {
	var job domain.Job
	if err := validID(id); err != nil {
		return job, err
	}
	raw, err := os.ReadFile(s.jobPath(id))
	if err != nil {
		return job, err
	}
	if err := json.Unmarshal(raw, &job); err != nil {
		return job, err
	}
	return job, nil
}

func (s *Store) DeleteJob(id string) error {
	if err := validID(id); err != nil {
		return err
	}
	err := os.Remove(s.jobPath(id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *Store) ListJobs() ([]domain.Job, error) {
	entries, err := os.ReadDir(s.JobsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var jobs []domain.Job
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		job, err := s.LoadJob(id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (s *Store) AppendLog(entry LogEntry) error {
	if err := os.MkdirAll(s.Root, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(s.Root, 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.LogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	return err
}

func validID(id string) error {
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") || strings.Contains(id, string(filepath.Separator)) {
		return fmt.Errorf("job id invalide")
	}
	for _, r := range id {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			return fmt.Errorf("job id invalide")
		}
	}
	return nil
}
