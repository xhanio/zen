// Measuring only — no DOM mutation. Painting lives in highlightText.ts.
//
// The offset space is the card body's rendered text: text nodes under the body
// root in document order. Two rules keep it stable:
//
//   * <style> and <script> are skipped. HtmlBody appends a <style> into the
//     shadow root and Design-group cards carry their own; counting that CSS
//     would shift every offset after it.
//   * existing .zen-ref marks are traversed INTO. Their text is still body
//     text — skipping it would move every offset once a highlight is painted,
//     so the same span would measure differently before and after.

const SKIP_TAGS = new Set(['STYLE', 'SCRIPT']);

export function collectTextNodes(root: Node, out: Text[] = []): Text[] {
  for (let child = root.firstChild; child; child = child.nextSibling) {
    if (child.nodeType === Node.TEXT_NODE) {
      out.push(child as Text);
    } else if (child.nodeType === Node.ELEMENT_NODE) {
      if (SKIP_TAGS.has((child as Element).tagName)) continue;
      collectTextNodes(child, out);
    }
  }
  return out;
}

export function renderedText(root: Node): string {
  let s = '';
  for (const n of collectTextNodes(root)) s += n.data;
  return s;
}

// Null when the range is collapsed, inverted, or not inside root — all of
// which mean "not anchorable", not "error".
export function offsetsForRange(root: Node, range: Range): { start: number; end: number } | null {
  let start = -1;
  let end = -1;
  let acc = 0;
  for (const n of collectTextNodes(root)) {
    if (n === range.startContainer) start = acc + range.startOffset;
    if (n === range.endContainer) end = acc + range.endOffset;
    acc += n.data.length;
  }
  if (start < 0 || end < 0 || end <= start) return null;
  return { start, end };
}

// The selection text is reported trimmed, so the window must be trimmed too or
// the paint-side checksum can never match.
export function trimToText(
  full: string,
  start: number,
  end: number,
): { start: number; end: number } {
  let s = start;
  let e = end;
  while (s < e && /\s/.test(full[s])) s++;
  while (e > s && /\s/.test(full[e - 1])) e--;
  return { start: s, end: e };
}
