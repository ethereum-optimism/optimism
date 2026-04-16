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
	Profile       string         `json:"profile,omitempty"` // test profile (e.g. "custom_gas_token")
	Signal        float64        `json:"signal"`
	RunCost       float64        `json:"run_cost"`
	SkipCost      float64        `json:"skip_cost"`
	Prerequisites []string       `json:"prerequisites,omitempty"` // item IDs
}

// ResolvedCommand builds the full shell command from the check type, scope, and config.
// If the item specifies a profile, profile env vars are prepended via `cd ... && ENV=val command`.
func (e *ExecutionItem) ResolvedCommand(ct *catalog.CheckType) string {
	return e.ResolvedCommandWithCatalog(ct, nil)
}

// ResolvedCommandWithCatalog is like ResolvedCommand but takes a catalog to look
// up profile env vars. If cat is nil or the profile isn't found, env vars are omitted.
func (e *ExecutionItem) ResolvedCommandWithCatalog(ct *catalog.CheckType, cat *catalog.Catalog) string {
	if ct == nil {
		return ""
	}

	cmd := ct.Command

	// For non-scopeable checks, return the base command
	if !ct.Scopeable || len(e.Scope) == 0 {
		return prependEnv(appendKnobFlags(cmd, ct, e.Config), e.Profile, cat)
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
		return prependEnv(flagged+" "+strings.Join(scopeParts, " "), e.Profile, cat)
	}

	// For flagged scope (forge test --match-path), scope comes after command, then knobs
	cmd = cmd + " " + strings.Join(scopeParts, " ")
	return prependEnv(appendKnobFlags(cmd, ct, e.Config), e.Profile, cat)
}

// prependEnv prepends profile env vars to the command.
// Handles commands with `cd X && <real command>` by injecting env vars after the &&.
func prependEnv(cmd string, profileName string, cat *catalog.Catalog) string {
	if profileName == "" || cat == nil {
		return cmd
	}
	profile := cat.ProfileByName(profileName)
	if profile == nil || len(profile.Env) == 0 {
		return cmd
	}

	// Build "K1=v1 K2=v2" prefix (sorted for determinism)
	keys := make([]string, 0, len(profile.Env))
	for k := range profile.Env {
		keys = append(keys, k)
	}
	sortStrings(keys)
	var envParts []string
	for _, k := range keys {
		envParts = append(envParts, fmt.Sprintf("%s=%s", k, profile.Env[k]))
	}
	envPrefix := strings.Join(envParts, " ")

	// If the command starts with "cd X && ", inject env vars after the &&
	if idx := strings.Index(cmd, " && "); idx >= 0 && strings.HasPrefix(cmd, "cd ") {
		return cmd[:idx+4] + envPrefix + " " + cmd[idx+4:]
	}
	return envPrefix + " " + cmd
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
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
