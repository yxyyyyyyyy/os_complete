import { reactive } from 'vue'
import {
  getAgents,
  getContextPages,
  getContextStats,
  getExperimentResults,
  getIPCMetrics,
  getIPCTopics,
  getSchedulerDecisions,
  getTasks,
  postAgentAction,
  runDemo,
  type Agent,
  type ContextPage,
  type ContextStats,
  type ExperimentResults,
  type IPCMetric,
  type IPCTopics,
  type RuntimeEvent,
  type SchedulerDecision,
  type Task
} from '../api/client'

export const runtimeStore = reactive({
  tasks: [] as Task[],
  agents: [] as Agent[],
  events: [] as RuntimeEvent[],
  contextPages: [] as ContextPage[],
  contextStats: {
    total_pages: 0,
    shared_pages: 0,
    saved_bytes: 0,
    saved_tokens: 0
  } as ContextStats,
  ipcMetrics: {
    total_messages: 0,
    delivered_messages: 0,
    topic_depth: 0,
    avoided_copy_bytes: 0
  } as IPCMetric,
  ipcTopics: {} as IPCTopics,
  schedulerDecisions: [] as SchedulerDecision[],
  experimentResults: {
    e1_scheduler: [],
    e2_fault: [],
    e3_context: {
      experiment: '',
      mode: '',
      runs: 0,
      total_prompt_tokens: 0,
      unique_page_tokens: 0,
      saved_tokens: 0,
      saved_bytes: 0,
      ipc_avoided_copy_bytes: 0,
      materialize_time_ms: 0
    }
  } as ExperimentResults,
  selectedTaskID: '',
  loading: false,
  error: '',
  connected: false
})

let eventSource: EventSource | null = null
let refreshTimer: number | undefined

export async function refreshTasks() {
  runtimeStore.tasks = await getTasks()
  runtimeStore.agents = await getAgents()
  runtimeStore.schedulerDecisions = await getSchedulerDecisions()
  runtimeStore.experimentResults = await getExperimentResults()
  await refreshContext()
  if (!runtimeStore.selectedTaskID && runtimeStore.tasks.length > 0) {
    runtimeStore.selectedTaskID = runtimeStore.tasks[0].task_id
  }
  for (const task of runtimeStore.tasks) {
    for (const event of task.events ?? []) {
      addEvent(event)
    }
  }
}

export async function runAgentAction(agentID: string, action: 'freeze' | 'unfreeze' | 'kill') {
  runtimeStore.error = ''
  try {
    await postAgentAction(agentID, action)
    await refreshTasks()
  } catch (error) {
    runtimeStore.error = error instanceof Error ? error.message : String(error)
  }
}

export async function refreshContext() {
  runtimeStore.contextPages = await getContextPages()
  runtimeStore.contextStats = await getContextStats()
  runtimeStore.ipcMetrics = await getIPCMetrics()
  runtimeStore.ipcTopics = await getIPCTopics()
}

export async function startDemo() {
  runtimeStore.loading = true
  runtimeStore.error = ''
  try {
    const result = await runDemo()
    runtimeStore.selectedTaskID = result.task_id
    await refreshTasks()
  } catch (error) {
    runtimeStore.error = error instanceof Error ? error.message : String(error)
  } finally {
    runtimeStore.loading = false
  }
}

export function connectEvents() {
  if (eventSource) {
    return
  }
  eventSource = new EventSource('/api/events')
  eventSource.onopen = () => {
    runtimeStore.connected = true
  }
  eventSource.onerror = () => {
    runtimeStore.connected = false
  }
  eventSource.onmessage = (message) => {
    addEvent(JSON.parse(message.data) as RuntimeEvent)
  }
  const eventTypes = [
    'runtime.connected',
    'task.created',
    'agent.created',
    'agent.registered',
    'agent.report',
    'agent.capsule_attached',
    'agent.heartbeat_lost',
    'agent.state_changed',
    'scheduler.selected',
    'syscall.started',
    'syscall.finished',
    'context.page.created',
    'context.page.reused',
    'context.page.mounted',
    'context.materialized',
    'ipc.published',
    'ipc.polled',
    'llm.called',
    'agent.spawn.requested',
    'agent.spawned',
    'supervisor.detected',
    'task.completed'
  ]
  for (const type of eventTypes) {
    eventSource.addEventListener(type, (message) => {
      addEvent(JSON.parse((message as MessageEvent).data) as RuntimeEvent)
    })
  }
}

export function selectedTask(): Task | undefined {
  return runtimeStore.tasks.find((task) => task.task_id === runtimeStore.selectedTaskID) ?? runtimeStore.tasks[0]
}

function addEvent(event: RuntimeEvent) {
  if (runtimeStore.events.some((existing) => existing.id === event.id && existing.type === event.type)) {
    return
  }
  runtimeStore.events.push(event)
  runtimeStore.events.sort((a, b) => a.timestamp - b.timestamp)
  if (
    event.type.startsWith('agent.') ||
    event.type.startsWith('context.') ||
    event.type.startsWith('ipc.') ||
    event.type.startsWith('llm.') ||
    event.type.startsWith('supervisor.') ||
    event.type.startsWith('syscall.')
  ) {
    scheduleRefresh()
  }
}

function scheduleRefresh() {
  if (refreshTimer !== undefined) {
    return
  }
  refreshTimer = window.setTimeout(() => {
    refreshTimer = undefined
    void refreshTasks().catch((error) => {
      runtimeStore.error = error instanceof Error ? error.message : String(error)
    })
  }, 250)
}
