import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { Card } from '../types/entity';
import { listTrash, restoreCard, purgeCard, emptyTrash } from '../api/client';
import { useTagsStore } from './tags';

export const useTrashStore = defineStore('trash', () => {
  const cards = ref<Card[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function load() {
    loading.value = true;
    error.value = null;
    try {
      const resp = await listTrash();
      cards.value = resp.cards;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function restore(id: string) {
    await restoreCard(id);
    cards.value = cards.value.filter((c) => c.id !== id);
    void useTagsStore().refresh();
  }

  async function purge(id: string) {
    await purgeCard(id);
    cards.value = cards.value.filter((c) => c.id !== id);
    void useTagsStore().refresh();
  }

  async function empty() {
    const resp = await emptyTrash();
    cards.value = [];
    void useTagsStore().refresh();
    return resp.purged;
  }

  return { cards, loading, error, load, restore, purge, empty };
});
