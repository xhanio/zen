import { collectTextNodes, renderedText } from './selectionRange';

export interface Highlight {
  id: string;
  text: string;
  // Character offsets into the body's rendered text. Absent or null means the
  // reference has no range — see the no-range path below.
  start?: number | null;
  end?: number | null;
}

// Wrap the referenced span with <mark class="zen-ref" data-ref-id="…"> for
// click delegation. Never paints a span it cannot verify:
//
//   ranged    → paint iff renderedText.slice(start, end) === text
//   no range  → paint iff the text occurs EXACTLY once
//
// The second rule is a deliberate change from marking every occurrence: a
// phrase appearing three times produced three marks, at most one of which was
// the span the user selected.
export function wrapHighlights(root: ParentNode, highlights: Highlight[]): void {
  const full = renderedText(root as Node);
  for (const h of highlights) {
    const span = locate(full, h);
    if (!span) continue;
    paint(root as Node, span.start, span.end, h.id);
  }
}

function locate(full: string, h: Highlight): { start: number; end: number } | null {
  if (h.start != null && h.end != null) {
    if (h.end <= h.start) return null;
    return full.slice(h.start, h.end) === h.text ? { start: h.start, end: h.end } : null;
  }
  if (!h.text) return null;
  const first = full.indexOf(h.text);
  if (first === -1) return null;
  if (full.indexOf(h.text, first + 1) !== -1) return null; // ambiguous — a guess, so don't
  return { start: first, end: first + h.text.length };
}

function paint(root: Node, start: number, end: number, id: string): void {
  // Snapshot the node windows BEFORE mutating: splitText inserts siblings, so
  // a live walk would revisit fragments it had just created.
  const windows: Array<{ node: Text; from: number; to: number }> = [];
  let acc = 0;
  for (const node of collectTextNodes(root)) {
    const nodeStart = acc;
    const nodeEnd = acc + node.data.length;
    acc = nodeEnd;
    if (nodeEnd <= start || nodeStart >= end) continue;
    const parent = node.parentElement;
    if (
      parent &&
      parent.tagName === 'MARK' &&
      parent.classList.contains('zen-ref') &&
      parent.getAttribute('data-ref-id') === id
    ) {
      continue; // already painted for this reference — keeps the pass idempotent
    }
    windows.push({
      node,
      from: Math.max(0, start - nodeStart),
      to: Math.min(node.data.length, end - nodeStart),
    });
  }

  for (const w of windows) {
    let target: Text = w.node;
    if (w.from > 0) target = target.splitText(w.from);
    if (w.to - w.from < target.data.length) target.splitText(w.to - w.from);
    const mark = document.createElement('mark');
    mark.className = 'zen-ref';
    mark.setAttribute('data-ref-id', id);
    target.parentNode!.replaceChild(mark, target);
    mark.appendChild(target);
  }
}
