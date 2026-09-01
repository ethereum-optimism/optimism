package sysgo

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	projectiongenesis "github.com/ethereum-optimism/optimism/op-private-interop/genesis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// A private chain and its public projection, out of one op-deployer world.
//
// Op-deployer runs once and emits the private-chain genesis plus a rollup config carrying the
// private_interop marker. The public projection is derived locally from those artifacts by the
// same pure functions used in production. There is no second deployer intent, role, or apply run.

// devstackLockVaultPlaceholder is the private chain's `lockVault` in the devstack.
//
// The real address is the ETHLockVault deployed on the counterparty chain, and the private chain's
// NativeMintBridge is constructed with it at genesis so that ETH locked over there can be minted
// over here. Nothing on the T1 path touches that bridge: the messenger e2e neither locks nor mints,
// and the intent's Check only requires the field to be non-zero.
//
// It is a NAMED PLACEHOLDER rather than an arbitrary address because the distinction matters to
// whoever reads a genesis: this is a value the devstack has not deployed yet, not a value it got
// wrong. The ETH lock-mint round trip (testing plan section 5.2) deploys the vault on chain A and
// replaces this; until it does, any test that calls the mint bridge will fail against an address
// with no code, which is the correct outcome for a bridge whose far side does not exist.
var devstackLockVaultPlaceholder = common.HexToAddress("0x00000000000000000000000000000000104c6f636b") // "…Lock"

// privateInteropDeployerOptions is the intent configuration for a private chain and its public
// projection, plus the custom gas token that makes it a private chain in the first place.
//
// The counterparty is left alone. Before WithCustomGasTokenOn existed the only door to the intent's
// per-chain CustomGasToken fanned out over every L2, so a preset could ask for "all chains CGT" or
// "no chains CGT" and nothing else; a pair needs precisely the in-between, since the private chain
// is CGT and its counterparty is not.
func privateInteropDeployerOptions(
	privateChainID, counterpartyChainID eth.ChainID,
	batcher common.Address,
) []DeployerOption {
	counterparty, ok := counterpartyChainID.Uint64()
	if !ok {
		panic("private interop: the counterparty's chain ID does not fit in a uint64")
	}
	// Application-level bridge liquidity remains in NativeAssetLiquidity until ETH is actually
	// locked on the counterparty; no account receives an unrelated genesis premine.
	liquidity := new(big.Int).Mul(big.NewInt(10_000_000), big.NewInt(1e18))
	return []DeployerOption{
		WithCustomGasTokenOn(privateChainID, "Private Interop Token", "PIT", liquidity, batcher),
		WithPrivateInterop(privateChainID, &state.PrivateInterop{
			CounterpartyChainID: counterparty,
			LockVault:           devstackLockVaultPlaceholder,
		}),
	}
}

// projectPrivateInteropGenesis derives the public projection locally from the private-chain
// artifacts. The world's outputs remain the private chain, which is what its sequencer executes.
func projectPrivateInteropGenesis(
	t devtest.T,
	wb *worldBuilder,
	chainID eth.ChainID,
) (*core.Genesis, *rollup.Config) {
	require := t.Require()
	privateChainGenesis, ok := wb.outL2Genesis[chainID]
	require.Truef(ok, "missing private-chain genesis for %s", chainID)
	privateRollup, ok := wb.outL2RollupCfg[chainID]
	require.Truef(ok, "missing private-chain rollup config for %s", chainID)

	publicProjectionGenesis, err := projectiongenesis.ProjectGenesisFrom(privateChainGenesis)
	require.NoError(err, "projecting the private-chain genesis")
	publicProjectionRollup, err := projectiongenesis.ProjectRollupConfigFrom(privateRollup, publicProjectionGenesis)
	require.NoError(err, "projecting the private-chain rollup config")
	require.NotEqual(privateRollup.Genesis.L2.Hash, publicProjectionRollup.Genesis.L2.Hash)
	return publicProjectionGenesis, publicProjectionRollup
}

// privateInteropPublicProjectionNetwork wraps the public projection as an L2Network the rest of sysgo can start
// nodes against.
//
// It shares the private chain's chain ID, deployment and keys -- one chain ID, one set of L1
// contracts, one batch inbox -- and differs in exactly the two things that make it a different
// chain: its genesis and its rollup config. The name differs too, because two networks that print
// the same name in a log line are two networks nobody can debug.
func privateInteropPublicProjectionNetwork(
	private *L2Network,
	genesis *core.Genesis,
	rollupCfg *rollup.Config,
) *L2Network {
	return &L2Network{
		name:       private.name + "-public-projection",
		chainID:    private.chainID,
		l1ChainID:  private.l1ChainID,
		genesis:    genesis,
		rollupCfg:  rollupCfg,
		deployment: private.deployment,
		opcmImpl:   private.opcmImpl,
		mipsImpl:   private.mipsImpl,
		keys:       private.keys,
	}
}
