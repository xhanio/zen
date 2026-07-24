<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRenderedSections } from '../composables/useRenderedSections';
import { storeToRefs } from 'pinia';
import { useRouter } from 'vue-router';
import { useCardsStore } from '../stores/cards';
import { useTagsStore } from '../stores/tags';
import { useGroupsStore } from '../stores/groups';
import CardBody from '../components/CardBody.vue';
import ContentBody from '../components/ContentBody.vue';
import CardExportButton from '../components/CardExportButton.vue';
import ConfirmDialog from '../components/ConfirmDialog.vue';
import SectionGutter from '../components/SectionGutter.vue';
import SectionActionsMenu from '../components/SectionActionsMenu.vue';
import ContainerScoreStrip from '../components/ContainerScoreStrip.vue';
import TagChipEditor from '../components/TagChipEditor.vue';
import AskBubble from '../components/chat/AskBubble.vue';
import { useSelectionBubble } from '../composables/useSelectionBubble';
import { useChatSidebar } from '../composables/useChatSidebar';
import { useTilePrefsStore } from '../stores/tilePrefs';
import { useContainerFilterStore } from '../stores/containerFilter';
import { BackendError } from '../types/api';
import type { Highlight } from '../utils/highlightText';
import type { Card } from '../types/entity';
import { ancestorsOf } from '../utils/titlePath';
import { levelColor } from '../utils/levelPalette';
import { sortCatalog } from '../utils/levelCatalog';
import CardReferencesPanel from '../components/CardReferencesPanel.vue';
import SectionConversationChip from '../components/SectionConversationChip.vue';

const props = defineProps<{ cardId: string }>();

const contentRoot = ref<HTMLElement | null>(null);
const selection = useSelectionBubble(contentRoot);

const router = useRouter();
const cardsStore = useCardsStore();
const tagsStore = useTagsStore();
const sidebar = useChatSidebar();
const tilePrefs = useTilePrefsStore();
const containerFilter = useContainerFilterStore();
const { byID, byChildren } = storeToRefs(cardsStore);
const { showTrashedSections } = storeToRefs(tilePrefs);

const allTagNames = computed(() => tagsStore.tags.map((t) => t.name));
const tagsError = ref<string | null>(null);
const savingTags = ref(false);

const card = computed(() => byID.value[props.cardId]);
const parent = computed(() =>
  card.value?.parent_card_id ? byID.value[card.value.parent_card_id] : null,
);
// Full ancestor chain (root-first) for the breadcrumb above the title. Empty for
// a top-level card. Built from the loaded store, so each crumb links to its card.
const ancestors = computed(() =>
  card.value ? ancestorsOf(card.value, byID.value) : [],
);
const groupsStore = useGroupsStore();
const group = computed(() =>
  card.value?.group_id ? groupsStore.get(card.value.group_id) : undefined,
);
const catalog = computed(() => group.value?.level_catalog ?? []);

// Tags are group-scoped; load this card's group's tags so the tag editor's
// autocomplete (allTagNames) is populated even though the sidebar tag cloud
// isn't mounted in reading mode.
watch(
  () => card.value?.group_id,
  (gid) => {
    if (gid) void tagsStore.load(gid);
  },
  { immediate: true },
);
const cardWeight = computed(() => {
  const eid = card.value?.level_entry_id;
  if (!eid) return null;
  return catalog.value.find((e) => e.id === eid)?.weight ?? null;
});
const palette = computed(() => levelColor(catalog.value, cardWeight.value));
const hasReferences = computed(() => (card.value?.references?.length ?? 0) > 0);

// Color-to-level legend at the top of the reading column. Applies to
// both leaf and container views so the outer structure is identical
// regardless of what the current card is. On a leaf the legend has one
// item (the leaf's own level). On a container it enumerates the levels
// actually present in the container's live children.
// One legend chip per catalog entry (id). Since each catalog entry has a
// distinct id, same-weight entries produce two independent chips — the
// filter can hide one without hiding the other. TBD_KEY covers Unfiled
// sections (level_entry_id === null).
const TBD_KEY = '__tbd__';
const legend = computed(() => {
  const c = card.value;
  if (!c) return [];
  const catalog = group.value?.level_catalog ?? [];
  const idsSeen = new Set<string>();
  let seenTbd = false;
  const sections = isContainerView.value ? liveChildren.value : [c];
  for (const section of sections) {
    if (section.level_entry_id) idsSeen.add(section.level_entry_id);
    else seenTbd = true;
  }
  const items: Array<{ key: string; name: string; weight: number | null }> = [];
  if (seenTbd) items.push({ key: TBD_KEY, name: 'Unfiled', weight: null });
  for (const e of sortCatalog(catalog)) {
    if (idsSeen.has(e.id)) items.push({ key: e.id, name: e.name, weight: e.weight });
  }
  return items;
});

const liveChildren = computed(() =>
  (byChildren.value[props.cardId] ?? []).filter((c) => !c.deleted_at),
);
const isContainerView = computed(() =>
  !!card.value && liveChildren.value.length > 0,
);

// The legend folds a whole level at once: clicking a chip collapses every
// section of that level, or expands them all if they're already collapsed.
// Collapse state is per-card (containerFilter), shared with the per-section
// title toggles in ContainerBody.
function legendCardIds(key: string): string[] {
  const secs = isContainerView.value ? liveChildren.value : (card.value ? [card.value] : []);
  return secs
    .filter((s) => (key === TBD_KEY ? !s.level_entry_id : s.level_entry_id === key))
    .map((s) => s.id);
}
function isLevelCollapsed(key: string): boolean {
  const ids = legendCardIds(key);
  return ids.length > 0 && ids.every((id) => containerFilter.isCollapsed(id));
}
function toggleLevel(key: string) {
  const ids = legendCardIds(key);
  containerFilter.setCollapsed(ids, !isLevelCollapsed(key));
}

// Grade gutter. It renders one dot per section and one floating pill, so it
// needs the sections it is grading and their DOM nodes.
//   - container: the sections are the rendered children; their nodes arrive via
//     ContainerBody's @anchors event.
//   - leaf: the section is the card itself; its node is the leaf <section>.
const anchors = ref<HTMLElement[]>([]);
const leafSection = ref<HTMLElement | null>(null);
const scrollRoot = ref<HTMLElement | null>(null);
onMounted(() => { scrollRoot.value = document.querySelector('main'); });

const containerSections = useRenderedSections(computed(() => props.cardId));
const gutterSections = computed<Card[]>(() =>
  isContainerView.value ? containerSections.value : (card.value ? [card.value] : []),
);

// The grade pill (dots + L/D/G) sits in a rail on the LEFT of the reading
// column; the ⋯ actions menu mirrors it on the RIGHT. The gutter owns targeting
// and hands us the current section + its TOP y so the right rail can dock the
// menu at the section's top (the pill stays at the section's bottom).
const gutterTarget = ref<Card | null>(null);
const gutterMenuY = ref(0);
function onGutterTarget(t: Card | null, y: number) {
  gutterTarget.value = t;
  gutterMenuY.value = y;
}

watch([leafSection, isContainerView], () => {
  if (!isContainerView.value && leafSection.value) anchors.value = [leafSection.value];
}, { flush: 'post' });

const deleteOpen = ref(false);
const deleteError = ref<string | null>(null);
const deleteCascade = ref(true);

// Live descendants of the current card, if any are cached locally.
// Used to size the cascade hint in the confirm dialog.
const descendantCount = computed(() => {
  const rootId = props.cardId;
  const trashed = new Set<string>([rootId]);
  let grew = true;
  while (grew) {
    grew = false;
    for (const c of Object.values(byID.value)) {
      if (!c.deleted_at && c.parent_card_id && trashed.has(c.parent_card_id) && !trashed.has(c.id)) {
        trashed.add(c.id);
        grew = true;
      }
    }
  }
  return trashed.size - 1;
});

const purgeOpen = ref(false);
const purgeError = ref<string | null>(null);

async function onRestore() {
  try {
    await cardsStore.restore(props.cardId);
  } catch (e) {
    if (e instanceof BackendError) {
      // surface via existing error UI if needed in future iteration
    }
  }
}

async function confirmPurge() {
  try {
    await cardsStore.purge(props.cardId);
    purgeOpen.value = false;
    router.push({ name: 'trash' });
  } catch (e) {
    if (e instanceof BackendError) {
      purgeError.value = e.message;
    }
  }
}

watch(
  () => props.cardId,
  async (id) => {
    // Filter is scoped to the current anchor card — reset it whenever
    // the reader lands on a different card so a stale filter from a
    // previous container doesn't quietly hide sections here.
    containerFilter.clear();
    await cardsStore.loadOne(id);
    // A container may now carry a preamble (non-empty content), so the
    // old empty-content bootstrap in CardBody.maybeLoad won't fire. Load
    // the anchor's children directly so container detection is
    // content-independent for the document being read.
    await cardsStore.loadChildren(id, showTrashedSections.value);
  },
  { immediate: true },
);

watch(
  () => card.value?.parent_card_id,
  async (pid) => {
    if (pid && !byID.value[pid]) await cardsStore.loadOne(pid);
  },
  { immediate: true },
);

async function saveTags(next: string[]) {
  tagsError.value = null;
  savingTags.value = true;
  try {
    // cardsStore.update refreshes tags itself when the request touches
    // the tag set, so no separate tagsStore.load() call is needed.
    await cardsStore.update(props.cardId, { tags: next });
  } catch (e) {
    if (e instanceof BackendError) tagsError.value = e.message;
  } finally {
    savingTags.value = false;
  }
}

async function confirmDelete() {
  const groupId = card.value?.group_id;
  try {
    await cardsStore.remove(props.cardId, deleteCascade.value);
    if (groupId) router.push({ name: 'group', params: { groupId } });
    else router.push({ name: 'home' });
  } catch (e) {
    if (e instanceof BackendError) {
      deleteError.value = e.message;
      deleteOpen.value = true;
    }
  }
}

const cardHighlights = computed<Highlight[]>(() =>
  (card.value?.references ?? []).map((r) => ({ id: r.id, text: r.selection_text })),
);

function onContentClick(event: MouseEvent) {
  // Two paths: composedPath() handles shadow-DOM marks (HtmlBody), and the
  // event.target.closest fallback handles light-DOM marks (Markdown/Text).
  const candidates: Element[] = [];
  const path = typeof event.composedPath === 'function' ? event.composedPath() : [];
  for (const el of path) {
    if (el instanceof Element) candidates.push(el);
  }
  const target = event.target as Element | null;
  if (target) {
    const closest = target.closest?.('[data-ref-id]');
    if (closest) candidates.push(closest);
  }
  for (const el of candidates) {
    const refId = (el as HTMLElement).dataset?.refId;
    if (!refId) continue;
    const ref = card.value?.references?.find((r) => r.id === refId);
    if (!ref) return;
    if (ref.conversation_id) {
      void sidebar.openForConversation(ref.conversation_id);
    } else {
      void router.push({ name: 'card', params: { cardId: ref.derived_card_id } });
    }
    event.preventDefault();
    return;
  }
}
</script>

<template>
  <div v-if="card" class="mx-auto flex h-full w-full max-w-6xl items-start gap-3">
    <SectionGutter
      v-if="!card.deleted_at && gutterSections.length > 0"
      :sections="gutterSections"
      :anchors="anchors"
      :scroll-root="scrollRoot"
      @target-change="onGutterTarget"
    />
    <div class="min-w-0 flex-1">
    <div class="mb-4 flex items-center justify-between">
      <RouterLink
        v-if="group"
        :to="{ name: 'group', params: { groupId: group.id } }"
        class="inline-flex items-center gap-1.5 text-xs text-muted-fg hover:text-fg"
        data-test="back-to-group"
      >
        <span aria-hidden="true">←</span>
        <span>{{ group.name }}</span>
      </RouterLink>
      <span v-else></span>
      <div class="flex items-center gap-2">
        <CardExportButton v-if="!card.deleted_at" :card="card" />
        <button
          v-if="!card.deleted_at"
          type="button"
          data-test="card-action-trash"
          class="rounded px-1.5 py-0.5 text-base leading-none text-destructive-fg hover:bg-destructive-bg"
          title="Move to Trash"
          @click="deleteOpen = true"
        >✕</button>
      </div>
    </div>
    <div
      v-if="card.deleted_at"
      data-test="trash-banner"
      class="mb-3 rounded border border-border bg-muted px-3 py-2 text-sm text-fg"
    >
      This card is in Trash. Restore it to bring it back, or delete permanently.
    </div>
    <div class="mb-2 flex items-start justify-between gap-3">
      <div class="flex-1">
        <nav
          v-if="ancestors.length"
          class="mb-2 flex flex-wrap items-center gap-x-1.5 gap-y-0.5 text-[0.8125rem] leading-snug text-muted-fg"
          aria-label="Card location"
        >
          <template v-for="(a, i) in ancestors" :key="a.id">
            <span v-if="i > 0" class="opacity-60" aria-hidden="true">›</span>
            <RouterLink
              :to="{ name: 'card', params: { cardId: a.id } }"
              class="hover:text-fg hover:underline"
            >{{ a.title }}</RouterLink>
          </template>
        </nav>
        <h1 class="font-serif text-3xl font-medium leading-tight text-fg">{{ card.title }}</h1>
      </div>
      <div class="flex gap-2">
        <button
          v-if="isContainerView"
          type="button"
          data-test="toggle-trashed-sections"
          :aria-pressed="showTrashedSections"
          class="rounded px-3 py-1 text-sm text-muted-fg hover:bg-muted"
          @click="tilePrefs.toggleShowTrashedSections"
        >{{ showTrashedSections ? 'Hide trashed sections' : 'Show trashed sections' }}</button>
        <SectionConversationChip
          :anchor-id="cardId"
          :source-conversation-id="card.source_conversation_id"
          persistent
          :disabled="!!card.deleted_at"
        />
        <button
          v-if="card.deleted_at"
          type="button"
          data-test="card-action-restore"
          class="rounded border border-border px-3 py-1 text-sm text-fg hover:bg-muted"
          @click="onRestore"
        >Restore</button>
        <button
          v-if="card.deleted_at"
          type="button"
          data-test="card-action-purge"
          class="rounded border border-destructive-border px-3 py-1 text-sm text-destructive-fg hover:bg-destructive-bg"
          @click="purgeOpen = true"
        >Delete Permanently</button>
      </div>
    </div>
    <div class="mb-4">
      <TagChipEditor
        :tags="card.tags"
        :all-tags="allTagNames"
        :saving="savingTags"
        :error="tagsError"
        :readonly="!!card.deleted_at"
        @update="saveTags"
      />
    </div>
    <div ref="contentRoot" class="mb-2 relative" @click="onContentClick">
      <!-- Every card view — leaf or container — sits inside the same
           outer skeleton: a legend row at the top, then a space-y-1
           wrapper holding one or more section shells. For leaves we
           inline the single section here; for containers we hand off
           to ContainerBody (which fills the wrapper with N shells). -->
      <div class="space-y-1.5">
        <ContainerScoreStrip
          v-if="isContainerView"
          :card="card"
          class="mb-3"
        />
        <div
          v-if="legend.length > 0"
          data-test="level-legend"
          class="mb-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-fg"
        >
          <button
            v-for="item in legend"
            :key="item.key"
            type="button"
            :data-test="`legend-toggle-${item.key}`"
            :aria-pressed="isLevelCollapsed(item.key)"
            :title="isLevelCollapsed(item.key) ? `Expand ${item.name} sections` : `Collapse ${item.name} sections`"
            :class="[
              'inline-flex items-center gap-1.5 rounded px-1.5 py-0.5 transition-opacity hover:bg-muted',
              isLevelCollapsed(item.key) ? 'opacity-40 line-through' : '',
            ]"
            @click="toggleLevel(item.key)"
          >
            <span class="inline-block h-2 w-2 rounded-full" :style="{ backgroundColor: levelColor(catalog, item.weight).fg }" aria-hidden="true"></span>
            <span>{{ item.name }}</span>
          </button>
        </div>
        <!-- The container's own content is the preamble: pure document
             metadata (e.g. a date/status block), rendered right above the
             sections in the same body font. It is NOT a section — no ribbon,
             grade pill, actions menu, or drag handle, and it's excluded from
             the grade gutter/legend/score-strip (those read children only). -->
        <div
          v-if="isContainerView && (card.content ?? '').trim() !== ''"
          data-test="container-preamble"
          class="pl-5 pr-6 py-1 bg-paper"
        >
          <ContentBody :source="card.content" :format="card.format ?? 'markdown'" />
        </div>
        <section
          v-if="!isContainerView"
          ref="leafSection"
          :data-card-id="card.id"
          class="flex gap-4 bg-paper"
        >
          <div
            class="w-1 shrink-0 self-stretch"
            :style="{ backgroundColor: palette.fg }"
          ></div>
          <div class="min-w-0 flex-1">
            <CardBody :card="card" :highlights="cardHighlights" />
          </div>
        </section>
        <CardBody v-else :card="card" :highlights="cardHighlights" @anchors="(els) => (anchors = els)" />
      </div>
    </div>
    <AskBubble
      v-if="!card.deleted_at"
      :rect="selection.rect.value"
      :selection-text="selection.text.value"
      anchor-kind="card"
      :anchor-id="selection.hostCardId.value ?? cardId"
      @opened="selection.clear()"
    />

    <footer class="mt-8 border-t border-border pt-3 text-xs text-muted-fg space-y-1">
      <p v-if="card.genesis">{{ card.genesis }}</p>
      <p v-if="card.parent_card_id">
        <RouterLink
          v-if="parent && !parent.deleted_at"
          :to="{ name: 'card', params: { cardId: parent.id } }"
          class="text-accent-fg hover:underline"
        >Parent: {{ parent.title }}</RouterLink>
        <span v-else-if="parent && parent.deleted_at">Parent: {{ parent.title }} (in Trash)</span>
        <span v-else>Parent card</span>
      </p>
    </footer>

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
          <template v-if="descendantCount > 0">
            Also move {{ descendantCount }} descendant{{ descendantCount === 1 ? '' : 's' }} to Trash
          </template>
          <template v-else>
            Also move any descendants to Trash
          </template>
        </span>
      </label>
    </ConfirmDialog>
    <ConfirmDialog
      v-model:open="purgeOpen"
      title="Delete card permanently?"
      :description="purgeError ?? 'This cannot be undone.'"
      confirm-label="Delete"
      destructive
      @confirm="confirmPurge"
    />
    </div>
    <div
      v-if="!card.deleted_at && gutterSections.length > 0"
      data-test="section-actions-rail"
      class="relative w-[34px] shrink-0 self-stretch"
    >
      <div
        v-if="gutterTarget"
        class="absolute left-0 top-2 flex w-[34px] justify-center transition-transform duration-200 ease-out"
        :style="{ transform: `translateY(${gutterMenuY}px)` }"
      >
        <SectionActionsMenu :card="gutterTarget" />
      </div>
    </div>
    <CardReferencesPanel v-if="hasReferences" :root-card="card" />
  </div>
  <p v-else-if="cardsStore.loading" class="text-sm text-muted-fg">Loading…</p>
  <p v-else class="text-sm text-muted-fg">Card not found.</p>
</template>
