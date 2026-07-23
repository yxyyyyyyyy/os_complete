package review

import "testing"

func TestLiveHooksPresent(t *testing.T) {
	if LiveResourceHook == "" || LiveContextHook == "" || LiveEvidenceHook == "" {
		t.Fatal("live hooks must be non-empty")
	}
}
