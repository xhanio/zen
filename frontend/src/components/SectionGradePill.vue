<script setup lang="ts">
import type { Card, ReviewGrade } from '../types/entity';
import { useCardsStore } from '../stores/cards';

const props = defineProps<{ card: Card }>();
const cardsStore = useCardsStore();

const grades: ReviewGrade[] = ['LGTM', 'DIGESTED', 'GRILLED'];

// The full grade word — shown on the pill, read by screen readers, and
// used as the tooltip.
const labels: Record<ReviewGrade, string> = {
  LGTM: 'LGTM',
  DIGESTED: 'Digested',
  GRILLED: 'Grilled',
};

// Fixed per-grade colors. Cool → warm → hot heat ladder:
//   LGTM     = sky      (cool surface glance)
//   DIGESTED = amber    (absorbed / warmed to it)
//   GRILLED  = rose     (interrogated / on fire)
// Reuses Zen's level-palette CSS variables so the pill stays in-family.
const gradeColor: Record<ReviewGrade, string> = {
  LGTM: 'var(--l-3-fg)',
  DIGESTED: 'var(--l-1-fg)',
  GRILLED: 'var(--l-0-fg)',
};

async function pick(g: ReviewGrade) {
  if (g === props.card.review_grade) return;
  try {
    await cardsStore.setReviewGrade(props.card.id, g);
  } catch {
    // Store already rolled back and reloaded; no toast plumbing for v0.12.
  }
}
</script>

<template>
  <div
    class="inline-flex flex-col items-stretch gap-px rounded-md bg-muted p-0.5 shadow-[0_2px_8px_rgba(15,23,42,0.10)]"
    data-test="grade-pill"
  >
    <button
      v-for="g in grades"
      :key="g"
      type="button"
      :aria-label="labels[g]"
      :title="labels[g]"
      :class="[
        'cursor-pointer rounded border-0 px-2 py-1 text-right font-inherit text-[10px] font-bold uppercase tracking-[0.06em] transition-colors',
        card.review_grade === g
          ? 'is-active bg-paper shadow-[0_1px_2px_rgba(15,23,42,0.06)]'
          : 'bg-transparent opacity-70 hover:opacity-100',
      ]"
      :style="{ color: gradeColor[g] }"
      :data-test="`grade-pill-${g.toLowerCase()}`"
      @click.stop="pick(g)"
    >{{ labels[g] }}</button>
  </div>
</template>
