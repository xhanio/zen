// The group's "default card format" is stored as one managed sentence inside
// the group's free-text rule. This module composes/parses that sentence so the
// UI can expose it as a control without a separate backend field.

export type CardFormat = 'text' | 'markdown' | 'html';

const LABEL: Record<CardFormat, string> = { text: 'plain text', markdown: 'Markdown', html: 'HTML' };
const FROM_LABEL: Record<string, CardFormat> = { 'plain text': 'text', Markdown: 'markdown', HTML: 'html' };

// A single line, anchored, matched anywhere in the rule via the /m flag.
const MANAGED_RE = /^By default, cards in this group should be (plain text|Markdown|HTML)\.$/m;

export function formatSentence(format: CardFormat): string {
  return `By default, cards in this group should be ${LABEL[format]}.`;
}

export function composeRule(body: string, format: CardFormat | null): string {
  const parts = [body.trim()];
  if (format) parts.push(formatSentence(format));
  return parts.filter(Boolean).join('\n\n');
}

export function parseRuleFormat(rule: string): { body: string; format: CardFormat | null } {
  const m = rule.match(MANAGED_RE);
  if (!m) return { body: rule, format: null };
  const body = rule.replace(MANAGED_RE, '').replace(/\n{3,}/g, '\n\n').trim();
  return { body, format: FROM_LABEL[m[1]] };
}
