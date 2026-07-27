import { describe, it, expect } from 'vitest';
import { renderedText, offsetsForRange, trimToText, collectTextNodes } from './selectionRange';

function host(html: string): HTMLElement {
  const el = document.createElement('div');
  el.innerHTML = html;
  return el;
}

describe('renderedText', () => {
  it('concatenates text nodes in document order across elements', () => {
    expect(renderedText(host('<p>Hello <strong>brave</strong> world</p>'))).toBe('Hello brave world');
  });

  // HtmlBody appends a <style> into the shadow root, and every Design-group
  // card carries its own inline <style>. Counting that CSS would shift every
  // offset after it.
  it('skips style and script content', () => {
    expect(renderedText(host('<style>.a{color:red}</style><p>visible</p>'))).toBe('visible');
    expect(renderedText(host('<script>var x=1</script><p>visible</p>'))).toBe('visible');
  });

  // The offset space must not move when a highlight is already painted, so
  // marks are traversed INTO rather than skipped.
  it('includes text inside existing zen-ref marks', () => {
    const plain = renderedText(host('<p>Hello brave world</p>'));
    const painted = renderedText(host('<p>Hello <mark class="zen-ref">brave</mark> world</p>'));
    expect(painted).toBe(plain);
  });

  it('is empty for an empty root', () => {
    expect(renderedText(host(''))).toBe('');
  });
});

describe('offsetsForRange', () => {
  it('measures a range inside one text node', () => {
    const root = host('<p>Hello brave world</p>');
    const t = root.firstChild!.firstChild as Text;
    const r = document.createRange();
    r.setStart(t, 6);
    r.setEnd(t, 11);
    expect(offsetsForRange(root, r)).toEqual({ start: 6, end: 11 });
    expect(renderedText(root).slice(6, 11)).toBe('brave');
  });

  it('measures a range spanning two elements', () => {
    const root = host('<p>Hello <strong>brave</strong> world</p>');
    const nodes = collectTextNodes(root);
    const r = document.createRange();
    r.setStart(nodes[0], 6); // inside "Hello "
    r.setEnd(nodes[2], 3); // inside " world"
    const got = offsetsForRange(root, r)!;
    expect(renderedText(root).slice(got.start, got.end)).toBe('brave wo');
  });

  it('returns null for a collapsed range', () => {
    const root = host('<p>abc</p>');
    const t = root.firstChild!.firstChild as Text;
    const r = document.createRange();
    r.setStart(t, 2);
    r.setEnd(t, 2);
    expect(offsetsForRange(root, r)).toBeNull();
  });

  it('returns null when the range is not inside the root', () => {
    const root = host('<p>abc</p>');
    const outside = host('<p>xyz</p>');
    const t = outside.firstChild!.firstChild as Text;
    const r = document.createRange();
    r.setStart(t, 0);
    r.setEnd(t, 2);
    expect(offsetsForRange(root, r)).toBeNull();
  });

  // A range measured before painting must measure the same after, or a second
  // selection of the same span would record different numbers.
  it('is unchanged by an existing mark earlier in the body', () => {
    const before = host('<p>Hello brave world</p>');
    const nodesBefore = collectTextNodes(before);
    const r1 = document.createRange();
    r1.setStart(nodesBefore[0], 12);
    r1.setEnd(nodesBefore[0], 17);
    const a = offsetsForRange(before, r1)!;

    const after = host('<p><mark class="zen-ref">Hello</mark> brave world</p>');
    const nodesAfter = collectTextNodes(after);
    const r2 = document.createRange();
    r2.setStart(nodesAfter[1], 7); // " brave world" → offset 12 overall
    r2.setEnd(nodesAfter[1], 12);
    const b = offsetsForRange(after, r2)!;

    expect(b).toEqual(a);
    expect(renderedText(after).slice(b.start, b.end)).toBe('world');
  });
});

// useSelectionBubble reports sel.toString().trim(); offsets taken from the raw
// DOM Range describe a wider span, so the checksum could never match.
describe('trimToText', () => {
  it('shrinks the window past leading and trailing whitespace', () => {
    const full = 'Hello  brave  world';
    const got = trimToText(full, 5, 14);
    expect(full.slice(got.start, got.end)).toBe('brave');
  });

  it('leaves an already-tight window alone', () => {
    const full = 'Hello brave world';
    expect(trimToText(full, 6, 11)).toEqual({ start: 6, end: 11 });
  });

  it('collapses an all-whitespace window to empty', () => {
    const full = 'a   b';
    const got = trimToText(full, 1, 4);
    expect(got.end).toBe(got.start);
  });
});
