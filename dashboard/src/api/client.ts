export interface RuntimeEvent {
  id: string
  task_id: string
  agent_id?: string
  type: string
  source: string
  timestamp: number
  payload: Record<string, unknown>
}

export interface Agent {
  id?: string
  agent_id?: string
  task_id?: string
  role: string
  state: string
  pid?: number
  cgroup_path?: string
  capsule_mode?: string
  memory_current?: number
  pids_current?: number
  retry_count?: number
  last_seen?: number
}

export interface DAGNode {
  id: string
  role: string
  dependencies: string[] | null
}

export interface Task {
  task_id: string
  status: string
  agents: Agent[]
  dag: DAGNode[]
  events: RuntimeEvent[]
}

export interface ContextPage {
  id: string
  kind: string
  content: string
  bytes: number
  token_count: number
  ref_count: number
  created_at: number
}

export interface ContextStats {
  total_pages: number
  shared_pages: number
  saved_bytes: number
  saved_tokens: number
}

export interface IPCMetric {
  topic?: string
  total_messages: number
  delivered_messages: number
  topic_depth: number
  avoided_copy_bytes: number
}

export interface IPCMessage {
  id: string
  topic: string
  publisher: string
  page_id: string
  size_bytes: number
  created_at: number
}

export type IPCTopics = Record<string, IPCMessage[]>

export interface SchedulerDecision {
  id: string
  task_id: string
  candidates: string[]
  selected_agent: string
  policy: string
  reason: string
  vruntime_before: Record<string, number>
  vruntime_after: Record<string, number>
  shared_pages: Record<string, number>
  created_at: number
}

export interface RecoveredTask {
  task_id: string
  sequence: number
  status: string
  agent_count: number
  completed_agents: string[]
  ready_agents: string[]
  page_table_refs: number
  scheduler_vruntime: Record<string, number>
  created_at: number
}

export interface RecoveryStatus {
  mode: string
  degraded: boolean
  reason: string
  task_count: number
  recovered_at: number
  recovered_tasks: RecoveredTask[]
}

export interface KernelStatus {
  enabled: boolean
  mode: string
  probe: string
  reason: string
  btf_available: boolean
  bpffs_ready: boolean
  event_count: number
}

export interface KernelEvent {
  id: string
  type: string
  source: string
  task_id: string
  agent_id: string
  pid: number
  command: string
  args: string[]
  cgroup_path?: string
  workspace?: string
  status: string
  mode: string
  probe: string
  timestamp: number
}

export interface PressureLine {
  kind: string
  avg10: number
  avg60: number
  avg300: number
  total: number
}

export interface PressureResource {
  some: PressureLine
  full: PressureLine
}

export interface PressureStatus {
  mode: string
  degraded: boolean
  reason?: string
  cpu: PressureResource
  memory: PressureResource
  io: PressureResource
  throttle: boolean
  throttle_reason?: string
  sampled_at: number
}

export interface E1SchedulerResult {
  experiment: string
  policy: string
  mode: string
  runs: number
  total_time_ms: number
  avg_wait_time_ms: number
  jain_fairness: number
  decision_count: number
}

export interface E2FaultResult {
  experiment: string
  mode: string
  runs: number
  affected_agents: number
  recovery_time_ms: number
  task_success: boolean
  rollback_success: boolean
  fault_count: number
}

export interface E3ContextResult {
  experiment: string
  mode: string
  runs: number
  total_prompt_tokens: number
  unique_page_tokens: number
  saved_tokens: number
  saved_bytes: number
  ipc_avoided_copy_bytes: number
  materialize_time_ms: number
}

export interface ExperimentResults {
  e1_scheduler: E1SchedulerResult[]
  e2_fault: E2FaultResult[]
  e3_context: E3ContextResult
}

export async function runDemo(): Promise<{ task_id: string }> {
  const response = await fetch('/api/demo/run', { method: 'POST' })
  if (!response.ok) {
    throw new Error(`demo run failed: ${response.status}`)
  }
  return response.json()
}

export async function getTasks(): Promise<Task[]> {
  const response = await fetch('/api/tasks')
  if (!response.ok) {
    throw new Error(`tasks request failed: ${response.status}`)
  }
  return response.json()
}

export async function getAgents(): Promise<Agent[]> {
  const response = await fetch('/api/agents')
  if (!response.ok) {
    throw new Error(`agents request failed: ${response.status}`)
  }
  return response.json()
}

export async function postAgentAction(agentID: string, action: 'freeze' | 'unfreeze' | 'kill'): Promise<void> {
  const response = await fetch(`/api/agents/${encodeURIComponent(agentID)}/${action}`, { method: 'POST' })
  if (!response.ok) {
    throw new Error(`agent ${action} failed: ${response.status}`)
  }
}

export async function getContextPages(): Promise<ContextPage[]> {
  const response = await fetch('/api/context/pages')
  if (!response.ok) {
    throw new Error(`context pages request failed: ${response.status}`)
  }
  return response.json()
}

export async function getContextStats(): Promise<ContextStats> {
  const response = await fetch('/api/context/stats')
  if (!response.ok) {
    throw new Error(`context stats request failed: ${response.status}`)
  }
  return response.json()
}

export async function getIPCMetrics(): Promise<IPCMetric> {
  const response = await fetch('/api/ipc/metrics')
  if (!response.ok) {
    throw new Error(`ipc metrics request failed: ${response.status}`)
  }
  return response.json()
}

export async function getIPCTopics(): Promise<IPCTopics> {
  const response = await fetch('/api/ipc/topics')
  if (!response.ok) {
    throw new Error(`ipc topics request failed: ${response.status}`)
  }
  return response.json()
}

export async function getSchedulerDecisions(): Promise<SchedulerDecision[]> {
  const response = await fetch('/api/scheduler/decisions')
  if (!response.ok) {
    throw new Error(`scheduler decisions request failed: ${response.status}`)
  }
  return response.json()
}

export async function getRecoveryStatus(): Promise<RecoveryStatus> {
  const response = await fetch('/api/recovery/status')
  if (!response.ok) {
    throw new Error(`recovery status request failed: ${response.status}`)
  }
  return response.json()
}

export async function getKernelStatus(): Promise<KernelStatus> {
  const response = await fetch('/api/kernel/status')
  if (!response.ok) {
    throw new Error(`kernel status request failed: ${response.status}`)
  }
  return response.json()
}

export async function getKernelEvents(): Promise<KernelEvent[]> {
  const response = await fetch('/api/kernel/events')
  if (!response.ok) {
    throw new Error(`kernel events request failed: ${response.status}`)
  }
  return response.json()
}

export async function getPressureStatus(): Promise<PressureStatus> {
  const response = await fetch('/api/pressure/status')
  if (!response.ok) {
    throw new Error(`pressure status request failed: ${response.status}`)
  }
  return response.json()
}

export async function getExperimentResults(): Promise<ExperimentResults> {
  const response = await fetch('/api/experiments/results')
  if (!response.ok) {
    throw new Error(`experiment results request failed: ${response.status}`)
  }
  return response.json()
}
