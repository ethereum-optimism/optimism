package pipeline

import (
	"context"
	"crypto/rand"
	"fmt"

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

		// The ReadSuperchainDeployment script (packages/contracts-bedrock/scripts/deploy/ReadSuperchainDeployment.s.sol)
		// uses the OPCM's semver version (>= 7.0.0 indicates v2) to determine how to populate the superchain state:
		// - OPCMv1 (< 7.0.0): Queries the OPCM contract to get SuperchainConfig and ProtocolVersions
		// - OPCMv2 (>= 7.0.0): Uses the provided SuperchainConfigProxy address; ProtocolVersions is deprecated
		superDeployment, superRoles, err := PopulateSuperchainState(env.L1ScriptHost, opcmAddr, superchainConfigAddr)
		if err != nil {
			return fmt.Errorf("error populating superchain state: %w", err)
		}
		st.SuperchainDeployment = superDeployment
		st.SuperchainRoles = superRoles

		if hasPredeployedOPCM && st.ImplementationsDeployment == nil {
			st.ImplementationsDeployment = &addresses.ImplementationsContracts{
				OpcmImpl: opcmAddr,
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

	// TODO: validate individual

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

// TODO(#18612): Remove OPCMAddress field when OPCMv1 gets deprecated
// TODO(#18612): Remove ProtocolVersions fields when OPCMv1 gets deprecated
func PopulateSuperchainState(host *script.Host, opcmAddr common.Address, superchainConfigProxy common.Address) (*addresses.SuperchainContracts, *addresses.SuperchainRoles, error) {
	readScript, err := opcm.NewReadSuperchainDeploymentScript(host)
	if err != nil {
		return nil, nil, fmt.Errorf("error generating read superchain deployment script: %w", err)
	}

	out, err := readScript.Run(opcm.ReadSuperchainDeploymentInput{
		OPCMAddress:           opcmAddr,
		SuperchainConfigProxy: superchainConfigProxy,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error reading superchain deployment: %w", err)
	}

	deployment := &addresses.SuperchainContracts{
		SuperchainProxyAdminImpl: out.SuperchainProxyAdmin,
		SuperchainConfigProxy:    out.SuperchainConfigProxy,
		SuperchainConfigImpl:     out.SuperchainConfigImpl,
		ProtocolVersionsProxy:    out.ProtocolVersionsProxy,
		ProtocolVersionsImpl:     out.ProtocolVersionsImpl,
	}
	roles := &addresses.SuperchainRoles{
		SuperchainProxyAdminOwner: out.SuperchainProxyAdminOwner,
		SuperchainGuardian:        out.Guardian,
		ProtocolVersionsOwner:     out.ProtocolVersionsOwner,
	}
	return deployment, roles, nil
}
