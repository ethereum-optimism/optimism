package pipeline

import (
	"context"
	"crypto/rand"
	"fmt"
	"reflect"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"

	"github.com/ethereum/go-ethereum/common"
)

func IsSupportedStateVersion(version int) bool {
	return version == 1
}

func InitLiveStrategy(ctx context.Context, env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "init", "strategy", "live")
	lgr.Info("initializing pipeline")

	if err := initCommonChecks(intent, st); err != nil {
		return err
	}

	hasPredeployedOPCM := intent.OPCMAddress != nil
	hasSuperchainConfigProxy := intent.SuperchainConfigProxy != nil

	if hasPredeployedOPCM || hasSuperchainConfigProxy {
		if intent.SuperchainRoles != nil {
			return fmt.Errorf("cannot set superchain roles when using predeployed OPCM or SuperchainConfig")
		}

		opcmAddr := common.Address{}
		if hasPredeployedOPCM {
			opcmAddr = *intent.OPCMAddress
		}

		superchainConfigAddr := common.Address{}
		if hasSuperchainConfigProxy {
			superchainConfigAddr = *intent.SuperchainConfigProxy
		}

		// If only an OPCM address is provided, resolve SuperchainConfigProxy from it on-chain.
		if superchainConfigAddr == (common.Address{}) && opcmAddr != (common.Address{}) {
			opcmContract := opcm.NewContract(opcmAddr, env.L1Client)
			resolved, err := opcmContract.SuperchainConfig(ctx)
			if err != nil {
				return fmt.Errorf("error resolving SuperchainConfig from OPCM at %s: %w", opcmAddr, err)
			}
			superchainConfigAddr = resolved
		}
		superDeployment, superRoles, err := PopulateSuperchainState(env, opcmAddr, superchainConfigAddr)
		if err != nil {
			return fmt.Errorf("error populating superchain state: %w", err)
		}
		st.SuperchainDeployment = superDeployment
		st.SuperchainRoles = superRoles

		if hasPredeployedOPCM && st.ImplementationsDeployment == nil {
			st.ImplementationsDeployment = &addresses.ImplementationsContracts{
				OpcmV2Impl: opcmAddr,
			}
		}
	}

	l1ChainID, err := env.L1Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get L1 chain ID: %w", err)
	}

	if l1ChainID.Cmp(intent.L1ChainIDBig()) != 0 {
		return fmt.Errorf("l1 chain ID mismatch: got %d, expected %d", l1ChainID, intent.L1ChainID)
	}

	deployerCode, err := env.L1Client.CodeAt(ctx, script.DeterministicDeployerAddress, nil)
	if err != nil {
		return fmt.Errorf("failed to get deployer code: %w", err)
	}
	if len(deployerCode) == 0 {
		return fmt.Errorf("deterministic deployer is not deployed on this chain - please deploy it first")
	}

	// If the state has never been applied, we don't need to perform
	// any additional checks.
	if st.AppliedIntent == nil {
		return nil
	}

	// If the state has been applied, we need to check if any immutable
	// fields have changed.
	if st.AppliedIntent.L1ChainID != intent.L1ChainID {
		return immutableErr("L1ChainID", st.AppliedIntent.L1ChainID, intent.L1ChainID)
	}

	if st.AppliedIntent.FundDevAccounts != intent.FundDevAccounts {
		return immutableErr("fundDevAccounts", st.AppliedIntent.FundDevAccounts, intent.FundDevAccounts)
	}

	// Once a chain has been applied, its L2 genesis allocs and its on-chain L1
	// SystemConfig are frozen: re-apply does not regenerate the L2 genesis
	// (l2genesis.go only runs when chainState.Allocs == nil) and does not redeploy
	// the OP chain contracts (opchain.go returns early when already deployed).
	// However, the genesis file and rollup config emitted by RenderGenesisAndRollup
	// are computed live from the current intent. If a field that feeds the genesis block
	// or the genesis SystemConfig changes on re-apply (e.g. gasLimit, roles.batcher,
	// operatorFeeScalar/Constant, or genesis-affecting deploy overrides), the emitted
	// genesis/rollup config would silently diverge from the already-deployed chain (the
	// on-chain SystemConfig holds the OLD value while op-node seeds its view from the NEW
	// rollup config Genesis.SystemConfig).
	//
	// Rather than enumerate individual fields, we compare the actual genesis OUTPUT
	// produced from the previously-applied intent vs the new intent, for every chain
	// that exists in both and has already been fully applied. Any difference in the
	// genesis block hash or the genesis SystemConfig is rejected as an immutable change.
	// This is precise: it only rejects changes that actually alter the genesis block or
	// SystemConfig, and it automatically covers future additions to those surfaces.
	// Changes that do not affect either surface are allowed -- including scheduling a
	// future hardfork time, L1-only/deploy-only params, brand-new chains, and
	// not-yet-applied chains.
	//
	// Note: this comparison intentionally does NOT cover fields that live only in the
	// frozen L2 allocs (e.g. fee-vault recipients in predeploy storage, which are not
	// regenerated on re-apply and therefore cannot diverge here) or fields that live only
	// in the genesis ChainConfig / broader rollup config but not the block hash or
	// SystemConfig (e.g. eip1559 params, l2BlockTime). The genesis ChainConfig is excluded
	// because its fork-activation times legitimately change for future-hardfork scheduling.
	for _, appliedChain := range st.AppliedIntent.Chains {
		chainID := appliedChain.ID

		chainState, err := st.Chain(chainID)
		if err != nil {
			// No chain state yet for this chain: it has not been applied, so there is
			// nothing frozen to diverge from. Skip.
			continue
		}
		if chainState.Allocs == nil || chainState.StartBlock == nil {
			// Chain is not fully applied (L2 genesis/start block not yet produced), so its
			// genesis is not frozen. Skip; RenderGenesisAndRollup would also be unable to
			// render it.
			continue
		}
		if _, err := intent.Chain(chainID); err != nil {
			// Chain present in the applied intent but absent from the new intent (removal).
			// RenderGenesisAndRollup(new) cannot render it; leave removal handling to other
			// stages and skip the immutability comparison.
			continue
		}

		oldGenesis, oldRollup, err := RenderGenesisAndRollup(st, chainID, st.AppliedIntent)
		if err != nil {
			return fmt.Errorf("failed to render genesis for applied intent (chain %s): %w", chainID, err)
		}
		newGenesis, newRollup, err := RenderGenesisAndRollup(st, chainID, intent)
		if err != nil {
			return fmt.Errorf("failed to render genesis for new intent (chain %s): %w", chainID, err)
		}

		if oldGenesis.ToBlock().Hash() != newGenesis.ToBlock().Hash() {
			return immutableErr(
				fmt.Sprintf("genesis block for chain %s", chainID),
				oldGenesis.ToBlock().Hash(),
				newGenesis.ToBlock().Hash(),
			)
		}
		if !reflect.DeepEqual(oldRollup.Genesis.SystemConfig, newRollup.Genesis.SystemConfig) {
			return immutableErr(
				fmt.Sprintf("genesis system config for chain %s", chainID),
				oldRollup.Genesis.SystemConfig,
				newRollup.Genesis.SystemConfig,
			)
		}
	}

	return nil
}

func initCommonChecks(intent *state.Intent, st *state.State) error {
	// Ensure the state version is supported.
	if !IsSupportedStateVersion(st.Version) {
		return fmt.Errorf("unsupported state version: %d", st.Version)
	}

	if st.Create2Salt == (common.Hash{}) {
		_, err := rand.Read(st.Create2Salt[:])
		if err != nil {
			return fmt.Errorf("failed to generate CREATE2 salt: %w", err)
		}
	}

	return nil
}

func InitGenesisStrategy(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "init", "strategy", "genesis")
	lgr.Info("initializing pipeline")

	if err := initCommonChecks(intent, st); err != nil {
		return err
	}

	if intent.SuperchainRoles == nil {
		return fmt.Errorf("superchain roles must be set for genesis strategy")
	}

	// Mostly a stub for now.

	return nil
}

func immutableErr(field string, was, is any) error {
	return fmt.Errorf("%s is immutable: was %v, is %v", field, was, is)
}

func PopulateSuperchainState(env *Env, opcmAddr common.Address, superchainConfigProxy common.Address) (*addresses.SuperchainContracts, *addresses.SuperchainRoles, error) {
	input := opcm.ReadSuperchainDeploymentInput{
		SuperchainConfigProxy: superchainConfigProxy,
	}

	var out opcm.ReadSuperchainDeploymentOutput
	var err error

	if env.UseForge {
		forgeEnv := &opcm.ForgeEnv{
			Client:   env.ForgeClient,
			Context:  env.Context,
			L1RPCUrl: env.L1RPCUrl,
		}
		out, err = opcm.ReadSuperchainDeploymentViaForge(forgeEnv, input)
		if err != nil {
			return nil, nil, err
		}
	} else {
		readScript, err := opcm.NewReadSuperchainDeploymentScript(env.L1ScriptHost)
		if err != nil {
			return nil, nil, fmt.Errorf("error generating read superchain deployment script: %w", err)
		}

		out, err = readScript.Run(input)
		if err != nil {
			return nil, nil, fmt.Errorf("error reading superchain deployment: %w", err)
		}
	}

	deployment := &addresses.SuperchainContracts{
		SuperchainProxyAdminImpl: out.SuperchainProxyAdmin,
		SuperchainConfigProxy:    out.SuperchainConfigProxy,
		SuperchainConfigImpl:     out.SuperchainConfigImpl,
	}
	roles := &addresses.SuperchainRoles{
		SuperchainProxyAdminOwner: out.SuperchainProxyAdminOwner,
		SuperchainGuardian:        out.Guardian,
	}
	return deployment, roles, nil
}
