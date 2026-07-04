<script setup lang="ts">
import { computed } from 'vue'
import MetricCard from '../components/MetricCard.vue'
import { runtimeStore } from '../stores/runtime'
import { t } from '../stores/i18n'

const e1MaxTime = computed(() => {
  const values = runtimeStore.experimentResults.e1_scheduler.map((item) => item.total_time_ms)
  return Math.max(1, ...values)
})

const e1RealMaxWall = computed(() => {
  const values = runtimeStore.experimentResults.e1_real_scheduler.map((item) => item.wall_time_ms)
  return Math.max(1, ...values)
})

const realCapsules = computed(() => runtimeStore.agents.filter((agent) => agent.capsule_mode === 'real').length)
const degradedCapsules = computed(() => runtimeStore.agents.filter((agent) => agent.capsule_mode && agent.capsule_mode !== 'real').length)

function barWidth(value: number): string {
  return `${Math.max(6, (value / e1MaxTime.value) * 100)}%`
}

function realBarWidth(value: number): string {
  return `${Math.max(6, (value / e1RealMaxWall.value) * 100)}%`
}

function formatPercent(value: number): string {
  return `${(value * 100).toFixed(1)}%`
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
        <h2>{{ t.common.evidence }}</h2>
        <span class="subtle-text">{{ t.experiments.smoke }}</span>
      </div>
      <div class="metrics-grid compact-metrics">
        <MetricCard :label="t.common.real" :value="realCapsules" />
        <MetricCard :label="t.common.degraded" :value="degradedCapsules" />
        <MetricCard :label="t.experiments.syscalls" :value="runtimeStore.experimentResults.e5_end_to_end.syscalls" />
        <MetricCard :label="t.experiments.finalStatus" :value="runtimeStore.experimentResults.e5_end_to_end.final_success ? t.common.success : t.common.pending" />
      </div>
      <div class="recovery-note">{{ t.experiments.smokeHint }}</div>
    </section>

    <section class="section-block">
      <div class="section-title">
        <h2>{{ t.experiments.e1Real }}</h2>
        <span class="subtle-text">real-runtime</span>
      </div>
      <div class="bar-list">
        <div v-for="item in runtimeStore.experimentResults.e1_real_scheduler" :key="item.policy" class="bar-row">
          <div class="bar-label">{{ item.policy }}</div>
          <div class="bar-track">
            <div class="bar-fill" :style="{ width: realBarWidth(item.wall_time_ms) }"></div>
          </div>
          <div class="bar-value">{{ item.wall_time_ms }} ms</div>
        </div>
      </div>
      <div class="inline-table">
        <table>
          <thead>
            <tr>
              <th>{{ t.common.policy }}</th>
              <th>{{ t.experiments.p95 }}</th>
              <th>{{ t.experiments.throughput }}</th>
              <th>{{ t.experiments.reuseRate }}</th>
              <th>{{ t.context.savedTokens }}</th>
              <th>{{ t.common.evidence }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in runtimeStore.experimentResults.e1_real_scheduler" :key="`${item.policy}-real`">
              <td>{{ item.policy }}</td>
              <td>{{ item.p95_latency_ms }} ms</td>
              <td>{{ item.throughput_tasks_per_sec.toFixed(2) }}</td>
              <td>{{ formatPercent(item.context_reuse_rate) }}</td>
              <td>{{ item.context_saved_tokens }}</td>
              <td>{{ item.evidence_mode }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="section-block">
      <div class="section-title">
        <h2>{{ t.experiments.e2Real }}</h2>
        <span class="subtle-text">supervisor</span>
      </div>
      <div class="inline-table">
        <table>
          <thead>
            <tr>
              <th>{{ t.experiments.faults }}</th>
              <th>{{ t.common.agent }}</th>
              <th>{{ t.experiments.affected }}</th>
              <th>{{ t.experiments.recovery }}</th>
              <th>{{ t.common.task }}</th>
              <th>{{ t.common.evidence }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in runtimeStore.experimentResults.e2_real_fault" :key="item.fault_type">
              <td>{{ item.fault_type }}</td>
              <td>{{ item.failed_agent }}</td>
              <td>{{ item.affected_agents }} / {{ item.total_agents }}</td>
              <td>{{ item.recovery_time_ms }} ms</td>
              <td>{{ item.system_survived && !item.cascade_failure ? t.common.success : t.common.failed }}</td>
              <td>{{ item.evidence_mode }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="section-block">
      <div class="section-title">
        <h2>{{ t.experiments.e3Real }}</h2>
        <span class="subtle-text">CVM</span>
      </div>
      <div class="inline-table">
        <table>
          <thead>
            <tr>
              <th>{{ t.common.mode }}</th>
              <th>{{ t.experiments.baselineTokens }}</th>
              <th>{{ t.experiments.actualTokens }}</th>
              <th>{{ t.context.savedTokens }}</th>
              <th>{{ t.experiments.summaryPages }}</th>
              <th>{{ t.experiments.reuseRate }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in runtimeStore.experimentResults.e3_real_context" :key="item.mode">
              <td>{{ item.mode }}</td>
              <td>{{ item.baseline_tokens }}</td>
              <td>{{ item.actual_materialized_tokens }}</td>
              <td>{{ item.saved_tokens }}</td>
              <td>{{ item.summary_pages }}</td>
              <td>{{ formatPercent(item.reuse_rate) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="section-block">
      <div class="section-title">
        <h2>{{ t.experiments.e4Real }}</h2>
        <span class="subtle-text">page-ref</span>
      </div>
      <div class="inline-table">
        <table>
          <thead>
            <tr>
              <th>{{ t.common.mode }}</th>
              <th>{{ t.context.ipcMessages }}</th>
              <th>{{ t.experiments.baselineTokens }}</th>
              <th>{{ t.experiments.actualTokens }}</th>
              <th>{{ t.experiments.avoidedCopy }}</th>
              <th>{{ t.experiments.p95 }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in runtimeStore.experimentResults.e4_real_ipc" :key="item.mode">
              <td>{{ item.mode }}</td>
              <td>{{ item.message_count }}</td>
              <td>{{ item.payload_bytes_baseline }}</td>
              <td>{{ item.payload_bytes_actual }}</td>
              <td>{{ item.avoided_copy_bytes }}</td>
              <td>{{ item.avg_poll_latency_ms.toFixed(4) }} ms</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="section-block">
      <div class="section-title">
        <h2>{{ t.experiments.e5Real }}</h2>
        <span class="subtle-text">{{ runtimeStore.experimentResults.e5_end_to_end.evidence_mode || t.common.pending }}</span>
      </div>
      <div class="metrics-grid compact-metrics">
        <MetricCard :label="t.experiments.wallTime" :value="`${runtimeStore.experimentResults.e5_end_to_end.wall_time_ms} ms`" />
        <MetricCard :label="t.common.agent" :value="runtimeStore.experimentResults.e5_end_to_end.agents" />
        <MetricCard :label="t.experiments.toolExec" :value="runtimeStore.experimentResults.e5_end_to_end.tool_exec" />
        <MetricCard :label="t.experiments.recovery" :value="runtimeStore.experimentResults.e5_end_to_end.fault_recovered ? 'ok' : t.common.pending" />
      </div>
    </section>

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
