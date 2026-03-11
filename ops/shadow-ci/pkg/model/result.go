package model

import "time"

// TestStatus represents the outcome of a test execution.
type TestStatus string

const (
	StatusPass  TestStatus = "pass"
	StatusFail  TestStatus = "fail"
	StatusSkip  TestStatus = "skip"
	StatusError TestStatus = "error" // infrastructure failure (OOM, timeout, crash)
)

// Classification represents the determined cause of a test failure.
type Classification string

const (
	Unclassified   Classification = "unclassified"
	RealFailure    Classification = "real"
	Flake          Classification = "flake"
	Infrastructure Classification = "infrastructure"
)

// TestIdentifier uniquely identifies a single test within a package.
type TestIdentifier struct {
	Name    string `json:"name"`    // Test function name
	Package string `json:"package"` // Package/file the test lives in
}

// Key returns a string key for indexing test results.
func (t TestIdentifier) Key() string {
	return t.Package + "/" + t.Name
}

// TestResult is the universal representation of a test outcome.
type TestResult struct {
	Test     TestIdentifier `json:"test"`
	Language string         `json:"language"`
	Config   string         `json:"config"`

	Status   TestStatus    `json:"status"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output,omitempty"`

	// Set by the core Classifier, not the adapter.
	Classification Classification `json:"classification"`

	// Set by the core Fingerprinter, not the adapter.
	Fingerprint string `json:"fingerprint,omitempty"`

	// Shadow deferral annotations.
	WouldDefer bool   `json:"would_defer,omitempty"` // test would have been deferred
	DeferTo    string `json:"defer_to,omitempty"`     // stage it would be deferred to

	// Retry chain.
	RetryOf   *TestResult `json:"retry_of,omitempty"`
	RetriedBy *TestResult `json:"retried_by,omitempty"`
}
