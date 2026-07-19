import { describe, it, expect } from 'vitest';
import { mount, RouterLinkStub } from '@vue/test-utils';
import RecentDocumentCard from './RecentDocumentCard.vue';
import type { Card } from '../../types/entity';

function card(over: Partial<Card>): Card {
  return {
    id: 'x', title: 'T', content: '', summary: '', format: 'markdown', level_entry_id: null,
    genesis: '', deleted_at: null, group_id: 'g1', position: 0, tags: [],
    parent_card_id: null, source_conversation_id: null, created_at: '', updated_at: '',
    review_grade: 'LGTM', review_score: null, reviewed_at: null, ...over,
  } as Card;
}

const mountCard = (doc: Card, sections = 3) =>
  mount(RecentDocumentCard, {
    props: { doc, groupName: 'Design', sections },
    global: { stubs: { RouterLink: RouterLinkStub, 'router-link': RouterLinkStub } },
  });

describe('RecentDocumentCard', () => {
  it('shows title, group, section count and links to the card', () => {
    const w = mountCard(card({ id: 'doc', title: 'My Doc', summary: 'a summary' }));
    expect(w.text()).toContain('My Doc');
    expect(w.text()).toContain('a summary');
    expect(w.text()).toContain('3 sections');
    expect(w.text()).toContain('Design');
    expect(w.getComponent(RouterLinkStub).props('to')).toEqual({ name: 'card', params: { cardId: 'doc' } });
  });

  it('falls back to genesis when there is no summary', () => {
    const w = mountCard(card({ summary: '', genesis: 'born from a chat' }));
    expect(w.text()).toContain('born from a chat');
  });
});
