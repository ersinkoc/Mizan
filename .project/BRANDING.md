# Mizan — Branding Guide

> *"Visual Config Architect for HAProxy & Nginx"*

---

## 1. Name & Etymology

**Mizan** (مِيزَان / Mizan) — Arabic and Turkish for **"scale"** or **"balance"**. The word carries both the literal meaning of a weighing instrument and the metaphysical sense of **just measure** — exactly what a load balancer does: weigh traffic and distribute it fairly across backends.

- **Pronunciation**: *mee-ZAHN* (IPA: /miːˈzɑːn/)
- **Cultural lineage**: Same root family as Ersin's other Turkish-anchored projects (Karadul, Duman, imece).
- **Domain candidates**: `mizan.dev`, `mizanproxy.com`, `mizan.io`
- **GitHub org**: `github.com/mizanproxy/mizan` (alternative: `github.com/mizan-io/mizan`)
- **Binary name**: `mizan`

### Why this name fits

A scale has **two pans** — perfect symbolism for a tool that targets **two engines** (HAProxy + Nginx) from a single **balance beam** (the universal intermediate representation). The user designs once on the beam; both pans receive correctly-weighted output.

---

## 2. Tagline & Positioning

**Primary tagline**: *Visual Config Architect for HAProxy & Nginx*

**Alternate taglines** (context-dependent):

- *Design once. Deploy to either. Watch both.*
- *The scales of traffic.*
- *Balance, by design.*
- *From topology to production, in one binary.*

**Positioning sentence (one-liner)**: Mizan is a single-binary Go application that lets engineers design HAProxy and Nginx configurations through a visual wizard and an interactive topology canvas, validate them, deploy them via SSH, and watch them run — all from a unified WebUI that ships embedded in the binary.

---

## 3. Color Palette

The palette is inspired by an **antique brass scale on dark obsidian** — heavy, judicial, precise.

### Core Colors

| Role | Name | Hex | Usage |
|------|------|-----|-------|
| Primary | **Mizan Crimson** | `#B91C1C` | Brand accent, the central beam of the scale, primary CTAs, alerts |
| Secondary | **Bronze Pan** | `#92400E` | Secondary accent, highlight states, hover backgrounds |
| Surface (dark) | **Obsidian** | `#0C0A09` | Dark mode background |
| Surface (light) | **Parchment** | `#F5F1E8` | Light mode background |
| Foreground (dark) | **Ivory** | `#F4F1EA` | Text on Obsidian |
| Foreground (light) | **Ink** | `#1C1917` | Text on Parchment |
| Muted | **Slate** | `#44403C` | Borders, dividers, muted text |
| Success | **Aegean** | `#0F766E` | Healthy backend, deploy success |
| Warning | **Amber** | `#D97706` | Validation warnings |
| Danger | **Crimson Bright** | `#DC2626` | Errors, failed health checks |

### Tailwind v4 CSS Variables (oklch-based)

```css
@theme {
  --color-mizan-crimson: oklch(0.51 0.20 27);
  --color-mizan-bronze:  oklch(0.46 0.13 50);
  --color-mizan-obsidian: oklch(0.13 0.01 50);
  --color-mizan-parchment: oklch(0.96 0.02 90);
  --color-mizan-ivory: oklch(0.95 0.02 85);
  --color-mizan-ink: oklch(0.18 0.01 50);
  --color-mizan-slate: oklch(0.36 0.01 60);
  --color-mizan-aegean: oklch(0.50 0.10 180);
  --color-mizan-amber: oklch(0.66 0.15 70);
}
```

### Topology Node Colors

React Flow nodes use semantic colors that map onto the IR concept:

| Node Type | Fill | Stroke | Icon Color |
|-----------|------|--------|------------|
| Frontend (Listener) | Bronze Pan @ 12% | Bronze Pan | Bronze Pan |
| Backend Pool | Aegean @ 12% | Aegean | Aegean |
| Server (Upstream) | Slate @ 12% | Slate | Ivory/Ink |
| ACL / Router | Mizan Crimson @ 12% | Mizan Crimson | Mizan Crimson |
| TLS Terminator | Amber @ 12% | Amber | Amber |
| Cache | Aegean @ 18% | Aegean (dashed) | Aegean |
| Rate Limiter | Mizan Crimson @ 18% | Mizan Crimson (dashed) | Mizan Crimson |

---

## 4. Typography

| Use | Family | Weight | Source |
|-----|--------|--------|--------|
| UI sans | **Inter** | 400 / 500 / 600 / 700 | Google Fonts |
| Display | **Fraunces** (semi-serif, italics for hero) | 400 / 600 / 700 | Google Fonts |
| Mono (code, configs) | **JetBrains Mono** | 400 / 500 / 700 | Google Fonts |

Fonts are self-hosted from the binary (embedded via `embed`) — no Google Fonts CDN call at runtime. The brand voice in display type leans **judicial and considered**, contrasted with the technical neutrality of Inter for the working interface.

---

## 5. Logo Concept

**Mark**: A minimalist scale (terazi) where the **two pans become an arrow** — one pan tilts down (inbound traffic, heavier), the other balances upward (distributed). Stylized so that the pivot point reads as the letter **M**.

**Logotype**: `MIZAN` set in **Fraunces 700**, slightly condensed, all caps, with the **A** and **N** sharing a serif stroke to suggest two backends sharing a single inbound flow.

**Three lockups required**:

1. Mark only (favicon, `16×16` to `512×512`)
2. Mark + wordmark (horizontal, primary lockup)
3. Mark + wordmark + tagline (footer / about page)

**Don'ts**: do not place the mark on busy photographic backgrounds; do not skew or rotate the pivot; do not recolor outside the palette; do not let the logo go below `24px` height.

---

## 6. Voice & Tone

Mizan speaks like a **senior infrastructure engineer who's tired of YAML and config drift**. Direct, occasionally dry, never marketing-fluff. Errors are honest. Successes are understated.

| Bad | Good |
|-----|------|
| 🚀 Deployment was a huge success! | Deployed to `lb-prod-1`. Reload OK in 2.3s. |
| ⚠️ Oops! Something went wrong. | `haproxy -c` failed: line 47, unknown directive `roundrobin2`. |
| ✨ Welcome to the future of load balancing! | New project. Pick HAProxy, Nginx, or both. |
| Let's get started on your journey! | Start with a frontend. Backends come next. |

Documentation tone is the same: technical, terse, examples-first. No emojis in product copy except for status icons (✓ ✗ ⚠) where they earn their place. Turkish locale (`tr-TR`) gets the same treatment — no overly polite filler, mirror the directness.

---

## 7. UI Density & Motion

- **Density**: Comfortable on desktop (default), compact mode toggleable for power users editing large clusters.
- **Motion**: Subtle. 150–200ms transitions, ease-out curves. No bouncing, no parallax. The topology canvas pans and zooms with momentum but everything else snaps.
- **Empty states**: Always include a single, obvious next action — no decorative empty illustrations beyond the brand mark grayscaled.
- **Loading**: Skeleton blocks in Slate @ 30% opacity. No spinners except for sub-200ms operations where skeletons would flicker.

---

## 8. Assets Checklist

- [ ] Logo SVG (mark, lockup, lockup+tagline) — light & dark variants
- [ ] Favicon set (`16`, `32`, `48`, `192`, `512`, `apple-touch-icon`)
- [ ] OG image (`1200×630`) — dark Obsidian background, mark center-left, tagline right
- [ ] README banner (`1280×400`) — same composition, web-optimized
- [ ] Topology node icon set (Frontend, Backend, Server, ACL, TLS, Cache, Rate-Limiter, Logger) — Lucide-compatible 24×24 strokes
- [ ] Twitter/X header (`1500×500`)
- [ ] CLI ASCII art for `mizan version` output

---

**Three Pans. One Beam. All Traffic.**
