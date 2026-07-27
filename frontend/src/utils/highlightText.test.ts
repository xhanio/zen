import { describe, it, expect } from 'vitest';
import { wrapHighlights, type Highlight } from './highlightText';

function host(html: string): HTMLElement {
  const el = document.createElement('div');
  el.innerHTML = html;
  document.body.appendChild(el);
  return el;
}
const marksIn = (el: HTMLElement) => Array.from(el.querySelectorAll('mark.zen-ref'));

describe('wrapHighlights with a range', () => {
  it('marks the located span and only that span', () => {
    const root = host('<p>alpha beta alpha</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'alpha', start: 11, end: 16 }]);
    const marks = marksIn(root);
    expect(marks).toHaveLength(1);
    expect(marks[0].textContent).toBe('alpha');
    expect(marks[0].getAttribute('data-ref-id')).toBe('r1');
    // The SECOND occurrence — the one the offsets name, not the first match.
    expect((root.querySelector('p') as HTMLElement).innerHTML).toMatch(/^alpha beta <mark/);
    root.remove();
  });

  // The defect a text search could never fix: indexOf works inside one text
  // node, so a drag across a <strong> produced a reference that never painted.
  it('marks a span crossing two elements as several marks sharing one id', () => {
    const root = host('<p>Hello <strong>brave</strong> world</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'brave wo', start: 6, end: 14 }]);
    const marks = marksIn(root);
    expect(marks.length).toBeGreaterThan(1);
    expect(marks.every((n) => n.getAttribute('data-ref-id') === 'r1')).toBe(true);
    expect(marks.map((n) => n.textContent).join('')).toBe('brave wo');
    root.remove();
  });

  it('refuses to paint when the text at the offsets does not match', () => {
    const root = host('<p>alpha beta</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'gamma', start: 0, end: 5 }]);
    expect(marksIn(root)).toHaveLength(0);
    root.remove();
  });

  // An edit BEFORE the span shifts it; the checksum catches that.
  it('refuses to paint when an edit shifted the span', () => {
    const root = host('<p>prefix added: alpha beta</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'alpha', start: 0, end: 5 }]);
    expect(marksIn(root)).toHaveLength(0);
    root.remove();
  });

  // An edit AFTER the span leaves earlier offsets valid — it still paints.
  it('still paints when the edit landed after the span', () => {
    const root = host('<p>alpha beta plus a whole lot of new trailing text</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'alpha', start: 0, end: 5 }]);
    expect(marksIn(root)).toHaveLength(1);
    root.remove();
  });

  it('tolerates an end beyond the rendered length', () => {
    const root = host('<p>short</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'short text', start: 0, end: 10 }]);
    expect(marksIn(root)).toHaveLength(0);
    root.remove();
  });

  it('is idempotent across repeated passes', () => {
    const root = host('<p>alpha beta</p>');
    const h: Highlight[] = [{ id: 'r1', text: 'beta', start: 6, end: 10 }];
    wrapHighlights(root, h);
    wrapHighlights(root, h);
    expect(marksIn(root)).toHaveLength(1);
    expect(root.textContent).toBe('alpha beta');
    root.remove();
  });

  it('handles multiple distinct highlights', () => {
    const root = host('<p>alpha beta gamma</p>');
    wrapHighlights(root, [
      { id: 'a', text: 'alpha', start: 0, end: 5 },
      { id: 'g', text: 'gamma', start: 11, end: 16 },
    ]);
    const marks = marksIn(root);
    expect(marks).toHaveLength(2);
    expect(marks.map((m) => m.getAttribute('data-ref-id')).sort()).toEqual(['a', 'g']);
    root.remove();
  });
});

describe('wrapHighlights with no range', () => {
  it('wraps a text that occurs exactly once', () => {
    const root = host('<p>The quick brown fox</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'quick' }]);
    const marks = marksIn(root);
    expect(marks).toHaveLength(1);
    expect(marks[0].getAttribute('data-ref-id')).toBe('r1');
    expect(marks[0].textContent).toBe('quick');
    root.remove();
  });

  // Behavior change from v1.1.0: this used to mark all three occurrences, at
  // most one of which was the span the user meant. Ambiguous is now unpainted.
  it('marks nothing when the text is ambiguous', () => {
    const root = host('<p>foo bar foo bar foo</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'foo' }]);
    expect(marksIn(root)).toHaveLength(0);
    root.remove();
  });

  it('does nothing when text is not found', () => {
    const root = host('<p>hello</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'goodbye' }]);
    expect(marksIn(root)).toHaveLength(0);
    root.remove();
  });

  it('is idempotent — wrapping twice still produces one mark', () => {
    const root = host('<p>hello world</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'hello' }]);
    wrapHighlights(root, [{ id: 'r1', text: 'hello' }]);
    expect(marksIn(root)).toHaveLength(1);
    root.remove();
  });
});
