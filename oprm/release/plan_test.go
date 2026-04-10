package release

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderedComponentIDs(t *testing.T) {
	ordered := OrderedComponentIDs([]string{"op-node", "op-geth", "op-batcher", "op-node"})
	require.Equal(t, []string{"op-geth", "op-batcher", "op-node"}, ordered)
}

func TestProposalForComponentDefaultsPatchForChangedComponents(t *testing.T) {
	proposal, err := ProposalForComponent("v1.2.3", "v1.2.4-rc.1", "", true, "", "")
	require.NoError(t, err)
	require.Equal(t, "patch", proposal.Bump)
	require.Equal(t, "v1.2.4", proposal.TargetRelease)
	require.Equal(t, "v1.2.4-rc.2", proposal.Proposed)
}

func TestProposalForComponentLeavesUnchangedComponentEmpty(t *testing.T) {
	proposal, err := ProposalForComponent("v1.2.3", "v1.2.4-rc.1", "", false, "", "")
	require.NoError(t, err)
	require.Empty(t, proposal.Bump)
	require.Empty(t, proposal.Proposed)
}

func TestProposalForComponentSupportsManualTarget(t *testing.T) {
	proposal, err := ProposalForComponent("v1.2.3", "v1.2.4-rc.1", "", false, "", "v1.9.0")
	require.NoError(t, err)
	require.Equal(t, "manual", proposal.Bump)
	require.Equal(t, "v1.9.0", proposal.TargetRelease)
	require.Equal(t, "v1.9.0-rc.1", proposal.Proposed)
	require.True(t, proposal.ManualOverride)
}

func TestProposalForComponentResumesDraftRC(t *testing.T) {
	proposal, err := ProposalForComponent("v1.16.11", "v1.16.12-rc.1", "v1.16.12-rc.1", true, "", "")
	require.NoError(t, err)
	require.True(t, proposal.ResumeDraft)
	require.Equal(t, "patch", proposal.Bump)
	require.Equal(t, "v1.16.12", proposal.TargetRelease)
	require.Equal(t, "v1.16.12-rc.1", proposal.Proposed)
}

func TestProposalForComponentBootstrapsFirstReleaseWhenNoHistoryExists(t *testing.T) {
	proposal, err := ProposalForComponent("", "", "", true, "", "")
	require.NoError(t, err)
	require.Equal(t, "manual", proposal.Bump)
	require.Equal(t, "v0.0.1", proposal.TargetRelease)
	require.Equal(t, "v0.0.1-rc.1", proposal.Proposed)
	require.False(t, proposal.ManualOverride)
}

func TestInferBumpKind(t *testing.T) {
	require.Equal(t, BumpPatch, InferBumpKind("v1.2.3", "v1.2.4"))
	require.Equal(t, BumpMinor, InferBumpKind("v1.2.3", "v1.3.0"))
	require.Equal(t, BumpMajor, InferBumpKind("v1.2.3", "v2.0.0"))
}
