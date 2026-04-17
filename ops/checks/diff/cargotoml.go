package diff

import (
	"sort"
	"strings"
)

// CargoTomlChange summarizes a Cargo.toml hunk set.
//
// AffectedDeps are crate-shaped keys whose lines changed in
// [dependencies], [dev-dependencies], or [build-dependencies] (or
// their [workspace.*-dependencies] / [workspace.dependencies]
// analogues). Phase 1 maps these to `rs:<crate>` (workspace member)
// or `mod:<crate>` (external) and feeds them into the reverse-walk
// the same way go.mod-affected module IDs are fed.
//
// ForceBlast fires when the diff touches structural sections that
// can't be captured as "a dep changed": the [workspace] members
// list, [package] identity fields (name, version, edition,
// rust-version, build, resolver), or a change to the [features]
// table. Callers fall back to full blast radius.
type CargoTomlChange struct {
	AffectedDeps []string
	ForceBlast   bool
}

// AnalyzeCargoTomlChange parses added + removed lines of a Cargo.toml
// FileDiff and returns the classified change.
//
// Implementation is line-level and intentionally forgiving: we track
// the most recent section header we saw inside the hunk, but hunks
// don't always include their enclosing header. When the section is
// ambiguous, heuristic line-shape rules decide whether a line looks
// like a dependency entry. This is the same trade-off as the go.mod
// parser: handle the common case cleanly, force blast for the rest.
func AnalyzeCargoTomlChange(d FileDiff) CargoTomlChange {
	if !isCargoTomlPath(d.Path) {
		return CargoTomlChange{}
	}

	seen := make(map[string]bool)
	out := CargoTomlChange{}

	for _, h := range d.Hunks {
		section := ""
		for _, l := range h.Context {
			if s, ok := sectionHeader(l); ok {
				section = s
			}
		}
		for _, line := range h.Added {
			if s, ok := sectionHeader(line); ok {
				section = s
				classifySectionHeaderChange(s, &out)
				continue
			}
			classifyCargoLine(line, section, seen, &out)
		}
		for _, line := range h.Removed {
			if s, ok := sectionHeader(line); ok {
				classifySectionHeaderChange(s, &out)
				continue
			}
			classifyCargoLine(line, section, seen, &out)
		}
	}

	for m := range seen {
		out.AffectedDeps = append(out.AffectedDeps, m)
	}
	sort.Strings(out.AffectedDeps)
	return out
}

// isCargoTomlPath matches "Cargo.toml" at any path depth.
func isCargoTomlPath(p string) bool {
	return p == "Cargo.toml" || strings.HasSuffix(p, "/Cargo.toml")
}

// sectionHeader extracts the bracketed section name from a line, if
// present. "[workspace]" → ("workspace", true).
func sectionHeader(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return strings.Trim(trimmed, "[]"), true
	}
	return "", false
}

// classifySectionHeaderChange fires ForceBlast when a structural
// section itself is added or removed. Dep-table header changes
// (the section being newly created or deleted) also fire ForceBlast
// because the old/new contents aren't necessarily in the hunk.
func classifySectionHeaderChange(section string, out *CargoTomlChange) {
	if isStructuralSection(section) {
		out.ForceBlast = true
	}
}

// isStructuralSection returns true for sections whose presence or
// absence changes the crate's identity or build behavior, not its
// deps.
func isStructuralSection(s string) bool {
	switch s {
	case "package", "workspace", "lib", "bin", "features", "profile",
		"patch", "replace", "target":
		return true
	}
	// Nested variants like [profile.release], [workspace.metadata.*]
	if strings.HasPrefix(s, "profile.") ||
		strings.HasPrefix(s, "target.") ||
		strings.HasPrefix(s, "workspace.metadata") ||
		strings.HasPrefix(s, "patch.") ||
		strings.HasPrefix(s, "replace.") {
		return true
	}
	return false
}

// packageFieldKeys are [package] table fields that identify the
// crate itself, not its deps.
var packageFieldKeys = map[string]bool{
	"name":         true,
	"version":      true,
	"edition":      true,
	"rust-version": true,
	"authors":      true,
	"description":  true,
	"license":      true,
	"license-file": true,
	"repository":   true,
	"homepage":     true,
	"documentation": true,
	"readme":       true,
	"keywords":     true,
	"categories":   true,
	"build":        true,
	"links":        true,
	"workspace":    true,
	"resolver":     true,
	"publish":      true,
	"autobins":     true,
	"autoexamples": true,
	"autotests":    true,
	"autobenches":  true,
	"include":      true,
	"exclude":      true,
	"default-run":  true,
	"metadata":     true,
}

// classifyCargoLine extracts a dep key from a line when the context
// warrants (dep section, or ambiguous section with a dep-shaped
// line), and flags structural changes in [package] or [workspace].
func classifyCargoLine(line, section string, seen map[string]bool, out *CargoTomlChange) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	if i := strings.Index(trimmed, "#"); i >= 0 {
		trimmed = strings.TrimSpace(trimmed[:i])
	}
	if trimmed == "" {
		return
	}

	idx := strings.Index(trimmed, "=")
	if idx <= 0 {
		return
	}
	key := strings.TrimSpace(trimmed[:idx])

	// Package identity + workspace member list: always structural.
	if section == "package" || section == "workspace" {
		out.ForceBlast = true
		return
	}
	// [features] table changes can flip compiled code paths across
	// every consumer — treat as structural.
	if section == "features" {
		out.ForceBlast = true
		return
	}

	// Even in unknown section, a package-field key *probably* means
	// [package] was near — force blast to be safe.
	if packageFieldKeys[key] {
		out.ForceBlast = true
		return
	}

	// Dep sections (or ambiguous section but dep-shaped key).
	switch {
	case section == "dependencies",
		section == "dev-dependencies",
		section == "build-dependencies",
		strings.HasSuffix(section, ".dependencies"),
		strings.HasSuffix(section, ".dev-dependencies"),
		strings.HasSuffix(section, ".build-dependencies"),
		section == "": // hunk didn't include header — lean on key shape
		if looksLikeCrateName(key) {
			seen[key] = true
		}
	}
}

// looksLikeCrateName accepts crate-name-like keys: lowercase
// alphanumeric with dashes or underscores, not a version, not a pure
// numeric.
func looksLikeCrateName(s string) bool {
	if s == "" {
		return false
	}
	hasAlpha := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			hasAlpha = true
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return false
		}
	}
	return hasAlpha
}
