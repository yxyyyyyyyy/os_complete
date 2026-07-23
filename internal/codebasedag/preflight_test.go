package codebasedag

import (
	"context"
	"testing"
)

func TestLocalPreflightPortableGates(t *testing.T) {
	repo := newTestGitRepo(t)
	writeFile(t, repo, "alpha/a.go", []byte("package alpha\nfunc A() {}\n"), 0o644)
	writeFile(t, repo, "beta/b.go", []byte("package beta\nfunc B() {}\n"), 0o644)
	writeFile(t, repo, "README.md", []byte("# test\n"), 0o644)
	writeFile(t, repo, ".gitignore", []byte("ignored.go\n"), 0o644)

	preflight := LocalPreflight{
		ManifestOptions: ManifestOptions{MinPhysical: 2, MinNonblank: 2, GitPath: fakeGitPath(t)},
		RequireAPIKey:   false,
	}
	result, err := preflight.Check(context.Background(), RunnerConfig{
		RunID:       "run",
		WorkloadDir: repo,
		Provider:    RequiredDeepSeekProvider,
		Model:       RequiredDeepSeekModel,
		MaxCalls:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || !result.Gates["provider_model"] || !result.Gates["source_manifest"] || !result.Gates["go_runtime"] {
		t.Fatalf("preflight = %#v", result)
	}
	if result.Manifest.PhysicalLines != 4 || result.Manifest.NonblankLines != 4 {
		t.Fatalf("manifest = %#v", result.Manifest)
	}
}

func TestLocalPreflightTableFailures(t *testing.T) {
	repo := newTestGitRepo(t)
	writeFile(t, repo, "alpha/a.go", []byte("package alpha\n"), 0o644)
	writeFile(t, repo, "beta/b.go", []byte("package beta\n"), 0o644)
	writeFile(t, repo, "README.md", []byte("# test\n"), 0o644)
	writeFile(t, repo, ".gitignore", []byte("ignored.go\n"), 0o644)

	cases := []struct {
		name      string
		cfg       RunnerConfig
		preflight LocalPreflight
		gate      string
	}{
		{
			name: "wrong model",
			cfg: RunnerConfig{
				RunID: "run", WorkloadDir: repo, Provider: RequiredDeepSeekProvider, Model: "deepseek-chat", MaxCalls: 10,
			},
			preflight: LocalPreflight{ManifestOptions: ManifestOptions{MinPhysical: 1, MinNonblank: 1, GitPath: fakeGitPath(t)}},
			gate:      "provider_model",
		},
		{
			name: "line gate",
			cfg: RunnerConfig{
				RunID: "run", WorkloadDir: repo, Provider: RequiredDeepSeekProvider, Model: RequiredDeepSeekModel, MaxCalls: 10,
			},
			preflight: LocalPreflight{ManifestOptions: ManifestOptions{MinPhysical: 999, MinNonblank: 999, GitPath: fakeGitPath(t)}},
			gate:      "source_manifest",
		},
		{
			name: "missing api key",
			cfg: RunnerConfig{
				RunID: "run", WorkloadDir: repo, Provider: RequiredDeepSeekProvider, Model: RequiredDeepSeekModel, MaxCalls: 10,
			},
			preflight: LocalPreflight{ManifestOptions: ManifestOptions{MinPhysical: 1, MinNonblank: 1, GitPath: fakeGitPath(t)}, RequireAPIKey: true},
			gate:      "api_key_present",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DEEPSEEK_API_KEY", "")
			result, err := tc.preflight.Check(context.Background(), tc.cfg)
			if err == nil {
				t.Fatal("preflight should fail")
			}
			if result.Passed || result.Gates[tc.gate] {
				t.Fatalf("gate %q should fail: %#v", tc.gate, result)
			}
		})
	}
}
