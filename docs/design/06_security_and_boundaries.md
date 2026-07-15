# 06 Security and Capability Boundaries

## 安全目标

保护公共输入、正常 Agent 工作区、宿主资源、API credential 和证据完整性；将失控 Agent 或工具进程的影响限制到可识别范围，并确保失败可审计。

## 信任边界

Runtime/Gateway 属于受信控制面；Agent 与工具进程是不受信执行主体；外部模型服务是独立信任域；宿主内核与 openEuler 配置是基础设施信任根。详细关系见 [THREAT_MODEL.md](THREAT_MODEL.md)。

## 机制与边界

| 机制 | 缓解 | 不提供 |
|---|---|---|
| cgroup v2 | CPU/memory/pids 边界、统计、kill | 文件/网络语义隔离、内核漏洞防护 |
| OverlayFS workspace | 写入分层、lowerdir 保护、回滚 | 完整 mount namespace 或宿主文件系统沙箱 |
| Gateway | 命令入口、timeout、audit | 新 Linux syscall、完整 seccomp policy |
| CVM/IPC | 内容引用、计数、受控传递 | 模型 KV Cache 隔离、加密通道 |
| Timeline/evidence | 事件追踪和失败留存 | 防篡改远程证明 |
| eBPF observer | 可选内核事件增强 | 当前环境的完整观测覆盖 |

## Credential

DeepSeek Key 仅从 `DEEPSEEK_API_KEY` 读取；结果仅记录 source/present/redacted 布尔值。`internal/review/agent_demo.go` 在写 timeline、summary、final result 和 report 前执行脱敏，测试使用内存 HTTP transport 验证密钥不落盘。

## 当前缺失

没有完整 namespace、seccomp、Landlock/SELinux/AppArmor policy、MAC、容器镜像边界或 VM 隔离；没有多租户网络策略和远程证据签名。项目不能描述为完整容器安全沙箱或完整 Agent OS。

## 失败处理

平台能力失败必须记录 fallback reason。eBPF 当前为 degraded；portable 场景的 cgroup/OverlayFS 不等同已有 openEuler real-only 证据。任何安全宣称需同时给出环境、evidence_mode 和原始文件路径。
