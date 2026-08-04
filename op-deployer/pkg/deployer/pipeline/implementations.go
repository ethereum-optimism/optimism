package pipeline

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

const (
	genesisMockSP1VerifierArtifact = "MockSP1Verifier.sol"
	genesisMockSP1VerifierContract = "MockSP1Verifier"
)

var sp1VerifierMethod = w3.MustNewFunc("sp1Verifier()", "address")

type sp1VerifierOverride struct {
	SP1Verifier common.Address `json:"sp1Verifier"`
}

func DeployImplementations(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-implementations")
	if env.DeployMockSP1Verifier && !env.IsGenesis {
		return fmt.Errorf("mock SP1 verifier deployment is only supported for genesis")
	}

	requestedSP1Verifier, hasSP1VerifierOverride, err := parseSP1VerifierOverride(intent.GlobalDeployOverrides)
	if err != nil {
		return err
	}
	if hasSP1VerifierOverride && requestedSP1Verifier == (common.Address{}) {
		return fmt.Errorf("sp1Verifier override must not be zero")
	}
	if intent.OPCMAddress != nil && hasSP1VerifierOverride {
		return fmt.Errorf("sp1Verifier must not be specified when using a predeployed OPCM")
	}

	proofParams, err := jsonutil.MergeJSON(
		state.SuperchainProofParams{
			WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
			MinProposalSizeBytes:            standard.MinProposalSizeBytes,
			ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
			ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
			DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
			DisputeMaxGameDepth:             standard.DisputeMaxGameDepth,
			DisputeSplitDepth:               standard.DisputeSplitDepth,
			DisputeClockExtension:           standard.DisputeClockExtension,
			DisputeMaxClockDuration:         standard.DisputeMaxClockDuration,
			MIPSVersion:                     standard.MIPSVersion,
			DevFeatureBitmap:                common.Hash{},
		},
		intent.GlobalDeployOverrides,
	)
	if err != nil {
		return fmt.Errorf("error merging proof params from overrides: %w", err)
	}
	zkEnabled := devfeatures.IsDevFeatureEnabled(proofParams.DevFeatureBitmap, devfeatures.ZKDisputeGameFlag)

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
		} else if intent.OPCMAddress == nil && env.IsGenesis && zkEnabled {
			if !env.DeployMockSP1Verifier {
				return fmt.Errorf("sp1Verifier must be specified when ZK dispute games are enabled")
			}
			if st.SP1Verifier == nil || *st.SP1Verifier == (common.Address{}) {
				return fmt.Errorf("reused ZK implementations do not record an SP1 verifier")
			}
			recordSP1VerifierOverride(intent, *st.SP1Verifier)
		}
		lgr.Info("implementations deployment not needed")
		return nil
	}

	lgr.Info("deploying implementations")

	var dio opcm.DeployImplementationsOutput
	input := opcm.DeployImplementationsInput{
		WithdrawalDelaySeconds:          new(big.Int).SetUint64(proofParams.WithdrawalDelaySeconds),
		MinProposalSizeBytes:            new(big.Int).SetUint64(proofParams.MinProposalSizeBytes),
		ChallengePeriodSeconds:          new(big.Int).SetUint64(proofParams.ChallengePeriodSeconds),
		ProofMaturityDelaySeconds:       new(big.Int).SetUint64(proofParams.ProofMaturityDelaySeconds),
		DisputeGameFinalityDelaySeconds: new(big.Int).SetUint64(proofParams.DisputeGameFinalityDelaySeconds),
		MipsVersion:                     new(big.Int).SetUint64(proofParams.MIPSVersion),
		DevFeatureBitmap:                proofParams.DevFeatureBitmap,
		FaultGameV2MaxGameDepth:         new(big.Int).SetUint64(proofParams.DisputeMaxGameDepth),
		FaultGameV2SplitDepth:           new(big.Int).SetUint64(proofParams.DisputeSplitDepth),
		FaultGameV2ClockExtension:       new(big.Int).SetUint64(proofParams.DisputeClockExtension),
		FaultGameV2MaxClockDuration:     new(big.Int).SetUint64(proofParams.DisputeMaxClockDuration),
		SuperchainConfigProxy:           st.SuperchainDeployment.SuperchainConfigProxy,
		SuperchainProxyAdmin:            st.SuperchainDeployment.SuperchainProxyAdminImpl,
		L1ProxyAdminOwner:               st.SuperchainRoles.SuperchainProxyAdminOwner,
		Challenger:                      st.SuperchainRoles.Challenger,
		SP1Verifier:                     proofParams.SP1Verifier,
	}
	if !zkEnabled && input.SP1Verifier != (common.Address{}) {
		return fmt.Errorf("sp1Verifier must not be specified when ZK dispute games are disabled")
	}
	if zkEnabled && input.SP1Verifier == (common.Address{}) {
		if !env.DeployMockSP1Verifier {
			return fmt.Errorf("sp1Verifier must be specified when ZK dispute games are enabled")
		}
		input.SP1Verifier, err = deployAndRecordGenesisMockSP1Verifier(env, intent)
		if err != nil {
			return err
		}
	}

	if env.UseForge {
		lgr.Info("using Forge for DeployImplementations")
		forgeEnv := &opcm.ForgeEnv{
			Client:     env.ForgeClient,
			Context:    env.Context,
			L1RPCUrl:   env.L1RPCUrl,
			PrivateKey: env.PrivateKey,
		}
		dio, err = opcm.DeployImplementationsViaForge(forgeEnv, input)
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
		SP1PlonkAdapterImpl:              dio.SP1PlonkAdapter,
		SuperFaultDisputeGameImpl:        dio.SuperFaultDisputeGameImpl,
		SuperPermissionedDisputeGameImpl: dio.SuperPermissionedDisputeGameImpl,
	}
	if input.SP1Verifier != (common.Address{}) {
		st.SP1Verifier = &input.SP1Verifier
	}

	return nil
}

func parseSP1VerifierOverride(overrides map[string]any) (common.Address, bool, error) {
	var value any
	found := false
	for key, candidate := range overrides {
		if !strings.EqualFold(key, "sp1Verifier") {
			continue
		}
		if found {
			return common.Address{}, false, fmt.Errorf("sp1Verifier is specified more than once")
		}
		value = candidate
		found = true
	}
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

func deployGenesisMockSP1Verifier(env *Env) (common.Address, error) {
	if env.L1ScriptHost == nil {
		return common.Address{}, fmt.Errorf("cannot deploy genesis MockSP1Verifier without an L1 script host")
	}
	artifact, err := env.L1ScriptHost.Artifacts().ReadArtifact(genesisMockSP1VerifierArtifact, genesisMockSP1VerifierContract)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to read genesis MockSP1Verifier artifact: %w", err)
	}
	verifier, err := env.L1ScriptHost.Create(env.Deployer, artifact.Bytecode.Object)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to deploy genesis MockSP1Verifier: %w", err)
	}
	if verifier == (common.Address{}) {
		return common.Address{}, fmt.Errorf("genesis MockSP1Verifier deployment produced no contract address")
	}
	return verifier, nil
}

func deployAndRecordGenesisMockSP1Verifier(env *Env, intent *state.Intent) (common.Address, error) {
	verifier, err := deployGenesisMockSP1Verifier(env)
	if err != nil {
		return common.Address{}, err
	}
	recordSP1VerifierOverride(intent, verifier)
	return verifier, nil
}

func recordSP1VerifierOverride(intent *state.Intent, verifier common.Address) {
	if intent.GlobalDeployOverrides == nil {
		intent.GlobalDeployOverrides = make(map[string]any)
	}
	intent.GlobalDeployOverrides["sp1Verifier"] = verifier
}

func shouldDeployImplementations(intent *state.Intent, st *state.State) bool {
	return st.ImplementationsDeployment == nil
}
