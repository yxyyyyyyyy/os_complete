# PHASE 20 Software Real Demo

## Status

Demo: `software-real`

Evidence mode: `real-runtime`

Endpoints:

- `POST /api/demo/software-real/run`
- `GET /api/demo/software-real/status`
- `GET /api/demo/software-real/result`

Artifact:

- `experiments/results/software_real_demo/result.json`
- Live openEuler worker/cgroup artifact directory:
  `experiments/results/software_real_demo/openeuler/`

## Runtime Flow

The demo runs:

```text
Planner -> Coder -> Tester -> Reviewer -> Fixer -> Reporter
```

It triggers runtime evidence for:

- `agent.spawn`
- scheduler decision
- `context.materialize`
- `context.write_delta`
- `llm.call`
- `tool.exec`
- `ipc.publish`
- `ipc.poll`
- `agent.report`
- checkpoint
- test failure recovery and tool timeout recovery

## Worker And Capsule Evidence

When the worker registry and capsule manager are enabled, `software-real`
creates/registers the six demo Agents as worker-backed runtime Agents before
the end-to-end flow runs. The result artifact now records each Agent's
`pid`, `capsule_mode`, and `cgroup_path`.

On openEuler 24.03 LTS with `cgroup_root=/sys/fs/cgroup/aort.slice`, those
records use real cgroup v2 capsule paths such as:

```json
{
  "pid": 12345,
  "capsule_mode": "real",
  "cgroup_path": "/sys/fs/cgroup/aort.slice/software-real-...-planner"
}
```

The runtime result remains labeled `evidence_mode=real-runtime`; cgroup proof
is carried per Agent through `capsule_mode`, `capsule_evidence_mode`, and
`cgroup_path`. Live openEuler capsule rows use
`capsule_evidence_mode=real-cgroup-v2`; local test-root rows use
`capsule_evidence_mode=test-cgroup-v2` so they are not confused with
`/sys/fs/cgroup` evidence. OpenEuler live capsule proof remains labeled
`real-cgroup-v2` in the openEuler evidence directories.

The live openEuler smoke entrypoint is:

```bash
bash scripts/smoke_software_real_openeuler.sh
```

It requires cgroup v2 + root, starts `aortd`, runs
`POST /api/demo/software-real/run`, and validates that all six software-real
Agents have real worker `pid`, `capsule_mode=real`,
`capsule_evidence_mode=real-cgroup-v2`, and cgroup paths under
`/sys/fs/cgroup/aort.slice/...`.

## Go Test Recovery

The Tester creates a tiny Go module with `NormalizeSpace` implemented
incorrectly and runs `go test ./...`; the first test fails. The Fixer then
creates the corrected implementation with `strings.Fields` and `strings.Join`,
runs `go test ./...` again, and the second test passes.

The result artifact records `first_test_status=failed`,
`second_test_status=passed`, syscall counts, scheduler decision counts, IPC
metrics, checkpoint evidence, and `final_status=success`.
