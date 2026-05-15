---
description: "UX model, keybinding conventions, hint bar design, action model, and anti-patterns for TUI plugins"
globs: ["plugins/**", "internal/ui/**", "pkg/sdk/frames/**"]
---

Full spec: `docs/tui-ux.md`

# UX Rules

## Navigation Architecture

Two layers of navigation, each with distinct operation types:

### App-Level Navigation (plugin routing)

Controls which plugin occupies the screen. Two behaviors declared via `NavBehavior`:

| Behavior | Constant | Effect | Example |
|----------|----------|--------|---------|
| **Replace** | `NavReplace` (default) | Lateral switch, clears return context | `:state`, `:plan`, `:context` |
| **Push** | `NavPush` | Preserves origin in `returnTo`; returns after completion or cancel | `:chdir`, `:workspaces` |

Key semantics at app level:
- `esc` — pops back to `returnTo` if set, otherwise does nothing (handled by frame stack)
- `q` — always goes home (app root), clears `returnTo`
- `:` — command mode, applies target's `NavBehavior`

Inter-plugin navigation: plugins emit `sdk.NavigateMsg{PluginID}` to request the app navigate to another plugin. The app applies the target's `NavBehavior`.

### Plugin-Level Navigation (frame stack)

Controls sub-views within a plugin. Input always routes to the topmost frame.

```
App: [Home] → [State Plugin (NavReplace)]
                └── Frame Stack: [List] → [Filter]
                                         → [Inspect] → [Confirm]
```

- Input goes to the deepest leaf frame only
- `esc` pops the innermost frame (if stack depth > 1) or triggers plugin deactivation
- `q` pops to app root (bypasses frame stack entirely)
- Each frame declares its own `Hints() []KeyHint`

SDK types (`pkg/sdk/`):
- `Frame` interface: `ID()`, `Update(msg) (Frame, Cmd)`, `View(w,h)`, `Hints()`
- `Stack`: LIFO container with `Push`, `Pop`, `Update`, `View`, `Hints`
- `Stackable` interface: on plugins that use frame-based navigation

Reusable frames (`pkg/sdk/frames/`):
- `FilterFrame`: consumes ALL printable keys; only esc/enter/arrows escape
- `InspectFrame`: scrollable detail + configurable action keys
- `ConfirmFrame`: blocks all input except y/n/esc

Frame lifecycle:
- Return `nil` from `Update` → frame is popped
- Return a different `Frame` → in-place replacement
- Return self → no change

### Navigation Flow Summary

```
:state → :workspaces (NavPush, returnTo=state)
  ├── User selects → WorkspaceChangedEvent → popIfPushed → back to state
  ├── User presses esc → DeactivateMsg → returnTo exists → back to state
  └── User presses q → global handler → home (returnTo cleared)
```

## UX Model (k9s-inspired)

- **`:` command mode**: type plugin name to switch views. Tab autocomplete. Built-in: `:q` quit (guarded when terraform holds a lock), `:q!` force quit.
- **`/` filter mode**: fzf-style fuzzy filter. `esc` exits.
- **`space` pin**: toggle pin on selected resource. Pinned = apply/plan target.
- **`enter` inspect**: show detail view with expanded values.
- **`d` delete**: remove from state — triggers confirmation.
- **`e` edit**: opens $EDITOR at resource's .tf file:line.
- **`m` move**: rename resource address in state.
- **`t` taint**: mark for recreation.
- **`T` untaint**: remove taint mark.
- **`n` import**: import existing resource.
- **`!` batch**: open batch action palette (only when pins > 0).
- **`ctrl+r` refresh**: reload data from terraform.
- **`ctrl+w` wrap**: toggle line wrapping.
- **`←→` pan**: horizontal scroll (10 chars/press, when wrap is off).

## Keybinding Ergonomics

Two layers — bare alpha keys talk to terraform, everything else talks to the interface:

- Bare lowercase = terraform mutation on cursor resource (`d` delete, `t` taint, `e` edit, `a` apply, `n` import, `u` force-unlock)
- Bare uppercase = non-terraform plugin switch (`C` Context, `R` Risk, `P` Phantom, `B` Blast Radius, `T` untaint)
- `ctrl+key` = interface control — view modes and reload (`^t` tree, `^w` wrap, `^r` refresh, `^p` pinned, `^u` unpin all)
- Non-alpha / punctuation = navigation and mode triggers (`/` filter, `!` batch, `[` collapse, `]` expand, `:` command, `Space` pin, `Enter` inspect)
- `q` / `Esc` = leave (universal TUI convention, exception to alpha rule)

Redundant keybindings for accessibility (hints show only primary):
- Navigation: `↓` (shown), `j` (alias); `↑` (shown), `k` (alias)
- Back to home: `q` (shown), `esc` when no sub-state (alias)

Rules:
- `enter` always means inspect — never overloaded
- `space` always means pin — never overloaded
- `q` shown at plugin top-level; `esc` shown only in sub-states
- Plugins must NOT start in filter mode by default
- Full keybinding map: `docs/tui-ux.md` §15

## Action Model (cursor vs batch)

**CRITICAL: direct keys always act on the cursor item. Batch operations go through `!` palette only.**

| Layer | Keys | Scope |
|-------|------|-------|
| Navigate | `↑↓`, `Enter`, `/`, `q`, `Esc` | — |
| Act (single) | `d`, `e`, `t`, `T`, `m`, `n` | Cursor |
| Batch | `!` (palette) | Pinned set |

- Direct action keys NEVER read the pinned set
- `!` is hidden when no pins exist
- Detail/inspect frame has no `!`
- Destructive batch ops always show confirmation with count

Pin semantics:
- PRIMARY: scoping `plan` and `apply` to specific resources
- SECONDARY: enabling batch state actions via `!` palette
- Pins are persistent (survive view switches and sessions)

## Hint Bar Design

- Always 1 line. Never 2 lines.
- One key per entry: `d delete`, `t taint`, `T untaint`
- Context-sensitive: content changes per frame/state
- No novel notation (no `d/D`, no `(t/T) taint/untaint`)
- Display preferences (`^w` wrap) shown only in detail frame
- List frame hints = navigation + state-changing actions
- Detail frame hints = display controls + single-item actions
- Dynamic hints show target state: `^t tree` means "press to enter tree mode"

## UX Anti-patterns (do NOT introduce)

- Shift+letter = batch version of same action
- Implicit batch based on pin state
- Novel hint bar notation (slash grouping, parenthetical pairs)
- 2-line hint bars
- Auto-batch for non-destructive actions

## Detail/Inspect View

- Shows expanded attribute values (JSON)
- Context actions remain: `space` pin, `d` delete, `e` edit
- Scroll indicator `[n/total]` when content overflows
- `[pinned]` indicator shown if resource is pinned

## Confirmation Ownership

Confirmations belong to the plugin that EXECUTES the action, not the plugin that requests it:
- Apply confirmation → owned by apply plugin (`StatusConfirming` state)
- State delete confirmation → owned by state plugin (via `InputConfirm`)

Never double-confirm: if plugin A requests an action from plugin B, only B confirms. Plugin A just emits the request message.

When a plugin is navigated to as a sub-state (has `returnTo`):
- `esc`/cancel during confirmation → emit `DeactivateMsg` → returns to origin
- Hints show `Esc cancel` (not `q back`)

## Staleness Guard

Before destructive operations (apply, state rm, state mv, import):
- Threshold: `cache.staleness_threshold` config (default 5m)
- If stale: prompt "State is Xm old. Refresh first? (y/n)"
- If nil: prompt "No state loaded. Load first?"
