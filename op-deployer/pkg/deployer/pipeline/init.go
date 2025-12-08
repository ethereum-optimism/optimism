package pipeline

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

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

	var opcmV2Enabled bool
	if devFeatureBitmap, ok := intent.GlobalDeployOverrides["devFeatureBitmap"].(common.Hash); ok {
		opcmV2Flag := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")
		opcmV2Enabled = isDevFeatureEnabled(devFeatureBitmap, opcmV2Flag)
	}

	if hasPredeployedOPCM && !opcmV2Enabled {
		if intent.SuperchainConfigProxy != nil {
			return fmt.Errorf("cannot set superchain config proxy for predeployed OPCM")
		}

		if intent.SuperchainRoles != nil {
			return fmt.Errorf("cannot set superchain roles for predeployed OPCM")
		}

		// Use OPCMv1 to populate superchain state
		superDeployment, superRoles, err := PopulateSuperchainState(env.L1ScriptHost, *intent.OPCMAddress, common.Address{})
		if err != nil {
			return fmt.Errorf("error populating superchain state: %w", err)
		}
		st.SuperchainDeployment = superDeployment
		st.SuperchainRoles = superRoles
		if st.ImplementationsDeployment == nil {
			st.ImplementationsDeployment = &addresses.ImplementationsContracts{
				OpcmImpl: *intent.OPCMAddress,
			}
		}
	}
	hasSuperchainConfigProxy := intent.SuperchainConfigProxy != nil
	if hasSuperchainConfigProxy && opcmV2Enabled {
		if intent.SuperchainRoles != nil {
			return fmt.Errorf("cannot set superchain roles for superchain config proxy")
		}

		// Use SuperchainConfigProxy to populate superchain state
		superDeployment, superRoles, err := PopulateSuperchainState(env.L1ScriptHost, common.Address{}, *intent.SuperchainConfigProxy)
		if err != nil {
			return fmt.Errorf("error populating superchain state: %w", err)
		}
		st.SuperchainDeployment = superDeployment
		st.SuperchainRoles = superRoles
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

// TODO: Remove OPCMAddress field when OPCMv1 gets deprecated
// TODO: Remove ProtocolVersions fields when OPCMv1 gets deprecated
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

// isDevFeatureEnabled checks if a specific development feature is enabled in a feature bitmap.
// This mirrors the function in devfeatures.go to avoid import cycles.
func isDevFeatureEnabled(bitmap, flag common.Hash) bool {
	b := new(big.Int).SetBytes(bitmap[:])
	f := new(big.Int).SetBytes(flag[:])

	featuresIsNonZero := f.Cmp(big.NewInt(0)) != 0
	bitmapContainsFeatures := new(big.Int).And(b, f).Cmp(f) == 0
	return featuresIsNonZero && bitmapContainsFeatures
}
