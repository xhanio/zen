import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import CardExportButton from './CardExportButton.vue';
import * as cardExport from '../composables/useCardExport';
import type { Card } from '../types/entity';

const card = {
  id: 'c1', title: 'Hello', summary: '', content: 'body', format: 'markdown' as const,
  level_entry_id: null, genesis: '', deleted_at: null, group_id: 'g1', position: 0,
  tags: [], parent_card_id: null, source_conversation_id: null, created_at: 'x',
  updated_at: 'x', review_grade: 'LGTM' as const, review_score: null, reviewed_at: null,
} satisfies Card;

beforeEach(() => {
  vi.restoreAllMocks();
  const { exporting, error } = cardExport.useCardExport();
  exporting.value = false;
  error.value = null;
});

describe('CardExportButton', () => {
  it('renders "Export" and calls exportCard(card.id) on click', async () => {
    const spy = vi.spyOn(cardExport.actions, 'exportCard').mockResolvedValue();
    const w = mount(CardExportButton, { props: { card } });
    const btn = w.find('[data-test="card-action-export"]');
    expect(btn.text()).toBe('Export');
    await btn.trigger('click');
    expect(spy).toHaveBeenCalledWith('c1');
  });

  it('shows "Exporting…" and disables the button while exporting', async () => {
    const { exporting } = cardExport.useCardExport();
    const w = mount(CardExportButton, { props: { card } });
    exporting.value = true;
    await nextTick();
    const btn = w.find('[data-test="card-action-export"]');
    expect(btn.text()).toBe('Exporting…');
    expect(btn.attributes('disabled')).toBeDefined();
  });

  it('renders the error message when export fails', async () => {
    const { error } = cardExport.useCardExport();
    const w = mount(CardExportButton, { props: { card } });
    error.value = 'nope';
    await nextTick();
    expect(w.find('[data-test="card-export-error"]').text()).toBe('nope');
  });

  it('shows no error node when there is no error', () => {
    const w = mount(CardExportButton, { props: { card } });
    expect(w.find('[data-test="card-export-error"]').exists()).toBe(false);
  });
});
