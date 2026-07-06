package main

import "testing"

func TestRetainMemoryTouchesRequestedBytes(t *testing.T) {
	retained := retainMemory(8192)
	if len(retained) != 8192 {
		t.Fatalf("retained bytes = %d, want 8192", len(retained))
	}
	nonZero := 0
	for _, value := range retained {
		if value != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatal("retained memory was not touched")
	}
}

func TestRetainMemoryHandlesDisabledRetention(t *testing.T) {
	if retained := retainMemory(0); retained != nil {
		t.Fatalf("retainMemory(0) = %v, want nil", retained)
	}
}
