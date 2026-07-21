import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { createRouter, createMemoryHistory, type Router } from 'vue-router';
import GroupNav from './GroupNav.vue';
import { useGroupsStore } from '../stores/groups';
import { useLevelFilterStore } from '../stores/levelFilter';
import { useCardsStore } from '../stores/cards';
import type { Card } from '../types/entity';

vi.mock('../api/client', () => ({ listCards: vi.fn().mockResolvedValue([]) }));

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'home', component: { template: '<div />' } },
      { path: '/g/:groupId', name: 'group', component: { template: '<div />' } },
    ],
  });
}
function card(over: Partial<Card>): Card {
  return {
    id: 'x', title: 'T', content: '', summary: '', format: 'markdown', level_entry_id: null,
    genesis: '', deleted_at: null, group_id: 'G1', position: 0, tags: [],
    parent_card_id: null, source_conversation_id: null, created_at: '', updated_at: '',
    review_grade: 'LGTM', review_score: null, reviewed_at: null, ...over,
  } as Card;
}
// Mount at a given path so route-driven expansion can be exercised.
async function mountAt(path: string) {
  const router = makeRouter();
  await router.push(path);
  await router.isReady();
  return mount(GroupNav, { global: { plugins: [router], stubs: { GroupEditDialog: true } } });
}

beforeEach(() => setActivePinia(createPinia()));

describe('GroupNav', () => {
  it('renders one row per group', async () => {
    const store = useGroupsStore();
    store.groups = [
      { id: 'G1', name: 'design', rule: '', position: 0, level_catalog: [], created_at: '', updated_at: '' },
      { id: 'G2', name: 'workflow', rule: '', position: 0, level_catalog: [], created_at: '', updated_at: '' },
    ];
    const w = await mountAt('/');
    expect(w.findAll('[data-test="group-row"]').length).toBe(2);
  });

  it('the current-route group shows its level rows + Misc + spine', async () => {
    const store = useGroupsStore();
    store.groups = [{
      id: 'G1', name: 'design', rule: '', position: 0,
      level_catalog: [{ id: 'E1', weight: 0, name: '原则' }, { id: 'E2', weight: 1, name: '决策' }],
      created_at: '', updated_at: '',
    }];
    const w = await mountAt('/g/G1');
    expect(w.find('[data-test="level-G1-E1"]').exists()).toBe(true);
    expect(w.find('[data-test="level-G1-E2"]').exists()).toBe(true);
    expect(w.find('[data-test="level-G1-misc"]').exists()).toBe(true);
    expect(w.find('[data-test="level-spine"]').exists()).toBe(true);
  });

  it('a group you are not viewing stays collapsed', async () => {
    const store = useGroupsStore();
    store.groups = [{
      id: 'G1', name: 'design', rule: '', position: 0,
      level_catalog: [{ id: 'E1', weight: 0, name: '原则' }],
      created_at: '', updated_at: '',
    }];
    const w = await mountAt('/');
    expect(w.find('[data-test="level-G1-E1"]').exists()).toBe(false);
  });

  it('clicking a selected level row deselects it in the filter store', async () => {
    const store = useGroupsStore();
    store.groups = [{
      id: 'G1', name: 'design', rule: '', position: 0,
      level_catalog: [{ id: 'E1', weight: 0, name: '原则' }],
      created_at: '', updated_at: '',
    }];
    const filter = useLevelFilterStore();
    const w = await mountAt('/g/G1');
    await w.find('[data-test="level-G1-E1"]').trigger('click'); // starts selected → deselect
    expect(filter.byGroup['G1']?.selectedEntryIds).toEqual([]);
  });

  it('shows card and document counts once cards are loaded', async () => {
    const store = useGroupsStore();
    store.groups = [{ id: 'G1', name: 'design', rule: '', position: 0, level_catalog: [], created_at: '', updated_at: '' }];
    const cards = useCardsStore();
    cards.byGroup['G1'] = [
      card({ id: 'doc', parent_card_id: null }),
      card({ id: 'sec', parent_card_id: 'doc' }),
      card({ id: 'leaf', parent_card_id: null }),
    ];
    const w = await mountAt('/');
    await flushPromises();
    const row = w.find('[data-test="group-row"]').text();
    expect(row).toContain('3'); // 3 live cards
    expect(row).toContain('1'); // 1 document
  });

  it('counts each level by its own entry, not by shared weight (same-score levels)', async () => {
    const store = useGroupsStore();
    store.groups = [{
      id: 'G1', name: 'design', rule: '', position: 0,
      // Two levels at the same weight (score) — a supported case.
      level_catalog: [{ id: 'E1', weight: 0, name: '原则' }, { id: 'E2', weight: 0, name: '决策' }],
      created_at: '', updated_at: '',
    }];
    const cards = useCardsStore();
    cards.byGroup['G1'] = [
      card({ id: 'a', level_entry_id: 'E1' }),
      card({ id: 'b', level_entry_id: 'E2' }),
      card({ id: 'c', level_entry_id: 'E2' }),
    ];
    const w = await mountAt('/g/G1');
    await flushPromises();
    const e1 = w.find('[data-test="level-G1-E1"]').text();
    const e2 = w.find('[data-test="level-G1-E2"]').text();
    expect(e1).toContain('1'); // E1 owns exactly one card
    expect(e1).not.toContain('3'); // not the combined same-weight total
    expect(e2).toContain('2'); // E2 owns exactly two cards
    expect(e2).not.toContain('3');
  });

  it('the Edit group button opens the edit dialog', async () => {
    const store = useGroupsStore();
    store.groups = [{ id: 'G1', name: 'design', rule: '', position: 0, level_catalog: [], created_at: '', updated_at: '' }];
    const w = await mountAt('/');
    expect(w.find('[data-test="edit-group-overlay"]').exists()).toBe(false);
    await w.find('[aria-label="Edit group design"]').trigger('click');
    expect(w.find('[data-test="edit-group-overlay"]').exists()).toBe(true);
  });
});
