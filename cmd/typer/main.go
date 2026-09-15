package main

import (
	"fmt"
	"io"
	"os"
	"strings"
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

	cmd, _ := splitCommand(args)
	switch cmd {
	case "list", "cancel", "fire":
		fmt.Fprintf(stderr, "typer %s: pas encore implémenté\n", cmd)
		return 1
	case "":
		if len(args) == 0 {
			fmt.Fprint(stdout, usage)
			return 0
		}
		fmt.Fprintln(stderr, "typer: programmation pas encore implémentée")
		return 1
	default:
		fmt.Fprintf(stderr, "typer: commande inconnue %q\n", cmd)
		return 1
	}
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
		return a, args[i+1:]
	}
	return "", nil
}
