package codebasedag

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type LocalPreflight struct {
	ManifestOptions ManifestOptions
	RequireAPIKey   bool
	RequireClean    bool
	WorkerCommand   string
}

func (p LocalPreflight) Check(ctx context.Context, cfg RunnerConfig) (PreflightResult, error) {
	result := PreflightResult{Gates: make(map[string]bool)}
	result.Gates["provider_model"] = cfg.Provider == RequiredDeepSeekProvider && cfg.Model == RequiredDeepSeekModel
	result.Gates["go_runtime"] = strings.HasPrefix(runtime.Version(), "go1.")
	result.Gates["os_arch"] = runtime.GOOS != "" && runtime.GOARCH != ""
	result.Gates["git_available"] = gitAvailable(p.ManifestOptions.GitPath)
	result.Gates["run_id_valid"] = runIDPattern.MatchString(cfg.RunID)
	if p.RequireAPIKey {
		result.Gates["api_key_present"] = os.Getenv("DEEPSEEK_API_KEY") != ""
	} else {
		result.Gates["api_key_present"] = true
	}
	if p.WorkerCommand != "" {
		_, err := exec.LookPath(p.WorkerCommand)
		result.Gates["worker_command"] = err == nil
	} else {
		result.Gates["worker_command"] = true
	}
	tmp := os.TempDir()
	probe := filepath.Join(tmp, "aort-preflight-write-probe")
	result.Gates["tmpdir_writable"] = os.WriteFile(probe, []byte("ok"), 0o600) == nil
	_ = os.Remove(probe)

	manifest, _, err := BuildSourceManifestWithOptions(ctx, cfg.WorkloadDir, p.ManifestOptions)
	if err == nil {
		result.Manifest = manifest
		result.Gates["source_manifest"] = true
		result.Gates["git_commit_present"] = manifest.GitCommit != ""
		if p.RequireClean {
			result.Gates["clean_worktree"] = !manifest.GitDirty
		} else {
			result.Gates["clean_worktree"] = true
			if manifest.GitDirty {
				result.Gates["dirty_worktree_risk"] = true
			}
		}
	} else {
		result.Gates["source_manifest"] = false
		result.Gates["git_commit_present"] = false
		result.Gates["clean_worktree"] = false
	}

	// cgroup probe: required only on Linux; elsewhere mark N/A as passed.
	cgroupOK, cgroupWritable := probeCgroupV2()
	if runtime.GOOS == "linux" {
		result.Gates["cgroup_v2_present"] = cgroupOK
		if os.Geteuid() == 0 {
			result.Gates["cgroup_writable"] = cgroupWritable
		} else {
			result.Gates["cgroup_writable"] = true
		}
	} else {
		result.Gates["cgroup_v2_present"] = true
		result.Gates["cgroup_writable"] = true
	}

	result.Passed = true
	for name, passed := range result.Gates {
		if name == "dirty_worktree_risk" {
			continue
		}
		if !passed {
			result.Passed = false
			break
		}
	}
	if !result.Passed {
		if err != nil {
			return result, fmt.Errorf("preflight failed: %w", err)
		}
		return result, fmt.Errorf("preflight failed")
	}
	return result, nil
}

func gitAvailable(gitPath string) bool {
	if gitPath == "" {
		gitPath = "git"
	}
	_, err := exec.LookPath(gitPath)
	return err == nil
}

func probeCgroupV2() (present, writable bool) {
	root := "/sys/fs/cgroup"
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return false, false
	}
	present = true
	testDir := filepath.Join(root, "aort.slice", "aort-preflight-probe")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		return present, false
	}
	_ = os.Remove(testDir)
	return present, true
}
