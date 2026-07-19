import { describe, it, expect } from 'vitest';
import { rangeInside, findCardId } from './useSelectionBubble';

describe('rangeInside', () => {
  it('returns true when both endpoints are inside target', () => {
    const target = document.createElement('div');
    const p = document.createElement('p');
    p.textContent = 'hello world';
    target.appendChild(p);
    document.body.appendChild(target);

    const range = document.createRange();
    range.selectNodeContents(p);
    expect(rangeInside(target, range)).toBe(true);

    target.remove();
  });

  it('returns false when selection is outside target', () => {
    const target = document.createElement('div');
    const outside = document.createElement('p');
    outside.textContent = 'outside';
    document.body.append(target, outside);

    const range = document.createRange();
    range.selectNodeContents(outside);
    expect(rangeInside(target, range)).toBe(false);

    target.remove();
    outside.remove();
  });

  it('returns true when selection is inside an html-body-host shadow root descendant', () => {
    const target = document.createElement('div');
    const host = document.createElement('div');
    host.className = 'html-body-host';
    target.appendChild(host);
    document.body.appendChild(target);

    const shadow = host.attachShadow({ mode: 'open' });
    const p = document.createElement('p');
    p.textContent = 'shadow text';
    shadow.appendChild(p);

    const range = document.createRange();
    range.selectNodeContents(p);
    expect(rangeInside(target, range)).toBe(true);

    target.remove();
  });

  it('returns false when shadow descendant exists but selection is elsewhere', () => {
    const target = document.createElement('div');
    const host = document.createElement('div');
    host.className = 'html-body-host';
    target.appendChild(host);
    const outside = document.createElement('p');
    outside.textContent = 'outside';
    document.body.append(target, outside);

    const shadow = host.attachShadow({ mode: 'open' });
    shadow.innerHTML = '<p>shadow</p>';

    const range = document.createRange();
    range.selectNodeContents(outside);
    expect(rangeInside(target, range)).toBe(false);

    target.remove();
    outside.remove();
  });
});

describe('findCardId', () => {
  it('returns the closest ancestor data-card-id', () => {
    const section = document.createElement('section');
    section.setAttribute('data-card-id', '01ABCCHILD');
    const inner = document.createElement('div');
    const p = document.createElement('p');
    p.textContent = 'body text';
    inner.appendChild(p);
    section.appendChild(inner);
    document.body.appendChild(section);

    expect(findCardId(p.firstChild)).toBe('01ABCCHILD');
    section.remove();
  });

  it('bridges shadow-root boundaries via host', () => {
    const section = document.createElement('section');
    section.setAttribute('data-card-id', '01ABCCHILD');
    const host = document.createElement('div');
    section.appendChild(host);
    document.body.appendChild(section);
    const shadow = host.attachShadow({ mode: 'open' });
    const span = document.createElement('span');
    span.textContent = 'selected';
    shadow.appendChild(span);

    expect(findCardId(span.firstChild)).toBe('01ABCCHILD');
    section.remove();
  });

  it('returns null when nothing carries data-card-id', () => {
    const outer = document.createElement('div');
    const inner = document.createElement('p');
    inner.textContent = 'x';
    outer.appendChild(inner);
    document.body.appendChild(outer);
    expect(findCardId(inner.firstChild)).toBe(null);
    outer.remove();
  });
});
