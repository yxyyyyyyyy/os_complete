# 02 Scenarios and Requirements

## 场景一：资源隔离

角色固定为 Planner、Coder-A、Coder-B、Tester、Reviewer、Fault-Agent。Fault-Agent 轮换 memory hog、pids hog、CPU hog 和受控 workspace rm-rf；所有压力有上限，删除目标必须位于本次 `os.MkdirTemp` 根目录下。

比较模式：

| 模式 | 资源边界 | 调度 | 工作区/恢复 |
|---|---|---|---|
| baseline | 无独立 Agent cgroup 声明 | 普通执行 | 仅临时目录安全边界 |
| isolation-only | 独立边界语义；平台不支持时 degraded | FIFO | 独立临时工作区 |
| aort-r | 支持环境使用现有 cgroup/采样能力 | resource-aware | 现有 OverlayFS 路径或 degraded-copy + cleanup |

必需输出：正常 Agent 完成率、任务成功率、fault containment scope、正常完成 P50/P95、memory/pids 峰值、检测/清理/恢复时延、lowerdir 哈希和跨 Agent 污染标志。

## 场景二：上下文共享

6 个 Agent 各自看到固定大小的逻辑上下文，其中公共比例为 0%、25%、50%、75%。模式为 full-copy、shared-ipc、aort-r。

| 模式 | 数据路径 | 调度 |
|---|---|---|
| full-copy | 每个 Agent 独立写入/传输完整上下文 | 无亲和性 |
| shared-ipc | 公共内容经 memfd/mmap；私有内容独立传输 | 无 CVM 页调度 |
| aort-r | CVM 公共页 + 私有页 + page reference | Prefix Affinity |

必需输出：logical/physical/transferred/materialized/saved bytes、shared/private pages、page hit ratio、IPC P50/P95、RSS 或 unsupported、总完成时间、吞吐、等待、公平性和 Prefix Affinity 命中。

## Demo 场景

`aortctl scenario real-agent-demo --provider mock|deepseek` 必须通过现有 Gateway 产生 context、`llm.call` 和至少三次 `tool.exec`。一个受控工具失败后其他 Agent 继续。mock 用于离线重复；DeepSeek 仅用于可用性验证且 Key 只来自环境。

## 证据需求

所有场景写 `raw/`, `summary.json`, `comparison.csv`, `report.md`。公共 schema 位于 `internal/review/metrics.go`，字段包含版本、run id、时间、Git、环境、seed、warmup、measured runs、失败原因、统计量和 artifact paths。

验收不以“命令退出 0”代替证据检查；JSON 必须可解析、CSV 列稳定、报告引用存在，且 `passed` 不得与失败运行矛盾。
