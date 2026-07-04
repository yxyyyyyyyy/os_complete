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

type Config struct {
	CVM           *cvm.Store
	Sink          EventSink
	WorkspaceRoot string
	ToolTimeout   time.Duration
	Reporter      func(Report)
}

type Gateway struct {
	mu            sync.RWMutex
	cvm           *cvm.Store
	sink          EventSink
	workspaceRoot string
	toolTimeout   time.Duration
	reporter      func(Report)
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
		sink:          cfg.Sink,
		workspaceRoot: root,
		toolTimeout:   timeout,
		reporter:      cfg.Reporter,
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
	case "tool.exec":
		return g.toolExec(ctx, req)
	case "agent.report":
		return g.agentReport(req)
	default:
		return Response{RequestID: req.RequestID, Status: StatusDenied, Error: "unsupported syscall " + req.Name}
	}
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
