<script setup lang="ts">
import StatusBadge from '../components/StatusBadge.vue'
import { selectedTask } from '../stores/runtime'

const task = selectedTask
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <h1>AVP & Capsule</h1>
        <p>V1 shows AVP state. V2 adds PID, cgroup, and capsule controls.</p>
      </div>
    </div>

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Agent</th>
            <th>Role</th>
            <th>State</th>
            <th>Retry</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="agent in task()?.agents || []" :key="agent.id">
            <td>{{ agent.id }}</td>
            <td>{{ agent.role }}</td>
            <td><StatusBadge :value="agent.state" /></td>
            <td>0</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
