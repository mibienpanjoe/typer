# Typer

CLI Linux one-shot : à une heure choisie, envoie un prompt + Entrée dans une fenêtre **Kitty** ou **GNOME Terminal** (Codex, Claude Code, …) quand les crédits reviennent.

Pas de daemon. Cible explicite. Formulaire TUI [Charm Huh](https://github.com/charmbracelet/huh).

## Build

```bash
export PATH=/usr/local/go/bin:$PATH   # si go n'est pas dans le PATH
go test ./...
go build -o typer ./cmd/typer
```

Kitty : remote control via socket Unix obligatoire. Dans `kitty.conf` :

```conf
allow_remote_control socket-only
listen_on unix:${XDG_RUNTIME_DIR}/kitty
```

GNOME Terminal : `wmctrl` + `xdotool`, session **X11 déverrouillée**. Garder l’onglet agent actif dans la fenêtre ciblée ; Typer refuse l’envoi si l’état de verrouillage est inconnu.

```bash
sudo apt install wmctrl xdotool
```

```bash
./typer --at 06:34 -m continue --yes   # une seule fenêtre agent
./typer                                # formulaire TUI
./typer list
./typer cancel
```

`TYPER_STATE` : répertoire des jobs (défaut `$XDG_STATE_HOME/typer`).

Specs : [PRD](docs/PRD.md) · [SRS](docs/SRS.md)
