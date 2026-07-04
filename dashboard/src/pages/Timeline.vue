<script setup lang="ts">
import EventTimeline from '../components/EventTimeline.vue'
import MetricCard from '../components/MetricCard.vue'
import { runtimeStore } from '../stores/runtime'
import { t } from '../stores/i18n'
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <span class="eyebrow">Observable Runtime</span>
        <h1>{{ t.timeline.title }}</h1>
        <p>{{ t.timeline.desc }}</p>
      </div>
    </div>
    <div class="metrics-grid compact-metrics">
      <MetricCard :label="t.timeline.kernelMode" :value="runtimeStore.kernelStatus.mode || 'degraded-proxy'" />
      <MetricCard :label="t.timeline.kernelProbe" :value="runtimeStore.kernelStatus.probe || 'syscall-gateway-proxy'" />
      <MetricCard :label="t.timeline.kernelEvents" :value="runtimeStore.kernelEvents.length" />
      <MetricCard :label="t.timeline.btf" :value="runtimeStore.kernelStatus.btf_available ? 'yes' : 'no'" />
    </div>
    <div v-if="runtimeStore.kernelStatus.reason" class="recovery-note">
      {{ runtimeStore.kernelStatus.reason }}
    </div>
    <EventTimeline :events="runtimeStore.events" />
  </section>
</template>
