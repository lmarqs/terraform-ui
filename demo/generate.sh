#!/bin/bash
set -euo pipefail

# Generate demo recordings from macro tapes.
# Produces frame directories in demo/output/<name>/
# Then stitches them into GIFs.
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
  TFUI=$(ls "$PROJECT_DIR"/dist/tfui_"$(go env GOOS)"_"$(go env GOARCH)"*/tfui 2>/dev/null | head -1)
fi

if [ -z "$TFUI" ] || [ ! -x "$TFUI" ]; then
  echo "ERROR: no binary found — run 'mise run build' first"
  echo "  or pass the binary path: ./demo/generate.sh ./path/to/tfui"
  exit 1
fi

FIXTURES="$SCRIPT_DIR/fixtures"
PLAN="$FIXTURES/plan-large.json"
STATE="$FIXTURES/state-large.json"
OUTPUTS="$FIXTURES/outputs.json"
VALIDATE_RESULT="$FIXTURES/validate.json"
WORKSPACES="$FIXTURES/workspaces.json"
OUTPUT="$SCRIPT_DIR/output"

echo "Using binary: $TFUI"
echo ""

# Optional: record a single tape for development
TAPE_FILTER="${2:-}"

# Record all tapes (or a single one if specified)
for tape in "$SCRIPT_DIR"/tapes/*.tape; do
  name=$(basename "$tape" .tape)
  if [ -n "$TAPE_FILTER" ] && [ "$name" != "$TAPE_FILTER" ]; then
    continue
  fi
  echo "Recording $name..."
  rm -rf "$OUTPUT/$name"
  mkdir -p "$OUTPUT/$name"
  "$TFUI" \
    --plan "$PLAN" \
    --state "$STATE" \
    --outputs "$OUTPUTS" \
    --validate-result "$VALIDATE_RESULT" \
    --workspaces "$WORKSPACES" \
    --macro "$tape" \
    --record "$OUTPUT/$name" \
    >/dev/null 2>&1 || echo "  WARNING: $name exited with error (may be expected for partial flows)"
done

echo ""
echo "Recordings complete. Frame directories in: $OUTPUT/"
echo ""

# Stitch into GIFs
echo "Stitching GIFs..."
for dir in "$OUTPUT"/*/; do
  [ -d "$dir" ] || continue
  name=$(basename "$dir")
  "$SCRIPT_DIR/stitch.sh" "$dir" "$OUTPUT/$name.gif" && echo "  $name.gif" || echo "  SKIP: $name (stitch failed)"
done

# Copy to docs assets for GitHub Pages
DOCS_ASSETS="$PROJECT_DIR/docs/assets/demo"
mkdir -p "$DOCS_ASSETS"
cp "$OUTPUT"/*.gif "$DOCS_ASSETS/" 2>/dev/null && echo "" && echo "Copied GIFs to docs/assets/demo/" || true

echo ""
echo "Done. GIFs in: $OUTPUT/"
