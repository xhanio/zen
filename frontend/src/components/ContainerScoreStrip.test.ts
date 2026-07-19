import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import ContainerScoreStrip from './ContainerScoreStrip.vue';
import type { Card } from '../types/entity';

const base: Card = {
  id: 'c1', title: 't', content: '', format: 'html',
  summary: '', level_entry_id: null, genesis: '', deleted_at: null,
  group_id: 'g', position: 0, tags: [], parent_card_id: null,
  source_conversation_id: null, created_at: '', updated_at: '',
  review_grade: 'LGTM', review_score: null, reviewed_at: null,
};

describe('ContainerScoreStrip', () => {
  it('renders X.X / 100 when review_score is a number', () => {
    const w = mount(ContainerScoreStrip, { props: { card: { ...base, review_score: 47.5 } } });
    expect(w.text()).toContain('47.5');
    expect(w.text()).toContain('/ 100');
  });

  it('renders em-dash when review_score is null', () => {
    const w = mount(ContainerScoreStrip, { props: { card: { ...base, review_score: null } } });
    expect(w.text()).toContain('—');
  });

  it('sets fill --pct to the score value', () => {
    const w = mount(ContainerScoreStrip, { props: { card: { ...base, review_score: 47.5 } } });
    const fill = w.find('[data-test="fill"]').element as HTMLElement;
    expect(fill.style.getPropertyValue('--pct')).toBe('47.5');
  });

  it('omits the fill entirely when score is null', () => {
    const w = mount(ContainerScoreStrip, { props: { card: { ...base, review_score: null } } });
    expect(w.find('[data-test="fill"]').exists()).toBe(false);
  });

  it('renders 100.0 at max score', () => {
    const w = mount(ContainerScoreStrip, { props: { card: { ...base, review_score: 100 } } });
    expect(w.text()).toContain('100.0');
  });
});
