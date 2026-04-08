package catalog

import "testing"

const validCatalogYAML = `check_types:
  - id: go-build
    name: "Go build"
    kind: build
    language: go
    command: "go build ./..."
    scopeable: false
    avg_duration: 120
  - id: go-test
    name: "Go tests"
    kind: test
    language: go
    command: "go test"
    scopeable: true
    scope_flag: ""
    scope_type: packages
    avg_duration: 600
    prerequisites: ["go-build"]
    knobs:
      - name: short
        type: bool
        flag: "-short"
        default: true
      - name: race
        type: bool
        flag: "-race"
        default: false
      - name: fuzz_runs
        type: int
        flag: "--fuzz-runs"
        default: 0
        min: 0
        max: 10000
`

func TestParse_ValidCheckTypes(t *testing.T) {
	c, err := Parse([]byte(validCatalogYAML))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.CheckTypes) != 2 {
		t.Errorf("expected 2 check types, got %d", len(c.CheckTypes))
	}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid catalog, got: %v", err)
	}
}

func TestValidate_KnobBounds(t *testing.T) {
	yaml := `check_types:
  - id: bad
    name: "Bad"
    kind: test
    language: go
    command: "test"
    avg_duration: 10
    knobs:
      - name: depth
        type: int
        flag: "--depth"
        default: 5
        min: 100
        max: 10
`
	c, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err == nil {
		t.Error("expected validation error for min > max")
	}
}

func TestValidate_EnumRequiresChoices(t *testing.T) {
	yaml := `check_types:
  - id: bad
    name: "Bad"
    kind: test
    language: go
    command: "test"
    avg_duration: 10
    knobs:
      - name: mode
        type: enum
        flag: "--mode"
        default: "fast"
`
	c, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err == nil {
		t.Error("expected validation error for enum without choices")
	}
}

func TestValidate_UnknownPrerequisite(t *testing.T) {
	yaml := `check_types:
  - id: test
    name: "Test"
    kind: test
    language: go
    command: "test"
    avg_duration: 10
    prerequisites: ["nonexistent"]
`
	c, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err == nil {
		t.Error("expected validation error for dangling prerequisite")
	}
}

func TestValidate_DuplicateID(t *testing.T) {
	yaml := `check_types:
  - id: dup
    name: "First"
    kind: test
    language: go
    command: "test1"
    avg_duration: 10
  - id: dup
    name: "Second"
    kind: test
    language: go
    command: "test2"
    avg_duration: 10
`
	c, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err == nil {
		t.Error("expected validation error for duplicate ID")
	}
}

func TestByID(t *testing.T) {
	c, _ := Parse([]byte(validCatalogYAML))
	ct := c.ByID("go-test")
	if ct == nil {
		t.Fatal("expected to find 'go-test'")
	}
	if ct.Name != "Go tests" {
		t.Errorf("expected 'Go tests', got %q", ct.Name)
	}
	if !ct.Scopeable {
		t.Error("expected go-test to be scopeable")
	}
	if len(ct.Knobs) != 3 {
		t.Errorf("expected 3 knobs, got %d", len(ct.Knobs))
	}
	if c.ByID("nonexistent") != nil {
		t.Error("expected nil for nonexistent ID")
	}
}

func TestMatchesTriggers(t *testing.T) {
	ct := &CheckType{
		Triggers: []string{"packages/contracts-bedrock/src/**", "go.mod"},
	}

	if !ct.MatchesTriggers([]string{"packages/contracts-bedrock/src/L1/Foo.sol"}) {
		t.Error("expected match for src/ file")
	}
	if !ct.MatchesTriggers([]string{"go.mod"}) {
		t.Error("expected match for go.mod")
	}
	if ct.MatchesTriggers([]string{"op-node/rollup/derive/foo.go"}) {
		t.Error("expected no match for Go file")
	}
}
