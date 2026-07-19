import { describe, it, expect } from 'vitest';
import { levelColor, levelSpineGradient } from './levelPalette';
import type { LevelEntry } from '../types/entity';

const cat = (...weights: number[]): LevelEntry[] =>
  weights.map((w, i) => ({ id: `e${i}`, weight: w, name: `L${i}` }));

describe('levelColor', () => {
  it('lowest level is the hottest anchor (rose), highest is the coldest (sky)', () => {
    const c = cat(0, 1, 2, 3);
    expect(levelColor(c, 0).fg).toBe('var(--l-0-fg)'); // hottest
    expect(levelColor(c, 3).fg).toBe('var(--l-3-fg)'); // coldest
  });

  it('spans the full range regardless of absolute weights (no clamping)', () => {
    const c = cat(0, 5, 10); // sparse, above the old 0..4 clamp
    expect(levelColor(c, 0).fg).toBe('var(--l-0-fg)');
    expect(levelColor(c, 10).fg).toBe('var(--l-3-fg)'); // highest → sky, not a clamped violet
    // amber sits at 15%, so the midpoint is already well into the amber→sky
    // cooldown — a blend, not the bare amber anchor.
    expect(levelColor(c, 5).fg).toContain('color-mix');
  });

  it('is cold-forward: the upper-middle level is majority-cold (sky)', () => {
    // 4 levels → ranks at t = 0, 1/3, 2/3, 1. With amber pulled to 15%, the
    // 3rd level (t = 2/3) is past the amber→sky midpoint, so sky is the
    // dominant share — the whole point of the cold-forward spacing.
    const mix = levelColor(cat(0, 1, 2, 3), 2).fg;
    const skyShare = mix.match(/--l-3-fg\)\s*(\d+)%/);
    expect(skyShare).not.toBeNull();
    expect(Number(skyShare![1])).toBeGreaterThan(50);
  });

  it('interpolates between anchors for levels off an anchor point', () => {
    // 4 evenly-spaced levels → ranks at t = 0, 1/3, 2/3, 1; the two middles
    // fall between anchors and blend.
    const c = cat(0, 1, 2, 3);
    expect(levelColor(c, 1).fg).toContain('color-mix');
    expect(levelColor(c, 2).fg).toContain('color-mix');
  });

  it('two levels use the extremes: rose and sky', () => {
    const c = cat(0, 1);
    expect(levelColor(c, 0).fg).toBe('var(--l-0-fg)');
    expect(levelColor(c, 1).fg).toBe('var(--l-3-fg)');
  });

  it('same-weight entries share a color and do not inflate the scale', () => {
    // Two entries at weight 1 collapse to one rank; the scale is 0,1,2 → 3 steps.
    const c = cat(0, 1, 1, 2);
    expect(levelColor(c, 0).fg).toBe('var(--l-0-fg)');
    expect(levelColor(c, 2).fg).toBe('var(--l-3-fg)');
  });

  it('null weight is the neutral tbd color', () => {
    expect(levelColor(cat(0, 1), null).fg).toBe('var(--l-tbd-fg)');
    expect(levelColor(cat(0, 1), null).bg).toBe('var(--l-tbd-bg)');
  });

  it('a single level is the hottest color', () => {
    expect(levelColor(cat(2), 2).fg).toBe('var(--l-0-fg)');
  });

  it('never touches the dropped violet (L4) or green (L2)', () => {
    const c = cat(0, 1, 2, 3, 4, 5, 6);
    for (const w of [0, 1, 2, 3, 4, 5, 6]) {
      const { fg, bg, border } = levelColor(c, w);
      for (const channel of [fg, bg, border]) {
        expect(channel).not.toContain('--l-4-');
        expect(channel).not.toContain('--l-2-');
      }
    }
  });

  it('spine: empty catalog → empty string', () => {
    expect(levelSpineGradient([])).toBe('');
  });
  it('spine: multi-level → a hot→cold linear-gradient spanning rose to sky', () => {
    const g = levelSpineGradient(cat(0, 1, 2, 3));
    expect(g).toContain('linear-gradient(180deg');
    expect(g).toContain('var(--l-0-fg)');
    expect(g).toContain('var(--l-3-fg)');
  });
  it('spine: single level → a solid hottest color, not a gradient', () => {
    expect(levelSpineGradient(cat(2))).toBe('var(--l-0-fg)');
  });

  it('bg and border share the fg hue (mixed toward the surface, not separate ramps)', () => {
    // The mismatch bug: bg/border were interpolated on their own
    // --l-N-bg / --l-N-border ramps, which diverge in hue from fg between
    // anchors — so a card tile (border) stopped matching the legend (fg).
    // They must now be the SAME hue as fg, only lightened toward the surface.
    const c = cat(0, 1, 2, 3);
    for (const w of [0, 1, 2, 3]) {
      const { fg, bg, border } = levelColor(c, w);
      for (const tint of [bg, border]) {
        // built from fg, lightened toward the card surface
        expect(tint).toContain('color-mix');
        expect(tint).toContain('var(--surface)');
        // never the independent bg/border tokens
        expect(tint).not.toContain('-bg)');
        expect(tint).not.toContain('-border)');
      }
      // exact-anchor fg (bare var) still appears verbatim inside its tints
      if (!fg.includes('color-mix')) {
        expect(bg).toContain(fg);
        expect(border).toContain(fg);
      }
    }
  });
});
