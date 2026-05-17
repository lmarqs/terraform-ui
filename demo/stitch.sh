#!/bin/bash
set -euo pipefail

# Convert a frame directory (from --record) into an animated GIF.
# Usage: ./stitch.sh <frame-dir> <output.gif>
#
# Uses Python + Pillow by default (batteries included).
# Override with STITCH_TOOL=agg or STITCH_TOOL=vhs if preferred.

FRAME_DIR="${1:?Usage: stitch.sh <frame-dir> <output.gif>}"
OUTPUT="${2:?Usage: stitch.sh <frame-dir> <output.gif>}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

python3 "$SCRIPT_DIR/stitch.py" "$FRAME_DIR" "$OUTPUT"
