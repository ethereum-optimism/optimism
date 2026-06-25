package state

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// These tests SKEPTICALLY validate the consequence of the op-deployer re-apply config
// drift mechanism. The MECHANISM (immutability guard ignores genesis-affecting fields;
// on-chain SystemConfig + L2 allocs frozen on re-apply; regenerated rollup/genesis read
// the NEW intent live) is established elsewhere. Here we test the CONSEQUENCE per field:
// does the NEW regenerated value cause real harm, or is it inert metadata?
//
// All tests drive REAL production functions:
//   - state.CombineDeployConfig            (op-deployer/pkg/deployer/state/deploy_config.go:24)
//   - genesis.NewL2Genesis                 (op-chain-ops/genesis/genesis.go:29) -- the exact
//     genesis-header builder used by genesis.BuildL2Genesis inside the pipeline's
//     RenderGenesisAndRollup (op-deployer/pkg/deployer/pipeline/env.go:128)
//   - genesis.DeployConfig.GenesisSystemConfig (op-chain-ops/genesis/config.go:1161)
//   - eth.SystemConfig.OperatorFee         (op-service/eth/types.go:735)

func consequenceBaseChain() ChainIntent {
	return ChainIntent{
		ID:                         common.HexToHash("0x123"),
		Eip1559Denominator:         1,
		Eip1559Elasticity:          2,
		GasLimit:                   standard.GasLimit,
		BaseFeeVaultRecipient:      common.HexToAddress("0x123"),
		L1FeeVaultRecipient:        common.HexToAddress("0x456"),
		SequencerFeeVaultRecipient: common.HexToAddress("0x789"),
		OperatorFeeVaultRecipient:  common.HexToAddress("0xabc"),
		Roles: ChainRoles{
			SystemConfigOwner: common.HexToAddress("0x123"),
			L1ProxyAdminOwner: common.HexToAddress("0x456"),
			L2ProxyAdminOwner: common.HexToAddress("0x789"),
			UnsafeBlockSigner: common.HexToAddress("0xabc"),
			Batcher:           common.HexToAddress("0xBA7C0E"),
		},
		OperatorFeeScalar:   100,
		OperatorFeeConstant: 200,
	}
}

func consequenceBaseIntent() Intent {
	return Intent{
		L1ChainID:          1,
		L1ContractsLocator: artifacts.EmbeddedLocator,
	}
}

// buildGenesisHeaderHash runs the REAL pipeline building blocks (CombineDeployConfig ->
// NewL2Genesis -> ToBlock) and returns the genesis BLOCK HASH. This mirrors the genesis
// header that RenderGenesisAndRollup produces; we use NewL2Genesis (the header builder)
// directly instead of BuildL2Genesis to avoid needing a full 2048-predeploy alloc set,
// since gasLimit only affects the header, not the alloc accounts.
func buildGenesisHeaderHash(t *testing.T, intent Intent, chain ChainIntent) (common.Hash, eth.SystemConfig) {
	t.Helper()
	chainState := ChainState{ID: chain.ID}
	st := State{SuperchainDeployment: &addresses.SuperchainContracts{}}

	cfg, err := CombineDeployConfig(&intent, &chain, &st, &chainState)
	require.NoError(t, err)

	// L2GenesisBlockNumber/timestamp inputs are deterministic from a zero start ref.
	startRef := eth.BlockRef{}
	g, err := genesis.NewL2Genesis(&cfg, &startRef)
	require.NoError(t, err)

	return g.ToBlock().Hash(), cfg.GenesisSystemConfig()
}

// TestReapplyDrift_GasLimit_ChangesGenesisBlockHash
//
// VERDICT INPUT for the gasLimit field: re-applying with a changed gasLimit produces a
// DIFFERENT genesis block hash. A regenerated rollup.json/genesis would not match the
// already-running chain's genesis -> new nodes computing genesis would disagree.
func TestReapplyDrift_GasLimit_ChangesGenesisBlockHash(t *testing.T) {
	intent := consequenceBaseIntent()

	oldChain := consequenceBaseChain()
	oldChain.GasLimit = 30_000_000

	newChain := consequenceBaseChain()
	newChain.GasLimit = 60_000_000 // CHANGED on re-apply

	oldHash, oldSys := buildGenesisHeaderHash(t, intent, oldChain)
	newHash, newSys := buildGenesisHeaderHash(t, intent, newChain)

	require.NotEqual(t, oldHash, newHash,
		"gasLimit change MUST alter the genesis block hash (header.GasLimit feeds the hash)")
	require.EqualValues(t, 30_000_000, oldSys.GasLimit)
	require.EqualValues(t, 60_000_000, newSys.GasLimit)

	t.Logf("gasLimit drift: genesis block hash %s (gasLimit 30M) -> %s (gasLimit 60M); "+
		"GenesisSystemConfig.GasLimit %d -> %d. A regenerated genesis would not match the "+
		"running chain's original genesis hash.", oldHash, newHash, oldSys.GasLimit, newSys.GasLimit)
}

// TestReapplyDrift_OperatorFee_DoesNotChangeGenesisBlockHash_ButChangesRollupSystemConfig
//
// VERDICT INPUT for operatorFee and batcher: these are NOT in the genesis block header,
// so they do NOT change the genesis block hash (no chain-join failure from them). BUT they
// DO change the rollup config's GenesisSystemConfig, which (per consumer-authority tracing)
// is the value op-node SEEDS its system-config view from at genesis -- so the NEW value is
// effective, while the frozen on-chain L1 SystemConfig holds the OLD value.
func TestReapplyDrift_OperatorFee_DoesNotChangeGenesisBlockHash_ButChangesRollupSystemConfig(t *testing.T) {
	intent := consequenceBaseIntent()

	oldChain := consequenceBaseChain()
	newChain := consequenceBaseChain()
	newChain.OperatorFeeScalar = 999
	newChain.OperatorFeeConstant = 888
	newChain.Roles.Batcher = common.HexToAddress("0xDEADBEEF")

	oldHash, oldSys := buildGenesisHeaderHash(t, intent, oldChain)
	newHash, newSys := buildGenesisHeaderHash(t, intent, newChain)

	// operatorFee + batcher are NOT header fields -> genesis block hash unchanged.
	require.Equal(t, oldHash, newHash,
		"operatorFee/batcher are not in the genesis header; genesis block hash is unaffected by them")

	// ... but the rollup config's GenesisSystemConfig DOES carry the NEW values.
	require.EqualValues(t, 100, oldSys.OperatorFee().Scalar)
	require.EqualValues(t, 200, oldSys.OperatorFee().Constant)
	require.Equal(t, common.HexToAddress("0xBA7C0E"), oldSys.BatcherAddr)

	require.EqualValues(t, 999, newSys.OperatorFee().Scalar,
		"NEW operatorFeeScalar lands in rollup GenesisSystemConfig (op-node seeds from here)")
	require.EqualValues(t, 888, newSys.OperatorFee().Constant,
		"NEW operatorFeeConstant lands in rollup GenesisSystemConfig")
	require.Equal(t, common.HexToAddress("0xDEADBEEF"), newSys.BatcherAddr,
		"NEW batcher lands in rollup GenesisSystemConfig")

	t.Logf("operatorFee/batcher drift: genesis block hash UNCHANGED (%s); but rollup "+
		"GenesisSystemConfig changed -> batcher %s->%s, operatorFeeScalar %d->%d, "+
		"operatorFeeConstant %d->%d. op-node seeds its sysConfig from this value at genesis "+
		"(PayloadToSystemConfig -> rollupCfg.Genesis.SystemConfig), so NEW is effective while "+
		"on-chain L1 SystemConfig stays OLD.",
		oldHash, oldSys.BatcherAddr, newSys.BatcherAddr,
		oldSys.OperatorFee().Scalar, newSys.OperatorFee().Scalar,
		oldSys.OperatorFee().Constant, newSys.OperatorFee().Constant)
}
