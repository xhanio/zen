<script setup lang="ts">
import { computed } from 'vue';
import type { Card } from '../../types/entity';
import { relativeTime } from '../../utils/relativeTime';

const props = defineProps<{ doc: Card; groupName: string; sections: number }>();
const preview = computed(() => props.doc.summary?.trim() || props.doc.genesis?.trim() || '');
</script>

<template>
  <RouterLink
    :to="{ name: 'card', params: { cardId: doc.id } }"
    data-test="recent-doc"
    class="flex min-h-[112px] flex-col gap-2 rounded-2xl border border-border bg-surface px-3.5 py-3 shadow-sm transition hover:-translate-y-0.5 hover:shadow"
  >
    <div class="font-serif text-base font-medium leading-tight text-fg">{{ doc.title || 'Untitled' }}</div>
    <p
      v-if="preview"
      class="text-xs leading-snug text-muted-fg"
      style="display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden"
    >{{ preview }}</p>
    <div class="mt-auto flex items-center gap-2 text-[11px] text-muted-fg">
      <span>▤ {{ sections }} sections</span>
      <span>· {{ groupName }}</span>
      <span class="ml-auto">{{ relativeTime(doc.updated_at) }}</span>
    </div>
  </RouterLink>
</template>
