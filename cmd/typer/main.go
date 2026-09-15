package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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

Programme un envoi unique (texte + Entrée) vers une fenêtre Kitty ou GNOME Terminal.

Flags:
  --at HH:MM        Heure locale d'envoi
  -m, --message     Texte à envoyer (défaut: continue)
  --yes             Confirme sans prompt si la cible est non ambiguë
  -h, --help        Aide

Commandes:
  list              Jobs en attente
  cancel            Annule un job encore en attente
  fire              Exécution interne à l'heure H (systemd)
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		fmt.Fprint(stdout, usage)
		return 0
	}

	d := deps(stdout, stderr)
	cmd, rest := splitCommand(args)
	switch cmd {
	case "list":
		return app.List(d)
	case "cancel":
		id := ""
		if len(rest) > 0 {
			id = rest[0]
		}
		return app.Cancel(d, id)
	case "fire":
		id := ""
		if len(rest) > 0 {
			id = rest[0]
		}
		return app.Fire(d, id)
	default:
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
		Form:    tui.RunForm,
		Receipt: tui.Card,
		Exe:     app.MustExe(),
		Alive:   app.DefaultAlive,
		Locked:  func() bool { return app.SessionLocked(nil) },
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

func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

func splitCommand(args []string) (cmd string, rest []string) {
	for i, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		switch a {
		case "list", "cancel", "fire":
			return a, args[i+1:]
		default:
			return "", nil
		}
	}
	return "", nil
}
