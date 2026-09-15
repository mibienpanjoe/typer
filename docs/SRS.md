# SRS — Typer

| | |
|---|---|
| **Logiciel** | Typer |
| **Commande** | `typer` |
| **Version spécifiée** | 1.0 (MVP) |
| **Statut** | Brouillon — à valider avant implémentation |
| **Date** | 2026-09-15 |
| **Référence produit** | [PRD](./PRD.md) |

Spécification des exigences logicielles. Ce n’est pas encore le plan d’implémentation ni le détail UX (copy, ordre des prompts). En cas de conflit, le PRD gagne sur l’intention ; ce SRS gagne sur le comportement testable.

---

## 1. Introduction

### 1.1 Objet

Typer est un CLI local Linux qui **programme un envoi unique** d’un texte + Entrée vers une fenêtre de terminal déjà ouverte (Kitty ou GNOME Terminal), à une heure civile choisie par l’opérateur.

### 1.2 Périmètre

Inclus : découverte des fenêtres cibles, **formulaire TUI Charm Huh**, confirmation, planification one-shot, injection à l’heure H, vérification de la cible, journal, annulation.

Exclus : voir PRD §5 (Cursor, daemon, cloud, GUI fenêtrée, détection de crédits, Windows/macOS, uinput global). Le TUI de programmation n’est pas une GUI.

### 1.3 Définitions

| Terme | Sens |
|---|---|
| **Opérateur** | L’utilisateur local qui lance `typer` |
| **Cible** | Une fenêtre Kitty ou GNOME Terminal désignée, plus les identifiants stables capturés à la programmation |
| **Job** | Unité one-shot : cible + heure + message + identifiant |
| **Heure H** | Instant local auquel l’injection doit partir |
| **Backend** | Mécanisme d’injection (`kitty` ou `gnome`) |
| **TUI agent** | Processus au premier plan du terminal qui lit stdin (Codex, Claude Code, …) |
| **Formulaire** | TUI Huh de programmation (pas `fire`) |

### 1.4 Hypothèses techniques

Corriger maintenant si faux — elles conditionnent le SRS.

1. Linux avec **systemd --user** (Pop!_OS).
2. Session graphique de l’opérateur ; pas de multi-siège.
3. **Kitty** : `allow_remote_control` au moins `socket-only` (socket Unix local). Sans ça, le backend Kitty est indisponible.
4. **GNOME Terminal** v1 : injection **fenêtre ciblée** sur session **déverrouillée**, X11 en priorité. Wayland GNOME sans API fenêtrée = limitation documentée, pas uinput.
5. Écrire sur `/dev/pts/N` n’injecte **pas** du stdin ; TIOCSTI n’est pas une dépendance (souvent désactivé).
6. **Go.** UI = Charm **Huh v2** (`charm.land/huh/v2`, [docs](https://pkg.go.dev/charm.land/huh/v2)) + Lipgloss. Ça écarte Python (collision avec la lib Python `typer`). La commande installée reste `typer`.
7. Fuseau : timezone de la machine.
8. Programmation interactive ⇒ stdin/stdout TTY. Sans TTY : flags complets requis, pas de Huh.

---

## 2. Vue d’ensemble du système

### 2.1 Contexte

```
[opérateur, 2e terminal] → typer (CLI + formulaire Huh si TTY)
                              │
                              ├─ découvre fenêtres Kitty / GNOME
                              ├─ formulaire : cible, heure, message, résumé, confirm
                              ├─ écrit job (0600) sous XDG
                              └─ systemd-run --user --on-calendar=H
                                        │
                                        ▼
                              typer fire <job-id>     (à H, sans TTY)
                                        │
                                        ├─ revalide la cible
                                        ├─ backend kitty | gnome
                                        ├─ append journal
                                        └─ détruit job + unité systemd
```

Pas de processus long entre la programmation et H, hors le timer systemd utilisateur.

### 2.2 Modes d’exécution

| Mode | Quand | TTY |
|---|---|---|
| Formulaire Huh / flags | Programmation | oui (Huh) ; sans TTY → flags uniquement |
| `fire` | Déclenché par systemd | non (batch, **jamais** Huh) |
| `list` / `cancel` | Avant H | oui (texte Lipgloss, pas le formulaire de setup) |

`fire` n’est pas un usage humain quotidien ; c’est le point d’entrée du timer.

### 2.3 Structure projet visée

```
docs/           PRD, SRS
cmd/typer/      main (binaire `typer`)
internal/       schedule, backends, job store, tui (Huh)
tests/          tests unitaires hors display
```

Tests Kitty/GNOME réels = manuels / opt-in. Le formulaire Huh n’est pas testé pixel par pixel en CI.

### 2.4 Commandes de dev (cible)

```
go build -o typer ./cmd/typer
go test ./...
typer --help
```

---

## 3. Exigences d’interface

### 3.1 CLI — surface v1

| Invocation | Rôle |
|---|---|
| `typer` | Formulaire Huh : cible, heure, message, résumé live, confirmation → schedule |
| `typer --at <HH:MM> [-m <texte>]` | Même flux, heure (et message) préremplis |
| `typer list` | Jobs en attente |
| `typer cancel [id]` | Annule ; un seul job → pas besoin d’id |
| `typer fire <id>` | Exécution interne à H |

Flags :

| Flag | Sémantique |
|---|---|
| `--at HH:MM` | Aujourd’hui si encore dans le futur (horloge locale). Si l’heure est déjà passée aujourd’hui → **demain** cette heure-là. |
| `-m, --message` | Texte à envoyer, sans le newline final (Typer ajoute Entrée). Défaut : `continue`. |
| `--yes` | Saute la confirmation si et seulement si la cible a été passée de façon non ambiguë (une seule candidate auto, ou id explicite). Sinon erreur. |

Hors v1 (réservés, ne pas implémenter sans mise à jour SRS) : `--dry-run`, `--at` ISO datetime, ciblage par PID en flag.

Sortie :

- Interactif : français, formulaire Huh, cible en clair (émulateur, titre, cwd/projet si connu), puis carte Lipgloss.
- `list` : id, heure H, backend, résumé cible, message **tronqué** (pas un dump).
- Codes de sortie : `0` succès ; `1` erreur utilisateur (pas de cible, heure invalide, annulé) ; `2` échec d’injection à H (cible morte, backend impossible).

### 3.2 Données persistées

Répertoires XDG :

| Usage | Chemin |
|---|---|
| Jobs en attente | `$XDG_STATE_HOME/typer/jobs/<id>.json` (défaut `~/.local/state/typer/jobs/`) |
| Journal | `$XDG_STATE_HOME/typer/log.jsonl` |
| Runtime optionnel | `$XDG_RUNTIME_DIR/typer/` |

Fichiers job : mode **0600**, répertoires **0700**, owner = opérateur.

Schéma job (champs requis) :

```json
{
  "id": "string",
  "created_at": "RFC3339",
  "at": "RFC3339",
  "message": "string",
  "backend": "kitty" | "gnome",
  "target": {
    "emulator": "kitty" | "gnome-terminal",
    "pid": 0,
    "window_id": "string",
    "kitty_id": "string | null",
    "tty": "string | null",
    "title": "string",
    "cwd": "string | null"
  },
  "systemd_unit": "string"
}
```

Le message est une donnée sensible (peut coller un secret par erreur). Pas de copie world-readable. Le journal d’échec/succès enregistre id, heure, backend, motif, **pas** le message en clair (ou alors hash / longueur seulement).

### 3.3 Interfaces externes

| Système | Usage | Contrainte |
|---|---|---|
| `systemd-run --user` | Calendar one-shot → `typer fire <id>` | Unité `typer-job-<id>.service` ; se retire après |
| Kitty remote control | `send-text` + newline vers la fenêtre matchée | Socket local ; pas TCP |
| Fenêtre GNOME (X11) | Envoi de keysyms **à cette fenêtre** | Pas de fallback « fenêtre focus » si la cible a disparu |
| `notify-send` | P3 seulement | Absence = pas d’échec du job |

Typer n’ouvre **aucune** socket réseau, n’appelle aucun HTTP.

---

## 4. Exigences fonctionnelles

### Découverte et ciblage

**FR-01** Typer liste les fenêtres Kitty et GNOME Terminal visibles de l’utilisateur courant, avec émulateur, titre, et cwd/projet s’il est déterminable.

**FR-02** L’opérateur désigne **une** cible avant schedule. Pas de ciblage « fenêtre actuellement focus » comme défaut.

**FR-03** S’il n’y a qu’une cible plausible (heuristique titre/processus agent, à figer en revue UX), Typer la propose et demande confirmation. S’il y en a plusieurs, liste numérotée, choix obligatoire.

**FR-04** À la programmation, Typer **fige** les identifiants de cible (pid, window id, kitty id si applicable, titre, tty). Ce snapshot est la seule cible autorisée à H.

### Programmation

**FR-05** Un job = une cible + une heure H + un message + Entrée. Pas de répétition.

**FR-06** Heure saisie en `HH:MM` locale selon §3.1. Refus si format invalide.

**FR-07** Message : défaut `continue` ; `-m` ou saisie interactive. Interdit : message vide après trim. Taille max 4096 octets UTF-8.

**FR-08** Avant enregistrement, un **résumé humain** est visible dans le formulaire (Note / panneau) : heure, message, émulateur, titre, limitation (ex. GNOME / écran verrouillé). Sauf `--yes` (conditions §3.1).

**FR-09** Refus de programmer un backend que Typer sait **déjà** inopérant (Kitty sans remote control ; GNOME sans injecteur dispo). Message d’erreur actionnable **dans** le formulaire ou en stderr si pas de TTY.

**FR-10** Sur cible GNOME, le résumé **avertit** que l’écran verrouillé à H fera échouer l’envoi. Le job reste programmable (cas majoritaire = écran ouvert).

### Formulaire TUI (programmation)

**FR-23** Si stdin et stdout sont un TTY et que la programmation n’est pas entièrement non-interactive (`--yes` valide), Typer ouvre un formulaire **Huh v2** (un groupe / une page) : `Select` cible, `Input` heure, `Input` message, résumé, `Confirm`.

**FR-24** Le résumé se met à jour quand cible, heure ou message changent, **avant** la confirmation. Après succès, une carte Lipgloss est imprimée sur stdout (scrollback) : H, cible, id, commande d’annulation.

**FR-25** `typer fire` n’initialise pas Huh, n’entre pas en alt-screen, n’attend pas d’input.

**FR-26** Annulation du formulaire (Esc / Ctrl+C / Confirm = non) : exit 1, **aucun** job créé.

**FR-27** Sans TTY : Huh interdit. Il faut `--at` et une cible non ambiguë ; sinon exit 1.

### Déclenchement (H)

**FR-11** À H, Typer revalide la cible **avant** toute injection : processus vivant, identifiants cohérents, titre encore compatible avec le snapshot (égalité ou règle de préfixe documentée dans le code). Échec → pas d’injection, journal, exit 2.

**FR-12** Si la revalidation échoue, Typer **n’essaie aucune autre fenêtre**.

**FR-13** Backend `kitty` : envoi du message puis newline via remote control vers **cette** fenêtre. Doit fonctionner écran verrouillé si le socket Kitty est accessible.

**FR-14** Backend `gnome` : envoi du message puis Entrée uniquement si la session est déverrouillée **et** la fenêtre cible toujours adressable. Sinon échec journalisé, pas de fallback Kitty, pas de fallback focus.

**FR-15** Après tentative (succès ou échec terminal), le fichier job et l’unité systemd associée sont retirés.

### Cycle de vie

**FR-16** `typer list` montre uniquement les jobs encore en attente.

**FR-17** `typer cancel [id]` stoppe l’unité systemd et supprime le job. Rien n’est envoyé à H.

**FR-18** Plusieurs jobs v1 : **autorisés s’ils visent des cibles distinctes**. Deux jobs sur la **même** cible → refus au schedule (conflit).

### Sécurité (comportement)

**FR-19** Pas de privilège root. Pas de setuid.

**FR-20** Pas d’écoute réseau. Kitty : socket Unix déjà prévu par Kitty, pas d’ouverture de port par Typer.

**FR-21** Pas d’injecteur global (uinput, ydotool, « taper dans la session entière »).

**FR-22** `fire` refuse de s’exécuter si le job n’appartient pas à l’uid courant ou si le fichier n’est pas 0600 / owner self (défense en profondeur).

---

## 5. Exigences non fonctionnelles

**NFR-01 Sécurité — moindre privilège.** systemd --user seulement. Fichiers 0600/0700.

**NFR-02 Sécurité — surface.** Un job en attente n’est pas un daemon d’accessibilité. Annulation = arrêt de la surface.

**NFR-03 Fiabilité.** Mieux vaut ne rien envoyer que d’envoyer au mauvais TTY. La fail-closed est la règle par défaut.

**NFR-04 Observabilité.** Toute exécution `fire` ajoute une ligne au journal : id, timestamp, résultat (`sent` \| `aborted_missing_target` \| `aborted_locked` \| `aborted_backend`), backend. Pas de message en clair.

**NFR-05 Performance.** Découverte et schedule < 2 s sur une machine locale typique (hors attente humaine).

**NFR-06 Portabilité v1.** Linux systemd + Kitty et/ou GNOME Terminal. Pas de promesse Wayland GNOME.

**NFR-07 Simplicité.** Dépendances : stdlib Go + Charm (Huh, Lipgloss) + outils déjà présents (`systemd-run`, `kitty`). Pas de toolkit graphique.

**NFR-08 UX temps.** Le flux « limite Codex → job posé » tient en moins d’une minute ; le formulaire est **une page**, pas un wizard.

**NFR-09 UI.** Look Charm (Huh thèmes + Lipgloss). Le TUI ne doit pas masquer le reçu : après `Run()`, impression d’une carte hors alt-screen.

---

## 6. Règles Always / Ask first / Never

**Always**

- Revalider la cible avant injection.
- Fail-closed si doute sur l’identité de la fenêtre.
- Permissions 0600 sur les jobs.
- Retirer job + unité après `fire` ou `cancel`.
- Tests des parseurs d’heure, du schéma job, et des règles fail-closed **sans** display.

**Ask first**

- Ajouter un backend (tmux, Cursor, Wayland uinput).
- Changer la sémantique `--at` (aujourd’hui vs datetime absolue).
- Logger le message en clair.
- Dépendance native nouvelle **hors** Charm Huh / Lipgloss / Bubble Tea.
- Comportement Persistent=true (envoi au réveil si le PC a dormi).

**Never**

- Injecter dans une autre fenêtre que le snapshot.
- Écouter le réseau / envoyer des télémétries.
- Exiger root.
- Daemon H24.
- Utiliser TIOCSTI ou l’écriture brute sur le slave PTY comme stratégie principale (mauvais modèle mental : ce n’est pas du stdin).

---

## 7. Cas d’erreur

| Situation | Comportement |
|---|---|
| Aucune fenêtre Kitty/GNOME | Exit 1, expliquer d’ouvrir l’agent dans Kitty ou GNOME Terminal |
| Kitty sans remote control | Backend Kitty indisponible ; si la cible est Kitty, refuser le schedule |
| Heure invalide | Exit 1 |
| Message vide / trop long | Exit 1 |
| Cible disparue à H | Pas d’injection, log `aborted_missing_target`, exit 2 |
| GNOME + session lockée à H | Pas d’injection, log `aborted_locked`, exit 2 |
| Titre de fenêtre trop divergent du snapshot | Traiter comme cible disparue (fail-closed) |
| Job file tampered / mauvais owner | `fire` refuse, log, exit 2 |
| systemd-run indisponible | Refuser le schedule, exit 1, dire que systemd --user est requis |

Notifications desktop : hors exigences v1.

---

## 8. Style et tests

### Style (contrat)

CLI : flags anglais, **formulaire et messages opérateur en français**. Noms internes en anglais (`Job`, `Target`, `Backend`). Stack UI : Huh v2 + Lipgloss ; un module backend = `send(target, message) -> Result`.

```
go test ./...
go build -o typer ./cmd/typer
```

### Tests v1 (obligatoires, sans écran)

- Parse `--at HH:MM` (futur aujourd’hui, déjà passé → demain, invalide).
- Validation message (vide, max length).
- Sérialisation job round-trip.
- FR-12 : si revalidation faux, `send` n’est **jamais** appelé (mock).
- FR-18 : second job même cible → erreur.
- `cancel` retire job et n’appelle pas `send`.
- Permissions : fichier job créé 0600 (si le FS de test le permet).

### Tests manuels (acceptation, cf. PRD §13)

Kitty déverrouillé, Kitty verrouillé, GNOME déverrouillé, cible fermée avant H, cancel.

---

## 9. Critères d’acceptation (logiciel)

Un build v1 est conforme si :

1. `typer` sur un TTY ouvre un formulaire Huh (cible, heure, message, résumé, confirm) puis imprime une carte résumé.
2. `typer --help` expose schedule / list / cancel.
3. Un job Kitty envoie message + Entrée à la fenêtre snapshot, y compris écran locké *si* remote control OK.
4. Un job GNOME le fait écran déverrouillé ; écran locké → aucun keystroke ailleurs + log d’abort.
5. Cible tuée avant H → aucun envoi + log.
6. `cancel` → aucun envoi à H.
7. Aucun listener TCP/UDP ouvert par Typer (vérifiable `ss` pendant un job en attente : seulement le timer systemd, pas un process typer).
8. Fichiers job non lisibles par un autre user.

---

## 10. Décisions déjà prises / encore ouvertes

**Tranché**

- Nom produit et commande : **Typer** / `typer`.
- One-shot, heure choisie par l’humain, cible explicite.
- Programmation : formulaire **Charm Huh** + résumé + carte Lipgloss après coup.
- Langage : **Go**.
- Backends : Kitty (lock OK via socket) + GNOME (unlock only).
- Scheduler : systemd --user, pas de daemon maison.
- Fail-closed, pas d’injecteur global.
- Match titre à H : égalité ou préfixe (un titre commence par l’autre).

**Ouvert (revue UX copy / fonctionnement détaillé, puis ADR si besoin)**

- Heuristique exacte « cible plausible ».
- Résumé live : `Note` Huh vs layout deux colonnes Bubble Tea.
- Copy des labels, `--yes`, dry-run, thème.
- Wayland GNOME : reporter ou abandonner ce backend sur COSMIC.

---

## 11. Traçabilité PRD → SRS

| PRD | SRS |
|---|---|
| P1 formulaire Huh + résumé | FR-08, FR-23–FR-27, NFR-08, NFR-09 |
| P1 one-shot heure + message + Entrée | FR-05, FR-07, FR-13, FR-14 |
| P1 écran déverrouillé | FR-13, FR-14 |
| P1 Kitty écran verrouillé | FR-13 |
| P1 refuse si fenêtre morte | FR-11, FR-12 |
| P1 cancel | FR-17 |
| P1 local, pas root, pas réseau | FR-19–FR-21, NFR-01, NFR-02 |
| P2 défaut `continue`, journal | FR-07, NFR-04 |
| Hors scope Cursor / daemon / uinput | §1.2, FR-21, Never |
