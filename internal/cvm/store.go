package cvm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"aort-r/internal/events"
	"aort-r/internal/evidence"
)

type PageKind string

const (
	KindSystem  PageKind = "system"
	KindProject PageKind = "project"
	KindTask    PageKind = "task"
	KindDelta   PageKind = "delta"
	KindSummary PageKind = "summary"
)

type EventSink interface {
	Publish(events.Event)
}

type Page struct {
	ID         string   `json:"id"`
	Kind       PageKind `json:"kind"`
	Content    string   `json:"content"`
	Bytes      int      `json:"bytes"`
	TokenCount int      `json:"token_count"`
	RefCount   int      `json:"ref_count"`
	CreatedAt  int64    `json:"created_at"`
}

type PageTable struct {
	AgentID string   `json:"agent_id"`
	PageIDs []string `json:"page_ids"`
}

type Stats struct {
	EvidenceMode evidence.Mode `json:"evidence_mode"`
	TotalPages   int           `json:"total_pages"`
	SharedPages  int           `json:"shared_pages"`
	SavedBytes   int64         `json:"saved_bytes"`
	SavedTokens  int64         `json:"saved_tokens"`
}

type Store struct {
	mu          sync.RWMutex
	pages       map[string]Page
	pageTables  map[string][]string
	savedBytes  int64
	savedTokens int64
	sink        EventSink
}

func NewStore(sink EventSink) *Store {
	return &Store{
		pages:      make(map[string]Page),
		pageTables: make(map[string][]string),
		sink:       sink,
	}
}

func (s *Store) CreatePage(kind PageKind, content string) (Page, error) {
	if kind == "" {
		return Page{}, fmt.Errorf("page kind is required")
	}
	id := hashContent(content)
	tokens := estimateTokens(content)
	s.mu.Lock()
	defer s.mu.Unlock()
	if page, ok := s.pages[id]; ok {
		page.RefCount++
		s.pages[id] = page
		s.savedBytes += int64(page.Bytes)
		s.savedTokens += int64(page.TokenCount)
		s.publishLocked("context.page.reused", "", page, map[string]any{"saved_bytes": page.Bytes, "saved_tokens": page.TokenCount})
		return page, nil
	}
	page := Page{
		ID:         id,
		Kind:       kind,
		Content:    content,
		Bytes:      len([]byte(content)),
		TokenCount: tokens,
		RefCount:   1,
		CreatedAt:  time.Now().UnixMilli(),
	}
	s.pages[id] = page
	s.publishLocked("context.page.created", "", page, nil)
	return page, nil
}

func (s *Store) MountPage(agentID, pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	page, ok := s.pages[pageID]
	if !ok {
		return fmt.Errorf("unknown page %q", pageID)
	}
	for _, existing := range s.pageTables[agentID] {
		if existing == pageID {
			return nil
		}
	}
	s.pageTables[agentID] = append(s.pageTables[agentID], pageID)
	if page.RefCount > 1 {
		s.savedBytes += int64(page.Bytes)
		s.savedTokens += int64(page.TokenCount)
	}
	page.RefCount++
	s.pages[pageID] = page
	s.publishLocked("context.page.mounted", agentID, page, nil)
	return nil
}

func (s *Store) WriteDelta(agentID, content string) (Page, error) {
	page, err := s.CreatePage(KindDelta, content)
	if err != nil {
		return Page{}, err
	}
	if err := s.MountPage(agentID, page.ID); err != nil {
		return Page{}, err
	}
	return page, nil
}

func (s *Store) Materialize(agentID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out string
	for _, pageID := range s.pageTables[agentID] {
		page, ok := s.pages[pageID]
		if !ok {
			return "", fmt.Errorf("page %q missing from store", pageID)
		}
		out += page.Content
	}
	s.publishLocked("context.materialized", agentID, Page{}, map[string]any{"bytes": len([]byte(out)), "tokens": estimateTokens(out)})
	return out, nil
}

func (s *Store) Page(pageID string) (Page, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	page, ok := s.pages[pageID]
	return page, ok
}

func (s *Store) Pages() []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pages := make([]Page, 0, len(s.pages))
	for _, page := range s.pages {
		pages = append(pages, page)
	}
	return pages
}

func (s *Store) PageTable(agentID string) PageTable {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return PageTable{AgentID: agentID, PageIDs: append([]string(nil), s.pageTables[agentID]...)}
}

func (s *Store) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := Stats{EvidenceMode: evidence.ModeRealPartial, TotalPages: len(s.pages), SavedBytes: s.savedBytes, SavedTokens: s.savedTokens}
	for _, page := range s.pages {
		if page.RefCount > 1 {
			stats.SharedPages++
		}
	}
	return stats
}

func (s *Store) publishLocked(eventType, agentID string, page Page, payload map[string]any) {
	if s.sink == nil {
		return
	}
	if payload == nil {
		payload = map[string]any{}
	}
	if page.ID != "" {
		payload["page_id"] = page.ID
		payload["kind"] = string(page.Kind)
		payload["ref_count"] = page.RefCount
	}
	s.sink.Publish(events.New(eventType, "", agentID, "runtime", payload))
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func estimateTokens(content string) int {
	tokens := len([]rune(content)) / 4
	if tokens == 0 && content != "" {
		return 1
	}
	return tokens
}
