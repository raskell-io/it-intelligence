"""Write the it-intelligence brand assets (concept 01 Basis) as SVG: mark, wordmark and lockup.

The wordmark is Schibsted Grotesk Bold (SIL OFL 1.1), shaped with HarfBuzz (kerning) and outlined into
paths, so the SVGs need no font.

Usage (from the repository root, with the font from google/fonts ofl/schibstedgrotesk):
  uv run --no-project --with fonttools --with uharfbuzz python docs/brand/gen.py SchibstedGrotesk[wght].ttf docs/brand
"""

import sys
from pathlib import Path

import uharfbuzz as hb
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont

FONT, OUT = sys.argv[1], Path(sys.argv[2])
OUT.mkdir(parents=True, exist_ok=True)

INK = "#1f2a44"
DERIVED = "#2f6fd6"
INFERRED = "#7a5ab8"
PALE = "#e8eefc"
TEXT = "it-intelligence"
TRACKING = -0.02  # em

# ---- the wordmark as one path ----
font = instantiateVariableFont(TTFont(FONT), {"wght": 700})
tmp = OUT / ".instance.ttf"
font.save(tmp)
blob = hb.Blob.from_file_path(str(tmp))
hbfont = hb.Font(hb.Face(blob))
buf = hb.Buffer()
buf.add_str(TEXT)
buf.guess_segment_properties()
hb.shape(hbfont, buf, {"kern": True, "liga": True})
upm = font["head"].unitsPerEm
glyphset = font.getGlyphSet()
order = font.getGlyphOrder()
pen = SVGPathPen(glyphset)
x = 0
for info, pos in zip(buf.glyph_infos, buf.glyph_positions):
    name = order[info.codepoint]
    # flip y: font units grow upwards, SVG downwards
    glyphset[name].draw(TransformPen(pen, (1, 0, 0, -1, x + pos.x_offset, -pos.y_offset)))
    x += pos.x_advance + TRACKING * upm
word_path = pen.getCommands()
word_width = x - TRACKING * upm
os2 = font["OS/2"]
asc, desc = os2.sTypoAscender, os2.sTypoDescender  # desc is negative
xh, cap = os2.sxHeight, os2.sCapHeight
tmp.unlink()


def mark(dark: bool, size: str = "") -> str:
    """The mark on its 64 grid. dark=True is the variant for dark backgrounds."""
    bg, fg = (PALE, INK) if dark else (INK, "#ffffff")
    der, inf = (DERIVED, INFERRED) if dark else ("#8fb4ff", "#c3b0ff")
    return f"""<rect x="2" y="2" width="60" height="60" rx="15" fill="{bg}"/>
  <line x1="18" y1="44" x2="32" y2="18" stroke="{fg}" stroke-width="3.6" stroke-linecap="round"/>
  <line x1="34.1" y1="17.6" x2="48.1" y2="43.6" stroke="{der}" stroke-width="2" stroke-linecap="round"/>
  <line x1="30.9" y1="19.2" x2="44.9" y2="45.2" stroke="{der}" stroke-width="2" stroke-linecap="round"/>
  <line x1="18" y1="44" x2="46" y2="44" stroke="{inf}" stroke-width="3" stroke-dasharray="3.6 3.4" stroke-linecap="round"/>
  <circle cx="32" cy="18" r="6" fill="{fg}"/>
  <circle cx="18" cy="44" r="5.4" fill="{fg}"/>
  <circle cx="46" cy="44" r="5.4" fill="{fg}"/>"""


def write(name: str, svg: str) -> None:
    (OUT / name).write_text(svg.strip() + "\n")
    print("wrote", name)


title = "<title>it-intelligence</title>"
for dark in (False, True):
    write(f"mark{'-on-dark' if dark else ''}.svg",
          f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" role="img" aria-label="it-intelligence">{title}\n  {mark(dark)}\n</svg>')

# Wordmark: tight box from the descender to the ascender.
pad = 20
vb_y, vb_h = -asc - pad, asc - desc + 2 * pad
for suffix, colour in (("dark", INK), ("light", PALE)):
    write(f"wordmark-{suffix}.svg",
          f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{-pad} {vb_y} {word_width + 2 * pad:.0f} {vb_h}" role="img" aria-label="it-intelligence">{title}\n'
          f'  <path fill="{colour}" d="{word_path}"/>\n</svg>')

# Lockup: the mark as tall as ascender to descender, centred on the x-height band, then the wordmark.
mark_size = asc - desc
scale = mark_size / 64
gap = 0.32 * upm
mark_y = -(xh / 2) - mark_size / 2
text_x = mark_size + gap
total_w = text_x + word_width
top, bottom = min(mark_y, -asc), max(mark_y + mark_size, -desc)
for suffix, colour, dark in (("dark", INK, False), ("light", PALE, True)):
    write(f"lockup-{suffix}.svg",
          f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 {top:.0f} {total_w:.0f} {bottom - top:.0f}" role="img" aria-label="it-intelligence">{title}\n'
          f'  <g transform="translate(0 {mark_y:.1f}) scale({scale:.4f})">\n  {mark(dark)}\n  </g>\n'
          f'  <path fill="{colour}" transform="translate({text_x:.0f} 0)" d="{word_path}"/>\n</svg>')

# Favicon: the mark, plus a dark-mode switch the browser understands.
write("favicon.svg", f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <style>.d{{display:none}} @media (prefers-color-scheme: dark) {{ .l{{display:none}} .d{{display:inline}} }}</style>
  <g class="l">{mark(False)}</g>
  <g class="d">{mark(True)}</g>
</svg>""")
print(f"upm {upm}, x-height {xh}, cap {cap}, word width {word_width:.0f}")
