package engine

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// Normalizer replaces dynamic content in error messages with stable placeholders.
type Normalizer struct {
	Pattern     *regexp.Regexp
	Replacement string
}

var defaultNormalizers = []Normalizer{
	{regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`), "<timestamp>"},
	{regexp.MustCompile(`0x[0-9a-fA-F]{8,}`), "<hex>"},
	{regexp.MustCompile(`:\d{4,5}`), ":<port>"},
	{regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), "<uuid>"},
	{regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`), "<ip>"},
	{regexp.MustCompile(`/tmp/[^\s]+`), "<tmppath>"},
	{regexp.MustCompile(`goroutine \d+`), "goroutine <N>"},
	{regexp.MustCompile(`\d+(\.\d+)?s`), "<duration>"},
}

// Fingerprinter produces normalized fingerprints for flake clustering.
type Fingerprinter struct {
	normalizers []Normalizer
}

// NewFingerprinter creates a Fingerprinter with default normalizers.
func NewFingerprinter() *Fingerprinter {
	return &Fingerprinter{normalizers: defaultNormalizers}
}

// Fingerprint produces a stable identifier for a test failure.
func (f *Fingerprinter) Fingerprint(result model.TestResult) string {
	errorLine := extractErrorLine(result.Output)

	normalized := errorLine
	for _, n := range f.normalizers {
		normalized = n.Pattern.ReplaceAllString(normalized, n.Replacement)
	}

	hash := sha256Short(normalized)
	return fmt.Sprintf("%s:%s:%s", result.Language, result.Test.Package, hash)
}

func extractErrorLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "fail") ||
			strings.Contains(lower, "panic") || strings.Contains(lower, "assert") {
			return line
		}
	}
	// Fallback: first non-empty line.
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func sha256Short(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:8])
}
