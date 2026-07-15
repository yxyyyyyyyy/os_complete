# Defense Q&A

## 为什么只聚焦两个问题？

评审要求形成问题到证据的闭环。资源干扰和上下文重复是现有代码最完整、最可量化的两条路径；其余模块只作为支撑，避免功能清单式叙事。

## baseline 公平吗？

三模式使用相同 seed、Agent 数、上下文大小、故障轮换和 measured runs。差异由模式的数据路径/feature switch 产生，不给 baseline 人为 sleep，也不写死结果。

## cgroup 是 Linux 的，项目贡献是什么？

cgroup 不是原创。贡献是 Agent Runtime 对象到 cgroup lifecycle、资源采样、调度、workspace、fault evidence 的组合和统一场景/指标；必须同时说明 Linux 机制来源。

## 通信效率如何测？

分别计 logical、physical written、transferred 和 materialized bytes。saved=logical-transferred，由 raw counter 推导；IPC 时延记录 P50/P95。0% 共享有 transferred=logical、saved=0 的回归不变量。

## CVM 是 KV Cache 吗？

不是。CVM 管理 Runtime 层上下文页和 materialization；尚未接入真实推理引擎内部 KV Cache。

## memfd/mmap 是真正零拷贝吗？

不能这样宣称。它减少用户态重复 payload 传输，但页映射、私有数据、序列化和模型输入仍可能产生复制；项目报告只写 measured transferred/physical bytes。

## 安全边界到哪里？

cgroup 控资源，OverlayFS 控工作区写层，Gateway 做入口/超时/audit。没有完整 namespace/seccomp/MAC/VM，因此不是完整安全沙箱。

## eBPF 为什么 degraded？

当前 attach 返回失败且未观察目标 worker PID。核心场景不依赖 eBPF；Timeline 仍有 Runtime/Gateway 事件，但不会把它包装成完整内核观测。

## mock 和真实 API 各自作用？

mock 用于可重复场景和统计；DeepSeek 用于可用性验证。网络波动不进入核心性能基准；Key 只从环境读取并脱敏。

## 为什么本轮没跑真实 DeepSeek？

当前环境没有 Key，因此按规则未发请求。历史 final 有 real-api 证据，但时间和提交不同，答辩时分开说明。

## 重复次数和统计方法？

默认 warmup=3、measured=20、固定 seed；使用 population mean/std、min/max、线性插值 P50/P95 和 success rate。失败样本保留且影响 success rate。

## openEuler 证据可信度？

旧 final 记录 openEuler 24.03 LTS、kernel、cgroup2fs、root、commit 和 dirty=false，并索引 cgroup/OverlayFS/raw files。新 portable review evidence 不覆盖它，只建立引用。

## 为什么资源场景没有宣称性能提升？

当前 portable run degraded 且三模式工作量很小，时延不足以归因于 cgroup/调度。我们只结论到成功率、hash 和污染；性能结论需在 openEuler 同机重跑。
