<script setup lang="ts">
import { ref } from 'vue';

// A controlled drag handle. The parent owns the width value and its
// persistence; this component owns only the interaction and emits the new,
// already-clamped width. `side` is the edge the handle sits on: 'right' means
// dragging right widens (left-anchored panels); 'left' means dragging left
// widens (right-anchored panels). Keyboard is uniform: Right widens.
const props = withDefaults(
  defineProps<{
    width: number;
    min: number;
    max: number;
    defaultWidth?: number;
    side?: 'left' | 'right';
    nudge?: number;
    ariaLabel?: string;
  }>(),
  { side: 'right', nudge: 16, ariaLabel: 'Resize panel' },
);

const emit = defineEmits<{ 'update:width': [number] }>();

function clamp(n: number): number {
  return Math.min(props.max, Math.max(props.min, Math.round(n)));
}

const dragging = ref(false);
let startX = 0;
let startW = 0;

function onPointerDown(e: PointerEvent) {
  dragging.value = true;
  startX = e.clientX;
  startW = props.width;
  (e.target as HTMLElement).setPointerCapture?.(e.pointerId);
  e.preventDefault();
}
function onPointerMove(e: PointerEvent) {
  if (!dragging.value) return;
  const delta = props.side === 'right' ? e.clientX - startX : startX - e.clientX;
  emit('update:width', clamp(startW + delta));
}
function onPointerUp(e: PointerEvent) {
  if (!dragging.value) return;
  dragging.value = false;
  (e.target as HTMLElement).releasePointerCapture?.(e.pointerId);
}
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowRight') {
    emit('update:width', clamp(props.width + props.nudge));
    e.preventDefault();
  } else if (e.key === 'ArrowLeft') {
    emit('update:width', clamp(props.width - props.nudge));
    e.preventDefault();
  } else if (e.key === 'Home' && props.defaultWidth !== undefined) {
    emit('update:width', clamp(props.defaultWidth));
    e.preventDefault();
  }
}
function onDblclick() {
  if (props.defaultWidth !== undefined) emit('update:width', clamp(props.defaultWidth));
}
</script>

<template>
  <div
    role="separator"
    aria-orientation="vertical"
    :aria-label="ariaLabel"
    :aria-valuenow="width"
    :aria-valuemin="min"
    :aria-valuemax="max"
    tabindex="0"
    class="cursor-col-resize transition-colors hover:bg-accent-border focus-visible:bg-accent-border focus-visible:outline-none"
    :class="dragging ? 'bg-accent-border' : ''"
    @pointerdown="onPointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @dblclick="onDblclick"
    @keydown="onKeydown"
  ></div>
</template>
