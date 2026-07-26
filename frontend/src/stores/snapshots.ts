import { defineStore } from 'pinia';
import { ref } from 'vue';
import { listSnapshots, getSnapshot } from '../api/client';
import type { CardDiff, CardSnapshot } from '../types/entity';

const EMPTY_DIFF: CardDiff = { fields: [], lines: [] };

// A truncated or malformed payload degrades to an empty diff rather than
// throwing: the card view can still show both bodies.
function parseDiff(raw: string): CardDiff {
  if (!raw) return EMPTY_DIFF;
  try {
    const parsed = JSON.parse(raw) as CardDiff;
    return { fields: parsed.fields ?? [], lines: parsed.lines ?? [] };
  } catch {
    return EMPTY_DIFF;
  }
}

export interface SnapshotDetailView {
  snapshot: CardSnapshot;
  previous: CardSnapshot | null;
  parsed: CardDiff;
}

export const useSnapshotsStore = defineStore('snapshots', () => {
  const byConversation = ref<Record<string, CardSnapshot[]>>({});
  const byCard = ref<Record<string, CardSnapshot[]>>({});
  const detail = ref<SnapshotDetailView | null>(null);
  const loading = ref(false);

  // `?? []` guards an unexpected body shape: records are a secondary surface,
  // and a malformed response must not throw through the caller and take the
  // transcript render down with it.
  async function loadForConversation(conversationID: string) {
    const resp = await listSnapshots({ conversationID });
    byConversation.value = { ...byConversation.value, [conversationID]: resp?.snapshots ?? [] };
  }

  async function loadForCard(cardID: string) {
    const resp = await listSnapshots({ cardID });
    byCard.value = { ...byCard.value, [cardID]: resp?.snapshots ?? [] };
  }

  async function loadDetail(id: string) {
    loading.value = true;
    try {
      const resp = await getSnapshot(id);
      detail.value = {
        snapshot: resp.snapshot,
        previous: resp.previous,
        parsed: parseDiff(resp.snapshot.diff),
      };
    } finally {
      loading.value = false;
    }
  }

  function clearDetail() {
    detail.value = null;
  }

  return {
    byConversation,
    byCard,
    detail,
    loading,
    loadForConversation,
    loadForCard,
    loadDetail,
    clearDetail,
  };
});
