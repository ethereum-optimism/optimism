package sysgo

import (
	"strings"
	"testing"
)

func TestNormalizePeerID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase without 0x",
			input:    "abc123def456",
			expected: "abc123def456",
		},
		{
			name:     "uppercase without 0x",
			input:    "ABC123DEF456",
			expected: "abc123def456",
		},
		{
			name:     "lowercase with 0x",
			input:    "0xabc123def456",
			expected: "abc123def456",
		},
		{
			name:     "uppercase with 0x",
			input:    "0xABC123DEF456",
			expected: "abc123def456",
		},
		{
			name:     "mixed case with 0x",
			input:    "0xAbC123DeF456",
			expected: "abc123def456",
		},
		{
			name:     "mixed case without 0x",
			input:    "AbC123DeF456",
			expected: "abc123def456",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only 0x",
			input:    "0x",
			expected: "",
		},
		{
			name:     "0x in middle (should not remove)",
			input:    "abc0x123",
			expected: "abc0x123",
		},
		{
			name:     "typical enode ID format",
			input:    "a448f24c6d18e575453db13171562b71999873db5b286df957af199ec94617f7",
			expected: "a448f24c6d18e575453db13171562b71999873db5b286df957af199ec94617f7",
		},
		{
			name:     "enode ID with 0x prefix",
			input:    "0xA448F24C6D18E575453DB13171562B71999873DB5B286DF957AF199EC94617F7",
			expected: "a448f24c6d18e575453db13171562b71999873db5b286df957af199ec94617f7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePeerID(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePeerID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNormalizePeerIDConsistency tests that different formats of the same ID normalize to the same value
func TestNormalizePeerIDConsistency(t *testing.T) {
	baseID := "a448f24c6d18e575453db13171562b71999873db5b286df957af199ec94617f7"
	// Create a mixed case variant manually
	mixedCase := "A448F24C6D18E575453DB13171562B71999873DB5B286DF957AF199EC94617F7"
	variants := []string{
		baseID,
		"0x" + baseID,
		strings.ToUpper(baseID),
		"0x" + strings.ToUpper(baseID),
		mixedCase, // Mixed case
		"0x" + mixedCase,
	}

	normalized := make(map[string]string)
	for _, variant := range variants {
		norm := normalizePeerID(variant)
		normalized[variant] = norm
	}

	// All variants should normalize to the same value
	expected := strings.ToLower(baseID)
	for variant, norm := range normalized {
		if norm != expected {
			t.Errorf("normalizePeerID(%q) = %q, expected all variants to normalize to %q", variant, norm, expected)
		}
	}
}

