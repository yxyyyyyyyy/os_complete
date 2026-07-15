# Eight-Minute Speech

## 0:00-0:45 定位

我们没有继续把功能越堆越多，而是从评审意见出发，选择两个问题：多 Agent 并发时怎样限制资源和工作区故障扩散；多个 Agent 共享上下文时怎样减少重复写入、传输和 materialization。AORT-R 是 openEuler/Linux 上的用户态 Agent Runtime 原型，不修改内核，也不声称完整 Agent OS。

## 0:45-1:30 两个场景

资源场景有 Planner、两个 Coder、Tester、Reviewer 和 Fault-Agent，轮换内存、进程、CPU 与受控删除故障。上下文场景使用相同 6 Agent，在 0、25、50、75% 公共比例下比较 full-copy、shared-ipc 和 aort-r。每个场景默认预热 3 次、测量 20 次、固定 seed，并保留失败 raw evidence。

## 1:30-2:20 总体架构

控制面是 aortctl 和 internal/review，负责参数、模式、超时、清理和 schema；数据面复用已有 worker、cgroup capsule、workspace、CVM、IPC、scheduler、Gateway 和 LLM Router。AVP 是运行时执行对象，不是 Linux 新进程类型；Gateway 是受控入口，不是新增内核 syscall。

## 2:20-3:20 资源隔离设计

三种模式使用相同 workload。每次运行创建独立临时根，删除前做绝对路径和相对路径校验，拒绝根目录、父目录和逃逸路径。我们记录正常 Agent 完成率、故障范围、P50/P95、资源峰值、检测/清理/恢复时延，以及 lowerdir 前后 hash 和跨 Agent 污染。当前 portable run 为 degraded，60 个 measured runs 均成功，aort-r lowerdir unchanged 为 1、污染为 0；真实 cgroup/OverlayFS 另由 openEuler final 证明。

## 3:20-4:20 上下文设计

full-copy 为每个 Agent 写完整副本；shared-ipc 用 memfd/mmap 共享公共部分；aort-r 再加入 CVM 内容页、page reference 和 Prefix Affinity。我们不只写“节省”，而是分别计 logical、physical、transferred、materialized。saved 等于 logical 减 transferred，0% 共享时必须为 0，这个不变量有自动测试。

## 4:20-5:15 上下文结果

每个 Agent 4096 bytes，6 Agent，240 个 measured observations。full-copy 在每档都是 24576 transferred bytes；aort-r 在 0、25、50、75% 时分别为 24576、18496、12352、6208，derived saved 为 0、6080、12224、18368。50% 时首个 Agent 后的 5 个 Agent 各产生一次 Prefix Affinity 命中。它说明 Runtime 上下文复用，不说明模型 KV Cache。

## 5:15-6:05 Demo

独立 Demo 复用 CVM、Router 和 Gateway。Planner materialize context 并发起一次 llm.call；五次工具调用中 Fault-Agent 的 `false` 命令失败，后续 Tester 和 Reviewer 继续。本轮 mock passed，可离线复现。DeepSeek 只有环境变量启用且 Key 存在才运行；当前环境没有 Key，所以不冒充本轮真实 API，历史 final 的 real-api 只作为已有可用性证据。

## 6:05-6:55 安全与边界

cgroup 控制资源，OverlayFS 保护工作区写层，Gateway 控入口/超时/audit，Timeline 保留失败证据。但我们没有完整 namespace、seccomp、MAC 或 VM，也没有真实模型 KV Cache 和完整 eBPF attach。eBPF 当前必须说 degraded。page reference 和 memfd/mmap 也不称 kernel zero-copy。

## 6:55-8:00 贡献与收口

本轮贡献是把现有机制组织成两个问题驱动的闭环，新增三条正式场景命令、统一统计与 evidence schema、威胁模型和答辩材料，并提供 review-final 总索引。所有结果能回到 raw JSON、CSV、代码和环境字段；不支持的能力明确 degraded/unsupported。下一步是在同一 openEuler 主机重跑新场景并接入真实推理引擎可观测缓存计数。
