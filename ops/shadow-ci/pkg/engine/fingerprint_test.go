package engine

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestFingerprinter_Stability(t *testing.T) {
	fp := NewFingerprinter()

	r1 := model.TestResult{
		Test:     model.TestIdentifier{Name: "TestFoo", Package: "pkg/bar"},
		Language: "go",
		Output:   "Error: connection refused at 127.0.0.1:8545 at 2026-03-07T10:30:00",
	}

	r2 := model.TestResult{
		Test:     model.TestIdentifier{Name: "TestFoo", Package: "pkg/bar"},
		Language: "go",
		Output:   "Error: connection refused at 10.0.0.1:9090 at 2026-03-08T15:45:30",
	}

	// Same error with different IPs, ports, timestamps should produce same fingerprint.
	f1 := fp.Fingerprint(r1)
	f2 := fp.Fingerprint(r2)
	assert.Equal(t, f1, f2)
}

func TestFingerprinter_DifferentErrors(t *testing.T) {
	fp := NewFingerprinter()

	r1 := model.TestResult{
		Test:     model.TestIdentifier{Name: "TestFoo", Package: "pkg/bar"},
		Language: "go",
		Output:   "Error: assertion failed: expected 42 got 0",
	}

	r2 := model.TestResult{
		Test:     model.TestIdentifier{Name: "TestFoo", Package: "pkg/bar"},
		Language: "go",
		Output:   "panic: runtime error: index out of range",
	}

	f1 := fp.Fingerprint(r1)
	f2 := fp.Fingerprint(r2)
	assert.NotEqual(t, f1, f2)
}

func TestFingerprinter_Format(t *testing.T) {
	fp := NewFingerprinter()

	r := model.TestResult{
		Test:     model.TestIdentifier{Name: "TestFoo", Package: "pkg/bar"},
		Language: "go",
		Output:   "Error: something failed",
	}

	f := fp.Fingerprint(r)
	assert.Contains(t, f, "go:")
	assert.Contains(t, f, "pkg/bar:")
}

func TestExtractErrorLine(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Error: something failed\nother output", "Error: something failed"},
		{"some output\npanic: nil pointer\nmore", "panic: nil pointer"},
		{"all good\nno issues\n", "all good"},
		{"", ""},
	}

	for _, tt := range tests {
		result := extractErrorLine(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}
