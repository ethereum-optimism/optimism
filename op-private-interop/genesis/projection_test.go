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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// The golden vector is shared with the Rust implementation (rust/op-reth/crates/chainspec/src/
// private_interop.rs): the same fixture must project to the same state root and block hash there.
//
// testdata/private-chain-genesis.json is a STOCK op-deployer genesis of a custom-gas-token chain
// with interop active at genesis (useInterop=true, devFeatureBitmap bit 0, Lagoon at offset zero),
// reduced to the predeploy proxies, their implementations, and one funded dev account. Nothing in it
// was hand-written; regenerate it with a stock op-deployer when the contract release moves and
// update StockL2ToL2CrossDomainMessengerCodeHash alongside it.
const (
	goldenPublicProjectionStateRoot = "0x0bcc2c671be43df86d2bdfc0e2ef4a8402924e2bdfaf5e8e432eb5841282c81f"
	goldenPublicProjectionBlockHash = "0xb16205791987c1d52be25be000d27661f42aeccf033183f3a1ea396ce23539be"
)

func TestProjectGenesisFromIsPureAndDeterministic(t *testing.T) {
	private := loadPrivateChainGenesis(t)
	before := cloneGenesis(private)

	first, err := ProjectGenesisFrom(private)
	require.NoError(t, err)
	second, err := ProjectGenesisFrom(private)
	require.NoError(t, err)

	require.Equal(t, genesisJSON(t, before), genesisJSON(t, private), "projection mutated its source genesis")
	require.Equal(t, genesisJSON(t, first), genesisJSON(t, second))
	require.Equal(t, first.ToBlock().Hash(), second.ToBlock().Hash())
	require.Zero(t, first.BaseFee.Sign())
	require.Equal(t, uint64(params.MaxGasLimit), first.GasLimit)
}

func TestPublicProjectionGoldenVector(t *testing.T) {
	private := loadPrivateChainGenesis(t)
	public, err := ProjectGenesisFrom(private)
	require.NoError(t, err)
	block := public.ToBlock()
	require.Equal(t, goldenPublicProjectionStateRoot, block.Root().Hex())
	require.Equal(t, goldenPublicProjectionBlockHash, block.Hash().Hex())
}

func TestFixtureIsTheStockShape(t *testing.T) {
	private := loadPrivateChainGenesis(t)
	require.Equal(t, StockL2ToL2CrossDomainMessengerCodeHash,
		implementationCodeHash(private.Alloc, predeploys.L2toL2CrossDomainMessengerAddr),
		"fixture messenger is not the pinned stock release")
	require.Equal(t, trueWord, private.Alloc[predeploys.L1BlockAddr].Storage[l1BlockInteropFeatureSlot])
	require.NotZero(t, private.Alloc[predeploys.L1BlockAddr].Storage[customGasTokenSlot])
	require.Equal(t, maxUint128(), private.Alloc[predeploys.ETHLiquidityAddr].Balance, "stock interop genesis funds ETHLiquidity")
	require.Equal(t, common.Hash{}, private.Alloc[predeploys.ClaimRegistryAddr].Storage[implementationSlot])
	require.Equal(t, common.Hash{}, private.Alloc[predeploys.EventReplayerAddr].Storage[implementationSlot])
}

func TestProjectGenesisFromRewritesOnlyThePublicProjectionState(t *testing.T) {
	private := loadPrivateChainGenesis(t)
	public, err := ProjectGenesisFrom(private)
	require.NoError(t, err)

	// Installed by the projection.
	for _, proxy := range []common.Address{
		predeploys.L1BlockAddr,
		predeploys.L2ToL1MessagePasserAddr,
		predeploys.L2toL2CrossDomainMessengerAddr,
		predeploys.ClaimRegistryAddr,
		predeploys.EventReplayerAddr,
	} {
		require.Equal(t, common.BytesToHash(codeNamespace(proxy).Bytes()), public.Alloc[proxy].Storage[implementationSlot])
		require.Equal(t, publicProjectionCode[codeNamespace(proxy)], public.Alloc[codeNamespace(proxy)].Code)
	}
	require.NotEqual(t, private.Alloc[codeNamespace(predeploys.L1BlockAddr)].Code, public.Alloc[codeNamespace(predeploys.L1BlockAddr)].Code, "L1BlockCGT replaced")
	require.NotEqual(t, private.Alloc[codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr)].Code, public.Alloc[codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr)].Code, "stock messenger replaced")

	// Removed by the projection.
	for _, proxy := range []common.Address{predeploys.NativeAssetLiquidityAddr, predeploys.LiquidityControllerAddr} {
		require.Equal(t, common.Hash{}, public.Alloc[proxy].Storage[implementationSlot])
		_, ok := public.Alloc[codeNamespace(proxy)]
		require.False(t, ok)
		require.Zero(t, public.Alloc[proxy].Balance.Sign())
	}
	require.Zero(t, public.Alloc[predeploys.L1BlockAddr].Storage[customGasTokenSlot])

	// Kept by the projection: the stock interop feature set is the source's.
	require.Equal(t, trueWord, public.Alloc[predeploys.L1BlockAddr].Storage[l1BlockInteropFeatureSlot], "INTEROP feature survives the L1Block implementation swap")
	require.True(t, devfeatures.IsDevFeatureEnabled(
		public.Alloc[predeploys.L2DevFeatureFlagsAddr].Storage[devFeatureBitmapSlot],
		devfeatures.OptimismPortalInteropFlag,
	))
	for _, proxy := range []common.Address{predeploys.CrossL2InboxAddr, predeploys.SuperchainETHBridgeAddr, predeploys.ETHLiquidityAddr} {
		requireSameAccount(t, private.Alloc[proxy], public.Alloc[proxy], "%s untouched", proxy)
		requireSameAccount(t, private.Alloc[codeNamespace(proxy)], public.Alloc[codeNamespace(proxy)], "%s implementation untouched", proxy)
	}
	require.Equal(t, maxUint128(), public.Alloc[predeploys.ETHLiquidityAddr].Balance)

	// An unrelated deployment-specific allocation is copied exactly.
	unrelated := common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	require.Contains(t, private.Alloc, unrelated)
	requireSameAccount(t, private.Alloc[unrelated], public.Alloc[unrelated], "unrelated allocation copied")
}

func TestProjectGenesisFromRejectsNonSources(t *testing.T) {
	t.Run("interop-active ETH genesis is accepted", func(t *testing.T) {
		// An ETH private chain is a supported source: there is no CGT machinery to strip, so the
		// source's own L1Block and L2ToL1MessagePasser implementations are left exactly as they are.
		g := loadPrivateChainGenesis(t)
		deleteStorage(g.Alloc, predeploys.L1BlockAddr, customGasTokenSlot)
		public, err := ProjectGenesisFrom(g)
		require.NoError(t, err)
		for _, proxy := range []common.Address{predeploys.L1BlockAddr, predeploys.L2ToL1MessagePasserAddr} {
			requireSameAccount(t, g.Alloc[codeNamespace(proxy)], public.Alloc[codeNamespace(proxy)], "%s implementation untouched", proxy)
		}
		require.Equal(t, trueWord, public.Alloc[predeploys.L1BlockAddr].Storage[l1BlockInteropFeatureSlot])
		// The projection's own contracts are installed either way.
		for _, proxy := range []common.Address{predeploys.L2toL2CrossDomainMessengerAddr, predeploys.ClaimRegistryAddr, predeploys.EventReplayerAddr} {
			require.Equal(t, common.BytesToHash(codeNamespace(proxy).Bytes()), public.Alloc[proxy].Storage[implementationSlot])
		}
	})
	t.Run("interop not active in L1Block", func(t *testing.T) {
		g := loadPrivateChainGenesis(t)
		deleteStorage(g.Alloc, predeploys.L1BlockAddr, l1BlockInteropFeatureSlot)
		_, err := ProjectGenesisFrom(g)
		require.ErrorIs(t, err, ErrInteropInactive)
	})
	t.Run("interop dev feature unset", func(t *testing.T) {
		g := loadPrivateChainGenesis(t)
		deleteStorage(g.Alloc, predeploys.L2DevFeatureFlagsAddr, devFeatureBitmapSlot)
		_, err := ProjectGenesisFrom(g)
		require.ErrorIs(t, err, ErrInteropInactive)
	})
	t.Run("messenger from another release", func(t *testing.T) {
		g := loadPrivateChainGenesis(t)
		impl := g.Alloc[codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr)]
		impl.Code = append([]byte{0x00}, impl.Code...)
		g.Alloc[codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr)] = impl
		_, err := ProjectGenesisFrom(g)
		require.ErrorIs(t, err, ErrMessengerNotStock)
	})
	t.Run("already projected", func(t *testing.T) {
		g := loadPrivateChainGenesis(t)
		public, err := ProjectGenesisFrom(g)
		require.NoError(t, err)
		_, err = ProjectGenesisFrom(public)
		require.Error(t, err)
		// The projection has no custom gas token and a replay messenger; whichever check fires
		// first, a projection is never accepted as a source.
		require.True(t, err == ErrMessengerNotStock || err == ErrAlreadyProjected, err.Error())

		// A projection that somehow kept the CGT marker and stock messenger is still caught by its
		// projection predeploys.
		g = loadPrivateChainGenesis(t)
		activateProxy(g.Alloc, predeploys.ClaimRegistryAddr, []byte{0xfe})
		_, err = ProjectGenesisFrom(g)
		require.ErrorIs(t, err, ErrAlreadyProjected)
	})
}

func TestProjectRollupConfigFrom(t *testing.T) {
	privateChainGenesis := loadPrivateChainGenesis(t)
	publicProjectionGenesis, err := ProjectGenesisFrom(privateChainGenesis)
	require.NoError(t, err)

	lagoon := uint64(0)
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
		LagoonTime: &lagoon,
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
	require.Equal(t, private.LagoonTime, projected.LagoonTime, "both views activate interop at genesis")
	require.Equal(t, uint64(5), private.Genesis.SystemConfig.MinBaseFee, "projection mutated its source config")

	// Only the genesis hash and the genesis system config differ.
	projected.Genesis = private.Genesis
	require.True(t, reflect.DeepEqual(private, projected))
}

func TestStockMessengerHashMatchesFixture(t *testing.T) {
	private := loadPrivateChainGenesis(t)
	code := private.Alloc[codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr)].Code
	require.Equal(t, StockL2ToL2CrossDomainMessengerCodeHash, crypto.Keccak256Hash(code))
}

func loadPrivateChainGenesis(t *testing.T) *core.Genesis {
	data, err := os.ReadFile("testdata/private-chain-genesis.json")
	require.NoError(t, err)
	var private core.Genesis
	require.NoError(t, json.Unmarshal(data, &private))
	return &private
}

// requireSameAccount compares accounts by value: big.Int carries an internal representation that
// reflect.DeepEqual sees through and the clone does not preserve.
func requireSameAccount(t *testing.T, want, got types.Account, msgAndArgs ...any) {
	t.Helper()
	require.Equal(t, want.Code, got.Code, msgAndArgs...)
	require.Equal(t, want.Storage, got.Storage, msgAndArgs...)
	require.Equal(t, want.Nonce, got.Nonce, msgAndArgs...)
	require.Zero(t, want.Balance.Cmp(got.Balance), msgAndArgs...)
}

func genesisJSON(t *testing.T, g *core.Genesis) string {
	t.Helper()
	data, err := json.Marshal(g)
	require.NoError(t, err)
	return string(data)
}

func maxUint128() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
}
