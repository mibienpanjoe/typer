# Typer

CLI Linux one-shot : à une heure choisie, envoie un prompt + Entrée dans une fenêtre **Kitty** ou **GNOME Terminal** (Codex, Claude Code, …) quand les crédits reviennent.

Pas de daemon. Cible explicite. Formulaire TUI [Charm Huh](https://github.com/charmbracelet/huh).

## Build

```bash
export PATH=/usr/local/go/bin:$PATH   # si go n'est pas dans le PATH
go test ./...
go build -o typer ./cmd/typer
```

Kitty : `allow_remote_control socket-only` (ou `yes`) dans `kitty.conf`.

GNOME Terminal : `wmctrl` + `xdotool`, session **déverrouillée** (X11).

```bash
./typer --at 06:34 -m continue --yes   # une seule fenêtre agent
./typer                                # formulaire TUI
./typer list
./typer cancel
```

`TYPER_STATE` : répertoire des jobs (défaut `$XDG_STATE_HOME/typer`).

Specs : [PRD](docs/PRD.md) · [SRS](docs/SRS.md)
