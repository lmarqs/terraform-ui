# terraform-ui

Animated terminal UI for `terraform plan` and `terraform apply` operations.

Provides spinner, elapsed timer, progress bar, and tree-view diff output — all in pure bash.

## Features

- Spinner with elapsed time counter
- Progress bar tracking resource count during plan/apply
- Tree-view output showing planned changes (`+`, `~`, `-`, `-/+`)
- Three display modes: rich (progress bar), simple (spinner only), plain (silent)
- Works on macOS (bash 3.2+) and any Linux distro
- Single dependency: `jq`

## Install

### curl

```bash
curl -fsSL https://raw.githubusercontent.com/lmarqs/terraform-ui/main/scripts/install.sh | bash
```

### Basher

```bash
basher install lmarqs/terraform-ui
```

### Homebrew

```bash
brew tap lmarqs/terraform-ui
brew install terraform-ui
```

### mise

```toml
# mise.toml
[plugins]
terraform-ui = "https://github.com/lmarqs/terraform-ui"
```

### Manual

Download a release tarball from [GitHub Releases](https://github.com/lmarqs/terraform-ui/releases) and extract it.

## Usage

```bash
source "/path/to/terraform-ui/lib/tfui.sh"

plan_file=$(mktemp)
tfui_init "$MODULE_DIR" "auto"
tfui_plan "Planning module: $MODULE" --out "$plan_file"
if tfui_confirm "$plan_file"; then
  tfui_apply "$plan_file" "Applying module: $MODULE"
fi
```

### API

| Function | Description |
|----------|-------------|
| `tfui_init <dir> [mode]` | Initialize working directory and choose UI strategy |
| `tfui_plan <msg> [args] --out <file>` | Run terraform plan, render tree view |
| `tfui_confirm <file> [--auto-approve]` | Check for changes, optionally prompt user |
| `tfui_apply <file> <msg> [args]` | Apply the saved plan |

### UI Modes

| Mode | Description |
|------|-------------|
| `auto` | Rich if terminal available, plain otherwise |
| `rich` | Two-line UI: spinner + progress bar |
| `simple` | One-line spinner with elapsed time |
| `plain` | No UI output, captures silently |

## Requirements

- bash 3.2+ (macOS default works)
- jq

## Testing

```bash
bash tests/tfui-test.sh
```

## License

MIT
