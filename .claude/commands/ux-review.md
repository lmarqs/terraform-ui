Review the UI/UX of the current state of the TUI application. Act as a senior UX engineer who uses k9s, lazygit, and other terminal UIs daily.

Analyze:

1. **Layout consistency** — Read `internal/ui/app.go`, `internal/ui/components/*.go`. Check header/content/footer spacing, separator usage, alignment.

2. **Keybinding coherence** — Read all plugin `handleKey` methods. Check:
   - Same keys do the same thing across plugins (esc, q, /, space, enter, r)
   - No conflicts between plugin keys and global keys
   - Hint text at the bottom matches actual bindings

3. **State transitions** — Check that every plugin handles: Idle → Loading → Done/Error correctly. Check that error states offer escape routes (esc, r to retry).

4. **Information density** — Check Views for wasted space, missing context info, truncation issues.

5. **Feedback loops** — After destructive actions (delete, apply), is there clear feedback? Loading indicators?

6. **Debug log review** — Read the latest debug log (`ls -t ~/.tfui/logs/debug-*.log | head -1`) to see real usage patterns and friction points.

Report findings as a prioritized list:
- 🔴 Breaking UX issues (user gets stuck, data lost, no feedback)
- 🟡 Friction (unnecessary steps, inconsistency, confusion)
- 🟢 Polish (nice-to-have improvements)

For each issue, specify the file and line, and suggest the fix.
