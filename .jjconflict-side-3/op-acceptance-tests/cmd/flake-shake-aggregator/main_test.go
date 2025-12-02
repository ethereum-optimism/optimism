package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeReport(t *testing.T, dir, name string, r FlakeShakeReport) string {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, b, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestRunAggregatesReports(t *testing.T) {
	tmp := t.TempDir()
	// create two worker reports
	r1 := FlakeShakeReport{
		Date:       "2025-01-01",
		Gate:       "flake-shake",
		Iterations: 10,
		Tests: []FlakeShakeResult{{
			TestName:    "pkg::T1",
			Package:     "pkg",
			TotalRuns:   10,
			Passes:      9,
			Failures:    1,
			Skipped:     0,
			AvgDuration: 100 * time.Millisecond,
			MinDuration: 80 * time.Millisecond,
			MaxDuration: 120 * time.Millisecond,
		}},
		GeneratedAt: time.Now(),
		RunID:       "abc",
	}
	r2 := FlakeShakeReport{
		Date:       "2025-01-01",
		Gate:       "flake-shake",
		Iterations: 5,
		Tests: []FlakeShakeResult{{
			TestName:    "pkg::T1",
			Package:     "pkg",
			TotalRuns:   5,
			Passes:      5,
			Failures:    0,
			Skipped:     0,
			AvgDuration: 90 * time.Millisecond,
			MinDuration: 70 * time.Millisecond,
			MaxDuration: 110 * time.Millisecond,
		}},
		GeneratedAt: time.Now(),
		RunID:       "abc",
	}
	// Place files under pattern
	d1 := filepath.Join(tmp, "flake-shake-results-worker-1")
	d2 := filepath.Join(tmp, "flake-shake-results-worker-2")
	if err := os.MkdirAll(d1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(d2, 0755); err != nil {
		t.Fatal(err)
	}
	writeReport(t, d1, "flake-shake-report.json", r1)
	writeReport(t, d2, "flake-shake-report.json", r2)

	out := filepath.Join(tmp, "final")
	if err := run(filepath.Join(tmp, "flake-shake-results-worker-*/flake-shake-report.json"), out, false); err != nil {
		t.Fatalf("run error: %v", err)
	}
	// verify outputs exist
	if _, err := os.Stat(filepath.Join(out, "flake-shake-report.json")); err != nil {
		t.Fatalf("missing json report: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "flake-shake-report.html")); err != nil {
		t.Fatalf("missing html report: %v", err)
	}
}

func TestGenerateHTMLReportBasic(t *testing.T) {
	r := &FlakeShakeReport{
		Gate:       "flake-shake",
		Iterations: 15,
		Tests: []FlakeShakeResult{
			{TestName: "pkg::T1", Package: "pkg", TotalRuns: 10, Passes: 10, Failures: 0, PassRate: 100},
			{TestName: "pkg::T2", Package: "pkg", TotalRuns: 5, Passes: 4, Failures: 1, PassRate: 80},
		},
		GeneratedAt: time.Now(),
	}
	html := generateHTMLReport(r)
	if len(html) == 0 {
		t.Fatal("empty html")
	}
	if !strings.Contains(html, "Flake-Shake Report") {
		t.Fatal("missing title")
	}
}
