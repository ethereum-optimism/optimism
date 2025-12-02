package main

import (
	"reflect"
	"slices"
	"sort"
	"testing"
	"time"
)

func TestKeyFor(t *testing.T) {
	if got := keyFor("pkg/a", ""); got != "pkg/a::" {
		t.Fatalf("empty name => got %q", got)
	}
	if got := keyFor("pkg/a", "  TestFoo  "); got != "pkg/a::TestFoo" {
		t.Fatalf("trim => got %q", got)
	}
}

func TestParseDayEnd(t *testing.T) {
	end := parseDayEnd("2025-01-02")
	want := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)
	if !end.Equal(want) {
		t.Fatalf("end != start-of-next-day: got %v want %v", end, want)
	}
}

func TestEnsureAggAndAggregate(t *testing.T) {
	// ensureAgg behavior
	m := map[string]*aggStats{}
	s := ensureAgg(m, "pkg::T1", "pkg", "T1", "2025-01-01")
	if s.FirstSeenDay != "2025-01-01" || s.LastSeenDay != "2025-01-01" {
		t.Fatalf("first/last not set")
	}
	s2 := ensureAgg(m, "pkg::T1", "pkg", "T1", "2025-01-02")
	if s2 != s {
		t.Fatalf("ensureAgg did not return same pointer for same key")
	}
	if !slices.Contains(s.DaysObserved, "2025-01-01") || !slices.Contains(s.DaysObserved, "2025-01-02") {
		t.Fatalf("days observed missing: %v", s.DaysObserved)
	}

	// aggregate behavior across days
	day1 := DailySummary{Date: "2025-01-01"}
	day1.UnstableTests = append(day1.UnstableTests, struct {
		TestName  string  `json:"test_name"`
		Package   string  `json:"package"`
		TotalRuns int     `json:"total_runs"`
		Passes    int     `json:"passes"`
		Failures  int     `json:"failures"`
		PassRate  float64 `json:"pass_rate"`
	}{TestName: "T1", Package: "pkg", TotalRuns: 10, Passes: 10, Failures: 0})

	day2 := DailySummary{Date: "2025-01-02"}
	day2.UnstableTests = append(day2.UnstableTests, struct {
		TestName  string  `json:"test_name"`
		Package   string  `json:"package"`
		TotalRuns int     `json:"total_runs"`
		Passes    int     `json:"passes"`
		Failures  int     `json:"failures"`
		PassRate  float64 `json:"pass_rate"`
	}{TestName: "T1", Package: "pkg", TotalRuns: 5, Passes: 4, Failures: 1})

	agg := aggregate(map[string]DailySummary{
		day1.Date: day1,
		day2.Date: day2,
	})
	a := agg["pkg::T1"]
	if a == nil || a.TotalRuns != 15 || a.Passes != 14 || a.Failures != 1 {
		t.Fatalf("bad aggregate: %+v", a)
	}
	// DaysObserved should be sorted and include both
	wantDays := []string{"2025-01-01", "2025-01-02"}
	gotDays := append([]string(nil), a.DaysObserved...)
	sort.Strings(gotDays)
	if !reflect.DeepEqual(gotDays, wantDays) {
		t.Fatalf("days mismatch: got %v want %v", gotDays, wantDays)
	}
}

func TestSelectPromotionCandidates(t *testing.T) {
	now := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	flake := map[string]testInfo{"pkg::T1": {Timeout: "1m"}}
	agg := map[string]*aggStats{
		"pkg::T1": {
			Package:      "pkg",
			TestName:     "T1",
			TotalRuns:    100,
			Passes:       100,
			Failures:     0,
			FirstSeenDay: "2025-01-01",
			DaysObserved: []string{"2025-01-01", "2025-01-02"},
		},
	}
	cands, reasons := selectPromotionCandidates(agg, flake, 50, 0.01, true, 3, now)
	if len(reasons) != 0 {
		t.Fatalf("unexpected reasons: %v", reasons)
	}
	if len(cands) != 1 || cands[0].Package != "pkg" || cands[0].TestName != "T1" {
		t.Fatalf("unexpected candidates: %+v", cands)
	}
}

func TestComputeUpdatedConfig(t *testing.T) {
	cfg := acceptanceYAML{Gates: []gateYAML{{ID: "flake-shake", Tests: []testEntry{{Package: "pkg", Name: "T1"}, {Package: "pkg", Name: "T2"}}}}}
	cands := []promoteCandidate{{Package: "pkg", TestName: "T1"}}
	updated := computeUpdatedConfig(cfg, "flake-shake", cands)
	tests := updated.Gates[0].Tests
	if len(tests) != 1 || tests[0].Name != "T2" {
		t.Fatalf("expected only T2 to remain, got %+v", tests)
	}
}
