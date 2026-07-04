<script setup lang="ts">
import StatusBadge from '../components/StatusBadge.vue'
import { runAgentAction, runtimeStore } from '../stores/runtime'
import type { Agent } from '../api/client'

function agentID(agent: Agent): string {
  return agent.agent_id ?? agent.id ?? ''
}
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <h1>AVP & Capsule</h1>
        <p>Agent process state, cgroup capsule evidence, and runtime controls.</p>
      </div>
    </div>

    <div v-if="runtimeStore.error" class="error-line">{{ runtimeStore.error }}</div>

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Agent</th>
            <th>Role</th>
            <th>State</th>
            <th>PID</th>
            <th>Capsule</th>
            <th>Memory</th>
            <th>PIDs</th>
            <th>Retry</th>
            <th>Actions</th>
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
              <button class="icon-button" title="Freeze Agent" @click="runAgentAction(agentID(agent), 'freeze')">F</button>
              <button class="icon-button" title="Unfreeze Agent" @click="runAgentAction(agentID(agent), 'unfreeze')">U</button>
              <button class="icon-button danger" title="Kill Agent" @click="runAgentAction(agentID(agent), 'kill')">K</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
