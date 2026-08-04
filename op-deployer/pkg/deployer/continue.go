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
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

// Continue resumes a prepared deployment after simulating and validating every pending chain.
// This global preflight prevents predictable partial deployments but cannot make independent L1
// transactions atomic. Each chain's pinned anchor and the deployer nonce are therefore rechecked
// immediately before broadcast, and any drift stops the remaining sends.
func Continue(ctx context.Context, cfg ContinueConfig) error {
	runner := &continuationRunner{ctx: ctx, cfg: cfg}
	return runner.run()
}

type continuationRunner struct {
	ctx context.Context
	cfg ContinueConfig

	intent     *state.Intent
	state      *state.State
	l1RPC      *rpc.Client
	l1Client   *ethclient.Client
	deployer   common.Address
	pinnedOPCM common.Address
	artifacts  artifacts.Bundle

	expectedNonce uint64
	preflightEnv  *pipeline.Env
	capture       *broadcaster.CaptureBroadcaster
}

type continuationChain struct {
	id       common.Hash
	state    *state.ChainState
	expected state.ChainState
	dci      opcm.DeployOPChainInput
	result   pipeline.OPChainDeploymentResult
	captured []script.Broadcast
}

func (r *continuationRunner) run() error {
	if err := r.cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for continue: %w", err)
	}

	intent, err := pipeline.ReadIntent(r.cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read intent: %w", err)
	}
	st, err := pipeline.ReadState(r.cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	if err := pipeline.ValidatePreparedDeployment(intent, st); err != nil {
		return fmt.Errorf("failed to validate prepared deployment: %w", err)
	}
	if err := pipeline.ValidateCommittedPrestateOverrides(intent, st); err != nil {
		return fmt.Errorf("failed to validate committed prestates: %w", err)
	}
	if err := pipeline.ValidateInputs(intent, st); err != nil {
		return fmt.Errorf("failed to validate continuation inputs: %w", err)
	}
	if err := pipeline.ValidateInteropDepSetMatchesIntent(intent.Chains, st.InteropDepSet); err != nil {
		return fmt.Errorf("failed to validate prepared chain set: %w", err)
	}
	bundle, err := artifacts.DownloadBundle(
		r.ctx,
		st.PreparedDeployment.L1Artifacts.Locator,
		st.PreparedDeployment.L2Artifacts.Locator,
		ioutil.BarProgressor(),
		r.cfg.CacheDir,
	)
	if err != nil {
		return err
	}
	if err := pipeline.ValidatePreparedArtifactContents(st.PreparedDeployment, bundle); err != nil {
		return fmt.Errorf("failed to validate prepared artifacts: %w", err)
	}

	l1RPC, err := rpc.DialContext(r.ctx, r.cfg.L1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer l1RPC.Close()
	l1Client := ethclient.NewClient(l1RPC)
	r.intent = intent
	r.state = st
	r.l1RPC = l1RPC
	r.l1Client = l1Client
	r.artifacts = bundle

	if err := validateL1ChainID(r.ctx, l1RPC, st.PreparedDeployment.Intent); err != nil {
		return err
	}

	deployer := crypto.PubkeyToAddress(r.cfg.privateKeyECDSA.PublicKey)
	pinnedOPCM := st.PreparedDeployment.OPCM
	r.deployer = deployer
	r.pinnedOPCM = pinnedOPCM
	if err := st.CheckL1PredictInputs(deployer, pinnedOPCM); err != nil {
		return err
	}
	startNonce, err := readContinuationStartNonce(r.ctx, l1Client, deployer)
	if err != nil {
		return err
	}
	r.expectedNonce = startNonce

	pending, err := r.classifyChains()
	if err != nil {
		return err
	}
	if err := validateContinuationGameTypes(pending); err != nil {
		return err
	}
	for i := range pending {
		if err := r.preflight(&pending[i]); err != nil {
			return err
		}
	}
	for i := range pending {
		if err := r.broadcastAndCheckpoint(&pending[i]); err != nil {
			return err
		}
	}

	if err := pipeline.WriteState(r.cfg.Workdir, st); err != nil {
		return fmt.Errorf("failed to write completed continuation state: %w", err)
	}
	return nil
}

func validateContinuationGameTypes(pending []continuationChain) error {
	gameTypes := make([]uint32, 0, len(pending))
	for _, chain := range pending {
		gameTypes = append(gameTypes, chain.dci.DisputeGameType)
	}
	if err := pipeline.ValidateInitialGameTypeSet(gameTypes); err != nil {
		return fmt.Errorf("invalid pending continuation game types: %w", err)
	}
	return nil
}

func (r *continuationRunner) classifyChains() ([]continuationChain, error) {
	latest, err := r.l1Client.HeaderByNumber(r.ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get current L1 head: %w", err)
	}

	pending := make([]continuationChain, 0, len(r.state.PreparedDeployment.Chains))
	for _, preparedChain := range r.state.PreparedDeployment.Chains {
		chainID := preparedChain.ID
		chainState, err := r.state.Chain(chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get prepared state for chain %s: %w", chainID.Hex(), err)
		}
		expected := continuationExpectedChainState(preparedChain, chainState)
		if expected.StartBlock == nil {
			return nil, fmt.Errorf("chain %s has no anchor block recorded by prepare", chainID.Hex())
		}
		if expected.GenesisTime == nil {
			return nil, fmt.Errorf("chain %s has no genesis time recorded by prepare", chainID.Hex())
		}
		if err := revalidateContinuationAnchor(r.ctx, r.l1RPC, chainID, &expected); err != nil {
			return nil, err
		}
		if uint64(*expected.GenesisTime) <= latest.Time {
			r.cfg.Logger.Warn(
				"committed genesis time has elapsed",
				"chainID", chainID.Hex(),
				"genesisTime", uint64(*expected.GenesisTime),
				"l1HeadTime", latest.Time,
			)
		}

		dci, err := pipeline.BuildContinuationDCI(chainID, r.state)
		if err != nil {
			return nil, fmt.Errorf("failed to build continuation input for chain %s: %w", chainID.Hex(), err)
		}
		liveState, err := classifyContinuationAddresses(
			r.ctx,
			r.l1Client,
			expected.OpChainContracts,
			latest.Number,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to classify chain %s predicted addresses: %w", chainID.Hex(), err)
		}
		if r.state.IsChainDeployed(chainID) {
			switch liveState {
			case continuationAddressesAbsent:
				r.cfg.Logger.Warn("continued deployment is no longer live; redeploying", "chainID", chainID.Hex())
				deployed := false
				chainState.Continuation = nil
				chainState.Deployed = &deployed
				if err := pipeline.WriteState(r.cfg.Workdir, r.state); err != nil {
					return nil, fmt.Errorf("failed to checkpoint reorg recovery for chain %s: %w", chainID.Hex(), err)
				}
			case continuationAddressesComplete:
				if err := r.reverifyDeployedChain(chainID, chainState, &expected, dci); err != nil {
					return nil, err
				}
				continue
			}
		}
		if chainState.Continuation != nil {
			return nil, fmt.Errorf(
				"pending chain %s has continuation checkpoint metadata but is not marked deployed",
				chainID.Hex(),
			)
		}

		switch liveState {
		case continuationAddressesAbsent:
			pending = append(pending, continuationChain{
				id:       chainID,
				state:    chainState,
				expected: expected,
				dci:      dci,
			})
		case continuationAddressesComplete:
			if err := r.reconcileLiveChain(chainID, chainState, &expected, dci); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("chain %s has an unknown continuation address classification", chainID.Hex())
		}
	}
	if err := verifyContinuationHead(r.ctx, r.l1Client, latest.Number, latest.Hash()); err != nil {
		return nil, fmt.Errorf("continuation classification head changed: %w", err)
	}
	return pending, nil
}

func (r *continuationRunner) reverifyDeployedChain(
	chainID common.Hash,
	chainState *state.ChainState,
	expected *state.ChainState,
	dci opcm.DeployOPChainInput,
) error {
	if chainState.Continuation == nil {
		return fmt.Errorf("deployed chain %s has no continuation metadata", chainID.Hex())
	}
	// note: reverifying a deployed chain assumes no mutable configs were changed between runs
	// and a chain config modification fails validation for the entire run
	if err := verifyContinuationDeployment(
		r.ctx,
		r.l1Client,
		expected.OpChainContracts,
		expected,
		dci,
	); err != nil {
		return fmt.Errorf("live deployment validation failed for deployed chain %s: %w", chainID.Hex(), err)
	}

	r.cfg.Logger.Info("reconciled deployed chain", "chainID", chainID.Hex())
	return nil
}

func (r *continuationRunner) reconcileLiveChain(
	chainID common.Hash,
	chainState *state.ChainState,
	expected *state.ChainState,
	dci opcm.DeployOPChainInput,
) error {
	if err := verifyContinuationDeployment(
		r.ctx,
		r.l1Client,
		expected.OpChainContracts,
		expected,
		dci,
	); err != nil {
		return fmt.Errorf("already-live deployment validation failed for chain %s: %w", chainID.Hex(), err)
	}
	if err := r.initPreflightEnv(); err != nil {
		return err
	}
	result, err := pipeline.ReconcileOPChainDeployment(
		r.preflightEnv,
		chainID,
		expected.OpChainContracts,
		r.pinnedOPCM,
	)
	if err != nil {
		return fmt.Errorf("failed to normalize already-live deployment for chain %s: %w", chainID.Hex(), err)
	}
	if captured := r.capture.Drain(); len(captured) != 0 {
		return fmt.Errorf("implementation readback unexpectedly captured %d broadcasts for chain %s", len(captured), chainID.Hex())
	}
	if err := pipeline.RecordOPChainDeployment(r.state, result); err != nil {
		return fmt.Errorf("failed to record reconciled deployment for chain %s: %w", chainID.Hex(), err)
	}
	if chainState.Continuation == nil {
		chainState.Continuation = new(state.ContinuationState)
	}
	if err := pipeline.WriteState(r.cfg.Workdir, r.state); err != nil {
		return fmt.Errorf("failed to checkpoint reconciled deployment for chain %s: %w", chainID.Hex(), err)
	}
	r.cfg.Logger.Info("reconciled already-live pending chain", "chainID", chainID.Hex())
	return nil
}

func (r *continuationRunner) preflight(chain *continuationChain) error {
	if err := r.initPreflightEnv(); err != nil {
		return err
	}
	result, err := pipeline.ExecuteOPChainDeployment(r.preflightEnv, r.state, chain.id, chain.dci)
	if err != nil {
		return fmt.Errorf("continuation preflight failed for chain %s: %w", chain.id.Hex(), err)
	}
	captured := r.capture.Drain()
	if err := validateContinuationBroadcast(captured, r.deployer, r.pinnedOPCM); err != nil {
		return fmt.Errorf("continuation preflight failed for chain %s: %w", chain.id.Hex(), err)
	}
	if err := verifyContinuationDeployment(
		r.ctx,
		newScriptHostReadBackend(r.preflightEnv.L1ScriptHost),
		result.Contracts(),
		&chain.expected,
		chain.dci,
	); err != nil {
		return fmt.Errorf("simulated deployment validation failed for chain %s: %w", chain.id.Hex(), err)
	}
	chain.result = result
	chain.captured = captured
	return nil
}

func (r *continuationRunner) initPreflightEnv() error {
	if r.preflightEnv != nil {
		return nil
	}
	capture := new(broadcaster.CaptureBroadcaster)
	l1Host, err := env.DefaultForkedScriptHost(r.ctx, capture, r.cfg.Logger, r.deployer, r.artifacts.L1, r.l1RPC)
	if err != nil {
		return fmt.Errorf("failed to initialize L1 preflight host: %w", err)
	}
	scripts, err := opcm.NewScripts(l1Host)
	if err != nil {
		return fmt.Errorf("failed to load OPCM scripts: %w", err)
	}
	r.capture = capture
	r.preflightEnv = &pipeline.Env{
		StateWriter:  pipeline.NoopStateWriter(),
		L1ScriptHost: l1Host,
		L1Client:     r.l1Client,
		Broadcaster:  capture,
		Deployer:     r.deployer,
		Logger:       r.cfg.Logger,
		Scripts:      scripts,
		Context:      r.ctx,
	}
	return nil
}

func (r *continuationRunner) broadcastAndCheckpoint(chain *continuationChain) error {
	if err := revalidateContinuationAnchor(r.ctx, r.l1RPC, chain.id, &chain.expected); err != nil {
		return err
	}
	if err := requireContinuationNonce(r.ctx, r.l1Client, r.deployer, r.expectedNonce); err != nil {
		return err
	}

	chainID := r.state.PreparedDeployment.Intent.L1ChainIDBig()
	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(r.cfg.privateKeyECDSA, chainID))
	liveBroadcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  r.cfg.Logger,
		ChainID: new(big.Int).Set(chainID),
		Client:  r.l1Client,
		Signer:  signer,
		From:    r.deployer,
	})
	if err != nil {
		return fmt.Errorf("failed to create live broadcaster: %w", err)
	}
	for _, bcast := range chain.captured {
		liveBroadcaster.Hook(bcast)
	}
	broadcastResults, broadcastErr := liveBroadcaster.Broadcast(r.ctx)
	broadcastResult, err := successfulContinuationBroadcast(broadcastResults, broadcastErr)
	if err != nil {
		return fmt.Errorf("failed to broadcast continuation for chain %s: %w", chain.id.Hex(), err)
	}
	if err := validateContinuationReceiptCanonicality(r.ctx, r.l1RPC, broadcastResult.Receipt); err != nil {
		return fmt.Errorf("deployment receipt for chain %s is not canonical: %w", chain.id.Hex(), err)
	}

	chain.state.Continuation = new(state.ContinuationState)
	if err := pipeline.RecordOPChainDeployment(r.state, chain.result); err != nil {
		return fmt.Errorf("failed to record deployment for chain %s: %w", chain.id.Hex(), err)
	}
	if err := pipeline.WriteState(r.cfg.Workdir, r.state); err != nil {
		return fmt.Errorf("failed to checkpoint deployment for chain %s: %w", chain.id.Hex(), err)
	}
	if err := verifyContinuationDeployment(
		r.ctx,
		r.l1Client,
		chain.expected.OpChainContracts,
		&chain.expected,
		chain.dci,
	); err != nil {
		return fmt.Errorf("live deployment validation failed for chain %s: %w", chain.id.Hex(), err)
	}
	r.expectedNonce++
	return nil
}

type continuationNonceReader interface {
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
}

func readContinuationStartNonce(
	ctx context.Context,
	client continuationNonceReader,
	deployer common.Address,
) (uint64, error) {
	latest, pending, err := readContinuationNonces(ctx, client, deployer)
	if err != nil {
		return 0, err
	}
	switch {
	case pending > latest:
		return 0, fmt.Errorf(
			"deployer %s has unconfirmed transactions (latest nonce %d, pending %d); "+
				"wait for them to be mined, then rerun op-deployer continue",
			deployer,
			latest,
			pending,
		)
	case pending < latest:
		return 0, fmt.Errorf(
			"deployer %s pending nonce %d is below latest nonce %d; the L1 RPC is serving inconsistent state",
			deployer,
			pending,
			latest,
		)
	}
	return latest, nil
}

func requireContinuationNonce(
	ctx context.Context,
	client continuationNonceReader,
	deployer common.Address,
	expected uint64,
) error {
	latest, pending, err := readContinuationNonces(ctx, client, deployer)
	if err != nil {
		return err
	}
	if latest != expected || pending != expected {
		return fmt.Errorf(
			"unexpected deployer nonce movement before send: expected latest and pending nonce %d, got latest %d and pending %d",
			expected,
			latest,
			pending,
		)
	}
	return nil
}

func readContinuationNonces(
	ctx context.Context,
	client continuationNonceReader,
	deployer common.Address,
) (uint64, uint64, error) {
	latest, err := client.NonceAt(ctx, deployer, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read latest deployer nonce: %w", err)
	}
	pending, err := client.PendingNonceAt(ctx, deployer)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read pending deployer nonce: %w", err)
	}
	return latest, pending, nil
}

type continuationAddressState uint8

const (
	continuationAddressesAbsent continuationAddressState = iota
	continuationAddressesComplete
)

func classifyContinuationAddresses(
	ctx context.Context,
	backend continuationReadBackend,
	contracts addresses.OpChainContracts,
	blockNumber *big.Int,
) (continuationAddressState, error) {
	type addressCode struct {
		name    string
		address common.Address
		hasCode bool
	}
	codeStates := make([]addressCode, 0)
	withCode := 0
	for _, contract := range continuationContractAddresses(contracts) {
		if !contract.deploymentMarker || contract.address == (common.Address{}) {
			continue
		}
		code, err := backend.CodeAt(ctx, contract.address, blockNumber)
		if err != nil {
			return 0, fmt.Errorf("failed to read %s code at %s: %w", contract.name, contract.address, err)
		}
		hasCode := len(code) != 0
		if hasCode {
			withCode++
		}
		codeStates = append(codeStates, addressCode{contract.name, contract.address, hasCode})
	}
	if len(codeStates) == 0 {
		return 0, fmt.Errorf("prepared state contains no nonzero predicted contract addresses")
	}
	if withCode == 0 {
		return continuationAddressesAbsent, nil
	}
	if withCode == len(codeStates) {
		return continuationAddressesComplete, nil
	}

	diagnostics := make([]string, 0, len(codeStates))
	for _, codeState := range codeStates {
		status := "has no code"
		if codeState.hasCode {
			status = "has code"
		}
		diagnostics = append(
			diagnostics,
			fmt.Sprintf("%s %s %s", codeState.name, codeState.address, status),
		)
	}
	return 0, fmt.Errorf("partial deployment at predicted addresses: %s", strings.Join(diagnostics, "; "))
}

func continuationExpectedChainState(
	prepared *state.PreparedChainState,
	chainState *state.ChainState,
) state.ChainState {
	expected := *chainState
	expected.OpChainContracts = prepared.OpChainContracts
	expected.StartBlock = prepared.StartBlock
	expected.GenesisTime = prepared.GenesisTime
	return expected
}

func validateContinuationReceiptCanonicality(
	ctx context.Context,
	l1 pipeline.L1BlockFetcher,
	receipt *types.Receipt,
) error {
	if receipt == nil {
		return fmt.Errorf("deployment transaction has no receipt")
	}
	if receipt.BlockNumber == nil {
		return fmt.Errorf("deployment transaction receipt has no block number")
	}
	canonical, err := pipeline.FetchL1BlockRefByNumber(ctx, l1, hexutil.EncodeBig(receipt.BlockNumber))
	if err != nil {
		return fmt.Errorf("failed to fetch receipt block %s: %w", receipt.BlockNumber, err)
	}
	if canonical.Hash != receipt.BlockHash {
		return fmt.Errorf(
			"receipt block hash mismatch at height %s: receipt %s, canonical %s",
			receipt.BlockNumber,
			receipt.BlockHash,
			canonical.Hash,
		)
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

func successfulContinuationBroadcast(
	results []broadcaster.BroadcastResult,
	broadcastErr error,
) (broadcaster.BroadcastResult, error) {
	if broadcastErr != nil {
		return broadcaster.BroadcastResult{}, broadcastErr
	}
	if len(results) != 1 {
		return broadcaster.BroadcastResult{}, fmt.Errorf("expected one broadcast result, got %d", len(results))
	}
	result := results[0]
	if result.Err != nil {
		return broadcaster.BroadcastResult{}, result.Err
	}
	if result.Receipt == nil {
		return broadcaster.BroadcastResult{}, fmt.Errorf("deployment transaction %s has no receipt", result.TxHash)
	}
	if result.Receipt.Status != types.ReceiptStatusSuccessful {
		return broadcaster.BroadcastResult{}, fmt.Errorf(
			"deployment transaction %s failed with receipt status %d",
			result.TxHash,
			result.Receipt.Status,
		)
	}
	return result, nil
}
