package opcm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPermissionedGamePrestatePlaceholder(t *testing.T) {
	require.Equal(
		t,
		common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000dead"),
		PermissionedGamePrestatePlaceholder,
	)
	require.NotEqual(t, common.BytesToHash(PermissionedGameStartingAnchorRoot[:32]), PermissionedGamePrestatePlaceholder)
}
