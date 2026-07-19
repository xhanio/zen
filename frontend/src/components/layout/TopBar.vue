<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import { search } from '../../api/client';
import { BackendError, type SearchResponse } from '../../types/api';
import { usePresenceStore } from '../../stores/presence';

const presence = usePresenceStore();
const { sessions } = storeToRefs(presence);

// App version, injected at build time from project.yaml (see vite.config.ts).
const appVersion = __APP_VERSION__;

// The topbar renders on every view, so this is where the app-lifetime socket
// is opened. It is deliberately not tied to a conversation.
onMounted(() => presence.connect());

const presenceLabel = computed(() => {
  const n = sessions.value.length;
  if (n === 0) return 'No AI connected';
  return n === 1 ? '1 session' : `${n} sessions`;
});

const router = useRouter();
const query = ref('');
const open = ref(false);
const loading = ref(false);
const errorMsg = ref<string | null>(null);
const result = ref<SearchResponse | null>(null);
const highlight = ref(-1);
const container = ref<HTMLElement | null>(null);

let debounceTimer: ReturnType<typeof setTimeout> | null = null;

interface Hit {
  kind: 'card';
  id: string;
  title: string;
  snippet: string;
}

const hits = computed<Hit[]>(() => {
  if (!result.value) return [];
  return result.value.cards.slice(0, 10).map<Hit>((h) => ({
    kind: 'card', id: h.id, title: h.title, snippet: h.snippet,
  }));
});

watch(query, (q) => {
  if (debounceTimer) clearTimeout(debounceTimer);
  highlight.value = -1;
  const trimmed = q.trim();
  if (trimmed.length < 2) {
    result.value = null;
    open.value = false;
    return;
  }
  debounceTimer = setTimeout(async () => {
    loading.value = true;
    errorMsg.value = null;
    try {
      result.value = await search(trimmed, 'all', 10);
      open.value = true;
    } catch (e) {
      errorMsg.value = e instanceof BackendError ? e.message : String(e);
      open.value = true;
    } finally {
      loading.value = false;
    }
  }, 300);
});

function submit() {
  if (highlight.value >= 0 && hits.value[highlight.value]) {
    activate(hits.value[highlight.value]);
    return;
  }
  const q = query.value.trim();
  if (!q) return;
  open.value = false;
  router.push({ path: '/search', query: { q } });
}

function activate(hit: Hit) {
  open.value = false;
  query.value = '';
  result.value = null;
  router.push({ name: 'card', params: { cardId: hit.id } });
}

function onKeydown(e: KeyboardEvent) {
  if (!open.value || hits.value.length === 0) return;
  if (e.key === 'ArrowDown') {
    e.preventDefault();
    highlight.value = (highlight.value + 1) % hits.value.length;
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    highlight.value = highlight.value <= 0 ? hits.value.length - 1 : highlight.value - 1;
  } else if (e.key === 'Escape') {
    open.value = false;
    highlight.value = -1;
  }
}

function onDocumentClick(e: MouseEvent) {
  if (!container.value) return;
  if (!container.value.contains(e.target as Node)) {
    open.value = false;
  }
}

onMounted(() => document.addEventListener('click', onDocumentClick));
onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick);
  if (debounceTimer) clearTimeout(debounceTimer);
});
</script>

<template>
  <header class="flex h-12 items-center gap-4 border-b border-border bg-surface px-4">
    <div class="flex items-baseline gap-1.5">
      <RouterLink to="/" class="text-lg font-semibold text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-500">Zen</RouterLink>
      <span data-test="app-version" class="text-xs font-normal text-muted-fg">{{ appVersion }}</span>
    </div>
    <form ref="container" class="relative flex-1" @submit.prevent="submit">
      <input
        v-model="query"
        type="search"
        placeholder="Search cards and documents…"
        aria-label="Search"
        autocomplete="off"
        class="w-full rounded border border-border px-3 py-1.5 text-sm focus:border-border focus:outline-none focus-visible:ring-2 focus-visible:ring-gray-500"
        @focus="open = result !== null"
        @keydown="onKeydown"
      />
      <div
        v-if="open"
        class="absolute left-0 right-0 top-full z-30 mt-1 max-h-80 overflow-y-auto rounded border border-border bg-surface shadow"
      >
        <div v-if="loading" class="px-3 py-2 text-sm text-muted-fg">Searching…</div>
        <div v-else-if="errorMsg" class="px-3 py-2 text-sm text-destructive-fg">{{ errorMsg }}</div>
        <div v-else-if="hits.length === 0" class="px-3 py-2 text-sm text-muted-fg">No matches.</div>
        <ul v-else role="listbox" class="text-sm">
          <li
            v-for="(hit, idx) in hits"
            :key="`${hit.kind}:${hit.id}`"
            role="option"
            :aria-selected="idx === highlight"
            class="cursor-pointer border-b border-border px-3 py-2 last:border-0"
            :class="idx === highlight ? 'bg-muted' : 'hover:bg-muted'"
            @mouseenter="highlight = idx"
            @click="activate(hit)"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="font-medium text-fg">{{ hit.title }}</span>
              <span class="text-[10px] uppercase tracking-wide text-muted-fg">{{ hit.kind }}</span>
            </div>
            <p class="mt-0.5 line-clamp-1 text-xs text-muted-fg" v-html="hit.snippet"></p>
          </li>
        </ul>
      </div>
    </form>
    <div
      class="flex shrink-0 items-center gap-1.5 text-xs"
      :class="sessions.length > 0 ? 'text-fg' : 'text-muted-fg'"
      :title="sessions.length > 0
        ? 'Claude Code sessions connected to Zen'
        : 'Start a Claude Code session with the zen plugin to talk to Zen'"
    >
      <span aria-hidden="true">{{ sessions.length > 0 ? '●' : '○' }}</span>
      <span>{{ presenceLabel }}</span>
    </div>
  </header>
</template>
