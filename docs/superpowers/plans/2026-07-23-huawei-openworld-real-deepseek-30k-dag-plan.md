# Huawei Open World Real DeepSeek 30k-DAG Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `aort-huawei-openeuler-deploy` for all Huawei Cloud SSH/SCP/smoke operations, and use `superpowers:subagent-driven-development` (or `executing-plans`) only after the user explicitly approves starting execution. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Hard stop:** Do not deploy, inject keys, call DeepSeek, or mutate the remote host until Phase 0 gates are confirmed by the user.

**Goal:** On a real Huawei Cloud openEuler host (not MOOC/mock/simulation), run a strict DeepSeek-only large-codebase DAG whose workload has ≥30,000 tracked Go physical lines and ≥30,000 tracked Go nonblank lines, then pull immutable evidence back.

**Architecture:** Reuse the already-approved design in `docs/superpowers/specs/2026-07-16-real-deepseek-large-dag-design.md` and the implementation breakdown in `docs/superpowers/plans/2026-07-16-real-deepseek-large-dag-plan.md`, with three deltas: (1) line gate raised from 20,000 → 30,000; (2) Open World real-only policy forbids MOOC/mock/fallback as acceptance evidence; (3) remote operations must go through the Huawei deploy skill, not ad-hoc scripts.

**Tech Stack:** Go 1.22+, DeepSeek OpenAI-compatible API (`deepseek-v4-flash`), Huawei Cloud ECS / openEuler 24.03 LTS, cgroup v2, OverlayFS, memfd/mmap, `ssh`/`scp`, existing AORT-R Runtime.

## Global Constraints

- Target host is real Huawei Cloud openEuler 24.03 LTS, UID 0, `cgroup2fs`. Local macOS or degraded Linux is never accepted as Open World evidence.
- “No MOOC mode” means: no mock LLM, no fallback-to-mock, no `simulation` / `degraded` / `planned` / `skipped` labels for required gates, and no teaching-demo path that fakes `/sys/fs/cgroup/aort.slice/...`.
- Provider is only `deepseek`; requested and returned model must be exactly `deepseek-v4-flash`.
- At least 7 successful real API calls; at most 10 calls total; zero mock/fallback calls.
- `DEEPSEEK_API_KEY` is read only from the remote process environment; never write keys, passwords, bearer tokens, or private keys into code, docs, scripts, JSON, or Git.
- Workload tracked Go physical lines ≥ 30,000 **and** nonblank lines ≥ 30,000 before any completion call. Current baseline (2026-07-23): physical `22114`, nonblank `20666` — so Phase 1 must grow the real Go tree by ~8k+ lines of non-padding product/test code.
- Result directories are create-exclusive by run ID; never overwrite previous runs.
- Prefer remote checkout/archive of a clean committed tree; do not upload local `.env*` unless the user explicitly confirms.
- Existing detailed Tasks 1–12 in `2026-07-16-real-deepseek-large-dag-plan.md` remain the implementation source of truth; this document is the Open World execution wrapper and 30k delta.

## Current Baseline (do not skip)

| Item | Status on 2026-07-23 |
| --- | --- |
| Design spec | Present: `docs/superpowers/specs/2026-07-16-real-deepseek-large-dag-design.md` |
| Implementation plan Tasks 1–12 | Present but **unchecked / unimplemented** |
| `internal/codebasedag/` | **Missing** |
| `aortctl scenario codebase-dag` | **Missing** |
| Tracked Go LOC | physical `22114`, nonblank `20666` (< 30k gate) |
| Huawei deploy skill | Present only under `deliverables/aort-huawei-openeuler-deploy-skill/.../SKILL.md`; **not installed** into `~/.codex/skills` |
| Historical cloud evidence host | `root@116.204.94.247` (presentation evidence); **not** present in `~/.ssh/config` |
| Local SSH aliases | Other hosts exist; Huawei Open World IP must be confirmed before any remote step |
| DeepSeek smoke path | `scripts/smoke_deepseek.sh` exists; still env-key gated |

## File Structure (delta only)

```text
# Install / reference
deliverables/aort-huawei-openeuler-deploy-skill/aort-huawei-openeuler-deploy/SKILL.md
~/.codex/skills/aort-huawei-openeuler-deploy/SKILL.md   # install copy after user approval

# LOC growth (Phase 1) — real product/test Go only, no filler generators in evidence claim
internal/**/*.go                                        # grow until both LOC gates ≥ 30000
cmd/**/*.go

# Implementation (Phases 2–3) — follow 2026-07-16 plan file paths exactly
internal/codebasedag/**
cmd/aort-code-worker/main.go
cmd/aortctl/main.go                                     # scenario codebase-dag + evidence codebase-dag
scripts/competition_verify_real.sh

# Evidence (Phase 5+)
experiments/results/codebase_dag/<run-id>/
experiments/results/deepseek_smoke/summary.json         # redacted only
```

---

### Phase 0: Preconditions And Skill Binding

**Files:**
- Read: `deliverables/aort-huawei-openeuler-deploy-skill/aort-huawei-openeuler-deploy/SKILL.md`
- Optional install: `~/.codex/skills/aort-huawei-openeuler-deploy/SKILL.md`
- Do not modify remote hosts in this phase.

**Interfaces:**
- Consumes: user-confirmed `SERVER_IP`, SSH access method, DeepSeek key availability (yes/no only).
- Produces: written gate checklist; execution blocked until all boxes pass.

- [ ] **Step 1: Confirm what “Huawei Open World / no MOOC” means for this run**

Record the acceptance policy verbatim in the run notes:

```text
required_evidence_modes = real-runtime | real-cgroup-v2 | real-overlayfs | real-api
forbidden_as_pass = mooc | mock | fallback | simulation | degraded | planned | skipped
provider = deepseek
model = deepseek-v4-flash
line_gate_physical = 30000
line_gate_nonblank = 30000
```

- [ ] **Step 2: Confirm Huawei server identity with the user**

Ask for and record only:

```text
SERVER_IP=...
SSH_USER=root
REMOTE_DIR=/root/aort-r-huawei-run
SSH_ALIAS_OR_KEY=...
```

Do not guess. Historical evidence used `116.204.94.247`; current `~/.ssh/config` does not list it. If the user provides a different IP, that IP wins.

- [ ] **Step 3: Bind the deploy skill**

Prefer installing the skill for Codex:

```bash
mkdir -p ~/.codex/skills/aort-huawei-openeuler-deploy
cp -R deliverables/aort-huawei-openeuler-deploy-skill/aort-huawei-openeuler-deploy/* \
  ~/.codex/skills/aort-huawei-openeuler-deploy/
```

If the user prefers not to install globally, agents must still open and follow the deliverables copy on every remote action.

- [ ] **Step 4: Confirm DeepSeek key readiness without reading the secret**

User answers only `ready` / `not-ready`. If not ready, stop before Phase 4. Key injection later must use `/run/aort-deepseek.env` mode `600` as in Task 11 of the 2026-07-16 plan.

- [ ] **Step 5: User explicit start gate**

Do not proceed to Phase 1 coding or Phase 4 remote work until the user replies with an explicit start instruction (for example: “按这个计划开始 Phase 1” or “从 Phase 0 远程探活开始”).

---

### Phase 1: Grow Workload To ≥30,000 Go Lines

**Files:**
- Modify / create real Go packages under `internal/` and `cmd/` that the review-remediation ticket actually needs (resource isolation, context transport, evidence validation helpers, tests).
- Modify: `docs/superpowers/specs/2026-07-16-real-deepseek-large-dag-design.md` line-gate numbers `20000` → `30000` after user approves the delta.
- Modify: `docs/superpowers/plans/2026-07-16-real-deepseek-large-dag-plan.md` matching gate constants.

**Interfaces:**
- Consumes: current tracked Go tree (`22114` / `20666`).
- Produces: clean commit where both counters are ≥ `30000`.

- [ ] **Step 1: Measure baseline with the same gate logic the runner will use**

```bash
python3 - <<'PY'
import pathlib, subprocess
files = subprocess.check_output(['git','ls-files','*.go'], text=True).splitlines()
phys = nonblank = 0
for f in files:
    lines = pathlib.Path(f).read_text(errors='ignore').splitlines()
    phys += len(lines)
    nonblank += sum(1 for line in lines if line.strip())
print({'files': len(files), 'physical': phys, 'nonblank': nonblank})
assert phys >= 1
PY
```

Expected today: physical ≈ `22114`, nonblank ≈ `20666`.

- [ ] **Step 2: Choose growth strategy (recommended: real remediation modules)**

Recommended approach A: implement missing real OS/test scaffolding that Phase 2–3 needs anyway, until both counters clear 30k.
Rejected for Open World claim: random comment padding, duplicated copies of the same file, generated dead code with no tests, counting Vue/Markdown as “DAG 三万行代码”.

- [ ] **Step 3: Add only reviewable Go + tests; keep `go test ./...` green locally**

Run: `GOCACHE=$PWD/.cache/go-build go test ./...`

Expected: PASS on portable tests; Linux-root-only tests may SKIP locally.

- [ ] **Step 4: Re-measure and hard-fail if still under gate**

Require:

```text
physical >= 30000
nonblank >= 30000
```

- [ ] **Step 5: Commit LOC growth separately before DAG runner work**

```bash
git add internal cmd docs/superpowers/specs/2026-07-16-real-deepseek-large-dag-design.md \
  docs/superpowers/plans/2026-07-16-real-deepseek-large-dag-plan.md
git commit -m "$(cat <<'EOF'
feat: grow tracked Go workload above 30k-line DAG gate

EOF
)"
```

Only after the user asks for a commit, or as part of an approved execution session.

---

### Phase 2: Implement Strict DeepSeek + Codebase DAG Runner

**Files:** Follow Tasks 1–9 in `docs/superpowers/plans/2026-07-16-real-deepseek-large-dag-plan.md` exactly, with these constant replacements everywhere:

```text
20000 -> 30000
"at least 20,000" -> "at least 30,000"
```

Critical packages:

- `internal/llm/*` strict model/metadata gates
- `internal/codebasedag/*` runner, manifest, runstore, validate
- `cmd/aort-code-worker`
- `cmd/aortctl` `scenario codebase-dag` / `evidence codebase-dag`

**Interfaces:**
- Consumes: ≥30k clean workload commit.
- Produces: local-verified runner commit that can refuse mock/fallback.

- [ ] **Step 1: Execute 2026-07-16 Task 1 (strict DeepSeek metadata)** with no fallback provider registered in the real runner path.
- [ ] **Step 2: Execute Task 2 with `MinPhysicalLines=30000` and `MinNonblankLines=30000`.**
- [ ] **Step 3: Execute Tasks 3–9 (DAG state, OverlayFS, memfd transport, capsules/replay, prompts/patches, runner, acceptance/CLI).**
- [ ] **Step 4: Local gate**

```bash
GOCACHE=$PWD/.cache/go-build go test ./...
go vet ./...
bash scripts/competition_verify.sh
rg -n "DEEPSEEK_API_KEY=.*sk-|Authorization: Bearer sk-|sk-[0-9a-f]{32,}" . --glob '!.git/**'
```

Expected: tests pass or skip only for missing live prerequisites; secret scan empty.

---

### Phase 3: Local Runner Release + Runbook

**Files:**
- Create: `docs/review_remediation/CODEBASE_DAG_RUNBOOK.md`
- Follow 2026-07-16 Task 10.

- [ ] **Step 1: Write runbook with Open World / no-MOOC checklist and 30k gates.**
- [ ] **Step 2: Record `RUNNER_COMMIT=$(git rev-parse HEAD)`.**
- [ ] **Step 3: Build release archive with `git archive` only (no dirty tree).**

---

### Phase 4: Huawei Host Probe (skill-driven, no DeepSeek spend yet)

**Files / remote paths:**
- Remote: `/root/aort-r-huawei-run` or `/root/aort-r-runner-$RUNNER_COMMIT`
- Skill: `aort-huawei-openeuler-deploy`

- [ ] **Step 1: Environment probe**

```bash
ssh root@$SERVER_IP 'cat /etc/os-release; stat -fc %T /sys/fs/cgroup; id -u; go version || true; nproc; free -h; df -h /; grep -w overlay /proc/filesystems || true'
```

Expected:

```text
openEuler 24.03 LTS
cgroup2fs
uid 0
Go >= repo go.mod (use GOTOOLCHAIN=go1.22.12 if image has 1.21.x)
overlay present
```

- [ ] **Step 2: DeepSeek endpoint reachability without spending completion tokens**

Prefer an unauthenticated `/models` or a deliberate unauthorized probe that returns HTTP 401 and proves network path. Do not send the API key in logs.

- [ ] **Step 3: Deploy clean runner + workload checkouts using the skill upload/checkout section.**
- [ ] **Step 4: Run non-LLM real OS smokes only**

```bash
export GOTOOLCHAIN=go1.22.12
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
go test ./...
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
bash scripts/smoke_cgroupv2_multi_agent.sh
bash scripts/smoke_cgroupv2_limits.sh
bash scripts/smoke_software_real_openeuler.sh
```

Expected software-real summary fields:

```json
{
  "evidence_mode": "real-runtime",
  "capsule_evidence_mode": "real-cgroup-v2",
  "worker_cgroup_backed_agents": 6,
  "success": true
}
```

If any smoke is degraded/mock, stop and report; do not continue to the 30k DAG.

---

### Phase 5: Real DeepSeek Smoke, Then 30k Codebase DAG

**Files:**
- Remote env: `/run/aort-deepseek.env` (mode 0600, user-injected)
- Out: `experiments/results/deepseek_smoke/`
- Out: `experiments/results/codebase_dag/$RUN_ID/`

- [ ] **Step 1: User injects key on the server (agent never prints it).**
- [ ] **Step 2: Small real-api smoke**

```bash
set -a
source /run/aort-deepseek.env
set +a
export AORT_LLM_PROVIDER=deepseek
export AORT_LLM_FALLBACK_PROVIDER=   # empty: Open World forbids mock fallback
export DEEPSEEK_BASE_URL=https://api.deepseek.com
export DEEPSEEK_MODEL=deepseek-v4-flash
bash scripts/smoke_deepseek.sh
```

Expected redacted summary: `evidence_mode=real-api`, `fallback=false`, model `deepseek-v4-flash`.

- [ ] **Step 3: Preflight-only dry check for LOC + OS gates if CLI supports it; otherwise rely on runner preflight.**
- [ ] **Step 4: Run the real DAG**

```bash
./bin/aortctl scenario codebase-dag \
  --provider deepseek \
  --model deepseek-v4-flash \
  --workload "/root/aort-r-workload-$RUNNER_COMMIT" \
  --ticket review-remediation \
  --worker-command ./bin/aort-code-worker \
  --max-calls 10 \
  --max-concurrent 2 \
  --node-timeout 180s \
  --run-timeout 45m \
  --run-id "$RUN_ID" \
  --out experiments/results/codebase_dag
```

Expected pass conditions (all required):

1. physical/nonblank ≥ 30000
2. ≥7 real `deepseek-v4-flash` calls, 0 mock/fallback
3. real cgroup / OverlayFS / memfd evidence
4. immutable acceptance scripts pass on final workload
5. secret scan clean

- [ ] **Step 5: Remote validate, pull evidence, delete `/run/aort-deepseek.env` only.**
- [ ] **Step 6: Locally re-validate and commit only redacted evidence after user approval.**

---

### Phase 6: Final Open World Acceptance Bundle

**Files:** Follow 2026-07-16 Task 12, but every required command uses `--require-real` / `--require-real-shm` / DeepSeek real provider with empty fallback.

- [ ] **Step 1: Apply only attributed DeepSeek patches; no human functional edits.**
- [ ] **Step 2: Redeploy final commit; rerun `competition_verify_real.sh` and review scenarios on Huawei.**
- [ ] **Step 3: Update defense/review docs from measured fields only.**
- [ ] **Step 4: Produce a short Open World claim sheet listing run IDs, LOC, call counts, and explicit non-claims (not KV-cache sharing, not full container/VM sandbox).**

---

## Approach Choice (locked recommendation)

1. **Recommended:** Grow the real AORT-R Go tree past 30k, implement `codebase-dag`, then run on Huawei with skill-driven deploy and strict DeepSeek-only gates.
2. **Fallback if time-constrained:** Keep the 20k design temporarily and only raise the marketing claim after LOC is real — **not acceptable** for the user’s 三万行 requirement.
3. **Rejected:** Count dashboard/docs lines, MOOC/mock demos, or Mac degraded runs as Open World evidence.

## Spec Coverage Self-Check

| User requirement | Plan coverage |
| --- | --- |
| Register/run on real Huawei Open World | Phases 0, 4, 5, 6 |
| No MOOC mode | Phase 0 policy + empty fallback + forbidden modes |
| Real DeepSeek API | Phase 5 smoke + DAG; Task 1 strict provider |
| Large engineering DAG ≈ 30k LOC | Phase 1 gate + Phase 2 manifest constants |
| Use local Huawei skill | Phase 0 skill bind + Phase 4/5 skill operations |
| Do not start arbitrarily | Phase 0 Step 5 hard stop |

## Execution Handoff

Plan saved to `docs/superpowers/plans/2026-07-23-huawei-openworld-real-deepseek-30k-dag-plan.md`.

**Do not start coding or SSH until the user answers Phase 0 questions and gives an explicit start command.**

After approval, preferred order:

1. Subagent-Driven implementation of Phases 1–3
2. Inline skill-driven Huawei Phases 4–6 with checkpoints after each remote gate
