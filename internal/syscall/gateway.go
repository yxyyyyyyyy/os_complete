package syscallgw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"aort-r/internal/cvm"
	"aort-r/internal/events"
	"aort-r/internal/ipc"
	"aort-r/internal/llm"
)

const (
	StatusOK      = "OK"
	StatusError   = "ERROR"
	StatusTimeout = "TIMEOUT"
	StatusDenied  = "DENIED"
)

type EventSink interface {
	Publish(events.Event)
}

type Request struct {
	RequestID string         `json:"request_id"`
	AgentID   string         `json:"agent_id"`
	TaskID    string         `json:"task_id"`
	Name      string         `json:"name"`
	Args      map[string]any `json:"args,omitempty"`
}

type Response struct {
	RequestID string         `json:"request_id"`
	Status    string         `json:"status"`
	Error     string         `json:"error,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type Record struct {
	ID         string `json:"id"`
	AgentID    string `json:"agent_id"`
	Name       string `json:"name"`
	StartTime  int64  `json:"start_time"`
	EndTime    int64  `json:"end_time"`
	DurationMS int64  `json:"duration_ms"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	InputSize  int    `json:"input_size"`
	OutputSize int    `json:"output_size"`
}

type Report struct {
	AgentID string
	TaskID  string
	Status  string
	Payload map[string]any
}

type SpawnRequest struct {
	AgentID       string
	TaskID        string
	ParentAgentID string
	Role          string
	Reason        string
	Dependencies  []string
}

type SpawnResult struct {
	AgentID string `json:"agent_id"`
	Role    string `json:"role"`
	TaskID  string `json:"task_id"`
	State   string `json:"state"`
}

type Config struct {
	CVM           *cvm.Store
	IPC           *ipc.Blackboard
	LLM           *llm.Router
	Sink          EventSink
	WorkspaceRoot string
	ToolTimeout   time.Duration
	Reporter      func(Report)
	Spawner       func(SpawnRequest) (SpawnResult, error)
}

type Gateway struct {
	mu            sync.RWMutex
	cvm           *cvm.Store
	ipc           *ipc.Blackboard
	llm           *llm.Router
	sink          EventSink
	workspaceRoot string
	toolTimeout   time.Duration
	reporter      func(Report)
	spawner       func(SpawnRequest) (SpawnResult, error)
	records       []Record
}

func NewGateway(cfg Config) *Gateway {
	timeout := cfg.ToolTimeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	root := cfg.WorkspaceRoot
	if root == "" {
		root = filepath.Join(os.TempDir(), "aort-workspaces")
	}
	return &Gateway{
		cvm:           cfg.CVM,
		ipc:           cfg.IPC,
		llm:           cfg.LLM,
		sink:          cfg.Sink,
		workspaceRoot: root,
		toolTimeout:   timeout,
		reporter:      cfg.Reporter,
		spawner:       cfg.Spawner,
	}
}

func (g *Gateway) Handle(ctx context.Context, req Request) Response {
	if req.RequestID == "" {
		req.RequestID = time.Now().Format("20060102150405.000000000")
	}
	start := time.Now()
	inputSize := jsonSize(req.Args)
	g.publish("syscall.started", req, map[string]any{
		"name":       req.Name,
		"request_id": req.RequestID,
		"input_size": inputSize,
	})

	resp := g.execute(ctx, req)
	end := time.Now()
	outputSize := jsonSize(resp.Payload)
	duration := end.Sub(start).Milliseconds()
	if duration == 0 {
		duration = 1
	}
	record := Record{
		ID:         req.RequestID,
		AgentID:    req.AgentID,
		Name:       req.Name,
		StartTime:  start.UnixMilli(),
		EndTime:    end.UnixMilli(),
		DurationMS: duration,
		Status:     resp.Status,
		Error:      resp.Error,
		InputSize:  inputSize,
		OutputSize: outputSize,
	}
	g.mu.Lock()
	g.records = append(g.records, record)
	g.mu.Unlock()
	g.publish("syscall.finished", req, map[string]any{
		"name":        req.Name,
		"request_id":  req.RequestID,
		"duration_ms": duration,
		"status":      resp.Status,
		"error":       resp.Error,
		"input_size":  inputSize,
		"output_size": outputSize,
	})
	return resp
}

func (g *Gateway) Records() []Record {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]Record, len(g.records))
	copy(out, g.records)
	return out
}

func (g *Gateway) execute(ctx context.Context, req Request) Response {
	switch req.Name {
	case "context.materialize":
		return g.contextMaterialize(req)
	case "context.write_delta":
		return g.contextWriteDelta(req)
	case "ipc.publish":
		return g.ipcPublish(req)
	case "ipc.poll":
		return g.ipcPoll(req)
	case "llm.call":
		return g.llmCall(ctx, req)
	case "tool.exec":
		return g.toolExec(ctx, req)
	case "agent.spawn":
		return g.agentSpawn(req)
	case "agent.report":
		return g.agentReport(req)
	default:
		return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "unsupported syscall " + req.Name}
	}
}

func (g *Gateway) llmCall(ctx context.Context, req Request) Response {
	if g.llm == nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: "llm router is not configured"}
	}
	prompt, _ := stringArg(req.Args, "prompt")
	if prompt == "" && g.cvm != nil {
		content, err := g.cvm.Materialize(req.AgentID)
		if err != nil {
			return Response{RequestID: req.RequestID, Status: StatusError, Error: err.Error()}
		}
		prompt = content
	}
	if prompt == "" {
		return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "prompt is required"}
	}
	provider, _ := stringArg(req.Args, "provider")
	role, _ := stringArg(req.Args, "role")
	resp, usage, err := g.llm.Complete(ctx, llm.Request{
		AgentID:  req.AgentID,
		Role:     role,
		Provider: provider,
		Prompt:   prompt,
	})
	if err != nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: err.Error()}
	}
	payload := map[string]any{
		"text":          resp.Text,
		"provider":      resp.Provider,
		"fallback_from": resp.FallbackFrom,
		"usage":         usagePayload(usage),
	}
	g.publish("llm.called", req, map[string]any{
		"provider":      resp.Provider,
		"fallback_from": resp.FallbackFrom,
		"prompt_tokens": usage.PromptTokens,
		"cached_tokens": usage.CachedTokens,
		"ttft_ms":       usage.TTFTMS,
		"mode":          usage.Mode,
	})
	return Response{RequestID: req.RequestID, Status: StatusOK, Payload: payload}
}

func (g *Gateway) contextMaterialize(req Request) Response {
	if g.cvm == nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: "cvm store is not configured"}
	}
	content, err := g.cvm.Materialize(req.AgentID)
	if err != nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: err.Error()}
	}
	return Response{
		RequestID: req.RequestID,
		Status:    StatusOK,
		Payload: map[string]any{
			"content": content,
			"bytes":   len([]byte(content)),
			"tokens":  estimateTokens(content),
		},
	}
}

func (g *Gateway) contextWriteDelta(req Request) Response {
	if g.cvm == nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: "cvm store is not configured"}
	}
	content, ok := stringArg(req.Args, "content")
	if !ok {
		return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "content is required"}
	}
	page, err := g.cvm.WriteDelta(req.AgentID, content)
	if err != nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: err.Error()}
	}
	return Response{RequestID: req.RequestID, Status: StatusOK, Payload: map[string]any{"page_id": page.ID, "bytes": page.Bytes, "tokens": page.TokenCount}}
}

func (g *Gateway) ipcPublish(req Request) Response {
	if g.ipc == nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: "ipc blackboard is not configured"}
	}
	topic, ok := stringArg(req.Args, "topic")
	if !ok || topic == "" {
		return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "topic is required"}
	}
	pageID, ok := stringArg(req.Args, "page_id")
	if !ok || pageID == "" {
		return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "page_id is required"}
	}
	sizeBytes := intArg(req.Args, "size_bytes")
	if g.cvm != nil {
		page, exists := g.cvm.Page(pageID)
		if !exists {
			return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "unknown page_id " + pageID}
		}
		if sizeBytes <= 0 {
			sizeBytes = page.Bytes
		}
	}
	metric := g.ipc.Publish(ipc.PublishRequest{
		Topic:     topic,
		Publisher: req.AgentID,
		PageID:    pageID,
		SizeBytes: sizeBytes,
	})
	g.publish("ipc.published", req, map[string]any{
		"topic":              topic,
		"page_id":            pageID,
		"avoided_copy_bytes": metric.AvoidedCopyBytes,
		"topic_depth":        metric.TopicDepth,
	})
	return Response{RequestID: req.RequestID, Status: StatusOK, Payload: map[string]any{
		"topic":              topic,
		"page_id":            pageID,
		"avoided_copy_bytes": metric.AvoidedCopyBytes,
		"topic_depth":        metric.TopicDepth,
	}}
}

func (g *Gateway) ipcPoll(req Request) Response {
	if g.ipc == nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: "ipc blackboard is not configured"}
	}
	topic, ok := stringArg(req.Args, "topic")
	if !ok || topic == "" {
		return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "topic is required"}
	}
	messages, metric := g.ipc.Poll(topic, req.AgentID)
	pageIDs := make([]string, 0, len(messages))
	for _, message := range messages {
		pageIDs = append(pageIDs, message.PageID)
		if g.cvm != nil {
			if err := g.cvm.MountPage(req.AgentID, message.PageID); err != nil {
				return Response{RequestID: req.RequestID, Status: StatusError, Error: err.Error()}
			}
		}
	}
	g.publish("ipc.polled", req, map[string]any{
		"topic":              topic,
		"page_ids":           pageIDs,
		"delivered_messages": metric.DeliveredMessages,
		"avoided_copy_bytes": metric.AvoidedCopyBytes,
	})
	return Response{RequestID: req.RequestID, Status: StatusOK, Payload: map[string]any{
		"topic":              topic,
		"page_ids":           pageIDs,
		"delivered_messages": metric.DeliveredMessages,
		"avoided_copy_bytes": metric.AvoidedCopyBytes,
	}}
}

func (g *Gateway) toolExec(ctx context.Context, req Request) Response {
	command, ok := stringArg(req.Args, "command")
	if !ok || command == "" {
		return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "command is required"}
	}
	args := stringSliceArg(req.Args, "args")
	workspace, err := g.workspaceForAgent(req.AgentID)
	if err != nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: err.Error()}
	}
	cwd := workspace
	if requestedCWD, ok := stringArg(req.Args, "cwd"); ok && requestedCWD != "" {
		cwd, err = confinedPath(workspace, requestedCWD)
		if err != nil {
			return Response{RequestID: req.RequestID, Status: StatusDenied, Error: err.Error()}
		}
	}
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: err.Error()}
	}
	timeout := g.toolTimeout
	if ms := intArg(req.Args, "timeout_ms"); ms > 0 {
		timeout = time.Duration(ms) * time.Millisecond
	}
	toolCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(toolCtx, command, args...)
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	payload := map[string]any{
		"command":   command,
		"args":      args,
		"cwd":       cwd,
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"exit_code": 0,
	}
	if toolCtx.Err() == context.DeadlineExceeded {
		payload["exit_code"] = -1
		return Response{RequestID: req.RequestID, Status: StatusTimeout, Error: "tool timeout", Payload: payload}
	}
	if err != nil {
		payload["exit_code"] = exitCode(err)
		return Response{RequestID: req.RequestID, Status: StatusError, Error: err.Error(), Payload: payload}
	}
	return Response{RequestID: req.RequestID, Status: StatusOK, Payload: payload}
}

func (g *Gateway) agentReport(req Request) Response {
	status, ok := stringArg(req.Args, "status")
	if !ok || status == "" {
		return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "status is required"}
	}
	if g.reporter != nil {
		g.reporter(Report{AgentID: req.AgentID, TaskID: req.TaskID, Status: status, Payload: req.Args})
	}
	return Response{RequestID: req.RequestID, Status: StatusOK, Payload: map[string]any{"status": status}}
}

func (g *Gateway) agentSpawn(req Request) Response {
	if g.spawner == nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: "agent spawner is not configured"}
	}
	role, ok := stringArg(req.Args, "role")
	if !ok || role == "" {
		return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "role is required"}
	}
	agentID, _ := stringArg(req.Args, "agent_id")
	reason, _ := stringArg(req.Args, "reason")
	deps := stringSliceArg(req.Args, "dependencies")
	spawnReq := SpawnRequest{
		AgentID:       agentID,
		TaskID:        req.TaskID,
		ParentAgentID: req.AgentID,
		Role:          role,
		Reason:        reason,
		Dependencies:  deps,
	}
	g.publish("agent.spawn.requested", req, map[string]any{
		"agent_id":     agentID,
		"role":         role,
		"reason":       reason,
		"dependencies": deps,
	})
	result, err := g.spawner(spawnReq)
	if err != nil {
		return Response{RequestID: req.RequestID, Status: StatusError, Error: err.Error()}
	}
	g.publish("agent.spawned", req, map[string]any{
		"agent_id":        result.AgentID,
		"role":            result.Role,
		"state":           result.State,
		"parent_agent_id": req.AgentID,
		"reason":          reason,
	})
	return Response{RequestID: req.RequestID, Status: StatusOK, Payload: map[string]any{
		"agent_id": result.AgentID,
		"role":     result.Role,
		"task_id":  result.TaskID,
		"state":    result.State,
	}}
}

func (g *Gateway) workspaceForAgent(agentID string) (string, error) {
	if agentID == "" {
		return "", fmt.Errorf("agent_id is required")
	}
	cleanAgentID := strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(agentID)
	root, err := filepath.Abs(g.workspaceRoot)
	if err != nil {
		return "", err
	}
	workspace := filepath.Join(root, cleanAgentID)
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return "", err
	}
	return workspace, nil
}

func (g *Gateway) publish(eventType string, req Request, payload map[string]any) {
	if g.sink == nil {
		return
	}
	g.sink.Publish(events.New(eventType, req.TaskID, req.AgentID, "syscall", payload))
}

func confinedPath(root, requested string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	var candidate string
	if filepath.IsAbs(requested) {
		candidate = filepath.Clean(requested)
	} else {
		candidate = filepath.Join(rootAbs, requested)
	}
	rel, err := filepath.Rel(rootAbs, candidate)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("cwd %q escapes agent workspace", requested)
	}
	return candidate, nil
}

func stringArg(args map[string]any, key string) (string, bool) {
	value, ok := args[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}

func stringSliceArg(args map[string]any, key string) []string {
	value, ok := args[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func intArg(args map[string]any, key string) int {
	value, ok := args[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := strconv.Atoi(string(typed))
		return parsed
	default:
		return 0
	}
}

func jsonSize(value any) int {
	if value == nil {
		return 0
	}
	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return len(data)
}

func exitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}

func estimateTokens(content string) int {
	tokens := len([]rune(content)) / 4
	if tokens == 0 && content != "" {
		return 1
	}
	return tokens
}

func usagePayload(usage llm.Usage) map[string]any {
	return map[string]any{
		"prompt_tokens":     usage.PromptTokens,
		"completion_tokens": usage.CompletionTokens,
		"cached_tokens":     usage.CachedTokens,
		"prompt_ms":         usage.PromptMS,
		"ttft_ms":           usage.TTFTMS,
		"total_ms":          usage.TotalMS,
		"mode":              usage.Mode,
	}
}
