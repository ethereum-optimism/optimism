package sysgo

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

// The two halves of a private interop pair, out of ONE op-deployer world.
//
// # Why one world and not two runs
//
// A pair is two genesis states under one chain ID (op-private-interop/docs/DESIGN.md, "Ratified
// decisions" item 6). The production runbook is two `op-deployer apply` runs differing only in the
// chain's `privateInterop` stanza, and that runbook carries a constraint recorded in
// op-deployer/pkg/deployer/integration_test/private_interop_rollup_config_test.go: BOTH runs must
// pin the same `l1StartBlockHash`, or the halves take their L2 genesis timestamps from different L1
// blocks, the two chains start numbering at different times, and the builder can never emit a block
// at both the private chain's number and its timestamp.
//
// A devstack cannot follow that runbook. `l1StartBlockHash` is resolved by reading the block off a
// live L1 (op-deployer/pkg/deployer/pipeline/start_block.go), and a genesis-target run has no L1 to
// read: it seals its own. Two runs would therefore be two L1s and two timestamps -- exactly the
// failure the constraint warns about, with no hook to prevent it.
//
// So this does something stronger than pinning. It runs the world ONCE, with chain B as the PRIVATE
// half, and then re-renders that one chain's L2 genesis with the stanza flipped to `rendering`,
// against the SAME state: the same L1 genesis, the same deployed L1 contracts, and above all the
// same `chainState.StartBlock`, which is where both halves' genesis timestamps come from. The
// halves cannot disagree about their L1 start block because there is only one, and there is nothing
// to pin. `TestPrivateInteropPairSharesItsStartBlock` asserts exactly that.
//
// The re-render is the real op-deployer stage -- `pipeline.GenerateL2Genesis` followed by
// `inspect.GenesisAndRollup`, the same two calls `op-deployer inspect genesis|rollup` makes -- so
// the rendering's genesis is produced by production code, not by a devstack-shaped imitation of it.
// Nothing in the L1 half of the pipeline is re-run, and nothing about the real state is mutated:
// the flip happens on a copy.

// privateInteropOperatorMnemonicIndex derives the operator EOA, the account that signs every replay
// and every claim on the rendering.
//
// It sits next to funderMnemonicIndex, outside the 30 standard role accounts, for the same reason:
// the operator is not one of op-deployer's roles, it is an ordinary EOA that happens to hold a
// genesis premine. Deriving it from the mnemonic rather than generating one keeps the pair's genesis
// reproducible across runs, which is what makes a builder-determinism test possible at all.
const privateInteropOperatorMnemonicIndex = 10_001

// PrivateInteropOperatorKey returns the operator EOA's key.
//
// The same key reaches two places that must agree: the deployer intent, which premines the address
// on the rendering, and the batcher, which signs the rendering's transactions with it. They are
// derived from one function so they cannot drift.
func PrivateInteropOperatorKey(keys devkeys.Keys) (*ecdsa.PrivateKey, error) {
	return keys.Secret(devkeys.UserKey(privateInteropOperatorMnemonicIndex))
}

// devstackLockVaultPlaceholder is the private half's `lockVault` in the devstack.
//
// The real address is the ETHLockVault deployed on the counterparty chain, and the private half's
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

// privateInteropDeployerOptions is the intent configuration for a pair: the private half's stanza,
// and the custom gas token that makes it a private chain in the first place.
//
// The counterparty is left alone. Before WithCustomGasTokenOn existed the only door to the intent's
// per-chain CustomGasToken fanned out over every L2, so a preset could ask for "all chains CGT" or
// "no chains CGT" and nothing else; a pair needs precisely the in-between, since the private chain
// is CGT and neither its counterparty nor its own rendering is.
func privateInteropDeployerOptions(
	privateChainID, counterpartyChainID eth.ChainID,
	operator common.Address,
	premineETH uint64,
) []DeployerOption {
	premine := new(big.Int).Mul(new(big.Int).SetUint64(premineETH), big.NewInt(1e18))
	counterparty, ok := counterpartyChainID.Uint64()
	if !ok {
		panic("private interop: the counterparty's chain ID does not fit in a uint64")
	}
	// The liquidity the private chain opens with. The operator's premine is minted out of it, so it
	// must be the larger of the two; the intent's own Check enforces that ordering.
	liquidity := new(big.Int).Mul(premine, big.NewInt(1000))
	return []DeployerOption{
		WithCustomGasTokenOn(privateChainID, "Private Interop Token", "PIT", liquidity, operator),
		WithPrivateInterop(privateChainID, &state.PrivateInterop{
			Role:                state.PrivateInteropPrivateChain,
			Operator:            operator,
			OperatorBalance:     (*hexutil.Big)(premine),
			CounterpartyChainID: counterparty,
			LockVault:           devstackLockVaultPlaceholder,
		}),
	}
}

// renderPrivateInteropRendering produces the PUBLIC half of the pair from the world the private half
// was built in.
//
// The world's own outputs are untouched: wb.outL2Genesis[chainID] stays the private half, which is
// the chain the sequencer builds and tests transact against. What comes back here is the rendering,
// which is what the supernode derives and what every counterparty means when it names this chain ID.
func renderPrivateInteropRendering(
	t devtest.T,
	wb *worldBuilder,
	chainID eth.ChainID,
	operator common.Address,
	premineETH uint64,
) (*core.Genesis, *rollup.Config) {
	require := t.Require()
	require.NotNil(wb.output, "the world must be built before its rendering can be")
	require.NotNil(wb.output.AppliedIntent, "the applied intent is where the pair's stanza lives")

	id := common.Hash(chainID.Bytes32())

	renderingIntent := cloneIntentAsRendering(t, wb.output.AppliedIntent, id, operator, premineETH)
	renderingState := cloneStateForRerender(wb.output, id, renderingIntent)

	bundle, err := artifacts.DownloadBundle(
		t.Ctx(),
		renderingIntent.L1ContractsLocator,
		renderingIntent.L2ContractsLocator,
		ioutil.BarProgressor(),
		wb.deployerCacheDir,
	)
	require.NoError(err, "resolving the contract artifacts the rendering's genesis is built from")

	// Only four Env fields are read by the L2-genesis stage: the logger, the deployer address, the
	// unoptimized-contracts allowance and (through the bundle) the L2 artifacts. Everything else on
	// Env belongs to the L1 stages, which are emphatically not being re-run.
	pEnv := &pipeline.Env{
		Logger:   t.Logger().New("stage", "private-interop-rendering"),
		Deployer: wb.deployerAddr,
		// Matches the genesis-target pipeline: devstack contracts come from the forge lite profile
		// and exceed EIP-170.
		AllowUnoptimizedContracts: true,
		IsGenesis:                 true,
	}
	require.NoError(
		pipeline.GenerateL2Genesis(pEnv, renderingIntent, bundle, renderingState, id),
		"rendering the public half of the private interop pair",
	)

	genesis, rollupCfg, err := inspect.GenesisAndRollup(renderingState, id)
	require.NoError(err, "reading the rendering's genesis and rollup config")

	// The pair's whole reason to exist is that these two are one chain seen two ways. If the
	// rendering came back at a different chain ID, every replay transaction the builder signs would
	// be for a chain nobody is running.
	private, ok := wb.outL2RollupCfg[chainID]
	require.Truef(ok, "missing the private half's rollup config for chain %s", chainID)
	require.Zero(rollupCfg.L2ChainID.Cmp(private.L2ChainID), "the pair's halves must share a chain ID")
	require.Equal(private.Genesis.L1, rollupCfg.Genesis.L1,
		"the halves must start from the same L1 block, or their genesis timestamps diverge and no block can be at both the private number and the private timestamp")
	require.Equal(private.Genesis.L2Time, rollupCfg.Genesis.L2Time, "the halves must share a genesis timestamp")
	require.Equal(private.Genesis.L2.Number, rollupCfg.Genesis.L2.Number, "the halves must start numbering at the same block")
	require.Equal(private.BlockTime, rollupCfg.BlockTime, "the halves must share a block time")
	require.NotEqual(private.Genesis.L2.Hash, rollupCfg.Genesis.L2.Hash,
		"the halves are different chains with the same ID; the genesis hash is what tells them apart, and it is why they must never gossip-peer")

	return genesis, rollupCfg
}

// cloneIntentAsRendering copies the applied intent and flips one chain to the rendering half.
//
// Two edits, and both are forced by the intent's own Check: the stanza's role changes, and the
// custom gas token goes away. The rendering pays gas in its own ETH -- it has to, because its replay
// transactions are ordinary signed transactions from an EOA with a premine -- so a CGT rendering is
// rejected outright.
func cloneIntentAsRendering(
	t devtest.T,
	src *state.Intent,
	id common.Hash,
	operator common.Address,
	premineETH uint64,
) *state.Intent {
	out := *src
	out.Chains = make([]*state.ChainIntent, len(src.Chains))
	var found bool
	for i, ch := range src.Chains {
		if ch.ID != id {
			out.Chains[i] = ch
			continue
		}
		cp := *ch
		cp.CustomGasToken = state.CustomGasToken{}
		cp.PrivateInterop = &state.PrivateInterop{
			Role:            state.PrivateInteropRendering,
			Operator:        operator,
			OperatorBalance: (*hexutil.Big)(new(big.Int).Mul(new(big.Int).SetUint64(premineETH), big.NewInt(1e18))),
		}
		out.Chains[i] = &cp
		found = true
	}
	t.Require().Truef(found, "no chain %s in the applied intent", id)
	return &out
}

// cloneStateForRerender copies the state with one chain's allocs cleared, so the L2-genesis stage
// regenerates that chain and only that chain.
//
// The stage's own guard is `Allocs == nil` (pipeline.shouldGenerateL2Genesis), so clearing the field
// is how you ask for a re-render; every other chain keeps its allocs and is skipped. Everything the
// L1 stages produced -- the deployed contract addresses, the dependency set, and critically the
// chain's StartBlock -- is shared by reference and never written to.
func cloneStateForRerender(src *state.State, id common.Hash, intent *state.Intent) *state.State {
	out := *src
	out.AppliedIntent = intent
	out.Chains = make([]*state.ChainState, len(src.Chains))
	for i, ch := range src.Chains {
		if ch.ID != id {
			out.Chains[i] = ch
			continue
		}
		cp := *ch
		cp.Allocs = nil
		out.Chains[i] = &cp
	}
	return &out
}

// privateInteropRenderingNetwork wraps the rendering as an L2Network the rest of sysgo can start
// nodes against.
//
// It shares the private half's chain ID, deployment and keys -- one chain ID, one set of L1
// contracts, one batch inbox -- and differs in exactly the two things that make it a different
// chain: its genesis and its rollup config. The name differs too, because two networks that print
// the same name in a log line are two networks nobody can debug.
func privateInteropRenderingNetwork(
	private *L2Network,
	genesis *core.Genesis,
	rollupCfg *rollup.Config,
) *L2Network {
	return &L2Network{
		name:       private.name + "-rendering",
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
