---
layout: default
title: TUI UX Guidelines
parent: Reference
nav_order: 5
description: UX design guidelines for the TUI interface
---

# UX Guidelines — terraform-ui

## 1. Layout Structure

```
 Project: ../medprev-cloud-iac                                     ╔╦╗╔═╗╦ ╦╦
 Chdir: modules/sa-east-1                                           ║ ╠╣ ║ ║║
 Workspace: default                                                 ╩ ╚  ╚═╝╩
┌────────────────────────────────────────────────────────────────────────────┐
│ :context                                                                   │
└────────────────────────────────────────────────────────────────────────────┘
┌────────────────────── State Browser (30/1549) ─────────────────────────────┐
│ content...                                                                 │
└────────────────────────────────────────────────────────────────────────────┘
 Enter inspect  / filter  Space pin  ^t flat  d delete  e edit  t taint  T untaint  q back    terraform
```

- **Header** (3 lines): left=Project/Chdir/Workspace, right=ASCII logo. Always visible.
- **Command bar**: bordered `:` input, visible only when active.
- **Content**: bordered box, view title + count embedded in top border.
- **Footer**: single hint line (left), binary name right-aligned faint.
- **No separators** — borders handle visual separation.

## 1b. Standalone Mode Layout

When invoked as `tfui <command>` (e.g., `tfui plan`, `tfui state`), the TUI runs in standalone mode with minimal chrome:

```
 my-infra │ modules/sa-east-1 │ production                              tfui
┌─────────────────────────────────────────────────────────────────────────────┐
│ plugin content fills the screen                                              │
└─────────────────────────────────────────────────────────────────────────────┘
 Enter inspect  / filter  Space pin  d delete  e edit  q quit                terraform
```

### Differences from full TUI

| Aspect | Full TUI | Standalone |
|--------|----------|------------|
| Header | 3-line (project/chdir/workspace + ASCII logo) | 1-line (project/chdir/workspace left, tfui right) |
| Border | Content border with title | No border |
| Navigation | `:` command mode, home screen, plugin switching | Disabled |
| `q` key | Go home (or quit from home) | Quit app |
| `esc` key | Pop frame / deactivate | Pop frame / quit when at root |
| NavPush | Works (plan→apply, state→taint) | Works identically |
| Output | None | stdout on exit (via `Outputter` interface) |

### Standalone rendering

- TUI renders on **stderr** (via `tea.WithOutput(os.Stderr)`)
- stdout stays clean for pipe output
- On quit: plugin's `Output(json)` is written to stdout
- Alt-screen on stderr: terminal restores on exit

### Standalone navigation rules

- `:` command mode: disabled (no inter-plugin navigation)
- `C` (context switch): disabled
- `q`: clears sub-frames first, then quits
- `DeactivateMsg` with empty navStack: quits the app
- `NavigateMsg` to non-NavPush plugins: rejected
- NavPush sub-navigation works normally (plan can push apply, state can push taint)

## 2. Information Architecture

| Location | Content |
|----------|---------|
| Header left | Project (+ pinned count), Chdir, Workspace |
| Header right | ASCII logo (brand identity) |
| Content border title | View name + filtered/total count |
| Footer | Context-sensitive key hints (from frame's Hints()) |
| Command bar | `:` input with autocomplete matches |

## 3. Keybinding Conventions

### Global keys (work everywhere)
| Key | Action |
|-----|--------|
| `q` | Back to home / quit |
| `:` | Command mode (`:q` quit — guarded during ops, `:q!` force quit) |
| `/` | Filter mode |
| `C` | Context dashboard (plugin switch) |
| `Esc` | Exit sub-state (filter, detail, overlay) |
| `ctrl+c` | Force quit |
| `ctrl+s` | Screen capture to debug log (hidden) |

### ctrl+char for action shortcuts
| Key | Action |
|-----|--------|
| `ctrl+t` | Toggle tree/flat view |
| `ctrl+w` | Toggle line wrap |

### Navigation
| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` | Jump to start |
| `G` | Jump to end |
| `Enter` | Inspect (leaf) / expand (branch) |
| `Space` | Pin/unpin |

### Tree-specific
| Key | Action |
|-----|--------|
| `[` | Collapse all |
| `]` | Expand all |
| `Enter` on branch | Toggle expand/collapse |
| `Enter` on leaf | Inspect |

### Plugin activation (home screen only)
Single plain letter: `s` (state), `p` (plan), `a` (apply), `w` (workspace), `o` (outputs), `v` (validate), `~` (console), `i` (init)

## 4. Visual Patterns

### Borders
- Content: `lipgloss.RoundedBorder()` with primary blue foreground
- Title embedded in top border line (manual construction)
- Overlays: rounded border, centered via `lipgloss.Place()`

### Tree connectors (2-char width)
```
├─  (branch, has siblings after)
└─  (last child)
│   (ancestor continuation)
```

### Indicators
| Symbol | Meaning |
|--------|---------|
| `*` (green) | Pinned |
| `▶` | Collapsed branch |
| `▼` | Expanded branch |
| `█` | Cursor in input field |

### Text styles
| Style | Usage |
|-------|-------|
| Bold + primary color | Titles, active keys |
| Faint (gray 241) | Secondary info (types, counts, hints) |
| Italic + faint | Placeholder text, loading messages |
| Error (red) | Error messages |
| Success (green) | Pin indicators, confirmations |
| Background (236) | Selected row highlight, header/footer bg |

## 5. State Transitions & Loading

### Core Principle: Honest Feedback

The user must always know what the app is doing. Every state the app enters must be visible, explicit, and traceable to a user action. No invisible work, no silent failures, no background magic.

**The rule: if the app is doing something, the user sees it. If the user didn't ask for it, the app doesn't do it.**

### Loading States

When the user triggers an async operation (opens a plugin, selects a field that requires I/O, runs a command), the app transitions through:

```
User Action → Loading → Done (or Error)
```

The user sees every step. There is no shortcut.

#### Visual Treatment

| Duration | What the user sees |
|----------|-------------------|
| Immediate | Faint italic text: `"Loading workspaces..."`, `"Running terraform plan..."` |
| After 2s | Append elapsed time: `"Running terraform plan... 4s"` |

Rendered with `sdk.StyleFaintItalic` — visually distinct from content, clearly transient.

#### Hint Bar During Loading

Only show actions that work during loading:

```go
// Loading state — only back is available
[]sdk.KeyHint{
    sdk.HintBack,
}
```

Never show `↑↓ navigate`, `Enter inspect`, or other content-dependent hints during loading — there's nothing to navigate or inspect yet. Showing unavailable actions is lying to the user.

#### State Lifecycle

Every plugin that does async work follows this lifecycle:

```go
StatusIdle    → user hasn't activated yet
StatusLoading → async operation in flight (user sees loading text)
StatusDone    → data available (user sees content)
StatusError   → operation failed (user sees error)
```

Transitions are always user-initiated:
- `Idle → Loading`: user activates the plugin or triggers a command
- `Loading → Done`: async response arrives
- `Loading → Error`: async response is an error
- `Error → Loading`: user presses `ctrl+r` to retry
- `Done → Loading`: user presses `ctrl+r` to refresh

The app NEVER transitions out of `Idle` on its own. The app NEVER refreshes without the user asking.

### Error States

- Floating modal overlay with error detail (not inline text that might be missed)
- Hints: `r retry` + `esc back` (+ `u force-unlock` if applicable)
- Error message is the raw terraform output — don't sanitize or summarize

### Other Transitions

- **Context required**: modal overlay prompting selection
- **Destructive ops**: confirmation modal with summary of what will change
- **Stale data**: prompt before destructive operations on data older than threshold

## 6. Filtering

| Aspect | Flat mode | Tree mode |
|--------|-----------|-----------|
| Algorithm | fzf FuzzyMatchV2 | fzf with score threshold (len*17) |
| Sorting | By score (best first) | Preserves hierarchy order |
| Input chars | Alphanumeric + `_` `-` `.` | Same |
| Space key | Pin selected item | Pin selected item |
| Results | Flat list | Auto-expanded tree showing matches |
| Toggle | `ctrl+t` switches to tree | `ctrl+t` switches to flat |

### Filter rules
- Only accept: `a-z A-Z 0-9 _ - .`
- Brackets, special chars pass through to action handlers
- `Esc` exits filter mode
- `Enter` inspects (leaf) or expands (branch)
- Backspace on empty filter: stays in filter mode (doesn't exit)
- Filter bar renders at the **top** of the list (above items, below any header)

## 7. Tree Navigation

- Default view: **flat** (fzf ranked list)
- `ctrl+t`: toggle to **tree** (hierarchical with expand/collapse)
- Tree builds from `SplitTerraform()` which splits on `module.X` boundaries
- Pinned items/groups float to top within their sibling level
- Filter in tree mode auto-expands all branches to reveal matches

## 8. Modal/Overlay Pattern

- Render: centered box with `lipgloss.RoundedBorder()` via `lipgloss.Place()`
- Full screen replacement while active (content not visible behind)
- Captures ALL input — nothing passes through to plugins
- `Esc` or completion action dismisses
- Status bar shows overlay-specific hints
- Used for: chdir picker, error display, confirmations

## 9. Picker Frame Pattern

A picker is an inline selection frame pushed onto a plugin's own stack. It replaces navigating away to a separate plugin — the user stays in context, `esc` pops back, and selection auto-pops to the parent frame.

### Core Principle: No Magic Behavior

Every action the app takes must be a direct, visible result of something the user did. The app never fetches data speculatively, never pre-loads behind the user's back, never does work the user didn't ask for.

**Magic behavior** = the app doing something without the user triggering it. Examples:
- Fetching workspace list on `Activate()` before the user selects the workspace field
- Pre-loading data "in case" the user might need it
- Background refreshes that silently update cached state
- Invisible network calls that consume resources and may fail silently

**Explicit behavior** = every side effect traces back to a user action. Examples:
- User selects "Workspace" → app fetches workspace list → shows loading → shows picker
- User presses `ctrl+r` → app refreshes data
- User opens a plugin → app loads that plugin's data (because they asked for it)

This means a picker with async items shows a loading state — that is honest UX. The user triggered the action, they see the consequence. A brief "Loading..." is infinitely better than hidden background magic that may silently fail, waste resources, or produce stale data the user never asked for.

### Two Picker Flavors

| Flavor | Items source | On select | User sees |
|--------|-------------|-----------|-----------|
| **Sync** | Already in memory (config, static list) | Push picker frame immediately | Instant list |
| **Async** | Requires I/O (terraform CLI call) | Kick fetch → loading text → push picker on response | `"Loading..."` → list |

Both share the same picker frame once items are available. The difference is only in _when_ the fetch happens — always as a direct result of the user's action, never speculatively.

For async pickers, the loading state follows §5 exactly:
- Form field value changes to faint italic loading text (e.g., `"Loading workspaces..."`)
- Hint bar reduces to `esc back` only
- Picker frame pushes when data arrives
- On error: field shows error, `r` retries

### Behavior Contract

| Input | Action |
|-------|--------|
| `↑↓` / `j/k` | Move cursor |
| `g` / `G` | Jump to start/end |
| `Enter` | Confirm selection → emit event → auto-pop frame |
| `esc` | Cancel → pop frame (no event emitted) |
| `q` | Same as esc |

### Visual Design

```
▸ modules/vpc          ← cursor (StyleSelected)
  modules/ecs
  modules/rds
```

- `▸` prefix on cursor item, rendered with `StyleSelected`
- Two-space indent on non-cursor items
- Viewport windowed via `Cursor.VisibleWindow(height)`
- No title inside the frame (the content border shows the parent plugin name)

### Hints

```go
[]sdk.KeyHint{
    {Key: "↑↓", Description: "navigate"},
    {Key: "Enter", Description: "select"},
    {Key: "esc", Description: "back"},
}
```

### Implementation

```go
// Sync picker — items already available:
func (p *Plugin) openChdirPicker() tea.Cmd {
    frame := newPickerFrame(p.members, p.current, func(selected string) tea.Cmd {
        // emit event, update state
    })
    p.stack.Push(frame)
    return nil
}

// Async picker — items require I/O:
func (p *Plugin) openWorkspacePicker() tea.Cmd {
    svc := p.svc
    return func() tea.Msg {
        // User triggered this — fetch is a direct consequence
        items, err := svc.WorkspaceList(context.Background())
        return workspaceListMsg{items: items, err: err}
    }
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
    switch msg := msg.(type) {
    case workspaceListMsg:
        if msg.err != nil { /* show error */ }
        frame := newPickerFrame(msg.items, p.current, p.onSelect)
        p.stack.Push(frame)
    }
    return p, nil
}
```

### Anti-patterns

- **Background pre-loading** — fetching data the user didn't ask for. Magic behavior: invisible side effects, wasted resources, stale data. Load when the user acts, not before.
- **Navigate away to a different plugin** — breaks back navigation, loses parent context
- **Block in OnSelect callback** — the callback returns a `tea.Cmd`, never blocks the UI thread
- **Speculative caching** — caching "in case" the user opens a picker. Only cache if the user has already seen the data and it's still valid.

## 10. Home Screen

- Shows after chdir selection (or when no plugin active)
- Plugin list sorted by workflow: State, Plan, Apply, Workspaces, Outputs, Validate, Console, then decorators (Risk, Phantom, Blast Radius, Scaffold)
- Current chdir visible in header
- Direct key activation (press letter) or j/k + Enter
- Context plugin in the list (accessed via `C` or `:context`) — manages Project + Chdir + Workspace

## 11. Color Palette

| Name | Value | Usage |
|------|-------|-------|
| Primary | Color 39 (blue) | Borders, keys, active elements |
| Text | Color 252 (light) | Main content |
| Faint | Color 241 (gray) | Secondary info |
| Background | Color 236 (dark gray) | Header/footer bg, selected rows |
| Success | Green | Pins, confirmations |
| Error | Red | Error text |
| Warning | Yellow | Risk indicators |
| Create | Green | Plan: resources to add |
| Update | Yellow | Plan: resources to change |
| Delete | Red | Plan: resources to destroy |
| Replace | Magenta | Plan: resources to replace |

## 12. Plugin View Contract

Every plugin's `View(width, height)` must:
- NOT include its own title (title goes in the content border)
- NOT add padding (the bordered box handles spacing)
- Return pure content that fills the available space
- Handle empty state gracefully (show informative placeholder)
- MAY include plugin-specific contextual hints (see §13)

## 13. Hint Placement Rules

### Two layers

| Layer | Location | Content | Source |
|-------|----------|---------|--------|
| **Hint bar** | Footer (status bar) | Standard keys that work in current state | `Frame.Hints()` (Stackable) or `Plugin.Hints()` (Hintable) |
| **Inline hints** | Inside plugin view | Plugin-specific contextual keys not in standard vocabulary | Hardcoded in `View()` near relevant content |

### Hint bar rules
- Shows standard `HintSet` vocabulary: navigate, inspect, pin, filter, back, retry, etc.
- MUST be state-aware: return different hints per plugin status
- Loading state: only `q back`
- Error state: `r retry` + `q back` (+ `u force-unlock` if locked)
- Done state: full navigation set for that plugin
- Never show keys that don't work in the current state

### Inline hint rules
- ONLY for plugin-specific keys with no `HintSet` equivalent
- Examples: `u force-unlock` (near lock info), `Space toggle` (in pattern selection)
- NEVER duplicate the hint bar (no `q back`, `r retry`, `↑↓ navigate` inline)
- Format: terse `key action` separated by double-space, styled with `sdk.StyleFaintItalic`
- Position: near the UI element they act on (proximity = comprehension)
- Dangerous/narrow-scope actions belong inline (visual friction is intentional)

### View content (NOT hints)
- State messages: "Loading terraform state...", "Running terraform plan..."
- Error detail: "Error: connection refused"
- Guidance: "Run plan first to analyze risk..."
- NEVER mix keybinding text with content ("Press r to retry" is wrong)

### Interface implementation
- Stackable plugins: hints come from `Frame.Hints()` on the active frame
- Non-stackable plugins: implement `Hintable` interface with `Hints() []KeyHint`
- Both must return state-appropriate hints (check plugin status in the method)

## 14. Performance

- Virtual scrolling: only render visible rows (viewport window)
- Tree flatten: O(visible nodes), not O(all nodes)
- Filter: debounce unnecessary on keystroke (fzf is fast enough for 1549 items)
- State load: show elapsed time indicator after 2s
- Context discovery: cache results, don't re-discover on every overlay open

## 15. Plugin UX Models

Plugins fall into two archetypes based on what the user is trying to accomplish. The archetype determines the flow, the "home state," and how results are presented.

### Data plugins (browse & act)

**Intent:** The user wants to explore data and optionally act on items.

**Home state:** A list/tree of items (always present after loading).

**Examples:** state, plan, output, validate, risk, phantom, blast-radius

**Flow:**
```
Activate → Loading → List/Tree (home state)
  ├── ↑↓ navigate, / filter, Enter inspect, Space pin
  ├── Action key (d, t, m, n) → confirmation → execute → refresh list
  └── ctrl+r → reload data
```

**Characteristics:**
- The user lingers here — this is a workspace
- Results (the list) ARE the home state
- Actions are targeted (cursor item or pinned set via `!`)
- After an action completes, stay on the list (updated)

### Action plugins (configure & execute)

**Intent:** The user wants to perform a one-shot operation.

**Home state:** A configuration form (the thing the user tweaks before executing).

**Examples:** init, apply (when entered from plan)

**Flow:**
```
Activate → Form (home state)
  ├── Configure fields
  └── Submit → Loading → Result (transient)
       ├── Success → auto-return to origin (one-shot is done)
       └── Error → Enter acknowledges → back to form (pre-filled for retry)
```

**Characteristics:**
- The user doesn't linger — they configure, execute, and leave
- Results are transient feedback, not a destination
- Success means the user's intent is satisfied — don't force them to dismiss a "done" screen
- Error means the user needs to fix something — return them to the form with context preserved
- `Enter` on error = "I've seen this, let me fix it" (acknowledge semantics, consistent with confirm)

### Choosing the archetype

| Question | Data plugin | Action plugin |
|----------|-------------|---------------|
| Does the user return here to browse? | Yes | No |
| Is the result the content itself? | Yes | No (result is feedback) |
| Does the user act repeatedly? | Yes (on different items) | No (one-shot, then leave) |
| After success, should user stay? | Yes (more items to explore) | No (intent satisfied) |

### UX anti-patterns for action plugins

- Lingering on a "success" screen (user must dismiss manually)
- "Re-run" as a concept (if they want to re-run, they'll re-enter the plugin)
- Showing content-navigation hints (↑↓, /, Enter inspect) on a result message
- Using `ctrl+r refresh` when there's nothing to refresh (init has no data to reload)

## 16. Keybinding Reference Map

### Design layers

| Layer | Keys | Scope | Examples |
|-------|------|-------|---------|
| Terraform verbs | bare lowercase | cursor resource | `d` delete, `t` taint, `e` edit, `m` move, `n` import, `u` force-unlock, `a` apply, `A` auto-approve |
| Plugin switches (home) | bare lowercase | home screen only | `s` state, `p` plan, `w` workspace, `o` output, `v` validate, `~` console, `i` init |
| Plugin switches (global) | bare uppercase | anywhere | `C` context, `R` risk, `P` phantom, `B` blast radius |
| Interface controls | ctrl+key | view modes, reload | `ctrl+t` tree, `ctrl+w` wrap, `ctrl+r` refresh, `ctrl+p` pinned, `ctrl+u` unpin all |
| Navigation & modes | non-alpha / punctuation | navigation, mode triggers | `/` filter, `!` batch, `[` collapse, `]` expand, `:` command, `Space` pin, `Enter` inspect |
| Leave | `q` / `Esc` | universal | `q` home, `Esc` pop sub-state |

### Bare lowercase (a-z)

| Key | Assignment | Category | Context |
|-----|-----------|----------|---------|
| `a` | apply | terraform verb / plugin switch | plan view: apply; home: activate apply |
| `b` | — | **free** | |
| `c` | — | **free** | |
| `d` | delete | terraform verb | state, workspace |
| `e` | edit | terraform verb | state, plan (opens $EDITOR) |
| `f` | — | **free** | |
| `g` | jump to start | navigation | lists, trees |
| `h` | — | **free** | |
| `i` | init | terraform verb / plugin switch | home: activate init |
| `j` | move down | navigation (vim alias) | ↓ is primary |
| `k` | move up | navigation (vim alias) | ↑ is primary |
| `l` | — | **free** | |
| `m` | move (rename) | terraform verb | state |
| `n` | new / import | terraform verb | state: import (navigates to import plugin); workspace: new |
| `o` | output | plugin switch | home |
| `p` | plan | plugin switch | home |
| `q` | back / home | navigation | global |
| `r` | — | **free** | note: `ctrl+r` is refresh |
| `s` | state | plugin switch | home |
| `t` | taint | terraform verb | state/plan: navigates to taint plugin |
| `u` | force-unlock | terraform verb | plan, state (only when locked) |
| `v` | validate | plugin switch | home |
| `w` | workspace | plugin switch | home |
| `x` | — | **free** | |
| `y` | confirm (yes) | confirmation | confirmation frames only |
| `z` | — | **free** | |

### Bare uppercase (A-Z)

| Key | Assignment | Category | Context |
|-----|-----------|----------|---------|
| `B` | blast radius | plugin switch (global) | decorator |
| `C` | context | plugin switch (global) | always available |
| `G` | jump to end | navigation | lists, trees |
| `P` | phantom | plugin switch (global) | decorator |
| `R` | risk | plugin switch (global) | decorator |
| `A` | auto-approve | terraform verb | plan: apply without confirmation |
| `T` | untaint | terraform verb | state/plan: navigates to untaint plugin |
| `Y` | confirm (yes) | confirmation | confirmation frames only |
| All others | — | **free** | |

### ctrl+key

| Key | Assignment | Category |
|-----|-----------|----------|
| `ctrl+c` | force quit | emergency exit |
| `ctrl+h` | backspace | input (emacs convention) |
| `ctrl+p` | pinned filter toggle | view mode |
| `ctrl+r` | refresh / retry | reload data |
| `ctrl+s` | screen capture | debug (hidden) |
| `ctrl+t` | tree/flat toggle | view mode |
| `ctrl+u` | unpin all | pin management |
| `ctrl+w` | line wrap toggle | view mode |

### Punctuation & special

| Key | Assignment | Category |
|-----|-----------|----------|
| `:` | command mode | mode switch |
| `/` | filter mode | mode switch |
| `!` | batch actions | mode switch (only when pins > 0) |
| `[` | collapse all | tree action |
| `]` | expand all | tree action |
| `Space` | pin / unpin | pin toggle |
| `Enter` | inspect / confirm | navigation |
| `Esc` | exit / cancel / pop | navigation |
| `←` `→` | horizontal pan | scroll (when wrap off) |

### Aliases (redundant bindings for accessibility)

| Primary | Alias | Rationale |
|---------|-------|-----------|
| `↓` | `j` | vim convention |
| `↑` | `k` | vim convention |
| `Enter` | `y`/`Y` | explicit "yes" in confirmations |

### Rules

- `Enter` always means inspect or confirm — never overloaded
- `Space` always means pin — never overloaded
- Aliases are never shown in hints (only primary is shown)
- Free keys should not be claimed without updating this map
