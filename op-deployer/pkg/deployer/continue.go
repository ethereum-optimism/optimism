package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

type ContinueConfig struct {
	Workdir    string
	L1RPCUrl   string
	PrivateKey string
	CacheDir   string
	Logger     log.Logger

	privateKeyECDSA *ecdsa.PrivateKey
}

func (c *ContinueConfig) Check() error {
	if c.Workdir == "" {
		return fmt.Errorf("workdir must be specified")
	}
	if c.Logger == nil {
		return fmt.Errorf("logger must be specified")
	}
	if c.PrivateKey == "" {
		return fmt.Errorf("private key must be specified")
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(c.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	c.privateKeyECDSA = privateKey
	if c.L1RPCUrl == "" {
		return fmt.Errorf("l1 RPC URL must be specified")
	}
	return nil
}

func newContinueConfig(cliCtx *cli.Context, lgr log.Logger) ContinueConfig {
	return ContinueConfig{
		Workdir:    cliCtx.String(WorkdirFlagName),
		L1RPCUrl:   cliCtx.String(L1RPCURLFlagName),
		PrivateKey: cliCtx.String(PrivateKeyFlagName),
		CacheDir:   cliCtx.String(CacheDirFlagName),
		Logger:     lgr,
	}
}

func ContinueCLI() func(cliCtx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(cliCtx)
		lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
		oplog.SetGlobalLogHandler(lgr.Handler())

		ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)
		return Continue(ctx, newContinueConfig(cliCtx, lgr))
	}
}

func Continue(ctx context.Context, cfg ContinueConfig) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for continue: %w", err)
	}

	intent, err := pipeline.ReadIntent(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read intent: %w", err)
	}
	st, err := pipeline.ReadState(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	if !st.Prepared {
		return fmt.Errorf("state was not produced by op-deployer prepare. Run op-deployer prepare before op-deployer continue")
	}
	if err := pipeline.ValidateInputs(intent, st); err != nil {
		return fmt.Errorf("failed to validate continuation inputs: %w", err)
	}
	if err := pipeline.ValidateInteropDepSetMatchesIntent(intent.Chains, st.InteropDepSet); err != nil {
		return fmt.Errorf("failed to validate prepared chain set: %w", err)
	}

	l1RPC, err := rpc.DialContext(ctx, cfg.L1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer l1RPC.Close()
	l1Client := ethclient.NewClient(l1RPC)

	if err := validateL1ChainID(ctx, l1RPC, intent); err != nil {
		return err
	}
	if st.L1PredictSenderAddress == nil || *st.L1PredictSenderAddress == (common.Address{}) {
		return fmt.Errorf("prepared state has no pinned deployer address. Rerun op-deployer prepare")
	}
	if st.L1PredictOPCMAddress == nil || *st.L1PredictOPCMAddress == (common.Address{}) {
		return fmt.Errorf("prepared state has no pinned OPCM address. Rerun op-deployer prepare")
	}

	deployer := crypto.PubkeyToAddress(cfg.privateKeyECDSA.PublicKey)
	pinnedOPCM := *st.L1PredictOPCMAddress
	if err := st.CheckL1PredictInputs(deployer, pinnedOPCM); err != nil {
		return err
	}
	if intent.OPCMAddress != nil && *intent.OPCMAddress != pinnedOPCM {
		return fmt.Errorf(
			"intent OPCM address changed after prepare: pinned %s, intent %s",
			pinnedOPCM,
			*intent.OPCMAddress,
		)
	}
	if err := resolveSuperchainConfigProxy(ctx, l1RPC, intent, pinnedOPCM); err != nil {
		return err
	}

	pending := make([]common.Hash, 0, len(intent.Chains))
	for _, chain := range intent.Chains {
		if !st.IsChainDeployed(chain.ID) {
			pending = append(pending, chain.ID)
		}
	}
	if len(pending) == 0 {
		return fmt.Errorf("no pending chains. Already-live reconciliation is not supported")
	}
	if len(pending) > 1 {
		return fmt.Errorf("continue supports exactly one pending chain but found %d", len(pending))
	}

	chainID := pending[0]
	chainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get prepared state for chain %s: %w", chainID.Hex(), err)
	}
	if chainState.StartBlock == nil {
		return fmt.Errorf("chain %s has no anchor block recorded by prepare", chainID.Hex())
	}
	if chainState.GenesisTime == nil {
		return fmt.Errorf("chain %s has no genesis time recorded by prepare", chainID.Hex())
	}
	if err := revalidateContinuationAnchor(ctx, l1RPC, chainID, chainState); err != nil {
		return err
	}
	latest, err := l1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get current L1 head: %w", err)
	}
	if uint64(*chainState.GenesisTime) <= latest.Time {
		cfg.Logger.Warn(
			"committed genesis time has elapsed",
			"chainID", chainID.Hex(),
			"genesisTime", uint64(*chainState.GenesisTime),
			"l1HeadTime", latest.Time,
		)
	}

	dci, err := pipeline.BuildContinuationDCI(intent, chainID, st)
	if err != nil {
		return fmt.Errorf("failed to build continuation input for chain %s: %w", chainID.Hex(), err)
	}

	l1ArtifactsFS, err := artifacts.Download(ctx, intent.L1ContractsLocator, ioutil.BarProgressor(), cfg.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to download L1 artifacts: %w", err)
	}
	capture := new(broadcaster.CaptureBroadcaster)
	l1Host, err := initForkHost(ctx, capture, cfg.Logger, deployer, l1ArtifactsFS, l1RPC)
	if err != nil {
		return fmt.Errorf("failed to initialize L1 preflight host: %w", err)
	}
	scripts, err := opcm.NewScripts(l1Host)
	if err != nil {
		return fmt.Errorf("failed to load OPCM scripts: %w", err)
	}
	pipelineEnv := &pipeline.Env{
		StateWriter:  pipeline.NoopStateWriter(),
		L1ScriptHost: l1Host,
		L1Client:     l1Client,
		Broadcaster:  capture,
		Deployer:     deployer,
		Logger:       cfg.Logger,
		Scripts:      scripts,
		Context:      ctx,
	}

	result, err := pipeline.ExecuteOPChainDeployment(pipelineEnv, st, chainID, dci)
	if err != nil {
		return fmt.Errorf("continuation preflight failed for chain %s: %w", chainID.Hex(), err)
	}
	captured := capture.Drain()
	if err := validateContinuationBroadcast(captured, deployer, pinnedOPCM); err != nil {
		return fmt.Errorf("continuation preflight failed for chain %s: %w", chainID.Hex(), err)
	}
	if result.Contracts() != chainState.OpChainContracts {
		return fmt.Errorf(
			"simulated contract addresses differ from the addresses prepared for chain %s: simulated %+v, prepared %+v",
			chainID.Hex(),
			result.Contracts(),
			chainState.OpChainContracts,
		)
	}

	deployRequirements, err := pipeline.ResolveInitialDeployRequirements(dci.DisputeGameType)
	if err != nil {
		return fmt.Errorf("failed to resolve validation requirements for chain %s: %w", chainID.Hex(), err)
	}
	var validator common.Address
	var validatorInput opcm.StandardValidatorInput
	if deployRequirements.Permissionless {
		validator, err = opcm.NewContract(pinnedOPCM, l1Client).OPCMStandardValidator(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve StandardValidator from pinned OPCM %s: %w", pinnedOPCM, err)
		}
		validatorInput = standardValidatorInput(dci, result.Contracts())
		if err := opcm.ValidateStandardDeployment(
			ctx,
			opcm.NewScriptHostCallBackend(l1Host),
			validator,
			validatorInput,
		); err != nil {
			return fmt.Errorf("simulated deployment validation failed for chain %s: %w", chainID.Hex(), err)
		}
	}

	if err := revalidateContinuationAnchor(ctx, l1RPC, chainID, chainState); err != nil {
		return err
	}
	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(cfg.privateKeyECDSA, intent.L1ChainIDBig()))
	liveBroadcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  cfg.Logger,
		ChainID: new(big.Int).Set(intent.L1ChainIDBig()),
		Client:  l1Client,
		Signer:  signer,
		From:    deployer,
	})
	if err != nil {
		return fmt.Errorf("failed to create live broadcaster: %w", err)
	}
	for _, bcast := range captured {
		liveBroadcaster.Hook(bcast)
	}
	broadcastResults, broadcastErr := liveBroadcaster.Broadcast(ctx)
	if err := validateContinuationReceipts(broadcastResults, broadcastErr); err != nil {
		return fmt.Errorf("failed to broadcast continuation for chain %s: %w", chainID.Hex(), err)
	}

	if err := pipeline.RecordOPChainDeployment(st, result); err != nil {
		return fmt.Errorf("failed to record deployment for chain %s: %w", chainID.Hex(), err)
	}
	if err := pipeline.WriteState(cfg.Workdir, st); err != nil {
		return fmt.Errorf("failed to checkpoint deployment for chain %s: %w", chainID.Hex(), err)
	}

	if err := validateContinuationCode(ctx, l1Client, chainState.OpChainContracts); err != nil {
		return fmt.Errorf("live deployment validation failed for chain %s: %w", chainID.Hex(), err)
	}
	if deployRequirements.Permissionless {
		if err := opcm.ValidateStandardDeployment(ctx, l1Client, validator, validatorInput); err != nil {
			return fmt.Errorf("live deployment validation failed for chain %s: %w", chainID.Hex(), err)
		}
	}

	st.AppliedIntent = intent
	if err := pipeline.WriteState(cfg.Workdir, st); err != nil {
		return fmt.Errorf("failed to write completed continuation state: %w", err)
	}
	return nil
}

func revalidateContinuationAnchor(
	ctx context.Context,
	l1RPC *rpc.Client,
	chainID common.Hash,
	chainState *state.ChainState,
) error {
	safe, err := pipeline.FetchL1BlockRefByNumber(ctx, l1RPC, "safe")
	if err != nil {
		return fmt.Errorf("failed to fetch L1 safe block: %w", err)
	}
	if _, err := selectAnchorBlock(ctx, l1RPC, safe, &chainState.StartBlock.Hash); err != nil {
		return fmt.Errorf("pinned anchor block for chain %s is no longer valid: %w", chainID.Hex(), err)
	}
	return nil
}

func validateContinuationBroadcast(
	broadcasts []script.Broadcast,
	deployer common.Address,
	pinnedOPCM common.Address,
) error {
	if len(broadcasts) != 1 {
		return fmt.Errorf("expected exactly one deployment call, got %d broadcasts", len(broadcasts))
	}
	bcast := broadcasts[0]
	if bcast.Type != script.BroadcastCall {
		return fmt.Errorf("expected deployment broadcast type %s, got %s", script.BroadcastCall, bcast.Type)
	}
	if bcast.To != pinnedOPCM {
		return fmt.Errorf("deployment broadcast target mismatch: expected %s, got %s", pinnedOPCM, bcast.To)
	}
	if bcast.From != deployer {
		return fmt.Errorf("deployment broadcast sender mismatch: expected %s, got %s", deployer, bcast.From)
	}
	return nil
}

func validateContinuationReceipts(results []broadcaster.BroadcastResult, broadcastErr error) error {
	if broadcastErr != nil {
		return broadcastErr
	}
	if len(results) != 1 {
		return fmt.Errorf("expected one broadcast result, got %d", len(results))
	}
	result := results[0]
	if result.Err != nil {
		return result.Err
	}
	if result.Receipt == nil {
		return fmt.Errorf("deployment transaction %s has no receipt", result.TxHash)
	}
	if result.Receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("deployment transaction %s failed with receipt status %d", result.TxHash, result.Receipt.Status)
	}
	return nil
}

func standardValidatorInput(
	dci opcm.DeployOPChainInput,
	contracts addresses.OpChainContracts,
) opcm.StandardValidatorInput {
	gameType := embedded.GameType(dci.DisputeGameType)
	useDevInput := gameType == embedded.GameTypeCannonKona || gameType == embedded.GameTypeSuperCannonKona
	return opcm.StandardValidatorInput{
		SystemConfig:        contracts.SystemConfigProxy,
		AbsolutePrestate:    dci.DisputeAbsolutePrestate,
		CannonPrestate:      dci.CannonAbsolutePrestate,
		CannonKonaPrestate:  dci.DisputeAbsolutePrestate,
		L2ChainID:           dci.L2ChainId,
		Proposer:            dci.Proposer,
		UseDevFeaturesInput: useDevInput,
	}
}

type namedAddress struct {
	name    string
	address common.Address
}

func validateContinuationCode(
	ctx context.Context,
	client *ethclient.Client,
	contracts addresses.OpChainContracts,
) error {
	for _, contract := range continuationContractAddresses(contracts) {
		if contract.address == (common.Address{}) {
			continue
		}
		code, err := client.CodeAt(ctx, contract.address, nil)
		if err != nil {
			return fmt.Errorf("failed to read %s code at %s: %w", contract.name, contract.address, err)
		}
		if len(code) == 0 {
			return fmt.Errorf("%s has no code at %s", contract.name, contract.address)
		}
	}
	return nil
}

func continuationContractAddresses(contracts addresses.OpChainContracts) []namedAddress {
	return []namedAddress{
		{"OpChainProxyAdminImpl", contracts.OpChainProxyAdminImpl},
		{"OptimismPortalProxy", contracts.OptimismPortalProxy},
		{"AddressManagerImpl", contracts.AddressManagerImpl},
		{"L1Erc721BridgeProxy", contracts.L1Erc721BridgeProxy},
		{"SystemConfigProxy", contracts.SystemConfigProxy},
		{"OptimismMintableErc20FactoryProxy", contracts.OptimismMintableErc20FactoryProxy},
		{"L1StandardBridgeProxy", contracts.L1StandardBridgeProxy},
		{"L1CrossDomainMessengerProxy", contracts.L1CrossDomainMessengerProxy},
		{"EthLockboxProxy", contracts.EthLockboxProxy},
		{"DisputeGameFactoryProxy", contracts.DisputeGameFactoryProxy},
		{"AnchorStateRegistryProxy", contracts.AnchorStateRegistryProxy},
		{"FaultDisputeGameImpl", contracts.FaultDisputeGameImpl},
		{"FaultDisputeGameCannonKonaImpl", contracts.FaultDisputeGameCannonKonaImpl},
		{"PermissionedDisputeGameImpl", contracts.PermissionedDisputeGameImpl},
		{"DelayedWethPermissionedGameProxy", contracts.DelayedWethPermissionedGameProxy},
		{"DelayedWethPermissionlessGameProxy", contracts.DelayedWethPermissionlessGameProxy},
		{"AltDAChallengeProxy", contracts.AltDAChallengeProxy},
		{"AltDAChallengeImpl", contracts.AltDAChallengeImpl},
		{"L2OutputOracleProxy", contracts.L2OutputOracleProxy},
	}
}
