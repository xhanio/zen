import { describe, it, expect } from 'vitest';
import { sanitizeCardHtml } from './sanitizeHtml';

describe('sanitizeCardHtml', () => {
  it('strips <script> tags', () => {
    const out = sanitizeCardHtml('<p>hi</p><script>alert(1)</script>');
    expect(out).toContain('<p>hi</p>');
    expect(out).not.toContain('<script');
  });

  it('strips on* event handlers', () => {
    const out = sanitizeCardHtml('<p onclick="x()">hi</p>');
    expect(out).not.toContain('onclick');
    expect(out).toContain('hi');
  });

  it('strips javascript: hrefs', () => {
    const out = sanitizeCardHtml('<a href="javascript:alert(1)">x</a>');
    expect(out).not.toContain('javascript:');
  });

  it('preserves inline SVG', () => {
    const svg = '<svg viewBox="0 0 10 10"><rect width="10" height="10"/><text x="0" y="5">hi</text></svg>';
    const out = sanitizeCardHtml(svg);
    expect(out).toContain('<svg');
    expect(out).toContain('<rect');
    expect(out).toContain('<text');
    expect(out).toContain('hi');
  });

  it('preserves inline <style>', () => {
    const out = sanitizeCardHtml('<style>.x { color: red }</style><p class="x">hi</p>');
    expect(out).toContain('<style>');
    expect(out).toContain('.x { color: red }');
  });

  it('preserves data: image URLs', () => {
    const out = sanitizeCardHtml('<img src="data:image/png;base64,iVBOR" alt="x"/>');
    expect(out).toContain('data:image/png');
  });

  it('strips <iframe> and <object>', () => {
    expect(sanitizeCardHtml('<iframe src="x"></iframe>')).toBe('');
    expect(sanitizeCardHtml('<object data="x"></object>')).toBe('');
  });

  it('preserves MathML', () => {
    const out = sanitizeCardHtml('<math><mi>x</mi><mo>+</mo><mn>1</mn></math>');
    expect(out).toContain('<math');
    expect(out).toContain('<mi>x</mi>');
  });

  it('preserves SVG viewBox/preserveAspectRatio (sanitize-html lowercases these — browser case-corrects on parse)', () => {
    const svg = '<svg viewBox="0 0 100 60" preserveAspectRatio="xMidYMid meet"><rect width="100" height="60"/></svg>';
    const out = sanitizeCardHtml(svg);
    // Output is lowercased; browser HTML parser will case-correct to camelCase
    // for SVG-namespaced foreign content on insertion.
    expect(out).toContain('viewbox="0 0 100 60"');
    expect(out).toContain('preserveaspectratio');
    expect(out).toContain('<rect');
  });

  it('preserves camelCase SVG tags (linearGradient, clipPath) — lowercased then case-corrected', () => {
    const svg = '<svg><defs><linearGradient id="g"><stop offset="0"/></linearGradient><clipPath id="c"/></defs></svg>';
    const out = sanitizeCardHtml(svg);
    expect(out).toContain('<lineargradient');
    expect(out).toContain('<clippath');
    expect(out).toContain('id="g"');
  });
});
