package diff

import (
	"path/filepath"
	"strings"
)

// Impact describes what kind of change a FileDiff represents. The
// selector pairs this with each check's concern field to decide
// whether a content-only edit (a typo in a comment, whitespace
// reflow) should trigger that check.
//
//   - ImpactSemantic: the diff changes something that could affect
//     bytecode, ABI, or runtime behavior. Or: the classifier can't
//     prove the change is inert (unknown language, block comment
//     spanning hunks, renamed file, etc). Conservative default.
//   - ImpactTextOnly: every hunk is a pure comment / whitespace edit.
//     Semantic-concerned checks (tests, snapshots, AST validators)
//     can safely skip this file. Text-concerned checks (lint, fmt,
//     semgrep) still fire on it.
//
// We only name the ImpactTextOnly path where we can prove the change
// is inert; everything else is ImpactSemantic so false positives
// never hide real work.
type Impact int

const (
	ImpactSemantic Impact = iota
	ImpactTextOnly
)

// Classify returns ImpactTextOnly if every hunk is a pure comment or
// whitespace edit, and ImpactSemantic otherwise.
//
// The classifier is intentionally conservative: any construct it
// can't confidently strip (multi-line /* */ blocks that cross hunk
// boundaries, string literals that look like comments, unsupported
// languages, new/deleted/renamed files) falls through to
// ImpactSemantic. A false-negative only costs the existing
// structural over-selection; a false-positive would hide a real
// change and break selection correctness.
//
// Supported languages: Solidity (.sol), Go (.go), Rust (.rs). All
// three share // line-comment and /* */ block-comment syntax, plus
// doc-comment variants (///, //!, /** */) that collapse to the same
// stripping rule. Other extensions → ImpactSemantic.
func Classify(fd FileDiff) Impact {
	if fd.IsNew || fd.IsDelete || fd.IsRename {
		return ImpactSemantic
	}
	// Text-only file formats have no semantic dimension — every
	// edit is by definition "text". Catches .md READMEs inside
	// code subtrees that shouldn't trigger test runs just because
	// they happen to live under op-acceptance-tests/** or rust/**.
	if isTextOnlyLang(fd.Path) {
		return ImpactTextOnly
	}
	if !supportedLang(fd.Path) {
		return ImpactSemantic
	}
	// Some upstreams (CI-history replay events from CircleCI,
	// `git diff --name-only`, etc.) give us file paths with zero
	// hunks. We can't prove the edit is inert, so classify as
	// semantic — the safe default. Same conservatism applies when
	// any individual hunk has no populated Added/Removed content.
	if len(fd.Hunks) == 0 {
		return ImpactSemantic
	}
	for _, h := range fd.Hunks {
		if !hunkIsTextOnly(h) {
			return ImpactSemantic
		}
	}
	return ImpactTextOnly
}

// supportedLang reports whether this file's extension is one we
// have a comment-stripping rule for. Used to decide whether hunk-
// level text-only classification can even be attempted.
func supportedLang(path string) bool {
	switch filepath.Ext(path) {
	case ".sol", ".go", ".rs":
		return true
	}
	return false
}

// isTextOnlyLang reports whether a file extension has no semantic
// dimension — every edit is by definition textual. Used to
// unconditionally classify .md, .txt, etc. as ImpactTextOnly
// regardless of hunk content. Docs-only edits then only trigger
// concern=text checks (semgrep, lint, docs-build).
func isTextOnlyLang(path string) bool {
	switch filepath.Ext(path) {
	case ".md", ".markdown", ".txt", ".rst":
		return true
	}
	// Top-level AGENTS.md / NOTICE / LICENSE-style files without
	// an extension are also docs.
	switch filepath.Base(path) {
	case "AGENTS", "AGENTS.md", "NOTICE", "LICENSE", "LICENSE.md":
		return true
	}
	return false
}

// hunkIsTextOnly reports whether a hunk's removed and added lines
// collapse to identical non-comment content — i.e. the semantic
// source is unchanged, only comments or whitespace were edited.
//
// A hunk with no Added/Removed content populated (some upstream
// parsers or test fixtures describe hunks by line numbers only)
// can't be classified, and we return false: "don't know" is treated
// as semantic so no false positives hide real changes.
func hunkIsTextOnly(h Hunk) bool {
	if len(h.Removed) == 0 && len(h.Added) == 0 {
		return false
	}
	return normalizeLines(h.Removed) == normalizeLines(h.Added)
}

// normalizeLines joins lines after stripping comments and collapsing
// whitespace. Empty result is legal (pure comment/blank diff).
func normalizeLines(lines []string) string {
	var parts []string
	for _, l := range lines {
		stripped := stripLineComments(l)
		stripped = strings.Join(strings.Fields(stripped), " ")
		if stripped != "" {
			parts = append(parts, stripped)
		}
	}
	return strings.Join(parts, "\n")
}

// stripLineComments removes //... line comments and /* ... */ block
// comments that are entirely contained within the line. Block comments
// that span lines are not handled — the caller falls back to
// ImpactUnknown if any hunk's normalized text mismatches, which covers
// the "partial block comment" failure mode.
//
// Simple string-literal avoidance: we only strip // if it isn't inside
// a "..." or '...' pair on the same line. This rules out most false
// positives (e.g. a literal `"//"` in a URL string) without needing a
// full tokenizer. A purely-literal match still falls into the default-
// conservative ImpactUnknown path because the line content differs.
func stripLineComments(line string) string {
	// Strip inline /* ... */ first (repeatedly).
	for {
		i := strings.Index(line, "/*")
		if i < 0 {
			break
		}
		j := strings.Index(line[i+2:], "*/")
		if j < 0 {
			// Unclosed block comment — treat line as fully in comment
			// up to the end; normalizing to empty is fine here because
			// the _corresponding_ line in the other side has to match.
			return line[:i]
		}
		line = line[:i] + line[i+2+j+2:]
	}
	// Strip // up to end-of-line, ignoring // inside string literals.
	if idx := findLineCommentStart(line); idx >= 0 {
		line = line[:idx]
	}
	return line
}

// findLineCommentStart returns the byte index of // that begins a
// line comment, or -1 if none. Skips over // inside "..." or '...'
// single-line string literals.
func findLineCommentStart(line string) int {
	inStr := byte(0)
	for i := 0; i < len(line); i++ {
		c := line[i]
		if inStr != 0 {
			if c == '\\' && i+1 < len(line) {
				i++
				continue
			}
			if c == inStr {
				inStr = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			inStr = c
			continue
		}
		if c == '/' && i+1 < len(line) && line[i+1] == '/' {
			return i
		}
	}
	return -1
}
