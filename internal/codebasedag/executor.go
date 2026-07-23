package codebasedag

import (
	"context"
	"fmt"
)

type ModelNodeExecutor struct {
	model  *StrictModel
	ticket Ticket
}

func NewModelNodeExecutor(model *StrictModel, ticket Ticket) ModelNodeExecutor {
	return ModelNodeExecutor{model: model, ticket: ticket}
}

func (e ModelNodeExecutor) ExecuteNode(ctx context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
	if e.model == nil {
		return NodeExecutionResult{}, fmt.Errorf("strict model is required")
	}
	policy, ok := e.ticket.NodePolicies[req.NodeID]
	if !ok {
		return NodeExecutionResult{}, fmt.Errorf("node policy for %q is missing", req.NodeID)
	}
	prompt, err := BuildPrompt(PromptRequest{
		NodeID:          req.NodeID,
		Role:            policy.Role,
		Ticket:          e.ticket.ID,
		AllowedFiles:    policy.AllowedFiles,
		ImmutableFiles:  policy.ImmutableFiles,
		SharedContextID: e.ticket.SharedContext,
		PrivateContext:  policy.PrivateContext,
	})
	if err != nil {
		return NodeExecutionResult{}, err
	}
	_, record, err := e.model.Complete(ctx, ModelRequest{NodeID: req.NodeID, Role: string(policy.Role), Prompt: prompt})
	if err != nil {
		return NodeExecutionResult{}, err
	}
	return NodeExecutionResult{OutputSHA256: record.OutputSHA256, LLMCallID: record.CallID}, nil
}
