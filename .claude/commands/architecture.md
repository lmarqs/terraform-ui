---
allowed-tools: Read, Bash(grep:*), Bash(wc:*)
description: Show project architecture overview
---

## Architecture Summary

Read and summarize the current architecture:

- `lib/tfui.sh` — the library (source this to use)
- `tests/` — BATS test suite organized by feature
- `tests/fixtures/` — real terraform projects for integration testing
- `tests/helpers/` — shared test utilities
- `Dockerfile.coverage` — kcov coverage runner
- `scripts/` — install.sh, package.sh
- `.github/workflows/` — CI (build → test → release)

Key design principles:
- Pure bash (3.2+), no compiled deps except jq
- fd3 is the UI channel (stdout=data, stderr=errors, fd3=terminal UI)
- Strategy pattern: silent (plain), spinner (simple), progress (rich)
- Single library file — consumers just `source lib/tfui.sh`
- All tools managed via mise.toml

Show: line count, function count, test count, fixture count.
