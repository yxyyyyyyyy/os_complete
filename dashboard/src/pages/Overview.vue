<script setup lang="ts">
import { computed } from 'vue'
import MetricCard from '../components/MetricCard.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { runtimeStore, selectedTask, startDemo } from '../stores/runtime'
import { t } from '../stores/i18n'

const task = selectedTask
const recentDecisions = computed(() => runtimeStore.schedulerDecisions.slice(-6).reverse())
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <span class="eyebrow">AI Native OS</span>
        <h1>{{ t.overview.title }}</h1>
        <p>{{ t.overview.desc }}</p>
      </div>
      <button class="primary-button" :disabled="runtimeStore.loading" @click="startDemo">
        {{ runtimeStore.loading ? t.common.running : t.common.runDemo }}
      </button>
    </div>

    <div v-if="runtimeStore.error" class="error-line">{{ runtimeStore.error }}</div>

    <div class="metrics-grid">
      <MetricCard :label="t.overview.tasks" :value="runtimeStore.tasks.length" />
      <MetricCard :label="t.overview.events" :value="runtimeStore.events.length" />
      <MetricCard :label="t.overview.sse" :value="runtimeStore.connected ? 'online' : 'offline'" />
      <MetricCard :label="t.overview.runtimeMode" value="mock" />
    </div>

    <section v-if="task()" class="section-block">
      <div class="section-title">
        <h2>{{ t.overview.dag }} · {{ task()?.task_id }}</h2>
        <StatusBadge :value="task()?.status || 'unknown'" />
      </div>
      <div class="dag-grid">
        <article v-for="node in task()?.dag" :key="node.id" class="dag-node">
          <strong>{{ node.role }}</strong>
          <span>{{ node.dependencies?.length ? node.dependencies.join(', ') : t.overview.root }}</span>
        </article>
      </div>
    </section>

    <section class="section-block">
      <div class="section-title">
        <h2>{{ t.overview.scheduler }}</h2>
        <span class="subtle-text">{{ runtimeStore.schedulerDecisions.length }} {{ t.common.records }}</span>
      </div>
      <div class="inline-table">
        <table>
          <thead>
            <tr>
              <th>{{ t.common.policy }}</th>
              <th>{{ t.common.selected }}</th>
              <th>{{ t.common.reason }}</th>
              <th>{{ t.common.candidates }}</th>
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
        <div v-if="recentDecisions.length === 0" class="empty-state">{{ t.common.emptyScheduler }}</div>
      </div>
    </section>
  </section>
</template>
