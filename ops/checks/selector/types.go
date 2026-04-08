package selector

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
)

// ExecutionItem is a single thing to run — a command with scope and config.
type ExecutionItem struct {
	ID            string         `json:"id"`              // unique: "forge-test:L1" or "snapshots-check"
	CheckTypeID   string         `json:"check_type_id"`   // "forge-test", "go-test", "golangci-lint"
	Scope         []string       `json:"scope,omitempty"` // paths, packages, test names
	Config        map[string]any `json:"config,omitempty"`
	Signal        float64        `json:"signal"`
	RunCost       float64        `json:"run_cost"`
	SkipCost      float64        `json:"skip_cost"`
	Prerequisites []string       `json:"prerequisites,omitempty"` // item IDs
}

// ResolvedCommand builds the full shell command from the check type, scope, and config.
func (e *ExecutionItem) ResolvedCommand(ct *catalog.CheckType) string {
	if ct == nil {
		return ""
	}

	cmd := ct.Command

	// For non-scopeable checks, return the base command
	if !ct.Scopeable || len(e.Scope) == 0 {
		return appendKnobFlags(cmd, ct, e.Config)
	}

	// Build scope args
	var scopeParts []string
	for _, s := range e.Scope {
		if ct.ScopeFlag != "" {
			scopeParts = append(scopeParts, fmt.Sprintf("%s '%s'", ct.ScopeFlag, s))
		} else {
			scopeParts = append(scopeParts, s)
		}
	}

	// For positional scope (go test), knob flags come before scope
	if ct.ScopeFlag == "" {
		flagged := appendKnobFlags(cmd, ct, e.Config)
		return flagged + " " + strings.Join(scopeParts, " ")
	}

	// For flagged scope (forge test --match-path), scope comes after command, then knobs
	cmd = cmd + " " + strings.Join(scopeParts, " ")
	return appendKnobFlags(cmd, ct, e.Config)
}

func appendKnobFlags(cmd string, ct *catalog.CheckType, config map[string]any) string {
	if config == nil {
		return cmd
	}
	for _, knob := range ct.Knobs {
		val, ok := config[knob.Name]
		if !ok {
			continue
		}
		if knob.Flag == "" {
			continue // env var only, not a CLI flag
		}
		switch knob.Type {
		case "bool":
			if b, ok := val.(bool); ok && b {
				cmd += " " + knob.Flag
			}
		case "int":
			switch v := val.(type) {
			case int:
				if v != 0 {
					cmd += fmt.Sprintf(" %s %d", knob.Flag, v)
				}
			case float64:
				if v != 0 {
					cmd += fmt.Sprintf(" %s %d", knob.Flag, int(v))
				}
			}
		case "string":
			if s, ok := val.(string); ok && s != "" {
				cmd += fmt.Sprintf(" %s '%s'", knob.Flag, s)
			}
		}
	}
	return cmd
}

// Result is the output of the selection process.
type Result struct {
	Stage     string
	Items     []ExecutionItem
	Skipped   []ExecutionItem
	Schedule  Schedule
	WallClock float64
	TotalCPU  float64
}
