import { describe, it, expect, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { mount, RouterLinkStub } from '@vue/test-utils';
import CardItem from './CardItem.vue';
import { useCardsStore } from '../stores/cards';
import type { Card } from '../types/entity';

function stub(over: Partial<Card> = {}): Card {
  return {
    id: 'c1', title: 'T', content: '', summary: '', format: 'markdown', level_entry_id: null,
    genesis: '', deleted_at: null, group_id: 'g1', position: 0, tags: [],
    parent_card_id: null, source_conversation_id: null, created_at: '', updated_at: '',
    review_grade: '', review_score: null, reviewed_at: null,
    ...over,
  } as unknown as Card;
}

const mountItem = (card: Card, props: Record<string, unknown> = {}) =>
  mount(CardItem, {
    props: { card, ...props },
    global: {
      stubs: {
        'router-link': RouterLinkStub,
        SectionOutline: { template: '<div data-test="outline"></div>' },
      },
    },
  });

beforeEach(() => setActivePinia(createPinia()));

describe('CardItem', () => {
  it('is draggable by default', () => {
    const w = mountItem(stub({ content: 'body' }));
    expect(w.find('article').attributes('draggable')).toBe('true');
  });

  it('is not draggable when pinned', () => {
    const w = mountItem(stub({ content: 'body' }), { pinned: true });
    expect(w.find('article').attributes('draggable')).toBe('false');
  });

  it('shows the section outline for a container even with content (preamble)', () => {
    const s = useCardsStore();
    const parent = stub({ id: 'p', content: 'Date: 2026', group_id: 'g1' });
    s.byGroup['g1'] = [parent, stub({ id: 'k1', parent_card_id: 'p', group_id: 'g1', content: 'x' })];
    const w = mountItem(parent);
    expect(w.find('[data-test="outline"]').exists()).toBe(true);
  });
});
