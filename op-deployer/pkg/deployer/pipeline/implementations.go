package pipeline

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/lmittmann/w3"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

var sp1VerifierMethod = w3.MustNewFunc("sp1Verifier()", "address")

type sp1VerifierOverride struct {
	SP1Verifier common.Address `json:"sp1Verifier"`
}

// ResolveSuperchainProofParams merges the standard superchain-level proof parameters with the
// intent's global deploy overrides. These are the inputs the implementations are deployed with;
// per-chain values are resolved by ResolveChainProofParams.
func ResolveSuperchainProofParams(intent *state.Intent) (state.SuperchainProofParams, error) {
	return jsonutil.MergeJSON(
		state.SuperchainProofParams{
			MinProposalSizeBytes:               standard.MinProposalSizeBytes,
			ChallengePeriodSeconds:             standard.ChallengePeriodSeconds,
			MinProofMaturityDelaySeconds:       standard.MinProofMaturityDelaySeconds,
			MaxProofMaturityDelaySeconds:       standard.MaxProofMaturityDelaySeconds,
			MinDisputeGameFinalityDelaySeconds: standard.MinDisputeGameFinalityDelaySeconds,
			MaxDisputeGameFinalityDelaySeconds: standard.MaxDisputeGameFinalityDelaySeconds,
			MinWithdrawalDelaySeconds:          standard.MinWithdrawalDelaySeconds,
			MaxWithdrawalDelaySeconds:          standard.MaxWithdrawalDelaySeconds,
			DisputeMaxGameDepth:                standard.DisputeMaxGameDepth,
			DisputeSplitDepth:                  standard.DisputeSplitDepth,
			DisputeClockExtension:              standard.DisputeClockExtension,
			DisputeMaxClockDuration:            standard.DisputeMaxClockDuration,
			MIPSVersion:                        standard.MIPSVersion,
			DevFeatureBitmap:                   common.Hash{},
		},
		intent.GlobalDeployOverrides,
	)
}

func DeployImplementations(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-implementations")
	if env.DeployMockSP1Verifier {
		if err := CheckMockSP1Verifier(intent.L1ChainID); err != nil {
			return err
		}
	}

	requestedSP1Verifier, hasSP1VerifierOverride, err := parseSP1VerifierOverride(intent.GlobalDeployOverrides)
	if err != nil {
		return err
	}
	if hasSP1VerifierOverride && requestedSP1Verifier == (common.Address{}) {
		return fmt.Errorf("sp1Verifier override must not be zero")
	}
	proofParams, err := ResolveSuperchainProofParams(intent)
	if err != nil {
		return fmt.Errorf("error merging proof params from overrides: %w", err)
	}
	selectedSP1Verifier := requestedSP1Verifier
	if intent.OPCMAddress == nil && !hasSP1VerifierOverride && !env.DeployMockSP1Verifier {
		// A generated genesis contains no release-approved verifier, so it must be selected explicitly.
		if env.IsGenesis {
			return fmt.Errorf("sp1Verifier must be specified for genesis deployments; for development, " +
				"set ApplyPipelineOpts.DeployMockSP1Verifier (CLI: --deploy-mock-sp1-verifier)")
		}
		selectedSP1Verifier, err = standard.SP1VerifierFor(intent.L1ChainID)
		if err != nil {
			return fmt.Errorf("sp1Verifier must be specified on L1 chain ID %d (%w); for a development L1, "+
				"set ApplyPipelineOpts.DeployMockSP1Verifier (CLI: --deploy-mock-sp1-verifier)", intent.L1ChainID, err)
		}
	}

	if !shouldDeployImplementations(intent, st) {
		if hasSP1VerifierOverride {
			if env.IsGenesis {
				if st.SP1Verifier == nil || *st.SP1Verifier == (common.Address{}) {
					return fmt.Errorf("reused genesis implementations do not record an SP1 verifier")
				}
				if requestedSP1Verifier != *st.SP1Verifier {
					return fmt.Errorf("sp1Verifier %s does not match %s recorded for the reused implementations", requestedSP1Verifier, *st.SP1Verifier)
				}
			} else {
				if err := validateReusedSP1Verifier(env, st.ImplementationsDeployment.SP1PlonkAdapterImpl, requestedSP1Verifier); err != nil {
					return err
				}
			}
		} else if intent.OPCMAddress == nil {
			if st.SP1Verifier == nil || *st.SP1Verifier == (common.Address{}) {
				return fmt.Errorf("reused implementations do not record an SP1 verifier")
			}
		}
		lgr.Info("implementations deployment not needed")
		return nil
	}

	lgr.Info("deploying implementations")

	var dio opcm.DeployImplementationsOutput
	input := opcm.DeployImplementationsInput{
		MinWithdrawalDelaySeconds:          new(big.Int).SetUint64(proofParams.MinWithdrawalDelaySeconds),
		MaxWithdrawalDelaySeconds:          new(big.Int).SetUint64(proofParams.MaxWithdrawalDelaySeconds),
		MinProposalSizeBytes:               new(big.Int).SetUint64(proofParams.MinProposalSizeBytes),
		ChallengePeriodSeconds:             new(big.Int).SetUint64(proofParams.ChallengePeriodSeconds),
		MinProofMaturityDelaySeconds:       new(big.Int).SetUint64(proofParams.MinProofMaturityDelaySeconds),
		MaxProofMaturityDelaySeconds:       new(big.Int).SetUint64(proofParams.MaxProofMaturityDelaySeconds),
		MinDisputeGameFinalityDelaySeconds: new(big.Int).SetUint64(proofParams.MinDisputeGameFinalityDelaySeconds),
		MaxDisputeGameFinalityDelaySeconds: new(big.Int).SetUint64(proofParams.MaxDisputeGameFinalityDelaySeconds),
		MipsVersion:                        new(big.Int).SetUint64(proofParams.MIPSVersion),
		DevFeatureBitmap:                   proofParams.DevFeatureBitmap,
		FaultGameV2MaxGameDepth:            new(big.Int).SetUint64(proofParams.DisputeMaxGameDepth),
		FaultGameV2SplitDepth:              new(big.Int).SetUint64(proofParams.DisputeSplitDepth),
		FaultGameV2ClockExtension:          new(big.Int).SetUint64(proofParams.DisputeClockExtension),
		FaultGameV2MaxClockDuration:        new(big.Int).SetUint64(proofParams.DisputeMaxClockDuration),
		SuperchainConfigProxy:              st.SuperchainDeployment.SuperchainConfigProxy,
		SuperchainProxyAdmin:               st.SuperchainDeployment.SuperchainProxyAdminImpl,
		L1ProxyAdminOwner:                  st.SuperchainRoles.SuperchainProxyAdminOwner,
		Challenger:                         st.SuperchainRoles.Challenger,
		SP1Verifier:                        selectedSP1Verifier,
	}
	if env.DeployMockSP1Verifier && hasSP1VerifierOverride {
		lgr.Warn("not deploying the mock SP1 verifier: the sp1Verifier override takes precedence",
			"sp1Verifier", requestedSP1Verifier)
	}
	if env.DeployMockSP1Verifier && input.SP1Verifier == (common.Address{}) {
		input.SP1Verifier, err = deployMockSP1Verifier(env)
		if err != nil {
			return err
		}
	}
	lgr.Info("using SP1 verifier", "address", input.SP1Verifier)

	if env.Forge != nil {
		lgr.Info("using Forge for DeployImplementations")
		dio, err = opcm.DeployImplementationsViaForge(env.Forge, input)
		if err != nil {
			return err
		}
	} else {
		dio, err = env.Scripts.DeployImplementations.Run(input)
		if err != nil {
			return fmt.Errorf("error deploying implementations: %w", err)
		}
	}

	st.ImplementationsDeployment = &addresses.ImplementationsContracts{
		OpcmStandardValidatorImpl:        dio.OpcmStandardValidator,
		OpcmUtilsImpl:                    dio.OpcmUtils,
		OpcmMigratorImpl:                 dio.OpcmMigrator,
		OpcmV2Impl:                       dio.OpcmV2,
		OpcmContainerImpl:                dio.OpcmContainer,
		DelayedWethImpl:                  dio.DelayedWETHImpl,
		OptimismPortalImpl:               dio.OptimismPortalImpl,
		EthLockboxImpl:                   dio.ETHLockboxImpl,
		PreimageOracleImpl:               dio.PreimageOracleSingleton,
		MipsImpl:                         dio.MipsSingleton,
		SystemConfigImpl:                 dio.SystemConfigImpl,
		L1CrossDomainMessengerImpl:       dio.L1CrossDomainMessengerImpl,
		L1Erc721BridgeImpl:               dio.L1ERC721BridgeImpl,
		L1StandardBridgeImpl:             dio.L1StandardBridgeImpl,
		OptimismMintableErc20FactoryImpl: dio.OptimismMintableERC20FactoryImpl,
		DisputeGameFactoryImpl:           dio.DisputeGameFactoryImpl,
		AnchorStateRegistryImpl:          dio.AnchorStateRegistryImpl,
		FaultDisputeGameImpl:             dio.FaultDisputeGameImpl,
		PermissionedDisputeGameImpl:      dio.PermissionedDisputeGameImpl,
		ZkDisputeGameImpl:                dio.ZkDisputeGameImpl,
		StorageSetterImpl:                dio.StorageSetterImpl,
		SP1PlonkAdapterImpl:              dio.SP1PlonkAdapterSingleton,
		SuperFaultDisputeGameImpl:        dio.SuperFaultDisputeGameImpl,
		SuperPermissionedDisputeGameImpl: dio.SuperPermissionedDisputeGameImpl,
	}
	if input.SP1Verifier != (common.Address{}) {
		st.SP1Verifier = &input.SP1Verifier
	}

	return nil
}

func parseSP1VerifierOverride(overrides map[string]any) (common.Address, bool, error) {
	value, found := overrides["sp1Verifier"]
	if !found {
		return common.Address{}, false, nil
	}

	parsed, err := jsonutil.MergeJSON(sp1VerifierOverride{}, map[string]any{"sp1Verifier": value})
	if err != nil {
		return common.Address{}, false, fmt.Errorf("invalid sp1Verifier override: %w", err)
	}
	return parsed.SP1Verifier, true, nil
}

func validateReusedSP1Verifier(env *Env, adapter common.Address, requested common.Address) error {
	if adapter == (common.Address{}) {
		if requested != (common.Address{}) {
			return fmt.Errorf("sp1Verifier %s does not match reused implementations without an SP1PlonkAdapter", requested)
		}
		return nil
	}

	deployed, err := readReusedSP1Verifier(env, adapter)
	if err != nil {
		return err
	}
	if requested != deployed {
		return fmt.Errorf("sp1Verifier %s does not match %s used by reused SP1PlonkAdapter %s", requested, deployed, adapter)
	}
	return nil
}

func readReusedSP1Verifier(env *Env, adapter common.Address) (common.Address, error) {
	if adapter == (common.Address{}) {
		return common.Address{}, fmt.Errorf("cannot read sp1Verifier from a zero SP1PlonkAdapter address")
	}

	var backend opcm.CallContractBackend
	switch {
	case env.L1Client != nil:
		backend = env.L1Client
	case env.L1ScriptHost != nil:
		backend = opcm.NewScriptHostCallBackend(env.L1ScriptHost)
	default:
		return common.Address{}, fmt.Errorf("cannot read sp1Verifier from reused implementations without an L1 call backend")
	}

	ctx := env.Context
	if ctx == nil {
		ctx = context.Background()
	}
	calldata, err := sp1VerifierMethod.EncodeArgs()
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to encode SP1PlonkAdapter.sp1Verifier call: %w", err)
	}
	result, err := backend.CallContract(ctx, ethereum.CallMsg{To: &adapter, Data: calldata}, nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to read sp1Verifier from reused SP1PlonkAdapter %s: %w", adapter, err)
	}
	var deployed common.Address
	if err := sp1VerifierMethod.DecodeReturns(result, &deployed); err != nil {
		return common.Address{}, fmt.Errorf("failed to decode sp1Verifier from reused SP1PlonkAdapter %s: %w", adapter, err)
	}
	return deployed, nil
}

// CheckMockSP1Verifier refuses the mock SP1 verifier on an L1 that has a release verifier, for
// every deployment target.
func CheckMockSP1Verifier(l1ChainID uint64) error {
	if _, err := standard.SP1VerifierFor(l1ChainID); err == nil {
		return fmt.Errorf("L1 chain ID %d has a release SP1 verifier; the mock SP1 verifier is only for development L1s", l1ChainID)
	}
	return nil
}

// deployMockSP1Verifier deploys MockSP1Verifier: into the genesis state of forge or the L1 script
// host, or to the live L1 with forge's deployer or the broadcaster of the L1 script host.
func deployMockSP1Verifier(env *Env) (common.Address, error) {
	if !env.IsGenesis {
		env.Logger.Warn("deploying the accept-all mock SP1 verifier to a live L1; development use only")
		if env.Forge != nil {
			return opcm.DeployMockSP1VerifierLive(env.Forge)
		}
	} else if env.Forge != nil {
		return opcm.DeployMockSP1VerifierGenesis(env.Forge)
	}
	host := env.L1ScriptHost
	if host == nil {
		return common.Address{}, fmt.Errorf("cannot deploy MockSP1Verifier without an L1 script host")
	}
	nonce := host.GetNonce(env.Deployer)
	verifier, initCode, err := opcm.DeployMockSP1Verifier(host, env.Deployer)
	if err != nil || env.IsGenesis {
		return verifier, err
	}
	ctx := env.Context
	if ctx == nil {
		ctx = context.Background()
	}
	env.Broadcaster.Hook(broadcaster.Broadcast{
		From:  env.Deployer,
		To:    verifier,
		Input: initCode,
		Value: (*hexutil.U256)(uint256.NewInt(0)),
		Type:  broadcaster.BroadcastCreate,
		Nonce: nonce,
	})
	results, err := env.Broadcaster.Broadcast(ctx)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to send MockSP1Verifier deployment: %w", err)
	}
	if len(results) == 0 {
		return common.Address{}, fmt.Errorf("MockSP1Verifier deployment was not sent")
	}
	if deployed := results[len(results)-1].Receipt.ContractAddress; deployed != verifier {
		return common.Address{}, fmt.Errorf("MockSP1Verifier deployed at %s, expected %s", deployed, verifier)
	}
	return verifier, nil
}

func shouldDeployImplementations(intent *state.Intent, st *state.State) bool {
	return st.ImplementationsDeployment == nil
}
