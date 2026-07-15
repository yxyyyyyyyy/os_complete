# Ten-Slide Defense Outline

## 1. AORT-R：面向多 Agent 的 openEuler 用户态执行时

页面文字：AORT-R 聚焦多 Agent 资源/故障控制与上下文复用/通信优化。它运行在用户态，复用 Linux 机制，不修改内核。所有结论由场景化实验和 raw evidence 支撑。当前能力边界主动公开。

图表/数据：标题图使用两条主线，不放六大创新图标。数据源：`docs/design/01_problem_definition.md`。讲解备注：先说明整改后的定位，避免从 AVP/CVM 术语开场；强调“用户态原型”和“证据优先”。评委追问：是不是 Agent OS？回答：不是完整 OS，是 OS-inspired Runtime。

## 2. 我们选择的两个问题

页面文字：问题一是资源竞争、故障扩散与工作区污染。问题二是公共上下文重复写入、传输和 materialization。cgroup、OverlayFS、memfd/mmap 是已有 OS 技术。项目贡献是 Agent Runtime 的组合、策略、接口和验证闭环。

图表/数据：左右两列问题-指标映射。数据源：`docs/review_remediation/REVIEW_TO_TASK_MATRIX.md`。讲解备注：解释为何收敛，其他模块只作为支撑。评委追问：为何不是六个创新？回答：评审要求聚焦并可量化。

## 3. 两个真实场景

页面文字：资源场景固定 6 角色与四类受控故障。上下文场景固定 6 Agent、三模式、四档公共比例。每次运行有独立 run id、seed、timeout 和 raw evidence。失败也保留，不从统计中删除。

图表/数据：两个场景泳道。数据源：两个 scenario `summary.json` 的 parameters/per_run。讲解备注：强调 baseline 工作量相同、没有人为 sleep。评委追问：危险 rm-rf 如何保证安全？回答：generated-root-only guard 和自动测试。

## 4. 总体架构与两条路径

页面文字：aortctl/internal-review 是场景控制面。worker/capsule/workspace 构成资源路径。CVM/IPC/scheduler 构成上下文路径。Gateway/Timeline/evidence 连接两条路径。

图表/数据：复用 `docs/design/03_architecture_overview.md` Mermaid。数据源：`cmd/aortctl/main.go`, `internal/experiment/review_scenarios.go`。讲解备注：区分控制面和数据面；AVP/Gateway 都是用户态抽象。评委追问：哪些是真实 OS？按 evidence_mode 回答。

## 5. 资源隔离与故障控制设计

页面文字：三模式只改变边界、调度和 workspace feature。每轮测 lowerdir hash、污染、完成率与故障时延。所有压力有上限，所有删除位于自动临时根。平台不支持时明确 degraded。

图表/数据：故障检测-清理-恢复序列和路径安全框。数据源：`internal/review/resource_isolation.go`。讲解备注：不把便携场景说成真实 cgroup；真实 openEuler 证据单独展示。评委追问：隔离是否完整？回答：不是，见威胁模型。

## 6. 资源对比实验

页面文字：60 个 measured runs，三模式 success rate 均为 1.0。aort-r lowerdir unchanged=1，cross-agent contamination=0。便携运行 evidence_mode=degraded，因此不宣称 cgroup 性能收益。历史 openEuler final 证明 real-cgroup-v2 与 real-overlayfs。

图表/数据：三模式 success/P50/P95 表 + openEuler evidence badges。来源：resource summary 和 legacy final。讲解备注：把 portable 与 real-only 分成两个证据层。评委追问：为什么不比较性能提升？回答：当前环境不足以做因果归因。

## 7. 上下文复用与通信设计

页面文字：full-copy 复制完整上下文。shared-ipc 共享公共 payload。aort-r 使用 CVM 页、page reference 和 Prefix Affinity。CVM 不等于模型 KV Cache，memfd/mmap 不等于 kernel zero-copy。

图表/数据：公共页/私有页数据流和四个 byte counter。来源：`docs/design/05_context_sharing_design.md`。讲解备注：解释 logical、physical、transferred、materialized 的区别。评委追问：saved 如何算？回答：logical-transferred，由 raw counter 推导。

## 8. 通信效率量化实验

页面文字：240 个 measured observations。0% 共享时 aort-r transferred=full-copy=24576，saved=0。共享比例增加时 aort-r transferred 依次为 18496、12352、6208。50% 时 Prefix Affinity=5 hits/run。

图表/数据：四档 transferred bytes 分组柱状图，不画固定提升百分比。来源：context summary/comparison CSV。讲解备注：只结论到 Runtime 数据路径。评委追问：是否含模型调用？回答：不含，这正是边界。

## 9. openEuler + 多 Agent + DeepSeek Demo

页面文字：mock Demo 产生 6 Agent、1 LLM、5 tools 和一次故障后继续。DeepSeek 模式只读环境 Key并脱敏。本轮环境无 Key，真实 API 未运行。历史 final 有 real-api 和 openEuler 证据，时间/提交单独标注。

图表/数据：Timeline 事件条带 + provider status。来源：real-agent-demo summary/timeline 和 legacy final。讲解备注：诚实区分 mock 性能复现与真实 API 可用性。评委追问：为何不现场联网？回答：网络不是核心 benchmark，现场可按环境选择。

## 10. 贡献、边界与下一步

页面文字：贡献一是两条问题驱动闭环。贡献二是统一场景、统计和 review evidence。贡献三是安全边界与降级状态可审计。下一步是在 openEuler 重跑新场景并接入模型缓存计数和更强沙箱。

图表/数据：可以说/谨慎说/不能说三栏。来源：`CLAIMS_BOUNDARY.md`, `THREAT_MODEL.md`。讲解备注：主动承认 eBPF degraded、无 KV Cache、无完整 sandbox。评委追问：距离产品化还差什么？回答：隔离强化、分布式规模、真实模型计数和签名证据。
