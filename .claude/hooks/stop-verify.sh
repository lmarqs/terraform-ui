#!/bin/bash
set -euo pipefail

CHANGED=$(git diff --name-only HEAD 2>/dev/null; git diff --name-only 2>/dev/null; git ls-files --others --exclude-standard 2>/dev/null)

if ! echo "$CHANGED" | grep -qE '\.go$'; then
  exit 0
fi

if command -v golangci-lint &>/dev/null; then
  if ! LINT_OUT=$(golangci-lint run ./... 2>&1); then
    echo "$LINT_OUT" >&2
    exit 2
  fi
fi

exit 0
