package coverage

// RustCollector collects coverage from Rust tests.
// Stub — not yet implemented.
type RustCollector struct{}

func NewRustCollector() *RustCollector { return &RustCollector{} }

func (c *RustCollector) Language() string { return "rust" }

func (c *RustCollector) Collect(_ string, _ string) (*Report, error) {
	return &Report{Language: "rust", Covers: make(map[string][][2]int)}, nil
}
