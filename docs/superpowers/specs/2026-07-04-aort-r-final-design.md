# AORT-R 最终融合版总体设计

> 历史设计说明：本文保留早期目标态，不作为当前提交的能力证明。“满血版”、namespace、llama.cpp prefix cache、完整 eBPF、零拷贝等目标必须以最新代码和 evidence_mode 重新判断。当前评审设计见 `docs/design/README.md`。

版本：Final Design / 冲高分创新版  
日期：2026-07-04  
主平台：openEuler 24.03 LTS  
项目名：AORT-R, Agent-Oriented Runtime on openEuler - Real  
中文名：面向多智能体的操作系统级 Agent Runtime  
目标赛题：面向多智能体的操作系统级执行时（Agent Runtime）高校赛题  
团队约束：2 人开发，14 天交付，Go 后端 + Vue 前端，openEuler VM，有 root 权限，无 GPU  

---

## 1. 最终路线选择

本文记录早期融合目标：以 V2.0 设想的 OS 机制为方向，保留 V1.0 的 AVP / CVM / Gateway / Event / Trace 抽象。实际完成度不由本文标题推断。

V1.0 的优势是概念完整，适合讲清楚“Agent OS Runtime”的抽象；不足是大部分机制停留在用户态数据结构，容易被质疑为普通多 Agent Workflow 的 OS 类比包装。

V2.0 的优势是把关键抽象落到 openEuler/Linux 真实机制上：Agent 对应真实进程与 cgroup，上下文复用对应内容寻址页和 prefix cache，故障隔离对应 cgroup/namespace/overlayfs，系统观测对应应用 trace + eBPF 内核事件。它更贴合赛题要求的“操作系统级执行时”。

最终融合原则：

1. 用 V1.0 的五个抽象负责“讲清楚”：AVP、CVM、Syscall、Runtime Event、Replayable Trace。
2. 用 V2.0 的真实机制负责“做扎实”：cgroup v2、namespace、overlayfs、PSI、eBPF、llama.cpp prefix cache、systemd restart。
3. 所有创新点都必须有三类证据：实现模块、Dashboard 可见证据、实验指标。

一句话定位：

> 当前 AORT-R 是运行在 openEuler/Linux 用户态的多 Agent Runtime 原型，把 Agent 建模为运行时执行对象；真实 Linux 机制按环境和 evidence_mode 启用。

答辩核心话术：

> AORT-R 在应用编排之外管理 worker、可选 cgroup/workspace、CVM 页、受控调用、调度和 Timeline。cgroup、OverlayFS、memfd/mmap 与 eBPF 的可用性必须逐项引用证据，不能从架构图直接推断。

---

## 2. 与赛题要求的对应关系

| 赛题要求 | AORT-R 对应设计 | 证据形式 |
|---|---|---|
| 基于国内主流开源 OS | openEuler 24.03 LTS 主平台，openKylin/Ubuntu smoke test | 部署脚本、运行截图、录屏 |
| 多 Agent 任务依赖和动态生成 | DAG Manager，Reviewer 通过 `agent.spawn` 动态创建 Fixer | DAG 页面、trace 事件 |
| 资源感知调度 | token-CFS + prefix affinity + PSI throttle | 调度日志、E1 实验 |
| 统一 Agent 执行抽象 | AVP + Agent Capsule，生命周期状态机 | AVP & Capsule 页面 |
| 与 OS 资源关系 | 每个 AVP 对应 worker 进程、cgroup、overlay workspace | PID、cgroup 路径、资源曲线 |
| 容错与故障隔离 | cgroup 限额、pids.max、memory.events、overlayfs 回滚、Supervisor | E2 实验、故障注入演示 |
| 上下文复用、压缩、隔离 | CVM 内容寻址页、页表、CoW delta、summary page | Context Memory 页面、E3 实验 |
| Agent 高效通信 | page reference IPC + blackboard pub/sub | 避免拷贝字节数、IPC 延迟 |
| 系统级可观测与控制 | syscall audit、Replayable Trace、eBPF exec/open/connect、freeze/kill/replay | Timeline 页面、演示录屏 |
| 真实应用验证 | 多 Agent 软件工程 Demo | 端到端 Demo、实验报告 |

评分侧重点：

1. 系统设计与机制创新 30%：Agent Capsule、CVM、token-CFS、prefix affinity、双层 trace。
2. 功能完整性与实现深度 25%：aortd、worker、syscall、scheduler、supervisor、dashboard、demo。
3. 性能优化效果 20%：prefix cache 命中、TTFT、总时长、公平性、IPC 避免拷贝。
4. 工程实现质量 15%：Go 模块化后端、Vue Dashboard、systemd 部署、实验脚本。
5. 实验与分析质量 10%：E1/E2/E3/E4 多组对比实验，重复运行并报告均值和标准差。

---

## 3. 总体架构

```text
                 Web Dashboard (Vue 3 + Vite + ECharts)
   Overview / AVP Capsule / Context Memory / Timeline / Experiments
                          REST + SSE
                              |
                            aortd
     Go runtime daemon, systemd service, root on openEuler
                              |
  ----------------------------------------------------------------
  | AVP Manager        Agent 生命周期、状态机、DAG 依赖            |
  | Capsule Manager    cgroup v2、namespace、overlayfs workspace   |
  | CVM Manager        内容寻址页、页表、CoW、压缩、materialize    |
  | Scheduler          FIFO、token-CFS、prefix affinity、PSI throttle |
  | Syscall Gateway    capability、quota、timeout、audit、UDS JSON-RPC |
  | IPC Blackboard     page reference pub/sub，减少完整 payload 传输 |
  | Supervisor         retry、circuit breaker、OOM/PID 事件、spawn  |
  | Checkpoint         状态快照、workspace 快照、daemon crash 恢复  |
  | Trace Recorder     应用事件 JSONL + bbolt 索引 + replay         |
  | eBPF Observer      execve/openat/connect per cgroup             |
  | LLM Router         mock / DeepSeek relay / llama.cpp local      |
  ----------------------------------------------------------------
                              |
          UDS syscall channel /run/aort/aortd.sock
                              |
       Agent Worker Processes, one worker per AVP capsule
       Planner / Coder A / Coder B / Tester / Reviewer / Fixer
                              |
           Tool sandbox processes: shell, file, git, go test
```

核心原则：

1. Agent Worker 是真实独立进程，有 PID、有 cgroup、可冻结、可杀死、可恢复。
2. Worker 不直接访问文件、网络、模型或其他 Agent，所有外部能力必须经过 syscall gateway。
3. 每个 Agent 的工作区由 overlayfs 隔离，失败 Agent 的修改不会污染主快照。
4. 上下文不是字符串拼接，而是 CVM 页表；Agent 间通信优先传 page id，不复制大文本。
5. 调度器不只调 Agent 顺序，还同时优化 token 公平性、prefix cache 局部性和系统压力。
6. Dashboard 展示的是 Runtime 证据，不是聊天界面。

---

## 4. 核心创新点

### 4.1 Agent Capsule：把 Agent 变成受 Runtime 管理的执行对象

Agent Capsule 是 AORT-R 的首要创新。每个 Agent 不只是 Go 中的对象，而是一个真实 worker 进程，运行在独立 cgroup v2、命名空间和 overlayfs 工作区中。

对应机制：

| OS 概念 | AORT-R 机制 |
|---|---|
| Process | Agent Worker 进程 |
| PCB | AVP Control Block |
| cgroup | Agent Capsule 资源限额 |
| freezer | `cgroup.freeze` 挂起/恢复 |
| namespace | 工具执行隔离 |
| overlayfs | 每 Agent 的 CoW 工作区 |
| OOM event | `memory.events` 触发 Supervisor |

关键能力：

1. `cpu.max` 控制 CPU 配额。
2. `memory.max` 控制内存上限。
3. `pids.max` 防止 fork bomb。
4. `cgroup.freeze` 实现 Agent 暂停和恢复。
5. overlayfs upper 层记录 Agent 私有改动，失败后可直接丢弃。
6. Dashboard 显示每个 Agent 的 PID、cgroup、内存、CPU、重试次数和状态。

创新价值：

> Agent Capsule 让 Agent 从应用层任务对象变成可被 OS 资源管理机制直接约束的执行单元，是本项目区别于普通多 Agent 框架的第一证据。

### 4.2 CVM：面向多 Agent 的上下文虚拟内存

CVM, Context Virtual Memory，将多 Agent 上下文拆成不可变的内容寻址页，并为每个 AVP 维护独立页表。

核心结构：

```text
Page {
  id = sha256(content)
  kind = system | project | task | delta | summary
  bytes
  token_count
  ref_count
}

AVP Page Table:
  system pages -> project pages -> task pages -> private delta pages -> summary pages
```

关键机制：

1. 内容寻址：相同上下文只存一份。
2. 页表挂载：不同 Agent 可共享相同 project/task pages。
3. Copy-on-Write：Agent 修改共享上下文时生成私有 delta page。
4. Lazy Materialization：只有调用 LLM 前才展开成 prompt。
5. Summary Compression：超过 token budget 时把历史 delta 压缩为 summary page。
6. Reference Accounting：统计页引用次数，展示共享结构。

创新价值：

> CVM 把多 Agent 上下文从“反复复制的大字符串”变成“可共享、可追踪、可压缩、可审计的虚拟内存页”，直接回应赛题中的上下文冗余拷贝问题。

### 4.3 token-CFS + Prefix Affinity：推理缓存感知调度

传统多 Agent 调度通常只看任务依赖或优先级。AORT-R 的调度器同时考虑 token 公平性和 LLM prefix cache 局部性。

调度三层：

1. 公平层：token-CFS。
2. 亲和层：prefix affinity。
3. 资源层：PSI throttle。

token-CFS：

```text
vruntime += consumed_tokens / weight
每次从 READY 队列选择 vruntime 最小的 Agent
```

这借鉴 Linux CFS 的思想，但将 CPU 时间替换为 LLM token 消耗。它解决的问题是：多 Agent 共享模型调用预算时，不能让单个长上下文 Agent 长期占用推理资源。

Prefix Affinity：

```text
READY Agents 按 CVM 页表最长公共前缀分组
如果候选 Agent 与上一推理槽位有相同 prompt 前缀，
且 vruntime 与全局最小值差距不超过 K，
则优先调度该 Agent，以提高 prefix cache 命中。
```

PSI throttle：

读取 `/sys/fs/cgroup/aort.slice/cpu.pressure` 与 `memory.pressure`。当 avg10 超过阈值时，降低工具沙箱并发，避免系统压力继续放大。

创新价值：

> 这个调度器把 OS 公平调度、上下文页表和 LLM prefix cache 连接起来。它不仅能解释“为什么这样调度”，还能通过 cache hit tokens、TTFT 和总耗时证明性能收益。

### 4.4 Agent Syscall Gateway：统一模型、工具、通信和资源

Agent 不直接调用工具、文件、LLM 或网络，而是通过 UDS JSON-RPC 向 aortd 发起 syscall。

统一管线：

```text
request
 -> capability check
 -> quota check
 -> timeout
 -> audit start
 -> execute
 -> fault hook
 -> audit finish
 -> trace append
 -> response
```

syscall 清单：

```text
context.read
context.write_delta
context.materialize
llm.call
tool.exec
ipc.publish
ipc.poll
agent.spawn
agent.report
agent.freeze
agent.kill
```

工具能力：

```text
file.read
file.write
shell.exec
git.diff
go.test
http.request
```

创新价值：

> Syscall Gateway 将模型调用、工具调用、IPC 和 Agent 控制统一成系统调用抽象，使 Agent Runtime 具备类似 OS 的权限、配额、审计、超时和故障钩子。

### 4.5 Page Reference IPC：减少 Agent 间完整上下文传输

多 Agent 通信不传整段上下文，只传 CVM page id。

流程：

```text
Producer Agent:
  context.write_delta -> page_id
  ipc.publish(topic, page_id)

Consumer Agent:
  ipc.poll(topic) -> page_id
  mount page_id into own page table
```

可统计指标：

1. IPC 消息数。
2. page reference 数量。
3. 避免拷贝字节数：`page_size * (subscriber_count - 1)`。
4. 端到端通信延迟。

创新价值：

> Page Reference IPC 把 Agent 通信从文本复制变成页引用传递，和 CVM 形成闭环，是“高效通信机制”的直接落点。

### 4.6 Supervisor + Checkpoint：故障隔离与恢复

Supervisor 负责将单 Agent 故障限制在 Capsule 内，并决定重试、熔断、降级或动态 spawn。

故障类型：

```text
LLM_ERROR
TOOL_TIMEOUT
TOOL_EXIT_NONZERO
CAPSULE_OOM
PIDS_LIMIT
WORKSPACE_CONFLICT
CONTEXT_OVERFLOW
AGENT_CRASH
RETRY_LIMIT
DAEMON_CRASH
```

处理策略：

1. 工具超时：杀死工具进程，保留 worker，记录 syscall failure。
2. OOM：监听 `memory.events`，发现 `oom_kill` 增长后重建 capsule。
3. fork bomb：由 `pids.max` 阻断，Supervisor 标记 PIDS_LIMIT。
4. rm -rf：只破坏 overlay upper 层，丢弃 upper 即可回滚。
5. 测试失败：Reviewer 通过 `agent.spawn` 动态创建 Fixer。
6. daemon crash：systemd 拉起 aortd，从 checkpoint 继续。

Checkpoint 分两级：

1. 轻量 checkpoint：AVP 表、DAG 状态、页表引用、scheduler vruntime、trace offset。
2. 重量 checkpoint：Agent overlay upper 层快照，用于 daemon crash 后恢复工作区。

创新价值：

> Supervisor 不是简单 retry，而是与 cgroup、overlayfs、systemd、checkpoint 联动，实现单 Agent 故障隔离和运行时级恢复。

### 4.7 双层 Trace：应用事件 + eBPF 内核事件

AORT-R 的可观测性由两层组成。

应用层 trace：

```text
task.created
agent.created
agent.state_changed
scheduler.selected
context.page.created
context.page.mounted
context.materialized
syscall.started
syscall.finished
llm.called
ipc.published
supervisor.retry
checkpoint.created
task.completed
```

内核层 eBPF trace：

```text
sched_process_exec
sys_enter_openat
sys_enter_connect
```

过滤方式：

1. eBPF 捕获事件。
2. 用户态根据 cgroup id 映射回 AVP。
3. 写入同一条 Timeline。
4. Dashboard 按 Agent、事件类型、来源过滤。

创新价值：

> 双层 Trace 让系统既能看到 Agent 语义事件，也能看到其对应的真实 OS 行为，形成“系统级可观测性”的强证据。

---

## 5. Agent 生命周期

AVP 状态机：

```text
CREATED -> READY -> RUNNING

RUNNING -> WAITING_LLM  -> READY
RUNNING -> WAITING_TOOL -> READY
RUNNING -> WAITING_IPC  -> READY

RUNNING -> SUSPENDED -> READY
RUNNING -> COMPLETED
RUNNING -> FAILED -> READY
FAILED  -> KILLED
```

状态语义：

| 状态 | 含义 | OS 机制 |
|---|---|---|
| CREATED | AVP 已登记，capsule 尚未运行 | bbolt state |
| READY | 依赖满足，等待调度 | scheduler queue |
| RUNNING | Worker 正在执行 | process + cgroup |
| WAITING_LLM | 等待模型响应 | syscall pending |
| WAITING_TOOL | 等待工具沙箱结束 | child process |
| WAITING_IPC | 等待 blackboard 消息 | topic queue |
| SUSPENDED | 被人工或系统暂停 | `cgroup.freeze=1` |
| COMPLETED | Agent 成功完成 | checkpoint |
| FAILED | Agent 失败但可重试 | supervisor |
| KILLED | 超过重试或被强杀 | cgroup cleanup |

---

## 6. Demo 场景

主 Demo：多 Agent 自动软件工程任务。

用户输入：

```text
实现一个简单 Todo Web API，支持创建、查询、完成 Todo，并提供基础单元测试。
```

角色：

1. Planner：拆分任务并生成 DAG。
2. Coder A：实现 handler/API 层。
3. Coder B：实现 storage/model 层。
4. Tester：运行 go test。
5. Reviewer：审查 diff 和测试结果。
6. Fixer：Reviewer 动态 spawn，用于修复失败。

流程：

```text
Planner 生成 DAG
 -> Coder A / Coder B 并行执行，各自在独立 capsule 中修改 overlay upper
 -> aortd 检查文件冲突并合并生成新快照
 -> Tester 在工具沙箱运行 go test
 -> Reviewer 审查结果
 -> 若失败，Reviewer 调 agent.spawn 创建 Fixer
 -> Fixer 修复后 Tester 复测
 -> 任务完成，Dashboard 回放 Timeline
```

故障注入剧本：

| 剧本 | 注入方式 | 预期结果 |
|---|---|---|
| forkbomb | Tester 工具进程 fork 炸弹 | pids.max 阻断，Supervisor 重试 |
| oom | Coder 工具分配大量内存 | memory.events 捕获 OOM，重建 capsule |
| rmrf | 工具误删项目目录 | 只影响 overlay upper，主快照无损 |
| daemonkill | `kill -9 aortd` | systemd 重启，checkpoint 恢复续跑 |
| conflict | 两个 Coder 修改同一文件 | 产生 workspace.conflict，spawn Fixer |

---

## 7. Dashboard 设计

Dashboard 采用 5 页，避免页面过多导致实现分散。

### 7.1 Overview + DAG

展示内容：

1. 当前任务状态。
2. DAG 节点和边。
3. 动态生成节点高亮。
4. 当前调度策略。
5. 总耗时、成功率、重试次数。

### 7.2 AVP & Capsule

展示内容：

1. AVP 状态机。
2. Agent PID、cgroup path。
3. CPU、内存、PID 数曲线。
4. freeze、unfreeze、kill 操作按钮。
5. Capsule 限额和故障状态。

### 7.3 Context Memory

展示内容：

1. CVM page 列表。
2. page ref count。
3. 每个 Agent 的 page table。
4. materialize 记录。
5. prefix cache hit tokens 曲线。
6. IPC 避免拷贝字节数。

### 7.4 Timeline

展示内容：

1. 应用事件泳道。
2. syscall audit 泳道。
3. eBPF kernel event 泳道。
4. 按 Agent 和事件类型过滤。
5. replay 控制。

### 7.5 Experiments

展示内容：

1. E1 调度性能对比。
2. E2 故障隔离对比。
3. E3 上下文和 IPC 对比。
4. E4 可观测与恢复实验。
5. 实验运行配置和结果 JSON。

---

## 8. 实验设计

实验统一要求：

1. openEuler 24.03 LTS 主环境。
2. 每组重复运行至少 5 次。
3. 报告均值、标准差和失败样例。
4. 固定随机种子和 prompt。
5. 性能实验优先使用 llama.cpp local provider，确保可复现。

### E1 调度性能实验

目标：证明 token-CFS + prefix affinity 比 FIFO 更适合多 Agent 推理负载。

对照组：

1. FIFO。
2. token-CFS only。
3. token-CFS + prefix affinity。

负载：

1. 8 个 Agent。
2. 共享同一 project pages。
3. 每个 Agent 有少量 private delta pages。

指标：

1. 总墙钟时间。
2. TTFT。
3. prefill 耗时。
4. prefix cache hit tokens。
5. Jain 公平性指数。
6. 调度 decision log。

### E2 故障隔离实验

目标：证明 Agent Capsule 可以缩小故障爆炸半径。

对照组：

1. 无 per-agent capsule，仅全局兜底限制。
2. 启用 per-agent capsule。

注入：

1. forkbomb。
2. OOM。
3. rm -rf。

指标：

1. 兄弟 Agent 受影响数量。
2. 任务最终成功率。
3. 故障恢复时间。
4. OOM/PID limit 事件数。
5. workspace 污染情况。

### E3 上下文与 IPC 实验

目标：证明 CVM 和 page reference IPC 降低上下文复制开销。

对照组：

1. 全量复制上下文。
2. CVM page sharing。
3. CVM page sharing + summary compression。

指标：

1. 总 prompt token。
2. 去重节约 token。
3. page ref count。
4. IPC 避免拷贝字节数。
5. materialize 耗时。

### E4 可观测与恢复实验

目标：证明系统具有可回放、可诊断、可恢复能力。

实验：

1. 开启 trace replay，记录完整 Demo。
2. 捕获 eBPF exec/open/connect。
3. 运行中 kill -9 aortd。
4. systemd 拉起后从 checkpoint 续跑。

指标：

1. trace event 总数。
2. kernel event 总数。
3. replay 可重现阶段数。
4. checkpoint 写入耗时。
5. daemon crash 后恢复耗时。

---

## 9. API 摘要

REST API：

```text
POST /api/tasks
GET  /api/tasks
GET  /api/tasks/{task_id}
GET  /api/tasks/{task_id}/dag

GET  /api/agents
GET  /api/agents/{agent_id}
POST /api/agents/{agent_id}/freeze
POST /api/agents/{agent_id}/unfreeze
POST /api/agents/{agent_id}/kill

GET  /api/context/pages
GET  /api/context/agents/{agent_id}/pagetable

GET  /api/syscalls
GET  /api/traces/{task_id}
GET  /api/traces/{task_id}/replay

GET  /api/experiments/{name}
POST /api/demo/run
POST /api/demo/fault/{kind}

GET  /api/events
```

SSE 事件：

```text
task.updated
agent.state_changed
scheduler.selected
context.page_created
context.page_mounted
syscall.finished
ipc.published
supervisor.retry
kernel.event
checkpoint.created
experiment.updated
```

---

## 10. 工程模块划分

后端 Go 模块：

```text
cmd/aortd
cmd/aort-worker
internal/avp
internal/capsule
internal/cvm
internal/scheduler
internal/syscall
internal/ipc
internal/supervisor
internal/checkpoint
internal/trace
internal/ebpf
internal/llm
internal/tool
internal/api
internal/demo
internal/experiment
```

前端 Vue 模块：

```text
dashboard/src/pages/Overview.vue
dashboard/src/pages/AvpCapsule.vue
dashboard/src/pages/ContextMemory.vue
dashboard/src/pages/Timeline.vue
dashboard/src/pages/Experiments.vue
dashboard/src/api
dashboard/src/components
dashboard/src/stores
```

运行目录：

```text
/etc/aort/config.yaml
/run/aort/aortd.sock
/var/lib/aort/state.db
/var/lib/aort/pages/
/var/lib/aort/snapshots/
/var/lib/aort/capsules/
/var/lib/aort/checkpoints/
/var/lib/aort/traces/
/sys/fs/cgroup/aort.slice/
```

---

## 11. 14 天计划

D1：规格冻结、API schema、仓库骨架、systemd 与配置骨架。  
D2：AVP Manager、worker re-exec、UDS 注册、基础状态机。  
D3：Capsule v1：cgroup 创建、资源统计、freeze/unfreeze/kill。  
D4：overlayfs 工作区、工具沙箱、收编与回滚。  
D5：CVM page store、page table、materialize、引用计数。  
D6：LLM Router：mock、DeepSeek relay、llama.cpp local、usage/timing 采集。  
D7：Syscall Gateway、capability、quota、timeout、audit。  
D8：Scheduler：FIFO、token-CFS、prefix affinity、decision log。  
D9：IPC Blackboard、page reference publish/poll、避免拷贝统计。  
D10：Supervisor：retry、circuit breaker、OOM/PID 事件、agent.spawn。  
D11：Checkpoint/recovery、daemonkill 演示。  
D12：eBPF Observer、Timeline 三泳道联调。  
D13：E1/E2/E3/E4 实验脚本、图表、openKylin/Ubuntu smoke。  
D14：文档、PPT、录屏、答辩彩排、bug 修复与提交。

两人分工：

成员 A：aortd 后端主线，包括 AVP、Capsule、CVM、Syscall、Scheduler、Supervisor、Checkpoint、eBPF、部署。  
成员 B：Dashboard、Demo 剧本、llama.cpp 环境、实验脚本、图表、PPT、录屏。

并行原则：

1. D1 冻结 API schema。
2. 前端先使用 mock SSE 数据开发。
3. 后端每天保证一个可运行 demo slice。
4. 实验脚本从 D8 开始持续积累，避免最后两天才补数据。

---

## 12. 砍单线

为了冲高分但保证提交，按以下顺序裁剪：

1. eBPF openat/connect 裁掉，只保留 execve。
2. checkpoint 的 workspace tar 快照裁掉，只保留运行时状态快照。
3. PSI throttle 裁掉，只保留 token-CFS + prefix affinity。
4. 双 Coder 并行合并裁掉，保留单 Coder + Fixer。
5. summary compression 实验裁掉，保留 CVM page sharing。
6. openKylin smoke 裁掉，保留 Ubuntu smoke。

不可裁剪底线：

1. AVP 状态机。
2. 真实 worker 进程。
3. per-agent cgroup。
4. Syscall Gateway。
5. CVM page table。
6. token-CFS。
7. overlayfs 回滚。
8. 至少 3 个故障注入。
9. Dashboard 五页中的 Overview、AVP、Context、Timeline。
10. 至少 E1/E2/E3 三组实验。

---

## 13. 风险与对策

| 风险 | 对策 |
|---|---|
| cgroup/overlayfs/eBPF 依赖 root 和内核能力 | 主平台固定 openEuler 24.03 LTS，安装脚本做环境自检 |
| eBPF 工程耗时过长 | 保 execve，openat/connect 作为增强 |
| llama.cpp CPU 推理慢 | 性能实验用短输出合成负载，Demo 质量输出走 DeepSeek relay，mock 兜底 |
| prefix cache 指标不可用 | D6 优先验证 llama.cpp timing 字段；不可用时记录 TTFT/prefill 作为替代 |
| 两人开发压力大 | 前后端 mock 并行，按砍单线控制范围 |
| 被质疑只是工作流框架 | 答辩绑定真实 PID/cgroup/overlay/eBPF/cache hit 数据 |
| 故障注入影响 VM | 所有危险实验放入 capsule 和全局兜底 cgroup，实验前 VM 快照 |

---

## 14. 验收标准

1. openEuler 24.03 LTS 上可编译、安装、systemd 启动。
2. 每个 Agent 有真实 worker 进程、PID 和 cgroup。
3. Dashboard 可展示 AVP 状态、资源曲线、freeze/unfreeze/kill。
4. overlayfs 工作区隔离生效，rm -rf 注入不会污染主快照。
5. forkbomb 和 OOM 注入能被 pids.max/memory.events 捕获。
6. CVM 支持 page store、page table、CoW delta、materialize、ref count。
7. token-CFS 和 prefix affinity 有可解释 decision log。
8. llama.cpp local 能采集 prefix cache 或等价 timing 指标。
9. page reference IPC 能统计避免拷贝字节数。
10. Supervisor 能 retry、熔断、spawn Fixer。
11. checkpoint/recovery 能演示 daemon crash 后续跑。
12. Timeline 能合并应用事件、syscall audit 和至少一种 eBPF 内核事件。
13. E1/E2/E3 至少三组实验完成，图表可进入 PPT。
14. 软件工程 Demo 端到端跑通。
15. 提交包含源码、部署文档、实验报告、PPT 和演示录屏。

---

## 15. PPT 叙事建议

PPT 主线不按模块堆砌，而按问题回答。

1. 问题：多 Agent 系统为什么需要 OS 级 Runtime。
2. 核心观点（已整改）：Agent 是受用户态 Runtime 管理的执行对象，可在支持环境关联 worker/cgroup。
3. 总体架构：aortd + Agent Capsule + CVM + Scheduler + Trace。
4. 创新一：Agent Capsule，用真实 cgroup/overlayfs 隔离 Agent。
5. 创新二：CVM，把上下文变成可共享、可压缩、可审计的页。
6. 创新三：token-CFS + prefix affinity，把 OS 公平调度和 LLM cache 优化结合。
7. 创新四：Syscall Gateway + page reference IPC，统一工具、模型和通信。
8. 创新五：双层 Trace + checkpoint，提供系统级可观测与恢复。
9. Demo：软件工程任务和故障注入。
10. 实验：E1 性能、E2 容错、E3 上下文/IPC。
11. 总结：AORT-R 与普通 Workflow、AIOS 概念系统的区别。

最重要的答辩句：

> AORT-R 的创新不是“多个 Agent 互相聊天”，而是把 Agent 放进可调度、可隔离、可观测、可恢复的 OS 级执行环境，并把上下文管理与推理缓存优化纳入 Runtime 调度闭环。

---

## 16. 最终建议

采用冲高分创新路线：

1. 主线必须做硬：Agent Capsule、CVM、Syscall Gateway、token-CFS、prefix affinity、Supervisor。
2. 展示必须做亮：Dashboard 五页、Timeline 回放、故障注入、daemonkill 恢复。
3. 实验必须做实：E1/E2/E3 至少三组对比，避免只讲概念。
4. 裁剪必须有序：eBPF 和完整 checkpoint 是冲奖增强项，但不能拖垮主线。

最终项目名称建议：

> AORT-R：面向多智能体的操作系统级 Agent Runtime

最终定位建议：

> AORT-R 是一个运行在 openEuler 上的 AI Native OS Runtime 原型。它不修改 Linux 内核，而是在用户态把 Linux 已有的进程、cgroup、namespace、overlayfs、eBPF 和 systemd 能力重新组织为面向 Agent 的统一执行时，为多 Agent 系统提供调度、隔离、上下文优化、通信、容错和系统级可观测。
