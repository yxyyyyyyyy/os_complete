# AORT V2.0（AORT-R）需求规格与总体设计

版本：V2.0（代号 AORT-R，R = Real）
日期：2026-07-04
主平台：openEuler 24.03 LTS（x86_64，内核 6.6）
项目名：AORT, Agent-Oriented Runtime on openEuler
中文名：面向多智能体的操作系统级运行时
团队约束：2 人开发（成员 A 后端 / 成员 B 前端+实验），2026-07-18 提交，Go 后端 + Vue 前端
环境约束：有 root 权限的 openEuler VM，无 GPU；LLM 走 DeepSeek 中转 API（OpenAI 兼容）+ 本地 llama.cpp CPU 推理双轨

---

## 0. 与 V1.0 的关系

V1.0 提出了 AVP / CVM / Syscall / Interrupt / Trace 五个抽象，但全部实现为用户态 Go 数据结构，未触及任何真实 OS 机制，性能收益仅有"估算 token 数"支撑。V2.0 保留 V1.0 的骨架（Agent 抽象命名、syscall 网关、trace、Dashboard、软件工程 Demo、两人分工模式），将每个抽象落到真实内核机制或真实推理侧优化上，并以实测数字支撑性能声明。

V2.0 相对 V1.0 的变更总表：

| V1.0 机制 | V2.0 处置 | 落点 |
|---|---|---|
| AVP（数据结构） | 升级 | Agent Worker 独立进程 + 独立 cgroup v2，SUSPENDED = cgroup.freeze |
| CVM（字符串页字典） | 升级 | 内容寻址不可变页 + 字节级稳定前缀 → 推理侧 prefix cache 实测命中 |
| （无） | 新增 | Agent Capsule：cgroup 限额 + 命名空间 + overlayfs CoW 工作区 |
| Resource-Aware 加权打分调度 | 替换 | token-CFS 公平调度 + 前缀亲和批调度 + cgroup PSI 节流 |
| Interrupt Controller（独立子系统） | 降级 | 收编为运行时事件；人工审批 = 冻结 capsule |
| `aort.ipc.send`（未设计） | 新增 | 页引用零拷贝 IPC + 黑板发布订阅，UDS 传输 |
| Trace（应用层） | 升级 | 应用层 trace + eBPF 内核事件双层时间线 |
| （无） | 新增 | 检查点/恢复（含工作区快照），守护进程被杀后无损续跑 |
| Reporter 角色、8 页 Dashboard | 删减 | 砍 Reporter；Dashboard 8 页 → 5 页 |

---

## 1. 项目定位与叙事

AORT-R 是运行在 openEuler 用户态的多 Agent 操作系统级运行时。它将 Agent 提升为"操作系统一等公民"：每个 Agent 是一个真实进程，装在真实的 cgroup + 命名空间胶囊里，通过统一 syscall 网关访问 LLM、工具与彼此，由感知推理侧缓存与系统压力的调度器统一调度，故障被内核级机制隔离，全过程可在应用层与内核层双时间线上观测与回放。

一句话描述：

> AORT-R 在 openEuler 用户态实现 Agent 运行时：每个 OS 抽象（进程、虚拟内存、系统调用、调度、隔离、观测）背后都有一个真实内核机制（cgroup v2、overlayfs、namespace、PSI、eBPF）或真实推理侧优化（prefix cache 命中）支撑，性能与容错收益全部可测。

与 AIOS 的区别（答辩核心话术）：AIOS 是应用层框架加 OS 类比，其"调度器/内存管理器"是 Python 模块；AORT-R 的每个抽象由真实内核机制承载，调度决策直接影响实测的推理缓存命中率与首 token 延迟，故障隔离由 cgroup 限额与 overlayfs 回滚在内核层面保证。

## 2. 赛题要求覆盖映射

| 赛题要求 | AORT-R 对应机制 | 证据形态 |
|---|---|---|
| 基于国内主流开源 OS | openEuler 24.03 LTS 主平台，openKylin/Ubuntu smoke test | 部署脚本、截图、录屏 |
| 任务依赖、动态生成任务、资源感知调度 | DAG 依赖管理；Reviewer 动态 spawn Fixer；token-CFS + 前缀亲和 + PSI 节流 | 调度决策日志、实验 E1 |
| 统一 Agent 执行抽象、生命周期、与 OS 资源关系 | AVP 状态机；Worker 进程 + cgroup 一一对应；SUSPENDED = freeze | Dashboard AVP&Capsule 页 |
| 容错：隔离单体故障避免级联 | Agent Capsule（pids.max/memory.max/overlayfs 回滚）+ Supervisor（重试/熔断/OOM 事件） | 故障注入剧目、实验 E2 |
| 上下文复用、压缩、隔离 | CVM 内容寻址页 + CoW delta + 摘要压缩 + 前缀亲和命中 | 实验 E1/E3、Context 页 |
| 复杂多 Agent 应用场景 | 多 Agent 软件工程 Demo（含并行 Coder、故障注入） | 端到端录屏 |
| Agent 高效通信 | 页引用零拷贝 IPC + 黑板发布订阅（UDS） | 实验 E3 避免拷贝字节数 |
| 系统级可观测与控制（痛点 4） | 双层时间线（应用 trace + eBPF 内核事件）、冻结/杀死/回放/检查点 | Timeline 页、检查点演示 |

评分维度支撑：机制创新 30%（五个真机制，见第 4-8 章）；完整性与深度 25%（每机制有数据结构、接口与内核落点）；性能 20%（实验 E1/E3 实测数字）；工程质量 15%（Go + systemd + eBPF + 前后端分离）；实验 10%（3 组对比实验 + 微基准，重复 ≥5 次）。

## 3. 设计目标与非目标

### 3.1 V2.0 必须实现

1. `aortd` Go 运行时守护进程（systemd 服务，root 运行）。
2. Agent Worker 独立进程模型（同一二进制 re-exec），UDS syscall 通道。
3. Agent Capsule：cgroup v2 限额、mount/pid namespace、overlayfs CoW 工作区、合并与回滚。
4. CVM：内容寻址页、页表、materialize、CoW delta、摘要压缩。
5. LLM Router：DeepSeek 中转（OpenAI 兼容）+ 本地 llama.cpp 双 provider，缓存命中指标采集，mock/录制回放模式。
6. 调度器：FIFO（基线）、token-CFS + 前缀亲和 + PSI 节流（主打）。
7. Syscall 网关：capability、配额、超时、审计。
8. IPC：页引用传递 + 黑板发布订阅。
9. Supervisor：重试、熔断、cgroup OOM 事件响应、动态 spawn。
10. 检查点/恢复：状态快照 + 工作区快照，崩溃续跑。
11. Trace：应用事件 + eBPF 内核事件（execve/openat/connect），时间线回放。
12. Web Dashboard 5 页。
13. 多 Agent 软件工程 Demo + 4 个故障注入剧目。
14. 实验 E1/E2/E3 + 微基准，openEuler 部署文档。

### 3.2 非目标（写入未来工作）

1. 不修改 Linux 内核。
2. 不做 seccomp/landlock 细粒度系统调用过滤（未来工作）。
3. 不做 GPU/vLLM 推理（本项目无 GPU；接口预留 provider 扩展点）。
4. 不做多机分布式调度。
5. 不做完整 MCP 协议兼容。
6. 不主攻 OpenHarmony 适配。
7. 不训练或微调模型。

## 4. 总体架构

```text
                Web Dashboard (Vue 3 + Vite + ECharts, 5 页)
                        REST + SSE (/api/events)
                              |
  aortd  (Go >= 1.22, systemd service, root/CAP_SYS_ADMIN)
  ├─ AVP Manager        生命周期状态机、DAG 依赖
  ├─ Capsule Manager    cgroup v2 + namespace + overlayfs
  ├─ CVM Manager        内容寻址页存储、页表、materialize、压缩
  ├─ Scheduler          FIFO / token-CFS + 前缀亲和 + PSI 节流
  ├─ Syscall Gateway    capability、配额、超时、审计（UDS JSON-RPC）
  ├─ IPC & Blackboard   页引用传递、topic 发布订阅
  ├─ Supervisor         重试、熔断、OOM 事件、动态 spawn
  ├─ Checkpoint         状态 + 工作区快照、崩溃恢复
  ├─ Trace Recorder     应用事件 + 内核事件、JSONL + bbolt、回放
  ├─ eBPF Observer      execve/openat/connect per cgroup (cilium/ebpf, CO-RE)
  └─ LLM Router         deepseek-relay / llamacpp-local / mock
        |  (UDS /run/aort/aortd.sock)
  Agent Worker 进程（每个在自己的 capsule cgroup 内）
  Planner / Coder x2 / Tester / Reviewer / Fixer（Fixer 动态 spawn）
        |  (由 aortd 代理执行)
  工具沙箱进程（嵌套 ns + overlayfs 工作区）: shell.exec / file.* / git.diff / go.test / http.request
```

核心原则：

1. Agent Worker 是真实独立进程：有 PID、有 cgroup、可冻结、可杀死、可从 cgroup 文件读实时资源统计。
2. Worker 无任何直接外部访问能力：文件、工具、LLM、IPC 全部经 UDS syscall 由 aortd 代理执行并审计。
3. 所有关键行为进 trace；Dashboard 只做展示与控制，不承载调度逻辑。
4. 单机单守护进程；状态持久化于 bbolt + JSONL WAL。

目录约定：

```text
/etc/aort/config.yaml            全局配置
/run/aort/aortd.sock             syscall UDS
/var/lib/aort/pages/             内容寻址页存储（sha256 前缀分桶）
/var/lib/aort/snapshots/<n>/     项目快照链（只读 lower）
/var/lib/aort/capsules/<avp>/    upper/ work/ merged/ 每胶囊 overlay 目录
/var/lib/aort/state.db           bbolt
/var/lib/aort/traces/<task>.jsonl
/var/lib/aort/checkpoints/<task>/
/sys/fs/cgroup/aort.slice/avp-<id>/   每 AVP cgroup
```

## 5. AVP 与 Agent Capsule

### 5.1 AVP 数据结构与状态机

```go
type AVP struct {
    AgentID       string
    TaskID        string
    Role          string        // planner|coder|tester|reviewer|fixer
    State         AgentState
    Weight        int           // token-CFS 权重，默认 100
    VRuntime      uint64        // 累计 token / weight
    ParentAgentID string
    Dependencies  []string      // DAG 前驱 AgentID
    PageTable     []PageMount   // 有序上下文页表
    Capabilities  []string      // 允许的 syscall/工具名
    Capsule       CapsuleSpec   // cgroup 限额 + 工作区配置
    RetryCount    int
    PID           int
    CgroupPath    string
    CreatedAt, StartedAt, FinishedAt time.Time
}
```

状态机（V1.0 保留，语义升级）：

```text
CREATED -> READY -> RUNNING
RUNNING -> WAITING_LLM | WAITING_TOOL | WAITING_IPC -> READY
RUNNING -> SUSPENDED（= 写 cgroup.freeze=1，真实冻结）-> READY（解冻）
RUNNING -> COMPLETED
RUNNING -> FAILED -> READY（重试，换新 capsule）| KILLED（超重试上限）
```

### 5.2 Capsule 生命周期

创建 AVP 时 Capsule Manager 执行：

1. `mkdir /sys/fs/cgroup/aort.slice/avp-<id>`，写入 `cpu.max`（默认 "100000 100000"，即 1 核）、`memory.max`（默认 256M）、`pids.max`（默认 64）；另建嵌套子组 `avp-<id>/tools/`，独立限额（默认 memory.max 512M、pids.max 64），工具故障的爆炸半径与 Worker 本体隔离。
2. 以 fork+exec 启动 `aortd worker --avp-id <id>`：子进程 exec 前经管道与父进程同步，父进程先将其 PID 写入 `cgroup.procs` 再放行。
3. Worker 通过 UDS 连接 aortd 完成注册握手（携带 avp-id + 一次性 token 防伪造）。

工具执行（`tool.exec` syscall）时，aortd 起嵌套沙箱子进程：

1. `unshare(CLONE_NEWNS | CLONE_NEWPID)`，设置 no-new-privs。
2. 挂载 overlayfs：`lowerdir=/var/lib/aort/snapshots/<latest>`（只读项目快照），`upperdir=/var/lib/aort/capsules/<avp>/upper`，工作目录切到 merged 视图。
3. 私有 /tmp；工具进程加入该 AVP 的 `tools/` 嵌套 cgroup（限额独立于 Worker 本体，OOM 只杀工具不杀 Worker）。

### 5.3 工作区收编与回滚

1. 收编：Reviewer 批准后，aortd 将该 AVP 的 upper 层以 rsync 复制方式合并生成新快照 `snapshots/<n+1>`，作为后续 Agent 的 lower。
2. 并行合并：两个 Coder 的 upper 层触碰文件集合不相交时可先后合并；出现路径交集则产生 `workspace.conflict` 事件，路由给 Fixer 处理。
3. 回滚：FAILED/KILLED 的 AVP 直接删除 upper 目录，主快照零污染。

### 5.4 故障隔离行为（内核级）

| 故障注入 | 内核机制反应 | Supervisor 反应 |
|---|---|---|
| 工具 fork 炸弹 | pids.max 拒绝 fork，工具报错退出 | 记 PIDS_LIMIT，换新 capsule 重试 |
| 工具内存泄漏 | capsule 内 OOM kill（memory.events oom_kill+1） | 监听 memory.events，记 CAPSULE_OOM，重试 |
| 工具 rm -rf 项目 | 只删除 overlay upper 层 | 丢弃 upper，回滚重试 |
| Worker 崩溃 | 进程退出，cgroup 清空 | AGENT_CRASH，按策略重试或 KILLED |

## 6. CVM 与 LLM Router

### 6.1 上下文页与页表

```go
type Page struct {
    ID      string  // sha256(content)，内容寻址、不可变
    Kind    string  // system|project|task|delta|summary
    Bytes   []byte
    Tokens  int     // 经 llama.cpp /tokenize 或 len/3 启发式估算
    Meta    map[string]string
}
type PageMount struct { PageID, Role string; Seq int }
```

规则：

1. 页不可变；Agent"修改"共享上下文 = 追加私有 delta 页（CoW）。
2. Materialize 按规范顺序拼接：system → project（按 Seq）→ task → delta（按创建序）。同一页集合的 Agent 产生字节级相同 prompt 前缀，这是 prefix cache 命中的前提。
3. 页表 token 总量超预算（本地引擎默认 8192）时，最老的 delta 页序列经一次 LLM 调用压缩为 summary 页替换之，原页保留于页存储供审计。
4. 引用计数：每页记录被多少 AVP 挂载，Dashboard 展示共享结构。

### 6.2 LLM Router

```go
type Provider interface {
    Complete(ctx, req) (Resp, Usage)  // Usage 含 PromptTokens/CachedTokens/TTFTms
}
```

三个 provider：

1. `llamacpp-local`：llama-server（`--parallel 2 --cache-reuse 256`，`cache_prompt: true`），模型 Qwen2.5-Coder-1.5B-Instruct Q4_K_M GGUF；从响应 timings 读取 `prompt_n` 与缓存复用 token 数。承担全部性能实验（可控、免费、可复现）。
2. `deepseek-relay`：OpenAI 兼容中转；若 usage 透传 `prompt_cache_hit_tokens/prompt_cache_miss_tokens` 则记录为第二组数据（D5 验证，不透传不影响主线）。承担 Demo 质量输出。
3. `mock`：录制/回放固定响应，用于开发、联调与答辩兜底。

路由策略：按 AVP 角色配置默认 provider，Supervisor 可在故障时切换（fallback model）。

## 7. 调度器

三层决策，每次调度输出 decision log（含被选者、分组、原因、当时指标）：

1. 公平层（token-CFS）：`vruntime += 本次消耗 token / weight`；从 READY 集合选 vruntime 最小者。防止单 Agent 在 token 预算/速率限制下饿死他人。
2. 亲和层（前缀亲和批调度）：将 READY AVP 按页表最长公共前缀（页 ID 序列）分组；若存在与"上一次派发同槽位"前缀相同的候选，且其 vruntime 与全局最小值差 ≤ K（默认 2000），则优先派发该候选到同一推理槽位以命中 prefix cache。K 是公平与缓存局部性的显式权衡参数。
3. 资源层（PSI 节流）：读 `/sys/fs/cgroup/aort.slice/cpu.pressure` 与 `memory.pressure`，avg10 超阈值（默认 40）时将工具沙箱并发上限从 2 降为 1，恢复后回升。

基线调度器：FIFO、无亲和轮转（实验对照用，配置项切换）。

## 8. Syscall、IPC、Supervisor、检查点、Trace

### 8.1 Syscall 网关

传输：UDS JSON-RPC 2.0。统一管线：capability 检查 → 配额（token/次数）→ 超时设置 → 审计开始 → 执行 → 故障钩子 → 审计结束 → trace 记录。

syscall 清单：

```text
context.read / context.write(delta) / context.materialize
llm.call
tool.exec        // 工具名限于 capability: file.read file.write shell.exec git.diff go.test http.request
ipc.publish / ipc.poll
agent.spawn      // 动态生成任务（Reviewer -> Fixer）
agent.report     // 提交阶段结果与状态
```

审计字段沿用 V1.0（syscall_id、agent_id、耗时、权限结果、exit_code、error_type、output_ref）。

### 8.2 页引用零拷贝 IPC

1. `ipc.publish(topic, page_id)`：内容已在页存储，仅发布引用。
2. 订阅方 `ipc.poll(topic)` 收到页引用后挂载进自己的页表；内容全程只存一份。
3. 统计指标：消息数、引用传递避免拷贝的字节数（= 页大小 × (订阅者数-1)）、端到端延迟。

### 8.3 Supervisor

故障分类：`LLM_ERROR / TOOL_TIMEOUT / TOOL_EXIT_NONZERO / CAPSULE_OOM / PIDS_LIMIT / WORKSPACE_CONFLICT / CONTEXT_OVERFLOW / AGENT_CRASH / RETRY_LIMIT`。

策略：重试（默认上限 2 次，每次换全新 capsule + 干净 upper）；provider 降级切换；上下文压缩（CONTEXT_OVERFLOW 时触发 6.1 规则 3）；动态 spawn Fixer（由 Reviewer 通过 agent.spawn 发起）；熔断器按 (工具名 | provider) 维度计数，open → half-open → close；连续失败暂停任务待人工（冻结相关 capsule）。

OOM 检测实现：inotify 监听各 capsule 的 `memory.events` 文件，oom_kill 计数增加即产生故障事件。

### 8.4 检查点与恢复

1. 检查点分两级：轻量级（AVP 表 + 页表引用 + DAG 状态 + 调度器 vruntime，bbolt 事务写入）在每次 syscall 完成的步骤边界执行；重量级（追加各 capsule upper 层 tar 包）仅在 DAG 节点完成时执行。
2. 恢复：aortd 启动时发现未完成任务 → 取最近重量级检查点解包 upper → 用其后的轻量级状态推进 → 重建 AVP（新进程、新 cgroup）→ 从最近步骤边界续跑。
3. 演示：任务中途 `kill -9 aortd`，systemd `Restart=always` 拉起，任务续跑完成。

### 8.5 Trace 与 eBPF Observer

应用层事件沿用 V1.0 清单（task/agent/scheduler/context/syscall/llm/supervisor 事件），持久化为每任务 JSONL + bbolt 索引；回放 = 按原时间轴经 SSE 重放。

eBPF Observer（Go + cilium/ebpf，CO-RE，内核 6.6）：

1. 挂载点：tracepoint `sched:sched_process_exec`（execve）、`syscalls:sys_enter_openat`、`syscalls:sys_enter_connect`。
2. 过滤：仅保留 cgroup id 属于 aort.slice 子树的进程；用户态按 cgroup path → AVP 映射归属。
3. 事件经 ring buffer 送入 Trace，`source=kernel`，与应用层事件同轴展示。
4. 降级路径：进度紧张时只保留 execve（见第 14 章砍单线）。

## 9. Dashboard（5 页）

1. Overview + DAG：任务总览、DAG 图（含动态生成节点高亮）、当前调度策略、成功率、总耗时。
2. AVP & Capsule：状态机视图、每 Agent 实时 CPU/内存曲线（读 cgroup stat 文件）、重试次数、冻结/解冻/杀死操作按钮。
3. Context Memory：页列表、引用计数、页共享关系图、materialize 记录、cache 命中 token 数曲线。
4. Timeline：应用事件 + 内核事件 + syscall 审计三泳道合并时间线，支持按 Agent/类型过滤与整段回放。
5. Experiments：E1/E2/E3 图表（ECharts），支持从实验 JSON 结果一键渲染。

技术栈：Vue 3 + Vite + ECharts + SSE。开发期以 mock 数据先行（D1 冻结 API schema）。

## 10. Demo 场景

任务输入沿用 V1.0："实现一个简单 Todo Web API（Go），支持创建、查询、完成 Todo，并提供基础单元测试。"

角色：Planner、Coder×2（并行，负责不相交模块：如 handlers 与 storage）、Tester、Reviewer、Fixer（动态）。

主线剧本：

```text
Planner 建 DAG（拆分两个 Coder 子任务）
-> 双 Coder 并行（各自 capsule + overlay upper，展示前缀亲和调度命中 cache）
-> 依次收编工作区 -> Tester 在沙箱跑 go test
-> Reviewer 审查；失败则 agent.spawn Fixer -> Fixer 修复 -> Tester 复测
-> 任务完成，Timeline 回放全程
```

故障注入剧目（Dashboard 一键触发，`POST /api/demo/fault/{kind}`）：

1. `forkbomb`：Tester 工具内注入 fork 炸弹 → pids.max 掐死 → 重试成功。
2. `oom`：Coder 工具内存暴涨 → capsule 内 OOM → memory.events 捕获 → 重试。
3. `rmrf`：注入误删 → 仅 upper 层受损 → 回滚重试，主快照无损。
4. `daemonkill`：`kill -9 aortd` → systemd 拉起 → 检查点续跑。

## 11. API 摘要

```text
POST /api/tasks                  创建任务
GET  /api/tasks /api/tasks/{id} /api/tasks/{id}/dag
GET  /api/agents /api/agents/{id}            含 cgroup 实时统计
POST /api/agents/{id}/freeze|unfreeze|kill   控制操作
GET  /api/context/pages
GET  /api/syscalls
GET  /api/traces/{task_id}       支持 ?replay=1
GET  /api/experiments/{name}     实验结果 JSON
POST /api/demo/run
POST /api/demo/fault/{kind}
GET  /api/events                 SSE 事件流
```

## 12. 实验设计

通用要求：每组重复 ≥5 次，报告均值±标准差；固定随机种子与 prompt；说明环境（VM 规格、内核、模型量化、llama.cpp 版本）、指标定义、失败案例分析。性能实验统一使用 `llamacpp-local`（可控可复现），API 侧数据如可得则作为附加验证。

E1 调度性能（对应性能分 20% 的主证据）：

- 负载：合成工作负载 8 个 Agent，共享同一 project 页集，各带小私有 delta。
- 对照：FIFO / 无亲和轮转 / token-CFS+前缀亲和。
- 指标：总墙钟时间、prefill 耗时合计、实测缓存复用 token 数（llama.cpp timings）、公平性（各 Agent 完成 token 的 Jain 指数）、决策日志样例。
- 预期：CPU 推理 prefill 慢，前缀命中带来的 TTFT 与总时长下降显著。

E2 隔离容错：

- 对照：per-agent capsule 关闭（仅全局兜底 cgroup，防实验毁 VM）vs 开启。
- 注入：forkbomb / oom / rmrf 三类，各 5 次。
- 指标：爆炸半径（兄弟 Agent 受损数）、故障恢复时间、任务最终成功率、oom_kill 计数。

E3 上下文与通信：

- 对照：全量拷贝上下文传递 vs CVM 页引用；压缩开/关。
- 指标：任务总 prompt token、去重节约 token、IPC 避免拷贝字节数、上下文页数与引用计数。

微基准：syscall 网关往返开销（μs 级分布）、检查点/恢复耗时、eBPF 采集对工具执行的开销百分比。

## 13. openEuler 适配与交付物

1. openEuler 24.03 LTS 完整测试（内核 6.6：cgroup v2、overlayfs、eBPF CO-RE、PSI 均满足）；22.03 LTS 可运行但 eBPF 工具链受限，文档注明。
2. systemd unit（`Restart=always`，root 运行）+ 安装脚本（含 llama.cpp 编译、模型下载、cgroup v2 检查）。
3. openKylin 或 Ubuntu 编译启动 smoke test。
4. 交付：源码仓库、部署文档、实验报告、PPT、演示录屏。

## 14. 两人分工与 14 天计划（D1 = 7 月 5 日）

成员 A：aortd 全部后端模块 + openEuler 部署。成员 B：Dashboard、llama.cpp 环境、Demo 剧本、实验脚本与报告、PPT/录屏。协作原则沿用 V1.0：D1 冻结 API schema，前端 mock 先行，后端每日出可运行版本。

```text
D1  规格冻结、仓库脚手架、API schema、CI     A: aortd 骨架+AVP 状态机      B: Vue 脚手架+mock
D2  A: Capsule v1（cgroup 限额+worker re-exec+freeze）                    B: AVP&Capsule 页
D3  A: overlayfs 工作区+工具沙箱+收编回滚                                  B: Overview+DAG 页
D4  A: CVM 页存储+页表+materialize+压缩                                    B: Context 页
D5  A: LLM Router（中转+llama.cpp+指标采集+mock）  B: VM 编译 llama.cpp、下模型、吞吐基准、验证中转 cache 字段
D6  A: syscall 网关+UDS+IPC 黑板                                           B: Timeline v1
D7  A: 调度器（FIFO/token-CFS/前缀亲和/PSI/决策日志）                       B: cache 命中图表
D8  A: Supervisor（重试/熔断/OOM 事件/agent.spawn）                        B: 故障演示 UI+实验页
D9  A: 检查点/恢复                                                         B: 回放 UI
D10 A: eBPF Observer                                                       B: 内核事件泳道
D11 联调：Demo 端到端 + 4 个故障剧目
D12 实验脚本化运行与数据采集（E1/E2/E3+微基准）
D13 openKylin/Ubuntu smoke、文档、图表、PPT
D14 录屏、答辩彩排、buffer 与提交
```

砍单线（进度落后按序放弃，不影响主线叙事）：

1. eBPF 的 openat/connect（保 execve）。
2. 检查点的工作区 tar 快照（保运行时状态快照）。
3. 并行双 Coder（退单 Coder，保 overlay 隔离演示）。
4. E3 压缩开关对比（保页引用对比）。

## 15. 风险与对策

| 风险 | 对策 |
|---|---|
| cgroup/overlayfs/eBPF 需 root 与新内核 | 已确认 root VM；要求 24.03 LTS；安装脚本内置环境自检 |
| llama.cpp CPU 推理慢拖累演示 | 实验用短输出合成负载；Demo 质量输出走中转 API；mock 兜底 |
| 中转不透传 cache 字段 | 性能主数据来自本地引擎，D5 早验证 |
| 系统编程工作量超预期 | 第 14 章砍单线；capsule 用直接写 cgroupfs + unshare 的最简实现，不引入容器运行时 |
| 被质疑与 AIOS 同质 | 答辩话术锚定"真实内核机制 + 实测数字"差异（第 1 章） |
| 故障注入实验毁掉 VM | E2 基线组保留全局兜底 cgroup；实验前打 VM 快照 |

## 16. 验收标准

1. openEuler 24.03 上可编译、systemd 启动、运行完整 Demo。
2. 每个 AVP 对应真实进程与 cgroup，Dashboard 可见实时资源曲线，可冻结/解冻/杀死。
3. overlayfs 工作区隔离生效：rm -rf 注入后主快照无损，回滚重试成功。
4. fork 炸弹与 OOM 注入被 capsule 限额拦截，Supervisor 捕获 memory.events 并恢复。
5. CVM 页共享 + 前缀亲和调度产生可测的缓存复用 token 数（llama.cpp timings 佐证）。
6. token-CFS 决策日志可见，公平性指标可计算。
7. 页引用 IPC 工作，避免拷贝字节数可统计。
8. kill -9 aortd 后检查点续跑成功。
9. Timeline 同屏展示应用层与 eBPF 内核层事件并支持回放。
10. E1/E2/E3 + 微基准数据齐备，图表进 PPT。
11. openKylin 或 Ubuntu smoke test 通过，交付物齐全。

## 17. 未来工作

seccomp/landlock 细粒度沙箱；GPU + vLLM prefix cache 深度控制与驱逐策略；多机分布式调度；MCP 协议兼容；OpenHarmony 适配。

## 18. 参考

1. AIOS: LLM Agent Operating System, arXiv:2403.16971
2. MemGPT: Towards LLMs as Operating Systems, arXiv:2310.08560
3. vLLM Automatic Prefix Caching 设计文档
4. Linux 内核文档：cgroup v2、overlayfs、PSI、BPF CO-RE
5. DeepSeek API 文档（prompt cache 计费字段）
6. llama.cpp server 文档（cache_prompt 与 timings）
