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
	Confirm bool
}

func RunForm(targets []domain.Target, pending []domain.Job, at, message string) (app.FormResult, error) {
	byID := map[string]domain.Target{}
	opts := make([]huh.Option[string], 0, len(targets)+1)
	choice := initialChoice(targets)
	if choice == "" {
		opts = append(opts, huh.NewOption("— Choisir une cible —", ""))
	}
	labels := targetLabels(targets)
	for i, t := range targets {
		id := fmt.Sprintf("%d", i)
		byID[id] = t
		opts = append(opts, huh.NewOption(labels[i], id))
	}

	st := &formState{Choice: choice, At: at, Message: message}
	if st.Message == "" {
		st.Message = domain.DefaultMessage
	}
	keys := huh.NewDefaultKeyMap()
	keys.Quit.SetKeys("ctrl+c", "esc")
	keys.Quit.SetHelp("esc", "annuler")
	keys.Input.Submit.SetHelp("entrée", "suivant")

	selected := func() *domain.Target {
		t, ok := byID[st.Choice]
		if !ok {
			return nil
		}
		return &t
	}
	recap := func() string {
		text := Summary(selected(), st.At, st.Message, time.Now())
		if panel := Panel(selected(), pending, 80); panel != "" {
			text += "\n\n" + panel
		}
		return text
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Cible").
				Description("Fenêtre qui recevra le prompt").
				Options(opts...).
				Value(&st.Choice).
				Validate(func(choice string) error {
					if choice == "" {
						return fmt.Errorf("choisissez une cible")
					}
					return nil
				}),
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
				Description("Texte envoyé, puis soumis").
				Value(&st.Message).
				Validate(func(s string) error {
					_, err := domain.ValidateMessage(s)
					return err
				}),
			huh.NewNote().
				Title("Résumé").
				DescriptionFunc(recap, st),
			huh.NewConfirm().
				Title("Programmer ce job ?").
				Affirmative("Oui, programmer").
				Negative("Non, annuler").
				Value(&st.Confirm),
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
	return app.FormResult{Target: t, At: st.At, Message: st.Message, Confirm: st.Confirm}, nil
}

func targetLabels(targets []domain.Target) []string {
	labels := make([]string, len(targets))
	counts := make(map[string]int)
	for i, target := range targets {
		labels[i] = discover.Label(target)
		counts[labels[i]]++
	}
	for i, target := range targets {
		if counts[labels[i]] > 1 {
			labels[i] += " · fenêtre " + target.WindowID
		}
	}
	return labels
}

func initialChoice(targets []domain.Target) string {
	picked, err := discover.PickForYes(targets)
	if err != nil {
		return ""
	}
	for i, target := range targets {
		if target.Identity() == picked.Identity() {
			return fmt.Sprintf("%d", i)
		}
	}
	return ""
}

func Summary(target *domain.Target, at, message string, now time.Time) string {
	when, timeErr := domain.ParseClock(at, now)
	msg, messageErr := domain.ValidateMessage(message)
	if target == nil || timeErr != nil || messageErr != nil {
		return "Complétez la cible, l'heure et le message pour afficher le récapitulatif."
	}
	text := fmt.Sprintf("%s, envoyer « %s » puis soumettre\n→ %s", formatWhen(when, now), msg, discover.Label(*target))
	if target.Emulator == domain.EmulatorGnome {
		text += "\n⚠ GNOME : l'envoi échouera si l'écran est verrouillé. Gardez l'onglet agent actif."
	}
	return text
}

func formatWhen(at, now time.Time) string {
	atDate := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch {
	case atDate.Equal(nowDate):
		return "Aujourd'hui à " + at.Format("15:04")
	case atDate.Equal(nowDate.AddDate(0, 0, 1)):
		return "Demain à " + at.Format("15:04")
	default:
		return "Le " + at.Format("02/01/2006 à 15:04")
	}
}

func Card(job domain.Job, warn string) string {
	inner := fmt.Sprintf(
		"Typer · job enregistré\n%s, envoyer « %s » puis soumettre (%s)\n→ %s\nid %s\nAnnuler : typer cancel %s",
		formatWhen(job.At, time.Now()),
		job.Message,
		submitLabel(job.SubmitKey),
		discover.Label(job.Target),
		job.ID,
		job.ID,
	)
	if warn != "" {
		inner += "\n" + warn
	}
	return box().Render(inner) + "\n"
}

func submitLabel(key string) string {
	normalized, err := domain.NormalizeSubmitKey(key)
	if err != nil {
		normalized = domain.DefaultSubmitKey
	}
	if normalized == "enter" {
		return "Entrée"
	}
	return "Ctrl+J"
}

func box() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)
}
