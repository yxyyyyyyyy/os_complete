<script setup lang="ts">
import EventTimeline from '../components/EventTimeline.vue'
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
    <EventTimeline :events="runtimeStore.events" />
  </section>
</template>
