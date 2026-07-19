<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { storeToRefs } from 'pinia';
import type { Card } from '../types/entity';
import { useGroupsStore } from '../stores/groups';
import { useCardsStore } from '../stores/cards';
import { documentsIn, childrenOf } from '../utils/documents';
import HomeGroupCard from '../components/home/HomeGroupCard.vue';
import RecentDocumentCard from '../components/home/RecentDocumentCard.vue';

const groupsStore = useGroupsStore();
const cardsStore = useCardsStore();
const { groups } = storeToRefs(groupsStore);
const { byGroup } = storeToRefs(cardsStore);

const loading = ref(true);

onMounted(async () => {
  loading.value = true;
  if (groups.value.length === 0) await groupsStore.load();
  await Promise.all(groups.value.map((g) => cardsStore.loadByGroup(g.id)));
  loading.value = false;
});

function liveCards(groupId: string): Card[] {
  return (byGroup.value[groupId] ?? []).filter((c) => !c.deleted_at);
}

const greeting = computed(() => {
  const h = new Date().getHours();
  return h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening';
});

const totalCards = computed(() =>
  groups.value.reduce((a, g) => a + liveCards(g.id).length, 0),
);
const totalDocs = computed(() =>
  groups.value.reduce((a, g) => a + documentsIn(byGroup.value[g.id] ?? []).length, 0),
);

interface RecentDoc { doc: Card; groupName: string; sections: number }
const recentDocs = computed<RecentDoc[]>(() => {
  const all: RecentDoc[] = [];
  for (const g of groups.value) {
    const groupCards = byGroup.value[g.id] ?? [];
    for (const doc of documentsIn(groupCards)) {
      all.push({ doc, groupName: g.name, sections: childrenOf(groupCards, doc.id).length });
    }
  }
  return all
    .sort((a, b) => (a.doc.updated_at < b.doc.updated_at ? 1 : -1))
    .slice(0, 8);
});
</script>

<template>
  <div v-if="loading" class="text-sm text-muted-fg">Loading…</div>
  <div v-else-if="groups.length === 0" data-test="home-empty" class="text-sm text-muted-fg">
    No groups yet — create one from the rail on the left.
  </div>
  <div v-else>
    <div class="text-[11px] font-semibold uppercase tracking-widest text-muted-fg">Home</div>
    <h1 class="mt-1 font-serif text-3xl font-medium text-fg">{{ greeting }}</h1>

    <div data-test="home-stats" class="mt-3 flex gap-6">
      <div><span class="block font-serif text-2xl text-fg">{{ groups.length }}</span><span class="text-xs text-muted-fg">groups</span></div>
      <div><span class="block font-serif text-2xl text-fg">{{ totalCards }}</span><span class="text-xs text-muted-fg">cards</span></div>
      <div><span class="block font-serif text-2xl text-fg">{{ totalDocs }}</span><span class="text-xs text-muted-fg">documents</span></div>
    </div>

    <h2 class="mb-3 mt-8 text-sm font-semibold text-fg">Your groups</h2>
    <div
      class="gap-4"
      style="display:grid;grid-template-columns:repeat(auto-fill,minmax(320px,1fr))"
    >
      <HomeGroupCard v-for="g in groups" :key="g.id" :group="g" :cards="liveCards(g.id)" />
    </div>

    <template v-if="recentDocs.length">
      <h2 class="mb-3 mt-8 text-sm font-semibold text-fg">Recently updated documents</h2>
      <div
        class="gap-3"
        style="display:grid;grid-template-columns:repeat(auto-fill,minmax(250px,1fr))"
      >
        <RecentDocumentCard
          v-for="r in recentDocs"
          :key="r.doc.id"
          :doc="r.doc"
          :group-name="r.groupName"
          :sections="r.sections"
        />
      </div>
    </template>
  </div>
</template>
