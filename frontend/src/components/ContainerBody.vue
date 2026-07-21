<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useCardsStore } from '../stores/cards';
import { useTilePrefsStore } from '../stores/tilePrefs';
import { useGroupsStore } from '../stores/groups';
import { useContainerFilterStore } from '../stores/containerFilter';
import CardBody from './CardBody.vue';
import HtmlBody from './HtmlBody.vue';
import SectionConversationChip from './SectionConversationChip.vue';
import { useRenderedSections } from '../composables/useRenderedSections';
import { colorForCard } from '../utils/levelPalette';
import type { Card } from '../types/entity';

// The stitched container view. Each child gets a slim vertical color
// bar (level palette) running its full height in the left margin.
// In v0.11 the ribbon doubles as a drag handle — grab it to reorder
// sections. Every section is itself a drop target: dragover computes
// which half of the section the cursor is in (top → insert before,
// bottom → insert after) and paints a dashed insertion line at the
// corresponding edge. Text selection inside the body is unaffected
// because only the ribbon is `draggable`.

const props = defineProps<{ parent: Card }>();

const cardsStore = useCardsStore();
const tilePrefs = useTilePrefsStore();
const { showTrashedSections } = storeToRefs(tilePrefs);
const groupsStore = useGroupsStore();
const catalog = computed(() => groupsStore.get(props.parent.group_id)?.level_catalog ?? []);

const { byChildren } = storeToRefs(cardsStore);
const containerFilter = useContainerFilterStore();

// Every live section in position order — collapsed sections included. A folded
// section still renders (title only, see template) so the outline stays whole.
// The grade gutter, by contrast, reads `rendered` below (which DROPS collapsed
// sections), so a folded section carries no grade dot.
const allSections = computed(() => {
  const all = (byChildren.value[props.parent.id] ?? []).slice().sort((a, b) => a.position - b.position);
  return showTrashedSections.value ? all : all.filter((c) => !c.deleted_at);
});
function isCollapsed(child: Card): boolean {
  return containerFilter.isCollapsed(child.id);
}

// Clicking anywhere on a section's article means the reader has engaged with it,
// so escalate its grade to DIGESTED (never lower). No collapse/expand dance
// required — a plain click on the section is enough. Best-effort: a failed
// escalation is swallowed (the store already rolled back), never thrown.
function markSectionRead(child: Card) {
  cardsStore.escalateReviewGrade(child.id, 'DIGESTED').catch(() => {});
}

// Render the section title as a real <h2> inside the card's OWN wrapper +
// <style> block, so it inherits that card's exact heading CSS (a .zen-spec
// card gets .zen-spec h2, a .zdoc card gets .zdoc h2). Returns null when the
// card has no html wrapper/style — the template falls back to a plain <h2>.
function buildTitleHtml(child: Card): string | null {
  if ((child.format ?? 'markdown') !== 'html' || !(child.content ?? '').trim()) return null;
  const root = new DOMParser().parseFromString(child.content ?? '', 'text/html').body.firstElementChild;
  if (!root) return null;
  const wrapper = root.cloneNode(false) as HTMLElement;
  const style = root.querySelector('style');
  if (style) wrapper.appendChild(style.cloneNode(true));
  const h2 = document.createElement('h2');
  h2.textContent = child.title;
  wrapper.appendChild(h2);
  return wrapper.outerHTML;
}
const titleHtmls = computed(() => {
  const m: Record<string, string | null> = {};
  for (const c of allSections.value) m[c.id] = buildTitleHtml(c);
  return m;
});

async function refresh() {
  await cardsStore.loadChildren(props.parent.id, showTrashedSections.value);
}

onMounted(refresh);
watch(showTrashedSections, refresh);
watch(() => props.parent.id, refresh);

const rendered = useRenderedSections(computed(() => props.parent.id));

// The grade gutter renders in CardView, outside this component, but it needs
// each section's DOM node to place a dot and target the pill. Read them straight
// from the rendered DOM after each layout, in document order. Querying beats
// per-element :ref callbacks here: a callback-populated array has to be cleared
// to drop removed sections, and the clear wipes what the callbacks filled before
// they re-fire — so the emit went out empty. The DOM is the source of truth.
const emit = defineEmits<{ (e: 'anchors', els: HTMLElement[]): void }>();
const listEl = ref<HTMLElement | null>(null);
function emitAnchors() {
  if (!listEl.value) return;
  emit('anchors', Array.from(listEl.value.querySelectorAll<HTMLElement>('section[data-test="section-shell"]')));
}
watch(rendered, async () => {
  await nextTick();
  emitAnchors();
}, { immediate: true, flush: 'post' });
onMounted(emitAnchors);

// dragOverIndex tracks where the dashed insertion line should paint:
// index N means "before section N" (equivalently, after section N-1).
// Range is [0, rendered.length].
const dragOverIndex = ref<number | null>(null);
const draggingId = ref<string | null>(null);

// Auto-scroll while dragging: native HTML5 drag doesn't scroll the page,
// so a user with a long container can't reach sections outside the
// viewport. Watch cursor Y on document-level `dragover` and scroll the
// enclosing MainPanel via rAF.
//
// Two guards protect against "grabbing a ribbon near the edge starts
// scrolling immediately":
//   1. Small hot zone (60px) at each viewport edge.
//   2. Auto-scroll only arms after the cursor has moved at least
//      DRAG_ARM_PX from the drag origin, so a still-held ribbon near
//      the edge doesn't scroll out from under you.
const SCROLL_HOT_ZONE_PX = 60;
const SCROLL_MAX_SPEED_PX_PER_FRAME = 16;
const DRAG_ARM_PX = 40;
let autoScrollRaf: number | null = null;
let cursorY = 0;
let dragStartY = 0;
let autoScrollArmed = false;
let scrollTarget: HTMLElement | null = null;

function autoScrollTick() {
  if (!scrollTarget || !draggingId.value || !autoScrollArmed) {
    autoScrollRaf = null;
    return;
  }
  const vh = window.innerHeight;
  let dy = 0;
  if (cursorY < SCROLL_HOT_ZONE_PX) {
    const t = 1 - cursorY / SCROLL_HOT_ZONE_PX;
    dy = -Math.max(1, Math.round(SCROLL_MAX_SPEED_PX_PER_FRAME * t));
  } else if (cursorY > vh - SCROLL_HOT_ZONE_PX) {
    const t = 1 - (vh - cursorY) / SCROLL_HOT_ZONE_PX;
    dy = Math.max(1, Math.round(SCROLL_MAX_SPEED_PX_PER_FRAME * t));
  }
  if (dy !== 0) scrollTarget.scrollBy({ top: dy });
  autoScrollRaf = requestAnimationFrame(autoScrollTick);
}

function onDocDragOver(e: DragEvent) {
  if (!draggingId.value) return;
  cursorY = e.clientY;
  if (!autoScrollArmed && Math.abs(cursorY - dragStartY) >= DRAG_ARM_PX) {
    autoScrollArmed = true;
  }
  if (autoScrollArmed && autoScrollRaf === null) {
    autoScrollRaf = requestAnimationFrame(autoScrollTick);
  }
}

function stopAutoScroll() {
  if (autoScrollRaf !== null) {
    cancelAnimationFrame(autoScrollRaf);
    autoScrollRaf = null;
  }
  scrollTarget = null;
  autoScrollArmed = false;
  document.removeEventListener('dragover', onDocDragOver);
}

function onRibbonDragStart(child: Card, e: DragEvent) {
  if (!e.dataTransfer) return;
  e.dataTransfer.setData('text/zen-section-id', child.id);
  e.dataTransfer.effectAllowed = 'move';
  draggingId.value = child.id;
  // The reading view is inside MainPanel — a <main> element with
  // overflow-auto. Grab it once per drag; if it can't be found, no-op.
  scrollTarget = document.querySelector('main');
  cursorY = e.clientY;
  dragStartY = e.clientY;
  autoScrollArmed = false;
  document.addEventListener('dragover', onDocDragOver);
}

function onRibbonDragEnd() {
  draggingId.value = null;
  dragOverIndex.value = null;
  stopAutoScroll();
}

// Cursor top-half → insert BEFORE this section (index = idx).
// Cursor bottom-half → insert AFTER (index = idx + 1).
function insertionIndexForSection(idx: number, section: HTMLElement, clientY: number): number {
  const rect = section.getBoundingClientRect();
  return clientY < rect.top + rect.height / 2 ? idx : idx + 1;
}

function onSectionDragOver(idx: number, e: DragEvent) {
  if (!draggingId.value) return;
  e.preventDefault();
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
  const section = e.currentTarget as HTMLElement;
  dragOverIndex.value = insertionIndexForSection(idx, section, e.clientY);
}

async function onSectionDrop(idx: number, e: DragEvent) {
  e.preventDefault();
  const id = e.dataTransfer?.getData('text/zen-section-id');
  const section = e.currentTarget as HTMLElement;
  const gapIndex = insertionIndexForSection(idx, section, e.clientY);
  dragOverIndex.value = null;
  draggingId.value = null;
  stopAutoScroll();
  if (!id) return;
  const siblings = allSections.value;
  const oldIdx = siblings.findIndex((c) => c.id === id);
  if (oldIdx < 0) return;
  const target = oldIdx >= gapIndex ? gapIndex : gapIndex - 1;
  if (target === oldIdx) return;
  try {
    await cardsStore.reorderChild(id, target);
  } catch {
    // Store rolls back via loadChildren; toast plumbing is polish for later.
  }
}
</script>

<template>
  <div ref="listEl" class="space-y-1.5">
    <template v-for="(child, idx) in allSections" :key="child.id">
      <section
        v-if="child.deleted_at"
        class="flex gap-4 bg-muted"
      >
        <div class="w-1 shrink-0 bg-muted"></div>
        <div class="flex-1 rounded px-3 py-2 text-sm text-muted-fg">
          [section in Trash — {{ child.title }}]
          <RouterLink
            :to="{ name: 'card', params: { cardId: child.id } }"
            class="ml-2 text-accent-fg hover:underline"
          >Restore</RouterLink>
        </div>
      </section>
      <!-- Live section. The title header is always shown and toggles this
           section (per-section accordion); the legend folds a whole level at
           once. Collapsed → title only, no body, no grade — a collapsed row
           is `section-collapsed`, not `section-shell`, so the gutter (which
           reads the expanded list) targets it with no dot. -->
      <section
        v-else
        :data-card-id="child.id"
        :data-test="isCollapsed(child) ? 'section-collapsed' : 'section-shell'"
        class="group relative flex gap-4 bg-paper"
        :class="draggingId === child.id ? 'opacity-50' : ''"
        @dragover="(e) => onSectionDragOver(idx, e)"
        @drop="(e) => onSectionDrop(idx, e)"
      >
        <div class="absolute right-2 top-2 z-10">
          <SectionConversationChip
            :anchor-id="child.id"
            :source-conversation-id="child.source_conversation_id"
          />
        </div>
        <!-- Feathered insertion line: 3px tall gradient centered in the 6px
             gap. Peaks at ~70% opacity in the middle and fades to transparent
             at both ends. Absolute-positioned so its appearance never nudges
             sibling sections. -->
        <div
          v-if="dragOverIndex === idx"
          aria-hidden="true"
          class="pointer-events-none absolute left-0 right-0 -top-[5px] h-[3px]"
          style="background: linear-gradient(90deg, transparent 0%, rgba(148,163,184,0.7) 50%, transparent 100%);"
        ></div>
        <div
          v-if="idx === allSections.length - 1 && dragOverIndex === allSections.length"
          aria-hidden="true"
          class="pointer-events-none absolute left-0 right-0 -bottom-[5px] h-[3px]"
          style="background: linear-gradient(90deg, transparent 0%, rgba(148,163,184,0.7) 50%, transparent 100%);"
        ></div>
        <div
          data-test="section-ribbon"
          draggable="true"
          class="w-1 shrink-0 self-stretch cursor-row-resize"
          :style="{ backgroundColor: colorForCard(catalog, child).fg }"
          @dragstart="(e) => onRibbonDragStart(child, e)"
          @dragend="onRibbonDragEnd"
        ></div>
        <div class="min-w-0 flex-1 pr-6" @click="markSectionRead(child)">
          <div
            data-test="section-title"
            role="button"
            tabindex="0"
            class="cursor-pointer"
            :aria-expanded="!isCollapsed(child)"
            :title="isCollapsed(child) ? 'Expand section' : 'Collapse section'"
            @click="containerFilter.toggleCard(child.id)"
            @keydown.enter.prevent="containerFilter.toggleCard(child.id)"
          >
            <HtmlBody v-if="titleHtmls[child.id]" :source="titleHtmls[child.id]!" />
            <h2
              v-else
              class="mt-[.5em] mb-[.5em] text-[1.15rem] font-semibold leading-[1.75] text-fg"
              :class="isCollapsed(child) ? '' : 'border-b border-[rgba(128,128,128,0.3)] pb-[.25em]'"
            >{{ child.title }}</h2>
          </div>
          <CardBody v-if="!isCollapsed(child)" :card="child" />
        </div>
      </section>
    </template>
    <p v-if="allSections.length === 0" class="italic text-muted-fg">
      No sections to display.
    </p>
  </div>
</template>
