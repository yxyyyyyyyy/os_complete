# V1 Foundation Report

## Done

- `aortd` starts in mock mode.
- Health API returns ok.
- Demo API creates a deterministic software-engineering task.
- SSE streams runtime events and replays recent events for late subscribers.
- Dashboard displays task, AVP, context/syscall evidence, timeline, and experiment entry page.

## How To Test

```bash
GOCACHE="$PWD/.cache/go-build" go test ./...
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aortd --config configs/dev.yaml
curl -s http://127.0.0.1:8080/api/health
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -N --max-time 2 http://127.0.0.1:8080/api/events
cd dashboard
npm install
npm run test
npm run build
npm run dev
```

## Expected Evidence

- Browser shows task DAG and AVP state changes after pressing Run Demo.
- Terminal SSE stream contains `task.completed`.
- `/api/tasks/{task_id}/dag` returns Planner, Coder A, Coder B, Tester, Reviewer, and Fixer nodes.

## Known Risks

- V1 uses mock execution and does not yet create real cgroups.
- V1 does not yet isolate tool processes with overlayfs.
- V1 displays context through syscall evidence; real CVM pages arrive in V2.

## Next Version

- V2 replaces mock execution with worker processes, cgroups, overlayfs, CVM, syscall gateway, scheduler, IPC, and Supervisor.
