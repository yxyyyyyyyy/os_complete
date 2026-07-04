package ipc

import "testing"

func TestBlackboardPublishesPageReferencesWithoutCopyingContent(t *testing.T) {
	board := NewBlackboard()

	metric := board.Publish(PublishRequest{
		Topic:     "review.feedback",
		Publisher: "reviewer-1",
		PageID:    "ctx-page-123",
		SizeBytes: 4096,
	})

	if metric.TotalMessages != 1 {
		t.Fatalf("total messages = %d", metric.TotalMessages)
	}
	if metric.AvoidedCopyBytes != 4096 {
		t.Fatalf("avoided copy bytes = %d", metric.AvoidedCopyBytes)
	}

	messages, pollMetric := board.Poll("review.feedback", "fixer-1")
	if len(messages) != 1 {
		t.Fatalf("messages = %#v", messages)
	}
	if messages[0].PageID != "ctx-page-123" {
		t.Fatalf("message = %#v", messages[0])
	}
	if messages[0].Content != "" {
		t.Fatalf("blackboard copied content into IPC message: %#v", messages[0])
	}
	if pollMetric.AvoidedCopyBytes != 4096 {
		t.Fatalf("poll avoided bytes = %d", pollMetric.AvoidedCopyBytes)
	}
}

func TestBlackboardPollIsPerSubscriber(t *testing.T) {
	board := NewBlackboard()
	board.Publish(PublishRequest{Topic: "topic", Publisher: "agent-a", PageID: "p1", SizeBytes: 10})

	first, _ := board.Poll("topic", "agent-b")
	second, _ := board.Poll("topic", "agent-b")
	other, _ := board.Poll("topic", "agent-c")

	if len(first) != 1 {
		t.Fatalf("first poll = %#v", first)
	}
	if len(second) != 0 {
		t.Fatalf("second poll should not replay to same subscriber: %#v", second)
	}
	if len(other) != 1 {
		t.Fatalf("other subscriber should receive existing page ref: %#v", other)
	}
}
