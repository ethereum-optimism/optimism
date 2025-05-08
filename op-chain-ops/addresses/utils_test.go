package addresses

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCheckNoZeroAddresses(t *testing.T) {
	t.Run("no zero addresses", func(t *testing.T) {
		roles := SuperchainRoles{
			SuperchainProxyAdminOwner: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			SuperchainGuardian:        common.HexToAddress("0x2222222222222222222222222222222222222222"),
			ProtocolVersionsOwner:     common.HexToAddress("0x3333333333333333333333333333333333333333"),
		}

		err := CheckNoZeroAddresses(roles)
		require.NoError(t, err)
	})

	t.Run("detects zero address", func(t *testing.T) {
		roles := SuperchainRoles{
			SuperchainProxyAdminOwner: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			ProtocolVersionsOwner:     common.HexToAddress("0x3333333333333333333333333333333333333333"),
		}

		require.Equal(t, roles.SuperchainGuardian, common.HexToAddress("0x0000000000000000000000000000000000000000"))
		err := CheckNoZeroAddresses(roles)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrZeroAddress)
		require.Contains(t, err.Error(), "SuperchainGuardian")
	})

	t.Run("error for non-address fields", func(t *testing.T) {
		roles := struct {
			SuperchainProxyAdminOwner common.Address
			ProtocolVersionsOwner     common.Address
			chainId                   uint64
		}{
			SuperchainProxyAdminOwner: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			ProtocolVersionsOwner:     common.HexToAddress("0x3333333333333333333333333333333333333333"),
			chainId:                   1,
		}

		err := CheckNoZeroAddresses(roles)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotAddressType)
		require.Contains(t, err.Error(), "chainId")
	})

	t.Run("struct pointer works", func(t *testing.T) {
		roles := SuperchainRoles{
			SuperchainProxyAdminOwner: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			SuperchainGuardian:        common.HexToAddress("0x2222222222222222222222222222222222222222"),
			ProtocolVersionsOwner:     common.HexToAddress("0x3333333333333333333333333333333333333333"),
		}

		err := CheckNoZeroAddresses(&roles)
		require.NoError(t, err)
	})

	t.Run("nil struct pointer fails", func(t *testing.T) {
		var roles *SuperchainRoles = nil

		err := CheckNoZeroAddresses(roles)
		require.Error(t, err)
		require.Contains(t, err.Error(), "nil pointer provided")
	})
}
