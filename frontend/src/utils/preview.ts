// previewText turns a card or document's stored content into a short
// plain-text preview suitable for grid / list cards. For HTML format it
// mirrors the backend's htmltext.Strip projection (the FTS5 search_hint
// column): tags dropped, only visible text kept. For markdown / text /
// unspecified format, the raw content reads well enough as-is.
//
// Output is trimmed, whitespace-collapsed, and truncated to maxChars with
// an ellipsis.
export function previewText(content: string, format?: string | null, maxChars = 140): string {
  if (!content) return '';

  let plain: string;
  if (format === 'html' && typeof DOMParser !== 'undefined') {
    const doc = new DOMParser().parseFromString(content, 'text/html');
    // Strip subtrees whose textContent we never want surfaced.
    doc.querySelectorAll('style, script').forEach((el) => el.remove());
    // Force whitespace between block-level elements so "<h1>Title</h1><p>Hi</p>"
    // renders as "Title Hi" instead of "TitleHi". textContent doesn't honor
    // block formatting; this is the same trick browsers' innerText does.
    doc.querySelectorAll(
      'address, article, aside, blockquote, br, div, dl, dt, dd, fieldset, figure, figcaption, footer, h1, h2, h3, h4, h5, h6, header, hr, li, main, nav, ol, p, pre, section, table, tr, td, th, ul',
    ).forEach((el) => el.append(' '));
    plain = doc.body?.textContent ?? '';
  } else {
    plain = content;
  }

  const trimmed = plain.trim().replace(/\s+/g, ' ');
  return trimmed.length > maxChars ? trimmed.slice(0, maxChars) + '…' : trimmed;
}
