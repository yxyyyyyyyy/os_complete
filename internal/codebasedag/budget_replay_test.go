package codebasedag

import "testing"

func TestCallBudgetRequiresSevenSuccessfulRoles(t *testing.T) {
	b := NewCallBudget()
	roles := []string{"planner", "resource-coder", "context-coder", "evidence-coder", "tester", "reviewer", "finalizer"}
	for _, role := range roles {
		res, err := b.Reserve(role, "normal")
		if err != nil {
			t.Fatalf("reserve %s: %v", role, err)
		}
		res.Commit(true)
	}
	if err := b.ValidateFinal(); err != nil {
		t.Fatal(err)
	}
	snap := b.Snapshot()
	if !snap.Satisfied || snap.Successes != 7 || snap.Attempts != 7 {
		t.Fatalf("snap=%#v", snap)
	}
}

func TestCallBudgetRejectsEighthNormalBeyondMaxWithFixers(t *testing.T) {
	b := NewCallBudgetWithLimits(8, 7, 1, 0)
	for i := 0; i < 8; i++ {
		role := "planner"
		if i < 7 {
			role = []string{"planner", "resource-coder", "context-coder", "evidence-coder", "tester", "reviewer", "finalizer"}[i]
		}
		res, err := b.Reserve(role, "normal")
		if err != nil {
			t.Fatalf("reserve %d: %v", i, err)
		}
		res.Commit(true)
	}
	if _, err := b.Reserve("fixer", "fixer"); err == nil {
		t.Fatal("expected exhaustion")
	}
}

func TestCallBudgetFixerAndSchemaRepairCaps(t *testing.T) {
	b := NewCallBudgetWithLimits(15, 7, 2, 3)
	roles := []string{"planner", "resource-coder", "context-coder", "evidence-coder", "tester", "reviewer", "finalizer"}
	for _, role := range roles {
		res, err := b.Reserve(role, "normal")
		if err != nil {
			t.Fatal(err)
		}
		res.Commit(true)
	}
	r1, err := b.Reserve("fixer", "fixer")
	if err != nil {
		t.Fatal(err)
	}
	r1.Commit(true)
	r2, err := b.Reserve("fixer", "fixer")
	if err != nil {
		t.Fatal(err)
	}
	r2.Commit(false)
	if _, err := b.Reserve("fixer", "fixer"); err == nil {
		t.Fatal("third fixer should fail")
	}
	s1, err := b.Reserve("planner", "schema-repair")
	if err != nil {
		t.Fatal(err)
	}
	s1.Commit(true)
	s2, err := b.Reserve("coder", "schema-repair")
	if err != nil {
		t.Fatal(err)
	}
	s2.Commit(true)
	s3, err := b.Reserve("coder2", "schema-repair")
	if err != nil {
		t.Fatal(err)
	}
	s3.Commit(true)
	if _, err := b.Reserve("coder3", "schema-repair"); err == nil {
		t.Fatal("fourth schema-repair should fail")
	}
}

func TestCallBudgetUnknownKindAndDoubleCommit(t *testing.T) {
	b := NewCallBudget()
	if _, err := b.Reserve("planner", "weird"); err == nil {
		t.Fatal("unknown kind should fail")
	}
	res, err := b.Reserve("planner", "normal")
	if err != nil {
		t.Fatal(err)
	}
	res.Commit(true)
	res.Commit(true) // no double count
	snap := b.Snapshot()
	if snap.Successes != 1 || snap.Attempts != 1 {
		t.Fatalf("snap=%#v", snap)
	}
}

func TestCallBudgetValidateFinalMissingRole(t *testing.T) {
	b := NewCallBudget()
	for _, role := range []string{"planner", "resource-coder", "context-coder", "evidence-coder", "tester", "reviewer"} {
		res, err := b.Reserve(role, "normal")
		if err != nil {
			t.Fatal(err)
		}
		res.Commit(true)
	}
	if err := b.ValidateFinal(); err == nil {
		t.Fatal("missing finalizer should fail")
	}
}

func TestReplayGuardFailOnceWithoutDuplicateCall(t *testing.T) {
	g := NewReplayGuard()
	payload := []byte(`{"status":"ok","patch":"diff"}`)
	rec, err := g.Remember("resource-coder", "call-1", payload)
	if err != nil {
		t.Fatal(err)
	}
	if rec.OutputSHA256 == "" {
		t.Fatal("missing hash")
	}
	if err := g.MarkFailOnce("resource-coder"); err != nil {
		t.Fatal(err)
	}
	out, err := g.Replay("resource-coder")
	if err != nil {
		t.Fatal(err)
	}
	if out.CallID != "call-1" || string(out.Payload) != string(payload) {
		t.Fatalf("out=%#v", out)
	}
	if err := g.VerifyUnchanged("resource-coder", payload); err != nil {
		t.Fatal(err)
	}
	if err := g.VerifyUnchanged("resource-coder", []byte("changed")); err == nil {
		t.Fatal("changed payload should fail")
	}
	snap := g.Snapshot()
	if snap.CheckpointedNodes != 1 || snap.FailOnceNodes != 1 || snap.ReplayCounts["resource-coder"] != 1 {
		t.Fatalf("snap=%#v", snap)
	}
}

func TestReplayGuardRejectsConflictsAndInvalidTransitions(t *testing.T) {
	g := NewReplayGuard()
	if _, err := g.Remember("", "c", []byte("x")); err == nil {
		t.Fatal("empty node")
	}
	if _, err := g.Remember("n", "", []byte("x")); err == nil {
		t.Fatal("empty call")
	}
	if _, err := g.Remember("n", "c", nil); err == nil {
		t.Fatal("empty payload")
	}
	if _, err := g.Remember("n", "c", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Remember("n", "c2", []byte("two")); err == nil {
		t.Fatal("conflict should fail")
	}
	if err := g.MarkFailOnce("missing"); err == nil {
		t.Fatal("fail-once without checkpoint")
	}
	if _, err := g.Replay("n"); err == nil {
		t.Fatal("replay without fail-once")
	}
	if err := g.MarkFailOnce("n"); err != nil {
		t.Fatal(err)
	}
	if err := g.MarkFailOnce("n"); err == nil {
		t.Fatal("second fail-once")
	}
}
