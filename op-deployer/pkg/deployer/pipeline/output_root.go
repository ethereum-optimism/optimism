package pipeline

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ComputeGenesisOutputRoot builds the L2 genesis block from the chain's already-generated
// allocs, combined deploy config, and pinned anchor/genesis time, then computes and persists
// the genesis output root and block hash.
func ComputeGenesisOutputRoot(pEnv *Env, intent *state.Intent, st *state.State, chainID common.Hash) error {
	lgr := pEnv.Logger.New("stage", "compute-genesis-output-root")

	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain intent: %w", err)
	}

	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	if st.IsChainDeployed(chainID) {
		// Chains deployed via plain apply never populate these fields, so recomputing here
		// would fabricate a StartingAnchorRoot/GenesisBlockHash unrelated to what's actually on L1.
		lgr.Info("chain already deployed, not recomputing genesis output root", "id", chainID.Hex())
		return nil
	}

	if thisChainState.StartingAnchorRoot != nil && thisChainState.GenesisBlockHash != nil {
		lgr.Info("genesis output root already computed")
		return nil
	}

	if thisChainState.Allocs == nil {
		return fmt.Errorf("cannot compute genesis output root for chain %s: L2 genesis allocs not yet generated", chainID.Hex())
	}
	if thisChainState.StartBlock == nil || thisChainState.GenesisTime == nil {
		return fmt.Errorf("cannot compute genesis output root for chain %s: anchor block and genesis time not yet pinned", chainID.Hex())
	}

	lgr.Info("computing genesis output root", "id", chainID.Hex())

	config, err := state.CombineDeployConfig(intent, thisIntent, st, thisChainState)
	if err != nil {
		return fmt.Errorf("failed to combine deploy config: %w", err)
	}

	l2Genesis, err := genesis.BuildL2Genesis(&config, thisChainState.Allocs.Data, thisChainState.StartBlock.ToBlockRef())
	if err != nil {
		return fmt.Errorf("failed to build L2 genesis: %w", err)
	}

	block := l2Genesis.ToBlock()
	header := block.Header()
	if header.WithdrawalsHash == nil {
		return fmt.Errorf(
			"chain %s: L2 genesis block has no withdrawals root; genesis output root computation requires Isthmus to be active at genesis",
			chainID.Hex(),
		)
	}

	outputRoot, err := rollup.ComputeL2OutputRootV0(eth.HeaderBlockInfo(header), *header.WithdrawalsHash)
	if err != nil {
		return fmt.Errorf("failed to compute genesis output root: %w", err)
	}

	blockHash := block.Hash()
	outputRootHash := common.Hash(outputRoot)

	proofParams, err := ResolveChainProofParams(intent, thisIntent)
	if err != nil {
		return fmt.Errorf("failed to resolve proof params for chain %s: %w", chainID.Hex(), err)
	}

	// l2SequenceNumber identifies the anchor's position in the chain's ordered sequence of
	// states, and the two game families measure that differently. Output-root games use the L2
	// block number, so the genesis anchor is 0. Super-root games use the timestamp of the
	// committed super root, so the genesis anchor carries the L2 genesis time.
	anchorRoot := outputRootHash
	var anchorSequenceNumber hexutil.Uint64
	if IsSuperGameType(proofParams.DisputeGameType) {
		// A solo chain's genesis anchor must be the SuperV1 encoding of just its own output
		superRoot := eth.NewSuperV1(header.Time, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromBytes32(chainID),
			Output:  eth.Bytes32(outputRootHash),
		})
		anchorRoot = common.Hash(eth.SuperRoot(superRoot))
		anchorSequenceNumber = hexutil.Uint64(header.Time)
	}

	thisChainState.GenesisBlockHash = &blockHash
	thisChainState.StartingAnchorRoot = &state.StartingAnchorProposal{
		Root:             anchorRoot,
		L2SequenceNumber: anchorSequenceNumber,
	}

	lgr.Info(
		"computed genesis output root",
		"outputRoot", outputRootHash,
		"anchorRoot", anchorRoot,
		"anchorSequenceNumber", uint64(anchorSequenceNumber),
		"blockHash", blockHash,
	)

	return nil
}
