<script setup lang="ts">
import { statusLabel } from '../stores/i18n'

defineProps<{ value: string }>()
</script>

<template>
  <span class="status-badge" :data-state="value">{{ statusLabel(value) }}</span>
</template>
