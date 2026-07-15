package review

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAgentDemoMockHasSixRolesAndContinuesAfterFault(t *testing.T) {
	out := t.TempDir()
	result, err := RunAgentDemo(context.Background(), AgentDemoConfig{
		Provider: "mock",
		Seed:     17,
		Timeout:  3 * time.Second,
		OutDir:   out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "passed" || len(result.Agents) != 6 || len(result.LLMCalls) < 1 || len(result.ToolCalls) < 3 {
		t.Fatalf("demo contract not met: %+v", result)
	}
	if !result.Fault.Injected || !result.Fault.Contained || !result.Fault.Continued {
		t.Fatalf("fault contract not met: %+v", result.Fault)
	}
	if result.ProviderActual != "mock" || result.EvidenceMode != "mock" {
		t.Fatalf("provider evidence = %s/%s", result.ProviderActual, result.EvidenceMode)
	}
	for _, path := range []string{"timeline.json", "final_result.json", "summary.json", "report.md"} {
		if _, err := os.Stat(filepath.Join(out, path)); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
}

func TestAgentDemoDeepSeekReadsEnvAndRedactsSecret(t *testing.T) {
	secret := "deepseek-test-secret-value"
	t.Setenv("AORT_ENABLE_REAL_LLM", "1")
	t.Setenv("DEEPSEEK_API_KEY", secret)
	client := &http.Client{Transport: demoRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer "+secret {
			t.Fatalf("authorization header was not sourced from env")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(`{"choices":[{"message":{"content":"accepted"}}],"usage":{"prompt_tokens":8,"completion_tokens":2,"total_tokens":10}}`)),
		}, nil
	})}
	out := t.TempDir()
	result, err := RunAgentDemo(context.Background(), AgentDemoConfig{
		Provider: "deepseek",
		BaseURL:  "https://deepseek.test",
		Model:    "deepseek-test",
		Client:   client,
		Seed:     19,
		Timeout:  3 * time.Second,
		OutDir:   out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ProviderActual != "deepseek" || result.EvidenceMode != "real-api" || !result.APIKeyRedacted {
		t.Fatalf("deepseek evidence = %+v", result)
	}
	for _, name := range []string{"timeline.json", "final_result.json", "summary.json", "report.md"} {
		data, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), secret) {
			t.Fatalf("secret leaked in %s", name)
		}
	}
}

type demoRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn demoRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestAgentDemoDeepSeekWithoutKeyIsExplicitlySkipped(t *testing.T) {
	t.Setenv("AORT_ENABLE_REAL_LLM", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	result, err := RunAgentDemo(context.Background(), AgentDemoConfig{Provider: "deepseek", OutDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "skipped" || result.EvidenceMode != "missing" || result.FailureReason == "" {
		t.Fatalf("skipped evidence = %+v", result)
	}
}

func TestAgentDemoRejectsUnknownProvider(t *testing.T) {
	_, err := RunAgentDemo(context.Background(), AgentDemoConfig{Provider: "unknown", OutDir: t.TempDir()})
	if err == nil {
		t.Fatal("unknown provider should fail")
	}
}
