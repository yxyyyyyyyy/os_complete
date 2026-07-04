<script setup lang="ts">
import MetricCard from '../components/MetricCard.vue'
import { runtimeStore } from '../stores/runtime'
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <h1>Context Memory</h1>
        <p>CVM pages, reference counts, and deduplication savings.</p>
      </div>
    </div>
    <div class="metrics-grid">
      <MetricCard label="Total Pages" :value="runtimeStore.contextStats.total_pages" />
      <MetricCard label="Shared Pages" :value="runtimeStore.contextStats.shared_pages" />
      <MetricCard label="Saved Bytes" :value="runtimeStore.contextStats.saved_bytes" />
      <MetricCard label="Saved Tokens" :value="runtimeStore.contextStats.saved_tokens" />
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Page</th>
            <th>Kind</th>
            <th>Bytes</th>
            <th>Tokens</th>
            <th>Refs</th>
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
