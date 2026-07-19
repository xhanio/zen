<script setup lang="ts">
import { ref } from 'vue';
import { storeToRefs } from 'pinia';
import { useChatSidebar } from '../../composables/useChatSidebar';
import ResizableSplitter from '../ResizableSplitter.vue';
import { useConversationsStore } from '../../stores/conversations';
import ChatHeader from './ChatHeader.vue';
import ChatThread from './ChatThread.vue';
import ChatComposer from './ChatComposer.vue';

const sidebar = useChatSidebar();
const store = useConversationsStore();
const { activeID } = storeToRefs(store);

const MIN = 320;
const MAX = 720;
const KEY = 'zen:chatWidth';

function clamp(n: number): number {
  return Math.min(MAX, Math.max(MIN, n));
}
const stored = Number(localStorage.getItem(KEY));
const width = ref(clamp(stored || 400));
function setWidth(n: number) {
  width.value = clamp(n);
  localStorage.setItem(KEY, String(width.value));
}

defineExpose({ width, setWidth });
</script>

<template>
  <aside
    v-if="sidebar.open.value"
    data-test="chat-panel"
    class="fixed right-0 top-12 z-30 flex h-[calc(100vh-3rem)] flex-col border-l border-border bg-surface shadow-xl"
    :style="{ width: width + 'px' }"
    aria-label="Chat sidebar"
    role="complementary"
  >
    <ResizableSplitter
      data-test="resize-handle"
      :width="width"
      :min="MIN"
      :max="MAX"
      :default-width="400"
      side="left"
      aria-label="Resize chat sidebar"
      class="absolute left-0 top-0 h-full w-1"
      @update:width="setWidth"
    />
    <ChatHeader @close="sidebar.close" />
    <ChatThread :conversation-id="activeID" />
    <ChatComposer />
  </aside>
</template>
