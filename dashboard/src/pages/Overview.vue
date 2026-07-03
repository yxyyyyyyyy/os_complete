<script setup lang="ts">
import MetricCard from '../components/MetricCard.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { runtimeStore, selectedTask, startDemo } from '../stores/runtime'

const task = selectedTask
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <h1>AORT-R Runtime</h1>
        <p>Agent execution, DAG flow, and runtime evidence.</p>
      </div>
      <button class="primary-button" :disabled="runtimeStore.loading" @click="startDemo">
        {{ runtimeStore.loading ? 'Running' : 'Run Demo' }}
      </button>
    </div>

    <div v-if="runtimeStore.error" class="error-line">{{ runtimeStore.error }}</div>

    <div class="metrics-grid">
      <MetricCard label="Tasks" :value="runtimeStore.tasks.length" />
      <MetricCard label="Events" :value="runtimeStore.events.length" />
      <MetricCard label="SSE" :value="runtimeStore.connected ? 'connected' : 'offline'" />
      <MetricCard label="Mode" value="mock" />
    </div>

    <section v-if="task()" class="section-block">
      <div class="section-title">
        <h2>{{ task()?.task_id }}</h2>
        <StatusBadge :value="task()?.status || 'unknown'" />
      </div>
      <div class="dag-grid">
        <article v-for="node in task()?.dag" :key="node.id" class="dag-node">
          <strong>{{ node.role }}</strong>
          <span>{{ node.dependencies?.length ? node.dependencies.join(', ') : 'root' }}</span>
        </article>
      </div>
    </section>
  </section>
</template>
