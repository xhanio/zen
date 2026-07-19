import { describe, it, expect } from 'vitest';
import { mount, RouterLinkStub } from '@vue/test-utils';
import SectionActionsMenu from './SectionActionsMenu.vue';
import type { Card } from '../types/entity';

const card = {
  id: 'c1', title: 'A section', content: 'x', summary: '', format: 'markdown' as const,
  level_entry_id: null, group_id: 'g1', position: 0, tags: [], genesis: '',
  deleted_at: null, parent_card_id: null, source_conversation_id: null,
  created_at: 'x', updated_at: 'x',
  review_grade: 'LGTM' as const, review_score: null, reviewed_at: null,
} satisfies Card;

const mountMenu = () =>
  mount(SectionActionsMenu, {
    props: { card },
    // Register both casings: the template writes <RouterLink>, and Vue also
    // resolves the kebab alias — stub whichever the compiler emits.
    global: { stubs: { RouterLink: RouterLinkStub, 'router-link': RouterLinkStub } },
  });

// The chip it replaces was a bare link. The menu exists so the NEXT action has
// somewhere to go without moving the control again.
describe('SectionActionsMenu', () => {
  it('is closed until the trigger is clicked', async () => {
    const w = mountMenu();
    expect(w.find('[data-test="section-menu"]').exists()).toBe(false);
    expect(w.find('[data-test="section-menu-trigger"]').attributes('aria-expanded')).toBe('false');

    await w.find('[data-test="section-menu-trigger"]').trigger('click');
    expect(w.find('[data-test="section-menu"]').exists()).toBe(true);
    expect(w.find('[data-test="section-menu-trigger"]').attributes('aria-expanded')).toBe('true');
  });

  it('offers Open, routed at the section card', async () => {
    const w = mountMenu();
    await w.find('[data-test="section-menu-trigger"]').trigger('click');
    const open = w.findComponent(RouterLinkStub);
    expect(open.props('to')).toEqual({ name: 'card', params: { cardId: 'c1' } });
    expect(w.find('[data-test="section-menu-open"]').text()).toContain('Open');
  });

  it('closes on Escape', async () => {
    const w = mountMenu();
    await w.find('[data-test="section-menu-trigger"]').trigger('click');
    await w.find('[data-test="section-menu-trigger"]').trigger('keydown', { key: 'Escape' });
    expect(w.find('[data-test="section-menu"]').exists()).toBe(false);
  });
});
