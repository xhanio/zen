import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { mount, RouterLinkStub, flushPromises } from '@vue/test-utils';
import HomeView from './HomeView.vue';
import { useGroupsStore } from '../stores/groups';
import type { Card } from '../types/entity';

vi.mock('../api/client', () => ({ listCards: vi.fn() }));
import { listCards } from '../api/client';

function card(over: Partial<Card>): Card {
  return {
    id: 'x', title: 'T', content: '', summary: '', format: 'markdown', level_entry_id: null,
    genesis: '', deleted_at: null, group_id: 'g1', position: 0, tags: [],
    parent_card_id: null, source_conversation_id: null, created_at: '', updated_at: '',
    review_grade: 'LGTM', review_score: null, reviewed_at: null, ...over,
  } as Card;
}

const stubs = {
  RouterLink: RouterLinkStub,
  'router-link': RouterLinkStub,
  HomeGroupCard: { props: ['group', 'cards'], template: '<div :data-group="group.id" :data-n="cards.length"></div>' },
  RecentDocumentCard: { props: ['doc', 'groupName', 'sections'], template: '<div :data-recent="doc.id"></div>' },
};

function seedGroups() {
  const groups = useGroupsStore();
  groups.groups = [
    { id: 'g1', name: 'Design', level_catalog: [], position: 0, rule: '', created_at: '', updated_at: '' },
  ] as unknown as typeof groups.groups;
}

beforeEach(() => setActivePinia(createPinia()));

describe('HomeView', () => {
  it('shows stats, one group card per group, and recent documents newest-first', async () => {
    seedGroups();
    (listCards as any).mockResolvedValue([
      card({ id: 'doc1', parent_card_id: null, updated_at: '2026-01-01' }),
      card({ id: 'sec1', parent_card_id: 'doc1' }),
      card({ id: 'doc2', parent_card_id: null, updated_at: '2026-05-01' }),
      card({ id: 'sec2', parent_card_id: 'doc2' }),
      card({ id: 'leaf', parent_card_id: null }),
    ]);
    const w = mount(HomeView, { global: { stubs } });
    await flushPromises();

    const stats = w.get('[data-test="home-stats"]').text();
    expect(stats).toContain('1'); // groups
    expect(stats).toContain('5'); // cards
    expect(stats).toContain('2'); // documents

    expect(w.findAll('[data-group]')).toHaveLength(1);
    const recents = w.findAll('[data-recent]').map((n) => n.attributes('data-recent'));
    expect(recents).toEqual(['doc2', 'doc1']); // newest first, leaves excluded
  });

  it('renders an empty state when there are no groups', async () => {
    const w = mount(HomeView, { global: { stubs } });
    await flushPromises();
    expect(w.find('[data-test="home-empty"]').exists()).toBe(true);
  });
});
