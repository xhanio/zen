import { onMounted, onBeforeUnmount, ref, type Ref } from 'vue';

// True if `range` is fully contained inside `target` OR inside the open
// shadow tree of any `.html-body-host` descendant of `target`. HtmlBody.vue
// renders sanitized card HTML into an open shadow root so document-level
// Selection sees user drags inside the card body.
export function rangeInside(target: HTMLElement, range: Range): boolean {
  if (target.contains(range.startContainer) && target.contains(range.endContainer)) {
    return true;
  }
  const hosts = target.querySelectorAll('.html-body-host');
  for (const host of Array.from(hosts)) {
    const sr = (host as HTMLElement).shadowRoot;
    if (sr && sr.contains(range.startContainer) && sr.contains(range.endContainer)) {
      return true;
    }
  }
  return false;
}

// Chrome retargets shadow-tree selections to the host element, so the
// document-level range has start === end === host and zero-sized rect.
// Fall back to ShadowRoot.getSelection() (non-standard but widely shipped)
// for the real selection rect.
export function selectionRect(target: HTMLElement, docRange: Range): DOMRect | null {
  let r = docRange.getBoundingClientRect();
  if (r.width > 0 || r.height > 0) return r;
  const hosts = target.querySelectorAll('.html-body-host');
  for (const host of Array.from(hosts)) {
    const sr = (host as HTMLElement).shadowRoot as (ShadowRoot & { getSelection?: () => Selection | null }) | null;
    if (!sr || typeof sr.getSelection !== 'function') continue;
    const sel = sr.getSelection();
    if (!sel || sel.rangeCount === 0) continue;
    r = sel.getRangeAt(0).getBoundingClientRect();
    if (r.width > 0 || r.height > 0) return r;
  }
  return null;
}

// Walk up the DOM from `node`, bridging shadow roots via their host, and
// return the nearest ancestor's `data-card-id` attribute. Used so that a
// selection inside a container's rendered child gets anchored to that
// child card, not the parent container that CardView is showing.
export function findCardId(node: Node | null): string | null {
  let cur: Node | null = node;
  while (cur) {
    if (cur.nodeType === Node.ELEMENT_NODE) {
      const id = (cur as Element).getAttribute('data-card-id');
      if (id) return id;
    }
    if (cur.parentNode) {
      cur = cur.parentNode;
    } else {
      const host = (cur as unknown as { host?: Element }).host;
      cur = host ?? null;
    }
  }
  return null;
}

export function useSelectionBubble(targetRef: Ref<HTMLElement | null>) {
  const rect = ref<DOMRect | null>(null);
  const text = ref('');
  const hostCardId = ref<string | null>(null);
  let timer: ReturnType<typeof setTimeout> | null = null;

  function clear() {
    rect.value = null;
    text.value = '';
    hostCardId.value = null;
  }

  function onSelectionChange() {
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => {
      const sel = window.getSelection();
      // NOTE: do NOT bail on sel.isCollapsed. Chrome retargets shadow-tree
      // selections to the host element, making range.start === range.end and
      // tripping isCollapsed even when toString() returns the actual selected
      // text. The text-emptiness check below is the real gate.
      if (!sel || sel.rangeCount === 0) {
        clear();
        return;
      }
      const target = targetRef.value;
      if (!target) {
        clear();
        return;
      }
      const t = sel.toString().trim();
      if (!t) {
        clear();
        return;
      }
      const range = sel.getRangeAt(0);
      if (!rangeInside(target, range)) {
        clear();
        return;
      }
      const r = selectionRect(target, range);
      if (!r) {
        clear();
        return;
      }
      rect.value = r;
      text.value = t;
      hostCardId.value = findCardId(range.startContainer);
    }, 100);
  }

  onMounted(() => {
    document.addEventListener('selectionchange', onSelectionChange);
  });
  onBeforeUnmount(() => {
    document.removeEventListener('selectionchange', onSelectionChange);
    if (timer) clearTimeout(timer);
  });

  return { rect, text, hostCardId, clear };
}
