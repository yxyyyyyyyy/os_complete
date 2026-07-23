package experiment

import (
	"context"

	"aort-r/internal/review"
)

func RunResourceIsolation(cfg review.ResourceIsolationConfig) (review.ScenarioResult, error) {
	return review.RunResourceIsolation(context.Background(), cfg)
}

func RunContextSharing(cfg review.ContextSharingConfig) (review.ScenarioResult, error) {
	return review.RunContextSharing(context.Background(), cfg)
}

func RunContextSharingMatrix(cfg review.ContextMatrixConfig) (review.ScenarioResult, error) {
	return review.RunContextSharingMatrix(context.Background(), cfg)
}

func RunAgentDemo(cfg review.AgentDemoConfig) (review.AgentDemoResult, error) {
	return review.RunAgentDemo(context.Background(), cfg)
}

func WriteReviewFinal(cfg review.ReviewFinalConfig) (review.ReviewEvidenceIndex, error) {
	return review.WriteReviewFinal(cfg)
}
