<p align="center">
  <img src="docs/typer-icon.png" width="168" alt="Typer">
</p>

# Typer

When a terminal agent (Codex, Claude Code, Cursor Agent, …) hits a usage limit, it sits there until someone types a prompt and presses Enter. Typer does that **once**, at an hour **you** choose, into a **window you pick**.

It is a local Linux CLI. No daemon, no cloud, no “whatever is focused.” After the send (or a hard failure), the job is gone.

The TUI is in French. Flags stay in English (`--at`, `-m`).

## Install

Needs [Go](https://go.dev/dl/) 1.25+ on `PATH` (on Pop!_OS, `/usr/local/go/bin` is often missing from `PATH`).

```bash
export PATH=/usr/local/go/bin:$PATH
make install
```

That puts `typer` in `~/.local/bin`. Then, from any terminal:

```bash
typer          # create a job (TUI)
typer list     # pending jobs only
typer cancel   # drop a job still waiting
```

`make uninstall` removes the binary. Override the prefix with `make install PREFIX=/usr/local`.

Jobs remember the **binary path** from schedule time. After moving the install, `typer cancel` and create the job again.

## Typical use

1. The agent is already open in **Kitty** or **GNOME Terminal** (not Cursor’s integrated terminal).
2. Open a **second** terminal — the agent owns the first TUI.
3. Run `typer`.
4. Pick the agent window, set local time `HH:MM`, keep or edit the message (default `continue`).
5. **Enter** on Message saves the job. Esc cancels. You get a receipt with the id and `typer cancel …`.

Leave the PC **on** (sleep/suspend is not handled). At that time Typer checks the window is still the same one, types the text, presses Enter, logs the result, and deletes the job.

`--at` and `-m` pre-fill the form. `--yes` skips the form only if there is exactly one plausible agent window.

```bash
typer --at 06:34 -m continue
typer --at 06:34 -m continue --yes   # one unambiguous target
```

## Commands

| Command | Role |
|---|---|
| `typer` | **Create** a one-shot job (form if you have a TTY) |
| `typer list` | **Read** pending jobs — does not create anything |
| `typer cancel [id]` | Remove a waiting job (id required if several) |
| `typer fire <id>` | Internal: systemd calls this at hour H. Do not run by hand. |

## Kitty vs GNOME Terminal

**Kitty** (including a locked screen): Unix-socket remote control is required.

```conf
# ~/.config/kitty/kitty.conf — then restart Kitty
allow_remote_control socket-only
listen_on unix:${XDG_RUNTIME_DIR}/kitty
```

**GNOME Terminal** (unlocked X11 session only):

```bash
sudo apt install wmctrl xdotool
```

Keep the agent **tab focused** in that window. Typer will not type into “whatever is in front.” If the session lock state cannot be read, it refuses to send.

## Behaviour

- One job per window. A second job for the same target is rejected until you cancel.
- If the window is gone, the title no longer matches, or the backend cannot send, **nothing else is typed**. The failure is in the local log, not in another terminal.
- Jobs live under `$XDG_STATE_HOME/typer` (override with `TYPER_STATE`). The log does not store the prompt text.

## Specs

[PRD](docs/PRD.md) · [SRS](docs/SRS.md)
