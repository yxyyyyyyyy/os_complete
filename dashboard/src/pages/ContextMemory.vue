<script setup lang="ts">
import MetricCard from '../components/MetricCard.vue'
import { runtimeStore } from '../stores/runtime'
import { t } from '../stores/i18n'
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
</template>
