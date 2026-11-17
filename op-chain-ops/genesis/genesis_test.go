package genesis

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestNewL2GenesisRefusesOPMainnetChainID(t *testing.T) {
	config := testDeployConfig(t)
	config.L2ChainID = opparams.OPMainnetChainID
	_, err := NewL2Genesis(config, &eth.BlockRef{Time: 1_700_000_000})
	require.ErrorIs(t, err, ErrInvalidDeployConfig)
	require.ErrorContains(t, err, "reserved for OP Mainnet")
}

func testDeployConfig(t *testing.T) *DeployConfig {
	data, err := os.ReadFile("testdata/test-deploy-config-full.json")
	require.NoError(t, err)
	config := new(DeployConfig)
	require.NoError(t, json.Unmarshal(data, config))
	return config
}
