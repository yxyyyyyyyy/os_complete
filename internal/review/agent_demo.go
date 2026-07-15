package review

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aort-r/internal/cvm"
	"aort-r/internal/events"
	"aort-r/internal/ipc"
	"aort-r/internal/llm"
	syscallgw "aort-r/internal/syscall"
)

type AgentDemoConfig struct {
	Provider string
	Seed     int64
	Timeout  time.Duration
	OutDir   string
	BaseURL  string
	Model    string
	Client   *http.Client
}

type DemoAgent struct {
	ID     string `json:"id"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

type DemoLLMCall struct {
	AgentID           string      `json:"agent_id"`
	Provider          string      `json:"provider"`
	Model             string      `json:"model"`
	EvidenceMode      string      `json:"evidence_mode"`
	PromptTokens      int         `json:"prompt_tokens"`
	CompletionTokens  int         `json:"completion_tokens"`
	NetworkMS         MetricValue `json:"network_ms"`
	ModelMS           MetricValue `json:"model_ms"`
	RuntimeOverheadMS MetricValue `json:"runtime_overhead_ms"`
}

type DemoToolCall struct {
	AgentID    string `json:"agent_id"`
	Command    string `json:"command"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	ExitCode   int    `json:"exit_code"`
}

type DemoFault struct {
	Injected  bool   `json:"injected"`
	Type      string `json:"type"`
	AgentID   string `json:"agent_id"`
	Contained bool   `json:"contained"`
	Continued bool   `json:"continued"`
	Reason    string `json:"reason,omitempty"`
}

type AgentDemoResult struct {
	SchemaVersion     string         `json:"schema_version"`
	ScenarioID        string         `json:"scenario_id"`
	RunID             string         `json:"run_id"`
	Timestamp         string         `json:"timestamp"`
	Seed              int64          `json:"seed"`
	Status            string         `json:"status"`
	FailureReason     string         `json:"failure_reason,omitempty"`
	ProviderRequested string         `json:"provider_requested"`
	ProviderActual    string         `json:"provider_actual"`
	EvidenceMode      string         `json:"evidence_mode"`
	APIKeySource      string         `json:"api_key_source"`
	APIKeyPresent     bool           `json:"api_key_present"`
	APIKeyRedacted    bool           `json:"api_key_redacted"`
	Agents            []DemoAgent    `json:"agents"`
	LLMCalls          []DemoLLMCall  `json:"llm_calls"`
	ToolCalls         []DemoToolCall `json:"tool_calls"`
	Fault             DemoFault      `json:"fault"`
	Timeline          []events.Event `json:"timeline"`
	ArtifactPaths     []string       `json:"artifact_paths"`
	Limitations       []string       `json:"limitations"`
}

func RunAgentDemo(parent context.Context, cfg AgentDemoConfig) (AgentDemoResult, error) {
	if cfg.Provider == "" {
		cfg.Provider = "mock"
	}
	if cfg.Provider != "mock" && cfg.Provider != "deepseek" {
		return AgentDemoResult{}, fmt.Errorf("unsupported agent demo provider %q", cfg.Provider)
	}
	if cfg.Seed == 0 {
		cfg.Seed = 20260713
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.OutDir == "" {
		cfg.OutDir = filepath.Join("experiments", "results", "review_remediation", "real_agent_demo")
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, cfg.Timeout)
	defer cancel()
	secret := os.Getenv("DEEPSEEK_API_KEY")
	result := AgentDemoResult{
		SchemaVersion:     SchemaVersion,
		ScenarioID:        "real-agent-demo",
		RunID:             fmt.Sprintf("real-agent-demo-%d", time.Now().UnixNano()),
		Timestamp:         time.Now().UTC().Format(time.RFC3339Nano),
		Seed:              cfg.Seed,
		Status:            "running",
		ProviderRequested: cfg.Provider,
		EvidenceMode:      "mock",
		APIKeySource:      "env",
		APIKeyPresent:     secret != "",
		APIKeyRedacted:    true,
		Agents: []DemoAgent{
			{ID: "planner-1", Role: "Planner", Status: "READY"},
			{ID: "coder-a-1", Role: "Coder-A", Status: "READY"},
			{ID: "coder-b-1", Role: "Coder-B", Status: "READY"},
			{ID: "tester-1", Role: "Tester", Status: "READY"},
			{ID: "reviewer-1", Role: "Reviewer", Status: "READY"},
			{ID: "fault-agent-1", Role: "Fault-Agent", Status: "READY"},
		},
		LLMCalls:      []DemoLLMCall{},
		ToolCalls:     []DemoToolCall{},
		Timeline:      []events.Event{},
		ArtifactPaths: []string{"timeline.json", "final_result.json", "summary.json", "report.md"},
		Limitations: []string{
			"Mock mode is the repeatable performance/demo path; DeepSeek mode is availability evidence only.",
			"DeepSeek does not expose separate remote model execution time, so that field is unsupported while end-to-end network/API time is measured.",
		},
	}
	if cfg.Provider == "deepseek" && (os.Getenv("AORT_ENABLE_REAL_LLM") != "1" || secret == "") {
		result.Status = "skipped"
		result.EvidenceMode = "missing"
		result.FailureReason = "DeepSeek demo requires AORT_ENABLE_REAL_LLM=1 and DEEPSEEK_API_KEY in the environment"
		if err := writeAgentDemoArtifacts(cfg.OutDir, &result, secret); err != nil {
			return result, err
		}
		return result, nil
	}

	workspaceRoot, err := os.MkdirTemp("", "aort-review-agent-demo-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(workspaceRoot)
	sink := &recordingSink{}
	store := cvm.NewStore(sink)
	blackboard := ipc.NewBlackboard()
	router := llm.NewRouter()
	router.Register("mock", llm.NewMockProvider("mock"))
	if cfg.Provider == "deepseek" {
		baseURL := firstText(cfg.BaseURL, os.Getenv("AORT_LLM_BASE_URL"), os.Getenv("DEEPSEEK_BASE_URL"))
		model := firstText(cfg.Model, os.Getenv("AORT_LLM_MODEL"), os.Getenv("DEEPSEEK_MODEL"))
		router.Register("deepseek", llm.NewDeepSeekProvider(llm.DeepSeekConfig{APIKey: secret, BaseURL: baseURL, Model: model, Client: cfg.Client}))
		router.SetDefault("deepseek")
	} else {
		router.SetDefault("mock")
	}
	page, err := store.CreatePage(cvm.KindProject, fmt.Sprintf("AORT-R review task seed=%d", cfg.Seed))
	if err != nil {
		return result, err
	}
	if err := store.MountPage("planner-1", page.ID); err != nil {
		return result, err
	}
	gateway := syscallgw.NewGateway(syscallgw.Config{
		CVM:           store,
		IPC:           blackboard,
		LLM:           router,
		Sink:          sink,
		WorkspaceRoot: workspaceRoot,
		ToolTimeout:   2 * time.Second,
	})
	materialized := gateway.Handle(ctx, syscallgw.Request{RequestID: "context-1", AgentID: "planner-1", TaskID: result.RunID, Name: "context.materialize"})
	if materialized.Status != syscallgw.StatusOK {
		result.Status = "failed"
		result.FailureReason = materialized.Error
		result.Timeline = sink.Events()
		_ = writeAgentDemoArtifacts(cfg.OutDir, &result, secret)
		return result, fmt.Errorf("context materialize: %s", materialized.Error)
	}
	llmStarted := time.Now()
	llmResponse := gateway.Handle(ctx, syscallgw.Request{
		RequestID: "llm-1",
		AgentID:   "planner-1",
		TaskID:    result.RunID,
		Name:      "llm.call",
		Args: map[string]any{
			"provider": cfg.Provider,
			"role":     "planner",
			"prompt":   fmt.Sprintf("Plan a six-agent review task with seed %d.", cfg.Seed),
		},
	})
	llmElapsedMS := max64(1, time.Since(llmStarted).Milliseconds())
	if llmResponse.Status != syscallgw.StatusOK {
		result.Status = "failed"
		result.FailureReason = redactText(llmResponse.Error, secret)
		result.Timeline = sanitizeEvents(sink.Events(), secret)
		_ = writeAgentDemoArtifacts(cfg.OutDir, &result, secret)
		return result, fmt.Errorf("llm.call failed: %s", result.FailureReason)
	}
	usage := mapValue(llmResponse.Payload["usage"])
	providerActual := textValue(llmResponse.Payload["provider"])
	model := textValue(llmResponse.Payload["model"])
	evidenceMode := textValue(llmResponse.Payload["evidence_mode"])
	providerMS := int64Value(usage["total_ms"])
	if providerMS <= 0 {
		providerMS = llmElapsedMS
	}
	network := MetricValue{Value: 0, Kind: MeasurementUnsupported, Unit: "ms"}
	modelTime := MetricValue{Value: float64(providerMS), Kind: MeasurementMeasured, Unit: "ms"}
	if providerActual == "deepseek" {
		network = MetricValue{Value: float64(providerMS), Kind: MeasurementMeasured, Unit: "ms"}
		modelTime = MetricValue{Value: 0, Kind: MeasurementUnsupported, Unit: "ms"}
	}
	result.LLMCalls = append(result.LLMCalls, DemoLLMCall{
		AgentID:           "planner-1",
		Provider:          providerActual,
		Model:             model,
		EvidenceMode:      evidenceMode,
		PromptTokens:      int(int64Value(usage["prompt_tokens"])),
		CompletionTokens:  int(int64Value(usage["completion_tokens"])),
		NetworkMS:         network,
		ModelMS:           modelTime,
		RuntimeOverheadMS: MetricValue{Value: float64(max64(0, llmElapsedMS-providerMS)), Kind: MeasurementDerived, Unit: "ms"},
	})
	result.ProviderActual = providerActual
	result.EvidenceMode = evidenceMode

	toolSpecs := []struct {
		agentID string
		command string
	}{
		{"coder-a-1", "true"},
		{"coder-b-1", "true"},
		{"fault-agent-1", "false"},
		{"tester-1", "true"},
		{"reviewer-1", "true"},
	}
	faultSeen := false
	continued := false
	for _, spec := range toolSpecs {
		call := runDemoTool(ctx, gateway, result.RunID, spec.agentID, spec.command)
		result.ToolCalls = append(result.ToolCalls, call)
		if spec.agentID == "fault-agent-1" {
			faultSeen = call.Status != syscallgw.StatusOK
			result.Fault = DemoFault{Injected: true, Type: "tool_process_failure", AgentID: spec.agentID, Contained: faultSeen, Reason: "controlled false command returned non-zero"}
			setDemoAgentStatus(result.Agents, spec.agentID, "FAILED_RECOVERABLE")
			continue
		}
		if faultSeen && call.Status == syscallgw.StatusOK {
			continued = true
		}
		if call.Status == syscallgw.StatusOK {
			setDemoAgentStatus(result.Agents, spec.agentID, "COMPLETED")
		}
	}
	result.Fault.Continued = continued
	setDemoAgentStatus(result.Agents, "planner-1", "COMPLETED")
	if len(result.LLMCalls) >= 1 && len(result.ToolCalls) >= 3 && result.Fault.Injected && result.Fault.Contained && result.Fault.Continued {
		result.Status = "passed"
	} else {
		result.Status = "failed"
		result.FailureReason = "demo acceptance contract was not met"
	}
	result.Timeline = sanitizeEvents(sink.Events(), secret)
	if err := writeAgentDemoArtifacts(cfg.OutDir, &result, secret); err != nil {
		return result, err
	}
	if result.Status != "passed" {
		return result, fmt.Errorf("%s", result.FailureReason)
	}
	return result, nil
}

func runDemoTool(ctx context.Context, gateway *syscallgw.Gateway, taskID, agentID, command string) DemoToolCall {
	started := time.Now()
	response := gateway.Handle(ctx, syscallgw.Request{
		RequestID: "tool-" + agentID,
		AgentID:   agentID,
		TaskID:    taskID,
		Name:      "tool.exec",
		Args:      map[string]any{"command": command, "timeout_ms": 1000},
	})
	exitCode := 0
	if response.Payload != nil {
		exitCode = int(int64Value(response.Payload["exit_code"]))
	}
	return DemoToolCall{AgentID: agentID, Command: command, Status: response.Status, DurationMS: max64(1, time.Since(started).Milliseconds()), ExitCode: exitCode}
}

func writeAgentDemoArtifacts(outDir string, result *AgentDemoResult, secret string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	result.FailureReason = redactText(result.FailureReason, secret)
	result.Timeline = sanitizeEvents(result.Timeline, secret)
	if err := writeJSONFile(filepath.Join(outDir, "timeline.json"), result.Timeline); err != nil {
		return err
	}
	final := map[string]any{
		"schema_version":        result.SchemaVersion,
		"scenario_id":           result.ScenarioID,
		"run_id":                result.RunID,
		"status":                result.Status,
		"provider":              result.ProviderActual,
		"completed_agents":      completedAgentCount(result.Agents),
		"llm_call_count":        len(result.LLMCalls),
		"tool_call_count":       len(result.ToolCalls),
		"fault_contained":       result.Fault.Contained,
		"continued_after_fault": result.Fault.Continued,
	}
	if err := writeJSONFile(filepath.Join(outDir, "final_result.json"), final); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(outDir, "summary.json"), result); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "report.md"), []byte(renderAgentDemoReport(*result)), 0o644)
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func renderAgentDemoReport(result AgentDemoResult) string {
	return fmt.Sprintf("# Real Agent Demo\n\n- status: `%s`\n- requested provider: `%s`\n- actual provider: `%s`\n- evidence_mode: `%s`\n- agents: %d\n- llm.call: %d\n- tool.exec: %d\n- fault contained: %t\n- continued after fault: %t\n- API key source: env (redacted=%t)\n\nMock is used for repeatable execution; DeepSeek is optional availability evidence.\n", result.Status, result.ProviderRequested, result.ProviderActual, result.EvidenceMode, len(result.Agents), len(result.LLMCalls), len(result.ToolCalls), result.Fault.Contained, result.Fault.Continued, result.APIKeyRedacted)
}

type recordingSink struct {
	mu     sync.Mutex
	events []events.Event
}

func (s *recordingSink) Publish(event events.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *recordingSink) Events() []events.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]events.Event(nil), s.events...)
}

func sanitizeEvents(input []events.Event, secret string) []events.Event {
	output := make([]events.Event, len(input))
	for i, event := range input {
		output[i] = event
		output[i].Payload = sanitizeMap(event.Payload, secret)
	}
	return output
}

func sanitizeMap(input map[string]any, secret string) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case string:
			output[key] = redactText(typed, secret)
		case map[string]any:
			output[key] = sanitizeMap(typed, secret)
		case []string:
			items := make([]string, len(typed))
			for i, item := range typed {
				items[i] = redactText(item, secret)
			}
			output[key] = items
		default:
			output[key] = value
		}
	}
	return output
}

func redactText(value, secret string) string {
	if secret != "" {
		value = strings.ReplaceAll(value, secret, "[REDACTED]")
	}
	for _, marker := range []string{"Bearer ", "api_key=", "api-key="} {
		if index := strings.Index(strings.ToLower(value), strings.ToLower(marker)); index >= 0 {
			end := strings.IndexAny(value[index+len(marker):], " \t\r\n,;\"")
			if end < 0 {
				end = len(value) - index - len(marker)
			}
			value = value[:index+len(marker)] + "[REDACTED]" + value[index+len(marker)+end:]
		}
	}
	return value
}

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func textValue(value any) string {
	text, _ := value.(string)
	return text
}

func int64Value(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func setDemoAgentStatus(agents []DemoAgent, agentID, status string) {
	for i := range agents {
		if agents[i].ID == agentID {
			agents[i].Status = status
		}
	}
}

func completedAgentCount(agents []DemoAgent) int {
	count := 0
	for _, agent := range agents {
		if strings.HasPrefix(agent.Status, "COMPLETED") {
			count++
		}
	}
	return count
}

func firstText(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
