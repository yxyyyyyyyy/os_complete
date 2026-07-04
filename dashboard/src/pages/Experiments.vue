<script setup lang="ts">
import { computed } from 'vue'
import MetricCard from '../components/MetricCard.vue'
import { runtimeStore } from '../stores/runtime'
import { t } from '../stores/i18n'

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
        <span class="eyebrow">Benchmark Evidence</span>
        <h1>{{ t.experiments.title }}</h1>
        <p>{{ t.experiments.desc }}</p>
      </div>
    </div>

    <section class="section-block">
      <div class="section-title">
        <h2>{{ t.experiments.e1 }}</h2>
        <span class="subtle-text">{{ runtimeStore.experimentResults.e1_scheduler.length }} {{ t.experiments.policies }}</span>
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
        <h2>{{ t.experiments.e2 }}</h2>
        <span class="subtle-text">{{ t.experiments.e2Hint }}</span>
      </div>
      <div class="inline-table">
        <table>
          <thead>
            <tr>
              <th>{{ t.common.mode }}</th>
              <th>{{ t.experiments.affected }}</th>
              <th>{{ t.experiments.recovery }}</th>
              <th>{{ t.common.task }}</th>
              <th>{{ t.experiments.rollback }}</th>
              <th>{{ t.experiments.faults }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in runtimeStore.experimentResults.e2_fault" :key="item.mode">
              <td>{{ item.mode }}</td>
              <td>{{ item.affected_agents }}</td>
              <td>{{ item.recovery_time_ms }} ms</td>
              <td>{{ item.task_success ? t.common.success : t.common.failed }}</td>
              <td>{{ item.rollback_success ? 'ok' : 'none' }}</td>
              <td>{{ item.fault_count }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="section-block">
      <div class="section-title">
        <h2>{{ t.experiments.e3 }}</h2>
        <span class="subtle-text">{{ runtimeStore.experimentResults.e3_context.mode || t.common.pending }}</span>
      </div>
      <div class="metrics-grid">
        <MetricCard :label="t.experiments.fullCopyTokens" :value="runtimeStore.experimentResults.e3_context.total_prompt_tokens" />
        <MetricCard :label="t.experiments.uniqueTokens" :value="runtimeStore.experimentResults.e3_context.unique_page_tokens" />
        <MetricCard :label="t.context.savedTokens" :value="runtimeStore.experimentResults.e3_context.saved_tokens" />
        <MetricCard :label="t.context.ipcAvoidedBytes" :value="runtimeStore.experimentResults.e3_context.ipc_avoided_copy_bytes" />
      </div>
    </section>
  </section>
</template>
