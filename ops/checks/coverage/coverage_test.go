package coverage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReport_RoundTrip(t *testing.T) {
	r := &Report{
		Test:     "test/L1/OptimismPortal2.t.sol",
		Language: "solidity",
		Covers: map[string][][2]int{
			"src/L1/OptimismPortal2.sol": {{140, 180}, {200, 250}},
			"src/libraries/Hashing.sol":  {{10, 30}},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "coverage.json")

	if err := SaveReport(r, path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadReport(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Test != r.Test {
		t.Errorf("expected test %q, got %q", r.Test, loaded.Test)
	}
	if loaded.Language != r.Language {
		t.Errorf("expected language %q, got %q", r.Language, loaded.Language)
	}
	if len(loaded.Covers) != 2 {
		t.Errorf("expected 2 covered files, got %d", len(loaded.Covers))
	}

	portalRanges := loaded.Covers["src/L1/OptimismPortal2.sol"]
	if len(portalRanges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(portalRanges))
	}
	if portalRanges[0] != [2]int{140, 180} {
		t.Errorf("expected [140,180], got %v", portalRanges[0])
	}
}

func TestLoadReports_Directory(t *testing.T) {
	dir := t.TempDir()

	for i, name := range []string{"a.json", "b.json"} {
		r := &Report{Test: name, Language: "solidity", Covers: map[string][][2]int{
			"src/file.sol": {{i * 10, i*10 + 5}},
		}}
		data, _ := json.Marshal(r)
		os.WriteFile(filepath.Join(dir, name), data, 0644)
	}

	reports, err := LoadReports(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 2 {
		t.Errorf("expected 2 reports, got %d", len(reports))
	}
}

func TestCompactRanges(t *testing.T) {
	tests := []struct {
		input    []int
		expected [][2]int
	}{
		{[]int{1, 2, 3, 5, 6, 10}, [][2]int{{1, 3}, {5, 6}, {10, 10}}},
		{[]int{1}, [][2]int{{1, 1}}},
		{[]int{3, 1, 2}, [][2]int{{1, 3}}},
		{nil, nil},
	}

	for _, tt := range tests {
		got := compactRanges(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("compactRanges(%v) = %v, want %v", tt.input, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("compactRanges(%v)[%d] = %v, want %v", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestParseLCOV(t *testing.T) {
	lcov := `SF:src/L1/OptimismPortal2.sol
DA:10,1
DA:11,1
DA:12,0
DA:13,1
DA:20,1
DA:21,1
end_of_record
SF:src/libraries/Hashing.sol
DA:5,1
DA:6,1
end_of_record
`
	covers := parseLCOV(lcov)

	if len(covers) != 2 {
		t.Fatalf("expected 2 files, got %d", len(covers))
	}

	portal := covers["src/L1/OptimismPortal2.sol"]
	// Lines 10,11 hit, 12 miss, 13 hit, 20,21 hit → 3 ranges: [10,11], [13,13], [20,21]
	if len(portal) != 3 {
		t.Errorf("expected 3 ranges for portal, got %d: %v", len(portal), portal)
	}

	hashing := covers["src/libraries/Hashing.sol"]
	if len(hashing) != 1 || hashing[0] != [2]int{5, 6} {
		t.Errorf("expected [[5,6]] for hashing, got %v", hashing)
	}
}

func TestParseGoCoverprofile(t *testing.T) {
	profile := `mode: set
github.com/org/repo/pkg/foo.go:10.2,20.5 3 1
github.com/org/repo/pkg/foo.go:25.2,30.5 2 0
github.com/org/repo/pkg/bar.go:5.2,8.5 1 1
`
	covers := parseGoCoverprofile(profile)

	if len(covers) != 2 {
		t.Fatalf("expected 2 files, got %d", len(covers))
	}

	foo := covers["github.com/org/repo/pkg/foo.go"]
	if len(foo) != 1 || foo[0] != [2]int{10, 20} {
		t.Errorf("expected [[10,20]] for foo (count=0 excluded), got %v", foo)
	}

	bar := covers["github.com/org/repo/pkg/bar.go"]
	if len(bar) != 1 || bar[0] != [2]int{5, 8} {
		t.Errorf("expected [[5,8]] for bar, got %v", bar)
	}
}

func TestFindOverlaps(t *testing.T) {
	reports := []*Report{
		{
			Test:     "testA",
			Language: "solidity",
			Covers: map[string][][2]int{
				"file.sol": {{1, 10}}, // lines 1-10
			},
		},
		{
			Test:     "testB",
			Language: "solidity",
			Covers: map[string][][2]int{
				"file.sol": {{5, 15}}, // lines 5-15, overlaps with A on 5-10
			},
		},
		{
			Test:     "testC",
			Language: "solidity",
			Covers: map[string][][2]int{
				"other.sol": {{1, 5}}, // no overlap with A or B
			},
		},
	}

	overlaps := FindOverlaps(reports)

	// A and B overlap on lines 5-10 (6 lines)
	// A has 10 lines, B has 11 lines
	// fractionA = 6/10 = 0.6, fractionB = 6/11 ≈ 0.55
	foundAB := false
	foundBA := false
	for _, o := range overlaps {
		if o.TestA == "testA" && o.TestB == "testB" {
			foundAB = true
			if o.Fraction < 0.5 || o.Fraction > 0.7 {
				t.Errorf("expected A→B overlap ~0.6, got %f", o.Fraction)
			}
		}
		if o.TestA == "testB" && o.TestB == "testA" {
			foundBA = true
		}
	}
	if !foundAB {
		t.Error("expected overlap from A to B")
	}
	if !foundBA {
		t.Error("expected overlap from B to A")
	}
}
