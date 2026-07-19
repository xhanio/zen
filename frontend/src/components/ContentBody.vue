<script setup lang="ts">
import MarkdownBody from './MarkdownBody.vue';
import HtmlBody from './HtmlBody.vue';
import TextBody from './TextBody.vue';
import type { Highlight } from '../utils/highlightText';

defineProps<{
  format: 'markdown' | 'html' | 'text';
  source: string;
  highlights?: Highlight[];
}>();
</script>

<template>
  <HtmlBody v-if="format === 'html'" :source="source" :highlights="highlights" />
  <TextBody v-else-if="format === 'text'" :source="source" :highlights="highlights" />
  <MarkdownBody v-else :source="source" :highlights="highlights" />
</template>
