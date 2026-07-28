import { defineStore } from 'pinia';
import { ref } from 'vue';
import { listCardSelections } from '../api/client';
import type { CardSelection, Message } from '../types/entity';

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

  // Paint a just-sent selection without a round trip. The SPA is the origin of
  // these offsets — it measured them at drag time and just posted them — so
  // re-asking the server for data it was handed a moment ago would only delay
  // the mark. Ignores a message with no range: those never paint.
  function add(cardID: string, m: Message): void {
    if (m.selection_start == null || m.selection_end == null || !m.selection_text) return;
    const list = byCard.value[cardID] ?? [];
    if (list.some((s) => s.message_id === m.id)) return;
    byCard.value[cardID] = [
      ...list,
      {
        message_id: m.id,
        conversation_id: m.conversation_id,
        selection_text: m.selection_text,
        selection_start: m.selection_start,
        selection_end: m.selection_end,
        selection_seq: m.selection_seq ?? null,
        created_at: m.created_at,
      },
    ];
  }

  return { byCard, stale, load, markStale, add };
});
