---
allowed-tools: Bash(ls:*), Bash(cat:*), Read
description: Review a recorded tfui session (frames + tape)
---

The user will provide a recording directory path (produced by `--record`).

Read `manifest.json` to understand timing and dimensions. Read `recording.tape` (if present) to see what the user did. Read key frames to see what was rendered.

Analyze the session:
- What did the user do? (reconstruct from tape commands or frame progression)
- Were there unexpected UI states? (check frame content for errors, empty views, or broken layouts)
- How long between interactions? (check delay_ms in manifest — long gaps may indicate confusion)
- Any navigation dead-ends? (esc/q appearing frequently)
- Suggestions for UX improvements based on the flow

Usage: `/record-review <path-to-recording-dir>`
