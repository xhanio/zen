<script setup lang="ts">
import { ref, onBeforeUnmount } from 'vue';
import type { Card } from '../types/entity';

// Replaces the "↗ open" chip. Only Open moves here today; the menu exists so
// Copy link / Move to Trash can arrive later without relocating the control.
const props = defineProps<{ card: Card }>();
const open = ref(false);
const root = ref<HTMLElement | null>(null);

function close() {
  open.value = false;
  document.removeEventListener('click', onDocClick, true);
}
function toggle() {
  open.value = !open.value;
  if (open.value) document.addEventListener('click', onDocClick, true);
  else document.removeEventListener('click', onDocClick, true);
}
function onDocClick(e: MouseEvent) {
  if (root.value && !root.value.contains(e.target as Node)) close();
}
onBeforeUnmount(() => document.removeEventListener('click', onDocClick, true));
</script>

<template>
  <div ref="root" class="relative">
    <button
      type="button"
      data-test="section-menu-trigger"
      aria-haspopup="menu"
      :aria-expanded="open ? 'true' : 'false'"
      :aria-label="'Actions for ' + props.card.title"
      class="flex h-6 w-6 items-center justify-center cursor-pointer rounded border border-border bg-paper text-lg leading-none text-muted-fg hover:bg-muted"
      @click.stop="toggle"
      @keydown.esc="close"
    >⋯</button>

    <div
      v-if="open"
      data-test="section-menu"
      role="menu"
      class="absolute right-0 top-7 z-20 w-32 rounded-md border border-border bg-paper p-1 shadow-lg"
    >
      <RouterLink
        :to="{ name: 'card', params: { cardId: props.card.id } }"
        role="menuitem"
        data-test="section-menu-open"
        class="block rounded px-2 py-1 text-xs text-fg hover:bg-muted"
        @click="close"
      >↗ Open</RouterLink>
    </div>
  </div>
</template>
