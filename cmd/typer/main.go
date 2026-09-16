package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-isatty"

	"github.com/mibienpanjoe/typer/internal/app"
	"github.com/mibienpanjoe/typer/internal/discover"
	"github.com/mibienpanjoe/typer/internal/domain"
	"github.com/mibienpanjoe/typer/internal/store"
	"github.com/mibienpanjoe/typer/internal/tui"
)

const usage = `Usage: typer [flags]
       typer list
       typer cancel [id]
       typer fire <id>

Programme un envoi unique (texte + soumission) vers une fenêtre Kitty ou GNOME Terminal.

Flags:
  --at HH:MM        Heure locale d'envoi
  -m, --message     Texte à envoyer (défaut: continue)
  --yes             Confirme sans prompt si la cible est non ambiguë
  -h, --help        Aide

Env:
  TYPER_SUBMIT_KEY  Touche finale: ctrl+j (défaut) ou enter

Commandes:
  list              Jobs en attente
  cancel            Annule un job encore en attente
  fire              Exécution interne à l'heure H (systemd)
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprint(stdout, usage)
		return 0
	}

	d := deps(stdout, stderr)
	if len(args) > 0 && args[0] == "list" {
		if len(args) != 1 {
			fmt.Fprintln(stderr, "Usage: typer list")
			return app.ExitUser
		}
		return app.List(d)
	}
	if len(args) > 0 && args[0] == "cancel" {
		if len(args) > 2 {
			fmt.Fprintln(stderr, "Usage: typer cancel [id]")
			return app.ExitUser
		}
		id := ""
		if len(args) == 2 {
			id = args[1]
		}
		return app.Cancel(d, id)
	}
	if len(args) > 0 && args[0] == "fire" {
		if len(args) != 2 {
			fmt.Fprintln(stderr, "Usage: typer fire <id>")
			return app.ExitUser
		}
		return app.Fire(d, args[1])
	}

	at, msg, yes, extra, err := parseScheduleFlags(args, stderr)
	if err != nil {
		return app.ExitUser
	}
	if len(extra) > 0 {
		fmt.Fprintf(stderr, "typer: commande inconnue %q\n", extra[0])
		return app.ExitUser
	}
	return app.Schedule(d, at, msg, yes)
}

func deps(stdout, stderr io.Writer) app.Deps {
	root := store.DefaultRoot()
	if p := os.Getenv("TYPER_STATE"); p != "" {
		root = p
	}
	s := store.New(root)
	return app.Deps{
		Store:  s,
		Runner: discover.DefaultRunner,
		Stdout: stdout,
		Stderr: stderr,
		IsTTY: func() bool {
			return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
		},
		Form:     tui.RunForm,
		Receipt:  tui.Card,
		JobsView: tui.JobList,
		Exe:      app.MustExe(),
		Alive:    app.DefaultAlive,
		Getenv:   os.Getenv,
		PrepareGnome: func() (string, error) {
			return discover.PrepareGnome(nil, os.Getenv, os.Getuid())
		},
		Locked: func(target domain.Target) (bool, error) {
			return app.SessionLocked(nil, target)
		},
	}
}

func parseScheduleFlags(args []string, stderr io.Writer) (at, msg string, yes bool, extra []string, err error) {
	fs := flag.NewFlagSet("typer", flag.ContinueOnError)
	fs.SetOutput(stderr)
	atp := fs.String("at", "", "heure HH:MM")
	msgp := fs.String("message", domain.DefaultMessage, "texte")
	fs.StringVar(msgp, "m", domain.DefaultMessage, "texte")
	yesp := fs.Bool("yes", false, "sans confirmation")
	if err := fs.Parse(args); err != nil {
		return "", "", false, nil, err
	}
	return *atp, *msgp, *yesp, fs.Args(), nil
}
