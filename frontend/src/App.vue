<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import TopBar from './components/layout/TopBar.vue';
import LeftRail from './components/layout/LeftRail.vue';
import MainPanel from './components/layout/MainPanel.vue';
import GroupNav from './components/GroupNav.vue';
import TagCloud from './components/TagCloud.vue';
import ChatSidebar from './components/chat/ChatSidebar.vue';
import { useGroupsStore } from './stores/groups';

const groups = useGroupsStore();
const route = useRoute();

// "Reading mode": when the user opens a card, the browsing chrome (left
// rail with groups + level filters + tag cloud) gets out of the way.
// Reading is the app's single job here — everything else is preparation
// for it. The rail is available again on group / trash / search / chat.
const readingMode = computed(() => route.name === 'card');

onMounted(() => {
  groups.load();
});
</script>

<template>
  <div class="flex h-full flex-col">
    <TopBar />
    <div class="flex flex-1 overflow-hidden">
      <LeftRail v-if="!readingMode">
        <template #tree>
          <div v-if="groups.loading" class="px-2 py-1 text-xs text-muted-fg">Loading…</div>
          <div v-else-if="groups.error" class="px-2 py-1 text-xs text-destructive-fg">{{ groups.error.message }}</div>
          <GroupNav v-else />
        </template>
        <template #tags>
          <template v-if="route.params.groupId">
            <div class="mb-2 text-[10px] font-medium uppercase tracking-wide text-muted-fg">Tags</div>
            <TagCloud />
          </template>
        </template>
      </LeftRail>
      <MainPanel>
        <RouterView />
      </MainPanel>
    </div>
    <ChatSidebar />
  </div>
</template>
