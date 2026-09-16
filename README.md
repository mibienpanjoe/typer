# Typer

CLI Linux one-shot : à une heure choisie, envoie un prompt + Entrée dans une fenêtre **Kitty** ou **GNOME Terminal** (Codex, Claude Code, …) quand les crédits reviennent.

Pas de daemon. Cible explicite. Formulaire TUI [Charm Huh](https://github.com/charmbracelet/huh).

## Install

Place `typer` dans `~/.local/bin` (déjà dans le PATH sur Pop!_OS) :

```bash
export PATH=/usr/local/go/bin:$PATH   # si go n'est pas dans le PATH
make install
```

Ensuite, depuis n’importe quel terminal : `typer`, `typer list`, `typer cancel`.

`make uninstall` retire le binaire. `PREFIX` / `BINDIR` si tu veux un autre préfixe (`make install PREFIX=/usr/local`).

Les jobs déjà programmés gardent le chemin du binaire d’alors : `typer cancel` puis reprogrammer après un déplacement.

## Build (sans installer)

```bash
export PATH=/usr/local/go/bin:$PATH
go test ./...
go build -o typer ./cmd/typer
./typer
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
typer --at 06:34 -m continue --yes   # une seule fenêtre agent
typer                                # formulaire TUI
typer list
typer cancel
```

`TYPER_STATE` : répertoire des jobs (défaut `$XDG_STATE_HOME/typer`).

Specs : [PRD](docs/PRD.md) · [SRS](docs/SRS.md)
