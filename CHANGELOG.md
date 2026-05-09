# Changelog

## Unreleased

### CI

- read version from artifact, not workflow outputs
- fix version to v0.35.0 (continues from v0.34.0)
- include bin/tfui and VERSION in build artifact, package as tarball
- resolve version at build step, release only consumes artifacts
- replace release-please with direct versioning from VERSION file
- only run release on push to main, skip on PRs
- grant pull-requests write permission to release job
- replace commit-count versioning with release-please
- add comment explaining Node 24 env var
- add FORCE_JAVASCRIPT_ACTIONS_TO_NODE24 to all workflows
- opt into Node.js 24 for GitHub Actions
- clean up release assets
- publish lib/tfui.sh as build artifact
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

- rewrite README for clarity and add visual examples
- rewrite README with CLI reference and architecture
- update CLAUDE.md for BATS test workflow
- fix install.sh URL path after move to scripts/
- add project documentation and config

### Features

- add CLI entry point (bin/tfui)
- add project slash commands for common workflows
- add install methods (curl, basher, homebrew)
- add tfui library

### Miscellaneous

- pin jq to major version 1
- add claude code configuration

### Refactor

- build as mise task, move syntax check to test
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

