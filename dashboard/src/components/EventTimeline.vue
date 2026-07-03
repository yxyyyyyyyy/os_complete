<script setup lang="ts">
import type { RuntimeEvent } from '../api/client'

defineProps<{ events: RuntimeEvent[] }>()
</script>

<template>
  <div class="timeline-list">
    <article v-for="event in events" :key="`${event.id}-${event.type}`" class="timeline-row">
      <span class="event-time">{{ event.timestamp }}</span>
      <span class="event-type">{{ event.type }}</span>
      <span class="event-source">{{ event.source }}</span>
      <span class="event-agent">{{ event.agent_id || event.task_id || 'runtime' }}</span>
    </article>
  </div>
</template>
