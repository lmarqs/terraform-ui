---
allowed-tools: Bash(ls:*), Bash(cat:*), Read
description: Review debug log from last tfui session
---

Find and read the latest debug log:

```bash
ls -t ~/.tfui/logs/debug-*.log | head -1
```

Then read that file and analyze the session:
- What did the user do? (reconstruct the sequence of actions from key presses and plugin activations)
- Were there errors or slow terraform operations? (check durations)
- Any unexpected state transitions?
- Suggestions for UX improvements based on usage patterns
- If backspace/filter issues: check key.press events around filter changes
