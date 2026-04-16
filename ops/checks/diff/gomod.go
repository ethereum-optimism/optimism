package diff

import (
	"sort"
	"strings"
)

// GoModChange summarizes a go.mod hunk set.
//
// AffectedModules are module paths that appear on the added or removed
// side of any hunk — i.e., modules whose version, presence, or
// indirect status changed. Phase 1 feeds these into the reverse-walk
// as synthetic changed nodes.
//
// ForceBlast is set when the diff contains structural changes that
// can't be resolved by module-level reverse-walk alone: a `module`
// line change (repo rename), a `go` or `toolchain` directive change
// (compiler version), or a parse failure that leaves affected modules
// uncertain. Callers should fall back to running everything in that case.
type GoModChange struct {
	AffectedModules []string
	ForceBlast      bool
}

// AnalyzeGoModChange parses the added/removed lines of a go.mod FileDiff
// and returns the affected module set plus a ForceBlast flag for
// structural changes. Works on line-diff content without needing the
// full pre-state of the file.
//
// Recognizes:
//   - Require block lines   "    github.com/foo v1.2.3"
//   - Single-line require   "require github.com/foo v1.2.3"
//   - Replace directive     "replace foo v1 => bar v2" (LHS module)
//   - Exclude directive     "exclude foo v1"
//   - Structural directives (module, go, toolchain) → ForceBlast
func AnalyzeGoModChange(d FileDiff) GoModChange {
	if !isGoModPath(d.Path) {
		return GoModChange{}
	}

	seen := make(map[string]bool)
	var change GoModChange

	for _, h := range d.Hunks {
		for _, line := range h.Added {
			classifyGoModLine(line, seen, &change)
		}
		for _, line := range h.Removed {
			classifyGoModLine(line, seen, &change)
		}
	}

	for m := range seen {
		change.AffectedModules = append(change.AffectedModules, m)
	}
	sort.Strings(change.AffectedModules)
	return change
}

// isGoModPath returns true for "go.mod" or any "*/go.mod" (nested
// modules, like in a workspace).
func isGoModPath(p string) bool {
	return p == "go.mod" || strings.HasSuffix(p, "/go.mod")
}

func classifyGoModLine(line string, seen map[string]bool, out *GoModChange) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	// Strip trailing comments ("// indirect", etc.)
	if i := strings.Index(trimmed, "//"); i >= 0 {
		trimmed = strings.TrimSpace(trimmed[:i])
	}
	if trimmed == "" || trimmed == "(" || trimmed == ")" {
		return
	}

	// Structural directives: a change here invalidates the graph's
	// assumptions about module identity or toolchain, so we force
	// blast-radius rather than trying to map to specific consumers.
	if strings.HasPrefix(trimmed, "module ") ||
		strings.HasPrefix(trimmed, "go ") ||
		strings.HasPrefix(trimmed, "toolchain ") {
		out.ForceBlast = true
		return
	}

	// Strip an optional leading directive keyword to normalize block-
	// and single-line forms. `replace` must come before the `=>` check
	// since a single-line replace carries the directive and the arrow.
	for _, prefix := range []string{"require ", "exclude ", "retract ", "replace "} {
		trimmed = strings.TrimPrefix(trimmed, prefix)
	}
	trimmed = strings.TrimSpace(trimmed)

	// Replace directive: "foo [v1] => bar v2". Both sides are module-
	// shaped; include the LHS module as affected (the "before").
	if arrow := strings.Index(trimmed, "=>"); arrow >= 0 {
		lhs := strings.TrimSpace(trimmed[:arrow])
		if m := firstModulePath(lhs); m != "" {
			if !seen[m] {
				seen[m] = true
			}
		}
		return
	}

	if m := firstModulePath(trimmed); m != "" {
		if !seen[m] {
			seen[m] = true
		}
	}
}

// firstModulePath returns the first module-shaped token in line, or ""
// if none. Rejects version strings and pure-path fragments.
func firstModulePath(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	candidate := fields[0]
	if !looksLikeModulePath(candidate) {
		return ""
	}
	return candidate
}

// looksLikeModulePath performs a cheap syntactic filter: module paths
// contain at least one slash OR a dot (for paths like "example.com").
// Rejects bare identifiers and version strings.
func looksLikeModulePath(s string) bool {
	if s == "" || strings.HasPrefix(s, "v") && looksLikeVersion(s) {
		return false
	}
	return strings.Contains(s, "/") || strings.Contains(s, ".")
}

func looksLikeVersion(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	// v<digit>… is the canonical form.
	return s[1] >= '0' && s[1] <= '9'
}
