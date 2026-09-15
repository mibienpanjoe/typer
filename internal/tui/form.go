package tui

import (
	"errors"
	"fmt"
	"os"
	"time"

	huh "charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/mibienpanjoe/typer/internal/app"
	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
)

type formState struct {
	Choice  string
	At      string
	Message string
}

func RunForm(targets []domain.Target, at, message string) (app.FormResult, error) {
	byID := map[string]domain.Target{}
	opts := make([]huh.Option[string], 0, len(targets))
	for i, t := range targets {
		id := fmt.Sprintf("%d", i)
		byID[id] = t
		opts = append(opts, huh.NewOption(discover.Label(t), id))
	}

	st := &formState{At: at, Message: message}
	if st.Message == "" {
		st.Message = domain.DefaultMessage
	}
	if len(targets) > 0 {
		st.Choice = "0"
	}
	if p, err := discover.PickForYes(targets); err == nil {
		for i, t := range targets {
			if t.Identity() == p.Identity() {
				st.Choice = fmt.Sprintf("%d", i)
				break
			}
		}
	}

	keys := huh.NewDefaultKeyMap()
	keys.Quit.SetKeys("ctrl+c", "esc")
	keys.Quit.SetHelp("esc", "annuler")
	keys.Input.Submit.SetHelp("entrée", "programmer")

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Cible").
				Description("Fenêtre qui recevra le prompt").
				Options(opts...).
				Value(&st.Choice),
			huh.NewInput().
				Title("Heure").
				Placeholder("06:34").
				Value(&st.At).
				Validate(func(s string) error {
					_, err := domain.ParseClock(s, time.Now())
					return err
				}),
			huh.NewInput().
				Title("Message").
				Description("Entrée enregistre le job · Esc annule").
				Value(&st.Message),
		),
	).WithKeyMap(keys)

	if os.Getenv("ACCESSIBLE") != "" {
		form = form.WithAccessible(true)
	}
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return app.FormResult{Confirm: false}, nil
		}
		return app.FormResult{}, err
	}
	t, exists := byID[st.Choice]
	if !exists {
		return app.FormResult{Confirm: false}, fmt.Errorf("cible invalide")
	}
	return app.FormResult{Target: t, At: st.At, Message: st.Message, Confirm: true}, nil
}

func Card(job domain.Job, warn string) string {
	inner := fmt.Sprintf(
		"Typer · job enregistré\nÀ %s  envoyer « %s » + Entrée\n→ %s\nid %s\nAnnuler : typer cancel %s",
		job.At.Format("15:04"),
		job.Message,
		discover.Label(job.Target),
		job.ID,
		job.ID,
	)
	if warn != "" {
		inner += "\n" + warn
	}
	return box().Render(inner) + "\n"
}

func box() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)
}
