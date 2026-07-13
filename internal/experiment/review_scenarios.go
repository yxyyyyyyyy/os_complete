package experiment

import (
	"context"

	"aort-r/internal/review"
)

func RunResourceIsolation(cfg review.ResourceIsolationConfig) (review.ScenarioResult, error) {
	return review.RunResourceIsolation(context.Background(), cfg)
}
