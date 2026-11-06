"""Convert a frame directory (from --record) into an animated GIF.

Usage: python demo/stitch.py <frame-dir> <output.gif>

Reads manifest.json from the frame directory, renders each frame's text in
monospace with its ANSI colors and weights, and produces a GIF.
"""

import json
import os
import re
import sys
from dataclasses import dataclass, replace
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

SGR = re.compile(r"\x1b\[([0-9;]*)m")
OTHER_ESCAPES = re.compile(r"\x1b\[[0-9;]*[A-Za-ln-z]|\x1b\].*?\x07|\x1b[()][AB012]")

# The docs site's terminal palette, so the GIFs match the panels around them.
BG_COLOR = (26, 16, 12)
FG_COLOR = (230, 217, 206)
BRIGHT = {
    0: (59, 42, 36),
    1: (255, 92, 77),
    2: (79, 209, 139),
    3: (255, 179, 71),
    4: (95, 179, 230),
    5: (255, 121, 198),
    7: (241, 230, 220),
}
# The pin count marker; the bundled fonts have no glyph for it, so it is drawn.
POSITION_INDICATOR = "\u2316"
FONT_SIZE = 14
PADDING = 16

# Each entry is the regular face and its bold, italic and bold-italic siblings;
# a missing sibling falls back to the regular face.
FONT_FAMILIES = [
    ("/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf", "-Bold", "-Oblique", "-BoldOblique"),
    ("/usr/share/fonts/dejavu-sans-mono-fonts/DejaVuSansMono.ttf", "-Bold", "-Oblique", "-BoldOblique"),
    ("/usr/share/fonts/TTF/DejaVuSansMono.ttf", "-Bold", "-Oblique", "-BoldOblique"),
    ("/System/Library/Fonts/Menlo.ttc", None, None, None),
    ("/System/Library/Fonts/SFMono-Regular.otf", None, None, None),
    ("C:/Windows/Fonts/consola.ttf", None, None, None),
]


@dataclass(frozen=True)
class Style:
    fg: tuple[int, int, int] = FG_COLOR
    bg: tuple[int, int, int] | None = None
    bold: bool = False
    italic: bool = False


def apply_sgr(style: Style, params: str) -> Style:
    for code in (int(p) if p else 0 for p in params.split(";")):
        if code == 0:
            style = Style()
        elif code == 1:
            style = replace(style, bold=True)
        elif code == 3:
            style = replace(style, italic=True)
        elif 90 <= code <= 97:
            style = replace(style, fg=BRIGHT[code - 90])
        elif 100 <= code <= 107:
            style = replace(style, bg=BRIGHT[code - 100])
    return style


def parse_line(line: str, style: Style) -> tuple[list[tuple[str, Style]], Style]:
    """Split a line into runs of text sharing one style; returns the style in effect at its end."""
    runs: list[tuple[str, Style]] = []
    pos = 0
    for match in SGR.finditer(line):
        if match.start() > pos:
            runs.append((line[pos : match.start()], style))
        style = apply_sgr(style, match.group(1))
        pos = match.end()
    if pos < len(line):
        runs.append((line[pos:], style))
    return runs, style


def load_fonts() -> dict[tuple[bool, bool], ImageFont.FreeTypeFont]:
    for regular, bold, italic, bold_italic in FONT_FAMILIES:
        if not os.path.exists(regular):
            continue
        stem, ext = os.path.splitext(regular)
        variants = {(False, False): regular}
        for key, suffix in (((True, False), bold), ((False, True), italic), ((True, True), bold_italic)):
            path = f"{stem}{suffix}{ext}" if suffix else regular
            variants[key] = path if os.path.exists(path) else regular
        return {key: ImageFont.truetype(path, FONT_SIZE) for key, path in variants.items()}
    font = ImageFont.load_default()
    return {key: font for key in ((False, False), (True, False), (False, True), (True, True))}


def measure_cell_dimensions(font: ImageFont.FreeTypeFont) -> tuple[float, float]:
    img = Image.new("RGB", (2000, 200))
    draw = ImageDraw.Draw(img)
    ref = "X" * 120
    bbox = draw.textbbox((0, 0), ref, font=font)
    cell_width = (bbox[2] - bbox[0]) / 120.0
    lines = "\n".join(["X"] * 10)
    bbox = draw.textbbox((0, 0), lines, font=font)
    cell_height = (bbox[3] - bbox[1]) / 10.0
    return cell_width, cell_height


def draw_position_indicator(
    draw: ImageDraw.ImageDraw, x: int, y: int, cell_width: float, cell_height: float, fill: tuple[int, int, int]
) -> None:
    cx, cy = x + cell_width / 2, y + cell_height / 2
    r = cell_width * 0.32
    draw.ellipse((cx - r, cy - r, cx + r, cy + r), outline=fill)
    arm = cell_width * 0.48
    draw.line((cx - arm, cy, cx + arm, cy), fill=fill)
    draw.line((cx, cy - arm, cx, cy + arm), fill=fill)


def render_frame(
    text: str,
    width: int,
    height: int,
    fonts: dict[tuple[bool, bool], ImageFont.FreeTypeFont],
    cell_width: float,
    cell_height: float,
) -> Image.Image:
    img_width = int(cell_width * width) + PADDING * 2
    img_height = int(cell_height * height) + PADDING * 2
    img = Image.new("RGB", (img_width, img_height), BG_COLOR)
    draw = ImageDraw.Draw(img)

    style = Style()
    for row, line in enumerate(OTHER_ESCAPES.sub("", text).split("\n")[:height]):
        runs, style = parse_line(line, style)
        y = PADDING + int(row * cell_height)
        col = 0
        for chunk, run_style in runs:
            chunk = chunk[: width - col]
            if not chunk:
                break
            x = PADDING + int(col * cell_width)
            if run_style.bg is not None:
                draw.rectangle(
                    (x, y, PADDING + int((col + len(chunk)) * cell_width), y + int(cell_height)),
                    fill=run_style.bg,
                )
            draw.text(
                (x, y),
                chunk.replace(POSITION_INDICATOR, " "),
                font=fonts[(run_style.bold, run_style.italic)],
                fill=run_style.fg,
            )
            for offset, char in enumerate(chunk):
                if char == POSITION_INDICATOR:
                    draw_position_indicator(draw, x + int(offset * cell_width), y, cell_width, cell_height, run_style.fg)
            col += len(chunk)

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

    fonts = load_fonts()
    cell_width, cell_height = measure_cell_dimensions(fonts[(False, False)])
    images: list[Image.Image] = []
    durations: list[int] = []

    for meta in frames_meta:
        frame_path = frame_dir / meta["file"]
        if not frame_path.exists():
            continue
        text = frame_path.read_text()
        img = render_frame(text, width, height, fonts, cell_width, cell_height)
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
