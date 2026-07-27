import { defineStore } from 'pinia';
import { ref } from 'vue';
import { listCardSelections } from '../api/client';
import type { CardSelection } from '../types/entity';

export const useSelectionsStore = defineStore('selections', () => {
  const byCard = ref<Record<string, CardSelection[]>>({});
  // Message ids whose stored offsets no longer match the rendered card. Only
  // ever written by a card body that actually rendered — absence means "not
  // determined", not "still matches". The server cannot decide this: it has no
  // notion of rendered text, which is why offsets are the SPA's job.
  const stale = ref<Record<string, boolean>>({});

  async function load(cardID: string): Promise<void> {
    try {
      const resp = await listCardSelections(cardID);
      byCard.value[cardID] = resp?.selections ?? [];
    } catch {
      // Underlines are a secondary surface; a failure here must not take the
      // card down with it.
      byCard.value[cardID] = [];
    }
  }

  function markStale(ids: string[]): void {
    for (const id of ids) stale.value[id] = true;
  }

  return { byCard, stale, load, markStale };
});
