package opcm_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestDeploySuperchain2(t *testing.T) {
	t.Run("should not fail with current version of DeploySuperchain2 contract", func(t *testing.T) {
		// First we grab a test host
		host := createTestHost(t)

		// Then we deploy
		output, err := opcm.DeploySuperchain2(host, opcm.DeploySuperchain2Input{
			Guardian:                   common.BigToAddress(big.NewInt(1)),
			ProtocolVersionsOwner:      common.BigToAddress(big.NewInt(2)),
			SuperchainProxyAdminOwner:  common.BigToAddress(big.NewInt(3)),
			Paused:                     true,
			RecommendedProtocolVersion: params.ProtocolVersion{1},
			RequiredProtocolVersion:    params.ProtocolVersion{2},
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)
	})
}
