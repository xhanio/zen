import { describe, it, expect } from 'vitest';
import { wrapHighlights } from './highlightText';

function host(html: string): HTMLElement {
  const el = document.createElement('div');
  el.innerHTML = html;
  document.body.appendChild(el);
  return el;
}

describe('wrapHighlights', () => {
  it('wraps a single match in <mark> with data-ref-id', () => {
    const root = host('<p>The quick brown fox</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'quick' }]);
    const marks = root.querySelectorAll('mark.zen-ref');
    expect(marks.length).toBe(1);
    expect(marks[0].getAttribute('data-ref-id')).toBe('r1');
    expect(marks[0].textContent).toBe('quick');
    root.remove();
  });

  it('wraps every occurrence of the same text', () => {
    const root = host('<p>foo bar foo bar foo</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'foo' }]);
    const marks = root.querySelectorAll('mark.zen-ref');
    expect(marks.length).toBe(3);
    root.remove();
  });

  it('does nothing when text is not found', () => {
    const root = host('<p>hello</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'goodbye' }]);
    expect(root.querySelectorAll('mark.zen-ref').length).toBe(0);
    root.remove();
  });

  it('is idempotent — wrapping twice still produces one mark per match', () => {
    const root = host('<p>hello world</p>');
    wrapHighlights(root, [{ id: 'r1', text: 'hello' }]);
    wrapHighlights(root, [{ id: 'r1', text: 'hello' }]);
    expect(root.querySelectorAll('mark.zen-ref').length).toBe(1);
    root.remove();
  });

  it('handles multiple distinct highlights', () => {
    const root = host('<p>alpha beta gamma</p>');
    wrapHighlights(root, [
      { id: 'a', text: 'alpha' },
      { id: 'g', text: 'gamma' },
    ]);
    const marks = root.querySelectorAll('mark.zen-ref');
    expect(marks.length).toBe(2);
    expect(Array.from(marks).map((m) => m.getAttribute('data-ref-id')).sort())
      .toEqual(['a', 'g']);
    root.remove();
  });
});
