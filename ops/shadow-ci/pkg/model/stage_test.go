package model

import "testing"

func TestDetermineStage_PR(t *testing.T) {
	tests := []struct {
		branch string
	}{
		{"feat/my-feature"},
		{"fix/bug-123"},
		{"main"},
		{""},
	}
	for _, tt := range tests {
		got := DetermineStage(tt.branch, false)
		if got != StagePR {
			t.Errorf("DetermineStage(%q, false) = %q, want %q", tt.branch, got, StagePR)
		}
	}
}

func TestDetermineStage_MergeQueue(t *testing.T) {
	tests := []string{
		"gh-readonly-queue/develop/pr-123-abc123",
		"gh-readonly-queue/main/pr-456-def456",
	}
	for _, branch := range tests {
		got := DetermineStage(branch, false)
		if got != StageMergeQueue {
			t.Errorf("DetermineStage(%q, false) = %q, want %q", branch, got, StageMergeQueue)
		}
	}
}

func TestDetermineStage_PostMerge(t *testing.T) {
	got := DetermineStage("develop", false)
	if got != StagePostMerge {
		t.Errorf("DetermineStage(develop, false) = %q, want %q", got, StagePostMerge)
	}
}

func TestDetermineStage_Nightly(t *testing.T) {
	// isSchedule overrides branch.
	for _, branch := range []string{"develop", "feat/x", ""} {
		got := DetermineStage(branch, true)
		if got != StageNightly {
			t.Errorf("DetermineStage(%q, true) = %q, want %q", branch, got, StageNightly)
		}
	}
}

func TestStageIndex(t *testing.T) {
	if StageIndex(StagePR) >= StageIndex(StageMergeQueue) {
		t.Error("PR should be before merge_queue")
	}
	if StageIndex(StageMergeQueue) >= StageIndex(StagePostMerge) {
		t.Error("merge_queue should be before post_merge")
	}
	if StageIndex(StagePostMerge) >= StageIndex(StageNightly) {
		t.Error("post_merge should be before nightly")
	}
	if StageIndex(Stage("unknown")) != 0 {
		t.Error("unknown stage should default to 0")
	}
}

func TestShouldRunAtStage(t *testing.T) {
	tests := []struct {
		placedAt Stage
		current  Stage
		want     bool
	}{
		{StagePR, StagePR, true},
		{StagePR, StageMergeQueue, true},
		{StagePR, StagePostMerge, true},
		{StagePR, StageNightly, true},
		{StageMergeQueue, StagePR, false},
		{StageMergeQueue, StageMergeQueue, true},
		{StageMergeQueue, StagePostMerge, true},
		{StageMergeQueue, StageNightly, true},
		{StagePostMerge, StagePR, false},
		{StagePostMerge, StageMergeQueue, false},
		{StagePostMerge, StagePostMerge, true},
		{StagePostMerge, StageNightly, true},
		{StageNightly, StagePR, false},
		{StageNightly, StageMergeQueue, false},
		{StageNightly, StagePostMerge, false},
		{StageNightly, StageNightly, true},
	}
	for _, tt := range tests {
		got := ShouldRunAtStage(tt.placedAt, tt.current)
		if got != tt.want {
			t.Errorf("ShouldRunAtStage(%q, %q) = %v, want %v", tt.placedAt, tt.current, got, tt.want)
		}
	}
}
