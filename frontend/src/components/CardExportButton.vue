<script setup lang="ts">
import { useCardExport } from '../composables/useCardExport';
import type { Card } from '../types/entity';

const props = defineProps<{ card: Card }>();
const { exporting, error, exportCard } = useCardExport();
</script>

<template>
  <div class="flex items-center gap-2">
    <span
      v-if="error"
      data-test="card-export-error"
      class="text-xs text-destructive-fg"
    >{{ error }}</span>
    <button
      type="button"
      data-test="card-action-export"
      class="rounded border border-border px-3 py-1 text-sm text-fg hover:bg-muted disabled:opacity-50"
      :disabled="exporting"
      :title="exporting ? 'Exporting…' : 'Export this card as a file'"
      @click="exportCard(props.card.id)"
    >{{ exporting ? 'Exporting…' : 'Export' }}</button>
  </div>
</template>
