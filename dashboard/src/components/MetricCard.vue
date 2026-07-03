<script setup lang="ts">
defineProps<{ label: string; value: string | number }>()
</script>

<template>
  <section class="metric-card">
    <span>{{ label }}</span>
    <strong>{{ value }}</strong>
  </section>
</template>
