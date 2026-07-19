// Level color: a continuous gradient across a group's levels, hottest (lowest
// weight, most abstract) → coldest (highest weight, most concrete). Colors are
// interpolated with color-mix(in oklch, …) between the existing level tokens,
// so light and dark themes come for free from the theme-scoped CSS vars.
//
// Anchor path: rose (L0, hot) → amber (L1, pulled early to 15%) → sky (L3,
// cold). Violet (L4) reads as warm-purple rather than cold, and green (L2)
// breaks the warm-to-cool sweep, so both are skipped. Unfiled (null) is the
// neutral slate `tbd`.
//
// "Cold-forward" spacing: with two warm anchors (rose, amber) and one cold one
// (sky), even spacing spends most of the range warm, so mid/high levels stay
// warm-muddy. Placing amber at 15% keeps only a quick warm kick up front and
// gives the rest of the range to the amber→sky cooldown, so mid/high levels
// read as genuinely cold.
//
// ONE hue per level. Only the fg channel is interpolated; bg and border are
// that same fg lightened toward the card surface. Interpolating separate
// --l-N-bg / --l-N-border ramps made them drift in hue from fg between anchors,
// so a card tile (border) stopped matching the sidebar legend (fg). Deriving
// the tints from fg keeps every channel of a level on a single hue at every
// rank — the tile and the legend read as the same color.

import type { LevelEntry } from '../types/entity';
import { weightForCard } from './levelCatalog';

// The fg anchor path, hottest → coldest, as { level CSS-var index, position on
// the 0..1 hot→cold range }. amber sits early (15%) — see the "cold-forward"
// note above. rose and sky pin the ends.
const ANCHORS: ReadonlyArray<{ v: number; p: number }> = [
  { v: 0, p: 0 }, // rose  — hottest
  { v: 1, p: 0.15 }, // amber — a quick warm kick near the hot end
  { v: 3, p: 1 }, // sky   — coldest
];

export interface LevelColor {
  fg: string; // text / solid accent (the hue)
  bg: string; // subtle background tint
  border: string; // border / divider
}

function tbd(): LevelColor {
  return { fg: 'var(--l-tbd-fg)', bg: 'var(--l-tbd-bg)', border: 'var(--l-tbd-border)' };
}

// Interpolate the fg hue at position t ∈ [0,1] along the anchor path. Finds the
// segment t falls in and blends its two anchors by their local fraction. Exact
// anchor hits return a bare var(); between anchors a color-mix blends them.
function mixFg(t: number): string {
  t = Math.min(1, Math.max(0, t));
  for (let i = 0; i < ANCHORS.length - 1; i++) {
    const lo = ANCHORS[i];
    const hi = ANCHORS[i + 1];
    // Keep scanning until t is within this segment (or it's the last one).
    if (t > hi.p && i < ANCHORS.length - 2) continue;
    const span = hi.p - lo.p;
    const frac = span <= 0 ? 0 : (t - lo.p) / span;
    if (frac <= 0) return `var(--l-${lo.v}-fg)`;
    if (frac >= 1) return `var(--l-${hi.v}-fg)`;
    return `color-mix(in oklch, var(--l-${hi.v}-fg) ${Math.round(frac * 100)}%, var(--l-${lo.v}-fg))`;
  }
  return `var(--l-${ANCHORS[ANCHORS.length - 1].v}-fg)`;
}

// A subtle tint of fg: the same hue lightened toward the card surface. `weight`
// is fg's share (%), so a smaller weight is a paler tint. Both themes work —
// --surface is near-white in light mode, near-black in dark — so the tint moves
// away from fg in whichever direction reads as "lighter".
function tint(fg: string, weight: number): string {
  return `color-mix(in oklch, ${fg} ${weight}%, var(--surface))`;
}

// The color for a level of the given weight, scaled by its rank among the
// group's DISTINCT weights — so same-weight entries share a color, and sparse
// or large catalogs still span the full hot→cold range with no clamping. A
// null weight (Unfiled) is the neutral tbd color.
export function levelColor(catalog: LevelEntry[], weight: number | null | undefined): LevelColor {
  if (weight === null || weight === undefined) return tbd();
  const weights = [...new Set(catalog.map((e) => e.weight))].sort((a, b) => a - b);
  const rank = weights.indexOf(weight);
  if (rank < 0) return tbd();
  const t = weights.length <= 1 ? 0 : rank / (weights.length - 1);
  const fg = mixFg(t);
  // bg (pill fill) and border (tile top bar / divider) are the same hue as fg,
  // just paler — so the grid tiles match the legend at every rank. Percentages
  // approximate the old hand-tuned --l-N-bg / --l-N-border shades.
  return { fg, bg: tint(fg, 14), border: tint(fg, 45) };
}

// Convenience: the color for a card, resolving its weight via the catalog.
export function colorForCard(
  catalog: LevelEntry[],
  card: { level_entry_id: string | null },
): LevelColor {
  return levelColor(catalog, weightForCard(catalog, card));
}

// A vertical hot→cold gradient across a group's level ranks, for the rail's
// level "spine". Empty catalog → '' (no spine); a single level → its solid
// hottest color (a one-level group has no range to sweep).
export function levelSpineGradient(catalog: LevelEntry[]): string {
  const weights = [...new Set(catalog.map((e) => e.weight))];
  if (weights.length === 0) return '';
  if (weights.length === 1) return mixFg(0);
  const stops: string[] = [];
  for (let i = 0; i <= 8; i++) stops.push(`${mixFg(i / 8)} ${Math.round((i / 8) * 100)}%`);
  return `linear-gradient(180deg, ${stops.join(', ')})`;
}
