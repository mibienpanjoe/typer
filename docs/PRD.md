# PRD — Typer

| | |
|---|---|
| **Produit** | Typer |
| **Commande** | `typer` |
| **Statut** | Brouillon — à valider avant implémentation |
| **Date** | 2026-09-15 |
| **Auteur** | Conversation produit (usage personnel) |
| **Public** | Une personne : l’opérateur de Codex / Claude Code dans un terminal Linux |
| **Liens** | [SRS](./SRS.md) |

Ce document décrit **pourquoi** Typer existe et **quoi** livrer. Le **comment** logiciel est dans le SRS. Le setup interactif est un **formulaire Charm Huh** (PRD §8) ; le copy exact des labels reste à peaufiner.

---

## 1. Problème

Les agents de code en terminal (Codex CLI, Claude Code, équivalents) s’arrêtent quand les crédits d’usage sont épuisés. Le reset a une heure connue (`try again at 6:32 AM`). L’agent **ne reprend pas tout seul** : il attend un nouveau prompt + Entrée.

L’opérateur est souvent **ailleurs ou endormi** à l’heure du reset. S’il programme un message trop tôt, les crédits ne sont pas encore revenus. S’il ne programme rien, la tâche reste gelée jusqu’à son retour.

**How might we :** comment reprendre une session agent déjà ouverte, à une heure choisie par l’humain, sans daemon et sans taper dans le mauvais terminal ?

## 2. Utilisateur

Personne unique, sur **Pop!_OS / Linux**, qui lance Codex (et d’autres agents TUI) dans **Kitty** ou **GNOME Terminal**.

Ce n’est pas un produit multi-utilisateur, ni un plugin Cursor, ni un service cloud.

## 3. Job to be done

> Quand je vois la limite d’usage dans Codex, je veux programmer **une seule** frappe (texte + Entrée) vers **cette** fenêtre, à **une heure que je choisis** (souvent 2 minutes après l’heure affichée), pour que la tâche reprenne pendant que je ne suis pas devant la machine.

## 4. Objectifs produit

1. **Reprise one-shot.** Un job = une fenêtre + une heure + un message. Après envoi (ou échec terminal), le job n’existe plus.
2. **L’humain décide l’heure.** Typer ne détecte pas le reset des crédits. `6h34` plutôt que `6h32` est un choix utilisateur, pas une heuristique.
3. **Cible explicite.** On n’envoie jamais « dans le terminal focus » par défaut. La fenêtre est désignée à la programmation.
4. **Deux émulateurs.** Kitty (y compris écran verrouillé, si le backend Kitty est dispo) et GNOME Terminal (écran déverrouillé). Le PC reste allumé ; la veille/suspend n’est pas un objectif v1.
5. **Confiance.** Local, pas de réseau, pas de root, pas de daemon H24, possibilité d’annuler avant l’heure H.

## 5. Non-objectifs (v1)

| Hors scope | Pourquoi |
|---|---|
| Terminal intégré Cursor / VS Code | Autre surface (Electron), autre menace |
| Détection auto de l’heure de reset | Fragile, spécifique à chaque CLI, inutile si l’humain lit le message |
| Daemon / service toujours actif | Surface d’attaque inutile pour un one-shot |
| Windows / macOS | L’usage réel est Linux |
| GUI | La commande suffit |
| tmux comme prérequis | L’usage réel n’est pas dans tmux |
| Injection globale clavier (uinput / ydotool root) | Trop de pouvoir ; tape n’importe où |
| Intégration propriétaire Codex / Claude | Un TUI qui lit stdin suffit |
| Reprise si le PC était en veille à l’heure H | Comportement systemd possible plus tard ; pas promis |

## 6. Parcours principal

```
Limite affichée dans Codex
        │
        ▼
Ouvrir un autre terminal (Codex occupe déjà le TUI)
        │
        ▼
typer   (ou typer --at 06:34 -m "continue")
        │
        ▼
Formulaire TUI Charm (Huh) — une page, pas un wizard
  · Select : fenêtre Kitty / GNOME (titre + projet)
  · Input  : heure locale (préremplie si --at)
  · Input  : message (défaut continue)
  · Résumé : carte à jour pendant la saisie
  · Confirm : Programmer
        │
        ▼
Job systemd --user one-shot ; le TUI se ferme ; une carte résumé reste dans le scrollback
        │
        ▼
À l’heure H : vérifier que la cible est encore là → injecter → Entrée → journaliser → détruire le job
```

À l’heure H, l’opérateur n’est **pas** dans la boucle. Succès = le TUI agent a reçu le texte comme s’il l’avait tapé.

## 7. Parcours secondaires

- **Annuler** un job encore en attente (`typer cancel`).
- **Lister** les jobs en attente (`typer list`).
- **Échec silencieux côté TUI, visible au réveil :** cible disparue, backend impossible (GNOME + écran verrouillé), socket Kitty absent → **ne rien envoyer ailleurs**, écrire un journal d’échec.

## 8. UX — formulaire Charm (v1)

La programmation n’est **pas** une suite de prompts bruts, ni une GUI. C’est un **formulaire TUI** Charm.

**Outil :** [Huh](https://github.com/charmbracelet/huh) (formulaires, au-dessus de Bubble Tea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) pour la carte résumé. Pas Gum : trop séquentiel. Pas une appli Bubble Tea plein écran pour v1, sauf si le résumé live l’exige (Huh s’embarque dans un `tea.Model`).

**Vitesse :** Tab / flèches / Entrée. Une page. Défauts déjà remplis. Avec `typer --at 06:34 -m "continue"`, on ne re-tape ni l’heure ni le message : fenêtre (si besoin) puis confirmation.

**Écran de setup (contrat ; copy encore ajustable) :**

| Champ | Type Huh | Défaut |
|---|---|---|
| Cible | `Select` — une ligne humaine (`Kitty · Codex · afrikoopps`) | la seule cible plausible, sinon rien de présélectionné |
| Heure | `Input` `HH:MM` | `--at` ou vide |
| Message | `Input` | `--message` ou `continue` |
| Résumé | `Note` et/ou panneau Lipgloss | se met à jour quand les champs changent |
| Programmer | `Confirm` | non, jusqu’à oui explicite |

Le résumé dit **exactement** ce qui partira, par exemple :

> À **06:34**, envoyer **« continue »** + Entrée  
> → **Kitty** · `gpt-5.6-sol` · `~/projects/afrikoopps`  
> ⚠ GNOME : échouera si l’écran est verrouillé à cette heure.

Après validation : le formulaire disparaît ; une **carte Lipgloss** reste dans le scrollback (heure, cible, id job, `typer cancel …`). C’est le reçu.

**Hors du formulaire :** `typer list` / `typer cancel` restent des commandes courtes (table Lipgloss, pas un second gros TUI). `typer fire` n’ouvre **jamais** de TUI (pas de TTY chez systemd).

**Accessibilité :** Huh `WithAccessible(true)` si `ACCESSIBLE` est défini — prompts linéaires ([doc Huh](https://pkg.go.dev/charm.land/huh/v2)).

Principes inchangés : moins d’une minute ; cible en langage humain ; limites (GNOME + lock) **dans** le résumé ; confirmer avant d’injecter ; pas de magie crédits.

## 9. Exigences produit (niveau PRD)

**P1 — Must**

- Programmer via un **formulaire TUI Charm Huh** (une page + résumé + confirmation), pas une GUI.
- Programmer un envoi unique (heure locale + message + Entrée) vers une fenêtre Kitty ou GNOME Terminal désignée.
- Fonctionner quand l’écran est déverrouillé (cas majoritaire).
- Sur Kitty, fonctionner aussi écran verrouillé **si** le contrôle distant Kitty est disponible.
- Refuser d’envoyer si la fenêtre/processus cible n’existe plus.
- Permettre d’annuler avant l’heure H.
- Ne pas écouter le réseau. Ne pas demander root.

**P2 — Should**

- Un seul candidat plausible → confirmation courte au lieu d’une liste longue.
- Journal local consultable au réveil (succès / motif d’échec).
- Message par défaut `continue`, surchargeable.

**P3 — Could (pas v1 sauf si trivial)**

- Notification desktop au moment de l’envoi.
- Plusieurs jobs en parallèle vers des fenêtres distinctes.
- Heuristique « +2 min après l’heure lue dans le terminal » (l’humain le fait déjà).

## 10. Succès

Typer v1 est réussi si, **sans être devant la machine** :

1. L’opérateur a programmé en une minute après le message de limite.
2. À l’heure choisie, la **bonne** session Codex (ou TUI équivalent) reçoit le message + Entrée.
3. Aucun autre terminal, aucun prompt mot de passe, aucune autre session agent ne reçoit la frappe.
4. S’il a changé d’avis, il a pu annuler.
5. S’il a fermé la fenêtre, rien n’a été envoyé « au hasard ».

## 11. Hypothèses (à corriger si faux)

1. Usage **personnel**, une session graphique Linux, user systemd.
2. Le PC reste **allumé** jusqu’à l’heure H (pas de suspend promis).
3. Codex / Claude Code sont des TUI qui consomment stdin dans le terminal ; un texte + `\n` suffit.
4. L’opérateur peut ouvrir **un second terminal** pour lancer `typer` (le premier est occupé par l’agent).
5. Kitty peut activer `allow_remote_control` (socket local) ; c’est le chemin fiable.
6. GNOME Terminal v1 = session **déverrouillée** ; écran verrouillé GNOME = échec assumé et annoncé.
7. Nom produit : **Typer**. Commande : `typer`. Dépôt : `typer`. Langage : **Go** (Charm Huh).
8. Langue : formulaire et messages en **français**. Flags CLI en anglais (`--at`, `-m`).

## 12. Questions ouvertes (produit / UX)

Ces points sont **volontairement** non figés ; revue UX + fonctionnement détaillé ensuite.

- Copy exact des labels Huh (titres, aide, bouton Programmer).
- Règle d’auto-sélection : titre contenant `codex` / `Codex` / `claude` ?
- Résumé live : `Note` Huh suffit, ou petit `tea.Model` à deux colonnes (formulaire + carte) comme l’exemple officiel Huh + Bubble Tea ?
- Faut-il un `typer dry-run` ?
- Thème Lipgloss (clair/sombre, couleurs).

## 13. Critères d’acceptation produit

Voir le SRS pour les critères testables numérotés. Côté produit, la démo d’acceptation est :

1. Ouvrir Codex dans Kitty, simuler une attente de prompt.
2. Depuis un autre terminal : `typer --at <dans 1 minute> -m "continue"`, choisir la fenêtre Codex.
3. Attendre.
4. Le prompt `continue` apparaît dans Codex et est soumis.
5. Recommencer avec GNOME Terminal, écran déverrouillé : même résultat.
6. Programmer, fermer la fenêtre cible avant H : rien n’est tapé ailleurs ; le journal dit « cible absente ».
7. Programmer, `typer cancel` : rien n’est envoyé à H.
