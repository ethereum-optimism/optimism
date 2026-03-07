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

func TestComparisonEngine_MultipleFlakesDeduped(t *testing.T) {
	ce := NewComparisonEngine(nil)

	// Two flaky tests with the same fingerprint should be deduped.
	shadow := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestFlaky1", Package: "pkg"}, Status: model.StatusFail, Classification: model.Flake, Fingerprint: "go:pkg:same"},
		{Test: model.TestIdentifier{Name: "TestFlaky2", Package: "pkg2"}, Status: model.StatusFail, Classification: model.Flake, Fingerprint: "go:pkg:same"},
		{Test: model.TestIdentifier{Name: "TestFlaky3", Package: "pkg3"}, Status: model.StatusFail, Classification: model.Flake, Fingerprint: "go:pkg:different"},
	}
	main := []model.TestResult{}

	result := ce.Compare(shadow, main)
	assert.Equal(t, 2, result.ShadowCIFlakes) // 2 unique fingerprints
	assert.Len(t, result.UniqueFingerprints, 2)
}

func TestComparisonEngine_PartialCatchRate(t *testing.T) {
	ce := NewComparisonEngine(nil)

	shadow := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestA", Package: "pkg"}, Status: model.StatusFail, Classification: model.RealFailure},
		// TestB not in shadow set
	}
	main := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestA", Package: "pkg"}, Status: model.StatusFail},
		{Test: model.TestIdentifier{Name: "TestB", Package: "pkg"}, Status: model.StatusFail},
	}

	result := ce.Compare(shadow, main)
	assert.Equal(t, 0.5, result.CatchRate) // caught 1 of 2
	assert.Equal(t, 1, result.ShadowCICaught)
	assert.Equal(t, 1, result.FalseNegatives)
}

func TestComparisonEngine_Performance(t *testing.T) {
	ce := NewComparisonEngine(nil)

	shadow := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestA", Package: "pkg"}, Duration: 5 * 60e9}, // 5 min
	}
	main := []model.TestResult{
		{Test: model.TestIdentifier{Name: "TestA", Package: "pkg"}, Duration: 10 * 60e9}, // 10 min
	}

	result := ce.Compare(shadow, main)
	assert.Equal(t, 2.0, result.Speedup) // 10min / 5min
	assert.InDelta(t, 0.5, result.ComputeReduction, 0.01)
}

func TestComparisonEngine_EmptyResults(t *testing.T) {
	ce := NewComparisonEngine(nil)
	result := ce.Compare(nil, nil)
	assert.Equal(t, 1.0, result.CatchRate) // no failures = perfect
	assert.Equal(t, 0, result.FalseNegatives)
	assert.Equal(t, 0, result.ShadowCICaught)
}

// Correlation decay tests

func TestCorrelationDecay_Detected(t *testing.T) {
	ce := NewComparisonEngine(nil)

	falseNegatives := []model.FalseNegativeDetail{
		{
			Test:     model.TestIdentifier{Name: "TestSlow", Package: "pkg"},
			Language: "go",
		},
	}

	correlations := &CorrelationMatrix{
		Correlations: []Correlation{
			{
				TestA:      "pkg/TestFast",
				TestB:      "pkg/TestSlow",
				CoFailRate: 0.95,
			},
		},
	}

	passedTests := map[string]bool{
		"pkg/TestFast": true,
	}

	signals := ce.CheckCorrelationDecay(falseNegatives, model.PlacementConfig{}, correlations, passedTests)
	assert.Len(t, signals, 1)
	assert.Equal(t, "pkg/TestSlow", signals[0].DeferredTest)
	assert.Equal(t, "pkg/TestFast", signals[0].CorrelatedTest)
	assert.Equal(t, 0.95, signals[0].PreviousCoFailRate)
}

func TestCorrelationDecay_NoDecay(t *testing.T) {
	ce := NewComparisonEngine(nil)

	falseNegatives := []model.FalseNegativeDetail{
		{
			Test:     model.TestIdentifier{Name: "TestSlow", Package: "pkg"},
			Language: "go",
		},
	}

	correlations := &CorrelationMatrix{
		Correlations: []Correlation{
			{
				TestA:      "pkg/TestFast",
				TestB:      "pkg/TestSlow",
				CoFailRate: 0.95,
			},
		},
	}

	// TestFast also failed — no decay.
	passedTests := map[string]bool{}

	signals := ce.CheckCorrelationDecay(falseNegatives, model.PlacementConfig{}, correlations, passedTests)
	assert.Empty(t, signals)
}

func TestCorrelationDecay_NilCorrelations(t *testing.T) {
	ce := NewComparisonEngine(nil)

	falseNegatives := []model.FalseNegativeDetail{
		{Test: model.TestIdentifier{Name: "TestX", Package: "pkg"}},
	}

	signals := ce.CheckCorrelationDecay(falseNegatives, model.PlacementConfig{}, nil, nil)
	assert.Nil(t, signals)
}
