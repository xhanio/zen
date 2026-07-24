import { describe, it, expect } from 'vitest';
import { serializeCard, filenameFor, EXPORT_FORMATS, type ExportNode } from './exportCard';
import type { Card, EntityFormat } from '../types/entity';

function card(overrides: Partial<Card> = {}): Card {
  return {
    id: 'c1', title: 'T', summary: '', content: '', format: 'markdown',
    level_entry_id: null, genesis: '', deleted_at: null, group_id: 'g1',
    position: 0, tags: [], parent_card_id: null, source_conversation_id: null,
    created_at: '', updated_at: '', review_grade: 'LGTM', review_score: null,
    reviewed_at: null, ...overrides,
  };
}
const leaf = (o: Partial<Card> = {}): ExportNode => ({ card: card(o), children: [] });
const node = (o: Partial<Card>, children: ExportNode[]): ExportNode => ({ card: card(o), children });

describe('serializeCard — leaf', () => {
  it('markdown leaf: H1 title + verbatim body + trailing newline', () => {
    expect(serializeCard(leaf({ title: 'T', content: 'hello' }), 'markdown')).toBe('# T\n\nhello\n');
  });
  it('text leaf: bare title line + body', () => {
    expect(serializeCard(leaf({ title: 'T', content: 'plain' }), 'text')).toBe('T\n\nplain\n');
  });
  it('html leaf: <h1> with HTML-escaped title + raw body', () => {
    expect(serializeCard(leaf({ title: 'A & B', content: '<p>hi</p>' }), 'html'))
      .toBe('<h1>A &amp; B</h1>\n<p>hi</p>\n');
  });
  it('empty leaf: heading only', () => {
    expect(serializeCard(leaf({ title: 'T', content: '' }), 'markdown')).toBe('# T\n');
  });
});

describe('serializeCard — container', () => {
  it('preamble + two sections, depth-based headings', () => {
    const tree = node({ title: 'Doc', content: 'Pre' }, [
      leaf({ title: 'A', content: 'a' }),
      leaf({ title: 'B', content: 'b' }),
    ]);
    expect(serializeCard(tree, 'markdown')).toBe('# Doc\n\nPre\n\n## A\n\na\n\n## B\n\nb\n');
  });
  it('empty preamble is skipped', () => {
    const tree = node({ title: 'Doc', content: '' }, [leaf({ title: 'A', content: 'a' })]);
    expect(serializeCard(tree, 'markdown')).toBe('# Doc\n\n## A\n\na\n');
  });
  it('recurses: nested container yields H3 under H2', () => {
    const tree = node({ title: 'Doc', content: '' }, [
      node({ title: 'Beta', content: '' }, [leaf({ title: 'Beta One', content: 'nested' })]),
    ]);
    expect(serializeCard(tree, 'markdown')).toBe('# Doc\n\n## Beta\n\n### Beta One\n\nnested\n');
  });
  it('child content is emitted verbatim (mixed format not converted)', () => {
    const tree = node({ title: 'Doc', content: '' }, [leaf({ title: 'H', content: '<p>raw</p>' })]);
    expect(serializeCard(tree, 'markdown')).toContain('<p>raw</p>');
  });
  it('heading depth clamps at 6', () => {
    // 7-deep chain: root(1) → … → leaf(7). Deepest heading must be '######' (6), not 7.
    let tree = leaf({ title: 'L7', content: 'x' });
    for (const t of ['L6', 'L5', 'L4', 'L3', 'L2', 'L1']) tree = node({ title: t, content: '' }, [tree]);
    const out = serializeCard(tree, 'markdown');
    expect(out).toContain('###### L6'); // depth 6
    expect(out).toContain('###### L7'); // depth 7 clamped to 6
    expect(out).not.toContain('####### '); // never 7 hashes
  });
});

describe('filenameFor', () => {
  it('keeps readable spaces + correct extension per format', () => {
    expect(filenameFor('Project Plan', 'markdown')).toBe('Project Plan.md');
    expect(filenameFor('Page', 'html')).toBe('Page.html');
    expect(filenameFor('Notes', 'text')).toBe('Notes.txt');
  });
  it('strips filesystem-illegal characters', () => {
    expect(filenameFor('a/b:c*?"<>|', 'markdown')).toBe('abc.md');
  });
  it('collapses whitespace and trims', () => {
    expect(filenameFor('  a   b  ', 'markdown')).toBe('a b.md');
  });
  it('strips leading/trailing dots', () => {
    expect(filenameFor('...hidden...', 'markdown')).toBe('hidden.md');
  });
  it('falls back to "card" when nothing survives sanitization', () => {
    expect(filenameFor('///', 'markdown')).toBe('card.md');
    expect(filenameFor('', 'markdown')).toBe('card.md');
  });
  it('caps the base name at 120 characters', () => {
    const base = filenameFor('x'.repeat(300), 'markdown').replace(/\.md$/, '');
    expect(base.length).toBe(120);
  });
  it('unknown format falls back to markdown extension', () => {
    expect(filenameFor('T', 'weird' as EntityFormat)).toBe('T.md');
  });
});

describe('EXPORT_FORMATS', () => {
  it('maps each format to ext + bare MIME', () => {
    expect(EXPORT_FORMATS.markdown).toEqual({ ext: 'md', mime: 'text/markdown' });
    expect(EXPORT_FORMATS.html).toEqual({ ext: 'html', mime: 'text/html' });
    expect(EXPORT_FORMATS.text).toEqual({ ext: 'txt', mime: 'text/plain' });
  });
});
