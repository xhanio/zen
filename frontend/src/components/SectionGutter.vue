<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount, onMounted } from 'vue';
import SectionGradePill from './SectionGradePill.vue';
import { scrollTargetIndex } from '../utils/sectionTarget';
import type { Card, ReviewGrade } from '../types/entity';

// The gutter is one control and N markers, not N controls.
//
//   dots  — one per section, parked at its BOTTOM edge, coloured by its grade.
//           This is the review state of the whole document, readable in one
//           glance, and the keyboard's section picker.
//   cell  — exactly one grade pill, translated to the target's bottom edge so it
//           sits at the end of that section, level with its last line.
//
// Bottom, not top: you grade a section once you have FINISHED reading it, so the
// dot and the pill that docks to it park where your eye lands when the section
// ends — not where it began. The ⋯ actions menu is NOT here; it lives in a
// mirror rail on the far side of the reading column (CardView), driven by the
// targetChange event this gutter emits.
//
// Targeting: the section under the cursor wins. When the cursor is over none of
// them, the section being READ wins (scrollTargetIndex). The fallback is not
// polish — grading is a write and it is one click, so a pill that keeps aiming
// at the last-hovered section would eventually grade something off-screen.
const props = defineProps<{
  sections: Card[];
  anchors: HTMLElement[];
  scrollRoot: HTMLElement | null;
}>();

const emit = defineEmits<{
  // The right-hand ⋯ rail mirrors this: the same target card, docked at the
  // section's TOP. The menu rides the START of the section (an action you reach
  // for as you arrive); the pill rides its END (you grade once you finish).
  targetChange: [card: Card | null, menuTopY: number];
}>();

const gradeColor: Record<ReviewGrade, string> = {
  LGTM: 'var(--l-3-fg)',
  DIGESTED: 'var(--l-1-fg)',
  GRILLED: 'var(--l-0-fg)',
};

// The grade line sits INSET px above a section's bottom edge so the dot rides
// the last line rather than the gap below it. DOT is the marker's own height
// (h-2 = 8px); its bottom is pinned to the grade line.
const INSET = 6;
const DOT = 8;

const root = ref<HTMLElement | null>(null);
const targetIndex = ref(0);
const pointerInside = ref(false);

// Positions are measured, not computed lazily. getBoundingClientRect is not
// reactive, so a computed that reads it caches its first value and never
// re-measures after layout — which left the cell stuck at translateY(0). We
// measure explicitly into refs and re-measure on every event that can move a
// section: mount, the anchor list changing, a new target, and scroll.
const dotBottoms = ref<number[]>([]);
const sectionTops = ref<number[]>([]);
const cellY = ref(0);
const menuTopY = ref(0);

const target = computed<Card | null>(() => props.sections[targetIndex.value] ?? null);
// translateY places the cell's top; the -100% then lifts it by its own height so
// its BOTTOM lands on the grade line — the pill grows upward from the last line.
const cellStyle = computed(() => ({ transform: `translateY(${cellY.value}px) translateY(-100%)` }));

function gradeLine(i: number): number {
  return (dotBottoms.value[i] ?? 0) - INSET;
}

// The section's offset from the top of the gutter, by client rect NOT offsetTop:
// the sections live deep inside the reading column while the gutter is a sibling
// of it, so their offsetParents differ and offsetTops are not comparable. The
// two columns share a top edge and scroll together, so the rect difference is
// stable under scroll and changes only when the layout does. We measure the
// section BOTTOM because the grade control docks at the end of the section.
function measure() {
  if (!root.value) return;
  const base = root.value.getBoundingClientRect().top;
  const rects = props.anchors.map((el) => el.getBoundingClientRect());
  dotBottoms.value = rects.map((r) => Math.max(0, r.bottom - base));
  sectionTops.value = rects.map((r) => Math.max(0, r.top - base));
  cellY.value = gradeLine(targetIndex.value);
  menuTopY.value = sectionTops.value[targetIndex.value] ?? 0;
}

function setTarget(i: number) {
  if (i >= 0 && i < props.sections.length) {
    targetIndex.value = i;
    cellY.value = gradeLine(i);
    menuTopY.value = sectionTops.value[i] ?? 0;
  }
}

function syncToScroll() {
  measure();
  if (pointerInside.value || !props.scrollRoot) return;
  // Selection is by which section is IN VIEW (its top), independent of where the
  // grade control is drawn (its bottom).
  const tops = props.anchors.map((el) => el.getBoundingClientRect().top);
  const rect = props.scrollRoot.getBoundingClientRect();
  setTarget(scrollTargetIndex(tops, rect.top, props.scrollRoot.clientHeight));
}

// Listeners live on the anchors, which belong to the parent. Attach and detach
// them as the rendered list changes, or a filtered-out section keeps a live
// handler pointing at a stale index.
const enterHandlers = new Map<HTMLElement, () => void>();
function bindAnchors(els: HTMLElement[]) {
  enterHandlers.forEach((h, el) => el.removeEventListener('mouseenter', h));
  enterHandlers.clear();
  els.forEach((el, i) => {
    const h = () => { pointerInside.value = true; setTarget(i); };
    el.addEventListener('mouseenter', h);
    enterHandlers.set(el, h);
  });
}

function onPaneLeave() { pointerInside.value = false; syncToScroll(); }

watch(() => props.anchors, (els) => {
  bindAnchors(els);
  if (targetIndex.value >= props.sections.length) targetIndex.value = 0;
  measure();
}, { immediate: true, flush: 'post' });

watch(() => props.scrollRoot, (pane, old) => {
  if (old) {
    old.removeEventListener('scroll', syncToScroll);
    old.removeEventListener('mouseleave', onPaneLeave);
  }
  if (pane) {
    pane.addEventListener('scroll', syncToScroll, { passive: true });
    pane.addEventListener('mouseleave', onPaneLeave);
  }
}, { immediate: true });

// Keep the mirror rail in lockstep: whenever the target section or its top moves,
// hand both to the parent so the ⋯ menu tracks the same section at its top.
watch([target, menuTopY], ([t, y]) => emit('targetChange', t, y), { immediate: true });

onMounted(() => { bindAnchors(props.anchors); measure(); });
onBeforeUnmount(() => {
  enterHandlers.forEach((h, el) => el.removeEventListener('mouseenter', h));
  if (props.scrollRoot) {
    props.scrollRoot.removeEventListener('scroll', syncToScroll);
    props.scrollRoot.removeEventListener('mouseleave', onPaneLeave);
  }
});
</script>

<template>
  <div
    ref="root"
    data-test="section-gutter"
    class="relative w-[34px] shrink-0 self-stretch"
  >
    <button
      v-for="(s, i) in sections"
      :key="s.id"
      type="button"
      :data-test="`gutter-dot-${s.id}`"
      :aria-label="`Grade: ${s.title}`"
      :title="s.title"
      class="absolute left-[13px] h-2 w-2 cursor-pointer rounded-full border-0 p-0 opacity-60 transition-opacity hover:opacity-100 focus-visible:opacity-100 focus-visible:ring-2 focus-visible:ring-accent-fg"
      :style="{ top: `${gradeLine(i) - DOT}px`, background: gradeColor[s.review_grade] }"
      @focus="setTarget(i)"
      @mouseenter="setTarget(i)"
    ></button>

    <div
      v-if="target"
      data-test="gutter-cell"
      class="absolute left-0 top-0 flex w-[34px] justify-end pr-1 transition-transform duration-200 ease-out"
      :style="cellStyle"
    >
      <SectionGradePill :card="target" />
    </div>
  </div>
</template>
