package pipeline

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

// genesisOutput is one chain's own plain V0 genesis output root, and the L2
// genesis timestamp that root commits to.
type genesisOutput struct {
	outputRoot common.Hash
	timestamp  uint64
}

// ComputeGenesisOutputRoots builds the L2 genesis block for every chain that has not been deployed
// yet, then computes and persists each chain's genesis block hash and starting anchor proposal.
//
// It runs in two passes: The first pass derives each chain's own V0 output root, the second decides
// what is handed to the AnchorStateRegistry and is the only writer of StartingAnchorRoot, so the
// two game families cannot drift apart.
func ComputeGenesisOutputRoots(pEnv *Env, intent *state.Intent, st *state.State) error {
	lgr := pEnv.Logger.New("stage", "compute-genesis-output-root")

	// One output root per chain
	outputs := make(map[common.Hash]genesisOutput, len(intent.Chains))
	for _, chain := range intent.Chains {
		if st.IsChainDeployed(chain.ID) {
			// A deployed chain's on-chain anchor is immutable, and chains deployed via plain apply
			// never populate these fields, so writing them here would fabricate values unrelated
			// to what is actually on L1.
			lgr.Info("chain already deployed, leaving its genesis output root alone", "id", chain.ID.Hex())
			continue
		}
		out, err := computeGenesisOutput(lgr, intent, st, chain.ID)
		if err != nil {
			return err
		}
		outputs[chain.ID] = out
	}

	useSuperRoots, err := DeploymentUsesSuperRoots(intent, st)
	if err != nil {
		return err
	}
	if useSuperRoots {
		return setSuperRootAnchors(lgr, intent, st, outputs)
	}
	return setOutputRootAnchors(lgr, intent, st, outputs)
}

// computeGenesisOutput builds one chain's L2 genesis block from its generated allocs,
// combined deploy config and pinned anchor/genesis time, persists the resulting block hash, and
// returns the chain's own plain V0 output root.
func computeGenesisOutput(lgr log.Logger, intent *state.Intent, st *state.State, chainID common.Hash) (genesisOutput, error) {
	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return genesisOutput{}, fmt.Errorf("failed to get chain intent: %w", err)
	}

	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return genesisOutput{}, fmt.Errorf("failed to get chain state: %w", err)
	}

	if thisChainState.Allocs == nil {
		return genesisOutput{}, fmt.Errorf("cannot compute genesis output root for chain %s: L2 genesis allocs not yet generated", chainID.Hex())
	}
	if thisChainState.StartBlock == nil || thisChainState.GenesisTime == nil {
		return genesisOutput{}, fmt.Errorf("cannot compute genesis output root for chain %s: anchor block and genesis time not yet pinned", chainID.Hex())
	}

	lgr.Info("computing genesis output root", "id", chainID.Hex())

	config, err := state.CombineDeployConfig(intent, thisIntent, st, thisChainState)
	if err != nil {
		return genesisOutput{}, fmt.Errorf("failed to combine deploy config: %w", err)
	}

	l2Genesis, err := genesis.BuildL2Genesis(&config, thisChainState.Allocs.Data, thisChainState.StartBlock.ToBlockRef())
	if err != nil {
		return genesisOutput{}, fmt.Errorf("failed to build L2 genesis: %w", err)
	}

	block := l2Genesis.ToBlock()
	header := block.Header()
	if header.WithdrawalsHash == nil {
		return genesisOutput{}, fmt.Errorf(
			"chain %s: L2 genesis block has no withdrawals root; genesis output root computation requires Isthmus to be active at genesis",
			chainID.Hex(),
		)
	}

	outputRoot, err := rollup.ComputeL2OutputRootV0(eth.HeaderBlockInfo(header), *header.WithdrawalsHash)
	if err != nil {
		return genesisOutput{}, fmt.Errorf("failed to compute genesis output root: %w", err)
	}

	blockHash := block.Hash()
	thisChainState.GenesisBlockHash = &blockHash

	lgr.Info(
		"computed genesis output root",
		"id", chainID.Hex(),
		"outputRoot", common.Hash(outputRoot),
		"blockHash", blockHash,
		"genesisTime", header.Time,
	)

	return genesisOutput{outputRoot: common.Hash(outputRoot), timestamp: header.Time}, nil
}

// setOutputRootAnchors seeds each chain with its own plain V0 genesis output root. For output-root
// games l2SequenceNumber is an L2 block number, and the genesis block is 0.
func setOutputRootAnchors(lgr log.Logger, intent *state.Intent, st *state.State, outputs map[common.Hash]genesisOutput) error {
	for _, chain := range intent.Chains {
		out, ok := outputs[chain.ID]
		if !ok {
			continue
		}
		if err := setStartingAnchor(st, chain.ID, out.outputRoot, 0); err != nil {
			return err
		}
		lgr.Info(
			"committed genesis anchor",
			"id", chain.ID.Hex(),
			"anchorRoot", out.outputRoot,
			"anchorSequenceNumber", 0,
		)
	}
	return nil
}

// setSuperRootAnchors seeds every chain with the one SuperV1 root covering the whole dependency
// set. Each chain keeps its own AnchorStateRegistry (registries are never shared) but all of
// them are anchored to the same cluster-wide super root.
//
// For super-root games l2SequenceNumber is the timestamp of the committed super root.
func setSuperRootAnchors(lgr log.Logger, intent *state.Intent, st *state.State, outputs map[common.Hash]genesisOutput) error {
	if st.InteropDepSet == nil {
		return fmt.Errorf("cannot compute a super-root genesis anchor: no interop dependency set recorded in state; rerun op-deployer prepare")
	}
	depSetChains := st.InteropDepSet.Chains()
	if len(depSetChains) == 0 {
		return fmt.Errorf("cannot compute a super-root genesis anchor: the recorded interop dependency set is empty; rerun op-deployer prepare")
	}

	// The super root covers the whole dependency set, so every member has to contribute an output
	// root derived in this run, all of the members MUST share the same genesis timestamp.
	chainOutputs := make([]eth.ChainIDAndOutput, 0, len(depSetChains))
	firstID := common.Hash(depSetChains[0].Bytes32())
	var timestamp uint64
	for i, depSetChain := range depSetChains {
		id := common.Hash(depSetChain.Bytes32())
		// Obtain the already computed genesis output root for this chain.
		out, ok := outputs[id]
		if !ok {
			return fmt.Errorf(
				"cannot compute a super-root genesis anchor: dependency set member %s has no genesis output root in this run; "+
					"it is either already deployed, in which case run op-deployer continue to reuse the committed anchors, "+
					"or missing from the intent",
				id.Hex(),
			)
		}
		if i == 0 {
			// The first member sets the timestamp for the super root.
			timestamp = out.timestamp
		} else if out.timestamp != timestamp {
			return fmt.Errorf(
				"cannot compute a super-root genesis anchor: dependency set members disagree on the L2 genesis timestamp "+
					"(chain %s pins %d, chain %s pins %d); a super root at a timestamp requires every member to exist at it, "+
					"so all chains in one super-root deployment must share a genesis time; check for a per-chain "+
					"l1StartBlockHash override",
				firstID.Hex(), timestamp, id.Hex(), out.timestamp,
			)
		}
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{
			ChainID: depSetChain,
			Output:  eth.Bytes32(out.outputRoot),
		})
	}

	superRoot := eth.NewSuperV1(timestamp, chainOutputs...)
	anchorRoot := common.Hash(eth.SuperRoot(superRoot))

	lgr.Info(
		"computed cluster-wide super-root genesis anchor",
		"chains", len(chainOutputs),
		"anchorRoot", anchorRoot,
		"anchorSequenceNumber", timestamp,
	)

	for _, chain := range intent.Chains {
		if _, ok := outputs[chain.ID]; !ok {
			continue
		}
		if err := setStartingAnchor(st, chain.ID, anchorRoot, hexutil.Uint64(timestamp)); err != nil {
			return err
		}
	}
	return nil
}

// setStartingAnchor records the proposal that continue hands to OPCM.deploy for this chain.
func setStartingAnchor(st *state.State, chainID common.Hash, root common.Hash, sequenceNumber hexutil.Uint64) error {
	chainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state for %s: %w", chainID.Hex(), err)
	}
	chainState.StartingAnchorRoot = &state.StartingAnchorProposal{
		Root:             root,
		L2SequenceNumber: sequenceNumber,
	}
	return nil
}
