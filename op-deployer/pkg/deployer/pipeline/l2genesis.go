package pipeline

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	"github.com/ethereum/go-ethereum/common"
)

type l2GenesisOverrides struct {
	FundDevAccounts                          bool                      `json:"fundDevAccounts"`
	BaseFeeVaultMinimumWithdrawalAmount      *hexutil.Big              `json:"baseFeeVaultMinimumWithdrawalAmount"`
	L1FeeVaultMinimumWithdrawalAmount        *hexutil.Big              `json:"l1FeeVaultMinimumWithdrawalAmount"`
	SequencerFeeVaultMinimumWithdrawalAmount *hexutil.Big              `json:"sequencerFeeVaultMinimumWithdrawalAmount"`
	OperatorFeeVaultMinimumWithdrawalAmount  *hexutil.Big              `json:"operatorFeeVaultMinimumWithdrawalAmount"`
	BaseFeeVaultWithdrawalNetwork            genesis.WithdrawalNetwork `json:"baseFeeVaultWithdrawalNetwork"`
	L1FeeVaultWithdrawalNetwork              genesis.WithdrawalNetwork `json:"l1FeeVaultWithdrawalNetwork"`
	SequencerFeeVaultWithdrawalNetwork       genesis.WithdrawalNetwork `json:"sequencerFeeVaultWithdrawalNetwork"`
	OperatorFeeVaultWithdrawalNetwork        genesis.WithdrawalNetwork `json:"operatorFeeVaultWithdrawalNetwork"`
	EnableGovernance                         bool                      `json:"enableGovernance"`
	GovernanceTokenOwner                     common.Address            `json:"governanceTokenOwner"`
}

type cgtConfig struct {
	UseCustomGasToken          bool
	GasPayingTokenName         string
	GasPayingTokenSymbol       string
	NativeAssetLiquidityAmount *big.Int
	LiquidityControllerOwner   common.Address
}

func GenerateL2Genesis(pEnv *Env, intent *state.Intent, bundle ArtifactsBundle, st *state.State, chainID common.Hash) error {
	lgr := pEnv.Logger.New("stage", "generate-l2-genesis")

	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain intent: %w", err)
	}

	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	if !shouldGenerateL2Genesis(thisChainState) {
		lgr.Info("L2 genesis generation not needed")
		return nil
	}

	lgr.Info("generating L2 genesis", "id", chainID.Hex())

	overrides, schedule, err := calculateL2GenesisOverrides(intent, thisIntent)
	if err != nil {
		return fmt.Errorf("failed to calculate L2 genesis overrides: %w", err)
	}

	cgt := buildCGTConfig(thisIntent)

	devFeatureBitmap, err := buildDevFeatureBitmap(intent)

	if err != nil {
		return err
	}

	input := opcm.L2GenesisInput{
		L1ChainID:                                new(big.Int).SetUint64(intent.L1ChainID),
		L2ChainID:                                chainID.Big(),
		L1CrossDomainMessengerProxy:              thisChainState.L1CrossDomainMessengerProxy,
		L1StandardBridgeProxy:                    thisChainState.L1StandardBridgeProxy,
		L1ERC721BridgeProxy:                      thisChainState.L1Erc721BridgeProxy,
		OpChainProxyAdminOwner:                   thisIntent.Roles.L2ProxyAdminOwner,
		BaseFeeVaultWithdrawalNetwork:            wdNetworkToBig(overrides.BaseFeeVaultWithdrawalNetwork),
		L1FeeVaultWithdrawalNetwork:              wdNetworkToBig(overrides.L1FeeVaultWithdrawalNetwork),
		SequencerFeeVaultWithdrawalNetwork:       wdNetworkToBig(overrides.SequencerFeeVaultWithdrawalNetwork),
		OperatorFeeVaultWithdrawalNetwork:        wdNetworkToBig(overrides.OperatorFeeVaultWithdrawalNetwork),
		SequencerFeeVaultMinimumWithdrawalAmount: overrides.SequencerFeeVaultMinimumWithdrawalAmount.ToInt(),
		BaseFeeVaultMinimumWithdrawalAmount:      overrides.BaseFeeVaultMinimumWithdrawalAmount.ToInt(),
		L1FeeVaultMinimumWithdrawalAmount:        overrides.L1FeeVaultMinimumWithdrawalAmount.ToInt(),
		OperatorFeeVaultMinimumWithdrawalAmount:  overrides.OperatorFeeVaultMinimumWithdrawalAmount.ToInt(),
		BaseFeeVaultRecipient:                    thisIntent.BaseFeeVaultRecipient,
		L1FeeVaultRecipient:                      thisIntent.L1FeeVaultRecipient,
		SequencerFeeVaultRecipient:               thisIntent.SequencerFeeVaultRecipient,
		OperatorFeeVaultRecipient:                thisIntent.OperatorFeeVaultRecipient,
		GovernanceTokenOwner:                     overrides.GovernanceTokenOwner,
		Fork:                                     big.NewInt(schedule.SolidityForkNumber(1)),
		EnableGovernance:                         overrides.EnableGovernance,
		FundDevAccounts:                          overrides.FundDevAccounts,
		// Custom Gas Token (CGT) configuration from intent
		UseCustomGasToken:          cgt.UseCustomGasToken,
		GasPayingTokenName:         cgt.GasPayingTokenName,
		GasPayingTokenSymbol:       cgt.GasPayingTokenSymbol,
		NativeAssetLiquidityAmount: cgt.NativeAssetLiquidityAmount,
		LiquidityControllerOwner:   cgt.LiquidityControllerOwner,
		DevFeatureBitmap:           devFeatureBitmap,
		UseInterop:                 intent.UseInterop,
	}

	engine := pEnv.ScriptEngine.Resolve()
	var dump *foundry.ForgeAllocs
	switch engine {
	case env.ScriptEngineRust:
		dump, err = runL2GenesisRust(pEnv, bundle, input)
	case env.ScriptEngineGo:
		dump, err = runL2GenesisGo(pEnv, bundle, input)
	default:
		return fmt.Errorf("unknown script engine %q", engine)
	}
	if err != nil {
		return fmt.Errorf("failed to run L2Genesis script (%s engine): %w", engine, err)
	}

	if err := genesis.CheckL2GenesisAllocs(dump, genesis.CheckL2AllocsOpts{
		FundDevAccounts: overrides.FundDevAccounts,
		// Tagged L2Genesis artifacts predating the #21339 prank nonce reset leave the
		// proxy admin owner with a bumped nonce, so allow it as a plain EOA.
		AllowedEOAs: []common.Address{thisIntent.Roles.L2ProxyAdminOwner},
	}); err != nil {
		return fmt.Errorf("L2 genesis allocs failed validation: %w", err)
	}

	thisChainState.Allocs = &state.GzipData[foundry.ForgeAllocs]{
		Data: dump,
	}

	return nil
}

// runL2GenesisGo runs the L2Genesis script on the in-process Go script.Host and returns the
// resulting allocs (deploy from the script deployer, wipe the deployer, dump).
func runL2GenesisGo(pEnv *Env, bundle ArtifactsBundle, input opcm.L2GenesisInput) (*foundry.ForgeAllocs, error) {
	host, err := env.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		pEnv.Logger,
		pEnv.Deployer,
		bundle.L2,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 script host: %w", err)
	}

	scr, err := opcm.NewL2GenesisScript(host)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2Genesis script: %w", err)
	}

	if err := scr.Run(input); err != nil {
		return nil, fmt.Errorf("failed to call L2Genesis script: %w", err)
	}

	host.Wipe(pEnv.Deployer)

	dump, err := host.StateDump()
	if err != nil {
		return nil, fmt.Errorf("failed to dump state: %w", err)
	}
	return dump, nil
}

// runL2GenesisRust runs the L2Genesis script in the out-of-process Rust op-script-engine over a
// Unix-socket JSON-RPC connection, mirroring runL2GenesisGo. The engine reads artifacts from the
// L2 bundle's on-disk directory and is driven with the same ABI-packed run(input) calldata.
func runL2GenesisRust(pEnv *Env, bundle ArtifactsBundle, input opcm.L2GenesisInput) (*foundry.ForgeAllocs, error) {
	artifactsDir, err := rustengine.ArtifactsDir(bundle.L2)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve L2 artifacts directory: %w", err)
	}

	art, err := (&foundry.ArtifactsFS{FS: bundle.L2}).ReadArtifact("L2Genesis.s.sol", "L2Genesis")
	if err != nil {
		return nil, fmt.Errorf("failed to read L2Genesis artifact: %w", err)
	}
	packed, err := art.ABI.Pack("run", input)
	if err != nil {
		return nil, fmt.Errorf("failed to ABI-pack L2Genesis run input: %w", err)
	}

	binPath, err := rustengine.EngineBinary(pEnv.Context, pEnv.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to provision op-script-engine binary: %w", err)
	}

	eng, err := rustengine.Spawn(binPath, artifactsDir, script.DefaultContext.ChainID.Uint64(), true, rustengine.NewLogWriter(pEnv.Logger))
	if err != nil {
		return nil, fmt.Errorf("failed to spawn op-script-engine: %w", err)
	}
	defer eng.Close()

	if _, err := eng.RunScript("L2Genesis.s.sol", "L2Genesis", packed, pEnv.Deployer); err != nil {
		return nil, fmt.Errorf("engine failed to run L2Genesis: %w", err)
	}
	if err := eng.Wipe(pEnv.Deployer); err != nil {
		return nil, fmt.Errorf("engine failed to wipe deployer: %w", err)
	}
	dump, err := eng.StateDump()
	if err != nil {
		return nil, fmt.Errorf("engine failed to dump state: %w", err)
	}
	return dump, nil
}

func calculateL2GenesisOverrides(intent *state.Intent, thisIntent *state.ChainIntent) (l2GenesisOverrides, *genesis.UpgradeScheduleDeployConfig, error) {
	schedule := standard.DefaultHardforkSchedule()

	overrides := defaultOverrides()
	// Special case for FundDevAccounts since it's both an intent value and an override.
	overrides.FundDevAccounts = intent.FundDevAccounts

	var err error
	if len(intent.GlobalDeployOverrides) > 0 {
		schedule, err = jsonutil.MergeJSON(schedule, intent.GlobalDeployOverrides)
		if err != nil {
			return l2GenesisOverrides{}, nil, fmt.Errorf("failed to merge global deploy overrides: %w", err)
		}
		overrides, err = jsonutil.MergeJSON(overrides, intent.GlobalDeployOverrides)
		if err != nil {
			return l2GenesisOverrides{}, nil, fmt.Errorf("failed to merge global deploy overrides: %w", err)
		}
	}

	if len(thisIntent.DeployOverrides) > 0 {
		schedule, err = jsonutil.MergeJSON(schedule, thisIntent.DeployOverrides)
		if err != nil {
			return l2GenesisOverrides{}, nil, fmt.Errorf("failed to merge L2 deploy overrides: %w", err)
		}
		overrides, err = jsonutil.MergeJSON(overrides, thisIntent.DeployOverrides)
		if err != nil {
			return l2GenesisOverrides{}, nil, fmt.Errorf("failed to merge global deploy overrides: %w", err)
		}
	}

	return overrides, schedule, nil
}

// buildCGTConfig returns the CGT configuration when enabled, otherwise an empty config.
func buildCGTConfig(intent *state.ChainIntent) cgtConfig {
	if !intent.IsCustomGasTokenEnabled() {
		return cgtConfig{
			UseCustomGasToken:          false,
			GasPayingTokenName:         "",
			GasPayingTokenSymbol:       "",
			NativeAssetLiquidityAmount: big.NewInt(0),
			LiquidityControllerOwner:   common.Address{},
		}
	}
	return cgtConfig{
		UseCustomGasToken:          true,
		GasPayingTokenName:         intent.CustomGasToken.Name,
		GasPayingTokenSymbol:       intent.CustomGasToken.Symbol,
		NativeAssetLiquidityAmount: intent.GetInitialLiquidity(),
		LiquidityControllerOwner:   intent.GetLiquidityControllerOwner(),
	}
}

func shouldGenerateL2Genesis(thisChainState *state.ChainState) bool {
	return thisChainState.Allocs == nil
}

func wdNetworkToBig(wd genesis.WithdrawalNetwork) *big.Int {
	n := wd.ToUint8()
	return big.NewInt(int64(n))
}

// buildDevFeatureBitmap reads the devFeatureBitmap from global overrides and returns an error if the interop feature
// bit does not match the UseInterop intent flag. This ensures that interop feature is explicitly enabled by both the intent and the boolean flag.
func buildDevFeatureBitmap(intent *state.Intent) (common.Hash, error) {
	var devFeatureBitmap common.Hash
	switch v := intent.GlobalDeployOverrides["devFeatureBitmap"].(type) {
	case common.Hash:
		devFeatureBitmap = v
	case string:
		devFeatureBitmap = common.HexToHash(v)
	}

	interopBitEnabled := devfeatures.IsDevFeatureEnabled(devFeatureBitmap, devfeatures.OptimismPortalInteropFlag)

	if intent.UseInterop != interopBitEnabled {
		return common.Hash{}, fmt.Errorf("interop feature in devFeatureBitmap does not match the UseInterop intent flag")
	}

	return devFeatureBitmap, nil
}

func defaultOverrides() l2GenesisOverrides {
	return l2GenesisOverrides{
		FundDevAccounts:                          false,
		BaseFeeVaultMinimumWithdrawalAmount:      standard.VaultMinWithdrawalAmount,
		L1FeeVaultMinimumWithdrawalAmount:        standard.VaultMinWithdrawalAmount,
		SequencerFeeVaultMinimumWithdrawalAmount: standard.VaultMinWithdrawalAmount,
		OperatorFeeVaultMinimumWithdrawalAmount:  standard.VaultMinWithdrawalAmount,
		BaseFeeVaultWithdrawalNetwork:            "local",
		L1FeeVaultWithdrawalNetwork:              "local",
		SequencerFeeVaultWithdrawalNetwork:       "local",
		OperatorFeeVaultWithdrawalNetwork:        "local",
		EnableGovernance:                         false,
		GovernanceTokenOwner:                     standard.GovernanceTokenOwner,
	}
}
