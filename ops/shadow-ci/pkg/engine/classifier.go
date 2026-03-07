package engine

import "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"

// Classifier determines whether a failure is real, flake, or infrastructure.
type Classifier struct{}

// NewClassifier creates a new Classifier.
func NewClassifier() *Classifier {
	return &Classifier{}
}

// Classify determines the classification of a failure given the original and retry results.
func (c *Classifier) Classify(original, retry model.TestResult) model.Classification {
	// Infrastructure: original crashed (not a test failure).
	if original.Status == model.StatusError {
		return model.Infrastructure
	}

	// Flake: failed then passed on retry with no code change.
	if original.Status == model.StatusFail && retry.Status == model.StatusPass {
		return model.Flake
	}

	// Real: failed on both runs.
	if original.Status == model.StatusFail && retry.Status == model.StatusFail {
		return model.RealFailure
	}

	// Infrastructure: retry itself crashed.
	if retry.Status == model.StatusError {
		return model.Infrastructure
	}

	return model.Unclassified
}
