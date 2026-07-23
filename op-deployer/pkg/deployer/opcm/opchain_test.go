package opcm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDefaultStartingAnchorProposal(t *testing.T) {
	first := DefaultStartingAnchorProposal()
	second := DefaultStartingAnchorProposal()

	require.Equal(t, DefaultStartingAnchorRoot.Root, first.Root)
	require.Equal(t, DefaultStartingAnchorRoot.Root, second.Root)
	require.Zero(t, first.L2SequenceNumber.Sign())
	require.Zero(t, second.L2SequenceNumber.Sign())
	require.NotSame(t, first.L2SequenceNumber, second.L2SequenceNumber)

	first.L2SequenceNumber.SetUint64(1)
	require.Zero(t, second.L2SequenceNumber.Sign())
}

func TestPermissionedCannonFallbackPrestatePlaceholder(t *testing.T) {
	require.Equal(
		t,
		common.HexToHash("0xdead000000000000000000000000000000000000000000000000000000000000"),
		PermissionedCannonFallbackPrestatePlaceholder,
	)
	require.Equal(
		t,
		common.BytesToHash(PermissionedGameStartingAnchorRoot[:common.HashLength]),
		PermissionedCannonFallbackPrestatePlaceholder,
	)
}
