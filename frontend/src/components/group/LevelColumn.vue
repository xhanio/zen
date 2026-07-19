<script setup lang="ts">
import { computed } from 'vue';
import CardItem from '../CardItem.vue';
import type { Card, LevelEntry } from '../../types/entity';
import { levelColor } from '../../utils/levelPalette';

const props = defineProps<{
  label: string;
  cards: Card[];
  isMisc: boolean;
  // The group's level catalog, so the header color scales this weight across
  // the group's full hot→cold range (not by its absolute value).
  catalog?: LevelEntry[];
  // The catalog entry's weight (drives the header color). Same-weight
  // entries render side by side and distinguish by entryId, not weight.
  weight?: number;
  // Entry id for the column's catalog entry. Same-weight entries have
  // distinct ids, so we pass this on drop instead of weight — otherwise
  // dropping on the second same-weight column always resolves to the
  // first entry via the weight lookup in the parent.
  entryId?: string;
}>();

const color = computed(() =>
  props.isMisc ? levelColor([], null) : levelColor(props.catalog ?? [], props.weight ?? null),
);

const emit = defineEmits<{
  (e: 'card-dropped', cardId: string, target: { entryId?: string; clearLevel: boolean }): void;
  (e: 'delete', id: string): void;
}>();

function onDrop(e: DragEvent) {
  e.preventDefault();
  const cardId = e.dataTransfer?.getData('text/zen-card-id');
  if (!cardId) return;
  const target = props.isMisc
    ? { clearLevel: true }
    : { entryId: props.entryId, clearLevel: false };
  emit('card-dropped', cardId, target);
}

function onDragOver(e: DragEvent) {
  e.preventDefault();
}
</script>

<template>
  <div
    class="flex-1 min-w-[140px] border-r border-border last:border-r-0"
    @dragover="onDragOver"
    @drop="onDrop"
  >
    <h3
      class="text-center text-xs font-semibold uppercase tracking-wide py-2 border-b"
      :style="{ color: color.fg, borderColor: color.border }"
    >
      {{ label }}
    </h3>
    <div
      data-test="column-dropzone"
      class="space-y-1 p-1.5 min-h-[80px]"
    >
      <CardItem
        v-for="c in cards"
        :key="c.id"
        :card="c"
        :hide-pill="true"
        @delete="(id) => emit('delete', id)"
      />
    </div>
  </div>
</template>
