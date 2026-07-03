import { reactive } from 'vue'
import { getTasks, runDemo, type RuntimeEvent, type Task } from '../api/client'

export const runtimeStore = reactive({
  tasks: [] as Task[],
  events: [] as RuntimeEvent[],
  selectedTaskID: '',
  loading: false,
  error: '',
  connected: false
})

let eventSource: EventSource | null = null

export async function refreshTasks() {
  runtimeStore.tasks = await getTasks()
  if (!runtimeStore.selectedTaskID && runtimeStore.tasks.length > 0) {
    runtimeStore.selectedTaskID = runtimeStore.tasks[0].task_id
  }
  for (const task of runtimeStore.tasks) {
    for (const event of task.events ?? []) {
      addEvent(event)
    }
  }
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
    'agent.state_changed',
    'scheduler.selected',
    'syscall.finished',
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
}
