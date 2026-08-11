package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
)

// NewPreparedDeployment freezes the inputs and predictions for undeployed intent chains.
// Values committed by separate stages, such as Prestate and StartingAnchorRoot, remain in ChainState.
func NewPreparedDeployment(
	intent *state.Intent,
	st *state.State,
	deployer common.Address,
	opcm common.Address,
	bundle artifacts.Bundle,
) (*state.PreparedDeployment, error) {
	pending := make(map[common.Hash]bool)
	for _, chain := range intent.Chains {
		pending[chain.ID] = !st.IsChainDeployed(chain.ID)
	}
	preparedIntent, err := canonicalPreparedIntent(intent, common.Address{}, pending)
	if err != nil {
		return nil, err
	}
	l1Digest, err := artifacts.ContentDigest(bundle.L1)
	if err != nil {
		return nil, fmt.Errorf("failed to fingerprint L1 artifacts: %w", err)
	}
	l2Digest, err := artifacts.ContentDigest(bundle.L2)
	if err != nil {
		return nil, fmt.Errorf("failed to fingerprint L2 artifacts: %w", err)
	}

	prepared := &state.PreparedDeployment{
		Intent:   preparedIntent,
		Deployer: deployer,
		OPCM:     opcm,
		L1Artifacts: state.PreparedArtifact{
			Locator:       intent.L1ContractsLocator,
			ContentDigest: l1Digest,
		},
		L2Artifacts: state.PreparedArtifact{
			Locator:       intent.L2ContractsLocator,
			ContentDigest: l2Digest,
		},
	}
	for _, chain := range intent.Chains {
		if st.IsChainDeployed(chain.ID) {
			continue
		}
		chainState, err := st.Chain(chain.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to freeze prepared chain %s: %w", chain.ID.Hex(), err)
		}
		if chainState.StartBlock == nil || chainState.GenesisTime == nil {
			return nil, fmt.Errorf("prepared chain %s has no pinned anchor and genesis time", chain.ID.Hex())
		}
		prepared.Chains = append(prepared.Chains, &state.PreparedChainState{
			ID:               chain.ID,
			OpChainContracts: chainState.OpChainContracts,
			StartBlock:       chainState.StartBlock,
			GenesisTime:      chainState.GenesisTime,
		})
	}

	// Detach the durable snapshot from all live intent and state pointers.
	return prepared.Clone()
}

// ValidatePreparedDeployment rejects live settings or artifact locators that no longer
// match the snapshot for chains awaiting deployment.
func ValidatePreparedDeployment(intent *state.Intent, st *state.State) error {
	if intent == nil {
		return fmt.Errorf("intent is missing")
	}
	if st == nil || st.PreparedDeployment == nil {
		return fmt.Errorf("state was not produced by op-deployer prepare; run op-deployer prepare")
	}
	prepared := st.PreparedDeployment
	if prepared.Intent == nil || prepared.Intent.SuperchainConfigProxy == nil {
		return fmt.Errorf("prepared deployment has no canonical intent; rerun op-deployer prepare")
	}
	if prepared.Deployer == (common.Address{}) || prepared.OPCM == (common.Address{}) {
		return fmt.Errorf("prepared deployment has no pinned deployer or OPCM; rerun op-deployer prepare")
	}
	if intent.OPCMAddress != nil && *intent.OPCMAddress != prepared.OPCM {
		return fmt.Errorf("intent OPCM address changed after prepare: pinned %s, intent %s", prepared.OPCM, *intent.OPCMAddress)
	}

	pending := make(map[common.Hash]bool, len(prepared.Intent.Chains))
	for _, chain := range prepared.Intent.Chains {
		pending[chain.ID] = true
	}
	currentPrepared := 0
	for _, chain := range intent.Chains {
		if pending[chain.ID] {
			currentPrepared++
			continue
		}
		if !st.IsChainDeployed(chain.ID) {
			return fmt.Errorf("deployment intent changed after prepare; rerun op-deployer prepare")
		}
	}
	if currentPrepared != len(prepared.Intent.Chains) {
		return fmt.Errorf("deployment intent changed after prepare; rerun op-deployer prepare")
	}
	current, err := canonicalPreparedIntent(intent, *prepared.Intent.SuperchainConfigProxy, pending)
	if err != nil {
		return fmt.Errorf("failed to canonicalize current intent: %w", err)
	}
	want, err := json.Marshal(prepared.Intent)
	if err != nil {
		return fmt.Errorf("failed to encode prepared intent: %w", err)
	}
	got, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("failed to encode current intent: %w", err)
	}
	if !bytes.Equal(want, got) {
		return fmt.Errorf("deployment intent changed after prepare; rerun op-deployer prepare")
	}
	if !sameLocator(intent.L1ContractsLocator, prepared.L1Artifacts.Locator) {
		return fmt.Errorf("L1 artifact locator changed after prepare; rerun op-deployer prepare")
	}
	if !sameLocator(intent.L2ContractsLocator, prepared.L2Artifacts.Locator) {
		return fmt.Errorf("L2 artifact locator changed after prepare; rerun op-deployer prepare")
	}
	return nil
}

// ValidateCommittedPrestateOverrides rejects explicit live overrides that conflict
// with a prestate already committed to ChainState.
func ValidateCommittedPrestateOverrides(intent *state.Intent, st *state.State) error {
	if intent == nil {
		return fmt.Errorf("intent is missing")
	}
	if st == nil || st.PreparedDeployment == nil || st.PreparedDeployment.Intent == nil {
		return fmt.Errorf("prepared deployment is missing")
	}

	for _, preparedChain := range st.PreparedDeployment.Intent.Chains {
		chain, err := intent.Chain(preparedChain.ID)
		if err != nil {
			return fmt.Errorf("failed to get current intent for chain %s: %w", preparedChain.ID.Hex(), err)
		}
		chainState, err := st.Chain(preparedChain.ID)
		if err != nil {
			return fmt.Errorf("failed to get prepared state for chain %s: %w", preparedChain.ID.Hex(), err)
		}
		if chainState.Prestate == (common.Hash{}) {
			continue
		}

		proofParams, err := ResolveChainProofParams(intent, chain)
		if err != nil {
			return fmt.Errorf("failed to resolve proof parameters for chain %s: %w", chain.ID.Hex(), err)
		}
		requirements, err := ResolveInitialDeployRequirements(proofParams.DisputeGameType)
		if err != nil {
			return fmt.Errorf("chain %s: %w", chain.ID.Hex(), err)
		}
		if !requirements.RequiresPrestate || !hasFaultGameAbsolutePrestateOverride(intent, chain) {
			continue
		}
		if proofParams.DisputeAbsolutePrestate != chainState.Prestate {
			return fmt.Errorf(
				"chain %s faultGameAbsolutePrestate override differs from the committed prestate. Rerun op-deployer prestate",
				chain.ID.Hex(),
			)
		}
	}
	return nil
}

func hasFaultGameAbsolutePrestateOverride(intent *state.Intent, chain *state.ChainIntent) bool {
	if _, ok := chain.DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey]; ok {
		return true
	}
	_, ok := intent.GlobalDeployOverrides[state.FaultGameAbsolutePrestateOverrideKey]
	return ok
}

// ValidatePreparedArtifactContents rejects bundles that no longer match the
// content digests recorded by prepare.
func ValidatePreparedArtifactContents(prepared *state.PreparedDeployment, bundle artifacts.Bundle) error {
	if prepared == nil {
		return fmt.Errorf("prepared deployment is missing")
	}
	l1Digest, err := artifacts.ContentDigest(bundle.L1)
	if err != nil {
		return fmt.Errorf("failed to fingerprint L1 artifacts: %w", err)
	}
	if l1Digest != prepared.L1Artifacts.ContentDigest {
		return fmt.Errorf("L1 artifact contents changed after prepare")
	}
	l2Digest, err := artifacts.ContentDigest(bundle.L2)
	if err != nil {
		return fmt.Errorf("failed to fingerprint L2 artifacts: %w", err)
	}
	if l2Digest != prepared.L2Artifacts.ContentDigest {
		return fmt.Errorf("L2 artifact contents changed after prepare")
	}
	return nil
}

// PreparedChainProofParams resolves proof parameters from the frozen intent,
// independently of later edits to the live intent.
func PreparedChainProofParams(st *state.State, chainID common.Hash) (state.ChainProofParams, error) {
	if st == nil || st.PreparedDeployment == nil || st.PreparedDeployment.Intent == nil {
		return state.ChainProofParams{}, fmt.Errorf("prepared deployment is missing")
	}
	chain, err := st.PreparedDeployment.Intent.Chain(chainID)
	if err != nil {
		return state.ChainProofParams{}, err
	}
	return ResolveChainProofParams(st.PreparedDeployment.Intent, chain)
}

func canonicalPreparedIntent(
	intent *state.Intent,
	defaultSuperchainConfig common.Address,
	include map[common.Hash]bool,
) (*state.Intent, error) {
	if intent == nil {
		return nil, fmt.Errorf("intent is missing")
	}
	superchainConfig := defaultSuperchainConfig
	if intent.SuperchainConfigProxy != nil {
		superchainConfig = *intent.SuperchainConfigProxy
	}
	if superchainConfig == (common.Address{}) {
		return nil, fmt.Errorf("intent.superchainConfigProxy must be set")
	}

	canonical := &state.Intent{
		ConfigType:            intent.ConfigType,
		L1ChainID:             intent.L1ChainID,
		SuperchainConfigProxy: &superchainConfig,
		FundDevAccounts:       intent.FundDevAccounts,
		GlobalDeployOverrides: intent.GlobalDeployOverrides,
		UseInterop:            intent.UseInterop,
		OutputRootBootstrap:   intent.OutputRootBootstrap,
	}

	for _, chain := range intent.Chains {
		if include[chain.ID] {
			canonical.Chains = append(canonical.Chains, chain)
		}
	}
	canonical, err := canonical.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone canonical intent: %w", err)
	}

	proofParams := make([]state.ChainProofParams, len(canonical.Chains))
	requirements := make([]InitialDeployRequirements, len(canonical.Chains))
	for i, chain := range canonical.Chains {
		proofParams[i], err = resolveProofParamsWithoutPrestate(canonical, chain)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve proof parameters for chain %s: %w", chain.ID.Hex(), err)
		}
		requirements[i], err = ResolveInitialDeployRequirements(proofParams[i].DisputeGameType)
		if err != nil {
			return nil, fmt.Errorf("chain %s: %w", chain.ID.Hex(), err)
		}
		if !requirements[i].RequiresPrestate {
			proofParams[i], err = ResolveChainProofParams(canonical, chain)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve proof parameters for chain %s: %w", chain.ID.Hex(), err)
			}
		}
	}
	delete(canonical.GlobalDeployOverrides, state.FaultGameAbsolutePrestateOverrideKey)
	for i, chain := range canonical.Chains {
		chain.L1StartBlockHash = nil
		chain.L2DevGenesisParams = nil
		if requirements[i].RequiresPrestate {
			delete(chain.DeployOverrides, state.FaultGameAbsolutePrestateOverrideKey)
			continue
		}
		if chain.DeployOverrides == nil {
			chain.DeployOverrides = make(map[string]any)
		}
		chain.DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = proofParams[i].DisputeAbsolutePrestate
	}
	return canonical, nil
}

func resolveProofParamsWithoutPrestate(intent *state.Intent, chain *state.ChainIntent) (state.ChainProofParams, error) {
	global := make(map[string]any, len(intent.GlobalDeployOverrides))
	for key, value := range intent.GlobalDeployOverrides {
		if key != state.FaultGameAbsolutePrestateOverrideKey {
			global[key] = value
		}
	}
	overrides := make(map[string]any, len(chain.DeployOverrides))
	for key, value := range chain.DeployOverrides {
		if key != state.FaultGameAbsolutePrestateOverrideKey {
			overrides[key] = value
		}
	}
	return ResolveChainProofParams(
		&state.Intent{GlobalDeployOverrides: global},
		&state.ChainIntent{DeployOverrides: overrides},
	)
}

func sameLocator(a, b *artifacts.Locator) bool {
	return a != nil && b != nil && a.Equal(b)
}
