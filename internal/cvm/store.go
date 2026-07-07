package cvm

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
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
	ID                  string   `json:"id"`
	PageID              string   `json:"page_id"`
	Kind                PageKind `json:"kind"`
	Content             string   `json:"content"`
	Hash                string   `json:"hash"`
	Bytes               int      `json:"bytes"`
	SizeBytes           int      `json:"size_bytes"`
	TokenCount          int      `json:"token_count"`
	RefCount            int      `json:"ref_count"`
	Pinned              bool     `json:"pinned"`
	Compressed          bool     `json:"compressed"`
	CompressedSizeBytes int      `json:"compressed_size_bytes,omitempty"`
	LastAccessTime      int64    `json:"last_access_time"`
	AccessCount         int      `json:"access_count"`
	OwnerAgents         []string `json:"owner_agents"`
	CreatedAt           int64    `json:"created_at"`
	UpdatedAt           int64    `json:"updated_at"`
	compressedData      []byte
}

type PageTable struct {
	AgentID string   `json:"agent_id"`
	PageIDs []string `json:"page_ids"`
}

type Stats struct {
	EvidenceMode          evidence.Mode `json:"evidence_mode"`
	TotalPages            int           `json:"total_pages"`
	SharedPages           int           `json:"shared_pages"`
	HotPages              int           `json:"hot_pages"`
	ColdPages             int           `json:"cold_pages"`
	CompressedPages       int           `json:"compressed_pages"`
	EvictedPages          int           `json:"evicted_pages"`
	PinnedPages           int           `json:"pinned_pages"`
	RefCountedPages       int           `json:"ref_counted_pages"`
	SavedBytes            int64         `json:"saved_bytes"`
	SavedTokens           int64         `json:"saved_tokens"`
	MemorySavedBytes      int64         `json:"memory_saved_bytes"`
	CompressionSavedBytes int64         `json:"compression_saved_bytes"`
	DedupSavedBytes       int64         `json:"dedup_saved_bytes"`
	DedupSavedTokens      int64         `json:"dedup_saved_tokens"`
}

type Store struct {
	mu                    sync.RWMutex
	pages                 map[string]Page
	pageTables            map[string][]string
	savedBytes            int64
	savedTokens           int64
	compressionSavedBytes int64
	evictedPages          int
	sink                  EventSink
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
		ID:             id,
		PageID:         id,
		Kind:           kind,
		Content:        content,
		Hash:           id,
		Bytes:          len([]byte(content)),
		SizeBytes:      len([]byte(content)),
		TokenCount:     tokens,
		RefCount:       1,
		LastAccessTime: time.Now().UnixMilli(),
		AccessCount:    1,
		CreatedAt:      time.Now().UnixMilli(),
		UpdatedAt:      time.Now().UnixMilli(),
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
	page.OwnerAgents = appendUniqueString(page.OwnerAgents, agentID)
	page.AccessCount++
	page.LastAccessTime = time.Now().UnixMilli()
	page.UpdatedAt = page.LastAccessTime
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
	s.mu.Lock()
	defer s.mu.Unlock()
	var out string
	for _, pageID := range s.pageTables[agentID] {
		page, ok := s.pages[pageID]
		if !ok {
			return "", fmt.Errorf("page %q missing from store", pageID)
		}
		content, err := page.materializedContent()
		if err != nil {
			return "", err
		}
		out += content
		page.AccessCount++
		page.LastAccessTime = time.Now().UnixMilli()
		page.UpdatedAt = page.LastAccessTime
		s.pages[pageID] = page
	}
	s.publishLocked("context.materialized", agentID, Page{}, map[string]any{"bytes": len([]byte(out)), "tokens": estimateTokens(out)})
	return out, nil
}

func (s *Store) TouchPage(pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	page, ok := s.pages[pageID]
	if !ok {
		return fmt.Errorf("unknown page %q", pageID)
	}
	page.AccessCount++
	page.LastAccessTime = time.Now().UnixMilli()
	page.UpdatedAt = page.LastAccessTime
	s.pages[pageID] = page
	return nil
}

func (s *Store) PinPage(pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	page, ok := s.pages[pageID]
	if !ok {
		return fmt.Errorf("unknown page %q", pageID)
	}
	page.Pinned = true
	page.UpdatedAt = time.Now().UnixMilli()
	s.pages[pageID] = page
	return nil
}

func (s *Store) ReleasePage(agentID, pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	page, ok := s.pages[pageID]
	if !ok {
		return fmt.Errorf("unknown page %q", pageID)
	}
	if page.RefCount > 0 {
		page.RefCount--
	}
	if agentID != "" {
		s.pageTables[agentID] = removeString(s.pageTables[agentID], pageID)
		page.OwnerAgents = removeString(page.OwnerAgents, agentID)
	}
	page.UpdatedAt = time.Now().UnixMilli()
	s.pages[pageID] = page
	return nil
}

func (s *Store) CompressColdPages(maxAccessCount int) (int, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if maxAccessCount <= 0 {
		maxAccessCount = 1
	}
	compressed := 0
	var saved int64
	for id, page := range s.pages {
		if page.Pinned || page.Compressed || page.AccessCount > maxAccessCount {
			continue
		}
		content := page.Content
		if content == "" {
			continue
		}
		data, err := gzipBytes([]byte(content))
		if err != nil {
			return compressed, saved, err
		}
		if len(data) >= len([]byte(content)) {
			data = []byte(content)
		}
		page.compressedData = data
		page.Compressed = true
		page.CompressedSizeBytes = len(data)
		page.Content = ""
		page.UpdatedAt = time.Now().UnixMilli()
		pageSaved := int64(max(0, page.Bytes-len(data)))
		s.compressionSavedBytes += pageSaved
		saved += pageSaved
		s.pages[id] = page
		compressed++
	}
	return compressed, saved, nil
}

func (s *Store) EvictColdPages(maxBytes int) (int, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.currentBytesLocked()
	if maxBytes < 0 || current <= maxBytes {
		return 0, 0
	}
	candidates := make([]Page, 0, len(s.pages))
	for _, page := range s.pages {
		if page.RefCount == 0 && !page.Pinned {
			candidates = append(candidates, page)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].AccessCount == candidates[j].AccessCount {
			return candidates[i].LastAccessTime < candidates[j].LastAccessTime
		}
		return candidates[i].AccessCount < candidates[j].AccessCount
	})
	evicted := 0
	var freed int64
	for _, page := range candidates {
		if current <= maxBytes {
			break
		}
		delete(s.pages, page.ID)
		for agentID, ids := range s.pageTables {
			s.pageTables[agentID] = removeString(ids, page.ID)
		}
		size := page.residentBytes()
		current -= size
		freed += int64(size)
		evicted++
	}
	s.evictedPages += evicted
	return evicted, freed
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
	stats := Stats{
		EvidenceMode:          evidence.ModeRealPartial,
		TotalPages:            len(s.pages),
		SavedBytes:            s.savedBytes,
		SavedTokens:           s.savedTokens,
		MemorySavedBytes:      s.savedBytes + s.compressionSavedBytes,
		CompressionSavedBytes: s.compressionSavedBytes,
		DedupSavedBytes:       s.savedBytes,
		DedupSavedTokens:      s.savedTokens,
		EvictedPages:          s.evictedPages,
	}
	for _, page := range s.pages {
		if page.RefCount > 1 {
			stats.SharedPages++
		}
		if page.RefCount > 0 {
			stats.RefCountedPages++
		}
		if page.Pinned {
			stats.PinnedPages++
		}
		if page.Compressed {
			stats.CompressedPages++
		}
		if page.AccessCount >= 4 {
			stats.HotPages++
		} else if page.RefCount == 0 && !page.Pinned {
			stats.ColdPages++
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

func (p Page) materializedContent() (string, error) {
	if !p.Compressed {
		return p.Content, nil
	}
	if len(p.compressedData) == p.Bytes {
		return string(p.compressedData), nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(p.compressedData))
	if err != nil {
		return "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (p Page) residentBytes() int {
	if p.Compressed {
		return p.CompressedSizeBytes
	}
	return p.Bytes
}

func (s *Store) currentBytesLocked() int {
	total := 0
	for _, page := range s.pages {
		total += page.residentBytes()
	}
	return total
}

func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func appendUniqueString(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func removeString(values []string, value string) []string {
	out := values[:0]
	for _, existing := range values {
		if existing != value {
			out = append(out, existing)
		}
	}
	return out
}
