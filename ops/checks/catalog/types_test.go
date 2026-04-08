package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

const validYAML = `checks:
  - id: go-build
    name: "Go build"
    kind: build
    language: go
    command: "go build ./..."
    avg_duration: 120
  - id: go-test
    name: "Go tests"
    kind: test
    language: go
    command: "go test ./..."
    avg_duration: 600
    packages: ["op-node"]
    prerequisites: ["go-build"]
  - id: lint
    name: "Lint"
    kind: lint
    language: go
    command: "golangci-lint run"
    avg_duration: 300
`

func TestParse_Valid(t *testing.T) {
	c, err := Parse([]byte(validYAML))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(c.Checks))
	}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid catalog, got error: %v", err)
	}
}

func TestLoad_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "checks.yaml")
	if err := os.WriteFile(path, []byte(validYAML), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(c.Checks))
	}
}

func TestValidate_DuplicateID(t *testing.T) {
	yaml := `checks:
  - id: dup
    name: "First"
    kind: test
    language: go
    command: "test1"
  - id: dup
    name: "Second"
    kind: test
    language: go
    command: "test2"
`
	c, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err == nil {
		t.Error("expected validation error for duplicate ID")
	}
}

func TestValidate_EmptyID(t *testing.T) {
	yaml := `checks:
  - id: ""
    name: "No ID"
    kind: test
    language: go
    command: "test"
`
	c, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err == nil {
		t.Error("expected validation error for empty ID")
	}
}

func TestValidate_DanglingPrerequisite(t *testing.T) {
	yaml := `checks:
  - id: test1
    name: "Test"
    kind: test
    language: go
    command: "test"
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

func TestValidate_EmptyCommand(t *testing.T) {
	yaml := `checks:
  - id: empty
    name: "Empty command"
    kind: test
    language: go
    command: ""
`
	c, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err == nil {
		t.Error("expected validation error for empty command")
	}
}

func TestByID(t *testing.T) {
	c, _ := Parse([]byte(validYAML))
	ch := c.ByID("go-test")
	if ch == nil {
		t.Fatal("expected to find 'go-test'")
	}
	if ch.Name != "Go tests" {
		t.Errorf("expected name 'Go tests', got %q", ch.Name)
	}

	if c.ByID("nonexistent") != nil {
		t.Error("expected nil for nonexistent ID")
	}
}

func TestByLanguage(t *testing.T) {
	c, _ := Parse([]byte(validYAML))
	goChecks := c.ByLanguage("go")
	if len(goChecks) != 3 {
		t.Errorf("expected 3 Go checks, got %d", len(goChecks))
	}
	solChecks := c.ByLanguage("solidity")
	if len(solChecks) != 0 {
		t.Errorf("expected 0 Solidity checks, got %d", len(solChecks))
	}
}
