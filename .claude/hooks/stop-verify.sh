#!/bin/bash
set -euo pipefail

CHANGED=$(git diff --name-only HEAD 2>/dev/null; git diff --name-only 2>/dev/null; git ls-files --others --exclude-standard 2>/dev/null)

if ! echo "$CHANGED" | grep -qE '\.go$'; then
  exit 0
fi

ERRORS=""

if ! go vet ./... 2>&1; then
  ERRORS="${ERRORS}go vet failed\n"
fi

if command -v golangci-lint &>/dev/null; then
  LINT_OUT=$(golangci-lint run ./... 2>&1) || ERRORS="${ERRORS}${LINT_OUT}\n"
fi

if [ -n "$ERRORS" ]; then
  echo -e "$ERRORS" >&2
  exit 2
fi

exit 0
