# UX Guidelines — terraform-ui

## 1. Layout Structure

```
 Project: ../medprev-cloud-iac                                     ╔╦╗╔═╗╦ ╦╦
 Scope: modules/sa-east-1                                          ║ ╠╣ ║ ║║
 Workspace: default                                                 ╩ ╚  ╚═╝╩
┌────────────────────────────────────────────────────────────────────────────┐
│ :context                                                                   │
└────────────────────────────────────────────────────────────────────────────┘
┌────────────────────── State Browser (30/1549) ─────────────────────────────┐
│ content...                                                                 │
└────────────────────────────────────────────────────────────────────────────┘
 ↑↓ navigate  Enter expand/inspect  Space pin  / filter  ^t flat  q back    terraform
```

- **Header** (3 lines): left=Project/Scope/Workspace, right=ASCII logo. Always visible.
- **Command bar**: bordered `:` input, visible only when active.
- **Content**: bordered box, view title + count embedded in top border.
- **Footer**: single hint line (left), binary name right-aligned faint.
- **No separators** — borders handle visual separation.

## 2. Information Architecture

| Location | Content |
|----------|---------|
| Header left | Project (+ pinned count), Scope, Workspace |
| Header right | ASCII logo (brand identity) |
| Content border title | View name + filtered/total count |
| Footer | Context-sensitive key hints (from frame's Hints()) |
| Command bar | `:` input with autocomplete matches |

## 3. Keybinding Conventions

### Global keys (work everywhere)
| Key | Action |
|-----|--------|
| `q` | Back to home / quit |
| `:` | Command mode |
| `/` | Filter mode |
| `C` | Context picker overlay |
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
| `Enter` | Expand branch / inspect leaf |
| `Space` | Pin/unpin |

### Tree-specific
| Key | Action |
|-----|--------|
| `[` | Collapse all |
| `]` | Expand all |
| `Enter` on branch | Toggle expand/collapse |
| `Enter` on leaf | Inspect |

### Plugin activation (home screen only)
Single plain letter: `s` (state), `p` (plan), `a` (apply), `w` (workspaces), `o` (outputs), `v` (validate), `t` (console)

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

## 5. State Transitions

- **Loading**: show elapsed time for operations > 2s
- **Error**: floating modal overlay (not inline text)
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
- Used for: scope picker, error display, confirmations

## 9. Home Screen

- Shows after scope selection (or when no plugin active)
- Plugin list sorted by workflow: State, Plan, Apply, Workspaces, Outputs, Validate, Console, then decorators (Risk, Phantom, Blast Radius, Init)
- Current scope visible in header
- Direct key activation (press letter) or j/k + Enter
- Context plugin in the list (accessed via `C` or `:context`) — manages Project + Scope + Workspace

## 10. Color Palette

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

## 11. Plugin View Contract

Every plugin's `View(width, height)` must:
- NOT include its own title (title goes in the content border)
- NOT include hint text (hints come from frame's `Hints()` method)
- NOT add padding (the bordered box handles spacing)
- Return pure content that fills the available space
- Handle empty state gracefully (show informative placeholder)

## 12. Performance

- Virtual scrolling: only render visible rows (viewport window)
- Tree flatten: O(visible nodes), not O(all nodes)
- Filter: debounce unnecessary on keystroke (fzf is fast enough for 1549 items)
- State load: show elapsed time indicator after 2s
- Context discovery: cache results, don't re-discover on every overlay open
