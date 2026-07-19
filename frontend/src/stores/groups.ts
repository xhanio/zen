import { ref } from 'vue';
import { defineStore } from 'pinia';
import {
  listGroups,
  createGroup,
  updateGroup,
  deleteGroup,
} from '../api/client';
import { BackendError } from '../types/api';
import type { Group, LevelEntry } from '../types/entity';

export const useGroupsStore = defineStore('groups', () => {
  const groups = ref<Group[]>([]);
  const loading = ref(false);
  const error = ref<BackendError | null>(null);

  // Sequence guard so stale list loads don't clobber fresh local writes.
  let seq = 0;

  async function load() {
    if (loading.value) return;
    const local = ++seq;
    loading.value = true;
    error.value = null;
    try {
      const fetched = await listGroups();
      if (local !== seq) return;
      groups.value = fetched;
    } catch (e) {
      error.value = e instanceof BackendError ? e : new BackendError(0, '', String(e));
    } finally {
      loading.value = false;
    }
  }

  function get(id: string): Group | undefined {
    return groups.value.find((g) => g.id === id);
  }

  async function create(name: string): Promise<Group> {
    const g = await createGroup({ name });
    groups.value.push(g);
    seq++;
    return g;
  }

  async function remove(id: string): Promise<void> {
    await deleteGroup(id);
    groups.value = groups.value.filter((g) => g.id !== id);
  }

  async function update(
    id: string,
    patch: { name?: string; level_catalog?: LevelEntry[]; rule?: string },
  ): Promise<Group> {
    const g = await updateGroup(id, patch);
    const idx = groups.value.findIndex((x) => x.id === id);
    if (idx >= 0) groups.value[idx] = g;
    return g;
  }

  return { groups, loading, error, load, get, create, remove, update };
});
