package pipeline

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lmittmann/w3"
)

func IsSupportedStateVersion(version int) bool {
	return version == 1
}

func ValidateInputs(intent *state.Intent, st *state.State) error {
	if err := intent.Check(); err != nil {
		return err
	}
	if intent.OPCMAddress != nil {
		if _, ok := intent.GlobalDeployOverrides["sp1Verifier"]; ok {
			return fmt.Errorf("sp1Verifier must not be specified when using a predeployed OPCM")
		}
	}
	if !IsSupportedStateVersion(st.Version) {
		return fmt.Errorf("unsupported state version: %d", st.Version)
	}
	return nil
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
			resolved, err := resolveSuperchainConfig(ctx, env.L1Client, opcmAddr)
			if err != nil {
				return fmt.Errorf("error resolving SuperchainConfig from OPCM at %s: %w", opcmAddr, err)
			}
			superchainConfigAddr = resolved
		}
		superDeployment, superRoles, err := PopulateSuperchainState(ctx, env.L1Client, superchainConfigAddr)
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

	// TODO: validate individual

	return nil
}

func initCommonChecks(intent *state.Intent, st *state.State) error {
	// Ensure the state version is supported.
	if !IsSupportedStateVersion(st.Version) {
		return fmt.Errorf("unsupported state version: %d", st.Version)
	}

	if err := st.EnsureCreate2Salt(); err != nil {
		return err
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

func resolveSuperchainConfig(ctx context.Context, client *ethclient.Client, opcmAddr common.Address) (common.Address, error) {
	opcmContract := opcm.NewContract(opcmAddr, client)
	validatorAddr, err := opcmContract.OPCMStandardValidator(ctx)
	if err == nil {
		return opcm.NewContract(validatorAddr, client).SuperchainConfig(ctx)
	}
	return opcmContract.SuperchainConfig(ctx)
}

var (
	proxyAdminMethod          = w3.MustNewFunc("admin()", "address")
	proxyImplementationMethod = w3.MustNewFunc("implementation()", "address")
	guardianMethod            = w3.MustNewFunc("guardian()", "address")
	ownerMethod               = w3.MustNewFunc("owner()", "address")
)

// PopulateSuperchainState reads the superchain deployment behind a SuperchainConfig proxy. OP's
// Proxy answers admin() and implementation() only when the sender is its admin or address(0), and
// CallGetter sends the call from address(0).
func PopulateSuperchainState(ctx context.Context, client bind.ContractCaller, superchainConfigProxy common.Address) (*addresses.SuperchainContracts, *addresses.SuperchainRoles, error) {
	code, err := client.CodeAt(ctx, superchainConfigProxy, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the code of the SuperchainConfig proxy %s: %w", superchainConfigProxy, err)
	}
	if len(code) == 0 {
		return nil, nil, fmt.Errorf("the SuperchainConfig proxy %s has no code", superchainConfigProxy)
	}
	proxyAdmin, err := opcm.CallGetter[common.Address](ctx, client, superchainConfigProxy, proxyAdminMethod)
	if err != nil {
		return nil, nil, err
	}
	impl, err := opcm.CallGetter[common.Address](ctx, client, superchainConfigProxy, proxyImplementationMethod)
	if err != nil {
		return nil, nil, err
	}
	guardian, err := opcm.CallGetter[common.Address](ctx, client, superchainConfigProxy, guardianMethod)
	if err != nil {
		return nil, nil, err
	}
	owner, err := opcm.CallGetter[common.Address](ctx, client, proxyAdmin, ownerMethod)
	if err != nil {
		return nil, nil, err
	}

	deployment := &addresses.SuperchainContracts{
		SuperchainProxyAdminImpl: proxyAdmin,
		SuperchainConfigProxy:    superchainConfigProxy,
		SuperchainConfigImpl:     impl,
	}
	roles := &addresses.SuperchainRoles{
		SuperchainProxyAdminOwner: owner,
		SuperchainGuardian:        guardian,
	}
	return deployment, roles, nil
}
