<script setup lang="ts">
import { computed } from 'vue'
import MetricCard from '../components/MetricCard.vue'
import { runtimeStore } from '../stores/runtime'

const e1MaxTime = computed(() => {
  const values = runtimeStore.experimentResults.e1_scheduler.map((item) => item.total_time_ms)
  return Math.max(1, ...values)
})

function barWidth(value: number): string {
  return `${Math.max(6, (value / e1MaxTime.value) * 100)}%`
}
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <h1>Experiments</h1>
        <p>E1 scheduler, E2 fault isolation, and E3 context sharing outputs.</p>
      </div>
    </div>

    <section class="section-block">
      <div class="section-title">
        <h2>E1 Scheduler</h2>
        <span class="subtle-text">{{ runtimeStore.experimentResults.e1_scheduler.length }} policies</span>
      </div>
      <div class="bar-list">
        <div v-for="item in runtimeStore.experimentResults.e1_scheduler" :key="item.policy" class="bar-row">
          <div class="bar-label">{{ item.policy }}</div>
          <div class="bar-track">
            <div class="bar-fill" :style="{ width: barWidth(item.total_time_ms) }"></div>
          </div>
          <div class="bar-value">{{ item.total_time_ms }} ms</div>
        </div>
      </div>
    </section>

    <section class="section-block">
      <div class="section-title">
        <h2>E2 Fault Isolation</h2>
        <span class="subtle-text">tool timeout / pids / rollback model</span>
      </div>
      <div class="inline-table">
        <table>
          <thead>
            <tr>
              <th>Mode</th>
              <th>Affected</th>
              <th>Recovery</th>
              <th>Task</th>
              <th>Rollback</th>
              <th>Faults</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in runtimeStore.experimentResults.e2_fault" :key="item.mode">
              <td>{{ item.mode }}</td>
              <td>{{ item.affected_agents }}</td>
              <td>{{ item.recovery_time_ms }} ms</td>
              <td>{{ item.task_success ? 'success' : 'failed' }}</td>
              <td>{{ item.rollback_success ? 'ok' : 'none' }}</td>
              <td>{{ item.fault_count }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="section-block">
      <div class="section-title">
        <h2>E3 Context Sharing</h2>
        <span class="subtle-text">{{ runtimeStore.experimentResults.e3_context.mode || 'pending' }}</span>
      </div>
      <div class="metrics-grid">
        <MetricCard label="Full Copy Tokens" :value="runtimeStore.experimentResults.e3_context.total_prompt_tokens" />
        <MetricCard label="Unique Tokens" :value="runtimeStore.experimentResults.e3_context.unique_page_tokens" />
        <MetricCard label="Saved Tokens" :value="runtimeStore.experimentResults.e3_context.saved_tokens" />
        <MetricCard label="Saved Bytes" :value="runtimeStore.experimentResults.e3_context.saved_bytes" />
      </div>
    </section>
  </section>
</template>
