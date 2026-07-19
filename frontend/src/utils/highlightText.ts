export interface Highlight {
  id: string;
  text: string;
}

// Wrap every occurrence of each highlight's text inside `root` with
// <mark class="zen-ref" data-ref-id="..."> for click delegation.
// Idempotent: text already inside a .zen-ref mark is skipped.
export function wrapHighlights(root: ParentNode, highlights: Highlight[]): void {
  for (const h of highlights) {
    if (!h.text) continue;
    wrapOne(root, h);
  }
}

function collectTextNodes(root: Node, out: Text[]): void {
  for (let child = root.firstChild; child; child = child.nextSibling) {
    if (child.nodeType === Node.TEXT_NODE) {
      out.push(child as Text);
    } else if (child.nodeType === Node.ELEMENT_NODE) {
      const el = child as Element;
      if (el.tagName === 'MARK' && el.classList.contains('zen-ref')) continue;
      collectTextNodes(child, out);
    }
  }
}

function wrapOne(root: ParentNode, h: Highlight): void {
  const texts: Text[] = [];
  collectTextNodes(root as Node, texts);
  for (const tn of texts) {
    let current: Text = tn;
    while (true) {
      const idx = current.data.indexOf(h.text);
      if (idx === -1) break;
      const matchNode = current.splitText(idx);
      const tail = matchNode.splitText(h.text.length);
      const mark = document.createElement('mark');
      mark.className = 'zen-ref';
      mark.setAttribute('data-ref-id', h.id);
      mark.textContent = h.text;
      matchNode.parentNode!.replaceChild(mark, matchNode);
      current = tail;
    }
  }
}
