package diff

import "testing"

func TestClassify_CommentOnlyIsTextOnly(t *testing.T) {
	// NatSpec typo fix — pure comment edit.
	fd := FileDiff{
		Path: "packages/contracts-bedrock/src/L1/L1StandardBridge.sol",
		Hunks: []Hunk{{
			Removed: []string{"/// @notice Transfering ETH and tokens."},
			Added:   []string{"/// @notice Transferring ETH and tokens."},
		}},
	}
	if got := Classify(fd); got != ImpactTextOnly {
		t.Fatalf("comment-only diff: want ImpactTextOnly, got %v", got)
	}
}

func TestClassify_CodeChangeIsSemantic(t *testing.T) {
	fd := FileDiff{
		Path: "src/Foo.sol",
		Hunks: []Hunk{{
			Removed: []string{"uint256 x = 1;"},
			Added:   []string{"uint256 x = 2;"},
		}},
	}
	if got := Classify(fd); got != ImpactSemantic {
		t.Fatalf("code change: want ImpactSemantic, got %v", got)
	}
}

func TestClassify_MixedHunksIsSemantic(t *testing.T) {
	// One text-only hunk + one semantic hunk → semantic overall.
	fd := FileDiff{
		Path: "src/Foo.sol",
		Hunks: []Hunk{
			{Removed: []string{"// old comment"}, Added: []string{"// new comment"}},
			{Removed: []string{"uint256 x = 1;"}, Added: []string{"uint256 x = 2;"}},
		},
	}
	if got := Classify(fd); got != ImpactSemantic {
		t.Fatalf("mixed: want ImpactSemantic, got %v", got)
	}
}

func TestClassify_TrailingCommentEditOnSameStatementIsTextOnly(t *testing.T) {
	// Code line is unchanged; only the trailing // comment differs.
	fd := FileDiff{
		Path: "src/Foo.sol",
		Hunks: []Hunk{{
			Removed: []string{"uint256 x = 1; // was wrong"},
			Added:   []string{"uint256 x = 1; // fixed typo"},
		}},
	}
	if got := Classify(fd); got != ImpactTextOnly {
		t.Fatalf("trailing-comment edit: want ImpactTextOnly, got %v", got)
	}
}

func TestClassify_UnsupportedLangIsSemantic(t *testing.T) {
	fd := FileDiff{
		Path: "README.md",
		Hunks: []Hunk{{
			Removed: []string{"old line"},
			Added:   []string{"new line"},
		}},
	}
	if got := Classify(fd); got != ImpactSemantic {
		t.Fatalf("unsupported extension: want ImpactSemantic, got %v", got)
	}
}

func TestClassify_NewFileIsSemantic(t *testing.T) {
	fd := FileDiff{
		Path:  "src/New.sol",
		IsNew: true,
		Hunks: []Hunk{{Added: []string{"// just a comment"}}},
	}
	if got := Classify(fd); got != ImpactSemantic {
		t.Fatalf("new file: want ImpactSemantic, got %v", got)
	}
}

func TestClassify_EmptyHunkIsSemantic(t *testing.T) {
	// Test fixtures sometimes describe hunks by line numbers only
	// with no populated Added/Removed. Don't mistake that for
	// text-only.
	fd := FileDiff{
		Path:  "src/Foo.sol",
		Hunks: []Hunk{{NewStart: 10, NewCount: 3}},
	}
	if got := Classify(fd); got != ImpactSemantic {
		t.Fatalf("empty hunk: want ImpactSemantic, got %v", got)
	}
}

func TestClassify_NoHunksIsSemantic(t *testing.T) {
	// CI-history replay events and `git diff --name-only` style
	// inputs populate Path but no Hunks at all. Without hunk
	// content we can't prove text-only, so classify semantic.
	fd := FileDiff{Path: "src/Foo.sol"}
	if got := Classify(fd); got != ImpactSemantic {
		t.Fatalf("no hunks: want ImpactSemantic, got %v", got)
	}
}

func TestClassify_StringLiteralNotComment(t *testing.T) {
	// A // inside a string literal shouldn't be mistaken for a
	// comment marker. Same code both sides → text-only.
	fd := FileDiff{
		Path: "src/Foo.sol",
		Hunks: []Hunk{{
			Removed: []string{`string constant URL = "https://example.com"; // old doc`},
			Added:   []string{`string constant URL = "https://example.com"; // new doc`},
		}},
	}
	if got := Classify(fd); got != ImpactTextOnly {
		t.Fatalf("string-literal with trailing-comment edit: want ImpactTextOnly, got %v", got)
	}
}
