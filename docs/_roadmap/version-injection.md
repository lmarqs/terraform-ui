---
title: Build-Time Version Injection
status: planned
priority: high
created: 2026-05-11
effort: small
tags: [debt, build, release]
depends_on: []
---

## Summary

The binary version depends on reading a `VERSION` file at build time. The hardcoded default in `main.go` is `"1.0.0-dev"` which doesn't match the actual version (0.39.0). Dev builds show useless version info with no git sha or timestamp.

## Need

What user pain does this solve? What's the current workaround?

- `go run ./cmd/tfui version` shows `1.0.0-dev` (wrong, confusing)
- `mise run build` reads VERSION file — a release artifact that shouldn't be a build input
- No useful info in dev builds (no timestamp, no "dirty" indicator)
- `go install ...@latest` users get `1.0.0-dev` (no ldflags injection possible)
- VERSION file exists as a coordination mechanism between semantic-release and goreleaser that shouldn't be needed

Current workaround: manually check git log or VERSION file to know what version is running.

## Expected UX

How the user interacts with this feature.

| Scenario | Version String |
|----------|---------------|
| `go run ./cmd/tfui version` | `0.0.0-SNAPSHOT` |
| `mise run build` (local dev) | `20260511143022-SNAPSHOT` |
| `mise run build 0.40.0` (CI) | `0.40.0` |
| goreleaser (tag v0.40.0) | `0.40.0` |
| `go install ...@v0.39.0` | `0.39.0` (via ReadBuildInfo) |

## Advantages

Why this is worth doing.

- Dev builds always distinguishable from releases
- No dependency on VERSION file for building
- `go install` users get proper version via module metadata
- Timestamp in dev builds helps with "which build am I running?"
- Cleaner separation between release automation and build process

## Effort Justification

Why the effort estimate is what it is.

Small effort (< 1 day):
- Single-file change to `cmd/tfui/main.go`
- One-line change to `mise.toml`
- Minor update to `.releaserc`
- Existing test infrastructure already uses ldflags injection

## Design

Technical approach.

Replace the hardcoded version string with a multi-source resolution strategy:

1. **Compile-time injection** (ldflags): `-X main.version=...` at build time
2. **Runtime fallback** (ReadBuildInfo): extract version from Go module metadata
3. **Dev fallback**: `0.0.0-SNAPSHOT` if no metadata available

```go
// cmd/tfui/main.go
var version string // empty default, injected at build time

func init() {
    if version == "" {
        version = resolveVersion()
    }
}

func resolveVersion() string {
    if info, ok := debug.ReadBuildInfo(); ok {
        if info.Main.Version != "" && info.Main.Version != "(devel)" {
            return info.Main.Version
        }
    }
    return "0.0.0-SNAPSHOT"
}
```

Build script changes:

```toml
# mise.toml
[tasks.build]
run = '''
  VERSION="${1:-$(date -u +%Y%m%d%H%M%S)-SNAPSHOT}"
  go build -ldflags "-X main.version=$VERSION" -o tfui ./cmd/tfui
'''
```

Release process:
- `.releaserc` continues to write VERSION file (for git commit tracking)
- goreleaser reads VERSION file for release metadata
- Build process never reads VERSION file directly

## Open Questions

None.

## Tasks

- [ ] Change `cmd/tfui/main.go`: `var version string` (empty default)
- [ ] Add `init()` with `resolveVersion()` function using `runtime/debug.ReadBuildInfo()` fallback
- [ ] Update `mise.toml` build task: `version="${1:-$(date -u +%Y%m%d%H%M%S)-SNAPSHOT}"`
- [ ] Update `.releaserc`: `prepareCmd` writes VERSION only (for git commit tracking), doesn't build binary
- [ ] VERSION file kept for semantic-release commit history but build process never reads it
- [ ] Verify integration tests still pass (they inject version via ldflags)
