// Package glob provides the shared glob matcher used by the builder
// and the selector. Supports the subset of globs used in the check
// catalog's inputs/outputs patterns:
//
//   - prefix/**           any path with this prefix
//   - **/*.ext            any path with this extension (anywhere)
//   - prefix/**/*.ext     any path under prefix/ with this extension
//   - **/basename         any path whose basename matches
//   - simple * globs      via filepath.Match
//   - exact literal match
package glob

import (
	"path/filepath"
	"strings"
)

// Match reports whether path matches the supplied glob pattern.
func Match(pattern, path string) bool {
	// `prefix/**/*.ext` → under prefix AND has extension
	if i := strings.Index(pattern, "/**/"); i != -1 {
		prefix := pattern[:i]
		rest := pattern[i+len("/**/"):]
		if !(strings.HasPrefix(path, prefix+"/") || strings.Contains(path, "/"+prefix+"/")) {
			return false
		}
		return matchTail(rest, path)
	}
	// `**/<tail>` → match tail anywhere
	if strings.HasPrefix(pattern, "**/") {
		return matchTail(pattern[len("**/"):], path)
	}
	// `prefix/**` → any path with prefix
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(path, prefix) || strings.Contains(path, "/"+prefix)
	}
	// filepath.Match on the full path
	if matched, _ := filepath.Match(pattern, path); matched {
		return true
	}
	// Basename fallback for patterns like `*.go`
	if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
		return true
	}
	return path == pattern
}

// MatchAny reports whether path matches any of the supplied patterns.
func MatchAny(path string, patterns []string) bool {
	for _, p := range patterns {
		if Match(p, path) {
			return true
		}
	}
	return false
}

// matchTail reports whether path ends with a segment that matches
// `rest` (via filepath.Match, applied to the basename). For `*.ext`
// patterns, this amounts to an extension check.
func matchTail(rest, path string) bool {
	if matched, _ := filepath.Match(rest, filepath.Base(path)); matched {
		return true
	}
	// For patterns like `subdir/*.ext`, check if path ends in /subdir/...
	if i := strings.Index(rest, "/"); i != -1 {
		segment := rest[:i]
		tail := rest[i+1:]
		idx := 0
		for {
			j := strings.Index(path[idx:], "/"+segment+"/")
			if j == -1 {
				break
			}
			sub := path[idx+j+len(segment)+2:]
			if matched, _ := filepath.Match(tail, filepath.Base(sub)); matched {
				return true
			}
			idx += j + 1
		}
	}
	return false
}
