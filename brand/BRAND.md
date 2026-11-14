# VYRO — Brand Guide

**Name:** VYRO
**Slogan:** Connect Beyond Limits — «فراتر از مرزها وصل شو»

## Logo

`logo.svg` — a rounded chat bubble in a blue→violet gradient, with a
white **V** inside, a signal dot, and small orbit dots suggesting
messages travelling beyond limits. The bubble tail points down-left,
like a sent message.

- Use the SVG at any size (it is vector). Minimum clear space: the
  height of the V stroke on all sides.
- Do not recolour the gradient, stretch the mark, or add a drop shadow
  in the logo file itself — shadows belong to the UI, not the mark.
- Monochrome fallback: use `#2563EB` (blue) on light backgrounds,
  white on dark/gradient backgrounds.

## Colours

| Name | Hex | Use |
|---|---|---|
| VYRO Blue | `#2563EB` | Primary, links, primary buttons |
| VYRO Violet | `#7C3AED` | Secondary, gradient end, accents |
| Gradient mid | `#5B4DEF` | Gradient midpoint |
| Light blue | `#60A5FA` | Orbit dots, highlights on dark |
| Light violet | `#A78BFA` | Orbit dots, highlights on dark |
| Ink (light theme text) | `#0F172A` | Headings / body |
| Surface (light) | `#FFFFFF` / `#F8FAFC` | Cards / background |
| Ink (dark theme text) | `#F1F5F9` | Headings / body on dark |
| Surface (dark) | `#1E293B` / `#0F172A` | Cards / background on dark |

Gradient (CSS): `linear-gradient(135deg, #2563EB, #7C3AED)`

## Typography

- Latin: system UI stack (`system-ui, -apple-system, "Segoe UI", sans-serif`).
- Persian: system stack with Tahoma fallback — no webfont is bundled
  in v0.1, so no paid / licensed font is required.
- Headings: bold, tight. Body: regular, generous line-height.

## Voice

Short, direct, honest. Never claim a feature that is not built.
Error messages say what happened and what to do next.
