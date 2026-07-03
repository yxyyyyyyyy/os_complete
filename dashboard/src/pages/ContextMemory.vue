<script setup lang="ts">
import { selectedTask } from '../stores/runtime'

const task = selectedTask
</script>

<template>
  <section class="page-band">
    <div class="page-heading">
      <div>
        <h1>Context Memory</h1>
        <p>V1 displays materialization events. V2 replaces this with CVM page tables.</p>
      </div>
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Event</th>
            <th>Agent</th>
            <th>Syscall</th>
            <th>Exit</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="event in (task()?.events || []).filter((item) => item.type === 'syscall.finished')"
            :key="event.id"
          >
            <td>{{ event.id }}</td>
            <td>{{ event.agent_id }}</td>
            <td>{{ event.payload.name }}</td>
            <td>{{ event.payload.exit_code }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
