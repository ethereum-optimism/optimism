package e2eutils

import (
	"encoding/hex"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
)

func TestWriteDefaultJWT(t *testing.T) {
	jwtPath := WriteDefaultJWT(t)
	data, err := os.ReadFile(jwtPath)
	require.NoError(t, err)
	require.Equal(t, "0x"+hex.EncodeToString(testingJWTSecret[:]), string(data))
}

func TestSetup(t *testing.T) {
	tp := &TestParams{
		MaxSequencerDrift:   40,
		SequencerWindowSize: 120,
		ChannelTimeout:      120,
		L1BlockTime:         15,
		AllocType:           config.DefaultAllocType,
	}
	dp := MakeDeployParams(t, tp)
	alloc := &AllocParams{PrefundTestUsers: true}
	sd := Setup(t, dp, alloc)
	require.Contains(t, sd.L1Cfg.Alloc, dp.Addresses.Alice)
	require.Equal(t, sd.L1Cfg.Alloc[dp.Addresses.Alice].Balance, Ether(1e12))

	require.Contains(t, sd.L2Cfg.Alloc, dp.Addresses.Alice)
	require.Equal(t, sd.L2Cfg.Alloc[dp.Addresses.Alice].Balance, Ether(1e12))

	expAllocs := config.L1Deployments(tp.AllocType)
	require.Contains(t, sd.L1Cfg.Alloc, expAllocs.AddressManager)
	require.Contains(t, sd.L2Cfg.Alloc, predeploys.L1BlockAddr)
}

func TestSetupL2AllocFrozenPreForkState(t *testing.T) {
	tp := &TestParams{
		MaxSequencerDrift:   40,
		SequencerWindowSize: 120,
		ChannelTimeout:      120,
		L1BlockTime:         15,
		AllocType:           config.DefaultAllocType,
	}
	dp := MakeDeployParams(t, tp)
	proxyAddr := predeploys.L2CrossDomainMessengerAddr
	defaultSetup := Setup(t, dp, &AllocParams{})
	generatedProxy, ok := defaultSetup.L2Cfg.Alloc[proxyAddr]
	require.True(t, ok)
	generatedImplementationHash, ok := generatedProxy.Storage[genesis.ImplementationSlot]
	require.True(t, ok)
	generatedImplementationAddr := common.BytesToAddress(generatedImplementationHash.Bytes())
	generatedImplementation, ok := defaultSetup.L2Cfg.Alloc[generatedImplementationAddr]
	require.True(t, ok)
	require.NotEmpty(t, generatedImplementation.Code)

	t.Run("sparse overlay retains the generated implementation", func(t *testing.T) {
		unrelatedAddr := common.HexToAddress("0x1234")
		unrelatedAccount := types.Account{
			Nonce:   3,
			Balance: big.NewInt(44),
			Code:    []byte{0x60, 0x01},
			Storage: map[common.Hash]common.Hash{
				common.HexToHash("0x02"): common.HexToHash("0x03"),
			},
		}

		setup := Setup(t, dp, &AllocParams{
			L2Alloc: types.GenesisAlloc{unrelatedAddr: unrelatedAccount},
		})
		sparseProxy := setup.L2Cfg.Alloc[proxyAddr]
		implementation, ok := sparseProxy.Storage[genesis.ImplementationSlot]
		require.True(t, ok)
		require.NotEqual(t, common.Hash{}, implementation)
		require.Equal(t, generatedProxy.Storage[genesis.ImplementationSlot], implementation)
		sparseImplementation, ok := setup.L2Cfg.Alloc[generatedImplementationAddr]
		require.True(t, ok)
		require.Equal(t, generatedImplementation, sparseImplementation)
		require.Equal(t, unrelatedAccount, setup.L2Cfg.Alloc[unrelatedAddr])
	})

	t.Run("frozen allocation removes the generated implementation", func(t *testing.T) {
		setup := Setup(t, dp, &AllocParams{L2AllocIsFrozenPreForkState: true})
		frozenProxy := setup.L2Cfg.Alloc[proxyAddr]

		expectedProxy := generatedProxy
		expectedProxy.Storage = make(map[common.Hash]common.Hash, len(generatedProxy.Storage)-1)
		for slot, value := range generatedProxy.Storage {
			expectedProxy.Storage[slot] = value
		}
		delete(expectedProxy.Storage, genesis.ImplementationSlot)

		require.Contains(t, generatedProxy.Storage, genesis.ImplementationSlot)
		require.Contains(t, generatedProxy.Storage, genesis.AdminSlot)
		require.Equal(t, expectedProxy, frozenProxy)
		_, generatedImplementationPresent := setup.L2Cfg.Alloc[generatedImplementationAddr]
		require.False(t, generatedImplementationPresent,
			"frozen allocation must not inherit the generated implementation account at %s", generatedImplementationAddr)
	})

	t.Run("explicit proxy and historical implementation remain authoritative when frozen", func(t *testing.T) {
		unrelatedSlot := common.HexToHash("0x01")
		historicalImplementationAddr := common.HexToAddress("0x1234")
		require.NotEqual(t, generatedImplementationAddr, historicalImplementationAddr)
		historicalImplementation := types.Account{
			Nonce:   8,
			Balance: big.NewInt(5678),
			Code:    []byte{0x60, 0x02, 0x56},
		}
		explicit := types.Account{
			Nonce:   7,
			Balance: big.NewInt(1234),
			Code:    []byte{0x60, 0x00, 0x56},
			Storage: map[common.Hash]common.Hash{
				genesis.ImplementationSlot: common.BytesToHash(historicalImplementationAddr.Bytes()),
				unrelatedSlot:              common.HexToHash("0x5678"),
			},
		}
		expected := explicit
		expected.Balance = new(big.Int).Set(explicit.Balance)
		expected.Code = append([]byte(nil), explicit.Code...)
		expected.Storage = make(map[common.Hash]common.Hash, len(explicit.Storage))
		for slot, value := range explicit.Storage {
			expected.Storage[slot] = value
		}

		setup := Setup(t, dp, &AllocParams{
			L2Alloc: types.GenesisAlloc{
				proxyAddr:                    explicit,
				historicalImplementationAddr: historicalImplementation,
			},
			L2AllocIsFrozenPreForkState: true,
		})
		require.Equal(t, expected, setup.L2Cfg.Alloc[proxyAddr])
		require.Equal(t, historicalImplementation, setup.L2Cfg.Alloc[historicalImplementationAddr])
		_, generatedImplementationPresent := setup.L2Cfg.Alloc[generatedImplementationAddr]
		require.False(t, generatedImplementationPresent,
			"frozen allocation must not retain the stale generated implementation account at %s", generatedImplementationAddr)
	})

	t.Run("proxy-disabled account is not pruned when frozen", func(t *testing.T) {
		wethAddr := predeploys.WETHAddr
		generatedWETH, ok := Setup(t, dp, &AllocParams{}).L2Cfg.Alloc[wethAddr]
		require.True(t, ok)
		require.NotEmpty(t, generatedWETH.Code)

		frozenWETH, ok := Setup(t, dp, &AllocParams{L2AllocIsFrozenPreForkState: true}).L2Cfg.Alloc[wethAddr]
		require.True(t, ok)
		require.Equal(t, generatedWETH, frozenWETH)
	})
}
