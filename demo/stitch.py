#!/usr/bin/env python3
"""Convert a frame directory (from --record) into an animated GIF.

Usage: python demo/stitch.py <frame-dir> <output.gif>

Reads manifest.json from the frame directory and renders each frame's text,
with its ANSI colors and weights, the way the docs site draws its terminal
panels: JetBrains Mono from docs/assets/fonts, the same palette, line height
and padding. Frames render at SCALE times the site's CSS size so they stay
sharp when the page scales them down.
"""

import io
import json
import os
import re
import sys
from dataclasses import dataclass, replace
from pathlib import Path

from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont
from PIL import Image, ImageDraw, ImageFont

SGR = re.compile(r"\x1b\[([0-9;]*)m")
OTHER_ESCAPES = re.compile(r"\x1b\[[0-9;]*[A-Za-ln-z]|\x1b\].*?\x07|\x1b[()][AB012]")

SITE_FONTS = Path(__file__).resolve().parent.parent / "docs" / "assets" / "fonts"
FALLBACK_FONT = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"

# docs/assets/site.css: .terminal and .terminal.stage pre.
SCALE = 2
FONT_SIZE = 13.5 * SCALE
LINE_HEIGHT = 1.55
PADDING_X = 16 * SCALE
PADDING_Y = 14 * SCALE
BG_COLOR = (11, 6, 5)
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

# The pin count marker; no font the site ships has a glyph for it, so it is drawn.
POSITION_INDICATOR = "⌖"


@dataclass(frozen=True)
class Style:
    fg: tuple[int, int, int] = FG_COLOR
    bg: tuple[int, int, int] | None = None
    bold: bool = False


def apply_sgr(style: Style, params: str) -> Style:
    """Italic (3) is ignored: the site ships no italic face, and the panels show none."""
    for code in (int(p) if p else 0 for p in params.split(";")):
        if code == 0:
            style = Style()
        elif code == 1:
            style = replace(style, bold=True)
        elif 90 <= code <= 97:
            style = replace(style, fg=BRIGHT[code - 90])
        elif 100 <= code <= 107:
            style = replace(style, bg=BRIGHT[code - 100])
    return style


def parse_line(line: str, style: Style) -> tuple[list[tuple[str, Style]], Style]:
    """Split a line into styled characters; returns the style in effect at its end."""
    cells: list[tuple[str, Style]] = []
    pos = 0
    for match in SGR.finditer(line):
        cells.extend((char, style) for char in line[pos : match.start()])
        style = apply_sgr(style, match.group(1))
        pos = match.end()
    cells.extend((char, style) for char in line[pos:])
    return cells, style


class Fonts:
    """The site's JetBrains Mono faces, picked per character the way the browser picks them."""

    def __init__(self) -> None:
        latin = SITE_FONTS / "jetbrains-mono-latin.woff2"
        symbols = SITE_FONTS / "jetbrains-mono-symbols.woff2"
        self.latin_chars = set(TTFont(latin).getBestCmap())
        self.symbol_chars = set(TTFont(symbols).getBestCmap())
        self.latin = {bold: self._load(latin, 700 if bold else 400) for bold in (False, True)}
        self.symbols = self._load(symbols, None)
        self.fallback = ImageFont.truetype(FALLBACK_FONT, FONT_SIZE) if os.path.exists(FALLBACK_FONT) else None

    @staticmethod
    def _load(path: Path, weight: int | None) -> ImageFont.FreeTypeFont:
        font = TTFont(path)
        font.flavor = None
        if weight is not None and "fvar" in font:
            font = instantiateVariableFont(font, {"wght": weight})
        buffer = io.BytesIO()
        font.save(buffer)
        buffer.seek(0)
        return ImageFont.truetype(buffer, FONT_SIZE)

    def for_char(self, char: str, bold: bool) -> ImageFont.FreeTypeFont | None:
        code = ord(char)
        if (0x2190 <= code <= 0x21FF or 0x2500 <= code <= 0x25FF) and code in self.symbol_chars:
            return self.symbols
        if code in self.latin_chars:
            return self.latin[bold]
        return self.fallback


def draw_position_indicator(
    draw: ImageDraw.ImageDraw, x: float, y: float, cell_width: float, cell_height: float, fill: tuple[int, int, int]
) -> None:
    cx, cy = x + cell_width / 2, y + cell_height / 2
    r = cell_width * 0.32
    draw.ellipse((cx - r, cy - r, cx + r, cy + r), outline=fill, width=SCALE)
    arm = cell_width * 0.48
    draw.line((cx - arm, cy, cx + arm, cy), fill=fill, width=SCALE)
    draw.line((cx, cy - arm, cx, cy + arm), fill=fill, width=SCALE)


def render_frame(text: str, width: int, height: int, fonts: Fonts, cell_width: float) -> Image.Image:
    cell_height = FONT_SIZE * LINE_HEIGHT
    size = (round(cell_width * width) + PADDING_X * 2, round(cell_height * height) + PADDING_Y * 2)
    img = Image.new("RGB", size, BG_COLOR)
    draw = ImageDraw.Draw(img)

    style = Style()
    for row, line in enumerate(OTHER_ESCAPES.sub("", text).split("\n")[:height]):
        cells, style = parse_line(line, style)
        y = PADDING_Y + row * cell_height
        for col, (char, cell_style) in enumerate(cells[:width]):
            x = PADDING_X + col * cell_width
            if cell_style.bg is not None:
                draw.rectangle((x, y, x + cell_width, y + cell_height), fill=cell_style.bg)
            if char == " ":
                continue
            if char == POSITION_INDICATOR:
                draw_position_indicator(draw, x, y, cell_width, cell_height, cell_style.fg)
                continue
            font = fonts.for_char(char, cell_style.bold)
            if font is not None:
                draw.text((x, y + cell_height / 2), char, font=font, fill=cell_style.fg, anchor="lm")

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

    fonts = Fonts()
    cell_width = fonts.latin[False].getlength("X")
    images: list[Image.Image] = []
    durations: list[int] = []

    for meta in frames_meta:
        frame_path = frame_dir / meta["file"]
        if not frame_path.exists():
            continue
        images.append(render_frame(frame_path.read_text(), width, height, fonts, cell_width))
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
