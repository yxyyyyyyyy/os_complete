package demo

import (
	"context"
	"reflect"
	"testing"
)

func TestSoftwareDemoProducesExpectedRoles(t *testing.T) {
	runner := NewSoftwareDemoRunner()
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	roles := result.Roles()
	want := []string{"planner", "coder-a", "coder-b", "tester", "reviewer", "fixer"}
	if !reflect.DeepEqual(roles, want) {
		t.Fatalf("roles = %#v", roles)
	}
}

func TestSoftwareDemoEmitsTaskCompletedLast(t *testing.T) {
	runner := NewSoftwareDemoRunner()
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Events) == 0 {
		t.Fatalf("no events emitted")
	}
	last := result.Events[len(result.Events)-1]
	if last.Type != "task.completed" {
		t.Fatalf("last event = %#v", last)
	}
}
