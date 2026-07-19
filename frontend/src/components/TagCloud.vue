<script setup lang="ts">
import { watch } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useTagsStore } from '../stores/tags';
import { useTagFilterStore } from '../stores/tagFilter';

const route = useRoute();
const store = useTagsStore();
const filter = useTagFilterStore();
const { tags } = storeToRefs(store);

// Tags are group-scoped: reload the active group's tags whenever the
// /g/:groupId route param changes, and clear any stale filter.
watch(
  () => route.params.groupId as string | undefined,
  (gid) => {
    filter.clear();
    if (gid) store.load(gid);
    else store.tags = [];
  },
  { immediate: true },
);
</script>

<template>
  <div class="text-xs">
    <div v-if="store.loading" class="text-muted-fg">Loading…</div>
    <div v-else-if="tags.length === 0" class="text-muted-fg">No tags yet.</div>
    <div v-else class="flex flex-wrap gap-1">
      <button
        v-for="tag in tags"
        :key="tag.id"
        type="button"
        :aria-pressed="filter.isActive(tag.name)"
        :title="filter.isActive(tag.name) ? `Remove ${tag.name} from filter` : `Add ${tag.name} to filter`"
        class="inline-flex items-center gap-1 rounded px-1.5 py-0.5 hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-500"
        :class="[
          filter.isActive(tag.name)
            ? 'bg-fg text-surface'
            : tag.card_count > 0
              ? 'bg-muted text-fg'
              : 'bg-muted text-muted-fg',
        ]"
        @click="filter.toggle(tag.name)"
      >
        <span>{{ tag.name }}</span>
        <span
          class="rounded px-1 text-[10px] tabular-nums"
          :class="filter.isActive(tag.name) ? 'bg-surface/20 text-surface' : 'bg-surface/80 text-muted-fg'"
          :title="`${tag.card_count} card${tag.card_count === 1 ? '' : 's'}`"
        >
          {{ tag.card_count }}
        </span>
      </button>
    </div>
  </div>
</template>
