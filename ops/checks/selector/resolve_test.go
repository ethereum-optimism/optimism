package selector

import (
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
)

func TestResolvedCommand_ForgeTestWithScope(t *testing.T) {
	ct := &catalog.CheckType{
		Command:   "cd packages/contracts-bedrock && forge test",
		Scopeable: true,
		ScopeFlag: "--match-path",
		ScopeType: "paths",
		Knobs: []catalog.Knob{
			{Name: "fuzz_runs", Type: "int", Flag: "--fuzz-runs"},
		},
	}

	item := &ExecutionItem{
		Scope:  []string{"./test/L1/*"},
		Config: map[string]any{"fuzz_runs": 128},
	}

	cmd := item.ResolvedCommand(ct)
	if !strings.Contains(cmd, "--match-path './test/L1/*'") {
		t.Errorf("expected --match-path in command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "--fuzz-runs 128") {
		t.Errorf("expected --fuzz-runs 128 in command, got: %s", cmd)
	}
}

func TestResolvedCommand_GoTestWithFlags(t *testing.T) {
	ct := &catalog.CheckType{
		Command:   "go test",
		Scopeable: true,
		ScopeFlag: "",
		ScopeType: "packages",
		Knobs: []catalog.Knob{
			{Name: "short", Type: "bool", Flag: "-short"},
			{Name: "race", Type: "bool", Flag: "-race"},
		},
	}

	item := &ExecutionItem{
		Scope:  []string{"./op-node/..."},
		Config: map[string]any{"short": true, "race": true},
	}

	cmd := item.ResolvedCommand(ct)
	if !strings.Contains(cmd, "-short") {
		t.Errorf("expected -short in command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "-race") {
		t.Errorf("expected -race in command, got: %s", cmd)
	}
	if !strings.HasSuffix(cmd, "./op-node/...") {
		t.Errorf("expected command to end with scope, got: %s", cmd)
	}
}

func TestResolvedCommand_BinaryCheck(t *testing.T) {
	ct := &catalog.CheckType{
		Command:   "just snapshots-check-no-build",
		Scopeable: false,
	}

	item := &ExecutionItem{}
	cmd := item.ResolvedCommand(ct)
	if cmd != "just snapshots-check-no-build" {
		t.Errorf("expected base command unchanged, got: %s", cmd)
	}
}

func TestResolvedCommand_BoolFalseOmitted(t *testing.T) {
	ct := &catalog.CheckType{
		Command:   "go test",
		Scopeable: true,
		ScopeFlag: "",
		Knobs: []catalog.Knob{
			{Name: "short", Type: "bool", Flag: "-short"},
			{Name: "race", Type: "bool", Flag: "-race"},
		},
	}

	item := &ExecutionItem{
		Scope:  []string{"./op-node/..."},
		Config: map[string]any{"short": true, "race": false},
	}

	cmd := item.ResolvedCommand(ct)
	if strings.Contains(cmd, "-race") {
		t.Errorf("expected -race to be omitted when false, got: %s", cmd)
	}
	if !strings.Contains(cmd, "-short") {
		t.Errorf("expected -short to be present, got: %s", cmd)
	}
}
