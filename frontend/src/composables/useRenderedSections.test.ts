import { describe, it, expect, beforeEach } from 'vitest';
import { ref } from 'vue';
import { setActivePinia, createPinia } from 'pinia';
import { useRenderedSections } from './useRenderedSections';
import { useCardsStore } from '../stores/cards';
import { useTilePrefsStore } from '../stores/tilePrefs';
import { useContainerFilterStore } from '../stores/containerFilter';

const child = (id: string, position: number, extra: Record<string, unknown> = {}) => ({
  id, title: id, content: 'x', summary: '', format: 'markdown' as const,
  level_entry_id: null, group_id: 'g1', position, tags: [], genesis: '',
  deleted_at: null, parent_card_id: 'p1', source_conversation_id: null,
  created_at: 'x', updated_at: 'x',
  review_grade: 'LGTM' as const, review_score: null, reviewed_at: null,
  ...extra,
});

describe('useRenderedSections', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('sorts by position and drops trashed children by default', () => {
    const cards = useCardsStore();
    cards.byChildren['p1'] = [
      child('b', 1),
      child('a', 0),
      child('gone', 2, { deleted_at: '2026-01-01' }),
    ] as never;
    const rendered = useRenderedSections(ref('p1'));
    expect(rendered.value.map((c) => c.id)).toEqual(['a', 'b']);
  });

  it('keeps trashed children when the pref is on', () => {
    const cards = useCardsStore();
    useTilePrefsStore().showTrashedSections = true;
    cards.byChildren['p1'] = [child('a', 0), child('gone', 1, { deleted_at: 'x' })] as never;
    expect(useRenderedSections(ref('p1')).value.map((c) => c.id)).toEqual(['a', 'gone']);
  });

  it('drops collapsed cards (by id) so the gutter skips them', () => {
    const cards = useCardsStore();
    cards.byChildren['p1'] = [child('a', 0), child('b', 1)] as never;
    useContainerFilterStore().toggleCard('b');
    expect(useRenderedSections(ref('p1')).value.map((c) => c.id)).toEqual(['a']);
  });
});
