# terraform-ui — Plan

## Recent Decisions (2026-05-09)

### Interaction Model (k9s-inspired)

- **`:` command mode** (app level): type plugin name to switch views. Tab autocomplete, prefix matching, hint bar shows matches.
- **`/` filter mode** (state browser): auto-focused on entry. Esc-only exit. Arrows navigate, enter inspects directly.
- **`ctrl+w` wrap toggle**: global across list and detail views. When wrap is on, content is not truncated by terminal width. When off, left/right arrows pan horizontally.
- **Fuzzy matching**: char-by-char order match ("rdscluster" → "aws_rds_cluster"). Space separates multiple terms (all must match).

### Keybinding Philosophy

- Normal mode: arrows, enter, esc, `/`, `:`, `r`, `g`, `G`, `w`
- Filter mode: all keys type into filter except esc/enter/arrows
- `esc` exits current stack level (filter → normal → home)
- `q` always exits to home (app level)
- `:` always opens command input (app level)
- `ctrl+w` toggles wrap from any mode

### State Browser Detail View

- Enter → instant inspect (state cached from initial StateList load)
- ↑↓ vertical scroll, ←→ horizontal pan (10 chars/press)
- `ctrl+w` toggles wrap, `w` also works in normal/detail mode
- Fixed header (title + address + scroll position)

### State Caching

- `TerraformService.stateCache` holds parsed state from `StateList()`
- `Show(address)` reuses cache (0ms vs ~35s re-pull)
- Cache invalidated on refresh (`r` key)

### Context Auto-Discovery

- No `context.paths` in tfui.yaml → recursively scans for terraform subdirs
- Skips hidden dirs, stops at dirs with `.tf` files
- If 2+ projects found, offers context picker

## TODO

### Polish (pre-release)

| Task | Priority | Notes |
|------|----------|-------|
| Refactor detail view to `bubbles/viewport` | High | Native scroll/wrap/pan handling |
| Help overlay (`?` key) | Medium | Show keybindings for current mode |
| Update CLAUDE.md for Go structure | Low | Current doc describes bash version |

### Post-v1.0

| Task | Description |
|------|-------------|
| `drift` plugin | Periodic drift detection |
| `cost` plugin | Infracost integration |
| `import` plugin | Interactive resource import wizard |
| `graph` plugin | Module dependency graph visualization |
| `history` plugin | Local operation history (SQLite) |
| External plugins | Third-party plugins via hashicorp/go-plugin (gRPC) |
