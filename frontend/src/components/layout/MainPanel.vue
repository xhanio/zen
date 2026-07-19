<script setup lang="ts"></script>

<template>
  <!-- scrollbar-gutter reserves scrollbar width even when content
       doesn't overflow, so pages with and without a scrollbar render
       at the same content width and the mx-auto centered column
       doesn't shift when navigating between short and tall views. -->
  <main class="flex-1 overflow-auto bg-surface p-4 [scrollbar-gutter:stable]">
    <slot />
  </main>
</template>
