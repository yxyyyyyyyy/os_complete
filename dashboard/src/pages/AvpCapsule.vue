<script setup lang="ts">
import StatusBadge from '../components/StatusBadge.vue'
import { runAgentAction, runtimeStore } from '../stores/runtime'
import type { Agent } from '../api/client'
import { t } from '../stores/i18n'

function agentID(agent: Agent): string {
  return agent.agent_id ?? agent.id ?? ''
}
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <span class="eyebrow">Agent Virtual Process</span>
        <h1>{{ t.avp.title }}</h1>
        <p>{{ t.avp.desc }}</p>
      </div>
    </div>

    <div v-if="runtimeStore.error" class="error-line">{{ runtimeStore.error }}</div>

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>{{ t.common.agent }}</th>
            <th>{{ t.common.role }}</th>
            <th>{{ t.common.state }}</th>
            <th>{{ t.avp.pid }}</th>
            <th>{{ t.avp.capsule }}</th>
            <th>{{ t.avp.memory }}</th>
            <th>{{ t.avp.pids }}</th>
            <th>{{ t.avp.retry }}</th>
            <th>{{ t.common.actions }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="agent in runtimeStore.agents" :key="agentID(agent)">
            <td class="mono-cell">{{ agentID(agent) }}</td>
            <td>{{ agent.role }}</td>
            <td><StatusBadge :value="agent.state" /></td>
            <td>{{ agent.pid || '-' }}</td>
            <td class="mono-cell">{{ agent.capsule_mode || '-' }}</td>
            <td>{{ agent.memory_current ?? 0 }}</td>
            <td>{{ agent.pids_current ?? 0 }}</td>
            <td>{{ agent.retry_count ?? 0 }}</td>
            <td class="action-row">
              <button class="icon-button" :title="t.avp.freeze" @click="runAgentAction(agentID(agent), 'freeze')">F</button>
              <button class="icon-button" :title="t.avp.unfreeze" @click="runAgentAction(agentID(agent), 'unfreeze')">U</button>
              <button class="icon-button danger" :title="t.avp.kill" @click="runAgentAction(agentID(agent), 'kill')">K</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
