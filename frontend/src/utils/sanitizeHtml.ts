import sanitize from 'sanitize-html';

// v0.4 allowlist: declarative HTML5 + inline SVG + MathML + inline <style>.
// No <script>, no on* handlers, no <iframe>/<object>/<embed>, no
// <link rel="stylesheet">. javascript:/vbscript: URLs are stripped.
//
// We keep the surface small on purpose — the spec's "rich visualizations"
// goal is met by inline SVG; everything beyond static markup is out of
// scope.
const ALLOWED_TAGS = [
  // Block layout
  'address', 'article', 'aside', 'blockquote', 'div', 'figure', 'figcaption',
  'footer', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'header', 'hgroup', 'hr',
  'main', 'nav', 'p', 'pre', 'section',
  // Lists
  'ol', 'ul', 'li', 'dl', 'dd', 'dt',
  // Tables
  'table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td', 'caption', 'colgroup', 'col',
  // Inline
  'a', 'abbr', 'b', 'bdi', 'bdo', 'br', 'cite', 'code', 'data', 'dfn', 'em',
  'i', 'kbd', 'mark', 'q', 's', 'samp', 'small', 'span', 'strong', 'sub',
  'sup', 'time', 'u', 'var', 'wbr',
  // Embedded
  'img',
  // Inline CSS for self-contained styling inside the shadow root
  'style',
  // SVG. NOTE: sanitize-html lowercases tag names before allowlist matching,
  // so camelCase SVG tags (linearGradient, radialGradient, clipPath) MUST be
  // listed lowercase here. The HTML parser case-corrects them back to
  // camelCase when the SVG is parsed in foreign-content mode, so the DOM ends
  // up with proper SVG-namespaced elements anyway.
  'svg', 'g', 'path', 'rect', 'circle', 'ellipse', 'line', 'polyline',
  'polygon', 'text', 'tspan', 'defs', 'lineargradient', 'radialgradient',
  'stop', 'clippath', 'mask', 'pattern', 'use', 'symbol', 'title', 'desc',
  // MathML
  'math', 'mrow', 'mi', 'mn', 'mo', 'msup', 'msub', 'msubsup', 'mfrac',
  'msqrt', 'mroot', 'mtext', 'mspace',
];

export function sanitizeCardHtml(raw: string): string {
  return sanitize(raw, {
    // <style> is on the allowlist intentionally: cards render inside a
    // closed shadow root (HtmlBody.vue), so card-scoped CSS doesn't leak
    // out and SPA CSS doesn't leak in. allowVulnerableTags acknowledges
    // sanitize-html's general warning about <style> exfil risk.
    allowVulnerableTags: true,
    allowedTags: ALLOWED_TAGS,
    allowedAttributes: {
      '*': ['class', 'style', 'id', 'lang', 'dir', 'title'],
      a: ['href', 'name', 'target', 'rel'],
      img: ['src', 'alt', 'width', 'height'],
      table: ['summary'],
      td: ['colspan', 'rowspan', 'align', 'valign'],
      th: ['colspan', 'rowspan', 'align', 'valign', 'scope'],
      // SVG attribute allowlist — sanitize-html requires explicit allow per tag.
      // BOTH keys (tag names) and values (attribute names) are matched
      // lowercase against incoming markup, so camelCase SVG attribute names
      // MUST be written lowercase here. Browsers case-correct them back to
      // camelCase when the SVG is parsed in foreign-content mode.
      svg: ['viewbox', 'width', 'height', 'xmlns', 'preserveaspectratio', 'fill'],
      path: ['d', 'fill', 'stroke', 'stroke-width', 'stroke-linecap', 'stroke-linejoin', 'opacity', 'transform'],
      rect: ['x', 'y', 'width', 'height', 'rx', 'ry', 'fill', 'stroke', 'stroke-width', 'opacity', 'transform'],
      circle: ['cx', 'cy', 'r', 'fill', 'stroke', 'stroke-width', 'opacity', 'transform'],
      ellipse: ['cx', 'cy', 'rx', 'ry', 'fill', 'stroke', 'stroke-width', 'opacity', 'transform'],
      line: ['x1', 'y1', 'x2', 'y2', 'stroke', 'stroke-width', 'opacity', 'transform'],
      polyline: ['points', 'fill', 'stroke', 'stroke-width', 'opacity', 'transform'],
      polygon: ['points', 'fill', 'stroke', 'stroke-width', 'opacity', 'transform'],
      text: ['x', 'y', 'dx', 'dy', 'text-anchor', 'fill', 'font-size', 'font-family', 'transform'],
      tspan: ['x', 'y', 'dx', 'dy', 'text-anchor', 'fill', 'font-size'],
      g: ['transform', 'fill', 'stroke', 'opacity'],
      defs: [],
      lineargradient: ['id', 'x1', 'y1', 'x2', 'y2', 'gradientunits'],
      radialgradient: ['id', 'cx', 'cy', 'r', 'fx', 'fy', 'gradientunits'],
      stop: ['offset', 'stop-color', 'stop-opacity'],
      use: ['href', 'x', 'y', 'width', 'height'],
      symbol: ['id', 'viewbox', 'width', 'height'],
    },
    allowedSchemes: ['data', 'http', 'https'],
    allowedSchemesByTag: {
      img: ['data', 'http', 'https'],
      a: ['http', 'https', 'mailto'],
    },
    transformTags: {
      a: sanitize.simpleTransform('a', { rel: 'noopener noreferrer' }),
    },
  });
}
