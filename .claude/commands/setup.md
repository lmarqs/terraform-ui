---
allowed-tools: Bash(mise:*)
description: Install tools and dependencies (mise install && mise run setup)
---

## Mise tasks: `install` + `setup`

Set up the development environment:

```bash
mise install        # Install all tools (jq, bats, terraform)
mise run setup      # Clone BATS helper libraries into tests/helpers/
```

After setup, verify with:
```bash
mise run build      # Syntax check lib/tfui.sh
mise run test:run   # Run full test suite
```

Related commands: /test-run, /coverage-run
