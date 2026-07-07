package experiment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aort-r/internal/evidence"
)

const (
	defaultDeepSeekRealBaseURL = "https://api.deepseek.com"
	defaultDeepSeekRealModel   = "deepseek-v4-flash"
)

type DeepSeekRealSmokeConfig struct {
	OutDir  string
	Enable  bool
	APIKey  string
	BaseURL string
	Model   string
	Client  *http.Client
	Timeout time.Duration
}

type DeepSeekRealSmokeResult struct {
	Experiment       string `json:"experiment"`
	Status           string `json:"status"`
	EvidenceMode     string `json:"evidence_mode"`
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	LLMMock          bool   `json:"llm_mock"`
	RequestSuccess   bool   `json:"request_success"`
	ResponseNonEmpty bool   `json:"response_non_empty"`
	StatusCode       int    `json:"status_code"`
	LatencyMS        int64  `json:"latency_ms"`
	APIKeySource     string `json:"api_key_source"`
	APIKeyPresent    bool   `json:"api_key_present"`
	APIKeyRedacted   bool   `json:"api_key_redacted"`
	CleanupSuccess   bool   `json:"cleanup_success"`
	FailureReason    string `json:"failure_reason,omitempty"`
}

func DeepSeekRealSmokeConfigFromEnv(outDir string) DeepSeekRealSmokeConfig {
	return DeepSeekRealSmokeConfig{
		OutDir:  outDir,
		Enable:  os.Getenv("AORT_ENABLE_REAL_LLM") == "1",
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL: firstNonEmpty(os.Getenv("AORT_LLM_BASE_URL"), os.Getenv("DEEPSEEK_BASE_URL"), defaultDeepSeekRealBaseURL),
		Model:   firstNonEmpty(os.Getenv("AORT_LLM_MODEL"), os.Getenv("DEEPSEEK_MODEL"), defaultDeepSeekRealModel),
	}
}

func RunDeepSeekRealSmoke(cfg DeepSeekRealSmokeConfig) (DeepSeekRealSmokeResult, error) {
	if cfg.OutDir == "" {
		cfg.OutDir = filepath.Join("experiments", "results", "deepseek_real")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultDeepSeekRealBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = defaultDeepSeekRealModel
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	result := DeepSeekRealSmokeResult{
		Experiment:     "deepseek_real_smoke",
		Status:         "skipped",
		EvidenceMode:   string(evidence.ModeMissing),
		Provider:       "deepseek",
		Model:          cfg.Model,
		LLMMock:        false,
		APIKeySource:   "env",
		APIKeyRedacted: true,
		CleanupSuccess: true,
	}
	outPath := filepath.Join(cfg.OutDir, "deepseek_real_smoke.json")
	writeResult := func() {
		_ = WriteJSON(outPath, result)
	}
	if !cfg.Enable {
		result.FailureReason = "AORT_ENABLE_REAL_LLM is not set to 1"
		writeResult()
		return result, nil
	}
	result.APIKeyPresent = cfg.APIKey != ""
	if cfg.APIKey == "" {
		result.Status = "failed"
		result.FailureReason = "DEEPSEEK_API_KEY is not set"
		writeResult()
		return result, fmt.Errorf("DeepSeek real smoke requires DEEPSEEK_API_KEY when AORT_ENABLE_REAL_LLM=1")
	}

	payload := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": "Reply with the single word ok."},
		},
		"stream": false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		result.Status = "failed"
		result.FailureReason = "encode request: " + err.Error()
		writeResult()
		return result, err
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		result.Status = "failed"
		result.FailureReason = "build request: " + err.Error()
		writeResult()
		return result, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	result.LatencyMS = maxDurationMS(time.Since(start).Milliseconds())
	if err != nil {
		result.Status = "failed"
		result.FailureReason = "api_error: " + err.Error()
		writeResult()
		return result, err
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		result.Status = "failed"
		result.FailureReason = "decode response: " + err.Error()
		writeResult()
		return result, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Status = "failed"
		result.FailureReason = fmt.Sprintf("deepseek status %d", resp.StatusCode)
		writeResult()
		return result, fmt.Errorf("%s", result.FailureReason)
	}
	result.RequestSuccess = true
	result.ResponseNonEmpty = len(decoded.Choices) > 0 && strings.TrimSpace(decoded.Choices[0].Message.Content) != ""
	if !result.ResponseNonEmpty {
		result.Status = "failed"
		result.FailureReason = "deepseek returned empty response"
		writeResult()
		return result, fmt.Errorf("%s", result.FailureReason)
	}
	result.Status = "passed"
	result.EvidenceMode = string(evidence.ModeRealAPI)
	writeResult()
	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func maxDurationMS(value int64) int64 {
	if value < 1 {
		return 1
	}
	return value
}
