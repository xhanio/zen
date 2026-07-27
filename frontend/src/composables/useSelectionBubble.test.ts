import { describe, it, expect } from 'vitest';
import { rangeInside, findCardId, measureSelection, bodyRootFor } from './useSelectionBubble';

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

describe('measureSelection', () => {
  // Builds the shape CardView renders for an html-format card: the
  // [data-card-id] section wrapping HtmlBody's .html-body-host shadow root.
  function htmlCard(id: string, markup: string) {
    const section = document.createElement('section');
    section.setAttribute('data-card-id', id);
    const host = document.createElement('div');
    host.className = 'html-body-host';
    section.appendChild(host);
    document.body.appendChild(section);
    const shadow = host.attachShadow({ mode: 'open' });
    shadow.innerHTML = markup;
    return { section, host, shadow };
  }

  it('measures a markdown card body from the document-level range', () => {
    const section = document.createElement('section');
    section.setAttribute('data-card-id', 'md1');
    section.innerHTML = '<p>Hello brave world</p>';
    document.body.appendChild(section);

    const t = section.querySelector('p')!.firstChild as Text;
    const range = document.createRange();
    range.setStart(t, 6);
    range.setEnd(t, 11);

    expect(measureSelection('md1', range, 'brave')).toEqual({ start: 6, end: 11 });
    section.remove();
  });

  // The v1.1.1 defect: Chrome retargets a shadow-tree selection to the host,
  // so the document-level range's boundaries are the host element and never
  // the text nodes inside. Every html-format card recorded no range.
  it('measures an html card when the range was retargeted to the host', () => {
    const { section, host, shadow } = htmlCard('h1', '<p>Hello brave world</p>');

    // The real selection, as ShadowRoot.getSelection() reports it.
    const inner = document.createRange();
    const t = shadow.querySelector('p')!.firstChild as Text;
    inner.setStart(t, 6);
    inner.setEnd(t, 11);
    (shadow as ShadowRoot & { getSelection?: () => unknown }).getSelection = () => ({
      rangeCount: 1,
      getRangeAt: () => inner,
    });

    // What window.getSelection() hands us: collapsed onto the host.
    const retargeted = document.createRange();
    const idx = Array.from(host.parentNode!.childNodes).indexOf(host);
    retargeted.setStart(host.parentNode!, idx);
    retargeted.setEnd(host.parentNode!, idx + 1);

    expect(measureSelection('h1', retargeted, 'brave')).toEqual({ start: 6, end: 11 });
    section.remove();
  });

  // The fallback must not accept whatever the shadow root happens to hold —
  // the checksum against the reported text is what makes it safe.
  it('records no range when the shadow selection disagrees with the text', () => {
    const { section, shadow } = htmlCard('h2', '<p>Hello brave world</p>');
    const inner = document.createRange();
    const t = shadow.querySelector('p')!.firstChild as Text;
    inner.setStart(t, 0);
    inner.setEnd(t, 5); // "Hello", not the reported "brave"
    (shadow as ShadowRoot & { getSelection?: () => unknown }).getSelection = () => ({
      rangeCount: 1,
      getRangeAt: () => inner,
    });

    const retargeted = document.createRange();
    retargeted.selectNodeContents(section);

    expect(measureSelection('h2', retargeted, 'brave')).toBeNull();
    section.remove();
  });

  it('records no range when the card body is not in the document', () => {
    const range = document.createRange();
    expect(measureSelection('absent', range, 'brave')).toBeNull();
  });
});

describe('bodyRootFor inside a container section', () => {
  // A section in container view renders its TITLE through HtmlBody too, and
  // that host precedes the body host in the DOM. Picking the first match
  // measured every drag against the title, so the range came out null and no
  // underline was ever recorded for a selection made in a container.
  function section(id: string) {
    const el = document.createElement('section');
    el.setAttribute('data-card-id', id);

    const titleHost = document.createElement('div');
    titleHost.className = 'html-body-host zen-title-html';
    el.appendChild(titleHost);
    const titleRoot = titleHost.attachShadow({ mode: 'open' });
    titleRoot.innerHTML = '<h2>决策</h2>';

    const bodyHost = document.createElement('div');
    bodyHost.className = 'html-body-host';
    el.appendChild(bodyHost);
    const bodyRoot = bodyHost.attachShadow({ mode: 'open' });
    bodyRoot.innerHTML = '<p>hello brave world</p>';

    document.body.appendChild(el);
    return { el, titleRoot, bodyRoot };
  }

  it('returns the body shadow root, not the title one', () => {
    const { el, bodyRoot } = section('sec1');
    expect(bodyRootFor('sec1')).toBe(bodyRoot);
    el.remove();
  });

  it('still works for a leaf card with only a body host', () => {
    const el = document.createElement('section');
    el.setAttribute('data-card-id', 'leaf1');
    const host = document.createElement('div');
    host.className = 'html-body-host';
    el.appendChild(host);
    const root = host.attachShadow({ mode: 'open' });
    root.innerHTML = '<p>leaf body</p>';
    document.body.appendChild(el);

    expect(bodyRootFor('leaf1')).toBe(root);
    el.remove();
  });
});
