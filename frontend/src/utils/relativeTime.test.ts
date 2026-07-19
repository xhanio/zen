import { describe, it, expect } from 'vitest';
import { relativeTime } from './relativeTime';

const NOW = Date.parse('2026-07-15T12:00:00Z');
const ago = (s: number) => new Date(NOW - s * 1000).toISOString();

describe('relativeTime', () => {
  it('empty / invalid input → empty string', () => {
    expect(relativeTime('', NOW)).toBe('');
    expect(relativeTime(null, NOW)).toBe('');
    expect(relativeTime('not-a-date', NOW)).toBe('');
  });
  it('under a minute → "just now"', () => {
    expect(relativeTime(ago(30), NOW)).toBe('just now');
  });
  it('minutes / hours / days / months', () => {
    expect(relativeTime(ago(5 * 60), NOW)).toBe('5m ago');
    expect(relativeTime(ago(3 * 3600), NOW)).toBe('3h ago');
    expect(relativeTime(ago(2 * 86400), NOW)).toBe('2d ago');
    expect(relativeTime(ago(40 * 86400), NOW)).toBe('1mo ago');
  });
});
