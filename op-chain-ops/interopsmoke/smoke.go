// Package interopsmoke implements interop smoke tests that run against the
// RPCs of live interoperable L2 chains: chain identity, ETH transfers,
// cross-chain ETH bridging, and valid/invalid executing-message checks.
//
// It is exposed both as the standalone op-chain-ops/cmd/interop-smoke tool and
// as the `smoke-interop` subcommand of op-up.
package interopsmoke

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopbridge"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	txIntentBindings "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

const smokeWaitTimeout = 60 * time.Second
const defaultReorgTimeout = 20 * time.Minute

const (
	l2URLFlagName             = "l2-rpc"
	privateKeyFlagName        = "private-key"
	invalidBlocksFlagName     = "blocks"
	invalidTxPerBlockFlagName = "tx-per-block"
	reorgTimeoutFlagName      = "reorg-timeout"
	directionFlagName         = "direction"
	requireCascadeFlagName    = "require-cascade"
	iterationsFlagName        = "iterations"
)

// bridgeTimeout bounds a single cross-chain bridge (send, relay, and balance check).
const bridgeTimeout = 2 * time.Minute

func smokeFlags(envPrefix string) []cli.Flag {
	return cliapp.ProtectFlags([]cli.Flag{
		&cli.StringSliceFlag{
			Name:    l2URLFlagName,
			Usage:   "RPC URL for an interoperable L2. Repeat for each chain.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_L2_RPC"),
		},
		&cli.StringFlag{
			Name:    privateKeyFlagName,
			Usage:   "Private key to fund smoke-test transactions. If empty, uses the default dev key.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_PRIVATE_KEY"),
		},
		&cli.UintFlag{
			Name:    iterationsFlagName,
			Usage:   "Number of times to repeat the selected smoke flow.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_ITERATIONS"),
			Value:   1,
		},
	})
}

type remoteChain struct {
	name      string
	url       string
	rpc       opclient.RPC
	ethClient apis.EthClient
	chainID   eth.ChainID
}

type remoteUser struct {
	chain   *remoteChain
	privKey *ecdsa.PrivateKey
	address common.Address
}

type initMessage struct {
	Tx      *txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput]
	Receipt *types.Receipt
}

type execMessage struct {
	Init    *initMessage
	Tx      *txintent.IntentTx[*txintent.ExecTrigger, *txintent.InteropOutput]
	Receipt *types.Receipt
}

// sentMessage is a message sent through the L2ToL2CrossDomainMessenger.
type sentMessage struct {
	Tx      *txintent.IntentTx[*txintent.SendTrigger, *txintent.InteropOutput]
	Receipt *types.Receipt
}

// relayedMessage is a relayMessage call on the L2ToL2CrossDomainMessenger. Its
// receipt holds the logs emitted by the relayed call's target, which are
// themselves initiating messages.
type relayedMessage struct {
	Tx      *txintent.IntentTx[*txintent.RelayTrigger, *txintent.InteropOutput]
	Receipt *types.Receipt
}

type smokeEnv struct {
	ctx               context.Context
	stderr            io.Writer
	logger            log.Logger
	chains            []*remoteChain
	users             []*remoteUser
	chainA            *remoteChain
	chainB            *remoteChain
	userA             *remoteUser
	userB             *remoteUser
	invalidBlocks     uint
	invalidTxPerBlock uint
	invalidDirection  string
	reorgTimeout      time.Duration
	requireCascade    bool
}

type chainPair struct {
	name               string
	initUser, execUser *remoteUser
}

func (m *initMessage) BlockNumber() uint64 {
	return bigs.Uint64Strict(m.Receipt.BlockNumber)
}

func (m *initMessage) BlockHash() common.Hash {
	return m.Receipt.BlockHash
}

func (m *execMessage) BlockNumber() uint64 {
	return bigs.Uint64Strict(m.Receipt.BlockNumber)
}

func (m *execMessage) BlockHash() common.Hash {
	return m.Receipt.BlockHash
}

func (m *relayedMessage) BlockNumber() uint64 {
	return bigs.Uint64Strict(m.Receipt.BlockNumber)
}

func (m *relayedMessage) BlockHash() common.Hash {
	return m.Receipt.BlockHash
}

func (u *remoteUser) plan() txplan.Option {
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

func (u *remoteUser) transfer(ctx context.Context, to common.Address, amount eth.ETH) (*txplan.PlannedTx, error) {
	tx := txplan.NewPlannedTx(
		u.plan(),
		txplan.WithTo(&to),
		txplan.WithValue(amount),
	)
	_, err := tx.Success.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (u *remoteUser) deployEventLogger(ctx context.Context) (common.Address, error) {
	tx := txplan.NewPlannedTx(u.plan(), txplan.WithData(common.FromHex(txIntentBindings.EventloggerBin)))
	receipt, err := tx.Included.Eval(ctx)
	if err != nil {
		return common.Address{}, err
	}
	return receipt.ContractAddress, nil
}

func (u *remoteUser) sendRandomInitMessage(ctx context.Context, rng *rand.Rand, eventLogger common.Address, topicCount, dataLen int) (*initMessage, error) {
	if topicCount > 4 {
		topicCount = 4
	}
	if topicCount < 1 {
		topicCount = 1
	}
	if dataLen < 1 {
		dataLen = 1
	}

	topics := make([][32]byte, topicCount)
	for i := range topics {
		copy(topics[i][:], testutils.RandomData(rng, 32))
	}

	trigger := &txintent.InitTrigger{
		Emitter:    eventLogger,
		Topics:     topics,
		OpaqueData: testutils.RandomData(rng, dataLen),
	}
	tx := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](u.plan())
	tx.Content.Set(trigger)
	receipt, err := tx.PlannedTx.Included.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return &initMessage{Tx: tx, Receipt: receipt}, nil
}

func (u *remoteUser) sendExecMessage(ctx context.Context, initMsg *initMessage) (*execMessage, error) {
	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](u.plan())
	tx.Content.DependOn(&initMsg.Tx.Result)
	tx.Content.Fn(txintent.ExecuteIndexed(predeploys.CrossL2InboxAddr, &initMsg.Tx.Result, 0))
	receipt, err := tx.PlannedTx.Included.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return &execMessage{
		Init:    initMsg,
		Tx:      tx,
		Receipt: receipt,
	}, nil
}

func (u *remoteUser) sendInvalidExecMessage(ctx context.Context, initMsg *initMessage) (*execMessage, error) {
	result, err := initMsg.Tx.Result.Eval(ctx)
	if err != nil {
		return nil, err
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("init tx produced no interop entries")
	}

	msg := result.Entries[0]
	msg.Identifier.LogIndex++

	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](u.plan())
	tx.Content.DependOn(&initMsg.Tx.Result)
	tx.Content.Fn(func(context.Context) (*txintent.ExecTrigger, error) {
		return &txintent.ExecTrigger{
			Executor: predeploys.CrossL2InboxAddr,
			Msg:      msg,
		}, nil
	})

	receipt, err := tx.PlannedTx.Included.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return &execMessage{
		Init:    initMsg,
		Tx:      tx,
		Receipt: receipt,
	}, nil
}

// sendMessage sends a cross-chain message via the L2ToL2CrossDomainMessenger.
// When relayed on the destination chain, the messenger calls target with
// calldata.
func (u *remoteUser) sendMessage(ctx context.Context, dest eth.ChainID, target common.Address, calldata []byte) (*sentMessage, error) {
	tx := txintent.NewIntent[*txintent.SendTrigger, *txintent.InteropOutput](u.plan())
	tx.Content.Set(&txintent.SendTrigger{
		Emitter:         predeploys.L2toL2CrossDomainMessengerAddr,
		DestChainID:     dest,
		Target:          target,
		RelayedCalldata: calldata,
	})
	receipt, err := tx.PlannedTx.Included.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return &sentMessage{Tx: tx, Receipt: receipt}, nil
}

// sendInvalidRelayMessage relays msg with a corrupted message identifier, so
// the relay is an invalid executing message. The identifier's log index is
// bumped past the real log, mirroring sendInvalidExecMessage. The payload is
// left intact so the relay still executes and calls its target: invalidity is
// enforced by block-level invalidation, not by the EVM.
func (u *remoteUser) sendInvalidRelayMessage(ctx context.Context, msg *sentMessage) (*relayedMessage, error) {
	result, err := msg.Tx.Result.Eval(ctx)
	if err != nil {
		return nil, err
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("send tx produced no interop entries")
	}
	if len(msg.Receipt.Logs) == 0 {
		return nil, fmt.Errorf("send tx produced no logs")
	}

	entryIdx := firstLogFrom(msg.Receipt.Logs, predeploys.L2toL2CrossDomainMessengerAddr)
	if entryIdx < 0 || entryIdx >= len(result.Entries) {
		return nil, fmt.Errorf("send tx produced no messenger entry")
	}
	entry := result.Entries[entryIdx]
	entry.Identifier.LogIndex++
	payload := messages.LogToMessagePayload(msg.Receipt.Logs[entryIdx])

	tx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](u.plan())
	tx.Content.DependOn(&msg.Tx.Result)
	tx.Content.Fn(func(context.Context) (*txintent.RelayTrigger, error) {
		return &txintent.RelayTrigger{
			ExecTrigger: txintent.ExecTrigger{
				Executor: predeploys.L2toL2CrossDomainMessengerAddr,
				Msg:      entry,
			},
			Payload: payload,
		}, nil
	})

	receipt, err := tx.PlannedTx.Included.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return &relayedMessage{Tx: tx, Receipt: receipt}, nil
}

// execEntry sends an executing message for an already-known initiating message.
func (u *remoteUser) execEntry(ctx context.Context, entry messages.Message) (*types.Receipt, error) {
	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](u.plan())
	tx.Content.Set(&txintent.ExecTrigger{
		Executor: predeploys.CrossL2InboxAddr,
		Msg:      entry,
	})
	return tx.PlannedTx.Included.Eval(ctx)
}

// firstLogFrom returns the index of the first log emitted by origin, or -1.
// Entry indexes in an InteropOutput align one-to-one with receipt log indexes.
func firstLogFrom(logs []*types.Log, origin common.Address) int {
	for i, l := range logs {
		if l.Address == origin {
			return i
		}
	}
	return -1
}

func newLogger(ctx context.Context, stderr io.Writer) log.Logger {
	logHandler := oplog.NewLogHandler(stderr, oplog.DefaultCLIConfig())
	logHandler = logfilter.WrapFilterHandler(logHandler)
	logHandler.(logfilter.FilterHandler).Set(logfilter.DefaultMute())
	logHandler = logfilter.WrapContextHandler(logHandler)
	logger := log.NewLogger(logHandler)
	oplog.SetGlobalLogHandler(logHandler)
	logger.SetContext(ctx)
	return logger
}

func newSmokeEnv(ctx context.Context, stderr io.Writer, l2URLs []string, privateKey string) (*smokeEnv, func(), error) {
	if err := validateL2URLs(l2URLs); err != nil {
		return nil, nil, err
	}
	logger := newLogger(ctx, stderr)

	chains := make([]*remoteChain, 0, len(l2URLs))
	for i, l2URL := range l2URLs {
		chain, err := connectRemoteChain(ctx, logger, fmt.Sprintf("L2%c", 'A'+i), l2URL)
		if err != nil {
			for _, connectedChain := range chains {
				connectedChain.ethClient.Close()
			}
			return nil, nil, err
		}
		chains = append(chains, chain)
	}
	if err := validateChainIDs(&smokeEnv{chains: chains}); err != nil {
		for _, chain := range chains {
			chain.ethClient.Close()
		}
		return nil, nil, err
	}

	privKey, address, err := resolveSmokeKey(privateKey)
	if err != nil {
		for _, chain := range chains {
			chain.ethClient.Close()
		}
		return nil, nil, err
	}
	users := make([]*remoteUser, 0, len(chains))
	for _, chain := range chains {
		users = append(users, &remoteUser{chain: chain, privKey: privKey, address: address})
	}

	env := &smokeEnv{
		ctx:    ctx,
		stderr: stderr,
		logger: logger,
		chains: chains,
		users:  users,
		chainA: chains[0],
		chainB: chains[1],
		userA:  users[0],
		userB:  users[1],
	}
	cleanup := func() {
		for _, chain := range chains {
			chain.ethClient.Close()
		}
	}
	return env, cleanup, nil
}

func validateL2URLs(l2URLs []string) error {
	if len(l2URLs) < 2 {
		return fmt.Errorf("at least two L2 RPC URLs are required")
	}
	return nil
}

func validateChainIDs(env *smokeEnv) error {
	chainNames := make(map[string]string, len(env.chains))
	for _, chain := range env.chains {
		chainID := chain.chainID.String()
		if previous, ok := chainNames[chainID]; ok {
			return fmt.Errorf("%s and %s have the same chain ID: %s", previous, chain.name, chainID)
		}
		chainNames[chainID] = chain.name
	}
	return nil
}

func connectRemoteChain(ctx context.Context, logger log.Logger, name, url string) (*remoteChain, error) {
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
	return &remoteChain{
		name:      name,
		url:       url,
		rpc:       rpcCl,
		ethClient: ethCl,
		chainID:   eth.ChainIDFromBig(chainIDBig),
	}, nil
}

func defaultSmokeKey() (*ecdsa.PrivateKey, common.Address, error) {
	hd, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("new mnemonic dev keys: %w", err)
	}
	const funderIndex = 10_000
	key := devkeys.UserKey(funderIndex)
	privKey, err := hd.Secret(key)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("secret: %w", err)
	}
	address := crypto.PubkeyToAddress(privKey.PublicKey)
	return privKey, address, nil
}

func resolveSmokeKey(privateKey string) (*ecdsa.PrivateKey, common.Address, error) {
	if privateKey == "" {
		return defaultSmokeKey()
	}
	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("parse private key: %w", err)
	}
	return privKey, crypto.PubkeyToAddress(privKey.PublicKey), nil
}

func withSmokeEnv(cliCtx *cli.Context, name string, fn func(env *smokeEnv) error) error {
	ctx := cliCtx.Context
	stderr := cliCtx.App.ErrWriter
	l2URLs := cliCtx.StringSlice(l2URLFlagName)
	privateKey := cliCtx.String(privateKeyFlagName)
	iterations := cliCtx.Uint(iterationsFlagName)
	if err := validateIterations(iterations); err != nil {
		return err
	}

	fmt.Fprintf(stderr, "\nSmoke: %s\n\n", name)

	env, cleanup, err := newSmokeEnv(ctx, stderr, l2URLs, privateKey)
	if err != nil {
		return err
	}
	defer cleanup()

	for _, chain := range env.chains {
		fmt.Fprintf(stderr, "%s RPC: %s (chain ID %s)\n", chain.name, chain.url, chain.chainID)
	}
	fmt.Fprintf(stderr, "Smoke Sender Address: %s\n\n", env.userA.address)

	iteration := uint(0)
	if err := runIterations(iterations, func() error {
		iteration++
		if iterations > 1 {
			fmt.Fprintf(stderr, "\nIteration %d/%d\n", iteration, iterations)
		}
		return fn(env)
	}); err != nil {
		fmt.Fprintf(stderr, "\nFAIL: %s (%v)\n", name, err)
		return err
	}
	fmt.Fprintf(stderr, "\nPASS: %s\n", name)
	return nil
}

func validateIterations(iterations uint) error {
	if iterations == 0 {
		return fmt.Errorf("iterations must be greater than zero")
	}
	return nil
}

func runIterations(iterations uint, fn func() error) error {
	if err := validateIterations(iterations); err != nil {
		return err
	}
	for i := uint(0); i < iterations; i++ {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

// Command returns the `smoke-interop` command tree for embedding in a host
// CLI such as op-up. envPrefix scopes the flag environment variables
// (e.g. "OP_UP" -> OP_UP_SMOKE_L2_RPC).
func Command(envPrefix string) *cli.Command {
	return &cli.Command{
		Name:        "smoke-interop",
		Usage:       "run interop smoke tests against remote chain RPCs",
		Subcommands: Subcommands(envPrefix),
	}
}

// Subcommands returns the individual smoke-test commands, for mounting at the
// top level of a standalone CLI.
func Subcommands(envPrefix string) []*cli.Command {
	flags := smokeFlags(envPrefix)

	return []*cli.Command{
		{
			Name:  "all",
			Usage: "run all smoke tests sequentially",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "All Tests", smokeAll)
			},
		},
		{
			Name:  "identity",
			Usage: "verify every chain has a unique chain ID",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "Chain Identity", smokeIdentity)
			},
		},
		{
			Name:  "transfer",
			Usage: "send an ETH transfer on every chain",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "ETH Transfers", smokeTransfer)
			},
		},
		{
			Name:  "bridge",
			Usage: "bridge ETH over every ordered chain pair via SuperchainETHBridge",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "Cross-Chain ETH Bridge", smokeBridge)
			},
		},
		{
			Name:  "valid-message",
			Usage: "send a valid executing message over every ordered chain pair and verify it stays in-chain",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "Valid Exec Message", smokeValidMessage)
			},
		},
		{
			Name:  "invalid-message",
			Usage: "send invalid executing messages over every ordered chain pair and verify they are reorged out",
			Flags: append(flags,
				&cli.UintFlag{
					Name:    invalidBlocksFlagName,
					Usage:   "Number of blocks per chain containing invalid transactions.",
					EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_BLOCKS"),
					Value:   1,
				},
				&cli.UintFlag{
					Name:    invalidTxPerBlockFlagName,
					Usage:   "Number of invalid transactions to land per block, on each chain.",
					EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_TX_PER_BLOCK"),
					Value:   1,
				},
				&cli.StringFlag{
					Name:  directionFlagName,
					Usage: "One directed chain pair to test, such as A->B. By default, tests every ordered pair.",
				},
				&cli.DurationFlag{
					Name:    reorgTimeoutFlagName,
					Usage:   "Maximum time to wait for each invalid block to be reorged.",
					EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_REORG_TIMEOUT"),
					Value:   defaultReorgTimeout,
				},
			),
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "Invalid Exec Message (reorg)", func(env *smokeEnv) error {
					env.invalidBlocks = cliCtx.Uint(invalidBlocksFlagName)
					env.invalidTxPerBlock = cliCtx.Uint(invalidTxPerBlockFlagName)
					env.invalidDirection = cliCtx.String(directionFlagName)
					env.reorgTimeout = cliCtx.Duration(reorgTimeoutFlagName)
					return smokeInvalidMessage(env)
				})
			},
		},
		{
			Name:  "chained-invalid-message",
			Usage: "chain a valid executing message to an invalid one and verify invalidation propagates",
			Flags: append(flags,
				&cli.DurationFlag{
					Name:    reorgTimeoutFlagName,
					Usage:   "Total time budget for the whole reorg cascade, shared across every block waited on.",
					EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_REORG_TIMEOUT"),
					Value:   defaultReorgTimeout,
				},
				&cli.BoolFlag{
					Name:    requireCascadeFlagName,
					Usage:   "Fail if Chain A included neither dependent message, since then transitive invalidation was never exercised.",
					EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_REQUIRE_CASCADE"),
				},
			),
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "Chained Invalid Exec Message (transitive reorg)", func(env *smokeEnv) error {
					env.reorgTimeout = cliCtx.Duration(reorgTimeoutFlagName)
					env.requireCascade = cliCtx.Bool(requireCascadeFlagName)
					return smokeChainedInvalidMessage(env)
				})
			},
		},
	}
}

func smokeAll(env *smokeEnv) error {
	env.invalidBlocks = 1
	env.invalidTxPerBlock = 1
	env.reorgTimeout = defaultReorgTimeout
	tests := []struct {
		name string
		fn   func(env *smokeEnv) error
	}{
		{"Chain Identity", smokeIdentity},
		{"ETH Transfers", smokeTransfer},
		{"Cross-Chain ETH Bridge", smokeBridge},
		{"Valid Exec Message", smokeValidMessage},
		{"Invalid Exec Message (reorg)", smokeInvalidMessage},
		// chained-invalid-message is deliberately not here: it waits on a second
		// reorg cascade after this one, which would roughly double the worst-case
		// runtime of `all`. Run it as its own subcommand.
	}

	var failed []string
	for _, test := range tests {
		fmt.Fprintf(env.stderr, "--- %s\n", test.name)
		if err := test.fn(env); err != nil {
			fmt.Fprintf(env.stderr, "    FAIL: %v\n", err)
			failed = append(failed, test.name)
		} else {
			fmt.Fprintf(env.stderr, "    PASS\n")
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed tests: %v", failed)
	}
	return nil
}

func smokeIdentity(env *smokeEnv) error {
	if err := validateChainIDs(env); err != nil {
		return err
	}
	for _, chain := range env.chains {
		fmt.Fprintf(env.stderr, "    %s: %s\n", chain.name, chain.chainID)
	}
	return nil
}

func smokeTransfer(env *smokeEnv) error {
	for _, user := range env.users {
		recipient := randomAddress()
		if _, err := user.transfer(env.ctx, recipient, eth.OneHundredthEther); err != nil {
			return fmt.Errorf("%s transfer failed: %w", user.chain.name, err)
		}
		if err := waitForBalance(env.ctx, user.chain, recipient, eth.OneHundredthEther); err != nil {
			return err
		}
		fmt.Fprintf(env.stderr, "    %s transfer: OK\n", user.chain.name)
	}
	return nil
}

func smokeBridge(env *smokeEnv) error {
	pairs, err := orderedPairs(env)
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		amount := eth.OneHundredthEther
		ctx, cancel := context.WithTimeout(env.ctx, bridgeTimeout)
		result, err := interopbridge.BridgeETHWithResult(ctx, env.logger, pair.initUser.chain.ethClient, pair.execUser.chain.ethClient,
			pair.execUser.chain.chainID, pair.initUser.privKey, amount, randomAddress())
		cancel()
		if err != nil {
			return fmt.Errorf("%s bridge failed: %w", pair.name, err)
		}
		fmt.Fprintf(env.stderr, "    [%s] %s\n", pair.name,
			flowOutput("Bridge", pair.initUser.chain, pair.execUser.chain, &amount, result.RelayReceipt))
	}
	return nil
}

func smokeValidMessage(env *smokeEnv) error {
	pairs, err := orderedPairs(env)
	if err != nil {
		return err
	}
	for i, pair := range pairs {
		if err := smokeValidMessagePair(env, pair, rand.New(rand.NewSource(int64(42+i)))); err != nil {
			return err
		}
	}
	return nil
}

func smokeValidMessagePair(env *smokeEnv, pair chainPair, rng *rand.Rand) error {
	eventLogger, err := pair.initUser.deployEventLogger(env.ctx)
	if err != nil {
		return fmt.Errorf("%s: deploy EventLogger on %s: %w", pair.name, pair.initUser.chain.name, err)
	}

	initMsg, err := pair.initUser.sendRandomInitMessage(env.ctx, rng, eventLogger, 2, 10)
	if err != nil {
		return fmt.Errorf("%s: send init message: %w", pair.name, err)
	}
	fmt.Fprintf(env.stderr, "    [%s] %s\n", pair.name,
		messageLandingOutput("Initiating message", pair.initUser.chain, pair.execUser.chain, pair.initUser.chain, initMsg.Receipt))

	if _, err := waitForNextBlock(env.ctx, pair.execUser.chain); err != nil {
		return fmt.Errorf("%s: wait for %s block: %w", pair.name, pair.execUser.chain.name, err)
	}
	execMsg, err := pair.execUser.sendExecMessage(env.ctx, initMsg)
	if err != nil {
		return fmt.Errorf("%s: send exec message: %w", pair.name, err)
	}
	if execMsg.Receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("%s: exec tx reverted", pair.name)
	}

	execBlockNum := execMsg.BlockNumber()
	execBlockHash := execMsg.BlockHash()
	fmt.Fprintf(env.stderr, "    [%s] %s\n", pair.name,
		messageLandingOutput("Executing message", pair.initUser.chain, pair.execUser.chain, pair.execUser.chain, execMsg.Receipt))

	if err := waitForHeadAtLeast(env.ctx, pair.execUser.chain, execBlockNum+2); err != nil {
		return fmt.Errorf("%s: wait for %s head: %w", pair.name, pair.execUser.chain.name, err)
	}

	currentBlock, err := pair.execUser.chain.ethClient.BlockRefByNumber(env.ctx, execBlockNum)
	if err != nil {
		return fmt.Errorf("%s: fetch %s block %d: %w", pair.name, pair.execUser.chain.name, execBlockNum, err)
	}
	if currentBlock.Hash != execBlockHash {
		return fmt.Errorf("%s: %s block was replaced: expected %s, got %s", pair.name, pair.execUser.chain.name, execBlockHash, currentBlock.Hash)
	}
	if err := assertTxInBlock(env.ctx, pair.execUser.chain, execBlockNum, execMsg.Receipt.TxHash); err != nil {
		return fmt.Errorf("%s: %w", pair.name, err)
	}
	fmt.Fprintf(env.stderr, "    [%s] %s block remained canonical after head advanced past it\n", pair.name, pair.execUser.chain.name)
	return nil
}

func flowOutput(flow string, source, destination *remoteChain, amount *eth.ETH, receipt *types.Receipt) string {
	output := fmt.Sprintf("%s: source chain ID %s, destination chain ID %s", flow, source.chainID, destination.chainID)
	if amount != nil {
		output += fmt.Sprintf(", amount %s wei", amount.ToBig())
	}
	return fmt.Sprintf("%s, tx %s, included in block %d", output, receipt.TxHash, bigs.Uint64Strict(receipt.BlockNumber))
}

func messageLandingOutput(message string, source, destination, landedOn *remoteChain, receipt *types.Receipt) string {
	role := "chain"
	if landedOn == source {
		role = "source chain"
	} else if landedOn == destination {
		role = "destination chain"
	}
	return fmt.Sprintf("%s landed on %s %s (chain ID %s), source chain ID %s, destination chain ID %s, tx %s, included in block %d",
		message, role, landedOn.name, landedOn.chainID, source.chainID, destination.chainID, receipt.TxHash, bigs.Uint64Strict(receipt.BlockNumber))
}

type invalidInclusion struct {
	txHash, blockHash common.Hash
	blockNum          uint64
}

// invalidDirection is one init-chain -> exec-chain pairing: init messages are
// emitted on the init user's chain and the invalid executing messages land on
// the exec user's chain.
type invalidDirection struct {
	name        string
	initUser    *remoteUser
	execUser    *remoteUser
	rng         *rand.Rand
	eventLogger common.Address
	pending     []*initMessage
	inclusions  []invalidInclusion
}

// smokeInvalidMessage lands invalid executing messages for every ordered chain
// pair. Each phase is serial because multiple pairs may send from the same
// source chain.
func smokeInvalidMessage(env *smokeEnv) error {
	if err := validateInvalidMessageOptions(env.invalidBlocks, env.invalidTxPerBlock); err != nil {
		return err
	}
	if env.reorgTimeout <= 0 {
		return fmt.Errorf("reorg-timeout must be greater than zero")
	}
	stderr := &lockedWriter{w: env.stderr}

	dirs, err := invalidDirections(env)
	if err != nil {
		return err
	}

	if err := eachDirection(dirs, func(d *invalidDirection) error {
		eventLogger, err := d.initUser.deployEventLogger(env.ctx)
		if err != nil {
			return fmt.Errorf("%s: deploy EventLogger on %s: %w", d.name, d.initUser.chain.name, err)
		}
		d.eventLogger = eventLogger
		return nil
	}); err != nil {
		return err
	}

	for block := uint(0); block < env.invalidBlocks; block++ {
		if err := eachDirection(dirs, func(d *invalidDirection) error {
			d.pending = make([]*initMessage, 0, env.invalidTxPerBlock)
			for tx := uint(0); tx < env.invalidTxPerBlock; tx++ {
				initMsg, err := d.initUser.sendRandomInitMessage(env.ctx, d.rng, d.eventLogger, 2, 10)
				if err != nil {
					return fmt.Errorf("%s: send init message: %w", d.name, err)
				}
				d.pending = append(d.pending, initMsg)
				fmt.Fprintf(stderr, "    [%s] %s\n", d.name, messageLandingOutput("Initiating message", d.initUser.chain, d.execUser.chain, d.initUser.chain, initMsg.Receipt))
			}
			return nil
		}); err != nil {
			return err
		}

		if err := eachDirection(dirs, func(d *invalidDirection) error {
			execChain := d.execUser.chain
			if _, err := waitForNextBlock(env.ctx, execChain); err != nil {
				return err
			}
			for _, initMsg := range d.pending {
				invalidExec, err := d.execUser.sendInvalidExecMessage(env.ctx, initMsg)
				if err != nil {
					return fmt.Errorf("%s: send invalid exec message: %w", d.name, err)
				}
				if invalidExec.Receipt.Status != types.ReceiptStatusSuccessful {
					return fmt.Errorf("%s: invalid exec tx reverted before inclusion", d.name)
				}
				inclusion := invalidInclusion{invalidExec.Receipt.TxHash, invalidExec.BlockHash(), invalidExec.BlockNumber()}
				d.inclusions = append(d.inclusions, inclusion)
				fmt.Fprintf(stderr, "    [%s] %s\n", d.name, messageLandingOutput("Invalid executing message", d.initUser.chain, execChain, execChain, invalidExec.Receipt))
			}
			return nil
		}); err != nil {
			return err
		}
	}

	// Check every invalid block on every chain, and report all failures rather
	// than stopping at the first one.
	type reorgCheck struct {
		dir       *invalidDirection
		inclusion invalidInclusion
	}
	var checks []reorgCheck
	for _, d := range dirs {
		for _, inclusion := range d.inclusions {
			checks = append(checks, reorgCheck{dir: d, inclusion: inclusion})
		}
	}

	results := make([]reorgResult, len(checks))
	var wg sync.WaitGroup
	waitStart := time.Now()
	for i, check := range checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			execChain := check.dir.execUser.chain
			start := time.Now()
			err := waitForReorgedOut(env.ctx, stderr, execChain, check.inclusion.blockNum, check.inclusion.blockHash, check.inclusion.txHash, env.reorgTimeout)
			results[i] = reorgResult{
				dir:      check.dir.name,
				chain:    execChain.name,
				blockNum: check.inclusion.blockNum,
				txHash:   check.inclusion.txHash,
				elapsed:  time.Since(start),
				err:      err,
			}
		}()
	}
	wg.Wait()
	return reportReorgResults(env.stderr, results, time.Since(waitStart))
}

type reorgResult struct {
	dir      string
	chain    string
	blockNum uint64
	txHash   common.Hash
	elapsed  time.Duration
	err      error
}

// reportReorgResults prints a per-block summary table and returns the joined
// errors of every block that was not reorged out.
func reportReorgResults(stderr io.Writer, results []reorgResult, total time.Duration) error {
	var errs []error
	reorged := 0
	stalled := 0
	chains := make(map[string]struct{})
	tw := tabwriter.NewWriter(stderr, 0, 0, 2, ' ', 0)
	fmt.Fprint(tw, "\n    DIRECTION\tCHAIN\tBLOCK\tINVALID TX\tRESULT\tELAPSED\n")
	for _, r := range results {
		chains[r.chain] = struct{}{}
		var status string
		switch {
		case r.err == nil:
			status = "REORGED"
			reorged++
		case errors.Is(r.err, errChainStalled):
			// The replacement was observed, so the invalidation worked. Counting
			// this as "not reorged" would point at the wrong component.
			status = "REORGED, CHAIN STALLED"
			reorged++
			stalled++
			errs = append(errs, fmt.Errorf("%s on %s: %w", r.dir, r.chain, r.err))
		default:
			status = "NOT REORGED"
			errs = append(errs, fmt.Errorf("%s on %s: %w", r.dir, r.chain, r.err))
		}
		fmt.Fprintf(tw, "    %s\t%s\t%d\t%s\t%s\t%s\n",
			r.dir, r.chain, r.blockNum, r.txHash.TerminalString(), status, r.elapsed.Round(time.Second))
	}
	// A flush failure only loses the table; the reorg failures still have to be
	// reported, so it joins the errors rather than replacing them.
	if err := tw.Flush(); err != nil {
		errs = append(errs, fmt.Errorf("flush summary table: %w", err))
	}
	fmt.Fprintf(stderr, "\n    %d/%d invalid txs reorged out across %d chain(s) in %s\n",
		reorged, len(results), len(chains), total.Round(time.Second))
	if stalled > 0 {
		fmt.Fprintf(stderr, "    %d block(s) were replaced correctly, but the chain then stopped producing blocks.\n", stalled)
		fmt.Fprintf(stderr, "    Invalidation looks correct; the liveness of the chain afterwards does not.\n")
	}
	fmt.Fprintln(stderr)

	return errors.Join(errs...)
}

func invalidDirections(env *smokeEnv) ([]*invalidDirection, error) {
	pairs, err := orderedPairs(env)
	if err != nil {
		return nil, err
	}
	if env.invalidDirection != "" {
		parts := strings.Split(env.invalidDirection, "->")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid-message direction must be in the form A->B")
		}
		if parts[0] == parts[1] {
			return nil, fmt.Errorf("invalid-message direction source and destination must differ: %s", env.invalidDirection)
		}
		for i, pair := range pairs {
			if pair.name == env.invalidDirection {
				pairs = pairs[i : i+1]
				break
			}
		}
		if len(pairs) != 1 || pairs[0].name != env.invalidDirection {
			return nil, fmt.Errorf("invalid-message direction %q is unknown", env.invalidDirection)
		}
	}
	dirs := make([]*invalidDirection, 0, len(pairs))
	for i, pair := range pairs {
		dirs = append(dirs, &invalidDirection{
			name:     pair.name,
			initUser: pair.initUser,
			execUser: pair.execUser,
			rng:      rand.New(rand.NewSource(int64(99 + i))),
		})
	}
	return dirs, nil
}

func orderedPairs(env *smokeEnv) ([]chainPair, error) {
	if len(env.users) < 2 {
		return nil, fmt.Errorf("at least two L2 RPC URLs are required")
	}
	var pairs []chainPair
	for i, initUser := range env.users {
		for j, execUser := range env.users {
			if i == j {
				continue
			}
			pairs = append(pairs, chainPair{
				name:     fmt.Sprintf("%s->%s", strings.TrimPrefix(initUser.chain.name, "L2"), strings.TrimPrefix(execUser.chain.name, "L2")),
				initUser: initUser,
				execUser: execUser,
			})
		}
	}
	return pairs, nil
}

// eachDirection runs fn for every direction serially, returning the first error.
func eachDirection(dirs []*invalidDirection, fn func(d *invalidDirection) error) error {
	for _, d := range dirs {
		if err := fn(d); err != nil {
			return err
		}
	}
	return nil
}

// lockedWriter serializes writes so concurrent directions do not interleave
// mid-line output.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// smokeChainedInvalidMessage verifies that invalidation is transitive.
//
// Chain A sends a cross-chain message targeting an EventLogger on chain B.
// Chain B relays it with a corrupted identifier, so the relay is invalid. The
// relay still executes, calling EventLogger.emitLog, which emits a brand new
// initiating message on chain B that exists only because the invalid relay ran.
// Chain A then sends a well-formed executing message for that new message.
//
// When chain B's block is replaced for containing the invalid relay, the new
// message disappears with it, so chain A's executing message must be reorged
// out too, even though it was valid when sent.
//
// A control message that does not depend on the invalid block is checked to
// still be present at the end, so that a node which simply discards a swathe of
// recent chain A blocks does not pass.
func smokeChainedInvalidMessage(env *smokeEnv) error {
	if env.reorgTimeout <= 0 {
		return fmt.Errorf("reorg-timeout must be greater than zero")
	}
	rng := rand.New(rand.NewSource(123))

	eventLoggerB, err := env.userB.deployEventLogger(env.ctx)
	if err != nil {
		return fmt.Errorf("deploy EventLogger on Chain B: %w", err)
	}

	// Control arm: an ordinary Chain B -> Chain A message, emitted before the
	// invalid relay so it cannot be part of the reorged lineage.
	controlInit, err := env.userB.sendRandomInitMessage(env.ctx, rng, eventLoggerB, 2, 10)
	if err != nil {
		return fmt.Errorf("send control init message: %w", err)
	}
	controlOut, err := controlInit.Tx.Result.Eval(env.ctx)
	if err != nil {
		return fmt.Errorf("read control init entries: %w", err)
	}
	if len(controlOut.Entries) == 0 {
		return fmt.Errorf("control init tx produced no interop entries")
	}
	if _, err := waitForNextBlock(env.ctx, env.chainA); err != nil {
		return err
	}
	controlReceipt, err := env.userA.execEntry(env.ctx, controlOut.Entries[0])
	if err != nil {
		return fmt.Errorf("send control exec message: %w", err)
	}
	controlBlockNum := bigs.Uint64Strict(controlReceipt.BlockNumber)
	controlBlockHash := controlReceipt.BlockHash
	fmt.Fprintf(env.stderr, "    Control exec message landed on Chain A block %d (%s)\n", controlBlockNum, controlBlockHash)

	// Force the control message into a strictly earlier Chain A block than the
	// chained exec message. Sharing a block would make the cascade correctly
	// reorg both, and the control assertion would misreport an over-reorg.
	if err := waitForHeadAtLeast(env.ctx, env.chainA, controlBlockNum+1); err != nil {
		return err
	}

	// Step 1: Chain A sends a message whose relayed call emits a log on Chain B.
	topics := make([][32]byte, 2)
	for i := range topics {
		copy(topics[i][:], testutils.RandomData(rng, 32))
	}
	emitCalldata, err := (&txintent.InitTrigger{
		Emitter:    eventLoggerB,
		Topics:     topics,
		OpaqueData: testutils.RandomData(rng, 10),
	}).EncodeInput()
	if err != nil {
		return fmt.Errorf("encode emitLog calldata: %w", err)
	}
	sent, err := env.userA.sendMessage(env.ctx, env.chainB.chainID, eventLoggerB, emitCalldata)
	if err != nil {
		return fmt.Errorf("send cross-chain message on Chain A: %w", err)
	}
	fmt.Fprintf(env.stderr, "    Message sent on Chain A (block %d), target %s on Chain B\n",
		bigs.Uint64Strict(sent.Receipt.BlockNumber), eventLoggerB)

	// Step 2: Chain B relays it invalidly. The relay executes anyway and emits
	// the bounced message.
	if _, err := waitForNextBlock(env.ctx, env.chainB); err != nil {
		return err
	}
	badRelay, err := env.userB.sendInvalidRelayMessage(env.ctx, sent)
	if err != nil {
		return fmt.Errorf("send invalid relay message: %w", err)
	}
	if badRelay.Receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("invalid relay tx reverted before inclusion")
	}
	badBlockNum, badBlockHash := badRelay.BlockNumber(), badRelay.BlockHash()
	fmt.Fprintf(env.stderr, "    Invalid relay landed on Chain B block %d (%s)\n", badBlockNum, badBlockHash)

	bounceIdx := firstLogFrom(badRelay.Receipt.Logs, eventLoggerB)
	if bounceIdx < 0 {
		return fmt.Errorf("invalid relay emitted no log from EventLogger %s; the relayed call did not reach its target", eventLoggerB)
	}
	relayOut, err := badRelay.Tx.Result.Eval(env.ctx)
	if err != nil {
		return fmt.Errorf("read invalid relay entries: %w", err)
	}
	if len(relayOut.Entries) <= bounceIdx {
		return fmt.Errorf("invalid relay entry %d missing, only have %d", bounceIdx, len(relayOut.Entries))
	}
	bounced := relayOut.Entries[bounceIdx]
	fmt.Fprintf(env.stderr, "    Bounced message emitted on Chain B by the invalid relay (log index %d)\n", bounceIdx)

	// Step 3: Chain A executes the bounced message, which depends on the invalid
	// Chain B block directly. Well-formed and valid at send time; it only
	// becomes invalid once that block goes.
	direct, err := env.execDependentOnA(bounced, "Direct", badBlockNum, badBlockHash, controlBlockNum)
	if err != nil {
		return err
	}

	// Step 3b: a plain log emitted on a later Chain B block. That block is
	// valid in itself and is reorged only because its ancestor is replaced, so
	// a dependency on it exercises invalidation through the lineage rather than
	// on the invalid block directly.
	descendantInit, err := env.userB.sendRandomInitMessage(env.ctx, rng, eventLoggerB, 2, 10)
	if err != nil {
		return fmt.Errorf("send descendant init message: %w", err)
	}
	descendantBlockNum := descendantInit.BlockNumber()
	if descendantBlockNum <= badBlockNum {
		return fmt.Errorf("descendant init landed on Chain B block %d, which is not after the invalid block %d",
			descendantBlockNum, badBlockNum)
	}
	descendantOut, err := descendantInit.Tx.Result.Eval(env.ctx)
	if err != nil {
		return fmt.Errorf("read descendant init entries: %w", err)
	}
	if len(descendantOut.Entries) == 0 {
		return fmt.Errorf("descendant init tx produced no interop entries")
	}
	fmt.Fprintf(env.stderr, "    Descendant init message emitted on Chain B block %d (after invalid block %d)\n",
		descendantBlockNum, badBlockNum)

	if _, err := waitForNextBlock(env.ctx, env.chainA); err != nil {
		return err
	}
	descendant, err := env.execDependentOnA(descendantOut.Entries[0], "Descendant", badBlockNum, badBlockHash, controlBlockNum)
	if err != nil {
		return err
	}

	// Wait for the invalid relay to be replaced even when neither dependent
	// message was accepted. This confirms the invalidation itself happened
	// before reporting that cascade coverage was not exercised.
	cascadeDeadline := time.Now().Add(env.reorgTimeout)
	if err := waitForReorgedOut(env.ctx, env.stderr, env.chainB, badBlockNum, badBlockHash, badRelay.Receipt.TxHash, time.Until(cascadeDeadline)); err != nil {
		return fmt.Errorf("invalid relay was not reorged out of Chain B: %w", err)
	}
	fmt.Fprintf(env.stderr, "    Invalid relay reorged out on Chain B block %d\n", badBlockNum)

	if direct == nil && descendant == nil {
		fmt.Fprintf(env.stderr, "    Chain A included neither dependent message, so no cascade was needed (prevention, not cure).\n")
		fmt.Fprintf(env.stderr, "    NOTE: transitive invalidation itself was NOT exercised by this run.\n")
		if err := assertControlIntact(env, controlBlockNum, controlBlockHash, controlReceipt.TxHash); err != nil {
			return err
		}
		if env.requireCascade {
			return fmt.Errorf("no dependent message was included, so transitive invalidation was never exercised (--%s)", requireCascadeFlagName)
		}
		return nil
	}

	// Step 4: the invalid block goes, and the cascade must follow.
	for _, arm := range []*dependentExec{direct, descendant} {
		if arm == nil {
			continue
		}
		if err := waitForReorgedOut(env.ctx, env.stderr, env.chainA, arm.blockNum, arm.blockHash, arm.txHash, time.Until(cascadeDeadline)); err != nil {
			return fmt.Errorf("%s dependent message was not reorged out of Chain A; invalidation did not propagate: %w", arm.name, err)
		}
		fmt.Fprintf(env.stderr, "    %s dependent message reorged out on Chain A block %d (invalidation propagated)\n", arm.name, arm.blockNum)
	}

	return assertControlIntact(env, controlBlockNum, controlBlockHash, controlReceipt.TxHash)
}

// dependentExec is an executing message on Chain A that depends, directly or
// through the block lineage, on the invalid Chain B block.
type dependentExec struct {
	name      string
	blockNum  uint64
	blockHash common.Hash
	txHash    common.Hash
}

// execDependentOnA sends an executing message for entry on Chain A. A nil
// result means the node refused it, which it may do while the source block is
// not yet cross-unsafe. The invalid block never will be, so refusal is
// prevention rather than cure and is not a failure, though it exercises none of
// the cascade logic.
//
// A refusal is only credited as prevention if Chain A is still answering
// afterwards. Otherwise the send failed for an unrelated reason - an unreachable
// RPC, a nonce or gas problem - and reporting that as prevention would let a
// broken run look like a pass.
func (env *smokeEnv) execDependentOnA(entry messages.Message, name string, badBlockNum uint64, badBlockHash common.Hash, controlBlockNum uint64) (*dependentExec, error) {
	if err := assertBlockUnchanged(env.ctx, env.chainB, badBlockNum, badBlockHash); err != nil {
		return nil, err
	}
	receipt, err := env.userA.execEntry(env.ctx, entry)
	if err != nil {
		if _, headErr := env.chainA.ethClient.BlockRefByLabel(env.ctx, eth.Unsafe); headErr != nil {
			return nil, fmt.Errorf("%s dependent message failed and Chain A is not answering, so the failure cannot be read as refusal: %w",
				name, errors.Join(err, headErr))
		}
		if err := assertBlockUnchanged(env.ctx, env.chainB, badBlockNum, badBlockHash); err != nil {
			return nil, err
		}
		fmt.Fprintf(env.stderr, "    Chain A refused the %s dependent message: %v\n", name, err)
		return nil, nil
	}
	blockNum := bigs.Uint64Strict(receipt.BlockNumber)
	if blockNum == controlBlockNum {
		return nil, fmt.Errorf("%s dependent message shares Chain A block %d with the control message; cannot attribute a reorg",
			name, blockNum)
	}
	fmt.Fprintf(env.stderr, "    %s dependent message landed on Chain A block %d (%s)\n", name, blockNum, receipt.BlockHash)
	if err := assertBlockUnchanged(env.ctx, env.chainB, badBlockNum, badBlockHash); err != nil {
		return nil, err
	}
	return &dependentExec{
		name:      name,
		blockNum:  blockNum,
		blockHash: receipt.BlockHash,
		txHash:    receipt.TxHash,
	}, nil
}

// assertBlockUnchanged reports an inconclusive run if the block has already
// been replaced. The chained test can only observe transitive invalidation if
// the invalid block is still canonical while the dependent message is sent.
func assertBlockUnchanged(ctx context.Context, chain *remoteChain, blockNum uint64, want common.Hash) error {
	current, err := chain.ethClient.BlockRefByNumber(ctx, blockNum)
	if err != nil {
		return fmt.Errorf("fetch %s block %d: %w", chain.name, blockNum, err)
	}
	if current.Hash != want {
		return fmt.Errorf("INCONCLUSIVE: %s block %d was already replaced (%s -> %s) before the dependent message was sent; rerun",
			chain.name, blockNum, want, current.Hash)
	}
	return nil
}

// assertControlIntact checks that a message which does not depend on the
// invalid block survived. Without this, a node that discards a swathe of recent
// Chain A blocks would look correct.
func assertControlIntact(env *smokeEnv, blockNum uint64, blockHash common.Hash, txHash common.Hash) error {
	current, err := env.chainA.ethClient.BlockRefByNumber(env.ctx, blockNum)
	if err != nil {
		return fmt.Errorf("fetch control block %d on Chain A: %w", blockNum, err)
	}
	if current.Hash != blockHash {
		return fmt.Errorf("control message's Chain A block %d was replaced (%s -> %s); invalidation over-reached",
			blockNum, blockHash, current.Hash)
	}
	if err := assertTxInBlock(env.ctx, env.chainA, blockNum, txHash); err != nil {
		return fmt.Errorf("control message did not survive: %w", err)
	}
	fmt.Fprintf(env.stderr, "    Control message still present on Chain A block %d\n", blockNum)
	return nil
}

func validateInvalidMessageOptions(blocks, txPerBlock uint) error {
	if blocks == 0 || txPerBlock == 0 {
		return fmt.Errorf("blocks and tx-per-block must be greater than zero")
	}
	return nil
}

func waitForBalance(ctx context.Context, chain *remoteChain, addr common.Address, want eth.ETH) error {
	deadline := time.Now().Add(smokeWaitTimeout)
	for {
		balance, err := chain.ethClient.BalanceAt(ctx, addr, nil)
		if err == nil && balance.Cmp(want.ToBig()) == 0 {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("timed out waiting for balance on %s: %w", chain.name, err)
			}
			return fmt.Errorf("timed out waiting for %s balance on %s; got %s", addr, chain.name, eth.WeiBig(balance))
		}
		time.Sleep(time.Second)
	}
}

func waitForNextBlock(ctx context.Context, chain *remoteChain) (eth.BlockRef, error) {
	head, err := chain.ethClient.BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return eth.BlockRef{}, fmt.Errorf("fetch latest block on %s: %w", chain.name, err)
	}
	if err := waitForHeadAtLeast(ctx, chain, head.Number+1); err != nil {
		return eth.BlockRef{}, err
	}
	return chain.ethClient.BlockRefByLabel(ctx, eth.Unsafe)
}

func waitForHeadAtLeast(ctx context.Context, chain *remoteChain, target uint64) error {
	deadline := time.Now().Add(smokeWaitTimeout)
	for {
		head, err := chain.ethClient.BlockRefByLabel(ctx, eth.Unsafe)
		if err == nil && head.Number >= target {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("timed out waiting for %s head >= %d: %w", chain.name, target, err)
			}
			return fmt.Errorf("timed out waiting for %s head >= %d; current head is %d", chain.name, target, head.Number)
		}
		time.Sleep(time.Second)
	}
}

func waitForReorgedOut(ctx context.Context, stderr io.Writer, chain *remoteChain, blockNum uint64, oldHash, txHash common.Hash, timeout time.Duration) error {
	start := time.Now()
	deadline := start.Add(timeout)
	for attempt := 0; ; attempt++ {
		elapsed := time.Since(start).Round(time.Second)
		currentBlock, err := chain.ethClient.BlockRefByNumber(ctx, blockNum)
		if err == nil {
			if currentBlock.Hash != oldHash {
				fmt.Fprintf(stderr, "    [%s] Reorg detected at block %d after %s: %s -> %s\n", chain.name, blockNum, elapsed, oldHash, currentBlock.Hash)
				break
			}
			if attempt == 0 || attempt%10 == 0 {
				fmt.Fprintf(stderr, "    [%s] Waiting for reorg at block %d (%s/%s): still %s\n", chain.name, blockNum, elapsed, timeout, currentBlock.Hash)
			}
		} else if errors.Is(eth.MaybeAsNotFoundErr(err), ethereum.NotFound) {
			if attempt == 0 || attempt%10 == 0 {
				fmt.Fprintf(stderr, "    [%s] Waiting for reorg at block %d (%s/%s): block temporarily missing\n", chain.name, blockNum, elapsed, timeout)
			}
		} else if attempt == 0 || attempt%10 == 0 {
			fmt.Fprintf(stderr, "    [%s] Waiting for reorg at block %d (%s/%s): lookup error: %v\n", chain.name, blockNum, elapsed, timeout, err)
		}

		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for block %d (%s) to be reorged out", blockNum, oldHash)
		}
		time.Sleep(time.Second)
	}

	if err := waitForHeadProgress(ctx, stderr, chain, blockNum+1, timeout); err != nil {
		return err
	}
	return assertTxNotInBlock(ctx, chain, blockNum, txHash)
}

// errChainStalled marks a block that was correctly replaced on a chain that
// then stopped producing blocks. The invalidation itself was right and the
// chain's liveness is what broke; reporting both as "not reorged" hides which
// component is at fault.
var errChainStalled = errors.New("chain stopped producing blocks after the replacement")

// waitForHeadProgress waits for the chain's head to reach target, reporting what
// it is waiting for as it goes. It is used after a reorg has already been
// observed, where a head that never advances means the chain wedged on the
// replacement block rather than that the reorg failed.
func waitForHeadProgress(ctx context.Context, stderr io.Writer, chain *remoteChain, target uint64, timeout time.Duration) error {
	start := time.Now()
	deadline := start.Add(timeout)
	var head uint64
	var haveHead bool
	for attempt := 0; ; attempt++ {
		elapsed := time.Since(start).Round(time.Second)
		ref, err := chain.ethClient.BlockRefByLabel(ctx, eth.Unsafe)
		if err == nil {
			if ref.Number >= target {
				fmt.Fprintf(stderr, "    [%s] Chain progressing again: head %d after %s\n", chain.name, ref.Number, elapsed)
				return nil
			}
			// Log on the first attempt, every ten seconds, and whenever the head
			// moves, so a chain inching forward looks different from a wedged one.
			if attempt == 0 || attempt%10 == 0 || !haveHead || ref.Number != head {
				fmt.Fprintf(stderr, "    [%s] Waiting for chain to resume building past the replaced block (%s/%s): head still %d, need %d\n",
					chain.name, elapsed, timeout, ref.Number, target)
			}
			head, haveHead = ref.Number, true
		} else if attempt == 0 || attempt%10 == 0 {
			fmt.Fprintf(stderr, "    [%s] Waiting for chain to resume building (%s/%s): head lookup error: %v\n",
				chain.name, elapsed, timeout, err)
		}

		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			if !haveHead {
				return fmt.Errorf("%w: %s head could not be read for %s after block %d was replaced",
					errChainStalled, chain.name, timeout, target-1)
			}
			return fmt.Errorf("%w: %s head stuck at %d for %s after block %d was replaced",
				errChainStalled, chain.name, head, timeout, target-1)
		}
		time.Sleep(time.Second)
	}
}

func assertTxInBlock(ctx context.Context, chain *remoteChain, blockNum uint64, txHash common.Hash) error {
	_, txs, err := chain.ethClient.InfoAndTxsByNumber(ctx, blockNum)
	if err != nil {
		return fmt.Errorf("fetch block %d txs on %s: %w", blockNum, chain.name, err)
	}
	for _, tx := range txs {
		if tx.Hash() == txHash {
			return nil
		}
	}
	return fmt.Errorf("tx %s not found in block %d on %s", txHash, blockNum, chain.name)
}

func assertTxNotInBlock(ctx context.Context, chain *remoteChain, blockNum uint64, txHash common.Hash) error {
	_, txs, err := chain.ethClient.InfoAndTxsByNumber(ctx, blockNum)
	if err != nil {
		return fmt.Errorf("fetch block %d txs on %s: %w", blockNum, chain.name, err)
	}
	for _, tx := range txs {
		if tx.Hash() == txHash {
			return fmt.Errorf("tx %s still present in block %d on %s", txHash, blockNum, chain.name)
		}
	}
	return nil
}

func randomAddress() common.Address {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	return crypto.PubkeyToAddress(privKey.PublicKey)
}
