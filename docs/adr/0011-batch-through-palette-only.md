# Batch operations only through ! palette

Direct keys (`d`, `t`, `m`, etc.) always act on the cursor item — never on the pinned set. Batch operations are exclusively accessed through the `!` palette. This separation is a UX signal: pressing `!` flags to the user that the danger level has increased.

A single keystroke on a single resource is low-stakes and recoverable. Acting on multiple resources at once is a different cognitive mode — the user should consciously enter it. The `!` gate is that conscious transition.

## Considered options

- **Shift+letter for batch** (e.g., `D` = delete all pinned) — rejected. The shift modifier is too subtle a signal for the escalation in blast radius. Easy to hit accidentally.
- **Implicit batch when pins exist** (e.g., `d` deletes pinned if any, cursor otherwise) — rejected. The same key meaning different things based on hidden state is a recipe for accidents. The user's mental model of "d = delete this one thing" must never be violated.
