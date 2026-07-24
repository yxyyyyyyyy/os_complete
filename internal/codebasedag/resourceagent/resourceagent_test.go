package resourceagent

import "testing"

func TestResourceAgentDeepSeekCorpusPresent(t *testing.T) {
	if CountGeneratedFiles() < 40 {
		t.Fatalf("generated files=%d", CountGeneratedFiles())
	}
	if PhysicalLinesApprox() < 20000 {
		t.Fatalf("physical lines=%d want >=20000", PhysicalLinesApprox())
	}
}
