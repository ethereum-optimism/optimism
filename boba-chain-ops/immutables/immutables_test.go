package immutables_test

import (
	"math/big"
	"testing"

	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/immutables"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestBuildOptimism(t *testing.T) {
	minimumWithdrawalAmountBig, _ := big.NewInt(0).SetString("8ac7230489e80000", 16)
	minimumWithdrawalAmount := (*hexutil.Big)(minimumWithdrawalAmountBig)

	config := immutables.ImmutableConfig{
		"L2StandardBridge": {
			"otherBridge": common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		"L2CrossDomainMessenger": {
			"otherMessenger": common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		"L2ERC721Bridge": {
			"otherBridge": common.HexToAddress("0x1234567890123456789012345678901234567890"),
			"messenger":   common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		"OptimismMintableERC721Factory": {
			"remoteChainId": big.NewInt(1),
			"bridge":        common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		"SequencerFeeVault": {
			"recipient":               common.HexToAddress("0x1234567890123456789012345678901234567890"),
			"minimumWithdrawalAmount": minimumWithdrawalAmount,
			"withdrawalNetwork":       uint8(0),
		},
		"L1FeeVault": {
			"recipient":               common.HexToAddress("0x1234567890123456789012345678901234567890"),
			"minimumWithdrawalAmount": minimumWithdrawalAmount,
			"withdrawalNetwork":       uint8(0),
		},
		"BaseFeeVault": {
			"recipient":               common.HexToAddress("0x1234567890123456789012345678901234567890"),
			"minimumWithdrawalAmount": minimumWithdrawalAmount,
			"withdrawalNetwork":       uint8(0),
		},
		"BobaL2": {
			"l2Bridge":  common.HexToAddress("0x1234567890123456789012345678901234567890"),
			"l1Token":   common.HexToAddress("0x0123456789012345678901234567890123456789"),
			"_name":     "BOBA Token",
			"_symbol":   "BOBA",
			"_decimals": uint8(18),
		},
	}

	results, err := immutables.BuildOptimism(config)
	require.NoError(t, err)
	require.NotNil(t, results)

	contracts := map[string]bool{
		"GasPriceOracle":                true,
		"L1Block":                       true,
		"L2CrossDomainMessenger":        true,
		"L2StandardBridge":              true,
		"L2ToL1MessagePasser":           true,
		"SequencerFeeVault":             true,
		"BaseFeeVault":                  true,
		"L1FeeVault":                    true,
		"OptimismMintableERC20Factory":  true,
		"DeployerWhitelist":             true,
		"LegacyMessagePasser":           true,
		"L1BlockNumber":                 true,
		"L2ERC721Bridge":                true,
		"OptimismMintableERC721Factory": true,
		"LegacyERC20ETH":                true,
		"EAS":                           true,
		"SchemaRegistry":                true,
		"BobaL2":                        true,
	}

	// Only the exact contracts that we care about are being
	// modified
	require.Equal(t, len(results), len(contracts))

	for name, bytecode := range results {
		// There is bytecode there
		require.NotEmpty(t, len(bytecode), 0)
		// It is in the set of contracts that we care about
		require.True(t, contracts[name])
	}

	// Duplicate results
	// The contract bytecode should be the same
	duplicateResults, err := immutables.BuildOptimism(config)
	require.NoError(t, err)
	require.Equal(t, results, duplicateResults)
}
