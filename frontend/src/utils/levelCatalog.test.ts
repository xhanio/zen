import { describe, it, expect } from 'vitest';
import { sortCatalog, findByWeight, findByName, findById } from './levelCatalog';

describe('sortCatalog', () => {
  it('sorts by weight ascending', () => {
    const out = sortCatalog([
      { id: 'B', weight: 1, name: 'b' },
      { id: 'M', weight: 0.5, name: 'm' },
      { id: 'A', weight: 0, name: 'a' },
    ]);
    expect(out.map((e) => e.name)).toEqual(['a', 'm', 'b']);
  });
});

describe('findByWeight', () => {
  it('returns the entry matching the float', () => {
    const cat = [
      { id: 'A', weight: 0, name: 'a' },
      { id: 'M', weight: 0.5, name: 'm' },
    ];
    expect(findByWeight(cat, 0.5)?.name).toBe('m');
    expect(findByWeight(cat, 9)).toBeUndefined();
  });
});

describe('findByName', () => {
  it('returns the entry matching the name', () => {
    const cat = [
      { id: 'A', weight: 0, name: 'a' },
      { id: 'B', weight: 1, name: 'b' },
    ];
    expect(findByName(cat, 'b')?.weight).toBe(1);
    expect(findByName(cat, 'x')).toBeUndefined();
  });
});

describe('findById', () => {
  it('returns the entry matching the id', () => {
    const cat = [
      { id: 'A', weight: 0, name: 'a' },
      { id: 'B', weight: 1, name: 'b' },
    ];
    expect(findById(cat, 'B')?.name).toBe('b');
    expect(findById(cat, 'ZZ')).toBeUndefined();
  });
});
