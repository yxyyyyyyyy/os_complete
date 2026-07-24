package codebasedag

import (
	"fmt"
	"strings"

	"aort-r/internal/cvm"
)

// LiveCVM holds page references published for coder nodes.
type LiveCVM struct {
	Store   *cvm.Store
	PageIDs []string
	Refs    []ContextPageRef
}

func (s *LiveSession) ensureCVM() {
	if s.CVM != nil && s.CVM.Store != nil {
		return
	}
	s.CVM = &LiveCVM{Store: cvm.NewStore(nil)}
}

// PublishPlannerPages stores shared planner context as CVM pages and records IDs.
func (s *LiveSession) PublishPlannerPages(shared string, indexSummary string) ([]string, error) {
	s.ensureCVM()
	pages := []struct {
		kind    cvm.PageKind
		content string
	}{
		{cvm.KindTask, shared},
		{cvm.KindProject, indexSummary},
		{cvm.KindSystem, "aort-r codebase-dag shared background"},
	}
	ids := make([]string, 0, len(pages))
	refs := make([]ContextPageRef, 0, len(pages))
	for _, p := range pages {
		page, err := s.CVM.Store.CreatePage(p.kind, p.content)
		if err != nil {
			return nil, err
		}
		ids = append(ids, page.PageID)
		refs = append(refs, ContextPageRef{PageID: page.PageID, Kind: string(p.kind), Bytes: page.Bytes})
		_ = s.CVM.Store.MountPage("planner", page.PageID)
	}
	s.CVM.PageIDs = ids
	s.CVM.Refs = refs
	s.PageIDs = ids
	return ids, nil
}

func (s *LiveSession) mountPagesForCoder(nodeID string) error {
	s.ensureCVM()
	for _, id := range s.CVM.PageIDs {
		if err := s.CVM.Store.MountPage(nodeID, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *LiveSession) communicationComparison() *CommunicationComparison {
	logical := 0
	for _, ref := range s.CVM.Refs {
		logical += ref.Bytes
	}
	if logical == 0 {
		logical = len(s.Ticket.SharedContext)
	}
	return BuildCommunicationComparison(logical, s.CVM.PageIDs)
}

func (s *LiveSession) cvmMetrics() *CVMMetrics {
	s.ensureCVM()
	stats := s.CVM.Store.Stats()
	m := CVMMetricsFromStats(stats)
	return &m
}

func (s *LiveSession) coderPromptExtra(nodeID string) string {
	if s.CVM == nil || len(s.CVM.PageIDs) == 0 {
		return ""
	}
	_ = s.mountPagesForCoder(nodeID)
	policy := s.Ticket.NodePolicies[nodeID]
	return BuildCoderPagePrompt(s.CVM.PageIDs, policy.PrivateContext)
}

func summarizeAllowlists(ticket Ticket) string {
	var b strings.Builder
	b.WriteString("allowlisted production surfaces:\n")
	for _, node := range []string{"resource-coder", "context-coder", "evidence-coder"} {
		policy := ticket.NodePolicies[node]
		fmt.Fprintf(&b, "%s:\n", node)
		for _, f := range policy.AllowedFiles {
			fmt.Fprintf(&b, "  - %s\n", f)
		}
	}
	return b.String()
}
