# Changelog

## 0.36.5 — 2026-05-09

### Documentation

- use version = "latest" in mise install example

### Miscellaneous

- v0.36.4 [skip ci]

## 0.36.4 — 2026-05-09

### Documentation

- add mise installation method to README

### Miscellaneous

- v0.36.3 [skip ci]

## 0.36.3 — 2026-05-09

### CI

- build tarball in build task, not release

### Miscellaneous

- v0.36.2 [skip ci]

## 0.36.2 — 2026-05-09

### Bug Fixes

- preserve executable permission in release tarball

### Miscellaneous

- v0.36.1 [skip ci]

## 0.36.1 — 2026-05-09

### CI

- add git-cliff changelog generation and semantic versioning

## 0.36.0 — 2026-05-09

### Documentation

- rewrite README for clarity and add visual examples
- rewrite README with CLI reference and architecture

## 0.35.0 — 2026-05-09

### CI

- read version from artifact, not workflow outputs
- fix version to v0.35.0 (continues from v0.34.0)
- include bin/tfui and VERSION in build artifact, package as tarball
- resolve version at build step, release only consumes artifacts
- replace release-please with direct versioning from VERSION file
- only run release on push to main, skip on PRs
- grant pull-requests write permission to release job
- replace commit-count versioning with release-please

### Features

- add CLI entry point (bin/tfui)

### Refactor

- build as mise task, move syntax check to test

## 0.34.0 — 2026-05-09

### CI

- add comment explaining Node 24 env var

## 0.33.0 — 2026-05-09

### CI

- add FORCE_JAVASCRIPT_ACTIONS_TO_NODE24 to all workflows

## 0.32.0 — 2026-05-09

### CI

- opt into Node.js 24 for GitHub Actions

## 0.31.0 — 2026-05-09

### CI

- clean up release assets

## 0.30.0 — 2026-05-09

### CI

- publish lib/tfui.sh as build artifact

## 0.29.0 — 2026-05-09

### CI

- publish test and coverage reports as artifacts in releases
- replace Codecov with GitHub step summary for coverage
- fix coverage job failures
- remove release and coverage from pipeline
- fix test reporter permissions and coverage job
- add Docker-based kcov coverage runner
- add JUnit test reporting and coverage job
- restructure pipeline into main, build, test, release
- add test and release-please workflows

### Documentation

- update CLAUDE.md for BATS test workflow
- fix install.sh URL path after move to scripts/
- add project documentation and config

### Features

- add project slash commands for common workflows
- add install methods (curl, basher, homebrew)
- add tfui library

### Miscellaneous

- pin jq to major version 1
- add claude code configuration

### Refactor

- align mise tasks and slash commands to noun-verb convention
- rename commands to noun-verb convention
- rename test_helper to helpers and update references
- move install.sh and package.sh into scripts/

### Testing

- replace mock terraform with real fixtures in flow tests
- add terraform fixtures for integration testing
- remove legacy custom test framework
- migrate all scenarios to BATS test files
- add BATS framework infrastructure
- add BDD-style test suite

