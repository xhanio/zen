import { describe, it, expect } from 'vitest';
import { previewText } from './preview';

describe('previewText', () => {
  it('returns "" for empty content', () => {
    expect(previewText('')).toBe('');
    expect(previewText('   ', 'markdown')).toBe('');
  });

  it('passes markdown / text content through (with whitespace collapse)', () => {
    expect(previewText('hello    world\nnext line', 'markdown')).toBe('hello world next line');
    expect(previewText('raw log\t\tdata', 'text')).toBe('raw log data');
  });

  it('strips HTML tags and keeps only visible text for format=html', () => {
    const html = '<style>p{color:red}</style><h1>Title</h1><p>Hello <strong>world</strong>.</p>';
    expect(previewText(html, 'html')).toBe('Title Hello world.');
  });

  it('keeps inline SVG <text> content', () => {
    const html = '<svg><text>banana</text><text>apple</text></svg>';
    expect(previewText(html, 'html')).toContain('banana');
    expect(previewText(html, 'html')).toContain('apple');
  });

  it('truncates long previews with an ellipsis', () => {
    const long = 'a'.repeat(200);
    const out = previewText(long, 'markdown', 50);
    expect(out.length).toBe(51); // 50 chars + ellipsis
    expect(out.endsWith('…')).toBe(true);
  });

  it('treats unknown / missing format as plain', () => {
    expect(previewText('plain content', undefined)).toBe('plain content');
    expect(previewText('plain content', null)).toBe('plain content');
  });
});
