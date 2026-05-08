# Contributing

## Setup

```bash
mise install
```

## Development

1. Edit `lib/tfui.sh`
2. Run tests: `bash tests/tfui-test.sh`
3. Syntax check: `bash -n lib/tfui.sh`

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add --timeout flag to tfui_plan
fix: handle empty terraform state list
docs: update usage examples
```

## Tests

All changes must pass the test suite. Add tests for new functionality in `tests/tfui-test.sh`.

## Requirements

- bash 3.2+ (must work on macOS default bash)
- No GNU-specific flags (stay POSIX-compatible for `grep`, `sed`, etc.)
- Only external dependency allowed: `jq`
