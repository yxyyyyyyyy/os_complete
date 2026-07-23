package codebasedag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aort-r/internal/llm"
)

// LiveSession holds mutable evidence for one live codebase-dag run.
type LiveSession struct {
	Store         *RunStore
	Model         *StrictModel
	Ticket        Ticket
	WorkloadDir   string
	GitPath       string
	WorktreeRoot  string
	Worktree      *WorktreeSession
	WorkerCommand string
	WorkerArgs    []string
	SkipRace      bool
	AllowDirty    bool
	TestTimeout   time.Duration
	ProcessRT     ProcessRuntime
	Limits        ResourceLimits
	MinPhysical   int
	MinNonblank   int

	PatchTexts    map[string]string
	PatchRecords  []PatchRecord
	Applies       []PatchApplyResult
	Tests         []TestRecord
	Commands      []CommandEvidence
	Processes     []ProcessResult
	ReviewVerdict string
	Nodes         []NodeRecord
	Manifest      SourceManifest
	Clock         func() time.Time
	schemaRepairs int
}

func newLiveSession(store *RunStore, model *StrictModel, ticket Ticket, workload string) *LiveSession {
	return &LiveSession{
		Store:       store,
		Model:       model,
		Ticket:      ticket,
		WorkloadDir: workload,
		GitPath:     "git",
		PatchTexts:  map[string]string{},
		Clock:       time.Now,
		Limits:      DefaultResourceLimits(),
	}
}

// LiveNodeExecutor runs LLM and tool nodes against a shared LiveSession.
type LiveNodeExecutor struct {
	Session *LiveSession
}

func NewLiveNodeExecutor(model *StrictModel, ticket Ticket, store *RunStore) LiveNodeExecutor {
	return LiveNodeExecutor{Session: newLiveSession(store, model, ticket, "")}
}

func (e LiveNodeExecutor) ExecuteNode(ctx context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
	if e.Session == nil {
		return NodeExecutionResult{}, fmt.Errorf("live session is required")
	}
	s := e.Session
	switch {
	case req.NodeID == "integrate":
		return s.executeIntegrate(ctx, req)
	case req.NodeID == "tester" || strings.HasPrefix(req.NodeID, "tester-recheck-"):
		return s.executeTester(ctx, req)
	case strings.HasPrefix(req.NodeID, "fixer-"):
		return s.executeFixer(ctx, req)
	default:
		return s.executeLLMNode(ctx, req)
	}
}

func (s *LiveSession) executeLLMNode(ctx context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
	if s.Model == nil {
		return NodeExecutionResult{}, fmt.Errorf("strict model is required")
	}
	policy, ok := s.Ticket.NodePolicies[req.NodeID]
	if !ok && strings.HasPrefix(req.NodeID, "fixer-") {
		policy = NodePolicy{
			Role:           KindFixer,
			AllowedFiles:   collectAllowlist(s.Ticket),
			ImmutableFiles: acceptanceScriptNames(),
			PrivateContext: "Emit a structured unified diff that repairs the latest review blocking findings.",
		}
	}
	if policy.Role == "" {
		return NodeExecutionResult{}, fmt.Errorf("node policy for %q is missing", req.NodeID)
	}
	prompt, err := s.buildPrompt(req.NodeID, policy)
	if err != nil {
		return NodeExecutionResult{}, err
	}
	text, record, err := s.Model.Complete(ctx, ModelRequest{NodeID: req.NodeID, Role: string(policy.Role), Prompt: prompt})
	if err != nil {
		return NodeExecutionResult{}, err
	}
	decoded, err := DecodeRoleOutput(policy.Role, req.NodeID, []byte(text))
	if err != nil {
		text, record, decoded, err = s.attemptSchemaRepair(ctx, req.NodeID, policy, text, record, err)
		if err != nil {
			return NodeExecutionResult{}, err
		}
	}
	switch policy.Role {
	case KindCoder, KindFixer:
		coder := decoded.(CoderOutput)
		materializeDir := s.WorkloadDir
		if s.Worktree != nil && s.Worktree.WorkDir != "" {
			materializeDir = s.Worktree.WorkDir
		}
		if err := MaterializeCoderPatch(materializeDir, policy.AllowedFiles, &coder); err != nil {
			text, record, decoded, err = s.attemptSchemaRepair(ctx, req.NodeID, policy, text, record, err)
			if err != nil {
				return NodeExecutionResult{}, err
			}
			coder = decoded.(CoderOutput)
			if matErr := MaterializeCoderPatch(materializeDir, policy.AllowedFiles, &coder); matErr != nil {
				return NodeExecutionResult{}, matErr
			}
		}
		allowed := map[string]struct{}{}
		for _, p := range policy.AllowedFiles {
			allowed[p] = struct{}{}
		}
		immutable := map[string]string{}
		for _, p := range policy.ImmutableFiles {
			immutable[p] = p
		}
		patchRec, patchErr := ValidatePatch(PatchPolicy{
			NodeID:         req.NodeID,
			AllowedFiles:   allowed,
			ImmutableFiles: immutable,
		}, coder, record.CallID)
		if patchErr == nil {
			patchErr = CheckPatchApplies(ctx, s.GitPath, materializeDir, coder.Patch)
		}
		if patchErr != nil {
			text, record, decoded, err = s.attemptSchemaRepair(ctx, req.NodeID, policy, text, record, patchErr)
			if err != nil {
				return NodeExecutionResult{}, err
			}
			coder = decoded.(CoderOutput)
			if matErr := MaterializeCoderPatch(materializeDir, policy.AllowedFiles, &coder); matErr != nil {
				return NodeExecutionResult{}, matErr
			}
			patchRec, patchErr = ValidatePatch(PatchPolicy{
				NodeID:         req.NodeID,
				AllowedFiles:   allowed,
				ImmutableFiles: immutable,
			}, coder, record.CallID)
			if patchErr == nil {
				patchErr = CheckPatchApplies(ctx, s.GitPath, materializeDir, coder.Patch)
			}
			if patchErr != nil {
				return NodeExecutionResult{}, patchErr
			}
		}
		decoded = coder
		s.PatchTexts[req.NodeID] = coder.Patch
		s.PatchRecords = append(s.PatchRecords, patchRec)
		if err := s.Store.WriteBytes(fmt.Sprintf("patches/%s.patch", req.NodeID), []byte(coder.Patch), 0o600); err != nil {
			return NodeExecutionResult{}, err
		}
	case KindReviewer:
		review := decoded.(ReviewOutput)
		s.ReviewVerdict = review.Verdict
		if review.Verdict == "reject" {
			return NodeExecutionResult{}, fmt.Errorf("reviewer rejected: %v", review.Blocking)
		}
	case KindTester:
		review := decoded.(ReviewOutput)
		if review.Verdict != "pass" {
			return NodeExecutionResult{}, fmt.Errorf("tester model verdict %q is not pass", review.Verdict)
		}
	case KindFinalizer:
		final := decoded.(FinalOutput)
		if final.Status != "passed" {
			return NodeExecutionResult{}, fmt.Errorf("finalizer status %q", final.Status)
		}
		if s.ReviewVerdict != "" && s.ReviewVerdict != "pass" {
			return NodeExecutionResult{}, fmt.Errorf("finalizer refused: review verdict is %q", s.ReviewVerdict)
		}
	case KindPlanner:
		_ = decoded.(PlanOutput)
	}

	payload, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return NodeExecutionResult{}, err
	}
	payload = append(payload, '\n')
	if err := s.Store.WriteBytes(fmt.Sprintf("outputs/%s.json", req.NodeID), payload, 0o600); err != nil {
		return NodeExecutionResult{}, err
	}
	if err := s.Store.AppendJSONL("llm_calls.jsonl", record); err != nil {
		return NodeExecutionResult{}, err
	}
	return NodeExecutionResult{OutputSHA256: record.OutputSHA256, LLMCallID: record.CallID}, nil
}

func (s *LiveSession) attemptSchemaRepair(ctx context.Context, nodeID string, policy NodePolicy, text string, record CallRecord, cause error) (string, CallRecord, any, error) {
	if writeErr := s.Store.WriteBytes(fmt.Sprintf("outputs/%s.decode-error.txt", nodeID), []byte(cause.Error()+"\n"), 0o600); writeErr != nil {
		return "", CallRecord{}, nil, errors.Join(fmt.Errorf("strict validation failed for %s: %w", nodeID, cause), writeErr)
	}
	if writeErr := s.Store.WriteBytes(fmt.Sprintf("outputs/%s.raw.txt", nodeID), []byte(text), 0o600); writeErr != nil {
		return "", CallRecord{}, nil, errors.Join(fmt.Errorf("strict validation failed for %s: %w", nodeID, cause), writeErr)
	}
	if s.schemaRepairs >= DefaultMaxSchemaRepairs {
		return "", CallRecord{}, nil, fmt.Errorf("strict validation failed for %s: %w", nodeID, cause)
	}
	repairPrompt, repairErr := BuildSchemaRepairPrompt(SchemaRepairRequest{
		NodeID:       nodeID,
		Role:         policy.Role,
		DecodeError:  cause.Error(),
		OriginalText: text,
		AllowedFiles: policy.AllowedFiles,
	})
	if repairErr != nil {
		return "", CallRecord{}, nil, errors.Join(fmt.Errorf("strict validation failed for %s: %w", nodeID, cause), repairErr)
	}
	s.schemaRepairs++
	repairedText, repairedRecord, repairCallErr := s.Model.Complete(ctx, ModelRequest{
		NodeID: nodeID,
		Role:   string(policy.Role) + "-schema-repair",
		Prompt: repairPrompt,
	})
	if repairCallErr != nil {
		return "", CallRecord{}, nil, errors.Join(fmt.Errorf("strict validation failed for %s: %w", nodeID, cause), repairCallErr)
	}
	if appendErr := s.Store.AppendJSONL("llm_calls.jsonl", record); appendErr != nil {
		return "", CallRecord{}, nil, appendErr
	}
	decoded, err := DecodeRoleOutput(policy.Role, nodeID, []byte(repairedText))
	if err != nil {
		_ = s.Store.WriteBytes(fmt.Sprintf("outputs/%s.repair-raw.txt", nodeID), []byte(repairedText), 0o600)
		return "", CallRecord{}, nil, fmt.Errorf("schema-repair decode failed for %s: %w", nodeID, err)
	}
	return repairedText, repairedRecord, decoded, nil
}

func (s *LiveSession) buildPrompt(nodeID string, policy NodePolicy) (string, error) {
	contents := map[string]string{}
	if policy.Role == KindCoder || policy.Role == KindFixer || policy.Role == KindPlanner {
		for _, rel := range policy.AllowedFiles {
			path := filepath.Join(s.WorkloadDir, rel)
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			contents[rel] = string(data)
		}
	}
	base, err := BuildPrompt(PromptRequest{
		NodeID:          nodeID,
		Role:            policy.Role,
		Ticket:          s.Ticket.ID,
		AllowedFiles:    policy.AllowedFiles,
		ImmutableFiles:  policy.ImmutableFiles,
		SharedContextID: s.Ticket.SharedContext,
		PrivateContext:  policy.PrivateContext,
		FileContents:    contents,
	})
	if err != nil {
		return "", err
	}
	if policy.Role != KindReviewer && policy.Role != KindTester && policy.Role != KindFinalizer && policy.Role != KindFixer {
		return base, nil
	}
	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\nmachine_evidence:\n")
	if s.Worktree != nil {
		fmt.Fprintf(&b, "worktree=%s base_commit=%s final_tree=%s\n", s.Worktree.WorkDir, s.Worktree.BaseCommit, s.Worktree.FinalTreeHash)
		if diff, err := s.Worktree.Diff(context.Background()); err == nil {
			b.WriteString("git_diff:\n")
			b.WriteString(truncateEvidence(diff, 8000))
			b.WriteByte('\n')
		}
	}
	for _, test := range s.Tests {
		fmt.Fprintf(&b, "test name=%s exit=%d cmd=%v\n", test.Name, test.ExitCode, test.Command)
	}
	for _, cmd := range s.Commands {
		if cmd.ExitCode != 0 || cmd.TimedOut {
			fmt.Fprintf(&b, "failed_command=%v exit=%d stderr=%s\n", cmd.Command, cmd.ExitCode, truncateEvidence(cmd.StderrExcerpt, 1000))
		}
	}
	return b.String(), nil
}

func (s *LiveSession) executeIntegrate(ctx context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
	if len(s.PatchRecords) < 3 {
		return NodeExecutionResult{}, fmt.Errorf("integrate requires at least 3 coder patches, got %d", len(s.PatchRecords))
	}
	plan, err := BuildIntegrationPlan(req.RunID, s.PatchRecords)
	if err != nil {
		return NodeExecutionResult{}, err
	}
	if conflicts := DetectHunkConflicts(s.PatchTexts); len(conflicts) > 0 {
		return NodeExecutionResult{}, fmt.Errorf("hunk conflicts: %s", strings.Join(conflicts, "; "))
	}
	root := s.WorktreeRoot
	if root == "" {
		root = filepath.Join(s.Store.Dir, "workspace")
	}
	wt, err := CreateWorktree(ctx, s.WorkloadDir, root, req.RunID, s.GitPath)
	if err != nil {
		return NodeExecutionResult{}, err
	}
	s.Worktree = &wt
	evidence := IntegrateEvidence{
		SchemaVersion: SchemaVersion,
		NodeID:        "integrate",
		Status:        "failed",
		BaseCommit:    wt.BaseCommit,
		BaseTreeHash:  wt.BaseTreeHash,
	}
	for _, rec := range plan.OrderedPatches {
		text := s.PatchTexts[rec.NodeID]
		if text == "" {
			return NodeExecutionResult{}, fmt.Errorf("missing patch text for %s", rec.NodeID)
		}
		applyRes, err := wt.ApplyValidatedPatch(ctx, rec.NodeID, text, rec.ChangedFiles)
		evidence.Applies = append(evidence.Applies, applyRes)
		s.Applies = append(s.Applies, applyRes)
		if err != nil {
			evidence.Conflicts = append(evidence.Conflicts, err.Error())
			if writeErr := s.Store.WriteJSON("outputs/integrate.json", evidence); writeErr != nil {
				return NodeExecutionResult{}, errors.Join(err, writeErr)
			}
			return NodeExecutionResult{}, err
		}
	}
	diff, err := wt.Diff(ctx)
	if err != nil {
		return NodeExecutionResult{}, err
	}
	sum := sha256.Sum256([]byte(diff))
	evidence.FinalDiffSHA = hex.EncodeToString(sum[:])
	evidence.FinalTreeHash = wt.FinalTreeHash
	evidence.Status = "merged"
	if err := s.Store.WriteJSON("outputs/integrate.json", evidence); err != nil {
		return NodeExecutionResult{}, err
	}
	if err := s.Store.WriteBytes("integrate/final.diff", []byte(diff), 0o600); err != nil {
		return NodeExecutionResult{}, err
	}
	if s.WorkerCommand != "" {
		if err := s.runWorkerOnce(ctx, req.RunID); err != nil {
			return NodeExecutionResult{}, err
		}
	}
	payload, err := json.Marshal(evidence)
	if err != nil {
		return NodeExecutionResult{}, err
	}
	h := sha256.Sum256(payload)
	return NodeExecutionResult{OutputSHA256: hex.EncodeToString(h[:])}, nil
}

func (s *LiveSession) runWorkerOnce(ctx context.Context, runID string) error {
	rt := s.ProcessRT
	if rt == nil {
		rt = NewDefaultProcessRuntime()
	}
	workDir := ""
	if s.Worktree != nil {
		workDir = s.Worktree.WorkDir
	}
	result, err := rt.StartPrepared(ctx, ProcessConfig{
		RunID:  runID,
		NodeID: "integrate-worker",
		Worker: WorkerSpec{
			Command: s.WorkerCommand,
			Args:    s.WorkerArgs,
			Dir:     workDir,
		},
		Limits: s.Limits,
	})
	if err != nil {
		return err
	}
	s.Processes = append(s.Processes, result)
	return s.Store.WriteJSON("processes/integrate-worker.json", result)
}

func (s *LiveSession) executeTester(ctx context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
	if s.Worktree == nil {
		return NodeExecutionResult{}, fmt.Errorf("tester requires integrated worktree")
	}
	changed, err := s.Worktree.ChangedFiles(ctx)
	if err != nil {
		return NodeExecutionResult{}, err
	}
	report, err := RunMachineTester(ctx, TesterConfig{
		WorkDir:        s.Worktree.WorkDir,
		ChangedGoFiles: goFilesOnly(changed),
		Timeout:        s.TestTimeout,
		SkipRace:       s.SkipRace,
	})
	report.NodeID = req.NodeID
	s.Commands = append(s.Commands, report.Commands...)
	s.Tests = append(s.Tests, report.Tests...)
	if writeErr := s.Store.WriteJSON(fmt.Sprintf("outputs/%s.json", req.NodeID), report); writeErr != nil {
		return NodeExecutionResult{}, writeErr
	}
	if err != nil {
		return NodeExecutionResult{}, err
	}
	// LLM confirmation after machine pass (keeps real-api call attribution for tester).
	if s.Model != nil && (req.NodeID == "tester" || hasPrefix(req.NodeID, "tester-recheck-")) {
		policy := s.Ticket.NodePolicies["tester"]
		if policy.Role == "" {
			policy = NodePolicy{Role: KindTester, ImmutableFiles: acceptanceScriptNames(), PrivateContext: "Confirm machine test evidence."}
		}
		prompt, perr := s.buildPrompt(req.NodeID, policy)
		if perr != nil {
			return NodeExecutionResult{}, perr
		}
		text, record, cerr := s.Model.Complete(ctx, ModelRequest{NodeID: req.NodeID, Role: string(KindTester), Prompt: prompt})
		if cerr != nil {
			return NodeExecutionResult{}, cerr
		}
		review, derr := DecodeReviewOutput(req.NodeID, []byte(text))
		if derr != nil {
			return NodeExecutionResult{}, derr
		}
		if review.Verdict != "pass" {
			return NodeExecutionResult{}, fmt.Errorf("tester model verdict %q contradicts machine pass", review.Verdict)
		}
		if err := s.Store.WriteBytes(fmt.Sprintf("outputs/%s-model.json", req.NodeID), []byte(text), 0o600); err != nil {
			return NodeExecutionResult{}, err
		}
		if err := s.Store.AppendJSONL("llm_calls.jsonl", record); err != nil {
			return NodeExecutionResult{}, err
		}
	}
	payload, mErr := json.Marshal(report)
	if mErr != nil {
		return NodeExecutionResult{}, mErr
	}
	h := sha256.Sum256(payload)
	return NodeExecutionResult{OutputSHA256: hex.EncodeToString(h[:])}, nil
}

func (s *LiveSession) executeFixer(ctx context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
	if s.Ticket.NodePolicies[req.NodeID].Role == "" {
		s.Ticket.NodePolicies[req.NodeID] = NodePolicy{
			Role:           KindFixer,
			AllowedFiles:   collectAllowlist(s.Ticket),
			ImmutableFiles: acceptanceScriptNames(),
			PrivateContext: "Repair blocking review findings with a unified diff.",
		}
	}
	res, err := s.executeLLMNode(ctx, req)
	if err != nil {
		return res, err
	}
	// Apply fixer patch onto existing worktree.
	text := s.PatchTexts[req.NodeID]
	rec := s.PatchRecords[len(s.PatchRecords)-1]
	applyRes, err := s.Worktree.ApplyValidatedPatch(ctx, req.NodeID, text, rec.ChangedFiles)
	s.Applies = append(s.Applies, applyRes)
	if err != nil {
		return NodeExecutionResult{}, err
	}
	return res, nil
}

func collectAllowlist(ticket Ticket) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, policy := range ticket.NodePolicies {
		for _, p := range policy.AllowedFiles {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}

// LiveRunConfig configures an Open World live codebase-dag execution.
type LiveRunConfig struct {
	RunnerConfig
	OutDir        string
	Ticket        string
	MinPhysical   int
	MinNonblank   int
	GitPath       string
	APIKey        string
	BaseURL       string
	JournalPath   string
	RequireKey    bool
	WorkerCommand string
	WorkerArgs    []string
	SkipRace      bool
	AllowDirty    bool
	TestTimeout   time.Duration
	Cleanup       bool
	// ProcessRT optional injected process/cgroup runtime (tests and non-default hosts).
	ProcessRT ProcessRuntime
	Limits    ResourceLimits
}

type LiveRunResult struct {
	Summary           Summary         `json:"summary"`
	Evidence          EvidenceSummary `json:"evidence"`
	Calls             []CallRecord    `json:"calls"`
	Dir               string          `json:"dir"`
	AllRequiredPassed bool            `json:"all_required_passed"`
}

func RunLive(ctx context.Context, cfg LiveRunConfig) (LiveRunResult, error) {
	if cfg.OutDir == "" {
		return LiveRunResult{}, fmt.Errorf("out dir is required")
	}
	if cfg.Ticket == "" {
		cfg.Ticket = "review-remediation"
	}
	if cfg.Ticket != "review-remediation" {
		return LiveRunResult{}, fmt.Errorf("unsupported ticket %q", cfg.Ticket)
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return LiveRunResult{}, fmt.Errorf("DEEPSEEK_API_KEY is required for live codebase-dag")
	}
	prevKey := os.Getenv("DEEPSEEK_API_KEY")
	if err := os.Setenv("DEEPSEEK_API_KEY", cfg.APIKey); err != nil {
		return LiveRunResult{}, err
	}
	defer func() { _ = os.Setenv("DEEPSEEK_API_KEY", prevKey) }()
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("DEEPSEEK_BASE_URL")
	}
	if cfg.Model == "" {
		cfg.Model = RequiredDeepSeekModel
	}
	if cfg.Provider == "" {
		cfg.Provider = RequiredDeepSeekProvider
	}
	if cfg.MaxCalls == 0 {
		cfg.MaxCalls = DefaultMaxModelCalls
	}
	if cfg.GitPath == "" {
		cfg.GitPath = "git"
	}
	if cfg.TestTimeout == 0 {
		cfg.TestTimeout = 10 * time.Minute
	}
	if !cfg.Cleanup {
		cfg.Cleanup = true
	}

	store, err := NewRunStore(cfg.OutDir, cfg.RunID)
	if err != nil {
		return LiveRunResult{}, err
	}
	journalPath := cfg.JournalPath
	if journalPath == "" {
		journalPath = filepath.Join(store.Dir, "process_journal.jsonl")
	}
	journal, err := NewEvidenceJournal(journalPath)
	if err != nil {
		return LiveRunResult{}, err
	}
	defer func() { _ = journal.Close() }()

	provider := llm.NewDeepSeekProvider(llm.DeepSeekConfig{
		APIKey:    cfg.APIKey,
		BaseURL:   cfg.BaseURL,
		Model:     cfg.Model,
		MaxTokens: 8192,
		Timeout:   180 * time.Second,
	})
	model, err := NewStrictModel(provider, StrictModelOptions{RequiredModel: cfg.Model, MaxCalls: cfg.MaxCalls})
	if err != nil {
		return LiveRunResult{}, err
	}
	if err := model.Validate(ctx); err != nil {
		return LiveRunResult{}, fmt.Errorf("deepseek model gate: %w", err)
	}

	preflight := LocalPreflight{
		ManifestOptions: ManifestOptions{MinPhysical: cfg.MinPhysical, MinNonblank: cfg.MinNonblank, GitPath: cfg.GitPath},
		RequireAPIKey:   true,
		RequireClean:    !cfg.AllowDirty,
		WorkerCommand:   cfg.WorkerCommand,
	}
	ticket := ReviewRemediationTicket()
	session := newLiveSession(store, model, ticket, cfg.WorkloadDir)
	session.GitPath = cfg.GitPath
	session.WorkerCommand = cfg.WorkerCommand
	session.WorkerArgs = cfg.WorkerArgs
	session.SkipRace = cfg.SkipRace
	session.AllowDirty = cfg.AllowDirty
	session.TestTimeout = cfg.TestTimeout
	session.WorktreeRoot = filepath.Join(store.Dir, "workspace")
	session.MinPhysical = cfg.MinPhysical
	session.MinNonblank = cfg.MinNonblank
	session.ProcessRT = cfg.ProcessRT
	if cfg.Limits != (ResourceLimits{}) {
		session.Limits = cfg.Limits
	}
	executor := LiveNodeExecutor{Session: session}

	summary, runErr := runWithFixerLoop(ctx, cfg.RunnerConfig, RunnerDeps{
		Preflight:    preflight,
		NodeExecutor: executor,
		Journal:      journal,
		Clock:        time.Now,
	}, session)

	var cleanupErr error
	if cfg.Cleanup && session.Worktree != nil {
		cleanupErr = session.Worktree.Cleanup(ctx)
	}

	calls := model.Records()
	writeErr := errors.Join(
		store.WriteJSON("llm_calls.json", calls),
		store.WriteJSON("runtime_summary.json", summary),
		store.WriteJSON("preflight.json", summary.Preflight),
		store.WriteJSON("patches.json", session.PatchRecords),
		store.WriteJSON("tests.json", session.Tests),
	)
	evidence, evErr := BuildEvidenceSummary(store, summary, session, calls)
	if evErr == nil {
		evidence.AllRequiredPassed = summary.AllRequiredPassed && runErr == nil && writeErr == nil
	}
	result := LiveRunResult{Summary: summary, Evidence: evidence, Calls: calls, Dir: store.Dir}
	if runErr != nil {
		return result, errors.Join(runErr, writeErr, cleanupErr, evErr)
	}
	if writeErr != nil {
		return result, writeErr
	}
	if evErr != nil {
		return result, evErr
	}
	if len(calls) < DefaultRequiredMinCalls {
		return result, fmt.Errorf("insufficient successful real calls: got %d want >= %d", len(calls), DefaultRequiredMinCalls)
	}
	for _, call := range calls {
		if call.Provider != RequiredDeepSeekProvider || call.ActualModel != RequiredDeepSeekModel || call.Fallback || call.EvidenceMode != "real-api" {
			return result, fmt.Errorf("non-real DeepSeek call recorded: %#v", call)
		}
	}
	// First hash pass (without summary), embed into evidence, write summary once, then finalize index.
	preArts, preErr := store.HashArtifacts()
	if preErr != nil {
		return result, preErr
	}
	evidence.Artifacts = preArts
	evidence.AllRequiredPassed = true
	if err := store.WriteJSON("summary.json", evidence); err != nil {
		return result, err
	}
	arts, err := store.FinalizeHashes()
	if err != nil {
		return result, err
	}
	evidence.Artifacts = arts
	if err := ValidateEvidenceSummary(evidence); err != nil {
		return result, fmt.Errorf("self ValidateEvidenceSummary failed: %w", err)
	}
	if err := ValidateRun(store.Dir); err != nil {
		return result, fmt.Errorf("self ValidateRun failed: %w", err)
	}
	result.Evidence = evidence
	result.AllRequiredPassed = true
	result.Summary.AllRequiredPassed = true
	if cleanupErr != nil {
		return result, fmt.Errorf("run passed but cleanup failed: %w", cleanupErr)
	}
	return result, nil
}

func runWithFixerLoop(ctx context.Context, cfg RunnerConfig, deps RunnerDeps, session *LiveSession) (Summary, error) {
	if err := validateRunnerConfig(cfg); err != nil {
		return Summary{}, err
	}
	graph := NewCodebaseGraph()
	if err := graph.Validate(); err != nil {
		return Summary{}, err
	}
	state := NewExecutionState(graph.Nodes())
	preflight, err := deps.Preflight.Check(ctx, cfg)
	if jerr := deps.Journal.Append(EvidenceEvent{RunID: cfg.RunID, Type: EventPreflight, At: deps.Clock(), Fields: preflightGateFields(preflight)}); jerr != nil {
		return Summary{}, jerr
	}
	session.Manifest = preflight.Manifest
	if err != nil {
		_ = state.Transition("preflight", NodeFailed, TransitionEvidence{Reason: err.Error(), At: deps.Clock()})
		return summaryFromState(cfg.RunID, preflight, state, false), err
	}
	if !preflight.Passed {
		_ = state.Transition("preflight", NodeFailed, TransitionEvidence{Reason: "preflight gates failed", At: deps.Clock()})
		return summaryFromState(cfg.RunID, preflight, state, false), fmt.Errorf("preflight gates failed")
	}
	for _, st := range []NodeStatus{NodeReady, NodeRunning, NodeSucceeded} {
		if err := state.Transition("preflight", st, TransitionEvidence{Reason: "preflight complete", At: deps.Clock()}); err != nil {
			return Summary{}, err
		}
	}

	execute := func(nodeID string, depsNodes []string) error {
		if jerr := deps.Journal.AppendNode(cfg.RunID, nodeID, EventNodeStart, "dependencies complete", deps.Clock()); jerr != nil {
			return jerr
		}
		if err := state.EnsureNode(nodeID); err != nil {
			return err
		}
		if err := state.Transition(nodeID, NodeReady, TransitionEvidence{Reason: "dependencies complete", At: deps.Clock()}); err != nil {
			return err
		}
		if err := state.Transition(nodeID, NodeRunning, TransitionEvidence{Reason: "node dispatched", At: deps.Clock()}); err != nil {
			return err
		}
		result, err := deps.NodeExecutor.ExecuteNode(ctx, NodeExecutionRequest{RunID: cfg.RunID, NodeID: nodeID, Dependencies: depsNodes})
		if err != nil {
			_ = state.Transition(nodeID, NodeFailed, TransitionEvidence{Reason: err.Error(), At: deps.Clock()})
			_ = deps.Journal.AppendNode(cfg.RunID, nodeID, EventNodeEnd, err.Error(), deps.Clock())
			return err
		}
		if err := state.Transition(nodeID, NodeSucceeded, TransitionEvidence{
			Reason: "node completed", OutputSHA256: result.OutputSHA256, LLMCallID: result.LLMCallID, At: deps.Clock(),
		}); err != nil {
			return err
		}
		if jerr := deps.Journal.AppendNode(cfg.RunID, nodeID, EventNodeEnd, "node completed", deps.Clock()); jerr != nil {
			return jerr
		}
		return nil
	}

	// Static DAG until reviewer.
	for {
		ready := graph.Ready(state.Completed())
		if len(ready) == 0 {
			break
		}
		progress := false
		for _, nodeID := range ready {
			if nodeID == "preflight" {
				continue
			}
			if nodeID == "finalizer" {
				// delay finalizer until review loop done
				continue
			}
			if err := execute(nodeID, graph.Dependencies(nodeID)); err != nil {
				return summaryFromState(cfg.RunID, preflight, state, false), err
			}
			progress = true
			if nodeID == "reviewer" {
				goto reviewLoop
			}
		}
		if !progress {
			break
		}
	}

reviewLoop:
	loops := graph.FixerLoops()
	maxIter := 2
	if len(loops) > 0 && loops[0].MaxIterations > 0 {
		maxIter = loops[0].MaxIterations
	}
	for iter := 1; session.ReviewVerdict == "fix"; iter++ {
		if iter > maxIter {
			return summaryFromState(cfg.RunID, preflight, state, false), fmt.Errorf("fixer loop exhausted after %d iterations", maxIter)
		}
		fixerID := fmt.Sprintf("fixer-%d", iter)
		recheckTester := fmt.Sprintf("tester-recheck-%d", iter)
		recheckReviewer := fmt.Sprintf("reviewer-recheck-%d", iter)
		if err := execute(fixerID, []string{"reviewer"}); err != nil {
			return summaryFromState(cfg.RunID, preflight, state, false), err
		}
		if err := execute(recheckTester, []string{fixerID}); err != nil {
			return summaryFromState(cfg.RunID, preflight, state, false), err
		}
		// reviewer recheck via LLM
		session.Ticket.NodePolicies[recheckReviewer] = session.Ticket.NodePolicies["reviewer"]
		session.Ticket.NodePolicies[recheckReviewer] = NodePolicy{
			Role:           KindReviewer,
			ImmutableFiles: acceptanceScriptNames(),
			PrivateContext: session.Ticket.NodePolicies["reviewer"].PrivateContext,
		}
		if err := execute(recheckReviewer, []string{recheckTester}); err != nil {
			return summaryFromState(cfg.RunID, preflight, state, false), err
		}
	}
	if session.ReviewVerdict != "pass" && session.ReviewVerdict != "" {
		return summaryFromState(cfg.RunID, preflight, state, false), fmt.Errorf("review verdict %q blocks finalizer", session.ReviewVerdict)
	}
	if err := execute("finalizer", []string{"reviewer"}); err != nil {
		return summaryFromState(cfg.RunID, preflight, state, false), err
	}
	return summaryFromState(cfg.RunID, preflight, state, true), nil
}
