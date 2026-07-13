# Review Remediation Plan

## Stage mapping

| Stage | Review issue | Code location | Experiment/command | Deliverable | Acceptance |
|---|---|---|---|---|---|
| P1 | Design and route unclear | `cmd/aortctl`, `internal/*`, existing scripts | `go test ./...`, repository audit | `AUDIT.md`, `PLAN.md`, matrix | every claim cites a path/field |
| P2 | Resource interference and fault spread | `internal/capsule`, `workspace`, `resource`, `scheduler` | `aortctl scenario resource-isolation` | raw/summary/CSV/report + design chapter | three modes, six roles, safe cleanup |
| P3 | Communication efficiency unquantified | `internal/cvm`, `ipc/shm`, `scheduler` | `aortctl scenario context-sharing` | raw/summary/CSV/report + design chapter | four ratios, three modes, measured byte counters |
| P4 | Results not comparable | `internal/review/metrics.go`, `internal/evidence` | `aortctl evidence review-final` | versioned index and summary | stable schema, percentiles, failure handling |
| P5 | Real service/Demo weakly separated | `internal/llm`, `internal/demo`, Gateway | `aortctl scenario real-agent-demo` | timeline/final result/report | mock deterministic; real API optional and redacted |
| P6 | Security boundary overclaimed | `internal/capsule`, `workspace`, `syscall`, `trace` | static claim scan + evidence inspection | threat model, claims boundary | residual risks explicit |
| P7 | Technical report feature-list shaped | all implemented paths | latest scenario evidence | `docs/design/01..10` | every chapter uses problem-driven structure |
| P8 | Defense material lacks evidence map | final reports and summaries | link/field validation | `docs/defense/*` | two main lines only; data sources named |
| P9 | Final acceptance not closed | all above | old + new smoke and final commands | checklist/change log/final report | no secret, contradiction, or unsafe cleanup |

## Order and compatibility

1. Establish the audit and common metrics contract.
2. Implement resource-isolation and context-sharing adapters.
3. Add the provider-separated Demo and review-final index.
4. Write documentation from the resulting code and evidence.
5. Run old and new acceptance commands in separate output roots, then commit/push.

The new outputs live below `experiments/results/review_remediation/` or an explicit `--out` path. Existing `experiments/results/final/` and historical directories are read-only inputs for review reports.

## Risks and mitigations

- cgroup/OverlayFS/eBPF support varies by host: use existing capability probes and record `degraded` plus a concrete reason.
- Pressure tests can leak processes or mounts: use bounded values, context deadlines, root-prefix checks, and deferred cleanup.
- Network API timing is not a performance benchmark: keep it in the availability Demo and separate network/model/runtime fields.
- Existing dirty workspace artifacts are not part of the remediation commit unless explicitly staged; only files listed in each stage are staged.
