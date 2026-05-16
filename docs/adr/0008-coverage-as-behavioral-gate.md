---
layout: default
title: "ADR-0008: 100% coverage as a behavioral forcing function"
parent: Architecture
nav_order: 0008
---

# 100% coverage as a behavioral forcing function

The pipeline enforces 100% code coverage on all packages (excluding `cmd/` glue). But tests must describe user-visible behavior — never written "for coverage." These two rules create deliberate tension: every branch must justify its existence through a behavioral test.

If a branch can't be reached by a behavioral test, the branch shouldn't exist. This eliminates speculative code: defensive guards for impossible states, error handling for conditions that can't occur, "just in case" branches. The coverage gate forces authors to think carefully about each branch — if you can't describe what the user experiences when this path executes, remove the path.

Coverage is a side-effect of complete behavioral specification. One test suite serves both purposes. There is no separation between "behavior tests" and "coverage tests."

## Consequences

- No unreachable defensive code. If a dependency can't fail in practice, don't handle its error.
- No speculative branches. Code only exists if a user-visible behavior requires it.
- When coverage drops, the question is "which behavior is untested?" not "which line is uncovered."
