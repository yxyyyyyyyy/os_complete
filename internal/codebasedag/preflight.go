package codebasedag

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
)

type LocalPreflight struct {
	ManifestOptions ManifestOptions
	RequireAPIKey   bool
}

func (p LocalPreflight) Check(ctx context.Context, cfg RunnerConfig) (PreflightResult, error) {
	result := PreflightResult{Gates: make(map[string]bool)}
	result.Gates["provider_model"] = cfg.Provider == RequiredDeepSeekProvider && cfg.Model == RequiredDeepSeekModel
	result.Gates["go_runtime"] = strings.HasPrefix(runtime.Version(), "go1.")
	if p.RequireAPIKey {
		result.Gates["api_key_present"] = os.Getenv("DEEPSEEK_API_KEY") != ""
	} else {
		result.Gates["api_key_present"] = true
	}
	manifest, _, err := BuildSourceManifestWithOptions(ctx, cfg.WorkloadDir, p.ManifestOptions)
	if err == nil {
		result.Manifest = manifest
		result.Gates["source_manifest"] = true
	} else {
		result.Gates["source_manifest"] = false
	}
	result.Passed = true
	for _, passed := range result.Gates {
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
