import { ref } from 'vue';
import { defineStore } from 'pinia';
import { listTags, renameTag, deleteTag } from '../api/client';
import { BackendError } from '../types/api';
import type { Tag } from '../types/entity';

export const useTagsStore = defineStore('tags', () => {
  const tags = ref<Tag[]>([]);
  const loading = ref(false);
  const error = ref<BackendError | null>(null);
  // The group whose tags are currently loaded — lets refresh() reload the
  // active group's tags after a card mutation without every caller having to
  // know which group is on screen.
  const currentGroupId = ref<string | null>(null);
  // Monotonic sequence so a later load()'s result always wins if two
  // requests race — switching groups can overlap an in-flight load.
  let seq = 0;

  // Tags are group-scoped: load a single group's tags.
  async function load(groupId: string) {
    currentGroupId.value = groupId;
    const mine = ++seq;
    loading.value = true;
    error.value = null;
    try {
      const next = await listTags(groupId);
      if (mine === seq) tags.value = next;
    } catch (e) {
      if (mine === seq) {
        error.value = e instanceof BackendError ? e : new BackendError(0, '', String(e));
      }
    } finally {
      if (mine === seq) loading.value = false;
    }
  }

  // Fire-and-forget refresh of the active group's tags (e.g. after a card
  // mutation changed card_count). No-op when no group is loaded.
  async function refresh() {
    if (currentGroupId.value) await load(currentGroupId.value);
  }

  async function rename(groupId: string, oldName: string, newName: string): Promise<Tag> {
    const t = await renameTag(groupId, oldName, { new_name: newName });
    const idx = tags.value.findIndex((x) => x.name === oldName);
    if (idx >= 0) tags.value[idx] = t;
    return t;
  }

  async function remove(groupId: string, name: string): Promise<void> {
    await deleteTag(groupId, name);
    tags.value = tags.value.filter((t) => t.name !== name);
  }

  return { tags, loading, error, currentGroupId, load, refresh, rename, remove };
});
