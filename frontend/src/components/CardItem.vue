<script setup lang="ts">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import type { Card, LevelEntry } from '../types/entity';
import { previewText } from '../utils/preview';
import { levelColor } from '../utils/levelPalette';
import { entryForCard } from '../utils/levelCatalog';
import { useTilePrefsStore } from '../stores/tilePrefs';
import { useCardsStore } from '../stores/cards';
import { useGroupsStore } from '../stores/groups';
import SectionOutline from './SectionOutline.vue';
import { isContainer } from '../utils/isContainer';

const props = defineProps<{
  card: Card;
  hidePill?: boolean;
  groupCatalog?: LevelEntry[];
  pinned?: boolean;
}>();
defineEmits<{ (e: 'delete', id: string): void }>();

const { hideSummaries } = storeToRefs(useTilePrefsStore());
const cardsStore = useCardsStore();
const { byGroup } = storeToRefs(cardsStore);
const groupsStore = useGroupsStore();
// Use the explicit prop if the caller passes one; otherwise fall back to
// the groups store lookup. Most call sites (LevelColumn, CardTile grids)
// don't pass the catalog, but the tile still needs it to derive the
// display weight for its palette and pill.
const catalog = computed(
  () => props.groupCatalog ?? groupsStore.get(props.card.group_id)?.level_catalog ?? [],
);

// A tile shows its section outline (mini-TOC) when the card is a
// container: it has at least one live child in the same group
// (content-independent — its content, if any, is the preamble). Peers
// are in byGroup already, so no separate fetch is needed.
const liveChildCount = computed(() => {
  const siblings = byGroup.value[props.card.group_id] ?? [];
  let n = 0;
  for (const c of siblings) {
    if (c.parent_card_id === props.card.id && !c.deleted_at) n++;
  }
  return n;
});
const isContainerTile = computed(
  () => isContainer(props.card, liveChildCount.value),
);

// Paper unit: every leaf tile is exactly one page tall. A container
// tile is N pages, where N is the smallest integer that holds the whole
// section outline — estimated from its title, tags, genesis, and
// outline rows (see below). GAP_PX matches the
// space-y-1 gap between tiles in LevelColumn so a 2-unit container
// ends at the same y as two stacked leaves — it absorbs the (N-1)
// inter-leaf gaps it "replaces".
const UNIT_HEIGHT_PX = 150;
const GAP_PX = 4;
// Rough per-element heights (px) to estimate a container tile's natural
// content height, so it spans exactly the paper units its outline needs.
// Overhead is conditional: no tags / no genesis → more outline rows per unit.
const PADDING_PX = 16;    // article py-2 (top+bottom)
const TITLE_PX = 24;      // title row + spacing
const HEADER_GAP_PX = 8;  // outline-area + list top margins
const OUTLINE_ROW_PX = 18;
const TAG_ROW_PX = 24;
const GENESIS_PX = 20;

const unitCount = computed(() => {
  if (!isContainerTile.value) return 1;
  const needed = PADDING_PX + TITLE_PX + HEADER_GAP_PX
    + liveChildCount.value * OUTLINE_ROW_PX
    + (props.card.tags.length > 0 ? TAG_ROW_PX : 0)
    + (props.card.genesis ? GENESIS_PX : 0);
  let n = 1;
  while (n * UNIT_HEIGHT_PX + (n - 1) * GAP_PX < needed) n++;
  return n;
});
// Compact mode: the "Hide summaries" toggle collapses every tile to a
// title + tags strip. Summary text and container outlines drop; tags
// stay because they're the load-bearing scan aid. One COMPACT_HEIGHT_PX
// applies to leaves and containers alike.
const COMPACT_HEIGHT_PX = 75;

const tileHeightStyle = computed(() => {
  if (hideSummaries.value) return { height: `${COMPACT_HEIGHT_PX}px` };
  const n = unitCount.value;
  const h = n * UNIT_HEIGHT_PX + (n - 1) * GAP_PX;
  return { height: `${h}px` };
});

// Human-authored summary wins over the auto content-derived preview.
// Empty summary → preview so tiles never look bare. When the viewer
// toggles summaries off the tile hides all description text — no
// preview fallback — for a compact scan. Container tiles skip preview
// text entirely; the section outline carries the meaning instead.
const tileText = computed(() => {
  if (isContainerTile.value) return '';
  if (hideSummaries.value) return '';
  const s = props.card.summary?.trim();
  if (s) return s;
  return previewText(props.card.content, props.card.format);
});

// A card's display weight/name are derived from its level_entry_id via
// the group's catalog. If the entry_id doesn't resolve (e.g. catalog
// hasn't loaded yet), fall back to the Unfiled palette.
const entry = computed(() => entryForCard(catalog.value, props.card) ?? null);
const color = computed(() => levelColor(catalog.value, entry.value?.weight ?? null));

const pillLabel = computed(() => {
  if (props.hidePill) return null;
  return entry.value?.name ?? null;
});
</script>

<template>
  <article
    :data-card-id="card.id"
    :draggable="!pinned"
    :style="[tileHeightStyle, { borderTopWidth: '2px', borderTopColor: color.border }]"
    :class="[
      'card-draggable group relative overflow-hidden rounded border border-border bg-surface px-2.5 py-2 shadow-sm transition hover:border-border hover:shadow',
      pinned ? 'cursor-pointer' : 'cursor-grab',
    ]"
    @dragstart="(e) => e.dataTransfer?.setData('text/zen-card-id', card.id)"
  >
    <span
      v-if="pillLabel"
      class="absolute right-6 top-0.5 rounded px-1 py-px text-[9px] leading-tight"
      :style="{ backgroundColor: color.bg, color: color.fg }"
    >{{ pillLabel }}</span>
    <button
      type="button"
      class="invisible absolute right-0.5 top-0.5 rounded px-1 text-xs text-destructive-fg hover:bg-destructive-bg group-hover:visible focus-visible:visible focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500"
      :aria-label="`Delete card ${card.title || 'untitled'}`"
      @click="$emit('delete', card.id)"
    >
      ×
    </button>
    <RouterLink
      :to="{ name: 'card', params: { cardId: card.id } }"
      class="flex h-full flex-col"
    >
      <div class="flex items-baseline gap-2 pr-5">
        <h3 class="min-w-0 flex-1 truncate font-serif text-base font-medium leading-tight text-fg">{{ card.title }}</h3>
      </div>
      <div v-if="!hideSummaries" class="mt-1 min-h-0 flex-1 overflow-hidden">
        <p
          v-if="tileText"
          class="text-xs leading-snug text-muted-fg"
        >{{ tileText }}</p>
        <SectionOutline v-if="isContainerTile" :parent="card" />
      </div>
      <div v-else class="flex-1"></div>
      <div v-if="card.tags.length > 0" class="mt-1 flex shrink-0 flex-wrap gap-1">
        <span
          v-for="tag in card.tags"
          :key="tag"
          class="rounded bg-muted px-1 py-px text-[10px] text-fg"
        >
          {{ tag }}
        </span>
      </div>
      <div
        v-if="card.genesis"
        :title="card.genesis"
        class="mt-1 w-full truncate text-left text-[10px] leading-tight text-muted-fg"
      >{{ card.genesis }}</div>
    </RouterLink>
  </article>
</template>
