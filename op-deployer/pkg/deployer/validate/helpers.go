package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-validator/pkg/service"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// DetectValidatorVersion determines the validator version to use based on the validate flag and state.
func DetectValidatorVersion(validateFlag string, appliedIntent *state.Intent, l log.Logger) string {
	if validateFlag == "auto" {
		if appliedIntent.L1ContractsLocator != nil {
			locatorBytes, err := appliedIntent.L1ContractsLocator.MarshalText()
			if err == nil {
				locatorStr := string(locatorBytes)
				if strings.HasPrefix(locatorStr, "tag://") {
					version := strings.TrimPrefix(locatorStr, "tag://")
					l.Info("Auto-detected version from state.json", "version", version)
					return version
				}
				l.Info("Using current standard tag (non-tag locator found)", "version", standard.CurrentTag)
				return standard.CurrentTag
			}
			locatorStr := appliedIntent.L1ContractsLocator.URL.String()
			if strings.HasPrefix(locatorStr, "tag://") {
				version := strings.TrimPrefix(locatorStr, "tag://")
				l.Info("Auto-detected version from state.json", "version", version)
				return version
			}
			l.Info("Using current standard tag", "version", standard.CurrentTag)
			return standard.CurrentTag
		}
		l.Info("Using current standard tag (no L1ContractsLocator found)", "version", standard.CurrentTag)
		return standard.CurrentTag
	}
	if !strings.HasPrefix(validateFlag, "op-contracts/") {
		return "op-contracts/" + validateFlag
	}
	return validateFlag
}

// GetAbsolutePrestate retrieves the absolute prestate for a chain from various sources.
func GetAbsolutePrestate(globalState *state.State, chainID common.Hash, l log.Logger) common.Hash {
	if globalState.PrestateManifest != nil {
		if hash, ok := (*globalState.PrestateManifest)[chainID.Big().String()]; ok {
			return common.HexToHash(hash)
		}
	}
	if globalState.AppliedIntent != nil {
		if chainIntent, err := globalState.AppliedIntent.Chain(chainID); err == nil {
			if proofParams, ok := chainIntent.DeployOverrides["faultGameAbsolutePrestate"]; ok {
				if hashStr, ok := proofParams.(string); ok {
					return common.HexToHash(hashStr)
				}
			}
		}
	}
	l.Info("Using standard absolute prestate (prestate not found in state)", "prestate", standard.DisputeAbsolutePrestate.Hex())
	return standard.DisputeAbsolutePrestate
}

// GetOPCMValidatorAddress retrieves the validator address from the OPCM contract.
func GetOPCMValidatorAddress(ctx context.Context, l log.Logger, opcmAddr *common.Address, l1RPCURL string) common.Address {
	if opcmAddr == nil {
		return common.Address{}
	}
	ethClient, err := ethclient.DialContext(ctx, l1RPCURL)
	if err != nil {
		l.Warn("Failed to dial L1 RPC to get validator address", "error", err)
		return common.Address{}
	}
	defer ethClient.Close()

	opcmContract := opcm.NewContract(*opcmAddr, ethClient)
	validatorAddr, err := opcmContract.OPCMStandardValidator(ctx)
	if err != nil {
		l.Warn("Failed to get OPCMStandardValidator address from OPCM", "opcm", opcmAddr.Hex(), "error", err)
		return common.Address{}
	}
	if validatorAddr == (common.Address{}) {
		l.Warn("OPCMStandardValidator address is zero", "opcm", opcmAddr.Hex())
		return common.Address{}
	}
	l.Info("Found OPCMStandardValidator address from OPCM", "validator", validatorAddr.Hex(), "opcm", opcmAddr.Hex())
	return validatorAddr
}

// BuildValidatorConfigFromState builds a validator configuration from the deployment state.
func BuildValidatorConfigFromState(ctx context.Context, l log.Logger, globalState *state.State, chainState *state.ChainState, chainID common.Hash, l1RPCURL string) (*service.Config, error) {
	validatorCfg := &service.Config{
		L1RPCURL: l1RPCURL,
	}

	validatorCfg.AbsolutePrestate = GetAbsolutePrestate(globalState, chainID, l)

	if chainState.OpChainProxyAdminImpl == (common.Address{}) {
		return nil, fmt.Errorf("proxy-admin not found in state for chain %s", chainID.Hex())
	}
	validatorCfg.ProxyAdmin = chainState.OpChainProxyAdminImpl

	if chainState.SystemConfigProxy == (common.Address{}) {
		return nil, fmt.Errorf("system-config not found in state for chain %s", chainID.Hex())
	}
	validatorCfg.SystemConfig = chainState.SystemConfigProxy

	validatorCfg.L2ChainID = chainID.Big()

	if globalState.AppliedIntent != nil {
		if chainIntent, err := globalState.AppliedIntent.Chain(chainID); err == nil {
			validatorCfg.Proposer = chainIntent.Roles.Proposer
			if validatorAddr := GetOPCMValidatorAddress(ctx, l, globalState.AppliedIntent.OPCMAddress, l1RPCURL); validatorAddr != (common.Address{}) {
				validatorCfg.ValidatorAddress = validatorAddr
			}
		}
	}

	return validatorCfg, nil
}
