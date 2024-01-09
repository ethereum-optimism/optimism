package genesis

import (
	"math/big"
	"testing"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/predeploys"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/ether"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/immutables"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/state"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/stretchr/testify/require"
)

func TestWipePredeployStorage(t *testing.T) {
	g := &types.Genesis{
		Config: &chain.Config{
			ChainID: big.NewInt(2888),
		},
		Alloc: types.GenesisAlloc{},
	}

	code := []byte{1, 2, 3}
	storeVal := common.Hash{31: 0xff}
	nonce := 100

	for _, addr := range predeploys.Predeploys {
		a := *addr
		g.Alloc[a] = types.GenesisAccount{
			Code: code,
			Storage: map[common.Hash]common.Hash{
				storeVal:                                storeVal,
				ether.BobaLegacyProxyOwnerSlot:          {31: 0xff},
				ether.BobaLegacyProxyImplementationSlot: {31: 0xff},
			},
			Nonce: uint64(nonce),
		}
	}

	err := WipePredeployStorage(g)
	require.NoError(t, err)

	for _, addr := range predeploys.Predeploys {
		if FrozenStoragePredeploys[*addr] {
			expected := types.GenesisAccount{
				Code: code,
				Storage: map[common.Hash]common.Hash{
					storeVal:                                storeVal,
					ether.BobaLegacyProxyOwnerSlot:          {31: 0xff},
					ether.BobaLegacyProxyImplementationSlot: {31: 0xff},
				},
				Nonce: uint64(nonce),
			}
			require.Equal(t, expected, g.Alloc[*addr])
			continue
		}
		expected := types.GenesisAccount{
			Code:    code,
			Storage: map[common.Hash]common.Hash{},
			Nonce:   uint64(nonce),
		}
		require.Equal(t, expected, g.Alloc[*addr])
	}
}

func TestSetImplementations(t *testing.T) {
	g := &types.Genesis{
		Config: &chain.Config{
			ChainID: big.NewInt(2888),
		},
		Alloc: types.GenesisAlloc{},
	}

	immutables := immutables.ImmutableConfig{
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
			"minimumWithdrawalAmount": (*hexutil.Big)(big.NewInt(100)),
			"withdrawalNetwork":       uint8(0),
		},
		"L1FeeVault": {
			"recipient":               common.HexToAddress("0x1234567890123456789012345678901234567890"),
			"minimumWithdrawalAmount": (*hexutil.Big)(big.NewInt(100)),
			"withdrawalNetwork":       uint8(0),
		},
		"BaseFeeVault": {
			"recipient":               common.HexToAddress("0x1234567890123456789012345678901234567890"),
			"minimumWithdrawalAmount": (*hexutil.Big)(big.NewInt(100)),
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
	storage := make(state.StorageConfig)
	storage["L2ToL1MessagePasser"] = state.StorageValues{
		"msgNonce": 0,
	}
	storage["L2CrossDomainMessenger"] = state.StorageValues{
		"_initialized":     1,
		"_initializing":    false,
		"xDomainMsgSender": "0x000000000000000000000000000000000000dEaD",
		"msgNonce":         0,
	}
	storage["L1Block"] = state.StorageValues{
		"number":         common.Big1,
		"timestamp":      0,
		"basefee":        common.Big1,
		"hash":           common.Hash{1},
		"sequenceNumber": 0,
		"batcherHash":    common.Hash{1},
		"l1FeeOverhead":  0,
		"l1FeeScalar":    0,
	}
	storage["LegacyERC20ETH"] = state.StorageValues{
		"_name":   "Ether",
		"_symbol": "ETH",
	}
	storage["WETH9"] = state.StorageValues{
		"name":     "Wrapped Ether",
		"symbol":   "WETH",
		"decimals": 18,
	}
	storage["ProxyAdmin"] = state.StorageValues{
		"_owner": common.Address{1},
	}
	storage["ProxyAdmin"] = state.StorageValues{
		"_owner": common.Address{1},
	}
	storage["BobaL2"] = state.StorageValues{
		"l2Bridge":  common.HexToAddress("0x1234567890123456789012345678901234567890"),
		"l1Token":   common.HexToAddress("0x0123456789012345678901234567890123456789"),
		"_name":     "BOBA Token",
		"_symbol":   "BOBA",
		"_decimals": uint8(18),
	}

	err := SetImplementations(g, storage, immutables)
	require.NoError(t, err)

	for name, address := range predeploys.Predeploys {
		if FrozenStoragePredeploys[*address] {
			continue
		}
		if *address == predeploys.LegacyERC20ETHAddr {
			continue
		}

		_, ok := g.Alloc[*address]
		require.True(t, ok, "predeploy %s not found in genesis", name)
		require.NotEqual(t, common.Hash{}, g.Alloc[*address].Storage[ImplementationSlot])
		codeAddr, err := AddressToCodeNamespace(*address)
		require.NoError(t, err)
		require.NotEqual(t, common.Hash{}, g.Alloc[codeAddr].Code)
	}
}
