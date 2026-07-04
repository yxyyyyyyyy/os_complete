<script setup lang="ts">
import { computed } from 'vue'
import MetricCard from '../components/MetricCard.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { runtimeStore, selectedTask, startDemo } from '../stores/runtime'

const task = selectedTask
const recentDecisions = computed(() => runtimeStore.schedulerDecisions.slice(-6).reverse())
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

    <section class="section-block">
      <div class="section-title">
        <h2>Scheduler Decisions</h2>
        <span class="subtle-text">{{ runtimeStore.schedulerDecisions.length }} records</span>
      </div>
      <div class="inline-table">
        <table>
          <thead>
            <tr>
              <th>Policy</th>
              <th>Selected</th>
              <th>Reason</th>
              <th>Candidates</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="decision in recentDecisions" :key="decision.id">
              <td>{{ decision.policy }}</td>
              <td class="mono-cell">{{ decision.selected_agent }}</td>
              <td>{{ decision.reason }}</td>
              <td class="mono-cell">{{ decision.candidates.length }}</td>
            </tr>
          </tbody>
        </table>
        <div v-if="recentDecisions.length === 0" class="empty-state">No scheduler decisions yet.</div>
      </div>
    </section>
  </section>
</template>
