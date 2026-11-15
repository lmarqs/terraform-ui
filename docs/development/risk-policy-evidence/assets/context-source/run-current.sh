#!/usr/bin/env bash
set -euo pipefail
root=$(git rev-parse --show-toplevel)
assets="$root/docs/development/risk-policy-evidence/assets/context-source"
work="$root/.ralph-tui/d02c"
mkdir -p "$work/tmp" "$work/go-build" "$work/go-cache"
export TMPDIR="$work/tmp" GOTMPDIR="$work/go-build" GOCACHE="$work/go-cache"
python3 - "$root" "$assets" "$work" <<'PYTHON'
import json,sys
from pathlib import Path
root,assets,work=map(Path,sys.argv[1:])
(work/'overlay.json').write_text(json.dumps({'Replace':{str(root/'internal/ui/context_source_experiment_test.go'):str(assets/'current_test.go.txt')}}))
PYTHON
cd "$root"
go test -overlay "$work/overlay.json" -count=1 -timeout=60s -run '^TestContextSource_' -v ./internal/ui
