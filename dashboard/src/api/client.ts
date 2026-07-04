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
