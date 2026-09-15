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

type formState struct {
	Choice  string
	At      string
	Message string
}

func Recap(targets map[string]domain.Target, st *formState) string {
	t, found := targets[st.Choice]
	if !found {
		return "Choisissez une fenêtre, une heure HH:MM, et le texte."
	}
	when := strings.TrimSpace(st.At)
	if when == "" {
		when = "??:??"
	}
	msg := strings.TrimSpace(st.Message)
	if msg == "" {
		msg = domain.DefaultMessage
	}
	warn := ""
	if t.Emulator == domain.EmulatorGnome || t.KittyID == nil {
		warn = "\n⚠ Écran verrouillé à l'heure H : l'envoi clavier échouera."
	}
	return fmt.Sprintf("À %s, envoyer « %s » + Entrée\n→ %s%s", when, msg, discover.Label(t), warn)
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
	if p, err := discover.PickForYes(targets); err == nil {
		for i, t := range targets {
			if t.Identity() == p.Identity() {
				st.Choice = fmt.Sprintf("%d", i)
				break
			}
		}
	}

	ok := false
	recap := func() string { return Recap(byID, st) }

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
				Value(&st.Message),
			huh.NewNote().
				Title("Résumé").
				DescriptionFunc(recap, st),
			huh.NewConfirm().
				Title("Programmer ce job ?").
				DescriptionFunc(recap, st).
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
	t, exists := byID[st.Choice]
	if !exists {
		return app.FormResult{Confirm: false}, fmt.Errorf("cible invalide")
	}
	return app.FormResult{Target: t, At: st.At, Message: st.Message, Confirm: ok}, nil
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
