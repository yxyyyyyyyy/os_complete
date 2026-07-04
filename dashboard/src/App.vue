<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Overview from './pages/Overview.vue'
import AvpCapsule from './pages/AvpCapsule.vue'
import ContextMemory from './pages/ContextMemory.vue'
import Timeline from './pages/Timeline.vue'
import Experiments from './pages/Experiments.vue'
import { connectEvents, refreshTasks } from './stores/runtime'
import { language, setLanguage, t, type Language } from './stores/i18n'

const tabs = computed(() => [
  { id: 'overview', label: t.value.nav.overview, component: Overview },
  { id: 'avp', label: t.value.nav.avp, component: AvpCapsule },
  { id: 'context', label: t.value.nav.context, component: ContextMemory },
  { id: 'timeline', label: t.value.nav.timeline, component: Timeline },
  { id: 'experiments', label: t.value.nav.experiments, component: Experiments }
])

const languages: Array<{ id: Language; label: string }> = [
  { id: 'zh', label: '中文' },
  { id: 'en', label: 'EN' }
]

const activeTab = ref('overview')
const activeComponent = computed(() => tabs.value.find((tab) => tab.id === activeTab.value)?.component ?? Overview)

onMounted(() => {
  connectEvents()
  void refreshTasks()
})
</script>

<template>
  <main class="app-shell">
    <aside class="sidebar">
      <div class="brand-block">
        <div class="brand-mark">AR</div>
        <div>
          <strong>AORT-R</strong>
          <span>{{ t.app.subtitle }}</span>
        </div>
      </div>
      <nav>
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="nav-button"
          :class="{ active: activeTab === tab.id }"
          @click="activeTab = tab.id"
        >
          <span class="nav-indicator"></span>
          {{ tab.label }}
        </button>
      </nav>
    </aside>
    <section class="workspace">
      <header class="topbar">
        <div class="runtime-pill">
          <span class="pulse-dot"></span>
          <span>{{ t.app.backend }}</span>
        </div>
        <div class="topbar-actions">
          <span class="language-label">{{ t.app.language }}</span>
          <div class="segmented-control">
            <button
              v-for="item in languages"
              :key="item.id"
              :class="{ active: language === item.id }"
              @click="setLanguage(item.id)"
            >
              {{ item.label }}
            </button>
          </div>
        </div>
      </header>
      <component :is="activeComponent" />
    </section>
  </main>
</template>
