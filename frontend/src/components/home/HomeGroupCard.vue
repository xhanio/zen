<script setup lang="ts">
import { computed } from 'vue';
import type { Card, Group } from '../../types/entity';
import { documentsIn } from '../../utils/documents';
import LevelLadder from './LevelLadder.vue';

const props = defineProps<{ group: Group; cards: Card[] }>();

const docCount = computed(() => documentsIn(props.cards).length);
const topTags = computed(() => {
  const freq = new Map<string, number>();
  for (const c of props.cards) for (const t of c.tags) freq.set(t, (freq.get(t) ?? 0) + 1);
  return [...freq.entries()].sort((a, b) => b[1] - a[1]).slice(0, 6).map(([t]) => t);
});
</script>

<template>
  <RouterLink
    :to="{ name: 'group', params: { groupId: group.id } }"
    data-test="home-group-card"
    class="flex flex-col gap-3 rounded-2xl border border-border bg-surface p-4 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md"
  >
    <div>
      <div class="font-serif text-xl font-semibold leading-tight text-fg">{{ group.name }}</div>
      <div class="text-xs text-muted-fg">
        {{ cards.length }} cards · {{ docCount }} documents · {{ group.level_catalog.length }} levels
      </div>
    </div>
    <LevelLadder :catalog="group.level_catalog" :cards="cards" />
    <div v-if="topTags.length" class="flex flex-wrap gap-1.5">
      <span v-for="t in topTags" :key="t" class="rounded-full bg-muted px-2 py-0.5 text-[11px] text-fg">{{ t }}</span>
    </div>
  </RouterLink>
</template>
