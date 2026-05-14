---
description: "UX model, keybinding conventions, hint bar design, action model, and anti-patterns for TUI plugins"
globs: ["plugins/**", "internal/ui/**", "pkg/sdk/frames/**"]
---

Full spec: `docs/tui-ux.md`

# UX Rules

## Navigation Stack (Android-style)

Plugins use a nested navigation stack. Input always routes to the topmost frame.

```
App Stack: [Home] → [State Plugin]
                      └── Plugin Stack: [List] → [Filter]
                                                → [Inspect] → [Confirm]
```

- Input goes to the deepest leaf frame only
- `esc` always pops the innermost frame (universal "back")
- `q` pops to app root (deactivate plugin)
- `:` side-navigates at app level (replaces plugin)
- Each frame declares its own `Hints() []KeyHint`

SDK types (`pkg/sdk/`):
- `Frame` interface: `ID()`, `Update(msg) (Frame, Cmd)`, `View(w,h)`, `Hints()`
- `Stack`: LIFO container with `Push`, `Pop`, `Update`, `View`, `Hints`
- `Stackable` interface: optional on plugins, returns their internal `*Stack`

Reusable frames (`pkg/sdk/frames/`):
- `FilterFrame`: consumes ALL printable keys; only esc/enter/arrows escape
- `InspectFrame`: scrollable detail + configurable action keys
- `ConfirmFrame`: blocks all input except y/n/esc

Frame lifecycle:
- Return `nil` from `Update` → frame is popped
- Return a different `Frame` → in-place replacement
- Return self → no change

## UX Model (k9s-inspired)

- **`:` command mode**: type plugin name to switch views. Tab autocomplete.
- **`/` filter mode**: fzf-style fuzzy filter. `esc` exits.
- **`space` pin**: toggle pin on selected resource. Pinned = apply/plan target.
- **`enter` / `i` inspect**: show detail view with expanded values.
- **`d` delete**: remove from state — triggers confirmation.
- **`e` edit**: opens $EDITOR at resource's .tf file:line.
- **`m` move**: rename resource address in state.
- **`t` taint**: mark for recreation.
- **`T` untaint**: remove taint mark.
- **`n` import**: import existing resource.
- **`!` batch**: open batch action palette (only when pins > 0).
- **`r` refresh**: reload data from terraform.
- **`ctrl+w` wrap**: toggle line wrapping.
- **`←→` pan**: horizontal scroll (10 chars/press, when wrap is off).

## Keybinding Ergonomics

- Capital letter = non-terraform feature (Context `C`, Risk `R`, Phantom `P`, Blast Radius `B`)
- Lowercase = terraform operation (state `s`, plan `p`, apply `a`, workspaces `w`)
- `ctrl+char` = modifier actions within a view (ctrl+w wrap, ctrl+s screen capture)
- Punctuation = mode/overlay triggers (`/` filter, `!` actions, `?` AI explain, `:` command)

Redundant keybindings for accessibility (hints show only primary):
- Inspect: `Enter` (shown), `i` (alias)
- Back to home: `q` (shown), `esc` when no sub-state (alias)

Rules:
- `enter`/`i` always means inspect — never overloaded
- `space` always means pin — never overloaded
- `q` shown at plugin top-level; `esc` shown only in sub-states
- Plugins must NOT start in filter mode by default

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

## Staleness Guard

Before destructive operations (apply, state rm, state mv, import):
- Threshold: `cache.staleness_threshold` config (default 5m)
- If stale: prompt "State is Xm old. Refresh first? (y/n)"
- If nil: prompt "No state loaded. Load first?"
