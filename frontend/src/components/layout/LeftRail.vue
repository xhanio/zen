<script setup lang="ts">
import { storeToRefs } from 'pinia';
import {
  useRailPrefsStore,
  RAIL_DEFAULT_WIDTH,
  RAIL_MIN_WIDTH,
  RAIL_MAX_WIDTH,
} from '../../stores/railPrefs';
import ResizableSplitter from '../ResizableSplitter.vue';

const rail = useRailPrefsStore();
const { width } = storeToRefs(rail);
</script>

<template>
  <aside
    class="relative flex h-full shrink-0 flex-col border-r border-border bg-nav"
    :style="{ width: `${width}px` }"
  >
    <nav class="space-y-0.5 px-2 py-2">
      <RouterLink
        to="/"
        class="flex items-center gap-2 rounded px-2 py-1 text-xs font-medium text-fg hover:bg-muted"
        active-class="bg-muted"
      >
        <span aria-hidden="true" class="w-4 text-center text-muted-fg">⌂</span>Home
      </RouterLink>
      <RouterLink
        to="/chat"
        class="flex items-center gap-2 rounded px-2 py-1 text-xs font-medium text-fg hover:bg-muted"
        active-class="bg-muted"
      >
        <span aria-hidden="true" class="w-4 text-center text-muted-fg">◇</span>Conversations
      </RouterLink>
      <RouterLink
        to="/trash"
        class="flex items-center gap-2 rounded px-2 py-1 text-xs font-medium text-fg hover:bg-muted"
        active-class="bg-muted"
      >
        <span aria-hidden="true" class="w-4 text-center text-muted-fg">⌫</span>Trash
      </RouterLink>
    </nav>
    <div class="flex items-center gap-2 px-3 pb-1 pt-1">
      <span class="text-[10px] font-semibold uppercase tracking-[0.16em] text-muted-fg">Groups</span>
      <span class="h-px flex-1 bg-border"></span>
    </div>
    <div class="flex-1 overflow-y-auto px-2 pb-2">
      <slot name="tree" />
    </div>
    <div class="border-t border-border p-2">
      <slot name="tags" />
    </div>

    <!-- Resize handle straddling the right border: drag, double-click to reset,
         ←/→ to nudge. Behavior lives in the shared ResizableSplitter. -->
    <ResizableSplitter
      data-test="rail-resize"
      :width="width"
      :min="RAIL_MIN_WIDTH"
      :max="RAIL_MAX_WIDTH"
      :default-width="RAIL_DEFAULT_WIDTH"
      side="right"
      aria-label="Resize sidebar"
      class="absolute top-0 -right-0.5 z-10 h-full w-1.5"
      @update:width="rail.setWidth"
    />
  </aside>
</template>
