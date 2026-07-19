<script setup lang="ts">
import { computed, onMounted, watch } from 'vue';
import { storeToRefs } from 'pinia';
import ContentBody from './ContentBody.vue';
import ContainerBody from './ContainerBody.vue';
import { useCardsStore } from '../stores/cards';
import { useTilePrefsStore } from '../stores/tilePrefs';
import { isContainer } from '../utils/isContainer';
import type { Highlight } from '../utils/highlightText';
import type { Card } from '../types/entity';

const props = defineProps<{ card: Card; highlights?: Highlight[] }>();

// Forwarded from ContainerBody up to CardView, which owns the grade gutter.
// Nested containers emit too; those inner emits are simply not listened to.
const emit = defineEmits<{ (e: 'anchors', els: HTMLElement[]): void }>();

const cardsStore = useCardsStore();
const { byChildren } = storeToRefs(cardsStore);
const { showTrashedSections } = storeToRefs(useTilePrefsStore());

const liveChildrenCount = computed(() =>
  (byChildren.value[props.card.id] ?? []).filter((c) => !c.deleted_at).length,
);

const container = computed(() => isContainer(props.card, liveChildrenCount.value));

// Empty-content cards are potential containers. Fetch their children
// on first sight so the recursive render can discover nested containers
// without the outer node having to know their depth. If the card ends
// up truly being an empty leaf (server returns zero children), this
// still renders as "(empty)" — the fetch just confirms it once.
async function maybeLoad() {
  if ((props.card.content ?? '').trim() !== '') return;
  if (byChildren.value[props.card.id] !== undefined) return;
  await cardsStore.loadChildren(props.card.id, showTrashedSections.value);
}

onMounted(maybeLoad);
watch(() => props.card.id, maybeLoad);
</script>

<template>
  <ContainerBody v-if="container" :parent="card" @anchors="(els) => emit('anchors', els)" />
  <ContentBody
    v-else-if="card.content"
    :source="card.content"
    :format="card.format ?? 'markdown'"
    :highlights="highlights"
  />
  <span v-else class="italic text-muted-fg">(empty)</span>
</template>
