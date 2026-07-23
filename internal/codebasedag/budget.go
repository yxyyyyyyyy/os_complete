package codebasedag

import (
	"fmt"
	"sync"
)

const (
	DefaultMaxCalls         = 10
	DefaultRequiredMinCalls = 7
	DefaultMaxFixerCalls    = 2
	DefaultMaxSchemaRepairs = 3
)

// CallBudget enforces the Open World DeepSeek call envelope:
// at least 7 successful real calls, at most 10 total attempts including fixer/schema-repair.
type CallBudget struct {
	mu            sync.Mutex
	maxCalls      int
	requiredMin   int
	maxFixer      int
	maxRepairs    int
	attempts      int
	successes     int
	fixerUsed     int
	repairsUsed   int
	byRole        map[string]int
	successByRole map[string]int
}

func NewCallBudget() *CallBudget {
	return NewCallBudgetWithLimits(DefaultMaxCalls, DefaultRequiredMinCalls, DefaultMaxFixerCalls, DefaultMaxSchemaRepairs)
}

func NewCallBudgetWithLimits(maxCalls, requiredMin, maxFixer, maxRepairs int) *CallBudget {
	if maxCalls <= 0 {
		maxCalls = DefaultMaxCalls
	}
	if requiredMin <= 0 {
		requiredMin = DefaultRequiredMinCalls
	}
	if maxFixer < 0 {
		maxFixer = DefaultMaxFixerCalls
	}
	if maxRepairs < 0 {
		maxRepairs = DefaultMaxSchemaRepairs
	}
	return &CallBudget{
		maxCalls:      maxCalls,
		requiredMin:   requiredMin,
		maxFixer:      maxFixer,
		maxRepairs:    maxRepairs,
		byRole:        make(map[string]int),
		successByRole: make(map[string]int),
	}
}

type BudgetReservation struct {
	Role   string
	Kind   string // "normal" | "fixer" | "schema-repair"
	Slot   int
	Commit func(success bool)
}

func (b *CallBudget) Reserve(role, kind string) (BudgetReservation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.attempts >= b.maxCalls {
		return BudgetReservation{}, fmt.Errorf("call budget exhausted: attempts=%d max=%d", b.attempts, b.maxCalls)
	}
	switch kind {
	case "", "normal":
		kind = "normal"
	case "fixer":
		if b.fixerUsed >= b.maxFixer {
			return BudgetReservation{}, fmt.Errorf("fixer call budget exhausted: used=%d max=%d", b.fixerUsed, b.maxFixer)
		}
	case "schema-repair":
		if b.repairsUsed >= b.maxRepairs {
			return BudgetReservation{}, fmt.Errorf("schema-repair call budget exhausted: used=%d max=%d", b.repairsUsed, b.maxRepairs)
		}
	default:
		return BudgetReservation{}, fmt.Errorf("unknown budget kind %q", kind)
	}
	b.attempts++
	slot := b.attempts
	b.byRole[role]++
	switch kind {
	case "fixer":
		b.fixerUsed++
	case "schema-repair":
		b.repairsUsed++
	}
	var committed bool
	return BudgetReservation{
		Role: role,
		Kind: kind,
		Slot: slot,
		Commit: func(success bool) {
			b.mu.Lock()
			defer b.mu.Unlock()
			if committed {
				return
			}
			committed = true
			if success {
				b.successes++
				b.successByRole[role]++
			}
		},
	}, nil
}

type CallBudgetSnapshot struct {
	Attempts      int            `json:"attempts"`
	Successes     int            `json:"successes"`
	RequiredMin   int            `json:"required_min"`
	MaxCalls      int            `json:"max_calls"`
	FixerUsed     int            `json:"fixer_used"`
	RepairsUsed   int            `json:"repairs_used"`
	ByRole        map[string]int `json:"by_role"`
	SuccessByRole map[string]int `json:"success_by_role"`
	Satisfied     bool           `json:"satisfied"`
}

func (b *CallBudget) Snapshot() CallBudgetSnapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	byRole := make(map[string]int, len(b.byRole))
	for k, v := range b.byRole {
		byRole[k] = v
	}
	successByRole := make(map[string]int, len(b.successByRole))
	for k, v := range b.successByRole {
		successByRole[k] = v
	}
	return CallBudgetSnapshot{
		Attempts:      b.attempts,
		Successes:     b.successes,
		RequiredMin:   b.requiredMin,
		MaxCalls:      b.maxCalls,
		FixerUsed:     b.fixerUsed,
		RepairsUsed:   b.repairsUsed,
		ByRole:        byRole,
		SuccessByRole: successByRole,
		Satisfied:     b.successes >= b.requiredMin && b.attempts <= b.maxCalls,
	}
}

func (b *CallBudget) ValidateFinal() error {
	snap := b.Snapshot()
	if snap.Successes < snap.RequiredMin {
		return fmt.Errorf("insufficient successful real calls: got %d want >= %d", snap.Successes, snap.RequiredMin)
	}
	if snap.Attempts > snap.MaxCalls {
		return fmt.Errorf("call attempts %d exceed max %d", snap.Attempts, snap.MaxCalls)
	}
	requiredRoles := []string{"planner", "resource-coder", "context-coder", "evidence-coder", "tester", "reviewer", "finalizer"}
	for _, role := range requiredRoles {
		if snap.SuccessByRole[role] < 1 {
			return fmt.Errorf("missing successful call for required role %q", role)
		}
	}
	return nil
}
