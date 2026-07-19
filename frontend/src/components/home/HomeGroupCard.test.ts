import { describe, it, expect } from 'vitest';
import { mount, RouterLinkStub } from '@vue/test-utils';
import HomeGroupCard from './HomeGroupCard.vue';
import type { Card, Group } from '../../types/entity';

function card(over: Partial<Card>): Card {
  return {
    id: 'x', title: 'T', content: '', summary: '', format: 'markdown', level_entry_id: null,
    genesis: '', deleted_at: null, group_id: 'g1', position: 0, tags: [],
    parent_card_id: null, source_conversation_id: null, created_at: '', updated_at: '',
    review_grade: 'LGTM', review_score: null, reviewed_at: null, ...over,
  } as Card;
}
const group: Group = {
  id: 'g1', name: 'Design', rule: '', position: 0,
  level_catalog: [{ id: 'e0', weight: 0, name: 'A' }], created_at: '', updated_at: '',
};

const mountCard = (cards: Card[]) =>
  mount(HomeGroupCard, {
    props: { group, cards },
    global: { stubs: { RouterLink: RouterLinkStub, 'router-link': RouterLinkStub, LevelLadder: true } },
  });

describe('HomeGroupCard', () => {
  it('shows name, counts and links to the group board', () => {
    const cards = [
      card({ id: 'doc', parent_card_id: null }),
      card({ id: 'sec', parent_card_id: 'doc' }),
      card({ id: 'leaf', parent_card_id: null }),
    ];
    const w = mountCard(cards);
    expect(w.text()).toContain('Design');
    expect(w.text()).toContain('3 cards');
    expect(w.text()).toContain('1 documents');
    expect(w.text()).toContain('1 levels');
    expect(w.getComponent(RouterLinkStub).props('to')).toEqual({ name: 'group', params: { groupId: 'g1' } });
  });

  it('shows the most frequent tags', () => {
    const w = mountCard([card({ id: 'a', tags: ['api', 'ui'] }), card({ id: 'b', tags: ['api'] })]);
    expect(w.text()).toContain('api');
    expect(w.text()).toContain('ui');
  });
});
