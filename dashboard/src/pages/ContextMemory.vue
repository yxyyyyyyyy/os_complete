<script setup lang="ts">
import { computed } from 'vue'
import MetricCard from '../components/MetricCard.vue'
import { runtimeStore } from '../stores/runtime'
import { t } from '../stores/i18n'

const ipcRows = computed(() => {
  return Object.entries(runtimeStore.ipcTopics).flatMap(([topic, messages]) =>
    messages.map((message) => ({ ...message, topic }))
  )
})
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <span class="eyebrow">Context Virtual Memory</span>
        <h1>{{ t.context.title }}</h1>
        <p>{{ t.context.desc }}</p>
      </div>
    </div>
    <div class="metrics-grid">
      <MetricCard :label="t.context.totalPages" :value="runtimeStore.contextStats.total_pages" />
      <MetricCard :label="t.context.sharedPages" :value="runtimeStore.contextStats.shared_pages" />
      <MetricCard :label="t.context.savedBytes" :value="runtimeStore.contextStats.saved_bytes" />
      <MetricCard :label="t.context.savedTokens" :value="runtimeStore.contextStats.saved_tokens" />
      <MetricCard :label="t.context.ipcMessages" :value="runtimeStore.ipcMetrics.total_messages" />
      <MetricCard :label="t.context.ipcDepth" :value="runtimeStore.ipcMetrics.topic_depth" />
      <MetricCard :label="t.context.avoidedCopy" :value="runtimeStore.ipcMetrics.avoided_copy_bytes" />
    </div>
    <section class="section-block">
      <div class="section-title">
        <h2>Context Pages</h2>
        <span class="subtle-text">CVM</span>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>{{ t.context.page }}</th>
              <th>{{ t.context.kind }}</th>
              <th>{{ t.context.bytes }}</th>
              <th>{{ t.context.tokens }}</th>
              <th>{{ t.context.refs }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="page in runtimeStore.contextPages" :key="page.id">
              <td class="mono-cell">{{ page.id.slice(0, 16) }}</td>
              <td>{{ page.kind }}</td>
              <td>{{ page.bytes }}</td>
              <td>{{ page.token_count }}</td>
              <td>{{ page.ref_count }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="section-block">
      <div class="section-title">
        <h2>{{ t.context.ipcTopics }}</h2>
        <span class="subtle-text">Page Reference IPC</span>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Topic</th>
              <th>{{ t.context.publisher }}</th>
              <th>{{ t.context.page }}</th>
              <th>{{ t.context.bytes }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="message in ipcRows" :key="message.id">
              <td>{{ message.topic }}</td>
              <td>{{ message.publisher }}</td>
              <td class="mono-cell">{{ message.page_id.slice(0, 16) }}</td>
              <td>{{ message.size_bytes }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </section>
</template>
