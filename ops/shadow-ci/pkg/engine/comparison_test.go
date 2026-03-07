package engine

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestComparisonEngine_PerfectCatchRate(t *testing.T) {
	ce := NewComparisonEngine(nil)

	shadow := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestA", Package: "pkg"}, Status: model.StatusFail, Classification: model.RealFailure},
		{Test: model.TestIdentifier{Name: "TestB", Package: "pkg"}, Status: model.StatusPass},
	}

	main := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestA", Package: "pkg"}, Status: model.StatusFail},
		{Test: model.TestIdentifier{Name: "TestB", Package: "pkg"}, Status: model.StatusPass},
	}

	result := ce.Compare(shadow, main)
	assert.Equal(t, 1.0, result.CatchRate)
	assert.Equal(t, 0, result.FalseNegatives)
	assert.Equal(t, 1, result.ShadowCICaught)
}

func TestComparisonEngine_FalseNegative(t *testing.T) {
	ce := NewComparisonEngine(nil)

	// Shadow CI didn't run TestC at all.
	shadow := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestA", Package: "pkg"}, Status: model.StatusPass},
	}

	main := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestA", Package: "pkg"}, Status: model.StatusPass},
		{Test: model.TestIdentifier{Name: "TestC", Package: "pkg"}, Status: model.StatusFail, Language: "go"},
	}

	result := ce.Compare(shadow, main)
	assert.Equal(t, 1, result.FalseNegatives)
	assert.Equal(t, 0.0, result.CatchRate)
	assert.Len(t, result.FalseNegativeDetails, 1)
	assert.Equal(t, "TestC", result.FalseNegativeDetails[0].Test.Name)
}

func TestComparisonEngine_NoFailures(t *testing.T) {
	ce := NewComparisonEngine(nil)

	shadow := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestA", Package: "pkg"}, Status: model.StatusPass},
	}
	main := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestA", Package: "pkg"}, Status: model.StatusPass},
	}

	result := ce.Compare(shadow, main)
	assert.Equal(t, 1.0, result.CatchRate) // No failures to miss.
	assert.Equal(t, 0, result.FalseNegatives)
}

func TestComparisonEngine_FlakeInShadow(t *testing.T) {
	ce := NewComparisonEngine(nil)

	shadow := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestFlaky", Package: "pkg"}, Status: model.StatusFail, Classification: model.Flake, Fingerprint: "go:pkg:abc123"},
	}
	main := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestFlaky", Package: "pkg"}, Status: model.StatusFail},
	}

	result := ce.Compare(shadow, main)
	assert.Equal(t, 1, result.MainCIFlakeFailures)
	assert.Equal(t, 1, result.ShadowCIFlakes)
}
