import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';

vi.mock('../api/client', () => ({
  listReferences: vi.fn(),
  getCard: vi.fn(),
  listCards: vi.fn(),
  listChildren: vi.fn().mockResolvedValue([]),
  listTags: vi.fn().mockResolvedValue([]),
  listGroups: vi.fn().mockResolvedValue([]),
  listSnapshots: vi.fn().mockResolvedValue({ snapshots: [] }),
  getSnapshot: vi.fn(),
}));

import CardReferencesPanel from './CardReferencesPanel.vue';
import { useCardsStore } from '../stores/cards';
import type { Card, Reference } from '../types/entity';

const card = (over: Partial<Card> = {}): Card =>
  ({
    id: 'c1', title: 'Source', content: 'alpha beta', summary: '', format: 'markdown',
    level_entry_id: null, group_id: 'g1', position: 0, tags: [], genesis: '',
    deleted_at: null, parent_card_id: null, source_conversation_id: null,
    created_at: '', updated_at: '', review_grade: 'LGTM', review_score: null,
    reviewed_at: null, references: [], ...over,
  }) as Card;

const ref = (over: Partial<Reference> = {}): Reference =>
  ({
    id: 'r1', source_card_id: 'c1', derived_card_id: 'c2', conversation_id: null,
    selection_text: 'anchored', selection_start: 3, selection_end: 11,
    selection_seq: 5, created_at: '', ...over,
  }) as Reference;

function mountPanel(references: Reference[]) {
  const cards = useCardsStore();
  const root = card({ references });
  cards.byID['c1'] = root;
  cards.byID['c2'] = card({ id: 'c2', title: 'Derived' });
  return mount(CardReferencesPanel, { props: { rootCard: root } });
}

const settle = () => new Promise((r) => setTimeout(r, 0));

describe('CardReferencesPanel snapshot label', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  // A reference is an annotation on a specific version; the marker says which,
  // and is the entry point for snapshot-view painting later.
  it('labels a reference with the snapshot it was taken against', async () => {
    const w = mountPanel([ref({ selection_seq: 5 })]);
    await settle();
    expect(w.find('[data-test="reference-seq"]').text()).toBe('#5');
  });

  it('omits the marker when the reference carries no snapshot label', async () => {
    const w = mountPanel([ref({ selection_start: null, selection_end: null, selection_seq: null })]);
    await settle();
    expect(w.find('[data-test="reference-seq"]').exists()).toBe(false);
  });
});
