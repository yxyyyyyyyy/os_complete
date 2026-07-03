<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Overview from './pages/Overview.vue'
import AvpCapsule from './pages/AvpCapsule.vue'
import ContextMemory from './pages/ContextMemory.vue'
import Timeline from './pages/Timeline.vue'
import Experiments from './pages/Experiments.vue'
import { connectEvents, refreshTasks } from './stores/runtime'

const tabs = [
  { id: 'overview', label: 'Overview', component: Overview },
  { id: 'avp', label: 'AVP', component: AvpCapsule },
  { id: 'context', label: 'Context', component: ContextMemory },
  { id: 'timeline', label: 'Timeline', component: Timeline },
  { id: 'experiments', label: 'Experiments', component: Experiments }
]

const activeTab = ref(tabs[0].id)
const activeComponent = computed(() => tabs.find((tab) => tab.id === activeTab.value)?.component ?? Overview)

onMounted(() => {
  connectEvents()
  void refreshTasks()
})
</script>

<template>
  <main class="app-shell">
    <aside class="sidebar">
      <div class="brand-block">
        <strong>AORT-R</strong>
        <span>Agent Runtime</span>
      </div>
      <nav>
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="nav-button"
          :class="{ active: activeTab === tab.id }"
          @click="activeTab = tab.id"
        >
          {{ tab.label }}
        </button>
      </nav>
    </aside>
    <component :is="activeComponent" />
  </main>
</template>
