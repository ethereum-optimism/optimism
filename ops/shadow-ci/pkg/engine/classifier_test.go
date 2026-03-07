package engine

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestClassifier(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name     string
		original model.TestResult
		retry    model.TestResult
		expected model.Classification
	}{
		{
			name:     "fail then pass is flake",
			original: model.TestResult{Status: model.StatusFail},
			retry:    model.TestResult{Status: model.StatusPass},
			expected: model.Flake,
		},
		{
			name:     "fail then fail is real",
			original: model.TestResult{Status: model.StatusFail},
			retry:    model.TestResult{Status: model.StatusFail},
			expected: model.RealFailure,
		},
		{
			name:     "error is infrastructure",
			original: model.TestResult{Status: model.StatusError},
			retry:    model.TestResult{Status: model.StatusPass},
			expected: model.Infrastructure,
		},
		{
			name:     "fail then error is infrastructure",
			original: model.TestResult{Status: model.StatusFail},
			retry:    model.TestResult{Status: model.StatusError},
			expected: model.Infrastructure,
		},
		{
			name:     "pass then pass is unclassified",
			original: model.TestResult{Status: model.StatusPass},
			retry:    model.TestResult{Status: model.StatusPass},
			expected: model.Unclassified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.original, tt.retry)
			assert.Equal(t, tt.expected, result)
		})
	}
}
