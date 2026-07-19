import { mount } from '@vue/test-utils';
import { describe, it, expect, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import LevelColumn from './LevelColumn.vue';
import type { Card } from '../../types/entity';

beforeEach(() => {
  setActivePinia(createPinia());
});

function makeCard(over: Partial<Card> = {}): Card {
  return {
    id: 'C1',
    title: 'hello',
    summary: '',
    content: 'world',
    format: 'markdown',
    level_entry_id: null,
    genesis: '',
    deleted_at: null,
    group_id: 'G1',
    position: 0,
    tags: [],
    parent_card_id: null,
    source_conversation_id: null,
    created_at: '',
    updated_at: '',
    review_grade: 'LGTM',
    review_score: null,
    reviewed_at: null,
    ...over,
  };
}

describe('LevelColumn', () => {
  it('renders the column header label', () => {
    const w = mount(LevelColumn, {
      props: { label: '原则', cards: [], isMisc: false },
      global: {
        stubs: { CardItem: { template: '<div></div>', props: ['card', 'hidePill'] } },
      },
    });
    expect(w.text()).toContain('原则');
  });

  it('renders each card in vertical stack', () => {
    const w = mount(LevelColumn, {
      props: {
        label: '原则',
        cards: [makeCard({ id: 'A', title: 'card-a' }), makeCard({ id: 'B', title: 'card-b' })],
        isMisc: false,
      },
      global: {
        stubs: {
          CardItem: {
            template: '<div data-test="card">{{ card.title }}</div>',
            props: ['card', 'hidePill'],
          },
        },
      },
    });
    expect(w.findAll('[data-test="card"]')).toHaveLength(2);
  });

  it('emits card-dropped with clearLevel=true when isMisc=true', async () => {
    const w = mount(LevelColumn, {
      props: { label: 'Misc', cards: [], isMisc: true },
      global: {
        stubs: { CardItem: { template: '<div></div>', props: ['card', 'hidePill'] } },
      },
    });
    const dz = w.find('[data-test="column-dropzone"]').element;
    const drop = new Event('drop', { bubbles: true }) as unknown as DragEvent;
    Object.defineProperty(drop, 'dataTransfer', {
      value: { getData: () => 'CARD-X' },
    });
    Object.defineProperty(drop, 'preventDefault', { value: () => {} });
    dz.dispatchEvent(drop);
    await w.vm.$nextTick();
    const events = w.emitted('card-dropped')!;
    expect(events[0]).toEqual(['CARD-X', { clearLevel: true }]);
  });

  it('emits card-dropped when the drop lands on the column header (h3)', async () => {
    const w = mount(LevelColumn, {
      props: { label: '原则', cards: [], isMisc: false, weight: 0 },
      global: {
        stubs: { CardItem: { template: '<div></div>', props: ['card', 'hidePill'] } },
      },
    });
    // Drop event bubbles up from the header to the column-wide handler.
    const h3 = w.find('h3').element;
    const drop = new Event('drop', { bubbles: true }) as unknown as DragEvent;
    Object.defineProperty(drop, 'dataTransfer', { value: { getData: () => 'CARD-Z' } });
    Object.defineProperty(drop, 'preventDefault', { value: () => {} });
    h3.dispatchEvent(drop);
    await w.vm.$nextTick();
    expect(w.emitted('card-dropped')![0]).toEqual(['CARD-Z', { entryId: undefined, clearLevel: false }]);
  });

  it('emits card-dropped with entry id set when isMisc=false', async () => {
    const w = mount(LevelColumn, {
      props: { label: '原则', cards: [], isMisc: false, weight: 0, entryId: 'ENTRY-A' },
      global: {
        stubs: { CardItem: { template: '<div></div>', props: ['card', 'hidePill'] } },
      },
    });
    const dz = w.find('[data-test="column-dropzone"]').element;
    const drop = new Event('drop', { bubbles: true }) as unknown as DragEvent;
    Object.defineProperty(drop, 'dataTransfer', {
      value: { getData: () => 'CARD-Y' },
    });
    Object.defineProperty(drop, 'preventDefault', { value: () => {} });
    dz.dispatchEvent(drop);
    await w.vm.$nextTick();
    const events = w.emitted('card-dropped')!;
    expect(events[0]).toEqual(['CARD-Y', { entryId: 'ENTRY-A', clearLevel: false }]);
  });

  it('same-weight columns emit distinct entry ids', async () => {
    // Two columns share weight=2 but have different entry ids. The drop
    // must resolve to the column's own id, not collapse via weight.
    const colA = mount(LevelColumn, {
      props: { label: '决策', cards: [], isMisc: false, weight: 2, entryId: 'ENTRY-A' },
      global: { stubs: { CardItem: { template: '<div></div>', props: ['card', 'hidePill'] } } },
    });
    const colB = mount(LevelColumn, {
      props: { label: '细节', cards: [], isMisc: false, weight: 2, entryId: 'ENTRY-B' },
      global: { stubs: { CardItem: { template: '<div></div>', props: ['card', 'hidePill'] } } },
    });
    const drop = () => {
      const ev = new Event('drop', { bubbles: true }) as unknown as DragEvent;
      Object.defineProperty(ev, 'dataTransfer', { value: { getData: () => 'CARD-X' } });
      Object.defineProperty(ev, 'preventDefault', { value: () => {} });
      return ev;
    };
    colA.find('[data-test="column-dropzone"]').element.dispatchEvent(drop());
    colB.find('[data-test="column-dropzone"]').element.dispatchEvent(drop());
    await colA.vm.$nextTick();
    await colB.vm.$nextTick();
    expect(colA.emitted('card-dropped')![0]).toEqual(['CARD-X', { entryId: 'ENTRY-A', clearLevel: false }]);
    expect(colB.emitted('card-dropped')![0]).toEqual(['CARD-X', { entryId: 'ENTRY-B', clearLevel: false }]);
  });
});
