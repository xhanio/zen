<script setup lang="ts">
import { computed } from 'vue';
import type { Card, LevelEntry } from '../../types/entity';
import { weightForCard } from '../../utils/levelCatalog';
import { levelColor } from '../../utils/levelPalette';

const props = defineProps<{ catalog: LevelEntry[]; cards: Card[] }>();

// One segment per distinct level weight, ascending (hottest → coldest).
const segments = computed(() => {
  const weights = [...new Set(props.catalog.map((e) => e.weight))].sort((a, b) => a - b);
  return weights.map((w) => ({
    weight: w,
    name: props.catalog.find((e) => e.weight === w)?.name ?? '',
    count: props.cards.filter((c) => weightForCard(props.catalog, c) === w).length,
    color: levelColor(props.catalog, w).fg,
  }));
});
const total = computed(() => segments.value.reduce((a, s) => a + s.count, 0));
</script>

<template>
  <div v-if="catalog.length === 0" data-test="ladder-empty" class="text-xs text-muted-fg">
    No levels defined
  </div>
  <div v-else data-test="level-ladder">
    <div class="flex h-2 overflow-hidden rounded-full bg-muted">
      <div
        v-for="s in segments"
        :key="s.weight"
        :data-test="`ladder-seg-${s.weight}`"
        class="h-full"
        :style="{ width: total > 0 ? `${(s.count / total) * 100}%` : '0%', backgroundColor: s.color }"
      ></div>
    </div>
    <div class="mt-2 flex flex-wrap gap-x-3 gap-y-1">
      <span
        v-for="s in segments"
        :key="s.weight"
        class="inline-flex items-center gap-1.5 text-[11px] text-muted-fg"
      >
        <span class="inline-block h-2 w-2 rounded-full" :style="{ backgroundColor: s.color }"></span>
        {{ s.name }} · {{ s.count }}
      </span>
    </div>
  </div>
</template>
