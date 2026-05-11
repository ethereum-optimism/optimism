package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	txIntentBindings "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/lmittmann/w3"
	"github.com/urfave/cli/v2"
)

const EnvVarPrefix = "CHECK_INTEROP"

var (
	SourceRPCFlag = &cli.StringFlag{
		Name:     "source-rpc",
		Usage:    "RPC URL for the source L2 chain",
		EnvVars:  opservice.PrefixEnvVar(EnvVarPrefix, "SOURCE_RPC"),
		Required: true,
	}
	DestRPCFlag = &cli.StringFlag{
		Name:     "dest-rpc",
		Usage:    "RPC URL for the destination L2 chain",
		EnvVars:  opservice.PrefixEnvVar(EnvVarPrefix, "DEST_RPC"),
		Required: true,
	}
	PrivateKeyFlag = &cli.StringFlag{
		Name:    "private-key",
		Usage:   "Hex-encoded private key for signing transactions. If not set, uses devkeys test mnemonic.",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "PRIVATE_KEY"),
	}
	DevKeyIndexFlag = &cli.UintFlag{
		Name:    "devkey-index",
		Usage:   "Devkey user index (used when --private-key is not set)",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "DEVKEY_INDEX"),
		Value:   10_000,
	}
	RelayFlag = &cli.BoolFlag{
		Name:    "relay",
		Usage:   "If set, also relay (execute) the message on the destination chain",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "RELAY"),
	}
	RawFlag = &cli.BoolFlag{
		Name:    "raw",
		Usage:   "Use raw EventLogger init/exec instead of L2ToL2CrossDomainMessenger (default: CDM)",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "RAW"),
	}
	EventLoggerFlag = &cli.StringFlag{
		Name:    "event-logger",
		Usage:   "Address of a pre-deployed EventLogger. If empty, deploys a new one.",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "EVENT_LOGGER"),
	}
	LoopFlag = &cli.UintFlag{
		Name:    "loop",
		Usage:   "Number of round-trips (send A->B then B->A). 0 means single one-way send.",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "LOOP"),
		Value:   0,
	}
)

func main() {
	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Name = "check-interop"
	app.Usage = "Send an interop transaction between two chains, and optionally relay it"
	app.Flags = []cli.Flag{
		SourceRPCFlag,
		DestRPCFlag,
		PrivateKeyFlag,
		DevKeyIndexFlag,
		RelayFlag,
		RawFlag,
		EventLoggerFlag,
		LoopFlag,
	}
	app.Action = run

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Crit("Application failed", "err", err)
	}
}

func run(cliCtx *cli.Context) error {
	ctx := cliCtx.Context
	logger := oplog.NewLogger(os.Stderr, oplog.DefaultCLIConfig())
	oplog.SetGlobalLogHandler(logger.Handler())

	privKey, err := resolvePrivateKey(cliCtx)
	if err != nil {
		return fmt.Errorf("resolve private key: %w", err)
	}
	address := crypto.PubkeyToAddress(privKey.PublicKey)
	logger.Info("Using sender", "address", address)

	source, err := connectChain(ctx, logger, "source", cliCtx.String(SourceRPCFlag.Name))
	if err != nil {
		return err
	}
	defer source.close()

	dest, err := connectChain(ctx, logger, "dest", cliCtx.String(DestRPCFlag.Name))
	if err != nil {
		return err
	}
	defer dest.close()

	logger.Info("Connected to chains", "source", source.chainID, "dest", dest.chainID)
	if source.chainID == dest.chainID {
		return fmt.Errorf("source and destination chains have the same chain ID: %s", source.chainID)
	}

	sourceUser := &user{chain: source, privKey: privKey}
	destUser := &user{chain: dest, privKey: privKey}

	loopCount := cliCtx.Uint(LoopFlag.Name)
	relay := cliCtx.Bool(RelayFlag.Name)

	if cliCtx.Bool(RawFlag.Name) {
		return runRawFlow(ctx, logger, sourceUser, destUser, cliCtx.String(EventLoggerFlag.Name), relay, loopCount)
	}
	return runCDMFlow(ctx, logger, sourceUser, destUser, relay, loopCount)
}

// runRawFlow uses EventLogger init + CrossL2Inbox exec.
func runRawFlow(ctx context.Context, logger log.Logger, sourceUser, destUser *user, eventLoggerAddr string, relay bool, loopCount uint) error {
	var eventLogger common.Address
	if eventLoggerAddr != "" {
		eventLogger = common.HexToAddress(eventLoggerAddr)
		logger.Info("Using existing EventLogger", "address", eventLogger)
	} else {
		logger.Info("Deploying EventLogger on source chain...")
		var err error
		eventLogger, err = sourceUser.deployEventLogger(ctx)
		if err != nil {
			return fmt.Errorf("deploy EventLogger: %w", err)
		}
		logger.Info("EventLogger deployed", "address", eventLogger)
	}

	trips := max(loopCount, 1)
	sender, receiver := sourceUser, destUser
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := range trips {
		direction := fmt.Sprintf("%s->%s", sender.chain.chainID, receiver.chain.chainID)
		logger.Info("Sending init message", "trip", i+1, "of", trips, "direction", direction)

		initMsg, err := sender.sendInitMessage(ctx, rng, eventLogger)
		if err != nil {
			return fmt.Errorf("trip %d: send init message: %w", i+1, err)
		}
		logger.Info("Init message included",
			"trip", i+1,
			"tx", initMsg.receipt.TxHash,
			"block", initMsg.receipt.BlockNumber,
		)

		if !relay {
			logger.Info("Skipping exec (--relay not set)", "trip", i+1)
		} else {
			if err := waitForTimestampPast(ctx, receiver.chain, initMsg.msg.Identifier.Timestamp); err != nil {
				return fmt.Errorf("trip %d: %w", i+1, err)
			}
			execReceipt, err := receiver.sendExecMessage(ctx, initMsg)
			if err != nil {
				return fmt.Errorf("trip %d: send exec message: %w", i+1, err)
			}
			if execReceipt.Status != types.ReceiptStatusSuccessful {
				return fmt.Errorf("trip %d: exec tx reverted (tx %s)", i+1, execReceipt.TxHash)
			}
			logger.Info("Exec message included",
				"trip", i+1,
				"tx", execReceipt.TxHash,
				"block", execReceipt.BlockNumber,
			)
		}

		// Swap direction for next trip
		sender, receiver = receiver, sender
	}

	logger.Info("All trips completed", "count", trips)
	return nil
}

var sendETHFn = w3.MustNewFunc("sendETH(address,uint256)", "bytes32")

// sendETHTrigger calls SuperchainETHBridge.sendETH to bridge ETH cross-chain.
type sendETHTrigger struct {
	Recipient   common.Address
	Destination eth.ChainID
}

func (t *sendETHTrigger) To() (*common.Address, error) {
	addr := predeploys.SuperchainETHBridgeAddr
	return &addr, nil
}

func (t *sendETHTrigger) EncodeInput() ([]byte, error) {
	return sendETHFn.EncodeArgs(t.Recipient, t.Destination.ToBig())
}

func (t *sendETHTrigger) AccessList() (types.AccessList, error) {
	return nil, nil
}

// runCDMFlow bridges random ETH amounts via SuperchainETHBridge and verifies balances.
func runCDMFlow(ctx context.Context, logger log.Logger, sourceUser, destUser *user, relay bool, loopCount uint) error {
	// Use a fresh recipient so balance checks are unambiguous (starts at 0).
	recipientKey, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("generate recipient key: %w", err)
	}
	recipient := crypto.PubkeyToAddress(recipientKey.PublicKey)
	logger.Info("Using recipient for balance verification", "address", recipient)

	trips := max(loopCount, 1)
	sender, receiver := sourceUser, destUser
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Track expected balance on each chain for the recipient.
	expectedBalance := map[eth.ChainID]*big.Int{
		sourceUser.chain.chainID: new(big.Int),
		destUser.chain.chainID:   new(big.Int),
	}

	for i := range trips {
		// Random amount between 1000 and 100_000 wei — tiny but nonzero.
		amount := eth.WeiU64(uint64(1000 + rng.Intn(99_001)))
		direction := fmt.Sprintf("%s->%s", sender.chain.chainID, receiver.chain.chainID)
		logger.Info("Sending ETH via SuperchainETHBridge",
			"trip", i+1, "of", trips, "direction", direction,
			"amount", amount, "recipient", recipient,
		)

		sendTx := txintent.NewIntent[*sendETHTrigger, *txintent.InteropOutput](
			sender.plan(),
			txplan.WithValue(amount),
		)
		sendTx.Content.Set(&sendETHTrigger{
			Recipient:   recipient,
			Destination: receiver.chain.chainID,
		})

		sendReceipt, err := sendTx.PlannedTx.Included.Eval(ctx)
		if err != nil {
			return fmt.Errorf("trip %d: sendETH: %w", i+1, err)
		}
		if sendReceipt.Status != types.ReceiptStatusSuccessful {
			return fmt.Errorf("trip %d: sendETH tx reverted (tx %s)", i+1, sendReceipt.TxHash)
		}
		logger.Info("sendETH included",
			"trip", i+1,
			"tx", sendReceipt.TxHash,
			"block", sendReceipt.BlockNumber,
		)

		if !relay {
			logger.Info("Skipping relay (--relay not set)", "trip", i+1)
			sender, receiver = receiver, sender
			continue
		}

		// Relay with retry — the message may not be relayable immediately.
		relayReceipt, err := relayWithRetry(ctx, logger, receiver, sendTx, i+1)
		if err != nil {
			return err
		}
		logger.Info("relayMessage included",
			"trip", i+1,
			"tx", relayReceipt.TxHash,
			"block", relayReceipt.BlockNumber,
		)

		// Update expected balance and verify.
		expectedBalance[receiver.chain.chainID].Add(expectedBalance[receiver.chain.chainID], amount.ToBig())
		expected := expectedBalance[receiver.chain.chainID]

		if err := waitForBalance(ctx, receiver.chain, recipient, expected); err != nil {
			return fmt.Errorf("trip %d: %w", i+1, err)
		}

		// Query actual balances on both chains for a clear summary.
		balSource, _ := sourceUser.chain.ethClient.BalanceAt(ctx, recipient, nil)
		balDest, _ := destUser.chain.ethClient.BalanceAt(ctx, recipient, nil)
		if balSource == nil {
			balSource = new(big.Int)
		}
		if balDest == nil {
			balDest = new(big.Int)
		}
		logger.Info("Balance check after trip",
			"trip", i+1,
			"sent", amount,
			"direction", direction,
			"source_balance", fmt.Sprintf("%s wei (chain %s)", balSource, sourceUser.chain.chainID),
			"dest_balance", fmt.Sprintf("%s wei (chain %s)", balDest, destUser.chain.chainID),
		)

		// Swap direction for next trip.
		sender, receiver = receiver, sender
	}

	logger.Info("All trips completed", "count", trips)
	return nil
}

func relayWithRetry(ctx context.Context, logger log.Logger, receiver *user, sendTx *txintent.IntentTx[*sendETHTrigger, *txintent.InteropOutput], trip uint) (*types.Receipt, error) {
	deadline := time.Now().Add(60 * time.Second)
	for attempt := 0; ; attempt++ {
		relayTx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](receiver.plan())
		relayTx.Content.DependOn(&sendTx.Result)
		relayTx.Content.Fn(txintent.RelayIndexed(
			predeploys.L2toL2CrossDomainMessengerAddr,
			&sendTx.Result,
			&sendTx.PlannedTx.Included,
			1, // sendETH emits 2 logs; index 1 is the SentMessage
		))

		receipt, err := relayTx.PlannedTx.Included.Eval(ctx)
		if err == nil && receipt.Status == types.ReceiptStatusSuccessful {
			return receipt, nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return nil, fmt.Errorf("trip %d: relay timed out: %w", trip, err)
			}
			return nil, fmt.Errorf("trip %d: relay tx kept reverting until timeout", trip)
		}
		if err != nil {
			logger.Info("Waiting for relayability", "trip", trip, "attempt", attempt+1, "err", err)
		} else {
			logger.Info("Relay reverted, retrying", "trip", trip, "attempt", attempt+1, "tx", receipt.TxHash)
		}
		time.Sleep(time.Second)
	}
}

func waitForBalance(ctx context.Context, c *chain, addr common.Address, want *big.Int) error {
	deadline := time.Now().Add(60 * time.Second)
	for {
		balance, err := c.ethClient.BalanceAt(ctx, addr, nil)
		if err == nil && balance.Cmp(want) == 0 {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("timed out waiting for balance on %s: %w", c.name, err)
			}
			return fmt.Errorf("timed out waiting for %s balance on %s; got %s, want %s", addr, c.name, balance, want)
		}
		time.Sleep(time.Second)
	}
}

// chain represents a connected remote L2 chain.
type chain struct {
	name      string
	ethClient *sources.EthClient
	chainID   eth.ChainID
}

func (c *chain) close() {
	c.ethClient.Close()
}

func connectChain(ctx context.Context, logger log.Logger, name, url string) (*chain, error) {
	chainLogger := logger.New("chain", name, "rpc", url)
	rpcCl, err := opclient.NewRPC(
		ctx,
		chainLogger,
		url,
		opclient.WithFixedDialBackoff(time.Second),
		opclient.WithDialAttempts(5),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s RPC %s: %w", name, url, err)
	}
	ethCl, err := sources.NewEthClient(rpcCl, chainLogger, nil, sources.DefaultEthClientConfig(10))
	if err != nil {
		rpcCl.Close()
		return nil, fmt.Errorf("create %s eth client: %w", name, err)
	}
	chainIDBig, err := ethCl.ChainID(ctx)
	if err != nil {
		ethCl.Close()
		return nil, fmt.Errorf("fetch %s chain ID: %w", name, err)
	}
	return &chain{
		name:      name,
		ethClient: ethCl,
		chainID:   eth.ChainIDFromBig(chainIDBig),
	}, nil
}

// user is an EOA on a specific chain.
type user struct {
	chain   *chain
	privKey *ecdsa.PrivateKey
}

func (u *user) plan() txplan.Option {
	return txplan.Combine(
		txplan.WithChainID(u.chain.ethClient),
		txplan.WithPrivateKey(u.privKey),
		txplan.WithPendingNonce(u.chain.ethClient),
		txplan.WithAgainstLatestBlock(u.chain.ethClient),
		txplan.WithEstimator(u.chain.ethClient, true),
		txplan.WithRetrySubmission(u.chain.ethClient, 5, retry.Exponential()),
		txplan.WithRetryInclusion(u.chain.ethClient, 5, retry.Exponential()),
		txplan.WithBlockInclusionInfo(u.chain.ethClient),
	)
}

func (u *user) deployEventLogger(ctx context.Context) (common.Address, error) {
	tx := txplan.NewPlannedTx(u.plan(), txplan.WithData(common.FromHex(txIntentBindings.EventloggerBin)))
	receipt, err := tx.Included.Eval(ctx)
	if err != nil {
		return common.Address{}, err
	}
	return receipt.ContractAddress, nil
}

// initResult holds the init message and its receipt.
type initResult struct {
	tx      *txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput]
	receipt *types.Receipt
	msg     suptypes.Message
}

func (u *user) sendInitMessage(ctx context.Context, rng *rand.Rand, eventLogger common.Address) (*initResult, error) {
	topics := make([][32]byte, 2)
	for i := range topics {
		copy(topics[i][:], testutils.RandomData(rng, 32))
	}

	trigger := &txintent.InitTrigger{
		Emitter:    eventLogger,
		Topics:     topics,
		OpaqueData: testutils.RandomData(rng, 10),
	}
	tx := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](u.plan())
	tx.Content.Set(trigger)
	receipt, err := tx.PlannedTx.Included.Eval(ctx)
	if err != nil {
		return nil, err
	}

	result, err := tx.Result.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("evaluate init result: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("init tx produced no interop log entries")
	}

	return &initResult{tx: tx, receipt: receipt, msg: result.Entries[0]}, nil
}

func (u *user) sendExecMessage(ctx context.Context, init *initResult) (*types.Receipt, error) {
	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](u.plan())
	tx.Content.DependOn(&init.tx.Result)
	tx.Content.Fn(txintent.ExecuteIndexed(predeploys.CrossL2InboxAddr, &init.tx.Result, 0))
	return tx.PlannedTx.Included.Eval(ctx)
}

func resolvePrivateKey(cliCtx *cli.Context) (*ecdsa.PrivateKey, error) {
	if pkHex := cliCtx.String(PrivateKeyFlag.Name); pkHex != "" {
		return crypto.HexToECDSA(pkHex)
	}
	hd, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	if err != nil {
		return nil, fmt.Errorf("new mnemonic dev keys: %w", err)
	}
	key := devkeys.UserKey(cliCtx.Uint(DevKeyIndexFlag.Name))
	return hd.Secret(key)
}

func waitForTimestampPast(ctx context.Context, c *chain, timestamp uint64) error {
	deadline := time.Now().Add(60 * time.Second)
	for {
		head, err := c.ethClient.BlockRefByLabel(ctx, eth.Unsafe)
		if err == nil && head.Time > timestamp {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s head timestamp > %d", c.name, timestamp)
		}
		time.Sleep(time.Second)
	}
}
