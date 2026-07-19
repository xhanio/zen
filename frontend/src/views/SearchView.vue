<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { search } from '../api/client';
import { BackendError, type SearchResponse } from '../types/api';
import type { SearchHit } from '../types/entity';
import { truncateTail } from '../utils/titlePath';

// The muted breadcrumb shown ahead of a hit's title: the nearest 2 ancestors,
// prefixed with "…" when the card is deeper. Empty for top-level cards.
function pathPrefix(hit: SearchHit): string {
  if (!hit.title_path?.length) return '';
  const { items, overflow } = truncateTail(hit.title_path, 2);
  return (overflow ? '… › ' : '') + items.join(' › ') + ' › ';
}

const route = useRoute();
const query = computed(() => (route.query.q as string | undefined) ?? '');

const result = ref<SearchResponse | null>(null);
const loading = ref(false);
const error = ref<BackendError | null>(null);

watch(
  query,
  async (q) => {
    result.value = null;
    if (!q) return;
    loading.value = true;
    error.value = null;
    try {
      result.value = await search(q, 'all', 50);
    } catch (e) {
      error.value = e instanceof BackendError ? e : new BackendError(0, '', String(e));
    } finally {
      loading.value = false;
    }
  },
  { immediate: true },
);
</script>

<template>
  <div>
    <h1 class="mb-3 text-xl font-semibold text-fg">
      Search<span v-if="query">: {{ query }}</span>
    </h1>
    <p v-if="!query" class="text-sm text-muted-fg">Use the search box above.</p>
    <p v-else-if="loading" class="text-sm text-muted-fg">Searching…</p>
    <p v-else-if="error" class="text-sm text-destructive-fg">{{ error.message }}</p>
    <template v-else-if="result">
      <section v-if="result.cards.length > 0" class="mb-4">
        <h2 class="mb-2 text-sm font-medium uppercase tracking-wide text-muted-fg">Cards</h2>
        <ul class="space-y-2 text-sm">
          <li v-for="hit in result.cards" :key="hit.id" class="rounded border border-border bg-surface p-3">
            <RouterLink :to="{ name: 'card', params: { cardId: hit.id } }" class="font-medium text-accent-fg hover:underline">
              <span v-if="pathPrefix(hit)" class="font-normal text-muted-fg">{{ pathPrefix(hit) }}</span>{{ hit.title }}
            </RouterLink>
            <p class="mt-1 text-xs text-muted-fg" v-html="hit.snippet"></p>
          </li>
        </ul>
      </section>
      <p v-if="result.cards.length === 0" class="text-sm text-muted-fg">
        No matches.
      </p>
    </template>
  </div>
</template>
