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

export async function getSchedulerDecisions(): Promise<SchedulerDecision[]> {
  const response = await fetch('/api/scheduler/decisions')
  if (!response.ok) {
    throw new Error(`scheduler decisions request failed: ${response.status}`)
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
