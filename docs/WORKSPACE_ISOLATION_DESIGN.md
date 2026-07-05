# Workspace Isolation Design

Each Agent receives a workspace under the runtime temp root:

```text
/tmp/aort-runtime/workspaces/{agent_id}/
  lower/
  upper/
  work/
  merged/
  output/
  evidence/
```

The lowerdir is initialized with:

- `README.txt`
- `src/main.txt`
- `config/runtime.json`

Modes:

- `real-overlayfs`: Linux root host, overlayfs is listed in `/proc/filesystems`, and the mount succeeds.
- `degraded-copy`: fallback mode for macOS, non-root Linux, missing overlayfs, or mount failure.

Real overlayfs mount:

```bash
mount -t overlay overlay -o lowerdir=<lower>,upperdir=<upper>,workdir=<work> <merged>
```

Degraded-copy fallback:

- `Create`: copy lowerdir into merged.
- `Commit`: copy merged into output and write `commit_manifest.json`.
- `Rollback`: delete merged contents and copy lowerdir again.
- `Destroy`: remove the Agent workspace directory.

Safety boundary:

- `EnsureUnderRoot(root, target)` uses absolute cleaned paths and verifies the target remains inside the runtime root.
- Delete, copy, commit, rollback, and destroy operations check paths before acting.
- Symlinks are not followed during commit/copy; symlink escapes are rejected.
- The workspace manager defaults to `os.TempDir()/aort-runtime/workspaces`, not the repository directory.

Fault demo:

```bash
go run ./cmd/aortctl demo fault workspace-rmrf
```

The demo creates `planner`, `coder`, and `reviewer` workspaces, deletes
`coder/merged/src`, verifies the other Agents and lowerdir remain intact, rolls
back `coder`, commits the restored workspace, destroys all demo workspaces, and
writes:

- `experiments/results/workspace_isolation_evidence.json`

Current limits:

- Real overlayfs depends on Linux root mount capability.
- Non-Linux or non-root hosts intentionally report `degraded-copy`.
- The demo proves workspace isolation and rollback; it does not claim kernel namespace isolation.
