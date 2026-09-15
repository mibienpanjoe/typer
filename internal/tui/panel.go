package tui

import (
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
	"github.com/mibienpanjoe/typer/internal/schedule"
)

func Panel(selected *domain.Target, pending []domain.Job, width int) string {
	if len(pending) == 0 {
		return ""
	}
	if width < 48 {
		width = 48
	}
	cibleW := max(18, width-32)

	w := table.NewWriter()
	style := table.StyleRounded
	style.Format.Header = text.FormatDefault
	style.Options.SeparateRows = false
	w.SetStyle(style)
	w.SetTitle("Jobs en attente")
	w.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMin: 5, WidthMax: 5, WidthMaxEnforcer: clip},
		{Number: 2, WidthMax: 14, WidthMaxEnforcer: clip},
		{Number: 3, WidthMax: cibleW, WidthMaxEnforcer: clip},
	})
	w.AppendHeader(table.Row{"Heure", "Message", "Cible"})

	jobs := append([]domain.Job(nil), pending...)
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].At.Before(jobs[j].At) })
	for _, j := range jobs {
		label := discover.Label(j.Target)
		if selected != nil && schedule.Conflict([]domain.Job{j}, *selected) {
			label += "  !"
		}
		w.AppendRow(table.Row{j.At.Format("15:04"), clipMsg(j.Message), label})
	}
	return w.Render()
}

func clipMsg(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return domain.DefaultMessage
	}
	return msg
}

func clip(s string, n int) string {
	return text.Snip(s, n, "…")
}
