# Review-to-Task Matrix

| Teacher feedback | Concrete gap | Remediation task | Code/docs | Evidence | Acceptance |
|---|---|---|---|---|---|
| Explain interference/isolation | no joined resource scenario | P2 | `internal/review/resource_isolation.go` | `resource-isolation` raw and summary | normal agents remain scoped; lowerdir hash unchanged |
| Explain design route | feature docs are scattered | P1/P7 | `docs/design/01..05` | links to functions and commands | problem -> design -> implementation flow |
| Quantify communication | legacy E3 lacks required modes/ratios | P3/P4 | `internal/review/context_sharing.go` | per-ratio comparison CSV | logical/physical/transferred/materialized counters |
| Prove real service/Demo | existing Demo has no standalone contract | P5 | `internal/review/agent_demo.go` | timeline/final result | six roles, 1 LLM, 3 tools, fault continuation |
| Focus on 1-2 problems | six modules presented equally | P6/P7/P8 | design + defense docs | review response and claims table | only two public problem lines |

Every row must remain traceable to a real code path, an executable command, and a JSON/CSV/report artifact. Unsupported or degraded capabilities are recorded as such instead of being promoted to PASS.
