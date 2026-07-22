package opcm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

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
