<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { wrapHighlights, type Highlight } from '../utils/highlightText';

const props = defineProps<{
  source: string;
  highlights?: Highlight[];
}>();

const root = ref<HTMLElement | null>(null);

function paint() {
  const el = root.value;
  if (!el) return;
  el.textContent = props.source;
  if (props.highlights && props.highlights.length > 0) {
    wrapHighlights(el, props.highlights);
  }
}

onMounted(paint);
watch(() => [props.source, props.highlights], paint);
</script>

<template>
  <pre ref="root" class="whitespace-pre-wrap font-mono text-sm bg-paper text-paper-fg"></pre>
</template>
