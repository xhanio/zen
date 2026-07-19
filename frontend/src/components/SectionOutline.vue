<script setup lang="ts">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import { useCardsStore } from '../stores/cards';
import { useGroupsStore } from '../stores/groups';
import { colorForCard } from '../utils/levelPalette';
import type { Card } from '../types/entity';

// Full table of contents for a container card. Reads the parent's live
// children out of the cards store (they're in the same group, so
// byGroup already holds them) and renders each as a level-color dot +
// title. Purely presentational — no fetches of its own.

const props = defineProps<{ parent: Card }>();

const cardsStore = useCardsStore();
const { byGroup } = storeToRefs(cardsStore);
const groupsStore = useGroupsStore();
const catalog = computed(() => groupsStore.get(props.parent.group_id)?.level_catalog ?? []);

const children = computed(() => {
  const siblings = byGroup.value[props.parent.group_id] ?? [];
  return siblings
    .filter((c) => c.parent_card_id === props.parent.id && !c.deleted_at)
    .sort((a, b) => a.position - b.position);
});

</script>

<template>
  <ul v-if="children.length > 0" class="mt-1 space-y-0.5">
    <li
      v-for="c in children"
      :key="c.id"
      class="flex items-center gap-1.5 text-xs leading-snug text-muted-fg"
    >
      <span
        class="h-1.5 w-1.5 shrink-0 rounded-full"
        :style="{ backgroundColor: colorForCard(catalog, c).fg }"
        aria-hidden="true"
      ></span>
      <span class="truncate">{{ c.title }}</span>
    </li>
  </ul>
</template>
