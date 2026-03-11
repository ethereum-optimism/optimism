package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// LLMAdvisor provides LLM-based placement override suggestions.
type LLMAdvisor struct {
	config  model.LLMAdvisorConfig
	apiKey  string
	emitter *events.Emitter
	client  *http.Client
}

// DefaultLLMAdvisorConfig returns sensible defaults (disabled).
func DefaultLLMAdvisorConfig() model.LLMAdvisorConfig {
	return model.LLMAdvisorConfig{
		Enabled:    false,
		Model:      "claude-sonnet-4-6",
		MaxTokens:  4096,
		TimeoutSec: 30,
	}
}

// NewLLMAdvisor creates a new LLMAdvisor. If API key is missing and Enabled is true,
// logs a warning and disables.
func NewLLMAdvisor(config model.LLMAdvisorConfig, emitter *events.Emitter) *LLMAdvisor {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")

	if config.Enabled && apiKey == "" {
		log.Printf("LLM advisor enabled but ANTHROPIC_API_KEY not set — disabling")
		config.Enabled = false
	}

	timeout := time.Duration(config.TimeoutSec) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &LLMAdvisor{
		config:  config,
		apiKey:  apiKey,
		emitter: emitter,
		client:  &http.Client{Timeout: timeout},
	}
}

// Advise takes the diff, current placements, and stats, and returns placement overrides.
// Returns empty overrides when disabled. Errors are non-fatal.
func (la *LLMAdvisor) Advise(diff string, currentPlacements []model.TestPlacement, stats map[string]*TestStats) ([]PlacementOverride, error) {
	if !la.config.Enabled {
		return nil, nil
	}

	prompt := la.buildPrompt(diff, currentPlacements, stats)

	resp, err := la.callAPI(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}

	overrides, err := la.parseResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if la.emitter != nil && len(overrides) > 0 {
		la.emitter.Emit(model.EventPlacementChanged, map[string]any{
			"source":    "llm_advisor",
			"overrides": len(overrides),
		})
	}

	return overrides, nil
}

func (la *LLMAdvisor) buildPrompt(diff string, placements []model.TestPlacement, stats map[string]*TestStats) string {
	var sb strings.Builder

	sb.WriteString("You are a CI placement advisor. Analyze this diff and test placements.\n\n")
	sb.WriteString("## Diff\n```\n")

	// Truncate diff to ~8k chars if needed.
	if len(diff) > 8000 {
		sb.WriteString(diff[:8000])
		sb.WriteString("\n... (truncated)\n")
	} else {
		sb.WriteString(diff)
	}
	sb.WriteString("```\n\n")

	sb.WriteString("## Current Placements\n")
	for _, p := range placements {
		sb.WriteString(fmt.Sprintf("- %s: stage=%s, confidence=%.2f, reason=%s\n",
			p.TestKey, p.AssignedStage, p.Confidence, p.Reason))
	}

	sb.WriteString("\n## Instructions\n")
	sb.WriteString("Identify tests where the statistical placement might be wrong for this specific change. ")
	sb.WriteString("Only override when you have high confidence that the statistical model is missing context. ")
	sb.WriteString("Return a JSON array of overrides:\n")
	sb.WriteString(`[{"test_key": "...", "override_to": "pr|merge_queue|post_merge", "reason": "...", "confidence": 0.0-1.0}]`)
	sb.WriteString("\nReturn [] if no overrides are needed.\n")

	return sb.String()
}

func (la *LLMAdvisor) callAPI(prompt string) (string, error) {
	reqBody := map[string]any{
		"model":      la.config.Model,
		"max_tokens": la.config.MaxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", la.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := la.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode API response: %w", err)
	}

	if len(result.Content) == 0 {
		return "[]", nil
	}

	return result.Content[0].Text, nil
}

func (la *LLMAdvisor) parseResponse(response string) ([]PlacementOverride, error) {
	// Find JSON array in response.
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, nil // no JSON array found — treat as no overrides
	}

	jsonStr := response[start : end+1]
	var overrides []PlacementOverride
	if err := json.Unmarshal([]byte(jsonStr), &overrides); err != nil {
		return nil, fmt.Errorf("failed to parse JSON overrides: %w", err)
	}

	return overrides, nil
}
