import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { mount, RouterLinkStub, flushPromises } from '@vue/test-utils';
import GroupView from './GroupView.vue';
import { useGroupsStore } from '../stores/groups';
import { useTilePrefsStore } from '../stores/tilePrefs';
import type { Card } from '../types/entity';

vi.mock('../api/client', () => ({
  listCards: vi.fn(),
  listTags: vi.fn().mockResolvedValue([]),
  deleteCard: vi.fn(),
  createCard: vi.fn(),
}));

import { listCards } from '../api/client';

function card(over: Partial<Card>): Card {
  return {
    id: 'x', title: 'T', content: '', summary: '', format: 'markdown', level_entry_id: null,
    genesis: '', deleted_at: null, group_id: 'g1', position: 0, tags: [],
    parent_card_id: null, source_conversation_id: null, created_at: '', updated_at: '',
    review_grade: '', review_score: null, reviewed_at: null,
    ...over,
  } as unknown as Card;
}

const mountView = () =>
  mount(GroupView, {
    props: { groupId: 'g1' },
    global: {
      stubs: {
        'router-link': RouterLinkStub,
        ConfirmDialog: true,
        LevelColumn: {
          props: ['cards', 'label', 'isMisc'],
          template: '<div class="lc"><span v-for="c in cards" :key="c.id" :data-col-card="c.id"></span></div>',
        },
        CardItem: {
          props: ['card', 'pinned'],
          template: '<div :data-strip-card="card.id"></div>',
        },
      },
    },
  });

beforeEach(() => {
  localStorage.clear();
  setActivePinia(createPinia());
  const groups = useGroupsStore();
  groups.groups = [
    { id: 'g1', name: 'Design', level_catalog: [], position: 0, rule: '', created_at: '', updated_at: '' },
  ] as unknown as typeof groups.groups;
});

describe('GroupView — Documents strip', () => {
  it('lifts top-level containers into the strip and out of the columns', async () => {
    (listCards as any).mockResolvedValue([
      card({ id: 'doc', parent_card_id: null }),
      card({ id: 'sec', parent_card_id: 'doc' }),
      card({ id: 'leaf', parent_card_id: null }),
    ]);
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="documents-column"]').exists()).toBe(true);
    expect(w.find('[data-strip-card="doc"]').exists()).toBe(true);
    expect(w.find('[data-col-card="doc"]').exists()).toBe(false);
    expect(w.find('[data-col-card="leaf"]').exists()).toBe(true);
  });

  it('renders no strip when the group has no containers', async () => {
    (listCards as any).mockResolvedValue([
      card({ id: 'leaf1', parent_card_id: null }),
      card({ id: 'leaf2', parent_card_id: null }),
    ]);
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="documents-column"]').exists()).toBe(false);
  });
});

describe('GroupView — documents/levels splitter', () => {
  it('renders the splitter only when documents exist', async () => {
    (listCards as any).mockResolvedValue([
      card({ id: 'doc', parent_card_id: null }),
      card({ id: 'sec', parent_card_id: 'doc' }),
    ]);
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="documents-splitter"]').exists()).toBe(true);
  });

  it('hides the splitter when the group has no documents', async () => {
    (listCards as any).mockResolvedValue([card({ id: 'leaf', parent_card_id: null })]);
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="documents-splitter"]').exists()).toBe(false);
  });

  it('binds the persisted documents width to the column', async () => {
    useTilePrefsStore().setDocumentsWidth(320);
    (listCards as any).mockResolvedValue([
      card({ id: 'doc', parent_card_id: null }),
      card({ id: 'sec', parent_card_id: 'doc' }),
    ]);
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="documents-column"]').attributes('style')).toContain('width: 320px');
  });
});
