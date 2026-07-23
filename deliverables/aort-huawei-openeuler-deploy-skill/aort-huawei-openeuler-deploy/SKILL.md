---
name: aort-huawei-openeuler-deploy
description: Upload, deploy, run, and verify the AORT-R project on a Huawei Cloud ECS/openEuler server. Use when Codex needs to copy this repository to a Huawei Cloud server, prepare an openEuler runtime environment, run AORT-R smoke tests, run real cgroup v2/software-real evidence collection, run DeepSeek smoke from environment variables, pull artifacts back, or prepare verified evidence for GitHub.
---

# AORT-R Huawei openEuler Deploy

## Principles

- Use plain `ssh` and `scp`; do not use unrelated deployment skills.
- Never write SSH passwords, DeepSeek API keys, bearer tokens, or private keys into code, docs, shell scripts, reports, or result JSON.
- Read API keys only from environment variables.
- Do not fake `/sys/fs/cgroup/aort.slice/...` evidence. If the host is not live openEuler root with cgroup v2, report that clearly.
- Keep generated binaries, request/response bodies with secrets, `.git`, `node_modules`, and local caches out of commits.

## Inputs

Collect these from the user or infer safely from the repo:

```text
SERVER_IP=<Huawei Cloud public IP>
SSH_USER=root
REMOTE_DIR=/root/aort-r-huawei-run
REPO_URL=https://github.com/yxyyyyyyyy/os_complete.git
BRANCH=main
```

When password SSH is used, enter it interactively. Prefer adding the local
public key to `/root/.ssh/authorized_keys` after a successful login so repeated
commands can use BatchMode.

## Remote Environment Check

Run:

```bash
ssh root@<SERVER_IP> 'cat /etc/os-release; stat -fc %T /sys/fs/cgroup; id -u; go version || true'
```

Required for real cgroup v2 evidence:

```text
openEuler 24.03 LTS
cgroup2fs
uid 0
```

If Go is older than the repo `go.mod`, use the Go toolchain shim:

```bash
export GOTOOLCHAIN=go1.22.12
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
```

Use this especially on openEuler 24.03 images that ship Go 1.21.x.

## Upload Or Checkout

Prefer a clean remote checkout:

```bash
ssh root@<SERVER_IP> 'test -d /root/aort-r-huawei-run/.git || git clone https://github.com/yxyyyyyyyy/os_complete.git /root/aort-r-huawei-run'
ssh root@<SERVER_IP> 'cd /root/aort-r-huawei-run && git fetch origin main && git checkout --detach origin/main'
```

If GitHub is not reachable from the server, upload an archive:

```bash
tar --exclude='.git' --exclude='.cache' --exclude='node_modules' --exclude='dashboard/node_modules' -czf /tmp/aort-r.tar.gz .
scp /tmp/aort-r.tar.gz root@<SERVER_IP>:/tmp/aort-r.tar.gz
ssh root@<SERVER_IP> 'rm -rf /root/aort-r-huawei-run && mkdir -p /root/aort-r-huawei-run && tar -xzf /tmp/aort-r.tar.gz -C /root/aort-r-huawei-run'
```

Do not upload local `.env` files unless the user explicitly asks and confirms
they are safe for that server.

## Run AORT-R Verification

Run the standard acceptance sequence on the server:

```bash
ssh root@<SERVER_IP> '
set -e
cd /root/aort-r-huawei-run
export GOTOOLCHAIN=go1.22.12
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn

go test ./...
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
bash scripts/smoke_cgroupv2_multi_agent.sh
bash scripts/smoke_cgroupv2_limits.sh
bash scripts/smoke_software_real_openeuler.sh
'
```

The software-real smoke should write:

```text
experiments/results/software_real_demo/openeuler/software_real_openeuler_summary.json
experiments/results/software_real_demo/openeuler/software_real_capsules.json
```

Expected key fields:

```json
{
  "evidence_mode": "real-runtime",
  "capsule_evidence_mode": "real-cgroup-v2",
  "worker_cgroup_backed_agents": 6,
  "success": true
}
```

Each capsule row should include:

```json
{
  "pid": 12345,
  "capsule_mode": "real",
  "capsule_evidence_mode": "real-cgroup-v2",
  "cgroup_path": "/sys/fs/cgroup/aort.slice/..."
}
```

## DeepSeek Smoke

Only run this when the user supplies a key at runtime. Do not persist it:

```bash
ssh root@<SERVER_IP> '
cd /root/aort-r-huawei-run
export AORT_LLM_PROVIDER=deepseek
export DEEPSEEK_BASE_URL=https://api.deepseek.com
export DEEPSEEK_MODEL=deepseek-v4-flash
export DEEPSEEK_API_KEY="$DEEPSEEK_API_KEY"
bash scripts/smoke_deepseek.sh
'
```

Commit only the redacted summary:

```text
experiments/results/deepseek_smoke/summary.json
```

Remove raw request/response files unless they are known to contain no secrets
and the user explicitly wants them.

## Pull Evidence Back

Use `scp -r`:

```bash
scp -r root@<SERVER_IP>:/root/aort-r-huawei-run/experiments/results/openeuler_smoke experiments/results/
scp -r root@<SERVER_IP>:/root/aort-r-huawei-run/experiments/results/openeuler_cgroupv2_multi experiments/results/
scp -r root@<SERVER_IP>:/root/aort-r-huawei-run/experiments/results/openeuler_cgroupv2_limits experiments/results/
scp -r root@<SERVER_IP>:/root/aort-r-huawei-run/experiments/results/software_real_demo/openeuler experiments/results/software_real_demo/
scp -r root@<SERVER_IP>:/root/aort-r-huawei-run/experiments/results/deepseek_smoke experiments/results/
```

Skip a path if that smoke was not run.

## Local Validation

Validate the software-real live evidence:

```bash
python3 - <<'PY'
import json, pathlib
base = pathlib.Path("experiments/results/software_real_demo/openeuler")
summary = json.loads((base / "software_real_openeuler_summary.json").read_text())
capsules = json.loads((base / "software_real_capsules.json").read_text())
assert summary["evidence_mode"] == "real-runtime"
assert summary["capsule_evidence_mode"] == "real-cgroup-v2"
assert summary["worker_cgroup_backed_agents"] >= 6
assert summary["success"] is True
for row in capsules["capsules"]:
    assert row["pid"] > 0
    assert row["capsule_mode"] == "real"
    assert row["capsule_evidence_mode"] == "real-cgroup-v2"
    assert row["cgroup_path"].startswith("/sys/fs/cgroup/aort.slice/")
print("software-real openEuler evidence ok")
PY
```

Then run:

```bash
go test ./...
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
rg -n "DEEPSEEK_API_KEY=.*sk-|Authorization: Bearer sk-|sk-[0-9a-f]{32,}|<server-password>" . --glob '!.git/**'
```

The final `rg` should return no matches.

## Commit And Push

Before committing, inspect:

```bash
git status --short
git diff --cached --stat
```

Commit only:

- evidence JSON/status/text files that prove the run,
- redacted DeepSeek summary,
- docs/report updates,
- script fixes needed for reproducible smoke runs.

Do not commit generated worker binaries, raw secrets, private keys, or unrelated
dirty files.
