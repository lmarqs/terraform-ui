#!/bin/bash
set -euo pipefail

# Generate demo recordings from macro tapes.
# Produces frame directories in demo/output/<name>/
# Then stitches them into GIFs.
#
# A tape that fails to record or stitch is skipped, the rest still run, and the
# script exits 1 after naming every failure.
#
# Prerequisites:
#   - Built binary (mise run build)
#   - Stitch tool (see demo/stitch.sh for options)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Find binary
if [ -n "${1:-}" ]; then
  TFUI="$1"
else
  TFUI=$(ls "$PROJECT_DIR"/dist/tfui_"$(go env GOOS)"_"$(go env GOARCH)"*/tfui 2>/dev/null | head -1 || true)
fi

if [ -z "$TFUI" ] || [ ! -x "$TFUI" ]; then
  echo "ERROR: no binary found — run 'mise run build' first"
  echo "  or pass the binary path: ./demo/generate.sh ./path/to/tfui"
  exit 1
fi

FIXTURES="$SCRIPT_DIR/fixtures"
PLAN="$FIXTURES/plan-large.json"
STATE="$FIXTURES/state-large.json"
OUTPUT="$SCRIPT_DIR/output"

echo "Using binary: $TFUI"
echo ""

# Optional: record a single tape for development
TAPE_FILTER="${2:-}"

# Record all tapes (or a single one if specified)
recorded=()
failed=()
for tape in "$SCRIPT_DIR"/tapes/*.tape; do
  name=$(basename "$tape" .tape)
  if [ -n "$TAPE_FILTER" ] && [ "$name" != "$TAPE_FILTER" ]; then
    continue
  fi
  echo "Recording $name..."
  rm -rf "$OUTPUT/$name"
  mkdir -p "$OUTPUT/$name"
  # Lipgloss drops colors when stdout is not a terminal; the GIFs need them.
  CLICOLOR_FORCE=1 "$TFUI" \
    --plan "$PLAN" \
    --state "$STATE" \
    --macro "$tape" \
    --record "$OUTPUT/$name" \
    >/dev/null 2>&1 && recorded+=("$name") || { echo "  FAILED: $name did not record"; failed+=("$name"); }
done

echo ""
echo "Recordings complete. Frame directories in: $OUTPUT/"
echo ""

# Stitch into GIFs
echo "Stitching GIFs..."
stitched=()
for name in ${recorded[@]+"${recorded[@]}"}; do
  if "$SCRIPT_DIR/stitch.sh" "$OUTPUT/$name/" "$OUTPUT/$name.gif"; then
    echo "  $name.gif"
    stitched+=("$name")
  else
    echo "  FAILED: $name did not stitch"
    failed+=("$name")
  fi
done

# Copy to docs assets for GitHub Pages
DOCS_ASSETS="$PROJECT_DIR/docs/assets/demo"
mkdir -p "$DOCS_ASSETS"
copied=0
for name in ${stitched[@]+"${stitched[@]}"}; do
  cp "$OUTPUT/$name.gif" "$DOCS_ASSETS/"
  copied=1
done
[ "$copied" -eq 0 ] || { echo ""; echo "Copied GIFs to docs/assets/demo/"; }

echo ""
if [ "${#failed[@]}" -gt 0 ]; then
  echo "FAILED: ${failed[*]}" >&2
  exit 1
fi
echo "Done. GIFs in: $OUTPUT/"
