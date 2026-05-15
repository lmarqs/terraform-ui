# tfui is a command builder, not a terraform abstraction

tfui helps users construct and execute the right terraform command. It does not abstract terraform away or introduce concepts that don't map to terraform operations. Every action the TUI performs corresponds to a terraform command the user could run manually.

This means: no magic. The user should always be able to understand what terraform command tfui would run on their behalf. In macro mode, this is made explicit — mutations return the exact command. In interactive mode, the mapping is implicit but always 1:1.

A user who learns tfui also learns terraform. The TUI is a better interface to the same tool, not a different tool.

## Consequences

- Every `sdk.Command` records the exact terraform invocation (binary, verb, args, flags, dir)
- Novel features (risk, phantom, blast-radius) are analysis layers on top of terraform output — they don't invoke terraform differently
- Error messages should surface terraform's errors, not reinterpret them
- When terraform's behavior is surprising, tfui makes it visible rather than hiding it
