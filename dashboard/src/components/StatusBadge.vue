<script setup lang="ts">
defineProps<{ value: string }>()
</script>

<template>
  <span class="status-badge" :data-state="value">{{ value }}</span>
</template>
