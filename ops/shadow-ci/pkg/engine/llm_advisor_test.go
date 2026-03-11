package engine

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func TestLLMAdvisor_Disabled(t *testing.T) {
	config := model.LLMAdvisorConfig{Enabled: false}
	advisor := NewLLMAdvisor(config, nil)

	overrides, err := advisor.Advise("diff content", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(overrides) != 0 {
		t.Errorf("expected no overrides when disabled, got %d", len(overrides))
	}
}

func TestLLMAdvisor_ParseResponse(t *testing.T) {
	advisor := &LLMAdvisor{}

	tests := []struct {
		name     string
		response string
		want     int
	}{
		{"valid JSON", `[{"test_key":"pkg/TestA","override_to":"pr","reason":"test","confidence":0.8}]`, 1},
		{"empty array", `[]`, 0},
		{"no JSON", "No overrides needed", 0},
		{"embedded JSON", `Here's my analysis:\n[{"test_key":"a","override_to":"pr","reason":"r","confidence":0.9}]\nDone.`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides, err := advisor.parseResponse(tt.response)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(overrides) != tt.want {
				t.Errorf("got %d overrides, want %d", len(overrides), tt.want)
			}
		})
	}
}

func TestLLMAdvisor_BuildPrompt(t *testing.T) {
	advisor := &LLMAdvisor{config: model.LLMAdvisorConfig{Model: "test-model"}}

	placements := []model.TestPlacement{
		{TestKey: "pkg/TestA", AssignedStage: model.StagePR, Confidence: 0.95, Reason: "retained"},
	}

	prompt := advisor.buildPrompt("diff here", placements, nil)
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
	if len(prompt) < 100 {
		t.Error("prompt suspiciously short")
	}
}

func TestLLMAdvisor_NoAPIKey(t *testing.T) {
	// When enabled but no API key, should auto-disable.
	config := model.LLMAdvisorConfig{Enabled: true}
	advisor := NewLLMAdvisor(config, nil)

	// Should behave as disabled.
	overrides, err := advisor.Advise("diff", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(overrides) != 0 {
		t.Errorf("expected no overrides when API key missing, got %d", len(overrides))
	}
}
