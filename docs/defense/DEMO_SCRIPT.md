# Four-Minute Demo Script

## 0:00-0:30 说明环境

```bash
git branch --show-current
go test ./...
```

讲解：当前分支为 `codex/aort-r-upgrade`。先证明代码回归，再展示独立场景证据；本机不是 openEuler 时，OS 能力会标 degraded。

## 0:30-1:30 资源隔离

```bash
go run ./cmd/aortctl scenario resource-isolation \
  --mode all --warmup 0 --runs 1 --seed 20260713 --timeout 5s \
  --out /tmp/aort-demo-resource
jq '{evidence_mode,summary}' /tmp/aort-demo-resource/summary.json
```

讲解：同一故障轮换三模式。重点看 task success、lowerdir hash、cross-agent contamination 和 fallback reason，不把 portable degraded 冒充 real cgroup。

## 1:30-2:30 上下文复用

```bash
go run ./cmd/aortctl scenario context-sharing \
  --mode all --warmup 0 --runs 1 --agents 6 --context-size 4096 \
  --out /tmp/aort-demo-context
jq '.summary | {full:."full-copy@50".bytes_transferred,aort:."aort-r@50".bytes_transferred,saved:."aort-r@50".saved_bytes}' \
  /tmp/aort-demo-context/summary.json
```

讲解：50% 公共上下文时比较相同逻辑输入下的 transferred bytes。saved 由本次 counter 推导；CVM 不是 KV Cache。

## 2:30-3:30 六 Agent Demo

```bash
go run ./cmd/aortctl scenario real-agent-demo \
  --provider mock --seed 20260713 --out /tmp/aort-demo-agents
jq '{status,provider_actual,agents:(.agents|length),llm:(.llm_calls|length),tools:(.tool_calls|length),fault}' \
  /tmp/aort-demo-agents/summary.json
```

讲解：Router/Gateway 产生一次 LLM 和五次工具调用；Fault-Agent 失败后 Tester/Reviewer 继续。mock 是可重复验证，不说真实 API。

## 3:30-4:00 总索引

```bash
go run ./cmd/aortctl evidence review-final \
  --resource-dir /tmp/aort-demo-resource \
  --context-dir /tmp/aort-demo-context \
  --demo-dir /tmp/aort-demo-agents \
  --legacy-final-dir experiments/results/final \
  --out /tmp/aort-demo-review-final
jq '{all_required_passed,scenarios,legacy_final}' /tmp/aort-demo-review-final/REVIEW_EVIDENCE_INDEX.json
```

讲解：新索引验证三场景并只读引用旧 openEuler final。任何缺失/失败都会写失败索引并让命令非零。

安全提醒：不要在演示命令中打印环境 Key；不要对仓库根执行删除或压力命令。
