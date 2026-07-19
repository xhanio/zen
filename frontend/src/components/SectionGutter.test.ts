import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount, RouterLinkStub } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import SectionGutter from './SectionGutter.vue';
import { useCardsStore } from '../stores/cards';
import type { Card, ReviewGrade } from '../types/entity';

const sec = (id: string, grade: ReviewGrade) => ({
  id, title: 'Section ' + id, content: 'x', summary: '', format: 'markdown' as const,
  level_entry_id: null, group_id: 'g1', position: 0, tags: [], genesis: '',
  deleted_at: null, parent_card_id: 'p1', source_conversation_id: null,
  created_at: 'x', updated_at: 'x',
  review_grade: grade, review_score: null, reviewed_at: null,
}) as Card;

// jsdom gives every element offsetTop 0 and a zeroed rect. Geometry has to be
// stubbed, or the assertions test jsdom rather than the component.
function fakeAnchor(offsetTop: number, viewportTop: number): HTMLElement {
  const el = document.createElement('section');
  Object.defineProperty(el, 'offsetTop', { value: offsetTop, configurable: true });
  // A real DOMRect carries bottom = top + height; the gutter now docks at the
  // section BOTTOM, so the stub must supply it too.
  el.getBoundingClientRect = () => ({ top: viewportTop, bottom: viewportTop + 100, height: 100 }) as DOMRect;
  return el;
}

function mountGutter(sections: Card[], anchors: HTMLElement[], scrollRoot: HTMLElement | null = null) {
  return mount(SectionGutter, {
    props: { sections, anchors, scrollRoot },
    global: { stubs: { RouterLink: RouterLinkStub, 'router-link': RouterLinkStub } },
  });
}

describe('SectionGutter', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('draws one dot per section, coloured by its grade', () => {
    const sections = [sec('a', 'LGTM'), sec('b', 'GRILLED')];
    const w = mountGutter(sections, [fakeAnchor(0, 0), fakeAnchor(200, 200)]);
    expect(w.find('[data-test="gutter-dot-a"]').exists()).toBe(true);
    expect(w.find('[data-test="gutter-dot-b"]').exists()).toBe(true);
    expect(w.find('[data-test="gutter-dot-b"]').attributes('style')).toContain('var(--l-0-fg)');
  });

  it('names each dot for a screen reader', () => {
    const w = mountGutter([sec('a', 'LGTM')], [fakeAnchor(0, 0)]);
    expect(w.find('[data-test="gutter-dot-a"]').attributes('aria-label')).toBe('Grade: Section a');
  });

  it('starts targeting the first section', () => {
    const w = mountGutter([sec('a', 'LGTM'), sec('b', 'GRILLED')], [fakeAnchor(0, 0), fakeAnchor(200, 200)]);
    // the pill is bound to the target: section a is LGTM, so L is active
    expect(w.find('[data-test="grade-pill-lgtm"]').classes()).toContain('is-active');
  });

  it('retargets when the pointer enters a section', async () => {
    const anchors = [fakeAnchor(0, 0), fakeAnchor(200, 200)];
    const w = mountGutter([sec('a', 'LGTM'), sec('b', 'GRILLED')], anchors);
    anchors[1].dispatchEvent(new MouseEvent('mouseenter'));
    await w.vm.$nextTick();
    expect(w.find('[data-test="grade-pill-grilled"]').classes()).toContain('is-active');
    // section 1 bottom = top(200) + height(100) = 300, minus the 6px grade-line inset.
    expect(w.find('[data-test="gutter-cell"]').attributes('style')).toContain('translateY(294px)');
  });

  // The ⋯ menu lives in a mirror rail on the far side of the reading column.
  // It has no targeting of its own — it rides this event so it tracks the same
  // section, docked at that section's TOP (while the pill sits at its bottom).
  it('emits the target card and its top y for the mirror rail', async () => {
    const anchors = [fakeAnchor(0, 0), fakeAnchor(200, 200)];
    const w = mountGutter([sec('a', 'LGTM'), sec('b', 'GRILLED')], anchors);
    anchors[1].dispatchEvent(new MouseEvent('mouseenter'));
    await w.vm.$nextTick();
    const events = w.emitted('targetChange');
    expect(events).toBeTruthy();
    const [card, y] = events![events!.length - 1] as [Card, number];
    expect(card.id).toBe('b');
    expect(y).toBe(200); // section 1 TOP (the pill uses the bottom; the menu the top)
  });

  it('retargets when a dot is focused — the keyboard path', async () => {
    const w = mountGutter([sec('a', 'LGTM'), sec('b', 'GRILLED')], [fakeAnchor(0, 0), fakeAnchor(200, 200)]);
    await w.find('[data-test="gutter-dot-b"]').trigger('focus');
    expect(w.find('[data-test="grade-pill-grilled"]').classes()).toContain('is-active');
  });

  it('grades the targeted section, not the first one', async () => {
    setActivePinia(createPinia());
    const store = useCardsStore();
    const spy = vi.spyOn(store, 'setReviewGrade').mockResolvedValue({} as Card);
    const w = mountGutter([sec('a', 'LGTM'), sec('b', 'LGTM')], [fakeAnchor(0, 0), fakeAnchor(200, 200)]);
    await w.find('[data-test="gutter-dot-b"]').trigger('focus');
    await w.find('[data-test="grade-pill-grilled"]').trigger('click');
    expect(spy).toHaveBeenCalledWith('b', 'GRILLED');
  });

  // THE test. Hover section 0, leave the pane, scroll to the bottom: the pill
  // must NOT still point at section 0. Grading is a write and it is one click;
  // a pill aimed off-screen writes to a card you cannot see.
  it('falls back to the scroll position, never pointing off-screen', async () => {
    const pane = document.createElement('div');
    Object.defineProperty(pane, 'clientHeight', { value: 300, configurable: true });
    pane.getBoundingClientRect = () => ({ top: 100, height: 300 }) as DOMRect;

    // section 0 has scrolled far above the pane; section 1 is what you're reading
    const anchors = [fakeAnchor(0, -400), fakeAnchor(500, 120)];
    const w = mountGutter([sec('a', 'LGTM'), sec('b', 'GRILLED')], anchors, pane);

    anchors[0].dispatchEvent(new MouseEvent('mouseenter'));
    await w.vm.$nextTick();
    expect(w.find('[data-test="grade-pill-lgtm"]').classes()).toContain('is-active');

    pane.dispatchEvent(new MouseEvent('mouseleave'));
    pane.dispatchEvent(new Event('scroll'));
    await w.vm.$nextTick();

    expect(w.find('[data-test="grade-pill-grilled"]').classes()).toContain('is-active');
  });

  it('renders nothing for an empty section list', () => {
    const w = mountGutter([], []);
    expect(w.find('[data-test="gutter-cell"]').exists()).toBe(false);
  });

  // Regression: the cell must be placed by getBoundingClientRect, not offsetTop.
  // In the real page the sections and the gutter have different offsetParents,
  // so their offsetTops are not comparable and the pill lands at the wrong y —
  // a bug the earlier tests missed because they stubbed both to equal values.
  // Here the two DISAGREE: offsetTop says 999, the rect says 200/300. The rect wins.
  it('positions the cell by client rect, not offsetTop', async () => {
    const el = document.createElement('section');
    Object.defineProperty(el, 'offsetTop', { value: 999, configurable: true });
    el.getBoundingClientRect = () => ({ top: 200, bottom: 300, height: 100 }) as DOMRect;

    const w = mountGutter([sec('a', 'LGTM')], [el]);
    await w.vm.$nextTick();
    // gutter root client top is 0 in jsdom; the cell docks at the section BOTTOM
    // (200 + 100 = 300) minus the 6px grade-line inset = 294.
    expect(w.find('[data-test="gutter-cell"]').attributes('style')).toContain('translateY(294px)');
  });
});
