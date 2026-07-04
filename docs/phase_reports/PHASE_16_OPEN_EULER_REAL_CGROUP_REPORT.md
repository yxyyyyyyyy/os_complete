# PHASE_16_OPEN_EULER_REAL_CGROUP_REPORT

mode=blocked-by-remote-cgroup-legacy

本报告记录 AORT-R 在远程 openEuler 服务器上尝试获取 real cgroup v2 capsule
证据的真实结果。本报告不把 degraded 结果伪装为 real。

## 1. 远程环境

| 项目 | 结果 |
| --- | --- |
| Git commit | `9029ec3` |
| OS | `openEuler 24.03 (LTS)` |
| Kernel | `6.6.0-112.0.0.104.oe2403.x86_64` |
| 用户 | `uid=0(root)` |
| Go | `go version go1.22.12 linux/amd64` |
| systemd | `systemd 255`, `default-hierarchy=legacy` |

## 2. 已执行命令

```bash
git -C /root/aort-r-smoke-git pull --ff-only
git -C /root/aort-r-smoke-git rev-parse --short HEAD
export GOTOOLCHAIN=go1.22.12
export GOPROXY=https://goproxy.cn,direct
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
GOCACHE="$PWD/.cache/go-build" go test ./...
```

## 3. 真实结果

当前 commit `9029ec3` 已在远程 openEuler 上执行 `go test ./...`，退出码为 `0`。
完整输出见：

```text
experiments/results/openeuler_smoke/go_test_latest.txt
```

`scripts/check_openeuler_env.sh` 已生成文本和 JSON 双格式证据。关键结果：

```text
stat -fc %T /sys/fs/cgroup: tmpfs
[FAIL] expected cgroup2fs
[FAIL] /sys/fs/cgroup is not writable
[FAIL] failed to create /sys/fs/cgroup/aort.slice
failures=3 warnings=4
```

`experiments/results/openeuler_smoke/env_check.json` 关键字段：

```json
{
  "evidence_mode": "degraded",
  "cgroup": {
    "fs_type": "tmpfs",
    "is_cgroup2fs": false,
    "writable": false,
    "aort_slice": "failed"
  },
  "failures": 3,
  "warnings": 4
}
```

`scripts/smoke_openeuler.sh` 真实退出码为 `1`，停在 cgroup v2 环境门槛。
这是预期的安全行为：环境不满足时拒绝生成 fake real capsule 证据。

本次远程证据文件：

```text
experiments/results/openeuler_smoke/go_test_latest.txt
experiments/results/openeuler_smoke/env_check_latest.txt
experiments/results/openeuler_smoke/smoke_latest.log
experiments/results/openeuler_smoke/env_check.json
experiments/results/openeuler_smoke/aort-r-openeuler-9029ec3-evidence.tgz
```

## 4. 非破坏性 cgroup2 探测

已尝试在临时目录挂载 cgroup2：

```bash
mkdir -p /tmp/aort-cg2
mount -t cgroup2 none /tmp/aort-cg2
stat -fc %T /tmp/aort-cg2
cat /tmp/aort-cg2/cgroup.controllers
umount /tmp/aort-cg2
```

结果：

```text
cgroup2fs
```

但 `cgroup.controllers` 为空。原因是当前系统以 legacy cgroup hierarchy 启动，
memory/pids/cpu 等控制器已经挂在 cgroup v1 层级，临时 cgroup2 mount 不能提供
任务书要求的 `memory.current`、`pids.current`、`cpu.max`、`pids.max` 等 real
控制器证据。

## 5. 当前结论

当前远程服务器不能作为 real cgroup v2 满血证据机。

| 要求 | 当前状态 |
| --- | --- |
| `/sys/fs/cgroup` 为 `cgroup2fs` | 未满足，当前是 `tmpfs` |
| `capsule_mode=real` | 未满足 |
| `cgroup_path=/sys/fs/cgroup/...` | 未满足 |
| `memory.current` 非 0 | 未满足 |
| `pids.current` 非 0 | 未满足 |
| freeze/unfreeze 2xx | 未满足 |
| kill 2xx | 未在 real capsule 下验证 |

## 6. 下一步

要完成任务书要求的 real cgroup v2 smoke，需要把远程 openEuler 从 legacy cgroup
启动模式切到 unified cgroup v2。当前 `/etc/default/grub` 中：

```text
GRUB_CMDLINE_LINUX="net.ifnames=0 consoleblank=600 console=tty0 console=ttyS0,115200n8 cgroup_disable=files apparmor=0 crashkernel=512M selinux=0"
```

建议下一步在获得明确授权后追加类似参数并重启：

```text
systemd.unified_cgroup_hierarchy=1 cgroup_no_v1=all
```

重启后必须重新验证：

```bash
stat -fc %T /sys/fs/cgroup
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
```

只有当 `/sys/fs/cgroup` 输出 `cgroup2fs`，且 `capsule_real.json` 中
`capsule_mode=real`、`memory_current>0`、`pids_current>0`、freeze/unfreeze/kill
均为 2xx，才能把本阶段标记为 real 完成。
