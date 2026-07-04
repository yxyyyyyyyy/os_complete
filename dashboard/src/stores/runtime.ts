import { reactive } from 'vue'
import {
  getAgents,
  getContextPages,
  getContextStats,
  getEvidenceReport,
  getExperimentResults,
  getKernelEvents,
  getKernelStatus,
  getIPCMetrics,
  getIPCTopics,
  getPressureStatus,
  getRecoveryStatus,
  getSchedulerDecisions,
  getTasks,
  postAgentAction,
  runDemo,
  type Agent,
  type ContextPage,
  type ContextStats,
  type EvidenceReport,
  type ExperimentResults,
  type IPCMetric,
  type IPCTopics,
  type KernelEvent,
  type KernelStatus,
  type PressureStatus,
  type RecoveryStatus,
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
  kernelStatus: {
    enabled: false,
    mode: 'degraded-proxy',
    probe: 'syscall-gateway-proxy',
    reason: '',
    btf_available: false,
    bpffs_ready: false,
    event_count: 0
  } as KernelStatus,
  kernelEvents: [] as KernelEvent[],
  pressureStatus: {
    mode: 'degraded',
    degraded: true,
    reason: '',
    cpu: {
      some: { kind: 'some', avg10: 0, avg60: 0, avg300: 0, total: 0 },
      full: { kind: 'full', avg10: 0, avg60: 0, avg300: 0, total: 0 }
    },
    memory: {
      some: { kind: 'some', avg10: 0, avg60: 0, avg300: 0, total: 0 },
      full: { kind: 'full', avg10: 0, avg60: 0, avg300: 0, total: 0 }
    },
    io: {
      some: { kind: 'some', avg10: 0, avg60: 0, avg300: 0, total: 0 },
      full: { kind: 'full', avg10: 0, avg60: 0, avg300: 0, total: 0 }
    },
    throttle: false,
    throttle_reason: '',
    sampled_at: 0
  } as PressureStatus,
  evidenceReport: {
    updated_at: 0,
    modules: []
  } as EvidenceReport,
  recoveryStatus: {
    mode: 'checkpoint-light',
    degraded: true,
    reason: '',
    task_count: 0,
    recovered_at: 0,
    recovered_tasks: []
  } as RecoveryStatus,
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
    },
    e1_real_scheduler: [],
    e2_real_fault: [],
    e3_real_context: [],
    e4_real_ipc: [],
    e5_end_to_end: {
      experiment: '',
      demo: '',
      evidence_mode: '',
      wall_time_ms: 0,
      agents: 0,
      syscalls: 0,
      tool_exec: 0,
      ipc_messages: 0,
      context_saved_tokens: 0,
      fault_recovered: false,
      final_success: false,
      throughput_score: 0
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
  runtimeStore.kernelStatus = await getKernelStatus()
  runtimeStore.kernelEvents = await getKernelEvents()
  runtimeStore.pressureStatus = await getPressureStatus()
  runtimeStore.evidenceReport = await getEvidenceReport()
  runtimeStore.recoveryStatus = await getRecoveryStatus()
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
    'agent.recovered',
    'agent.report',
    'agent.capsule_attached',
    'agent.heartbeat_lost',
    'agent.state_changed',
    'scheduler.selected',
    'scheduler.pressure_throttle',
    'pressure.sampled',
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
    'checkpoint.recovered',
    'runtime.recovered',
    'runtime.recovery_failed',
    'kernel.observer_disabled',
    'kernel.exec',
    'workspace.snapshot.created',
    'workspace.created',
    'workspace.rmrf',
    'workspace.rollback',
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
    event.type.startsWith('checkpoint.') ||
    event.type.startsWith('runtime.') ||
    event.type.startsWith('kernel.') ||
    event.type.startsWith('pressure.') ||
    event.type.startsWith('scheduler.') ||
    event.type.startsWith('workspace.') ||
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
