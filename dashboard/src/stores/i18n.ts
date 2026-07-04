import { computed, ref } from 'vue'

export type Language = 'zh' | 'en'

const messages = {
  zh: {
    app: {
      subtitle: '多智能体操作系统级执行时',
      statusOnline: '运行中',
      statusOffline: '离线',
      backend: 'Runtime Daemon',
      language: '语言'
    },
    nav: {
      overview: '运行总览',
      avp: 'AVP 胶囊',
      context: '上下文内存',
      timeline: '系统时间线',
      experiments: '实验分析'
    },
    common: {
      actions: '操作',
      agent: 'Agent',
      role: '角色',
      state: '状态',
      policy: '策略',
      selected: '选中',
      reason: '原因',
      candidates: '候选',
      records: '条记录',
      emptyScheduler: '暂无调度决策',
      running: '运行中',
      runDemo: '运行演示',
      mode: '模式',
      task: '任务',
      success: '成功',
      failed: '失败',
      pending: '等待中'
    },
    overview: {
      title: 'AORT-R 运行时驾驶舱',
      desc: '真实 worker、Agent 调度、系统调用与上下文复用证据。',
      tasks: '任务数',
      events: '事件数',
      sse: '事件流',
      runtimeMode: '运行模式',
      dag: '任务 DAG',
      scheduler: '调度决策',
      root: '根节点',
      recovery: 'Checkpoint 恢复',
      recoveryMode: '恢复模式',
      recoveredTasks: '恢复任务',
      readyAgents: '可续跑 Agent',
      pageRefs: '页表引用',
      degraded: '降级',
      full: '完整',
      noRecovery: '暂无恢复记录'
    },
    avp: {
      title: 'AVP 与 Capsule',
      desc: 'Agent 进程状态、cgroup 胶囊证据与运行时控制。',
      pid: 'PID',
      capsule: '胶囊',
      memory: '内存',
      pids: '进程数',
      retry: '重试',
      freeze: '冻结 Agent',
      unfreeze: '恢复 Agent',
      kill: '终止 Agent'
    },
    context: {
      title: 'CVM 上下文内存',
      desc: '上下文页、引用计数与去重节省指标。',
      totalPages: '总页数',
      sharedPages: '共享页',
      savedBytes: '节省字节',
      savedTokens: '节省 Token',
      ipcMessages: 'IPC 消息',
      ipcDepth: 'Topic 深度',
      avoidedCopy: '避免复制',
      ipcAvoidedBytes: 'IPC 避免复制',
      ipcTopics: 'IPC 黑板',
      publisher: '发布者',
      page: '页 ID',
      kind: '类型',
      bytes: '字节',
      tokens: 'Token',
      refs: '引用'
    },
    timeline: {
      title: '系统时间线',
      desc: 'Runtime、syscall、kernel、supervisor 与上下文事件流。',
      time: '时间',
      event: '事件',
      source: '来源',
      owner: '归属',
      kernelMode: 'Kernel 模式',
      kernelProbe: '观测探针',
      kernelEvents: 'Kernel 事件',
      btf: 'BTF'
    },
    experiments: {
      title: '实验分析',
      desc: 'E1 调度、E2 故障隔离、E3 上下文复用输出。',
      e1: 'E1 调度实验',
      policies: '种策略',
      e2: 'E2 故障隔离',
      e2Hint: 'tool timeout / pids / rollback 模型',
      e3: 'E3 上下文复用',
      affected: '影响 Agent',
      recovery: '恢复时间',
      rollback: '回滚',
      faults: '故障数',
      fullCopyTokens: '全量复制 Token',
      uniqueTokens: '唯一页 Token'
    },
    status: {
      CREATED: '已创建',
      READY: '就绪',
      RUNNING: '运行中',
      WAITING_LLM: '等待模型',
      WAITING_TOOL: '等待工具',
      WAITING_IPC: '等待通信',
      SUSPENDED: '已挂起',
      COMPLETED: '已完成',
      FAILED: '失败',
      KILLED: '已终止',
      unknown: '未知'
    }
  },
  en: {
    app: {
      subtitle: 'OS-level runtime for multi-agent execution',
      statusOnline: 'Online',
      statusOffline: 'Offline',
      backend: 'Runtime Daemon',
      language: 'Language'
    },
    nav: {
      overview: 'Overview',
      avp: 'AVP Capsule',
      context: 'Context Memory',
      timeline: 'Timeline',
      experiments: 'Experiments'
    },
    common: {
      actions: 'Actions',
      agent: 'Agent',
      role: 'Role',
      state: 'State',
      policy: 'Policy',
      selected: 'Selected',
      reason: 'Reason',
      candidates: 'Candidates',
      records: 'records',
      emptyScheduler: 'No scheduler decisions yet.',
      running: 'Running',
      runDemo: 'Run Demo',
      mode: 'Mode',
      task: 'Task',
      success: 'success',
      failed: 'failed',
      pending: 'pending'
    },
    overview: {
      title: 'AORT-R Runtime Cockpit',
      desc: 'Evidence for real workers, Agent scheduling, syscalls, and CVM sharing.',
      tasks: 'Tasks',
      events: 'Events',
      sse: 'SSE',
      runtimeMode: 'Runtime Mode',
      dag: 'Task DAG',
      scheduler: 'Scheduler Decisions',
      root: 'root',
      recovery: 'Checkpoint Recovery',
      recoveryMode: 'Recovery Mode',
      recoveredTasks: 'Recovered Tasks',
      readyAgents: 'Resumable Agents',
      pageRefs: 'Page References',
      degraded: 'Degraded',
      full: 'Full',
      noRecovery: 'No recovery records yet'
    },
    avp: {
      title: 'AVP & Capsule',
      desc: 'Agent process state, cgroup capsule evidence, and runtime controls.',
      pid: 'PID',
      capsule: 'Capsule',
      memory: 'Memory',
      pids: 'PIDs',
      retry: 'Retry',
      freeze: 'Freeze Agent',
      unfreeze: 'Unfreeze Agent',
      kill: 'Kill Agent'
    },
    context: {
      title: 'CVM Context Memory',
      desc: 'Context pages, reference counts, and deduplication savings.',
      totalPages: 'Total Pages',
      sharedPages: 'Shared Pages',
      savedBytes: 'Saved Bytes',
      savedTokens: 'Saved Tokens',
      ipcMessages: 'IPC Messages',
      ipcDepth: 'Topic Depth',
      avoidedCopy: 'Avoided Copy',
      ipcAvoidedBytes: 'IPC Avoided Copy',
      ipcTopics: 'IPC Blackboard',
      publisher: 'Publisher',
      page: 'Page',
      kind: 'Kind',
      bytes: 'Bytes',
      tokens: 'Tokens',
      refs: 'Refs'
    },
    timeline: {
      title: 'System Timeline',
      desc: 'Runtime, syscall, kernel, supervisor, and context event stream.',
      time: 'Time',
      event: 'Event',
      source: 'Source',
      owner: 'Owner',
      kernelMode: 'Kernel Mode',
      kernelProbe: 'Probe',
      kernelEvents: 'Kernel Events',
      btf: 'BTF'
    },
    experiments: {
      title: 'Experiment Analysis',
      desc: 'E1 scheduler, E2 fault isolation, and E3 context sharing outputs.',
      e1: 'E1 Scheduler',
      policies: 'policies',
      e2: 'E2 Fault Isolation',
      e2Hint: 'tool timeout / pids / rollback model',
      e3: 'E3 Context Sharing',
      affected: 'Affected',
      recovery: 'Recovery',
      rollback: 'Rollback',
      faults: 'Faults',
      fullCopyTokens: 'Full Copy Tokens',
      uniqueTokens: 'Unique Tokens'
    },
    status: {
      CREATED: 'Created',
      READY: 'Ready',
      RUNNING: 'Running',
      WAITING_LLM: 'Waiting LLM',
      WAITING_TOOL: 'Waiting Tool',
      WAITING_IPC: 'Waiting IPC',
      SUSPENDED: 'Suspended',
      COMPLETED: 'Completed',
      FAILED: 'Failed',
      KILLED: 'Killed',
      unknown: 'Unknown'
    }
  }
} as const

export const language = ref<Language>('zh')
export const t = computed(() => messages[language.value])

export function setLanguage(next: Language) {
  language.value = next
}

export function statusLabel(value: string): string {
  const labels = messages[language.value].status as Record<string, string>
  return labels[value] ?? labels.unknown
}
