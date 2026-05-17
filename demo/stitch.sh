#!/bin/bash
set -euo pipefail

# Convert a frame directory (from --record) into an animated GIF.
#
# Usage: ./stitch.sh <frame-dir> <output.gif>
#
# Requires one of:
#   - agg (asciinema GIF generator): cargo install agg
#   - vhs (Charm): brew install charmbracelet/tap/vhs
#   - Custom: override with STITCH_TOOL env var
#
# How it works:
#   1. Reads manifest.json from the frame directory
#   2. Converts ANSI text frames into an asciicast v2 file
#   3. Renders the asciicast to GIF using agg

FRAME_DIR="${1:?Usage: stitch.sh <frame-dir> <output.gif>}"
OUTPUT="${2:?Usage: stitch.sh <frame-dir> <output.gif>}"

MANIFEST="$FRAME_DIR/manifest.json"
if [ ! -f "$MANIFEST" ]; then
  echo "ERROR: no manifest.json in $FRAME_DIR"
  exit 1
fi

# Extract dimensions from manifest
WIDTH=$(python3 -c "import json; m=json.load(open('$MANIFEST')); print(m['width'])" 2>/dev/null || echo 80)
HEIGHT=$(python3 -c "import json; m=json.load(open('$MANIFEST')); print(m['height'])" 2>/dev/null || echo 24)

# Build asciicast v2 format from frames
CAST_FILE=$(mktemp /tmp/tfui-demo-XXXXXX.cast)
trap 'rm -f "$CAST_FILE"' EXIT

# Header
echo "{\"version\": 2, \"width\": $WIDTH, \"height\": $HEIGHT, \"timestamp\": $(date +%s)}" > "$CAST_FILE"

# Frames → asciicast events
python3 -c "
import json, os, sys

manifest = json.load(open('$MANIFEST'))
frames = manifest['frames']
time_offset = 0.0

for frame in frames:
    filepath = os.path.join('$FRAME_DIR', frame['file'])
    if not os.path.exists(filepath):
        continue
    with open(filepath, 'r') as f:
        content = f.read()
    delay_s = frame['delay_ms'] / 1000.0
    time_offset += delay_s
    # Clear screen + move to top + write content
    output = '\x1b[2J\x1b[H' + content
    event = json.dumps([time_offset, 'o', output])
    print(event)
" >> "$CAST_FILE"

# Render to GIF
if command -v agg >/dev/null 2>&1; then
  agg --font-size 14 --cols "$WIDTH" --rows "$HEIGHT" "$CAST_FILE" "$OUTPUT"
elif command -v vhs >/dev/null 2>&1; then
  # VHS can render asciicast files
  echo "Using VHS to render (slower than agg)..."
  # Create a minimal VHS tape that plays the cast
  VHS_TAPE=$(mktemp /tmp/tfui-vhs-XXXXXX.tape)
  trap 'rm -f "$CAST_FILE" "$VHS_TAPE"' EXIT
  cat > "$VHS_TAPE" <<TAPE
Output $OUTPUT
Set Width $WIDTH
Set Height $HEIGHT
Set FontSize 14
Set Theme "Dracula"
Source $CAST_FILE
TAPE
  vhs "$VHS_TAPE"
else
  echo "ERROR: no GIF renderer found. Install one of:"
  echo "  - agg: cargo install agg"
  echo "  - vhs: brew install charmbracelet/tap/vhs"
  echo ""
  echo "Asciicast saved to: $CAST_FILE"
  trap '' EXIT  # Don't clean up the cast file
  exit 1
fi

echo "Created: $OUTPUT"
