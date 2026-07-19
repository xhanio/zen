import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import SectionGradePill from './SectionGradePill.vue';
import type { Card, ReviewGrade } from '../types/entity';
import { useCardsStore } from '../stores/cards';

vi.mock('../api/client', () => ({
  listCards: vi.fn(),
  getCard: vi.fn(),
  createCard: vi.fn(),
  updateCard: vi.fn(),
  deleteCard: vi.fn(),
  restoreCard: vi.fn(),
  purgeCard: vi.fn(),
  listChildren: vi.fn(),
  reorderCard: vi.fn(),
  reviewCard: vi.fn(),
}));

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

function makeCard(grade: ReviewGrade): Card {
  return {
    id: 'c1', title: 't', content: 'body', summary: '', format: 'html',
    level_entry_id: null, genesis: '', deleted_at: null, group_id: 'g1', position: 0,
    tags: [], parent_card_id: 'p1', source_conversation_id: null,
    created_at: '', updated_at: '',
    review_grade: grade, review_score: null, reviewed_at: null,
  };
}

describe('SectionGradePill', () => {
  it('renders three pills with the correct active state', () => {
    const w = mount(SectionGradePill, { props: { card: makeCard('DIGESTED') } });
    const btns = w.findAll('button');
    expect(btns).toHaveLength(3);
    expect(btns[0].classes()).not.toContain('is-active'); // LGTM
    expect(btns[1].classes()).toContain('is-active');     // Digested
    expect(btns[2].classes()).not.toContain('is-active'); // Grilled
  });

  it('calls setReviewGrade when an inactive pill is clicked', async () => {
    const store = useCardsStore();
    const spy = vi.spyOn(store, 'setReviewGrade').mockResolvedValue(makeCard('GRILLED'));
    const w = mount(SectionGradePill, { props: { card: makeCard('LGTM') } });
    await w.findAll('button')[2].trigger('click');
    expect(spy).toHaveBeenCalledWith('c1', 'GRILLED');
  });

  it('is a no-op when the active pill is clicked', async () => {
    const store = useCardsStore();
    const spy = vi.spyOn(store, 'setReviewGrade');
    const w = mount(SectionGradePill, { props: { card: makeCard('DIGESTED') } });
    await w.findAll('button')[1].trigger('click');
    expect(spy).not.toHaveBeenCalled();
  });
});

// The pill lives in a 34px gutter now. Initials fit; three stacked words do not.
// The colour carries the meaning at a glance; the letter disambiguates; the
// accessible name says what it actually is.
describe('SectionGradePill — vertical form', () => {
  it('stacks the buttons and labels them with the full grade name', () => {
    const w = mount(SectionGradePill, { props: { card: makeCard('LGTM') } });
    expect(w.find('[data-test="grade-pill"]').classes()).toContain('flex-col');
    const btns = w.findAll('button');
    expect(btns.map((b) => b.text())).toEqual(['LGTM', 'Digested', 'Grilled']);
  });

  it('keeps the full grade word as the accessible name', () => {
    const w = mount(SectionGradePill, { props: { card: makeCard('LGTM') } });
    const btns = w.findAll('button');
    expect(btns.map((b) => b.attributes('aria-label'))).toEqual([
      'LGTM', 'Digested', 'Grilled',
    ]);
    expect(btns[2].attributes('title')).toBe('Grilled');
  });
});
