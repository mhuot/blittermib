# Handoff: blittermib — MIB Browser

## Overview

A high-fidelity redesign of an SNMP MIB browser for the **blittermib** product. The interface helps NMS operators parse cryptic SNMP MIB data — thousands of long camelCase identifiers, dotted OID paths, and SMI types — by encoding type information as the primary visual signal and de-emphasizing the repetitive shared OID prefixes.

The design pairs blittermib's brand chrome (top brand bar + bottom footer) with a three-pane working surface: tree navigator on the left, an `ls -l`-style aligned object table in the center, and a detail/decoder pane on the right.

## About the design files

The HTML/JSX files in this bundle are **design references** — a working React prototype showing the intended look, layout, copy, color system, and interactive behavior. They are not meant to be shipped as-is.

The implementation task is to **recreate this design inside blittermib's actual codebase**, using its established framework, component patterns, and routing. Wire the components to real MIB-parser data instead of the mock `data.js` shipped here. If no frontend exists yet, React + a CSS-Modules / Tailwind / vanilla-CSS approach all map cleanly to the prototype — pick whatever fits best.

## Fidelity

**High-fidelity.** Final colors (oklch values), typography, spacing scale, type-tag color system, row layout, and interactions are all considered shipping-ready. Match these pixel-for-pixel; only adapt where the codebase's existing primitives (button, input, etc.) already encode equivalent decisions.

## Screens / views

There is one screen — the MIB browser. Its anatomy:

### 1. Brand bar (top, ~50px)
- Left: blittermib logo (3 stacked horizontal bars, decreasing widths 100% / 70% / 45%, 3px tall, accent color, 3px gap) + wordmark `blittermib.` (the trailing period is in the accent color)
- Inline tagline next to brand: `Pixelperfect MIB browser` (muted)
- Right: `Self-hosted — your MIBs never leave your server` (muted, with "Self-hosted" slightly emphasized) + theme toggle icon button (☾ / ☀)
- Background: `--bg`. 1px bottom border `--line`.

### 2. Status bar (~40px)
- Left: `MODULE` label (uppercase tracking 0.1em, muted) + module name in accent color + module OID in mono muted
- Center: counts (`<n> objects`, `<n> counters`, `<n> gauges`, `<n> integers`, `<n> strings`, `<n> notifs`) — each number colored by type family
- Background: `--bg-elev`. 1px bottom border.

### 3. Three-pane workspace (fills remaining height)

**Left navigator (320px fixed):**
- "TREE" section header with `+` / `−` collapse buttons
- Mono-font tree using literal `├──` / `└──` branch-line glyphs (color `--fg-4`)
- Each row: branch prefix · expand chevron (▸/▾) · name (camelCase prefix dimmed in `--fg-4`, distinguishing tail in `--fg`)
- Selected row: 2px accent left bar + `--bg-row-active` background

**Center inspector (flex 1):**
- Toolbar: shell-prompt search input (`›` glyph, mono, placeholder `grep name | oid | desc …`, ⌘K kbd hint) + filter chips (all / scalars / tables / notifs)
- Breadcrumb strip showing scope path (`axMgmt / axSystem / axSysCpu`), last segment in accent
- Sticky column header: blank · NAME · SYNTAX · ACCESS · OID
- Aligned data rows (28 / 1.6fr / 110 / 70 / 1fr grid):
  1. **Type letter badge** — single capital letter (C / G / I / S / X / T / A / B / N / ·) in a 18×18 rounded square, family-colored
  2. **Name** — camelCase prefix dimmed, tail bright
  3. **Syntax** — type name in family color
  4. **Access** — `ro` (cyan), `rw` (amber), `na` (muted)
  5. **OID** — tabular-nums; prefix in `--fg-4`, last segment bright + bold, copy button revealed on hover
- 2px family-colored stripe at row left edge

**Right detail pane (380px fixed):**
- Kind label + colored dot
- Object name (mono, 16px, 600)
- Inline pills: type / access / status / units
- Action row: `Copy OID` (primary, accent fill), `Copy name`, `Close`
- **Description** block — quoted callout style with 2px accent left border, sans body
- **OID decode** — one row per numeric segment, labeled `iso / org / dod / internet / private / enterprises / a10networks / axProducts / axMgmt / …`. Last segment in accent.
- **Properties** — 2-column key/value grid (76px label column, mono)
- **Enumeration** table (when present): Value | Name
- **Children** list with type letter + name + last-OID-segment

### 4. Footer (~40px)
- Left: `blittermib` (accent, 600) + `— runs entirely on your server`
- Right: `Made with AI in ♥ for Open Source in Europe · About kit` (heart in `--t-notif`, "About kit" linked)
- Background `--bg`, 1px top border.

## Interactions & behavior

- **Tree:** chevron click toggles a node; row click selects + sets inspector scope (if node has children). Top 2 levels expanded by default.
- **Search (`⌘K` / `Ctrl+K` to focus):** filters across name + OID + description; auto-expands ancestors of any match; matched substring is `<mark>`-highlighted in row name. `Esc` clears.
- **Filter chips:** `all / scalars / tables / notifs` — narrows the inspector list by kind.
- **Breadcrumb:** clickable segments narrow the inspector scope.
- **Copy buttons:** write OID (or name) to clipboard, flip to `✓` for ~1.2s, fire a center-bottom toast (`✓ Copied <oid>`).
- **Selection:** click any inspector row → loads in detail pane. `Esc` deselects.
- **Theme toggle:** flips `data-theme` between `dark` (default) and `light` on `<html>`.
- **Density (tweak):** `compact` (22px row) / `comfortable` (26px) / `spacious` (32px).

## State management

- `expanded: Set<oid>` — which tree branches are open
- `q: string` — search query
- `selected: oid | null` — current detail target
- `scopeOid: oid` — inspector scope (root by default)
- `theme`, `density`, `kindFilter`, `showDetail` — persisted UI prefs

In production, the tree itself should come from the MIB parser. For wide/deep MIBs the inspector list should be virtualized (it currently isn't — fine for the prototype's data volume, not fine for a real 1900-row module).

## Design tokens

All colors are oklch — convert to hex with whatever tooling fits the codebase. Lightness/chroma are matched across families so no tag visually shouts louder than another.

### Dark theme (default)
| Token | Value |
|---|---|
| `--bg` | `oklch(0.16 0.012 60)` |
| `--bg-elev` | `oklch(0.19 0.013 60)` |
| `--bg-elev-2` | `oklch(0.22 0.014 60)` |
| `--bg-row-hover` | `oklch(0.21 0.018 60)` |
| `--bg-row-active` | `oklch(0.26 0.04 60)` |
| `--line` | `oklch(0.28 0.014 60)` |
| `--line-soft` | `oklch(0.22 0.012 60)` |
| `--fg` | `oklch(0.94 0.012 75)` |
| `--fg-2` | `oklch(0.78 0.018 75)` |
| `--fg-3` | `oklch(0.58 0.018 75)` |
| `--fg-4` | `oklch(0.42 0.014 75)` |
| `--accent` (terracotta) | `oklch(0.78 0.14 55)` |

### Type-family colors (shared across themes; light theme drops lightness ~0.25)
| Family | Letter | Color | Used for |
|---|---|---|---|
| `t-counter` | C | `oklch(0.82 0.16 75)` amber | Counter32 / Counter64 |
| `t-gauge` | G | `oklch(0.78 0.13 200)` cyan | Gauge32 / Unsigned32 |
| `t-int` | I | `oklch(0.74 0.16 305)` violet | Integer32 / INTEGER {…} |
| `t-text` | S | `oklch(0.78 0.13 140)` green | DisplayString / OCTET STRING |
| `t-index` | X | `oklch(0.76 0.16 0)` pink | not-accessible index columns |
| `t-time` | T | `oklch(0.78 0.16 50)` orange | TimeTicks |
| `t-addr` | A | `oklch(0.78 0.12 185)` teal | IpAddress / MacAddress / PhysAddress |
| `t-bool` | B | `oklch(0.78 0.12 130)` mint | TruthValue |
| `t-notif` | N | `oklch(0.78 0.18 25)` red | NOTIFICATION-TYPE |
| `t-struct` | · | `oklch(0.66 0.02 75)` neutral | OBJECT-IDENTITY / table / entry |

Tag fill = color at 0.18 alpha; border = color at 0.4; text = the color itself (`oklch(from var(--c) l c h / 0.18)` etc.).

### Spacing & sizing
- Row heights: compact 22, comfortable 26, spacious 32
- Padding scale: 4 / 6 / 8 / 10 / 12 / 14 / 16 / 18 / 20px
- Border radius: 3px (small), 4px (default), 6px (cards)
- Pane widths: navigator 320px, detail 380px

### Typography
- **Sans (UI chrome, brand, tagline, descriptions):** Inter, weights 400/500/600/700
- **Mono (everything else — names, OIDs, types, table headers, prompts):** JetBrains Mono, weights 400/500/600/700
- Tabular-nums on every OID and numeric segment
- Letter-spacing: -0.02em on brand, 0.1em+uppercase on section labels, default elsewhere

## Assets

- No external imagery. The blittermib logo is pure CSS (3 colored bars in a flex column). The heart in the footer is a Unicode `♥`. Type-letter badges are text. All chevrons / arrows are inline SVG or Unicode glyphs.
- Fonts loaded from Google Fonts (Inter + JetBrains Mono). Substitute the codebase's font-loading strategy.

## Files in this bundle

- `MIB Browser v2.html` — entry point that loads the rest
- `styles-v2.css` — all CSS, including tokens, layout, components, themes
- `app-v2.jsx` — React app: components (`NavRow`, `ListRow`, `Detail`, `App`), state, keyboard handlers
- `helpers.js` — pure helpers: `typeFamily`, `flatten`, `searchWithAncestors`, `splitOid`, `countTypes`, `findByOid`
- `data.js` — mock A10-AX-MIB tree shaped like the real parsed output a backend would return (`{ module, tree: { name, oid, kind, type, access, status, desc, children, indexes, enumVals, ... } }`). Mirror this shape from the real parser.
- `tweaks-panel.jsx` — design-time controls; remove from the production build.

## Notes on integrating with a real backend

- The tree node shape (`name`, `oid`, `kind`, `type`, `access`, `status`, `desc`, `units`, `indexes`, `indexFor`, `enumVals`, `objects`, `children`) is what the UI consumes. Whatever shape the parser emits, project to this.
- `kind` is one of: `module | object | scalar | table | entry | column | notification | group`. The visual encoding (stripe color, glyph, badge) all derives from `kind` + `type`.
- For modules with thousands of objects, virtualize the inspector list (react-window / TanStack Virtual). Tree should also be virtualized when expanded fully.
