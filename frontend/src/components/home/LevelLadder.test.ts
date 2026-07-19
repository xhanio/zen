import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import LevelLadder from './LevelLadder.vue';
import type { Card, LevelEntry } from '../../types/entity';

function card(over: Partial<Card>): Card {
  return {
    id: 'x', title: 'T', content: '', summary: '', format: 'markdown', level_entry_id: null,
    genesis: '', deleted_at: null, group_id: 'g1', position: 0, tags: [],
    parent_card_id: null, source_conversation_id: null, created_at: '', updated_at: '',
    review_grade: 'LGTM', review_score: null, reviewed_at: null, ...over,
  } as Card;
}
const catalog: LevelEntry[] = [
  { id: 'e0', weight: 0, name: 'Alpha' },
  { id: 'e1', weight: 1, name: 'Beta' },
];

describe('LevelLadder', () => {
  it('renders one segment per level weight with proportional widths', () => {
    const cards = [
      card({ id: 'a', level_entry_id: 'e0' }),
      card({ id: 'b', level_entry_id: 'e0' }),
      card({ id: 'c', level_entry_id: 'e1' }),
    ];
    const w = mount(LevelLadder, { props: { catalog, cards } });
    const seg0 = w.get('[data-test="ladder-seg-0"]');
    const seg1 = w.get('[data-test="ladder-seg-1"]');
    expect(seg0.attributes('style')).toContain('width: 66.66');
    expect(seg1.attributes('style')).toContain('width: 33.33');
    expect(w.text()).toContain('Alpha · 2');
    expect(w.text()).toContain('Beta · 1');
  });

  it('shows "No levels defined" for an empty catalog', () => {
    const w = mount(LevelLadder, { props: { catalog: [], cards: [] } });
    expect(w.find('[data-test="ladder-empty"]').exists()).toBe(true);
    expect(w.find('[data-test="level-ladder"]').exists()).toBe(false);
  });
});
