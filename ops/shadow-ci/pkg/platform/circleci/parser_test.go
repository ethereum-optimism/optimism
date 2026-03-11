package circleci

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func TestParseGotestsumJSON(t *testing.T) {
	input := `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"pkg/foo","Test":"TestA"}
{"Time":"2024-01-01T00:00:01Z","Action":"pass","Package":"pkg/foo","Test":"TestA","Elapsed":1.5}
{"Time":"2024-01-01T00:00:02Z","Action":"fail","Package":"pkg/bar","Test":"TestB","Elapsed":0.5}
{"Time":"2024-01-01T00:00:03Z","Action":"skip","Package":"pkg/baz","Test":"TestC","Elapsed":0}
{"Time":"2024-01-01T00:00:04Z","Action":"pass","Package":"pkg/foo","Elapsed":2.0}
`

	results, err := ParseGotestsumJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 results: TestA pass, TestB fail, TestC skip.
	// Package-level events (no Test field) are ignored.
	// "run" actions are ignored.
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Test.Name != "TestA" || results[0].Status != model.StatusPass {
		t.Errorf("results[0] = %v, want TestA pass", results[0])
	}
	if results[1].Test.Name != "TestB" || results[1].Status != model.StatusFail {
		t.Errorf("results[1] = %v, want TestB fail", results[1])
	}
	if results[2].Test.Name != "TestC" || results[2].Status != model.StatusSkip {
		t.Errorf("results[2] = %v, want TestC skip", results[2])
	}
}

func TestParseJUnitXML(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="pkg/foo" tests="2" failures="1" time="3.0">
    <testcase name="TestA" classname="pkg/foo" time="1.5"/>
    <testcase name="TestB" classname="pkg/foo" time="1.5">
      <failure message="assertion failed">expected 1 got 2</failure>
    </testcase>
  </testsuite>
</testsuites>`

	results, err := ParseJUnitXML([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Test.Name != "TestA" || results[0].Status != model.StatusPass {
		t.Errorf("results[0] = %v, want TestA pass", results[0])
	}
	if results[1].Test.Name != "TestB" || results[1].Status != model.StatusFail {
		t.Errorf("results[1] = %v, want TestB fail", results[1])
	}
	if results[1].Output != "expected 1 got 2" {
		t.Errorf("results[1].Output = %q, want failure body", results[1].Output)
	}
}

func TestParseForgeJSON(t *testing.T) {
	input := `{
  "test_results": [
    {"contract": "test/L1Bridge.t.sol", "test": "test_deposit", "status": "Success", "duration_ms": 150},
    {"contract": "test/L1Bridge.t.sol", "test": "test_withdraw", "status": "Failure", "duration_ms": 200, "reason": "revert"}
  ]
}`

	results, err := ParseForgeJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Test.Name != "test_deposit" || results[0].Status != model.StatusPass {
		t.Errorf("results[0] = %v, want test_deposit pass", results[0])
	}
	if results[1].Test.Name != "test_withdraw" || results[1].Status != model.StatusFail {
		t.Errorf("results[1] = %v, want test_withdraw fail", results[1])
	}
}
