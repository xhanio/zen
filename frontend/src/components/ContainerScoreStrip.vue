<script setup lang="ts">
import { computed } from 'vue';
import type { Card } from '../types/entity';

// S2 — Ladder-Gradient score strip. Sits above the container reading view.
// The fill IS the level ladder: rose (原则) → amber (决策) → emerald (模式) →
// sky (细节) → violet, pinned to the full 100pt scale via a background-size
// trick so as the score rises the bar "walks up the ladder" that produced it.
//
// Null score (empty container, all trashed, etc.) renders "—" with a fully
// muted track (no fill).

const props = defineProps<{ card: Card }>();

const score = computed(() => props.card.review_score);
const scoreLabel = computed(() =>
  score.value === null ? '—' : score.value.toFixed(1),
);
const pctVar = computed(() => (score.value === null ? '' : String(score.value)));
</script>

<template>
  <div class="flex flex-col gap-2 border-b border-border bg-paper px-4 py-3" data-test="container-score-strip">
    <div class="flex items-baseline justify-between">
      <span class="text-[10px] font-bold uppercase tracking-[0.12em] text-muted-fg">
        Review
      </span>
      <span class="tabular-nums">
        <span
          class="text-[15px] font-semibold -tracking-[0.01em]"
          :class="score === null ? 'text-muted-fg' : 'text-paper-fg'"
        >{{ scoreLabel }}</span>
        <span class="ml-0.5 text-[12px] text-muted-fg">/ 100</span>
      </span>
    </div>
    <div class="relative h-1.5 overflow-hidden rounded-full bg-muted">
      <div
        v-if="score !== null"
        data-test="fill"
        class="absolute inset-y-0 left-0 rounded-full"
        :style="{
          '--pct': pctVar,
          width: 'calc(1% * var(--pct))',
          background: 'linear-gradient(90deg, var(--l-0-fg) 0%, var(--l-1-fg) 25%, var(--l-2-fg) 50%, var(--l-3-fg) 75%, var(--l-4-fg) 100%)',
          backgroundSize: 'calc(100% * 100 / var(--pct)) 100%',
          backgroundRepeat: 'no-repeat',
        }"
      ></div>
    </div>
  </div>
</template>
