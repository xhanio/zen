<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useGroupsStore } from '../stores/groups';
import { useCardsStore } from '../stores/cards';
import { useLevelFilterStore } from '../stores/levelFilter';
import { useTilePrefsStore } from '../stores/tilePrefs';
import { useTagFilterStore } from '../stores/tagFilter';
import ConfirmDialog from '../components/ConfirmDialog.vue';
import LevelColumn from '../components/group/LevelColumn.vue';
import CardItem from '../components/CardItem.vue';
import ResizableSplitter from '../components/ResizableSplitter.vue';
import { useChatSidebar } from '../composables/useChatSidebar';
import { BackendError } from '../types/api';
import { sortCatalog } from '../utils/levelCatalog';
import { documentsIn } from '../utils/documents';

const props = defineProps<{ groupId: string }>();

const sidebar = useChatSidebar();
function discussGroup() { sidebar.openFor('group', props.groupId, null); }

const groupsStore = useGroupsStore();
const cardsStore = useCardsStore();
const levelFilter = useLevelFilterStore();
const tilePrefs = useTilePrefsStore();
const { hideSummaries, hideSections, documentsWidth } = storeToRefs(tilePrefs);
const tagFilter = useTagFilterStore();
const { activeTags } = storeToRefs(tagFilter);
const { byGroup } = storeToRefs(cardsStore);

const group = computed(() => groupsStore.get(props.groupId));
const cards = computed(() => {
  const all = byGroup.value[props.groupId] ?? [];
  const withoutSections = hideSections.value
    ? all.filter((c) => !c.parent_card_id)
    : all;
  const selected = activeTags.value;
  if (selected.length === 0) return withoutSections;
  return withoutSections.filter((c) => selected.every((t) => c.tags.includes(t)));
});

// A document is a top-level container that spans multiple levels, so it
// belongs above the level board, not in it. `documentsIn` is the shared
// definition (also used by the home dashboard). Apply the active tag filter
// so documents honor it just like the board cards do.
const documents = computed(() => {
  const docs = documentsIn(byGroup.value[props.groupId] ?? []);
  const selected = activeTags.value;
  return selected.length === 0
    ? docs
    : docs.filter((d) => selected.every((t) => d.tags.includes(t)));
});

const boardCards = computed(() => {
  const docIds = new Set(documents.value.map((d) => d.id));
  return cards.value.filter((c) => !docIds.has(c.id));
});

const showCreate = ref(false);
const newTitle = ref('');
const newContent = ref('');
const createError = ref<string | null>(null);

const deleteTargetId = ref<string | null>(null);
const deleteOpen = ref(false);
const deleteError = ref<string | null>(null);
const deleteCascade = ref(true);

// Live descendants of the delete target, if any are cached locally.
// Used to size the cascade hint in the confirm dialog. Mirrors the
// walk in CardView so behavior stays consistent.
const deleteDescendantCount = computed(() => {
  const rootId = deleteTargetId.value;
  if (!rootId) return 0;
  const reached = new Set<string>([rootId]);
  let grew = true;
  while (grew) {
    grew = false;
    for (const c of Object.values(cardsStore.byID)) {
      if (!c.deleted_at && c.parent_card_id && reached.has(c.parent_card_id) && !reached.has(c.id)) {
        reached.add(c.id);
        grew = true;
      }
    }
  }
  return reached.size - 1;
});

const reorderError = ref<string | null>(null);

watch(
  () => props.groupId,
  async (id) => {
    if (groupsStore.groups.length === 0) await groupsStore.load();
    await cardsStore.loadByGroup(id);
  },
  { immediate: true },
);

const sortedCatalog = computed(() => sortCatalog(group.value?.level_catalog ?? []));

watch(
  [() => props.groupId, sortedCatalog],
  ([gid, cat]) => {
    levelFilter.ensure(gid, cat);
  },
  { immediate: true },
);

const filterState = computed(
  () => levelFilter.byGroup[props.groupId] ?? { selectedEntryIds: [], showMisc: true },
);

// Cards are grouped into columns per catalog entry id. Two catalog
// entries at the same weight get two independent columns.
const visibleColumns = computed(() => {
  const cols = sortedCatalog.value
    .filter((e) => filterState.value.selectedEntryIds.includes(e.id))
    .map((entry) => ({
      key: `level-${entry.id}`,
      label: entry.name,
      entryId: entry.id as string | undefined,
      weight: entry.weight as number | undefined,
      isMisc: false,
      cards: boardCards.value.filter((c) => c.level_entry_id === entry.id),
    }));
  if (filterState.value.showMisc) {
    // Unfiled sits at the far right — the "not yet decided" bucket after the
    // ordered level columns (documents no longer land here).
    cols.push({
      key: 'misc',
      label: 'Unfiled',
      entryId: undefined,
      weight: undefined,
      isMisc: true,
      cards: boardCards.value.filter((c) => !c.level_entry_id),
    });
  }
  return cols;
});

async function onCardDropped(cardId: string, target: { entryId?: string; clearLevel: boolean }) {
  reorderError.value = null;
  try {
    if (target.clearLevel) {
      await cardsStore.update(cardId, { clear_level_entry: true });
    } else if (target.entryId) {
      await cardsStore.update(cardId, { level_entry_id: target.entryId });
    }
  } catch (e) {
    if (e instanceof BackendError) reorderError.value = e.message;
  }
}

function openCreate() {
  newTitle.value = '';
  newContent.value = '';
  createError.value = null;
  showCreate.value = true;
}

async function submitCreate() {
  const title = newTitle.value.trim();
  if (!title) {
    createError.value = 'Title required.';
    return;
  }
  try {
    await cardsStore.create({
      title,
      content: newContent.value,
      group_id: props.groupId,
    });
    showCreate.value = false;
  } catch (e) {
    if (e instanceof BackendError) createError.value = e.message;
  }
}

function openDelete(id: string) {
  deleteTargetId.value = id;
  deleteError.value = null;
  deleteCascade.value = true;
  deleteOpen.value = true;
}

async function confirmDelete() {
  if (!deleteTargetId.value) return;
  try {
    await cardsStore.remove(deleteTargetId.value, deleteCascade.value);
    deleteOpen.value = false;
    deleteTargetId.value = null;
  } catch (e) {
    if (e instanceof BackendError) {
      deleteError.value = e.message;
      deleteOpen.value = true;
    }
  }
}
</script>

<template>
  <div>
    <div class="mb-3 flex items-center justify-between">
      <h1 v-if="group" class="font-serif text-3xl font-medium text-fg">{{ group.name }}</h1>
      <p v-else class="text-sm text-muted-fg">Group not found.</p>
      <div v-if="group" class="flex gap-2">
        <button
          type="button"
          data-test="toggle-sections"
          :aria-pressed="!hideSections"
          :title="hideSections ? 'Show sections (children of decomposed cards) on the grid' : 'Hide sections (children of decomposed cards) from the grid'"
          class="rounded border border-border px-3 py-1 text-sm text-muted-fg hover:bg-muted"
          @click="tilePrefs.toggleSections"
        >{{ hideSections ? 'Show sections' : 'Hide sections' }}</button>
        <button
          type="button"
          data-test="toggle-summaries"
          :aria-pressed="hideSummaries"
          :title="hideSummaries ? 'Show summaries on tiles' : 'Hide summaries on tiles'"
          class="rounded border border-border px-3 py-1 text-sm text-muted-fg hover:bg-muted"
          @click="tilePrefs.toggleSummaries"
        >{{ hideSummaries ? 'Show summaries' : 'Hide summaries' }}</button>
        <button
          type="button"
          class="rounded bg-fg px-3 py-1 text-sm text-surface hover:bg-fg"
          @click="openCreate"
        >+ Card</button>
        <button
          type="button"
          class="rounded border border-accent-border px-3 py-1 text-sm text-accent-fg hover:bg-accent-bg"
          @click="discussGroup"
        >Discuss this group</button>
      </div>
    </div>

    <div v-if="activeTags.length > 0" class="mb-2 flex flex-wrap items-center gap-2 text-xs">
      <span class="text-muted-fg">Filtered by tag</span>
      <button
        v-for="t in activeTags"
        :key="t"
        type="button"
        data-test="active-tag-chip"
        class="inline-flex items-center gap-1 rounded bg-fg px-1.5 py-0.5 text-surface hover:bg-fg"
        :title="`Remove ${t} from filter`"
        @click="tagFilter.remove(t)"
      >
        <span>{{ t }}</span>
        <span aria-hidden="true">×</span>
      </button>
      <button
        v-if="activeTags.length > 1"
        type="button"
        data-test="clear-tag-filter"
        class="text-muted-fg hover:text-fg"
        @click="tagFilter.clear"
      >clear all</button>
    </div>
    <p v-if="reorderError" class="mb-2 text-xs text-destructive-fg">{{ reorderError }}</p>
    <div v-if="cardsStore.loading" class="text-sm text-muted-fg">Loading…</div>
    <div v-else class="flex items-start gap-1 overflow-x-auto">
      <!-- Documents: a dedicated column (not a level, not a drop target) set
           off from the level board by a gap. -->
      <div
        v-if="documents.length > 0"
        data-test="documents-column"
        class="shrink-0 rounded border border-border"
        :style="{ width: documentsWidth + 'px' }"
      >
        <h3 class="border-b border-border py-2 text-center text-xs font-semibold uppercase tracking-wide text-muted-fg">
          Documents
        </h3>
        <div class="space-y-1 p-1.5">
          <CardItem
            v-for="doc in documents"
            :key="doc.id"
            :card="doc"
            :hide-pill="true"
            pinned
            @delete="openDelete"
          />
        </div>
      </div>
      <ResizableSplitter
        v-if="documents.length > 0"
        data-test="documents-splitter"
        :width="documentsWidth"
        :min="160"
        :max="560"
        :default-width="240"
        side="right"
        aria-label="Resize documents column"
        class="w-1.5 shrink-0 self-stretch rounded"
        @update:width="tilePrefs.setDocumentsWidth"
      />
      <div class="flex min-w-0 flex-1 rounded border border-border overflow-x-auto">
        <LevelColumn
          v-for="col in visibleColumns"
          :key="col.key"
          :label="col.label"
          :catalog="sortedCatalog"
          :weight="col.weight"
          :entry-id="col.entryId"
          :is-misc="col.isMisc"
          :cards="col.cards"
          @card-dropped="onCardDropped"
          @delete="openDelete"
        />
        <div v-if="visibleColumns.length === 0" class="text-sm text-muted-fg p-4">
          No columns selected — check a level on the left.
        </div>
      </div>
    </div>

    <ConfirmDialog
      v-model:open="showCreate"
      title="New card"
      description="Create a card in this group."
      confirm-label="Create"
      @confirm="submitCreate"
    >
      <div class="space-y-2">
        <input
          v-model="newTitle"
          type="text"
          placeholder="Title"
          class="w-full rounded border border-border px-2 py-1 text-sm"
        />
        <textarea
          v-model="newContent"
          rows="4"
          placeholder="Markdown content (optional)"
          class="w-full rounded border border-border px-2 py-1 text-sm"
        />
        <p v-if="createError" class="text-xs text-destructive-fg">{{ createError }}</p>
      </div>
    </ConfirmDialog>

    <ConfirmDialog
      v-model:open="deleteOpen"
      title="Move card to Trash?"
      :description="deleteError ?? 'You can restore it later from the Trash view.'"
      confirm-label="Move to Trash"
      destructive
      @confirm="confirmDelete"
    >
      <label
        class="flex items-center gap-2 rounded border border-border bg-nav px-2 py-1.5 text-xs text-fg"
      >
        <input
          v-model="deleteCascade"
          type="checkbox"
          data-test="delete-cascade"
          class="h-3.5 w-3.5"
        />
        <span>
          <template v-if="deleteDescendantCount > 0">
            Also move {{ deleteDescendantCount }} descendant{{ deleteDescendantCount === 1 ? '' : 's' }} to Trash
          </template>
          <template v-else>
            Also move any descendants to Trash
          </template>
        </span>
      </label>
    </ConfirmDialog>
  </div>
</template>
