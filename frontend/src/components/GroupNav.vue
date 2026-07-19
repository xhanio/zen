<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useGroupsStore } from '../stores/groups';
import { useLevelFilterStore } from '../stores/levelFilter';
import { useCardsStore } from '../stores/cards';
import { sortCatalog, weightForCard } from '../utils/levelCatalog';
import { documentsIn } from '../utils/documents';
import { levelColor, levelSpineGradient } from '../utils/levelPalette';
import GroupEditDialog from './group/GroupEditDialog.vue';
import { BackendError } from '../types/api';
import type { Card, Group, LevelEntry } from '../types/entity';

const groupsStore = useGroupsStore();
const levelFilter = useLevelFilterStore();
const cardsStore = useCardsStore();
const { groups } = storeToRefs(groupsStore);
const { byGroup } = storeToRefs(cardsStore);
const route = useRoute();

// A group is "open" (shows its level filters) exactly when you're viewing its
// board — the level filter only affects that group, so expansion follows the
// route rather than a separate toggle.
function isOpen(gid: string): boolean {
  return route.params.groupId === gid;
}

const editingGroupId = ref<string | null>(null);
const editingGroup = computed(() => groups.value.find((g) => g.id === editingGroupId.value));

const creatingRoot = ref(false);
const newName = ref('');
const createError = ref<string | null>(null);

// Load every group's cards so the rail can show counts. Guarded so remounting
// the rail (e.g. leaving reading mode) doesn't refetch what's already cached.
function loadCards() {
  for (const g of groups.value) if (!byGroup.value[g.id]) void cardsStore.loadByGroup(g.id);
}
onMounted(loadCards);
watch(groups, loadCards);

function liveCards(gid: string): Card[] {
  return (byGroup.value[gid] ?? []).filter((c) => !c.deleted_at);
}
// null until the group's cards have loaded (render a dash meanwhile).
function cardCount(gid: string): number | null {
  return byGroup.value[gid] ? liveCards(gid).length : null;
}
function docCount(gid: string): number {
  return documentsIn(byGroup.value[gid] ?? []).length;
}
function levelCount(g: Group, weight: number): number {
  return liveCards(g.id).filter((c) => weightForCard(g.level_catalog ?? [], c) === weight).length;
}
function levelColorFor(catalog: LevelEntry[], weight: number): string {
  return levelColor(catalog, weight).fg;
}

function ensureFilter(gid: string, catalog: LevelEntry[]) {
  return levelFilter.ensure(gid, sortCatalog(catalog));
}
function isSelected(gid: string, entryId: string, catalog: LevelEntry[]): boolean {
  const state = levelFilter.byGroup[gid] ?? ensureFilter(gid, catalog);
  return state.selectedEntryIds.includes(entryId);
}
function showMisc(gid: string, catalog: LevelEntry[]): boolean {
  return levelFilter.byGroup[gid]?.showMisc ?? ensureFilter(gid, catalog).showMisc;
}
function toggleLevelEntry(g: Group, entry: LevelEntry) {
  const state = ensureFilter(g.id, g.level_catalog);
  const next = state.selectedEntryIds.includes(entry.id)
    ? state.selectedEntryIds.filter((id) => id !== entry.id)
    : [...state.selectedEntryIds, entry.id];
  levelFilter.setSelectedEntryIds(g.id, next);
}

async function submitCreate() {
  const name = newName.value.trim();
  if (!name) return;
  createError.value = null;
  try {
    await groupsStore.create(name);
    creatingRoot.value = false;
    newName.value = '';
  } catch (e) {
    if (e instanceof BackendError) createError.value = e.message;
  }
}
</script>

<template>
  <div>
    <ul class="text-base">
      <li v-for="g in groups" :key="g.id" data-test="group-row" class="my-0.5">
        <div
          class="group/row relative flex items-center gap-1 rounded px-1"
          :class="isOpen(g.id) ? 'ring-1 ring-inset ring-border' : 'hover:bg-muted'"
          :style="isOpen(g.id) ? { background: 'color-mix(in oklch, var(--paper) 55%, transparent)' } : {}"
        >
          <span
            v-if="!isOpen(g.id)"
            aria-hidden="true"
            class="pointer-events-none absolute bottom-1.5 left-0 top-1.5 w-0.5 rounded bg-fg opacity-0 transition-opacity group-hover/row:opacity-100"
          ></span>
          <RouterLink
            :data-test="`group-link-${g.id}`"
            :to="{ name: 'group', params: { groupId: g.id } }"
            class="block flex-1 rounded px-2 py-1 text-xs font-semibold uppercase tracking-[0.09em]"
            :class="isOpen(g.id) ? 'text-fg' : 'text-muted-fg'"
          >{{ g.name }}</RouterLink>
          <span class="shrink-0 pr-1 text-[11px] tabular-nums text-muted-fg group-hover/row:hidden">
            <template v-if="cardCount(g.id) !== null">{{ cardCount(g.id) }}<template v-if="docCount(g.id) > 0"> · {{ docCount(g.id) }}▤</template></template>
            <template v-else>—</template>
          </span>
          <div class="hidden shrink-0 items-center group-hover/row:flex">
            <button type="button" class="rounded px-1 text-xs text-muted-fg hover:bg-muted" :aria-label="`Edit group ${g.name}`" @click="editingGroupId = g.id">✎</button>
          </div>
        </div>

        <div v-if="isOpen(g.id)" class="relative ml-3 mb-1 mt-0.5 pl-3.5">
          <span
            v-if="levelSpineGradient(g.level_catalog ?? [])"
            data-test="level-spine"
            class="absolute bottom-8 left-1 top-1.5 w-0.5 rounded"
            :style="{ background: levelSpineGradient(g.level_catalog ?? []) }"
          ></span>
          <button
            v-for="entry in sortCatalog(g.level_catalog ?? [])"
            :key="entry.id"
            type="button"
            :data-test="`level-${g.id}-${entry.id}`"
            class="flex w-full items-center gap-2.5 rounded px-2 py-1 text-left text-sm hover:bg-muted"
            @click="toggleLevelEntry(g, entry)"
          >
            <span
              class="h-2.5 w-2.5 shrink-0 rounded-full"
              :style="isSelected(g.id, entry.id, g.level_catalog)
                ? { background: levelColorFor(g.level_catalog, entry.weight) }
                : { boxShadow: 'inset 0 0 0 1.5px var(--muted-fg)' }"
            ></span>
            <span class="flex-1" :class="isSelected(g.id, entry.id, g.level_catalog) ? 'text-fg' : 'text-muted-fg'">{{ entry.name }}</span>
            <span class="shrink-0 text-[11px] tabular-nums text-muted-fg">{{ levelCount(g, entry.weight) }}</span>
          </button>
          <button
            type="button"
            :data-test="`level-${g.id}-misc`"
            class="flex w-full items-center gap-2.5 rounded px-2 py-1 text-left text-sm hover:bg-muted"
            @click="levelFilter.setShowMisc(g.id, !showMisc(g.id, g.level_catalog))"
          >
            <span
              class="h-2.5 w-2.5 shrink-0 rounded-full"
              :style="showMisc(g.id, g.level_catalog)
                ? { background: 'var(--l-tbd-fg)' }
                : { boxShadow: 'inset 0 0 0 1.5px var(--muted-fg)' }"
            ></span>
            <span class="flex-1" :class="showMisc(g.id, g.level_catalog) ? 'text-fg' : 'text-muted-fg'">Unfiled</span>
          </button>
        </div>
      </li>
    </ul>

    <div class="mt-1 px-1">
      <button
        v-if="!creatingRoot"
        type="button"
        class="w-full rounded px-2 py-1.5 text-left text-xs text-muted-fg hover:bg-muted hover:text-fg"
        @click="creatingRoot = true; newName = ''; createError = null"
      >+ New group</button>
      <div v-else class="flex gap-1">
        <input
          v-model="newName"
          type="text"
          placeholder="Group name"
          autofocus
          class="flex-1 rounded border border-border px-2 py-1 text-xs"
          @keydown.enter="submitCreate"
          @keydown.escape="creatingRoot = false"
        />
        <button type="button" class="rounded bg-fg px-2 py-1 text-xs text-surface hover:bg-fg" @click="submitCreate">Add</button>
      </div>
      <p v-if="createError" class="mt-1 text-xs text-destructive-fg">{{ createError }}</p>
    </div>

    <div
      v-if="editingGroup"
      data-test="edit-group-overlay"
      class="fixed inset-0 z-40 flex items-start justify-center bg-black/30 pt-20"
      @click.self="editingGroupId = null"
    >
      <GroupEditDialog :group="editingGroup" @close="editingGroupId = null" />
    </div>
  </div>
</template>
