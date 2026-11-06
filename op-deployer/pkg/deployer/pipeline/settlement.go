package pipeline

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

// settlementChainIDKey is the deploy override that names a chain's OP Private settlement chain.
const settlementChainIDKey = "settlementChainId"

// SettlementChainIDOverride returns the chain's settlementChainId deploy override: 0 on a public
// chain, the chain's own ID on a settlement chain, and its settlement chain's ID on a private
// chain. It is the only reader of the override. A global override is refused, because it would
// make every other chain of the intent a private chain.
//
// TODO(#23953): promote settlementChainId to a typed ChainIntent field when OP_PRIVATE graduates.
func SettlementChainIDOverride(intent *state.Intent, chain *state.ChainIntent) (uint64, error) {
	for key := range intent.GlobalDeployOverrides {
		if strings.EqualFold(key, settlementChainIDKey) {
			return 0, fmt.Errorf("globalDeployOverrides key %q is not allowed: set %s in each chain's deployOverrides", key, settlementChainIDKey)
		}
	}
	overrides, err := jsonutil.MergeJSON(
		struct {
			SettlementChainID uint64 `json:"settlementChainId"`
		}{},
		chain.DeployOverrides,
	)
	if err != nil {
		return 0, fmt.Errorf("chain %s: failed to read the %s override: %w", chain.ID.Hex(), settlementChainIDKey, err)
	}
	return overrides.SettlementChainID, nil
}

// PrivateChainSettlement returns the settlement chain of an OP Private private chain: a chain
// whose settlementChainId deploy override names another chain. A private chain shares the L1
// contracts of its settlement chain, which must be in the same intent.
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
	id, err := SettlementChainIDOverride(intent, chain)
	if err != nil || id == 0 {
		return common.Hash{}, err
	}
	return common.BigToHash(new(big.Int).SetUint64(id)), nil
}

// checkSettlementGroups checks the OP Private chains of the intent. A private chain's settlement
// chain must be earlier in the same intent and must be its own settlement chain, so groups have
// one level and no cycles. A private chain must configure its shared L1 contracts as its
// settlement chain does.
func checkSettlementGroups(intent *state.Intent) error {
	position := make(map[common.Hash]int, len(intent.Chains))
	settlementOf := make(map[common.Hash]common.Hash, len(intent.Chains))
	for i, chain := range intent.Chains {
		settlementID, err := settlementChainOf(intent, chain)
		if err != nil {
			return err
		}
		position[chain.ID] = i
		settlementOf[chain.ID] = settlementID
	}

	for i, chain := range intent.Chains {
		settlementID := settlementOf[chain.ID]
		if settlementID == (common.Hash{}) {
			continue
		}
		if err := checkSettlementChainIDOverride(devFeatureBitmapOverride(intent), bigs.Uint64Strict(settlementID.Big())); err != nil {
			return fmt.Errorf("chain %s: %w", chain.ID.Hex(), err)
		}
		if settlementID == chain.ID {
			continue
		}

		j, ok := position[settlementID]
		if !ok {
			return fmt.Errorf("private chain %s: settlement chain %s must be in the same intent", chain.ID.Hex(), settlementID.Hex())
		}
		if j > i {
			return fmt.Errorf("private chain %s: settlement chain %s must come before it in the intent", chain.ID.Hex(), settlementID.Hex())
		}
		if settlementOf[settlementID] != settlementID {
			return fmt.Errorf("private chain %s: chain %s is not a settlement chain; its %s override must be its own chain ID", chain.ID.Hex(), settlementID.Hex(), settlementChainIDKey)
		}
		if err := checkPrivateChainL1Config(intent, chain, intent.Chains[j]); err != nil {
			return err
		}
	}
	return nil
}

// checkPrivateChainL1Config checks that a private chain configures its L1 contracts as its
// settlement chain does. op-deployer deploys no L1 contracts for the private chain, so any other
// value would never reach L1. The chain's own values, such as its batcher, gas limit, EIP-1559
// and operator fee parameters, start block and genesis time, are left alone.
func checkPrivateChainL1Config(intent *state.Intent, chain *state.ChainIntent, settlement *state.ChainIntent) error {
	if chain.DangerousAltDAConfig.UseAltDA || len(chain.AdditionalDisputeGames) > 0 {
		return fmt.Errorf("%w: private chain %s: alt-DA and additional dispute games must be unset; the chain shares its settlement chain's L1 contracts", state.ErrIncompatibleValue, chain.ID.Hex())
	}

	mismatch := func(field string) error {
		return fmt.Errorf("%w: private chain %s: %s must equal that of settlement chain %s, whose L1 contracts it shares", state.ErrIncompatibleValue, chain.ID.Hex(), field, settlement.ID.Hex())
	}
	roles := []struct {
		name              string
		chain, settlement common.Address
	}{
		{"roles.l1ProxyAdminOwner", chain.Roles.L1ProxyAdminOwner, settlement.Roles.L1ProxyAdminOwner},
		{"roles.systemConfigOwner", chain.Roles.SystemConfigOwner, settlement.Roles.SystemConfigOwner},
		{"roles.proposer", chain.Roles.Proposer, settlement.Roles.Proposer},
		{"roles.challenger", chain.Roles.Challenger, settlement.Roles.Challenger},
	}
	for _, role := range roles {
		if role.chain != role.settlement {
			return mismatch(role.name)
		}
	}
	if !reflect.DeepEqual(chain.CustomGasToken, settlement.CustomGasToken) {
		return mismatch("customGasToken")
	}

	chainProofParams, err := ResolveChainProofParams(intent, chain)
	if err != nil {
		return fmt.Errorf("private chain %s: failed to resolve dispute params: %w", chain.ID.Hex(), err)
	}
	settlementProofParams, err := ResolveChainProofParams(intent, settlement)
	if err != nil {
		return fmt.Errorf("settlement chain %s: failed to resolve dispute params: %w", settlement.ID.Hex(), err)
	}
	if chainProofParams != settlementProofParams {
		return mismatch("the dispute game deploy overrides")
	}
	return nil
}

// checkSettlementChainIDsUnchanged refuses a change of an applied chain's settlementChainId. The
// chain's L2 genesis records the value and is not generated again.
func checkSettlementChainIDsUnchanged(intent *state.Intent, st *state.State) error {
	if st.AppliedIntent == nil {
		return nil
	}
	for _, chain := range intent.Chains {
		applied, err := st.AppliedIntent.Chain(chain.ID)
		if err != nil || !st.IsChainDeployed(chain.ID) {
			continue
		}
		was, err := SettlementChainIDOverride(st.AppliedIntent, applied)
		if err != nil {
			return fmt.Errorf("applied intent: %w", err)
		}
		is, err := SettlementChainIDOverride(intent, chain)
		if err != nil {
			return err
		}
		if was != is {
			return fmt.Errorf("chain %s: %s changed from %d to %d after apply; the chain's L2 genesis records it and is not generated again", chain.ID.Hex(), settlementChainIDKey, was, is)
		}
	}
	return nil
}

// reuseSettlementContracts records the settlement chain's L1 contracts as the private chain's.
func reuseSettlementContracts(st *state.State, chainID common.Hash, settlementID common.Hash) error {
	if !st.IsChainDeployed(settlementID) {
		return fmt.Errorf("settlement chain %s of private chain %s is not deployed", settlementID.Hex(), chainID.Hex())
	}
	settlement, err := st.Chain(settlementID)
	if err != nil {
		return err
	}
	st.SetChainContracts(chainID, settlement.OpChainContracts, true)
	return nil
}
