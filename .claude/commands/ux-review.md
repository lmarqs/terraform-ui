Review the UX of the current state of the application across BOTH surfaces (TUI and CLI). Act as a senior UX engineer who uses k9s, lazygit, ripgrep, and gh daily.

## TUI Surface

Analyze using `docs/reference/tui-ux.md` and `.claude/rules/ux-tui.md`:

1. **Layout consistency** — Read `internal/ui/app.go`, `internal/ui/components/*.go`. Check header/content/footer spacing, separator usage, alignment.

2. **Keybinding coherence** — Read all plugin `handleKey` methods. Check:
   - Same keys do the same thing across plugins (esc, q, /, space, enter, r)
   - No conflicts between plugin keys and global keys
   - Hint text at the bottom matches actual bindings

3. **State transitions** — Check that every plugin handles: Idle → Loading → Done/Error correctly. Check that error states offer escape routes (esc, r to retry).

4. **Information density** — Check Views for wasted space, missing context info, truncation issues.

5. **Feedback loops** — After destructive actions (delete, apply), is there clear feedback? Loading indicators?

## CLI Surface

Analyze using `docs/reference/cli-ux.md`, `docs/reference/cli-io-contract.md`, and `.claude/rules/ux-cli.md`:

6. **Output channel compliance** — Verify data→stdout, progress→stderr separation in all command handlers.

7. **Spinner logic** — Verify the three-condition gate: `!ci && !jsonOutput && isStderrTTY()`.

8. **Flag consistency** — Check flag normalization sets match documentation. Verify novel flags use double-dash.

9. **Pipe safety** — Verify stdout is clean (no ANSI) when piped. Check `-json` produces zero stderr bytes.

10. **Exit code correctness** — Verify `os.Exit(2)` only in plan paths. Check error paths exit non-zero.

## Cross-Surface Coherence

11. **Mental model alignment** — Same operation should feel consistent whether invoked from TUI keybinding or CLI command (e.g., `tfui plan` CLI vs `p` key in TUI produce equivalent information).

12. **Error experience** — CLI errors and TUI error states convey the same information and suggest the same recovery paths.

## Debug Log

Read the latest debug log (`ls -t ~/.tfui/logs/debug-*.log | head -1`) to see real usage patterns and friction points.

## Report Format

Report findings as a prioritized list:

- **Critical** — User gets stuck, data lost, I/O contract broken, pipe safety violated
- **Warning** — Inconsistency between surfaces, missing feedback, friction
- **Info** — Polish, alignment, minor improvements

For each issue, specify the file and line, which surface it affects (TUI/CLI/both), and suggest the fix.
