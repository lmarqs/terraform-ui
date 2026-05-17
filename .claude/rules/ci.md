---
description: "CI/CD pipeline, semantic-release + goreleaser interaction, versioning, and mise task conventions"
globs: [".github/**", ".goreleaser.yaml", ".releaserc", "mise.toml"]
---

# CI/CD Pipeline

## Flow

```
PR / push to main → main.yaml
  ├── build.yaml    lint → unit tests (ubuntu+macos) → coverage → binaries
  ├── test.yaml     macro tapes + integration tests (against built artifacts)
  └── release.yaml  semantic-release → goreleaser (if new version)
```

## Stage Responsibilities

| Stage | File | Produces |
|-------|------|----------|
| Build | `build.yaml` | `dist/` artifact (4 binaries) |
| Test | `test.yaml` | Pass/fail (no artifacts) |
| Release | `release.yaml` | Git tag, CHANGELOG.md, GitHub release |

## semantic-release + goreleaser

goreleaser invoked via `@semantic-release/exec.publishCmd`:
1. semantic-release analyzes conventional commits → bumps version, creates tag + release
2. `publishCmd: "goreleaser release --clean"` → builds binaries and uploads

Config:
- `.releaserc`: `publishCmd` invokes goreleaser
- `.goreleaser.yaml`: `release.mode: append` (SR owns release), `changelog.disable: true` (SR owns changelog)

## Versioning

| Context | Source |
|---------|--------|
| CI build | git tag via ldflags |
| `go install @vX.Y.Z` | module metadata (ReadBuildInfo) |
| `go run ./cmd/tfui` (dev) | `"0.0.0-SNAPSHOT"` |
| goreleaser snapshot | git describe |

Resolution in `cmd/tfui/main.go`:
```
ldflags → ReadBuildInfo().Main.Version → "0.0.0-SNAPSHOT"
```

## Mise Task Convention

| Namespace | Purpose |
|-----------|---------|
| `check:*` | Static analysis (no build) |
| `build` | Produce artifacts |
| `test:*` | Verify correctness |
| `docs:*` | Jekyll documentation site |
| `demo:*` | Demo GIF generation and Python lockfile |
| `release` | Publish (CI only) |
| _(top-level)_ | Developer tools |

Rules:
- Each task does ONE thing
- CI orchestrates order, not mise
- All tasks callable standalone
- Task names map directly to workflow calls

## Adding a CI check

1. Create mise task in `mise.toml` under appropriate namespace
2. Call from relevant workflow stage
3. Workflow files stay thin — just `mise run <task>`
