# AORT-R Real DeepSeek Large-Codebase DAG Design

- Date: 2026-07-16
- Target branch: `codex/aort-r-upgrade`
- Runtime host: Huawei Cloud ECS, openEuler 24.03 LTS, root, cgroup v2
- Required model: `deepseek-v4-flash`
- Status: design direction approved; written-spec review pending

## 1. Decision

The experiment uses AORT-R itself as the engineering workload. At design time,
the tracked Go tree contains 22,114 physical lines and 20,666 nonblank lines.
Every real run recomputes both values from the clean workload commit and fails
before any model call unless both are at least 20,000.

"A 20,000-line task" means that the DAG operates on, builds, and tests an
integrated codebase of at least 20,000 source lines. It does not mean padding a
fixture or forcing the model to rewrite 20,000 lines. The accepted patch is
expected to be targeted, reviewable, and justified by failing acceptance
tests.

The real engineering ticket is the current review remediation gap:

1. Connect `resource-isolation` to real worker processes, cgroup capsules,
   resource sampling, resource-aware scheduling, kill/destroy, OverlayFS, and
   checkpoint/replay.
2. Make the AORT-R `context-sharing` path execute real memfd/mmap and FD
   passing, and derive byte counters from instrumented transport operations.
3. Make `review-final` reject incomplete, fabricated, overwritten, wrong-mode,
   or insufficient-run evidence.

The DeepSeek DAG must produce and validate the patches for these three areas.
The accepted experiment diff may contain only DeepSeek coder/fixer patches and
deterministic formatting. If human review finds a functional defect, the
finding is returned to a DeepSeek fixer and the affected gates are rerun.
Direct human functional edits make that run ineligible for the claim that
DeepSeek completed the task. Evidence preserves model-produced patches,
rejected attempts, test results, and the final accepted diff so attribution
remains auditable.

## 2. Alternatives Considered

### A. Current AORT-R repository (selected)

Use the real 22,000-plus-line Go repository and the actual review findings.
This is directly relevant to the competition deliverable and proves that the
DAG can complete a multi-package maintenance task. It is less deterministic
than a generated fixture, so strict output parsing, bounded retries, and full
test gates are required.

### B. Deterministic generated 24,000-line Go fixture

This is easier to reproduce and can contain known seeded defects. It is not
selected because line volume could be mistaken for artificial padding and the
result would not improve AORT-R.

### C. External open-source repository

This provides independent scale but adds download, license, dependency, and
network variables while producing no direct AORT-R remediation. It is not
selected.

## 3. Two-Repository Execution Model

The runtime-under-test and the workload checkout are separated:

- The runner checkout contains the new `codebase-dag` orchestration and real OS
  adapters.
- The workload checkout is a clean, immutable AORT-R Git snapshot used for the
  engineering ticket.
- Each code-producing agent receives its own OverlayFS workspace based on that
  workload snapshot.
- Only validated patches cross from agent workspaces into the integration
  workspace.
- The runner owns immutable acceptance tests outside model-controlled
  workspaces. Their hashes are recorded before they are copied into the
  integration workspace and verified again after every patch and test run.
- The original local repository is never the target of remote cleanup, fault
  injection, or model-generated patch application.

The workload commit, tree hash, tracked file list, physical line count,
nonblank line count, and per-file SHA-256 values are recorded before model
execution.

## 4. DAG

The command surface is:

```text
aortctl scenario codebase-dag \
  --provider deepseek \
  --model deepseek-v4-flash \
  --workload <clean-checkout> \
  --ticket review-remediation \
  --out experiments/results/codebase_dag
```

The graph is:

```text
preflight
  -> planner
  -> resource-coder -----\
  -> context-coder -------+-> integrate -> tester -> reviewer -> finalizer
  -> evidence-coder -----/                    |          |
                                               +-> fixer -+
```

The Huawei host has two vCPUs and 3.3 GiB RAM, so the three coder nodes are
parallel-ready but the resource-aware scheduler runs no more than two worker
processes concurrently.

### 4.1 Tool-only preflight

Preflight performs no LLM call. It must prove all of the following:

- openEuler 24.03 LTS, UID 0, and `cgroup2fs`;
- writable nested cgroup and real OverlayFS mount/unmount;
- real memfd/mmap and FD-passing smoke success;
- `deepseek-v4-flash` is accepted by the configured DeepSeek endpoint;
- clean workload Git status and stable workload commit;
- at least 20,000 tracked physical Go lines and 20,000 tracked nonblank Go
  lines;
- baseline `go test ./...` passes before the runner-owned acceptance tests are
  enabled;
- the immutable acceptance tests fail against the unremediated workload for
  the expected three contract violations;
- no result directory already exists for the chosen run ID.

Any failed gate ends the run as `failed`. No degraded result can satisfy this
experiment.

### 4.2 Planner

The planner receives the ticket, repository manifest, package list, current
review findings, acceptance-test contract, and relevant API summaries. It must
return schema-valid JSON containing task decomposition, file ownership,
dependencies, risks, and verification commands.

### 4.3 Coder fan-out

Each coder is a real worker process in an independent cgroup and OverlayFS
workspace:

- `resource-coder` owns the resource scenario and its tests.
- `context-coder` owns the context transport scenario and its tests.
- `evidence-coder` owns strict evidence validation and its tests.

Each coder receives shared pages plus only the relevant source/test files. It
must return a unified diff and a structured explanation. The runner rejects
absolute paths, binary patches, out-of-scope files, malformed diffs, secret
material, and patches that do not apply cleanly.

### 4.4 Integration and testing

The integration node applies accepted patches in a fresh integration
workspace, runs `gofmt`, `git diff --check`, focused tests, and then
`go test ./...`. Failed outputs remain in the immutable run directory.
Runner-owned acceptance tests are outside every coder's allowed patch scope.
Their file list and SHA-256 values must remain identical to preflight.

The tester is a real DeepSeek node. It receives the ticket, accepted diff,
test inventory, and sanitized command results, then returns a schema-valid test
assessment and any missing-test requests. Tool execution remains under the
Gateway and is bounded to the integration workspace.

### 4.5 Review, fixer, and finalizer

The reviewer checks correctness, safety boundaries, evidence semantics,
regression risk, and whether the patch actually connects the required real OS
paths. A fixer call is mandatory when tests fail or the reviewer reports a
blocking finding. At most two fixer attempts are allowed.

The finalizer runs only after all gates pass. It summarizes the accepted patch,
test evidence, remaining limitations, and model usage. A textual claim from
the finalizer cannot override a failed machine gate.

## 5. Real DeepSeek Contract

- Provider must be `deepseek` for every LLM node.
- Requested and recorded model must be exactly `deepseek-v4-flash`.
- API key is read only from `DEEPSEEK_API_KEY` in the remote process
  environment.
- No key, authorization header, raw environment, or bearer token is written to
  an artifact.
- The runner does not register a mock provider and does not retry with another
  model.
- Missing key, unknown model, HTTP error, timeout, malformed output, empty
  choice, or missing usage data fails the relevant node.
- The run requires at least seven successful real API calls: planner, three
  coders, tester, reviewer, and finalizer.
- The run permits at most ten calls, including up to two fixer calls and one
  schema-repair retry. Exceeding the bound fails the run.
- Actual prompt, completion, and total token usage and latency are recorded per
  call. Cost is not estimated or hard-coded.

The user supplies the key at runtime. A temporary mode-0600 file on `/run` may
be sourced by the user into the remote shell, but AORT-R itself reads only the
environment variable. The temporary file is deleted after the run.

## 6. Real OS Integration

### 6.1 Capsules and resource sampling

Every LLM or tool worker has a unique child cgroup under a run-specific root.
Evidence includes PID, cgroup path, configured limits, sampled CPU/memory/pids,
`cgroup.events`, kill method, destroy result, and timestamps. A worker reported
as `real-cgroup-v2` must have a live PID attached to the recorded cgroup during
sampling.

Coder workers use bounded memory, pids, and CPU limits suitable for the
two-vCPU host. The test worker receives a larger memory/pids allowance so full
repository tests are not starved. Limits and samples are recorded rather than
inferred from Go heap statistics.

### 6.2 Resource-aware scheduling

The scheduler consumes `CgroupSampler` output. Coder readiness, measured
pressure, virtual runtime, and shared-page affinity contribute to each
decision. Every selection records candidates, pressure, scores, selected
worker, and fallback reason. A missing sampler is a hard failure in this
experiment.

### 6.3 OverlayFS workspaces

Coder and integration workspaces must report `real-overlayfs`. The lowerdir is
hashed before and after the run. Model-generated commands can operate only in
the assigned upper/merged directory through the Gateway. Any lowerdir change,
cross-agent write, unsafe path, leaked mount, or failed unmount fails the run.

### 6.4 Checkpoint, crash, and replay

After a coder produces a valid patch, the runner checkpoints its AVP state,
page table, patch hash, and DAG completion state. One designated worker follows
a deterministic fail-once path before acknowledgement. The supervisor kills
and destroys that capsule, restores the checkpoint into a fresh capsule, and
replays the node without issuing a duplicate LLM call when the completed model
output is already checkpointed. The accepted patch hash must remain identical.

### 6.5 CVM and memfd/mmap

The ticket, repository manifest, acceptance contract, and common review context
are stored as CVM pages. The actual bytes are written to memfd, shared through
FD passing, and mapped read-only by worker processes. Workers verify content
hashes before use. Private prompts remain per-agent pages.

The runner records bytes written, SCM_RIGHTS control-message bytes, mapped
bytes, materialized prompt bytes, shared/private page counts, hash validation,
transport latency, and Prefix Affinity decisions. Derived saved-byte values
come only from these counters. The report must not call this model KV-cache
sharing or claim end-to-end zero-copy.

## 7. Failure Semantics

The following conditions make the whole experiment fail:

- any mock, fallback, skipped, planned, or degraded provider/runtime mode;
- wrong model or fewer than seven successful real model calls;
- workload below either 20,000-line threshold;
- an LLM node that does not contribute its required structured output;
- unsafe or out-of-scope patch content;
- any change to a runner-owned acceptance test or its recorded hash;
- any direct human functional edit in the accepted experiment diff;
- failure to create, attach, sample, kill, or destroy required cgroups;
- non-real OverlayFS or shared-memory evidence;
- replay that changes the checkpointed output hash or duplicates a completed
  API call;
- failing focused tests, full `go test ./...`, real-only smoke, secret scan, or
  artifact validation;
- missing, overwritten, contradictory, or unparsable evidence.

Errors are retained as sanitized artifacts. A report generator may explain a
failure but cannot convert it into pass.

## 8. Immutable Evidence

Each run writes to a new directory and refuses to overwrite it:

```text
experiments/results/codebase_dag/<run-id>/
  summary.json
  report.md
  preflight.json
  source_manifest.json
  dag.json
  llm_calls.jsonl
  scheduler_decisions.jsonl
  capsules.jsonl
  resource_samples.jsonl
  context_transport.json
  workspace.json
  checkpoint.json
  tool_calls.jsonl
  tests.json
  secret_scan.txt
  patches/
    resource-coder.diff
    context-coder.diff
    evidence-coder.diff
    fixer-*.diff
    accepted.diff
```

`summary.json` includes the runner commit, workload commit, dirty state,
environment, exact provider/model, call/token counts, line counts, DAG node
statuses, changed files, test commands, real OS modes, replay status, cleanup
status, artifact hashes, and limitations.

Raw HTTP authorization data and unsanitized process environments are never
stored. Model text is stored only after secret redaction. Patches and all
machine evidence receive SHA-256 hashes in the final index.

## 9. Strict Acceptance

The run passes only when all of these are true:

1. Huawei openEuler/root/cgroup-v2 environment gate passes.
2. Physical and nonblank tracked Go line counts are each at least 20,000.
3. The configured and actual provider/model are DeepSeek and
   `deepseek-v4-flash` for every LLM call.
4. At least seven successful real API calls occur with zero mock/fallback calls.
5. All three coder outputs are valid, applied, and represented in the accepted
   diff.
6. Immutable runner-owned acceptance tests fail for the expected reasons on
   the baseline and pass unchanged on the final workload.
7. The accepted diff consists only of recorded DeepSeek coder/fixer patches
   plus deterministic formatting; human functional-edit count is zero.
8. Real cgroup, resource sampler, scheduler, OverlayFS, CVM, memfd/mmap, FD
   passing, checkpoint/replay, and cleanup evidence passes strict validation.
9. Acceptance tests prove the three original review defects are fixed.
10. `gofmt`, `git diff --check`, focused tests, `go test ./...`, existing smoke,
   and real-only openEuler verification pass on the final workload.
11. Secret scanning finds no API key, bearer token, password, or private key.
12. The immutable evidence validator finds no missing files, status
    contradictions, unsupported modes, or overwritten run IDs.

After the model-produced patch is reviewed and incorporated into the target
branch, the final target commit is redeployed and the complete acceptance suite
is rerun. Only evidence tied to that final commit may be cited in final review
documents and defense materials.

## 10. Delivery

The implementation is staged as small commits:

1. strict tests and real-only evidence contracts;
2. real codebase-DAG orchestration and DeepSeek output schemas;
3. cgroup/OverlayFS/checkpoint adapters;
4. CVM plus real memfd/mmap worker transport;
5. DeepSeek-produced remediation patches;
6. immutable evidence validator and openEuler scripts;
7. final real run evidence and corrected documentation.

Only scoped source, tests, scripts, redacted evidence, and documentation are
committed. API keys, raw authorization data, generated binaries, caches,
temporary workspaces, mounts, and cgroups are excluded and cleaned.
