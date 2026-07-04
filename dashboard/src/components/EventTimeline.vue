<script setup lang="ts">
import type { RuntimeEvent } from '../api/client'
import { t } from '../stores/i18n'

defineProps<{ events: RuntimeEvent[] }>()

function formatTime(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString()
}
</script>

<template>
  <div class="timeline-list">
    <div class="timeline-head">
      <span>{{ t.timeline.time }}</span>
      <span>{{ t.timeline.event }}</span>
      <span>{{ t.timeline.source }}</span>
      <span>{{ t.timeline.owner }}</span>
    </div>
    <article v-for="event in events" :key="`${event.id}-${event.type}`" class="timeline-row">
      <span class="event-time">{{ formatTime(event.timestamp) }}</span>
      <span class="event-type">{{ event.type }}</span>
      <span class="event-source">{{ event.source }}</span>
      <span class="event-agent">{{ event.agent_id || event.task_id || 'runtime' }}</span>
    </article>
  </div>
</template>
