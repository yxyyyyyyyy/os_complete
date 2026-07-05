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

## Go Test Recovery

The Tester creates a tiny Go module with `NormalizeSpace` implemented
incorrectly and runs `go test ./...`; the first test fails. The Fixer then
creates the corrected implementation with `strings.Fields` and `strings.Join`,
runs `go test ./...` again, and the second test passes.

The result artifact records `first_test_status=failed`,
`second_test_status=passed`, syscall counts, scheduler decision counts, IPC
metrics, checkpoint evidence, and `final_status=success`.
