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
