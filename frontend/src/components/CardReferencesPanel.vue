<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useCardsStore } from '../stores/cards';
import { useGroupsStore } from '../stores/groups';
import { previewText } from '../utils/preview';
import { colorForCard, type LevelColor } from '../utils/levelPalette';
import { useTilePrefsStore } from '../stores/tilePrefs';
import type { Card } from '../types/entity';

// Right-column panel on CardView. Shows the outbound references of the
// current focus (initially the anchor card), with click-to-drill into
// each referenced card. Maintains its own navigation stack so the panel
// can browse a small neighborhood of linked cards without changing the
// SPA route.

const props = defineProps<{ rootCard: Card }>();

const cardsStore = useCardsStore();
const { byID } = storeToRefs(cardsStore);
const groupsStore = useGroupsStore();

function colorForId(id: string): LevelColor {
  const c = byID.value[id];
  if (!c) return colorForCard([], { level_entry_id: null });
  return colorForCard(groupsStore.get(c.group_id)?.level_catalog ?? [], c);
}

// Stack of card IDs currently focused inside the panel. Empty stack →
// panel shows the rootCard's references directly (rootCard itself is
// the "focus"). Push adds the drilled-in card; pop returns.
const stack = ref<string[]>([]);

const focusId = computed(() => stack.value[stack.value.length - 1] ?? props.rootCard.id);
const focus = computed<Card | undefined>(() => byID.value[focusId.value] ?? undefined);

const loading = ref(false);
const loadError = ref<string | null>(null);

async function ensureLoaded(id: string, opts: { silent?: boolean } = {}) {
  if (byID.value[id]) return;
  if (!opts.silent) {
    loading.value = true;
    loadError.value = null;
  }
  try {
    await cardsStore.loadOne(id);
  } catch (e) {
    if (!opts.silent) loadError.value = e instanceof Error ? e.message : String(e);
  } finally {
    if (!opts.silent) loading.value = false;
  }
}

watch(
  focusId,
  (id) => {
    if (id !== props.rootCard.id) void ensureLoaded(id);
  },
  { immediate: true },
);

// Reset the stack when the anchor card changes (SPA navigation to a
// different card).
watch(
  () => props.rootCard.id,
  () => {
    stack.value = [];
  },
);

// Outbound refs of the focused card. rootCard is passed in with its
// references already loaded; drilled-in cards get their references
// hydrated by ensureLoaded → cardsStore.loadOne.
const refs = computed(() => focus.value?.references ?? []);

// Pre-hydrate each referenced card so rows can show real titles without
// waiting for a click. Cheap on cache hits.
watch(
  refs,
  (list) => {
    for (const r of list) void ensureLoaded(r.derived_card_id, { silent: true });
  },
  { immediate: true },
);

// One row per reference — no dedup. If two highlights on the parent
// point at the same target card, they appear as two tiles.
interface RefRow {
  refId: string;
  targetId: string;
  selectionText: string;
}
const rows = computed<RefRow[]>(() =>
  refs.value.map((r) => ({
    refId: r.id,
    targetId: r.derived_card_id,
    selectionText: r.selection_text,
  })),
);

const tilePrefs = useTilePrefsStore();
const { hideSummaries } = storeToRefs(tilePrefs);

// Compute a CardItem-style preview for the target card. Same logic as
// CardItem.tileText so front-page and panel tiles read identically.
function targetTileText(targetId: string): string {
  if (hideSummaries.value) return '';
  const c = byID.value[targetId];
  if (!c) return '';
  const s = c.summary?.trim();
  if (s) return s;
  return previewText(c.content, c.format);
}

async function openRow(targetId: string) {
  await ensureLoaded(targetId);
  stack.value = [...stack.value, targetId];
}

function goBack() {
  if (stack.value.length > 0) {
    stack.value = stack.value.slice(0, -1);
  }
}

function goRoot() {
  stack.value = [];
}

const focusPreview = computed(() => {
  const f = focus.value;
  if (!f) return '';
  const s = f.summary?.trim();
  if (s) return s;
  return previewText(f.content, f.format);
});

// Breadcrumb — root card's title, then every drilled title.
const breadcrumbs = computed(() => {
  const items: Array<{ id: string; title: string; onClick: (() => void) | null }> = [
    { id: props.rootCard.id, title: props.rootCard.title, onClick: stack.value.length ? goRoot : null },
  ];
  stack.value.forEach((id, idx) => {
    const c = byID.value[id];
    const isTop = idx === stack.value.length - 1;
    items.push({
      id,
      title: c?.title ?? '…',
      onClick: isTop ? null : () => (stack.value = stack.value.slice(0, idx + 1)),
    });
  });
  return items;
});
</script>

<template>
  <aside
    data-test="references-panel"
    class="flex h-full w-[320px] shrink-0 flex-col border-l border-border bg-surface"
  >
    <header class="flex items-center gap-1 border-b border-border px-3 py-2 text-xs">
      <button
        v-if="stack.length > 0"
        type="button"
        class="rounded px-1.5 py-0.5 text-muted-fg hover:bg-muted"
        title="Back"
        @click="goBack"
      >‹</button>
      <span class="text-[10px] font-medium uppercase tracking-wide text-muted-fg">References</span>
      <span class="ml-auto rounded bg-muted px-1.5 py-px text-[10px] tabular-nums text-muted-fg">
        {{ rows.length }}
      </span>
    </header>

    <nav
      v-if="breadcrumbs.length > 1"
      class="flex flex-wrap items-center gap-1 border-b border-border px-3 py-1.5 text-[11px] text-muted-fg"
    >
      <template v-for="(b, i) in breadcrumbs" :key="b.id + i">
        <span v-if="i > 0" aria-hidden="true">/</span>
        <button
          v-if="b.onClick"
          type="button"
          class="truncate rounded px-1 hover:bg-muted"
          :title="b.title"
          @click="b.onClick"
        >{{ b.title }}</button>
        <span v-else class="truncate font-medium text-fg" :title="b.title">{{ b.title }}</span>
      </template>
    </nav>

    <section v-if="focus && stack.length > 0" class="border-b border-border px-3 py-2">
      <div class="flex items-start justify-between gap-2">
        <h4 class="font-serif text-sm font-medium leading-tight text-fg">{{ focus.title }}</h4>
        <RouterLink
          :to="{ name: 'card', params: { cardId: focus.id } }"
          class="shrink-0 rounded border border-accent-border px-1.5 py-px text-[10px] text-accent-fg hover:bg-accent-bg"
          :title="'Open ' + focus.title"
        >Open</RouterLink>
      </div>
      <p v-if="focusPreview" class="mt-1 line-clamp-3 text-[11px] leading-snug text-muted-fg">
        {{ focusPreview }}
      </p>
    </section>

    <div v-if="loadError" class="px-3 py-2 text-xs text-destructive-fg">{{ loadError }}</div>
    <div v-else-if="loading" class="px-3 py-2 text-xs text-muted-fg">Loading…</div>
    <ul v-else-if="rows.length === 0" class="px-3 py-2 text-xs text-muted-fg">
      <li>No references from this card.</li>
    </ul>
    <ul v-else class="flex-1 space-y-1.5 overflow-y-auto p-2">
      <li v-for="row in rows" :key="row.refId">
        <button
          type="button"
          data-test="ref-row"
          class="block w-full rounded border border-border bg-surface px-2.5 py-2 text-left shadow-sm transition hover:border-border hover:shadow"
          :style="{ borderTopWidth: '2px', borderTopColor: colorForId(row.targetId).border }"
          @click="openRow(row.targetId)"
        >
          <h3 class="truncate font-serif text-base font-medium leading-tight text-fg">
            {{ byID[row.targetId]?.title ?? '(loading…)' }}
          </h3>
          <p
            v-if="targetTileText(row.targetId)"
            class="mt-1 line-clamp-2 text-xs leading-snug text-muted-fg"
          >{{ targetTileText(row.targetId) }}</p>
          <div
            v-if="(byID[row.targetId]?.tags?.length ?? 0) > 0"
            class="mt-1 flex flex-wrap gap-1"
          >
            <span
              v-for="tag in byID[row.targetId]!.tags"
              :key="tag"
              class="rounded bg-muted px-1 py-px text-[10px] text-fg"
            >{{ tag }}</span>
          </div>
        </button>
      </li>
    </ul>
  </aside>
</template>
