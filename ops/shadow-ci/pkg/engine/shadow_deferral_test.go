package engine

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func TestShadowDeferralAnalyzer_NoMisses(t *testing.T) {
	analyzer := NewShadowDeferralAnalyzer(nil)

	results := map[string][]model.TestResult{
		"go_tests": {
			{Test: model.TestIdentifier{Package: "pkg/a", Name: "TestA"}, Status: model.StatusPass, Duration: 2 * time.Second, WouldDefer: true, DeferTo: "merge_queue"},
			{Test: model.TestIdentifier{Package: "pkg/b", Name: "TestB"}, Status: model.StatusPass, Duration: 5 * time.Second, WouldDefer: true, DeferTo: "merge_queue"},
			{Test: model.TestIdentifier{Package: "pkg/c", Name: "TestC"}, Status: model.StatusPass, Duration: 1 * time.Second},
		},
	}

	report := analyzer.Analyze("pipe-1", model.StagePR, results)

	if report.TotalTests != 3 {
		t.Errorf("TotalTests = %d, want 3", report.TotalTests)
	}
	if report.WouldDefer != 2 {
		t.Errorf("WouldDefer = %d, want 2", report.WouldDefer)
	}
	if report.ActualMisses != 0 {
		t.Errorf("ActualMisses = %d, want 0", report.ActualMisses)
	}
	if report.MissRate != 0 {
		t.Errorf("MissRate = %f, want 0", report.MissRate)
	}
	if report.EstTimeSaved != 7*time.Second {
		t.Errorf("EstTimeSaved = %v, want 7s", report.EstTimeSaved)
	}
}

func TestShadowDeferralAnalyzer_WithMiss(t *testing.T) {
	analyzer := NewShadowDeferralAnalyzer(nil)

	results := map[string][]model.TestResult{
		"go_tests": {
			{Test: model.TestIdentifier{Package: "pkg/a", Name: "TestA"}, Status: model.StatusPass, Duration: 2 * time.Second, WouldDefer: true, DeferTo: "merge_queue"},
			{Test: model.TestIdentifier{Package: "pkg/b", Name: "TestB"}, Status: model.StatusFail, Classification: model.RealFailure, Duration: 5 * time.Second, WouldDefer: true, DeferTo: "merge_queue"},
			{Test: model.TestIdentifier{Package: "pkg/c", Name: "TestC"}, Status: model.StatusPass, Duration: 1 * time.Second},
		},
	}

	report := analyzer.Analyze("pipe-2", model.StagePR, results)

	if report.ActualMisses != 1 {
		t.Errorf("ActualMisses = %d, want 1", report.ActualMisses)
	}
	if report.WouldDefer != 2 {
		t.Errorf("WouldDefer = %d, want 2", report.WouldDefer)
	}
	if report.MissRate != 0.5 {
		t.Errorf("MissRate = %f, want 0.5", report.MissRate)
	}
}

func TestShadowDeferralAnalyzer_FlakeNotMiss(t *testing.T) {
	analyzer := NewShadowDeferralAnalyzer(nil)

	results := map[string][]model.TestResult{
		"go_tests": {
			{Test: model.TestIdentifier{Package: "pkg/a", Name: "TestA"}, Status: model.StatusFail, Classification: model.Flake, Duration: 3 * time.Second, WouldDefer: true, DeferTo: "merge_queue"},
		},
	}

	report := analyzer.Analyze("pipe-3", model.StagePR, results)

	if report.ActualMisses != 0 {
		t.Errorf("ActualMisses = %d, want 0 (flake should not count as miss)", report.ActualMisses)
	}
	if report.WouldDefer != 1 {
		t.Errorf("WouldDefer = %d, want 1", report.WouldDefer)
	}
}
