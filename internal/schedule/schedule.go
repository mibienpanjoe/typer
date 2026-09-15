package schedule

import (
	"errors"
	"fmt"
	"time"

	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
)

var ErrSystemd = errors.New("systemd --user indisponible")

func UnitName(jobID string) string {
	return "typer-job-" + jobID
}

func Calendar(at time.Time) string {
	return at.Format("2006-01-02 15:04:00")
}

func Start(run discover.Runner, unit, exe string, at time.Time, jobID string) error {
	if run == nil {
		run = discover.DefaultRunner
	}
	_, err := run(
		"systemd-run",
		"--user",
		"--collect",
		"--unit="+unit,
		"--on-calendar="+Calendar(at),
		exe,
		"fire",
		jobID,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSystemd, err)
	}
	return nil
}

func Stop(run discover.Runner, unit string) error {
	if run == nil {
		run = discover.DefaultRunner
	}
	_, err := run("systemctl", "--user", "stop", unit+".timer")
	if err != nil {
		_, err2 := run("systemctl", "--user", "stop", unit+".service")
		if err2 != nil {
			return fmt.Errorf("%w: stop %s: %v / %v", ErrSystemd, unit, err, err2)
		}
	}
	return nil
}

func Conflict(jobs []domain.Job, target domain.Target) bool {
	id := target.Identity()
	for _, j := range jobs {
		if j.Target.Identity() == id {
			return true
		}
	}
	return false
}
