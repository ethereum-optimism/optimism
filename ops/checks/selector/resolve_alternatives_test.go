package selector

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

// altCatalog builds a tiny catalog with two alternatives sharing an
// assertion and one unrelated unconditional check. Matches the shape
// of forge-test vs forge-test-dev + an untagged lint check.
func altCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	cat, err := catalog.Parse([]byte(`
check_types:
  - id: forge-test
    name: forge-test
    kind: test
    language: solidity
    command: forge test
    avg_duration: 3600
    assertion: contracts-tests-pass
    tags: [profile:prod]
  - id: forge-test-dev
    name: forge-test-dev
    kind: test
    language: solidity
    command: forge test --dev
    avg_duration: 900
    assertion: contracts-tests-pass
    tags: [profile:dev]
  - id: unrelated-lint
    name: unrelated-lint
    kind: lint
    language: solidity
    command: solhint
    avg_duration: 5
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return cat
}

func cands(ids ...string) []Candidate {
	out := make([]Candidate, 0, len(ids))
	for _, id := range ids {
		out = append(out, Candidate{CheckID: id, Signal: 1.0})
	}
	return out
}

func TestApplyStagePolicy_DevStagePicksDev(t *testing.T) {
	cat := altCatalog(t)
	stage := policy.StageConfig{Name: "commit", AcceptTags: []string{"profile:dev"}}

	got := applyStagePolicy(cands("forge-test", "forge-test-dev", "unrelated-lint"), cat, stage)
	if len(got) != 2 {
		t.Fatalf("want 2 survivors, got %d: %+v", len(got), got)
	}
	if ids := idsOf(got); !contains(ids, "forge-test-dev") || !contains(ids, "unrelated-lint") || contains(ids, "forge-test") {
		t.Fatalf("want forge-test-dev + unrelated-lint, got %v", ids)
	}
}

func TestApplyStagePolicy_PrStagePicksProdByPreference(t *testing.T) {
	cat := altCatalog(t)
	stage := policy.StageConfig{
		Name:       "pr",
		AcceptTags: []string{"profile:prod", "profile:dev"},
	}

	got := applyStagePolicy(cands("forge-test", "forge-test-dev", "unrelated-lint"), cat, stage)
	if len(got) != 2 {
		t.Fatalf("want 2 survivors, got %d: %+v", len(got), got)
	}
	if ids := idsOf(got); !contains(ids, "forge-test") || contains(ids, "forge-test-dev") {
		t.Fatalf("PR stage should keep forge-test and drop forge-test-dev, got %v", ids)
	}
}

func TestApplyStagePolicy_NoAcceptTagsAcceptsEverything(t *testing.T) {
	cat := altCatalog(t)
	stage := policy.StageConfig{Name: "open"}

	got := applyStagePolicy(cands("forge-test", "forge-test-dev", "unrelated-lint"), cat, stage)
	if len(got) != 2 {
		t.Fatalf("want 2 (one per assertion group collapses, lint untouched), got %d: %+v", len(got), got)
	}
	// Collapse tiebreaker: no preference distinguishes them → cheaper
	// AvgDuration wins → forge-test-dev (900 < 3600).
	if ids := idsOf(got); !contains(ids, "forge-test-dev") || contains(ids, "forge-test") {
		t.Fatalf("empty accept_tags should collapse by cost, got %v", ids)
	}
}

func TestApplyStagePolicy_TagMismatchDropsCheck(t *testing.T) {
	cat := altCatalog(t)
	// A stage that accepts only profile:dev should drop a prod-only
	// check that isn't in an assertion group together with a dev
	// alternative — but here forge-test IS in a group with forge-test-dev,
	// so we cover the "lone prod-tagged survivor gets dropped" case by
	// feeding only forge-test.
	stage := policy.StageConfig{Name: "commit", AcceptTags: []string{"profile:dev"}}

	got := applyStagePolicy(cands("forge-test"), cat, stage)
	if len(got) != 0 {
		t.Fatalf("lone prod-tagged check at dev stage should be dropped, got %+v", got)
	}
}

func TestApplyStagePolicy_UntaggedSurvivesEveryStage(t *testing.T) {
	cat := altCatalog(t)
	for _, tags := range [][]string{nil, {"profile:dev"}, {"profile:prod"}, {"profile:prod", "profile:dev"}} {
		stage := policy.StageConfig{Name: "x", AcceptTags: tags}
		got := applyStagePolicy(cands("unrelated-lint"), cat, stage)
		if len(got) != 1 {
			t.Fatalf("untagged check dropped at stage %v: %+v", tags, got)
		}
	}
}

func TestStageConfig_AcceptsTags(t *testing.T) {
	cases := []struct {
		name       string
		acceptTags []string
		checkTags  []string
		want       bool
	}{
		{"empty accept accepts everything", nil, []string{"profile:prod"}, true},
		{"empty check tags always accepted", []string{"profile:dev"}, nil, true},
		{"overlap accepts", []string{"profile:dev"}, []string{"profile:dev"}, true},
		{"no overlap rejects", []string{"profile:dev"}, []string{"profile:prod"}, false},
		{"any overlap accepts", []string{"profile:prod", "profile:dev"}, []string{"profile:dev"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := policy.StageConfig{AcceptTags: tc.acceptTags}
			if got := s.AcceptsTags(tc.checkTags); got != tc.want {
				t.Fatalf("AcceptsTags(%v) with accept_tags=%v: got %v, want %v",
					tc.checkTags, tc.acceptTags, got, tc.want)
			}
		})
	}
}

func idsOf(cs []Candidate) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.CheckID
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
