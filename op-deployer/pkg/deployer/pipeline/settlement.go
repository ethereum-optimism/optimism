package pipeline

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

// PrivateChainSettlement returns the settlement chain of an OP Private private chain: a chain
// whose settlementChainId deploy override names another chain. A private chain shares the L1
// contracts and the L1 start block of its settlement chain, which must be in the same intent.
// The chain's own override takes precedence over the global one.
func PrivateChainSettlement(intent *state.Intent, chain *state.ChainIntent) (common.Hash, bool, error) {
	settlementID, err := settlementChainOf(intent, chain)
	if err != nil {
		return common.Hash{}, false, err
	}
	if settlementID == (common.Hash{}) || settlementID == chain.ID {
		return common.Hash{}, false, nil
	}
	return settlementID, true, nil
}

// settlementChainOf returns the chain's settlementChainId deploy override as a chain ID, or the
// zero hash if there is none.
func settlementChainOf(intent *state.Intent, chain *state.ChainIntent) (common.Hash, error) {
	overrides, err := jsonutil.MergeJSON(
		struct {
			SettlementChainID uint64 `json:"settlementChainId"`
		}{},
		intent.GlobalDeployOverrides,
		chain.DeployOverrides,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("chain %s: failed to read the settlementChainId override: %w", chain.ID.Big(), err)
	}
	if overrides.SettlementChainID == 0 {
		return common.Hash{}, nil
	}
	return common.BigToHash(new(big.Int).SetUint64(overrides.SettlementChainID)), nil
}

// checkSettlementGroups checks the OP Private chains of the intent. A private chain's settlement
// chain must be earlier in the same intent and must be its own settlement chain, so groups have
// one level and no cycles. A private chain must not configure L1 state of its own.
func checkSettlementGroups(intent *state.Intent) error {
	position := make(map[common.Hash]int, len(intent.Chains))
	for i, chain := range intent.Chains {
		position[chain.ID] = i
	}
	for i, chain := range intent.Chains {
		settlementID, err := settlementChainOf(intent, chain)
		if err != nil {
			return err
		}
		if settlementID == (common.Hash{}) {
			continue
		}
		if err := checkSettlementChainIDOverride(devFeatureBitmapOverride(intent), bigs.Uint64Strict(settlementID.Big())); err != nil {
			return fmt.Errorf("chain %s: %w", chain.ID.Big(), err)
		}
		if settlementID == chain.ID {
			continue
		}

		j, ok := position[settlementID]
		if !ok {
			return fmt.Errorf("private chain %s: settlement chain %s must be in the same intent", chain.ID.Big(), settlementID.Big())
		}
		if j > i {
			return fmt.Errorf("private chain %s: settlement chain %s must come before it in the intent", chain.ID.Big(), settlementID.Big())
		}
		ofSettlement, err := settlementChainOf(intent, intent.Chains[j])
		if err != nil {
			return err
		}
		if ofSettlement != settlementID {
			return fmt.Errorf("private chain %s: chain %s is not a settlement chain; its settlementChainId override must be its own chain ID", chain.ID.Big(), settlementID.Big())
		}
		if chain.L1StartBlockHash != nil {
			return fmt.Errorf("private chain %s: l1StartBlockHash must be unset; the chain uses its settlement chain's start block", chain.ID.Big())
		}
		if chain.DangerousAltDAConfig.UseAltDA || len(chain.AdditionalDisputeGames) > 0 {
			return fmt.Errorf("private chain %s: alt-DA and additional dispute games must be unset; the chain shares its settlement chain's L1 contracts", chain.ID.Big())
		}
	}
	return nil
}

// reuseSettlementContracts records the settlement chain's L1 contracts as the private chain's.
func reuseSettlementContracts(st *state.State, chainID common.Hash, settlementID common.Hash) error {
	if !st.IsChainDeployed(settlementID) {
		return fmt.Errorf("settlement chain %s of private chain %s is not deployed", settlementID.Big(), chainID.Big())
	}
	settlement, err := st.Chain(settlementID)
	if err != nil {
		return err
	}
	st.SetChainContracts(chainID, settlement.OpChainContracts, true)
	return nil
}

// reuseSettlementStartBlock gives a private chain the start block and genesis time of its
// settlement chain. It reports whether the chain is a private chain.
func reuseSettlementStartBlock(intent *state.Intent, st *state.State, chainID common.Hash) (bool, error) {
	chain, err := intent.Chain(chainID)
	if err != nil {
		return false, fmt.Errorf("failed to get chain intent: %w", err)
	}
	settlementID, private, err := PrivateChainSettlement(intent, chain)
	if err != nil || !private {
		return false, err
	}
	settlement, err := st.Chain(settlementID)
	if err != nil {
		return true, err
	}
	if settlement.StartBlock == nil {
		return true, fmt.Errorf("settlement chain %s of private chain %s has no start block", settlementID.Big(), chainID.Big())
	}
	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return true, fmt.Errorf("failed to get chain state: %w", err)
	}
	startBlock := *settlement.StartBlock
	thisChainState.StartBlock = &startBlock
	thisChainState.GenesisTime = nil
	if settlement.GenesisTime != nil {
		genesisTime := *settlement.GenesisTime
		thisChainState.GenesisTime = &genesisTime
	}
	return true, nil
}
