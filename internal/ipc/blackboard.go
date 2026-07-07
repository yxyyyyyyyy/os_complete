package ipc

import (
	"sync"
	"time"

	"aort-r/internal/evidence"
)

type PublishRequest struct {
	Topic     string `json:"topic"`
	Publisher string `json:"publisher"`
	PageID    string `json:"page_id"`
	SizeBytes int    `json:"size_bytes"`
	IPCMode   string `json:"ipc_mode,omitempty"`
}

type Message struct {
	ID        string `json:"id"`
	Topic     string `json:"topic"`
	Publisher string `json:"publisher"`
	PageID    string `json:"page_id"`
	SizeBytes int    `json:"size_bytes"`
	IPCMode   string `json:"ipc_mode,omitempty"`
	Content   string `json:"content,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

type Metric struct {
	EvidenceMode      evidence.Mode `json:"evidence_mode"`
	Topic             string        `json:"topic"`
	TotalMessages     int           `json:"total_messages"`
	DeliveredMessages int           `json:"delivered_messages"`
	TopicDepth        int           `json:"topic_depth"`
	AvoidedCopyBytes  int           `json:"avoided_copy_bytes"`
	IPCMode           string        `json:"ipc_mode,omitempty"`
}

type Blackboard struct {
	mu          sync.RWMutex
	topics      map[string][]Message
	offsets     map[string]map[string]int
	total       int
	avoidedCopy int
}

func NewBlackboard() *Blackboard {
	return &Blackboard{
		topics:  make(map[string][]Message),
		offsets: make(map[string]map[string]int),
	}
}

func (b *Blackboard) Publish(req PublishRequest) Metric {
	b.mu.Lock()
	defer b.mu.Unlock()
	if req.SizeBytes < 0 {
		req.SizeBytes = 0
	}
	if req.IPCMode == "" {
		req.IPCMode = "page-reference"
	}
	message := Message{
		ID:        time.Now().Format("20060102150405.000000000"),
		Topic:     req.Topic,
		Publisher: req.Publisher,
		PageID:    req.PageID,
		SizeBytes: req.SizeBytes,
		IPCMode:   req.IPCMode,
		CreatedAt: time.Now().UnixMilli(),
	}
	b.topics[req.Topic] = append(b.topics[req.Topic], message)
	b.total++
	b.avoidedCopy += req.SizeBytes
	metric := b.metricLocked(req.Topic, 0, req.SizeBytes)
	metric.IPCMode = req.IPCMode
	return metric
}

func (b *Blackboard) Poll(topic, subscriber string) ([]Message, Metric) {
	b.mu.Lock()
	defer b.mu.Unlock()
	messages := b.topics[topic]
	if b.offsets[topic] == nil {
		b.offsets[topic] = make(map[string]int)
	}
	start := b.offsets[topic][subscriber]
	if start > len(messages) {
		start = len(messages)
	}
	delivered := append([]Message(nil), messages[start:]...)
	b.offsets[topic][subscriber] = len(messages)
	avoided := 0
	for _, message := range delivered {
		avoided += message.SizeBytes
	}
	metric := b.metricLocked(topic, len(delivered), avoided)
	if len(delivered) > 0 {
		metric.IPCMode = delivered[len(delivered)-1].IPCMode
	}
	return delivered, metric
}

func (b *Blackboard) Metrics() Metric {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return Metric{
		EvidenceMode:     evidence.ModeRealPartial,
		TotalMessages:    b.total,
		TopicDepth:       b.total,
		AvoidedCopyBytes: b.avoidedCopy,
	}
}

func (b *Blackboard) Topics() map[string][]Message {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make(map[string][]Message, len(b.topics))
	for topic, messages := range b.topics {
		out[topic] = append([]Message(nil), messages...)
	}
	return out
}

func (b *Blackboard) metricLocked(topic string, delivered, avoided int) Metric {
	return Metric{
		EvidenceMode:      evidence.ModeRealPartial,
		Topic:             topic,
		TotalMessages:     b.total,
		DeliveredMessages: delivered,
		TopicDepth:        len(b.topics[topic]),
		AvoidedCopyBytes:  avoided,
	}
}
