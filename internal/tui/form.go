package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	huh "charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/mibienpanjoe/typer/internal/app"
	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
)

func RunForm(targets []domain.Target, at, message string) (app.FormResult, error) {
	byID := map[string]domain.Target{}
	opts := make([]huh.Option[string], 0, len(targets))
	for i, t := range targets {
		id := fmt.Sprintf("%d", i)
		byID[id] = t
		opts = append(opts, huh.NewOption(discover.Label(t), id))
	}

	choice := ""
	if p, err := discover.PickForYes(targets); err == nil {
		for i, t := range targets {
			if t.Identity() == p.Identity() {
				choice = fmt.Sprintf("%d", i)
				break
			}
		}
	}
	if message == "" {
		message = domain.DefaultMessage
	}
	ok := false

	summary := func() string {
		t, found := byID[choice]
		if !found {
			return "Choisissez une fenêtre, une heure HH:MM, et le texte à envoyer."
		}
		warn := ""
		if t.Emulator == domain.EmulatorGnome {
			warn = "\n⚠ GNOME : échouera si l'écran est verrouillé à cette heure."
		}
		msg := message
		if strings.TrimSpace(msg) == "" {
			msg = domain.DefaultMessage
		}
		when := at
		if when == "" {
			when = "??:??"
		}
		return fmt.Sprintf("À %s, envoyer « %s » + Entrée\n→ %s%s", when, msg, discover.Label(t), warn)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Cible").
				Description("Fenêtre qui recevra le prompt").
				Options(opts...).
				Value(&choice),
			huh.NewInput().
				Title("Heure").
				Placeholder("06:34").
				Value(&at).
				Validate(func(s string) error {
					_, err := domain.ParseClock(s, time.Now())
					return err
				}),
			huh.NewInput().
				Title("Message").
				Value(&message),
			huh.NewConfirm().
				Title("Programmer ?").
				DescriptionFunc(summary, []any{&choice, &at, &message}).
				Affirmative("Programmer").
				Negative("Annuler").
				Value(&ok),
		),
	)
	if os.Getenv("ACCESSIBLE") != "" {
		form = form.WithAccessible(true)
	}
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return app.FormResult{Confirm: false}, nil
		}
		return app.FormResult{}, err
	}
	t, exists := byID[choice]
	if !exists {
		return app.FormResult{Confirm: false}, fmt.Errorf("cible invalide")
	}
	return app.FormResult{Target: t, At: at, Message: message, Confirm: ok}, nil
}

func Card(job domain.Job, warn string) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)
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
	return box.Render(inner) + "\n"
}
