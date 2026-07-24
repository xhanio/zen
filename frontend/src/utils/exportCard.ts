import type { Card, EntityFormat } from '../types/entity';

// A pre-fetched card and its ordered children — the input to serialization.
// Built by useCardExport.buildNode; serializeCard itself never does I/O.
export interface ExportNode {
  card: Card;
  children: ExportNode[];
}

// format → file extension + bare MIME type (charset is appended at download).
export const EXPORT_FORMATS: Record<EntityFormat, { ext: string; mime: string }> = {
  markdown: { ext: 'md', mime: 'text/markdown' },
  html: { ext: 'html', mime: 'text/html' },
  text: { ext: 'txt', mime: 'text/plain' },
};

function fmt(format: EntityFormat): { ext: string; mime: string } {
  return EXPORT_FORMATS[format] ?? EXPORT_FORMATS.markdown;
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// The title rendered as a heading of the given depth, in the file's format.
// Only the title is format-encoded; node content is always emitted raw.
function headingFor(title: string, depth: number, format: EntityFormat): string {
  const level = Math.min(depth, 6);
  if (format === 'html') return `<h${level}>${escapeHtml(title)}</h${level}>`;
  if (format === 'text') return title;
  return `${'#'.repeat(level)} ${title}`; // markdown
}

// Depth-first: this node's heading, then its verbatim content (if any), then
// each child one level deeper. `format` is the ROOT card's format for every
// node — the scaffolding (headings/spacing) is uniform across the document.
function appendBlocks(node: ExportNode, depth: number, format: EntityFormat, out: string[]): void {
  out.push(headingFor(node.card.title, depth, format));
  const content = (node.card.content ?? '').trim();
  if (content) out.push(content);
  for (const child of node.children) appendBlocks(child, depth + 1, format, out);
}

// Standalone-document CSS for HTML exports: a centered 800px content card on a
// tinted page background, system font, and automatic dark mode. Applied only to
// the html format so the file reads well when opened directly; md/text stay plain.
const HTML_EXPORT_STYLE = `
  :root { color-scheme: light dark; }
  * { box-sizing: border-box; }
  html {
    background: #ebedf0;
    padding: 40px 16px;
  }
  body {
    max-width: 800px;
    margin: 0 auto;
    padding: 28px 36px;
    background: #ffffff;
    border-radius: 6px;
    box-shadow: 0 1px 8px rgba(0, 0, 0, 0.12);
    font-family: system-ui, -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    line-height: 1.6;
    color: #1a1a1a;
  }
  h1 { font-size: 1.7rem; line-height: 1.25; margin: 0 0 0.6em; padding-bottom: 0.3em; border-bottom: 1px solid #e2e5e9; }
  h2 { line-height: 1.3; margin-top: 1.6em; }
  h3 { margin-top: 1.2em; color: #333333; }
  a { color: #0b66c3; }
  img, pre, table { max-width: 100%; }
  pre { overflow-x: auto; }
  @media (prefers-color-scheme: dark) {
    html { background: #0d0e10; }
    body { background: #17191c; color: #e6e7e9; box-shadow: 0 1px 12px rgba(0, 0, 0, 0.5); }
    h1, h2 { color: #f4f5f6; }
    h1 { border-bottom-color: #2c2f34; }
    h3 { color: #c7c9cc; }
    a { color: #7bb3ff; }
  }`;

// Wrap an HTML body fragment in a standalone, styled document: title tab,
// charset, responsive viewport, the reading-column stylesheet, and dark mode.
// The title is HTML-escaped.
function htmlDocument(bodyHtml: string, title: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${escapeHtml(title)}</title>
<style>${HTML_EXPORT_STYLE}
</style>
</head>
<body>
${bodyHtml}
</body>
</html>
`;
}

export function serializeCard(node: ExportNode, format: EntityFormat): string {
  const out: string[] = [];
  appendBlocks(node, 1, format, out);
  // html exports become a full, styled document; md/text stay plain fragments.
  if (format === 'html') {
    return htmlDocument(out.join('\n'), node.card.title);
  }
  return out.join('\n\n') + '\n';
}

export function filenameFor(title: string, format: EntityFormat): string {
  const base = title
    .replace(/[\/\\:*?"<>|\x00-\x1F]/g, '') // strip illegal + control chars
    .replace(/\s+/g, ' ')                    // collapse whitespace runs
    .trim()
    .replace(/^\.+|\.+$/g, '')               // strip leading/trailing dots
    .trim()
    .slice(0, 120)
    .trim();
  return `${base || 'card'}.${fmt(format).ext}`;
}

// The one impure helper: write `text` to the user's machine as `filename`.
// Not unit-tested (jsdom/happy-dom has no real download) — covered by the
// Playwright e2e.
export function downloadText(filename: string, mime: string, text: string): void {
  const blob = new Blob([text], { type: `${mime};charset=utf-8` });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
