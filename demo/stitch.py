#!/usr/bin/env python3
"""Convert a frame directory (from --record) into an animated GIF.

Usage: python demo/stitch.py <frame-dir> <output.gif>

Reads manifest.json from the frame directory, strips ANSI codes from each
frame, renders monospace text onto images with Pillow, and produces a GIF.
"""

import json
import os
import re
import sys
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ANSI_ESCAPE = re.compile(r"\x1b\[[0-9;]*[A-Za-z]|\x1b\].*?\x07|\x1b[()][AB012]")

BG_COLOR = (40, 42, 54)
FG_COLOR = (248, 248, 242)
FONT_SIZE = 14
CELL_WIDTH = 8
CELL_HEIGHT = 18
PADDING = 16


def strip_ansi(text: str) -> str:
    return ANSI_ESCAPE.sub("", text)


def load_font() -> ImageFont.FreeTypeFont:
    mono_paths = [
        "/usr/share/fonts/TTF/JetBrainsMonoNerdFont-Regular.ttf",
        "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
        "/usr/share/fonts/dejavu-sans-mono-fonts/DejaVuSansMono.ttf",
        "/usr/share/fonts/TTF/DejaVuSansMono.ttf",
        "/System/Library/Fonts/Menlo.ttc",
        "/System/Library/Fonts/SFMono-Regular.otf",
        "C:/Windows/Fonts/consola.ttf",
    ]
    for p in mono_paths:
        if os.path.exists(p):
            return ImageFont.truetype(p, FONT_SIZE)
    return ImageFont.load_default()


def render_frame(text: str, width: int, height: int, font: ImageFont.FreeTypeFont) -> Image.Image:
    img_width = width * CELL_WIDTH + PADDING * 2
    img_height = height * CELL_HEIGHT + PADDING * 2
    img = Image.new("RGB", (img_width, img_height), BG_COLOR)
    draw = ImageDraw.Draw(img)

    clean = strip_ansi(text)
    lines = clean.split("\n")[:height]

    for row, line in enumerate(lines):
        truncated = line[:width]
        x = PADDING
        y = PADDING + row * CELL_HEIGHT
        draw.text((x, y), truncated, font=font, fill=FG_COLOR)

    return img


def main() -> None:
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <frame-dir> <output.gif>", file=sys.stderr)
        sys.exit(1)

    frame_dir = Path(sys.argv[1])
    output_path = Path(sys.argv[2])

    manifest_path = frame_dir / "manifest.json"
    if not manifest_path.exists():
        print(f"ERROR: no manifest.json in {frame_dir}", file=sys.stderr)
        sys.exit(1)

    manifest = json.loads(manifest_path.read_text())
    width = manifest["width"]
    height = manifest["height"]
    frames_meta = manifest["frames"]

    if not frames_meta:
        print("ERROR: no frames in manifest", file=sys.stderr)
        sys.exit(1)

    font = load_font()
    images: list[Image.Image] = []
    durations: list[int] = []

    for meta in frames_meta:
        frame_path = frame_dir / meta["file"]
        if not frame_path.exists():
            continue
        text = frame_path.read_text()
        img = render_frame(text, width, height, font)
        images.append(img)
        durations.append(max(meta["delay_ms"], 50))

    if not images:
        print("ERROR: no renderable frames", file=sys.stderr)
        sys.exit(1)

    output_path.parent.mkdir(parents=True, exist_ok=True)
    images[0].save(
        output_path,
        save_all=True,
        append_images=images[1:],
        duration=durations,
        loop=0,
        optimize=True,
    )
    print(f"Created: {output_path} ({len(images)} frames)")


if __name__ == "__main__":
    main()
