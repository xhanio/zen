import { describe, it, expect } from 'vitest';
import { formatSentence, composeRule, parseRuleFormat, type CardFormat } from './groupRuleFormat';

describe('groupRuleFormat', () => {
  it('formatSentence per format', () => {
    expect(formatSentence('text')).toBe('By default, cards in this group should be plain text.');
    expect(formatSentence('markdown')).toBe('By default, cards in this group should be Markdown.');
    expect(formatSentence('html')).toBe('By default, cards in this group should be HTML.');
  });

  it('composeRule appends the sentence after the trimmed body', () => {
    expect(composeRule('  hi  ', 'html')).toBe('hi\n\nBy default, cards in this group should be HTML.');
  });
  it('composeRule with null format is the trimmed body', () => {
    expect(composeRule('  hi  ', null)).toBe('hi');
  });
  it('composeRule with empty body is just the sentence', () => {
    expect(composeRule('', 'markdown')).toBe('By default, cards in this group should be Markdown.');
  });

  it('parseRuleFormat strips the managed line and detects the format', () => {
    expect(parseRuleFormat('hi\n\nBy default, cards in this group should be HTML.'))
      .toEqual({ body: 'hi', format: 'html' });
  });
  it('parseRuleFormat returns None when no managed line matches', () => {
    expect(parseRuleFormat('Card must be in format of HTML'))
      .toEqual({ body: 'Card must be in format of HTML', format: null });
  });
  it('parseRuleFormat on a body-only rule', () => {
    expect(parseRuleFormat('just prose')).toEqual({ body: 'just prose', format: null });
  });

  it('round-trips for every format including null', () => {
    const body = 'some rule text';
    for (const fmt of ['text', 'markdown', 'html', null] as (CardFormat | null)[]) {
      expect(parseRuleFormat(composeRule(body, fmt))).toEqual({ body, format: fmt });
    }
  });
});
