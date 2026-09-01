package genesis

import (
	"encoding/json"
	"math/big"
	"os"
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

const (
	goldenPublicProjectionStateRoot = "0xb4a3530d2e27fd008c94ba70c47760c3109a032d3c646872591ca622b4a59cae"
	goldenPublicProjectionBlockHash = "0xaaca1c33c16e560b035482ad57fd6f38436d9c5cf45faf57e3f3cb3512335347"
)

func TestProjectGenesisFromIsPureAndDeterministic(t *testing.T) {
	private := testPrivateChainGenesis()
	before := cloneGenesis(private)

	first, err := ProjectGenesisFrom(private)
	require.NoError(t, err)
	second, err := ProjectGenesisFrom(private)
	require.NoError(t, err)

	require.True(t, reflect.DeepEqual(before, private), "projection mutated its source genesis")
	require.True(t, reflect.DeepEqual(first, second))
	require.Equal(t, first.ToBlock().Hash(), second.ToBlock().Hash())
	require.Zero(t, first.BaseFee.Sign())
	require.Equal(t, uint64(params.MaxGasLimit), first.GasLimit)
}

func TestPublicProjectionGoldenVector(t *testing.T) {
	data, err := os.ReadFile("testdata/private-chain-genesis.json")
	require.NoError(t, err)
	var private core.Genesis
	require.NoError(t, json.Unmarshal(data, &private))
	public, err := ProjectGenesisFrom(&private)
	require.NoError(t, err)
	block := public.ToBlock()
	require.Equal(t, goldenPublicProjectionStateRoot, block.Root().Hex())
	require.Equal(t, goldenPublicProjectionBlockHash, block.Hash().Hex())
}

func TestProjectGenesisFromRewritesOnlyThePublicProjectionState(t *testing.T) {
	private := testPrivateChainGenesis()
	public, err := ProjectGenesisFrom(private)
	require.NoError(t, err)

	for _, proxy := range []common.Address{
		predeploys.SuperchainETHBridgeAddr,
		predeploys.ETHLiquidityAddr,
		predeploys.L2toL2CrossDomainMessengerAddr,
		predeploys.ClaimRegistryAddr,
		predeploys.EventReplayerAddr,
	} {
		require.Equal(t, common.BytesToHash(codeNamespace(proxy).Bytes()), public.Alloc[proxy].Storage[implementationSlot])
		require.NotEmpty(t, public.Alloc[codeNamespace(proxy)].Code)
	}
	for _, proxy := range []common.Address{
		predeploys.NativeAssetLiquidityAddr,
		predeploys.LiquidityControllerAddr,
		predeploys.NativeMintBridgeAddr,
	} {
		require.Equal(t, common.Hash{}, public.Alloc[proxy].Storage[implementationSlot])
		_, ok := public.Alloc[codeNamespace(proxy)]
		require.False(t, ok)
	}
	require.Zero(t, public.Alloc[predeploys.L1BlockAddr].Storage[customGasTokenSlot])
	require.False(t, devfeatures.IsDevFeatureEnabled(
		public.Alloc[predeploys.L2DevFeatureFlagsAddr].Storage[devFeatureBitmapSlot],
		devfeatures.PrivateInteropFlag,
	))
	require.True(t, devfeatures.IsDevFeatureEnabled(
		public.Alloc[predeploys.L2DevFeatureFlagsAddr].Storage[devFeatureBitmapSlot],
		devfeatures.OptimismPortalInteropFlag,
	), "projection must retain unrelated feature bits")
	require.Zero(t, public.Alloc[predeploys.LiquidityControllerAddr].Balance.Sign())
	require.Equal(t, maxUint128, public.Alloc[predeploys.ETHLiquidityAddr].Balance)

	// An unrelated deployment-specific allocation is copied exactly.
	unrelated := common.HexToAddress("0x1234")
	require.Equal(t, private.Alloc[unrelated], public.Alloc[unrelated])
}

func TestProjectGenesisFromRejectsOrdinaryGenesis(t *testing.T) {
	ordinary := testPrivateChainGenesis()
	bridge := ordinary.Alloc[predeploys.NativeMintBridgeAddr]
	delete(bridge.Storage, implementationSlot)
	ordinary.Alloc[predeploys.NativeMintBridgeAddr] = bridge

	_, err := ProjectGenesisFrom(ordinary)
	require.ErrorContains(t, err, "NativeMintBridge is inactive")
}

func TestProjectRollupConfigFrom(t *testing.T) {
	privateChainGenesis := testPrivateChainGenesis()
	publicProjectionGenesis, err := ProjectGenesisFrom(privateChainGenesis)
	require.NoError(t, err)

	private := &rollup.Config{
		Genesis: rollup.Genesis{
			L2: eth.BlockID{Hash: privateChainGenesis.ToBlock().Hash()},
			SystemConfig: eth.SystemConfig{
				GasLimit:          privateChainGenesis.GasLimit,
				Scalar:            eth.EncodeScalar(eth.EcotoneScalars{BaseFeeScalar: 1, BlobBaseFeeScalar: 2}),
				OperatorFeeParams: eth.EncodeOperatorFeeParams(eth.OperatorFeeParams{Scalar: 3, Constant: 4}),
				MinBaseFee:        5,
			},
		},
		PrivateInterop: &rollup.PrivateInteropConfig{ExtraEmitters: []common.Address{common.HexToAddress("0xbeef")}},
	}
	projected, err := ProjectRollupConfigFrom(private, publicProjectionGenesis)
	require.NoError(t, err)

	require.Equal(t, publicProjectionGenesis.ToBlock().Hash(), projected.Genesis.L2.Hash)
	require.Equal(t, uint64(params.MaxGasLimit), projected.Genesis.SystemConfig.GasLimit)
	scalars, err := eth.DecodeScalar(projected.Genesis.SystemConfig.Scalar)
	require.NoError(t, err)
	require.Equal(t, eth.EcotoneScalars{}, scalars)
	require.Equal(t, eth.OperatorFeeParams{}, projected.Genesis.SystemConfig.OperatorFee())
	require.Zero(t, projected.Genesis.SystemConfig.MinBaseFee)
	require.NotSame(t, private.PrivateInterop, projected.PrivateInterop)
	require.Equal(t, uint64(5), private.Genesis.SystemConfig.MinBaseFee, "projection mutated its source config")
}

func testPrivateChainGenesis() *core.Genesis {
	config := *params.AllDevChainProtocolChanges
	config.ChainID = big.NewInt(901)
	genesis := &core.Genesis{
		Config:     &config,
		Timestamp:  1_700_000_000,
		GasLimit:   30_000_000,
		Difficulty: new(big.Int),
		BaseFee:    big.NewInt(1_000_000_000),
		Alloc:      make(types.GenesisAlloc),
	}

	for _, proxy := range []common.Address{
		predeploys.L1BlockAddr,
		predeploys.L2ToL1MessagePasserAddr,
		predeploys.L2toL2CrossDomainMessengerAddr,
		predeploys.SuperchainETHBridgeAddr,
		predeploys.ETHLiquidityAddr,
		predeploys.NativeAssetLiquidityAddr,
		predeploys.LiquidityControllerAddr,
		predeploys.ClaimRegistryAddr,
		predeploys.EventReplayerAddr,
		predeploys.NativeMintBridgeAddr,
	} {
		genesis.Alloc[proxy] = types.Account{
			Code:    []byte{0x60, 0x00},
			Balance: new(big.Int),
			Storage: map[common.Hash]common.Hash{adminSlot: proxyAdminWord},
		}
	}
	activateTestProxy(genesis.Alloc, predeploys.NativeAssetLiquidityAddr)
	activateTestProxy(genesis.Alloc, predeploys.LiquidityControllerAddr)
	activateTestProxy(genesis.Alloc, predeploys.NativeMintBridgeAddr)
	l1Block := genesis.Alloc[predeploys.L1BlockAddr]
	l1Block.Storage[customGasTokenSlot] = common.BigToHash(big.NewInt(1))
	genesis.Alloc[predeploys.L1BlockAddr] = l1Block
	genesis.Alloc[common.HexToAddress("0x1234")] = types.Account{
		Balance: big.NewInt(123),
		Nonce:   7,
		Code:    []byte{0xde, 0xad},
	}
	genesis.Alloc[predeploys.L2DevFeatureFlagsAddr] = types.Account{
		Balance: new(big.Int),
		Storage: map[common.Hash]common.Hash{
			devFeatureBitmapSlot: devfeatures.EnableDevFeature(
				devfeatures.OptimismPortalInteropFlag, devfeatures.PrivateInteropFlag,
			),
		},
	}
	return genesis
}

func activateTestProxy(alloc types.GenesisAlloc, proxy common.Address) {
	account := alloc[proxy]
	account.Storage[implementationSlot] = common.BytesToHash(codeNamespace(proxy).Bytes())
	alloc[proxy] = account
	alloc[codeNamespace(proxy)] = types.Account{
		Code:    []byte{0xfe},
		Balance: new(big.Int),
		Storage: map[common.Hash]common.Hash{adminSlot: proxyAdminWord},
	}
}
