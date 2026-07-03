<script setup lang="ts">
import EventTimeline from '../components/EventTimeline.vue'
import { runtimeStore } from '../stores/runtime'
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <h1>Timeline</h1>
        <p>Runtime and syscall event stream. Kernel lane arrives in V3.</p>
      </div>
    </div>
    <EventTimeline :events="runtimeStore.events" />
  </section>
</template>
