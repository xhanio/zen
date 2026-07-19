<script setup lang="ts">
import {
  AlertDialogRoot,
  AlertDialogPortal,
  AlertDialogOverlay,
  AlertDialogContent,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogCancel,
  AlertDialogAction,
} from 'reka-ui';

withDefaults(
  defineProps<{
    open: boolean;
    title: string;
    description: string;
    confirmLabel?: string;
    cancelLabel?: string;
    destructive?: boolean;
  }>(),
  { confirmLabel: 'Confirm', cancelLabel: 'Cancel', destructive: false },
);

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void;
  (e: 'confirm'): void;
}>();

function onConfirm() {
  emit('confirm');
  emit('update:open', false);
}
</script>

<template>
  <AlertDialogRoot :open="open" @update:open="(v) => emit('update:open', v)">
    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-40 bg-black/30" />
      <AlertDialogContent
        class="fixed left-1/2 top-1/2 z-50 w-full max-w-sm -translate-x-1/2 -translate-y-1/2 rounded-md bg-surface p-5 shadow-lg"
      >
        <AlertDialogTitle class="text-base font-semibold text-fg">
          {{ title }}
        </AlertDialogTitle>
        <AlertDialogDescription class="mt-2 text-sm text-muted-fg">
          {{ description }}
        </AlertDialogDescription>
        <div class="mt-3">
          <slot />
        </div>
        <div class="mt-5 flex justify-end gap-2">
          <AlertDialogCancel
            class="rounded border border-border px-3 py-1.5 text-sm text-fg hover:bg-muted"
          >
            {{ cancelLabel }}
          </AlertDialogCancel>
          <AlertDialogAction
            :class="[
              'rounded px-3 py-1.5 text-sm text-surface',
              destructive ? 'bg-destructive-fg hover:bg-destructive-fg' : 'bg-fg hover:bg-fg',
            ]"
            @click="onConfirm"
          >
            {{ confirmLabel }}
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
