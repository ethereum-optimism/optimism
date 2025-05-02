package addresses

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSuperchainRoles_CheckNoZeroAddresses(t *testing.T) {
	t.Run("no zero addresses", func(t *testing.T) {
		roles := SuperchainRoles{
			SuperchainProxyAdminOwner: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			SuperchainGuardian:        common.HexToAddress("0x2222222222222222222222222222222222222222"),
			ProtocolVersionsOwner:     common.HexToAddress("0x3333333333333333333333333333333333333333"),
		}

		err := roles.CheckNoZeroAddresses()
		require.NoError(t, err)
	})

	t.Run("detects zero address", func(t *testing.T) {
		roles := SuperchainRoles{
			SuperchainProxyAdminOwner: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			ProtocolVersionsOwner:     common.HexToAddress("0x3333333333333333333333333333333333333333"),
		}

		require.Equal(t, roles.SuperchainGuardian, common.HexToAddress("0x0000000000000000000000000000000000000000"))
		err := roles.CheckNoZeroAddresses()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrSuperchainRoleZeroAddress)
		require.Contains(t, err.Error(), "SuperchainGuardian")
	})
}
