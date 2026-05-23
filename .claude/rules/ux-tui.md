---
description: "TUI UX: keybinding conventions, hint bar design, navigation, action model, and anti-patterns"
globs: ["plugins/**", "internal/ui/**", "pkg/sdk/frames/**"]
---

Full spec: `docs/reference/tui-ux.md`

# TUI UX Rules

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
  ├── User selects → ContextChangedEvent → popIfPushed → back to state
  ├── User presses esc → DeactivateMsg → returnTo exists → back to state
  └── User presses q → global handler → home (returnTo cleared)
```

## Keybinding Rules

Two layers — bare alpha keys talk to terraform, everything else talks to the interface:

- Bare lowercase = terraform mutation on cursor resource
- Bare uppercase = non-terraform plugin switch
- `ctrl+key` = interface control (view modes and reload)
- Non-alpha / punctuation = navigation and mode triggers
- `q` / `Esc` = leave (universal TUI convention, exception to alpha rule)

Invariants:
- `enter` always means inspect — never overloaded
- `space` always means pin — never overloaded
- `q` shown at plugin top-level; `esc` shown only in sub-states
- Plugins must NOT start in filter mode by default

Full keybinding map and assignments: `docs/reference/tui-ux.md` §7 and §16

## Action Model (cursor vs batch)

**CRITICAL: direct keys always act on the cursor item. Batch operations go through `!` palette only.**

| Layer | Keys | Location | Scope |
|-------|------|----------|-------|
| Navigate | `Enter`, `/`, `Space`, `^t`, `^r`, `q`, `Esc` | Hint bar (outside border) | — |
| Act (single) | `d`, `e`, `t`, `T`, `m`, `n` | Actions bar (inside frame) | Cursor |
| Batch | `!` (palette) | Actions bar (inside frame) | Pinned set |

- Direct action keys NEVER read the pinned set
- `!` is hidden when no pins exist
- Detail/inspect frame has no `!` (single-item actions only)
- Destructive batch ops always show confirmation with count

Pin semantics:
- PRIMARY: scoping `plan` and `apply` to specific resources
- SECONDARY: enabling batch state actions via `!` palette
- Pins are persistent (survive view switches and sessions)

## Actions Bar (inside bordered frame)

Terraform mutation keys live inside the plugin frame as two-tone button chips (bold white key on purple `#bd93f9`, label on muted purple `#644e84`).

- SDK rendering primitive — plugins own when/where to show it
- Pinned to bottom of available frame space, blank line separator above
- Left-aligned, slim: tight background, single space between buttons
- Static per frame — does NOT change based on cursor position
- Contains: `d`, `e`, `t`, `T`, `a`, `A`, `m`, `n`, `u`, `!` (bare alpha + `!`)
- `!` appears only when pins > 0
- Not rendered when plugin has no terraform actions
- In inspect frame: shows single-item actions for the inspected resource

## Hint Bar (outside border, footer)

UI/navigation keys only. Always 1 line.

- Contains: `Enter`, `/`, `Space`, `^t`, `^r`, `^w`, `^p`, `^u`, `:`, `q`, `Esc`
- Does NOT show `↑↓ navigate` (scroll indicators teach this)
- Context-sensitive: content changes per frame/state
- No novel notation (no `d/D`, no `(t/T) taint/untaint`)
- Dynamic hints show target state: `^t tree` means "press to enter tree mode"

## The Split Rule

**Bare key = actions bar. Ctrl+key or punctuation = hint bar.** The modifier is the visual signal.

## Scroll Indicators

- Scrollbar gutter: `▲` top cap, `┃` thumb, `│` track, `▼` bottom cap
- Only appears when content overflows the viewport
- Spans content rows only (not blank separator or actions bar)
- Lines padded to consistent width before gutter chars (prevents misalignment with styled rows)
- `[cursor/navigable]` counter always visible in title bar
- `(filtered/total)` count also in title bar
- Counter reflects navigable items (respects collapse state and filter)

## UX Anti-patterns (do NOT introduce)

- Shift+letter = batch version of same action
- Implicit batch based on pin state
- Novel hint bar notation (slash grouping, parenthetical pairs)
- 2-line hint bars
- Auto-batch for non-destructive actions
- Terraform action keys in the hint bar (they belong in the actions bar)
- UI/navigation keys in the actions bar (they belong in the hint bar)
- Cursor-reactive actions bar (content must be static per frame)

## Detail/Inspect View

- Shows expanded attribute values (JSON)
- Actions bar shows single-item mutations: `d` delete, `e` edit, `t` taint, `T` untaint
- Hint bar shows UI controls: `^w` wrap, `Space` pin, `Esc` back
- Scroll gutter appears when content overflows
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
