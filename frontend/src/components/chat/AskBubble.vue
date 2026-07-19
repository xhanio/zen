<script setup lang="ts">
import { computed } from 'vue';
import { useChatSidebar } from '../../composables/useChatSidebar';

const props = defineProps<{
  rect: DOMRect | null;
  selectionText: string;
  anchorKind: 'card';
  anchorId: string;
}>();

const emit = defineEmits<{ (e: 'opened'): void }>();

const sidebar = useChatSidebar();

const style = computed(() => {
  if (!props.rect) return { display: 'none' };
  const top = Math.max(8, props.rect.top + window.scrollY - 36);
  const left = Math.min(window.innerWidth - 80, props.rect.right + window.scrollX + 4);
  return {
    position: 'absolute' as const,
    top: `${top}px`,
    left: `${left}px`,
    zIndex: 40,
  };
});

async function ask() {
  await sidebar.openFor(props.anchorKind, props.anchorId, props.selectionText);
  emit('opened');
}
</script>

<template>
  <button
    v-if="rect"
    type="button"
    :style="style"
    class="rounded-full bg-accent-fg px-3 py-1 text-xs font-medium text-surface shadow hover:bg-accent-fg"
    @mousedown.prevent
    @click="ask"
  >
    Ask
  </button>
</template>
