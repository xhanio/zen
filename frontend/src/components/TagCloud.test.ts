import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';

vi.mock('vue-router', () => ({ useRoute: () => ({ params: { groupId: 'G1' } }) }));

import TagCloud from './TagCloud.vue';
import { useTagsStore } from '../stores/tags';

describe('TagCloud', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it("loads and renders the current group's tags", () => {
    const store = useTagsStore();
    const loadSpy = vi.spyOn(store, 'load').mockResolvedValue();
    store.tags = [{ id: 't1', group_id: 'G1', name: 'spec', card_count: 3 }];
    const w = mount(TagCloud);
    expect(loadSpy).toHaveBeenCalledWith('G1');
    expect(w.text()).toContain('spec');
    expect(w.text()).toContain('3');
  });
});
