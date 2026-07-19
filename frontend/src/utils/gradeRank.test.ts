import { describe, it, expect } from 'vitest';
import { gradeRank } from './gradeRank';

describe('gradeRank', () => {
  it('orders ungraded < LGTM < DIGESTED < GRILLED', () => {
    expect(gradeRank('')).toBe(0);
    expect(gradeRank('LGTM')).toBe(1);
    expect(gradeRank('DIGESTED')).toBe(2);
    expect(gradeRank('GRILLED')).toBe(3);
    expect(gradeRank('') < gradeRank('LGTM')).toBe(true);
    expect(gradeRank('LGTM') < gradeRank('DIGESTED')).toBe(true);
    expect(gradeRank('DIGESTED') < gradeRank('GRILLED')).toBe(true);
  });

  it('treats an unknown grade as 0', () => {
    expect(gradeRank('WAT' as unknown as '')).toBe(0);
  });
});
