# 10 Limitations and Future Work

## 当前限制

1. Portable review resource scenario 不会把当前进程移入宿主 cgroup；真实边界依赖已有 openEuler real-only smoke。
2. CVM 没有接入 vLLM/llama.cpp/DeepSeek 内部 KV Cache，也不控制模型 prefix cache。
3. memfd/mmap 只在 Linux 支持；page reference 和 shared memory 都不宣称 kernel zero-copy。
4. eBPF attach 当前证据 degraded，不能作为完整 exec/open/connect 观测证明。
5. 主要规模是单机 6 Agent；没有分布式一致性、多租户网络和跨主机故障实验。
6. 没有完整 namespace/seccomp/Landlock/MAC/VM 隔离与远程证据签名。
7. 本轮没有环境 Key，未重复 DeepSeek real API；历史 real-api 仅证明当时可用。
8. RSS 在无 `/proc` 平台标 unsupported；runtime heap 不能替代 OS RSS。

## 后续优先级

1. 在 openEuler 上直接运行三个新 scenario，并把 cgroup create/attach/sample/kill/destroy 事件纳入同一 raw schema。
2. 接入真实推理服务的可观测 prefix/KV cache counters，在保持 provider 边界的前提下扩展 context benchmark。
3. 用 namespace + seccomp/Landlock/SELinux 建立独立安全增强阶段，并新增逃逸测试。
4. 修复 eBPF attach 兼容性，只有实际观察目标 worker PID 后才升级为 real-ebpf。
5. 增加多轮统计显著性和不同上下文大小/Agent 数量的扩展实验。

未来工作必须沿用 measured/derived/unsupported 和 evidence_mode 规则，不能用设计目标替代现有证据。
