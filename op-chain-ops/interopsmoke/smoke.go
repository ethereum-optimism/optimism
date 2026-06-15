// Package interopsmoke implements interop smoke tests that run against the
// RPCs of two live interoperable L2 chains: chain identity, ETH transfers,
// cross-chain ETH bridging, and valid/invalid executing-message checks.
//
// It is exposed both as the standalone op-chain-ops/cmd/interop-smoke tool and
// as the `smoke-interop` subcommand of op-up.
package interopsmoke

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
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
	"github.com/lmittmann/w3"
)

const smokeWaitTimeout = 60 * time.Second

const (
	l2AURLFlagName           = "l2a-rpc"
	l2BURLFlagName           = "l2b-rpc"
	privateKeyFlagName       = "private-key"
	configFlagName           = "config"
	filterAdminRPCsFlagName  = "filter.admin-rpc"
	filterJWTSecretsFlagName = "filter.jwt-secret"
	testFailsafeFlagName     = "test-failsafe"

	failsafePropagationWait = 6 * time.Second
	failsafeRelayAttempts   = 3
	failsafeSubmitGasLimit  = uint64(300_000)
)

// rpcErrOutOfScope is the JSON-RPC error code for ErrOutOfScope from op-alloy (-321100).
// Returned by op-reth when the relay tx references a block not yet in scope on the target chain.
const rpcErrOutOfScope = -321100

// rpcErrCode extracts the JSON-RPC error code from err, if present.
// Uses an explicit unwrap loop with direct type assertion because errors.As
// does not reliably match non-error interface targets across all Go versions.
func rpcErrCode(err error) (int, bool) {
	for err != nil {
		if e, ok := err.(interface{ ErrorCode() int }); ok {
			return e.ErrorCode(), true
		}
		err = errors.Unwrap(err)
	}
	return 0, false
}

var sendETHFn = w3.MustNewFunc("sendETH(address,uint256)", "bytes32")

func smokeFlags(envPrefix string) []cli.Flag {
	return cliapp.ProtectFlags([]cli.Flag{
		&cli.StringFlag{
			Name:    l2AURLFlagName,
			Usage:   "RPC URL for chain A.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_L2A_RPC"),
			Value:   "http://localhost:8545",
		},
		&cli.StringFlag{
			Name:    l2BURLFlagName,
			Usage:   "RPC URL for chain B.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_L2B_RPC"),
			Value:   "http://localhost:8546",
		},
		&cli.StringFlag{
			Name:    privateKeyFlagName,
			Usage:   "Private key to fund smoke-test transactions. If empty, uses the default dev key.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_PRIVATE_KEY"),
		},
	})
}

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

type smokeEnv struct {
	ctx    context.Context
	stderr io.Writer
	chainA *remoteChain
	chainB *remoteChain
	userA  *remoteUser
	userB  *remoteUser
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

func newSmokeEnv(ctx context.Context, stderr io.Writer, l2AURL, l2BURL, privateKey string) (*smokeEnv, func(), error) {
	logger := newLogger(ctx, stderr)

	chainA, err := connectRemoteChain(ctx, logger, "L2A", l2AURL)
	if err != nil {
		return nil, nil, err
	}
	chainB, err := connectRemoteChain(ctx, logger, "L2B", l2BURL)
	if err != nil {
		chainA.ethClient.Close()
		return nil, nil, err
	}

	privKey, address, err := resolveSmokeKey(privateKey)
	if err != nil {
		chainA.ethClient.Close()
		chainB.ethClient.Close()
		return nil, nil, err
	}

	env := &smokeEnv{
		ctx:    ctx,
		stderr: stderr,
		chainA: chainA,
		chainB: chainB,
		userA:  &remoteUser{chain: chainA, privKey: privKey, address: address},
		userB:  &remoteUser{chain: chainB, privKey: privKey, address: address},
	}
	cleanup := func() {
		chainA.ethClient.Close()
		chainB.ethClient.Close()
	}
	return env, cleanup, nil
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
	l2AURL := cliCtx.String(l2AURLFlagName)
	l2BURL := cliCtx.String(l2BURLFlagName)
	privateKey := cliCtx.String(privateKeyFlagName)

	fmt.Fprintf(stderr, "\nSmoke: %s\n\n", name)

	env, cleanup, err := newSmokeEnv(ctx, stderr, l2AURL, l2BURL, privateKey)
	if err != nil {
		return err
	}
	defer cleanup()

	fmt.Fprintf(stderr, "Chain A RPC: %s (chain ID %s)\n", env.chainA.url, env.chainA.chainID)
	fmt.Fprintf(stderr, "Chain B RPC: %s (chain ID %s)\n", env.chainB.url, env.chainB.chainID)
	fmt.Fprintf(stderr, "Smoke Sender Address: %s\n\n", env.userA.address)

	if err := fn(env); err != nil {
		fmt.Fprintf(stderr, "\nFAIL: %s (%v)\n", name, err)
		return err
	}
	fmt.Fprintf(stderr, "\nPASS: %s\n", name)
	return nil
}

// Command returns the `smoke-interop` command tree for embedding in a host
// CLI such as op-up. envPrefix scopes the flag environment variables
// (e.g. "OP_UP" -> OP_UP_SMOKE_L2A_RPC).
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
	bridgeFlags := cliapp.ProtectFlags([]cli.Flag{
		&cli.StringFlag{
			Name:  configFlagName,
			Usage: "Path to TOML config file. CLI flags and env vars override it.",
		},
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    l2AURLFlagName,
			Usage:   "RPC URL for chain A.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_L2A_RPC"),
			Value:   "http://localhost:8545",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    l2BURLFlagName,
			Usage:   "RPC URL for chain B.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_L2B_RPC"),
			Value:   "http://localhost:8546",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    privateKeyFlagName,
			Usage:   "Private key (hex, no 0x). If empty, uses the default dev key.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_PRIVATE_KEY"),
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:    "loops",
			Usage:   "number of round-trip loops (0 = one one-way A→B trip; N = N*2 trips alternating A→B and B→A)",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_BRIDGE_LOOPS"),
			Value:   10,
		}),
		altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
			Name:    filterAdminRPCsFlagName,
			Usage:   "op-interop-filter admin RPC URLs (repeat for multiple filters). Required with --test-failsafe.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_FILTER_ADMIN_RPC"),
		}),
		altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
			Name:    filterJWTSecretsFlagName,
			Usage:   "32-byte hex JWT secrets (one per --filter.admin-rpc entry, in matching order). 0x prefix optional.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_FILTER_JWT_SECRET"),
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:  testFailsafeFlagName,
			Usage: "Run failsafe lifecycle test: bridge succeeds, enable failsafe, verify relay blocked, disable failsafe, bridge succeeds.",
		}),
	})

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
			Usage: "verify both chains have different chain IDs",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "Chain Identity", smokeIdentity)
			},
		},
		{
			Name:  "transfer",
			Usage: "send ETH transfers on both chains",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "ETH Transfers", smokeTransfer)
			},
		},
		{
			Name:   "bridge",
			Usage:  "bridge ETH from chain A to chain B via SuperchainETHBridge",
			Flags:  bridgeFlags,
			Before: altsrc.InitInputSourceWithContext(bridgeFlags, altsrc.NewTomlSourceFromFlagFunc(configFlagName)),
			Action: func(cliCtx *cli.Context) error {
				srcURL := cliCtx.String(l2AURLFlagName)
				dstURL := cliCtx.String(l2BURLFlagName)
				privateKey := cliCtx.String(privateKeyFlagName)
				stderr := cliCtx.App.ErrWriter
				ctx := cliCtx.Context

				if cliCtx.Bool(testFailsafeFlagName) {
					adminRPCs := cliCtx.StringSlice(filterAdminRPCsFlagName)
					jwtSecrets := cliCtx.StringSlice(filterJWTSecretsFlagName)
					if len(adminRPCs) == 0 {
						return fmt.Errorf("--test-failsafe requires at least one --filter.admin-rpc")
					}
					envAB, cleanupAB, err := newSmokeEnv(ctx, stderr, srcURL, dstURL, privateKey)
					if err != nil {
						return err
					}
					defer cleanupAB()
					envBA, cleanupBA, err := newSmokeEnv(ctx, stderr, dstURL, srcURL, privateKey)
					if err != nil {
						return err
					}
					defer cleanupBA()
					logger := newLogger(ctx, stderr)
					filterClients, filterCleanup, err := dialFilterAdmins(ctx, logger, adminRPCs, jwtSecrets)
					if err != nil {
						return err
					}
					defer filterCleanup()
					return smokeFailsafe(envAB, envBA, filterClients)
				}

				loops := cliCtx.Int("loops")
				trips := loops * 2
				if loops == 0 {
					trips = 1
				}
				rng := rand.New(rand.NewSource(time.Now().UnixNano()))

				for trip := 1; trip <= trips; trip++ {
					l2AURL, l2BURL := srcURL, dstURL
					if trip%2 == 0 {
						l2AURL, l2BURL = dstURL, srcURL
					}
					amount := eth.GWei(uint64(rng.Int63n(9_999_999)) + 1)

					env, cleanup, err := newSmokeEnv(ctx, stderr, l2AURL, l2BURL, privateKey)
					if err != nil {
						return err
					}
					fmt.Fprintf(stderr, "\nBridge trip %d/%d: chain %s → chain %s  amount: %s ETH\n",
						trip, trips, env.chainA.chainID, env.chainB.chainID, amount.EtherString())

					err = smokeBridge(env, amount)
					cleanup()
					if err != nil {
						fmt.Fprintf(stderr, "FAIL: bridge trip %d/%d: %v\n", trip, trips, err)
						return err
					}
					fmt.Fprintf(stderr, "PASS: bridge trip %d/%d\n", trip, trips)
				}
				return nil
			},
		},
		{
			Name:  "valid-message",
			Usage: "send a valid executing message and verify it stays in-chain",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "Valid Exec Message", smokeValidMessage)
			},
		},
		{
			Name:  "invalid-message",
			Usage: "send an invalid executing message and verify it is reorged out",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				return withSmokeEnv(cliCtx, "Invalid Exec Message (reorg)", smokeInvalidMessage)
			},
		},
	}
}

func smokeAll(env *smokeEnv) error {
	tests := []struct {
		name string
		fn   func(env *smokeEnv) error
	}{
		{"Chain Identity", smokeIdentity},
		{"ETH Transfers", smokeTransfer},
		{"Cross-Chain ETH Bridge", func(env *smokeEnv) error { return smokeBridge(env, eth.OneHundredthEther) }},
		{"Valid Exec Message", smokeValidMessage},
		{"Invalid Exec Message (reorg)", smokeInvalidMessage},
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
	if env.chainA.chainID == env.chainB.chainID {
		return fmt.Errorf("chains have the same ID: %s", env.chainA.chainID)
	}
	fmt.Fprintf(env.stderr, "    Chain A: %s, Chain B: %s\n", env.chainA.chainID, env.chainB.chainID)
	return nil
}

func smokeTransfer(env *smokeEnv) error {
	recipientA := randomAddress()
	if _, err := env.userA.transfer(env.ctx, recipientA, eth.OneHundredthEther); err != nil {
		return fmt.Errorf("chain A transfer failed: %w", err)
	}
	if err := waitForBalance(env.ctx, env.chainA, recipientA, eth.OneHundredthEther); err != nil {
		return err
	}
	fmt.Fprintf(env.stderr, "    Chain A transfer: OK\n")

	recipientB := randomAddress()
	if _, err := env.userB.transfer(env.ctx, recipientB, eth.OneHundredthEther); err != nil {
		return fmt.Errorf("chain B transfer failed: %w", err)
	}
	if err := waitForBalance(env.ctx, env.chainB, recipientB, eth.OneHundredthEther); err != nil {
		return err
	}
	fmt.Fprintf(env.stderr, "    Chain B transfer: OK\n")
	return nil
}

func smokeBridge(env *smokeEnv, amount eth.ETH) error {
	recipient := randomAddress()

	senderBefore, err := env.chainA.ethClient.BalanceAt(env.ctx, env.userA.address, nil)
	if err != nil {
		return fmt.Errorf("fetch sender balance before bridge: %w", err)
	}
	recipientBefore, err := env.chainB.ethClient.BalanceAt(env.ctx, recipient, nil)
	if err != nil {
		return fmt.Errorf("fetch recipient balance before bridge: %w", err)
	}
	fmt.Fprintf(env.stderr, "    Sender    chain %s balance before: %s ETH\n", env.chainA.chainID, eth.WeiBig(senderBefore).EtherString())
	fmt.Fprintf(env.stderr, "    Recipient chain %s balance before: %s ETH\n", env.chainB.chainID, eth.WeiBig(recipientBefore).EtherString())

	sendTx := txintent.NewIntent[*sendETHTrigger, *txintent.InteropOutput](
		env.userA.plan(),
		txplan.WithValue(amount),
	)
	sendTx.Content.Set(&sendETHTrigger{
		Recipient:   recipient,
		Destination: env.chainB.chainID,
	})

	sendReceipt, err := sendTx.PlannedTx.Included.Eval(env.ctx)
	if err != nil {
		return fmt.Errorf("sendETH failed: %w", err)
	}
	if sendReceipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("sendETH tx reverted")
	}
	fmt.Fprintf(env.stderr, "    sendETH tx included: %s\n", sendReceipt.TxHash)

	relayReceipt, err := waitForRelaySuccess(env, sendTx)
	if err != nil {
		return err
	}
	fmt.Fprintf(env.stderr, "    relayETH tx included: %s\n", relayReceipt.TxHash)

	if err := waitForBalance(env.ctx, env.chainB, recipient, amount); err != nil {
		return err
	}

	senderAfter, err := env.chainA.ethClient.BalanceAt(env.ctx, env.userA.address, nil)
	if err != nil {
		return fmt.Errorf("fetch sender balance after bridge: %w", err)
	}
	fmt.Fprintf(env.stderr, "    Sender    chain %s balance after:  %s ETH\n", env.chainA.chainID, eth.WeiBig(senderAfter).EtherString())
	fmt.Fprintf(env.stderr, "    Recipient chain %s balance after:  %s ETH (received %s ETH)\n", env.chainB.chainID, amount.EtherString(), amount.EtherString())
	return nil
}

func smokeValidMessage(env *smokeEnv) error {
	rng := rand.New(rand.NewSource(42))

	eventLogger, err := env.userA.deployEventLogger(env.ctx)
	if err != nil {
		return fmt.Errorf("deploy EventLogger: %w", err)
	}

	initMsg, err := env.userA.sendRandomInitMessage(env.ctx, rng, eventLogger, 2, 10)
	if err != nil {
		return fmt.Errorf("send init message: %w", err)
	}
	fmt.Fprintf(env.stderr, "    Init message sent on Chain A (block %d)\n", initMsg.BlockNumber())

	if _, err := waitForNextBlock(env.ctx, env.chainB); err != nil {
		return err
	}
	execMsg, err := env.userB.sendExecMessage(env.ctx, initMsg)
	if err != nil {
		return fmt.Errorf("send exec message: %w", err)
	}
	if execMsg.Receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("exec tx reverted")
	}

	execBlockNum := execMsg.BlockNumber()
	execBlockHash := execMsg.BlockHash()
	fmt.Fprintf(env.stderr, "    Exec message sent on Chain B (block %d)\n", execBlockNum)

	if err := waitForHeadAtLeast(env.ctx, env.chainB, execBlockNum+2); err != nil {
		return err
	}

	currentBlock, err := env.chainB.ethClient.BlockRefByNumber(env.ctx, execBlockNum)
	if err != nil {
		return fmt.Errorf("fetch block %d: %w", execBlockNum, err)
	}
	if currentBlock.Hash != execBlockHash {
		return fmt.Errorf("block was replaced: expected %s, got %s", execBlockHash, currentBlock.Hash)
	}
	if err := assertTxInBlock(env.ctx, env.chainB, execBlockNum, execMsg.Receipt.TxHash); err != nil {
		return err
	}
	fmt.Fprintf(env.stderr, "    Block remained canonical after head advanced past it\n")
	return nil
}

func smokeInvalidMessage(env *smokeEnv) error {
	rng := rand.New(rand.NewSource(99))

	eventLogger, err := env.userA.deployEventLogger(env.ctx)
	if err != nil {
		return fmt.Errorf("deploy EventLogger: %w", err)
	}

	initMsg, err := env.userA.sendRandomInitMessage(env.ctx, rng, eventLogger, 2, 10)
	if err != nil {
		return fmt.Errorf("send init message: %w", err)
	}
	if _, err := waitForNextBlock(env.ctx, env.chainB); err != nil {
		return err
	}

	invalidExec, err := env.userB.sendInvalidExecMessage(env.ctx, initMsg)
	if err != nil {
		return fmt.Errorf("send invalid exec message: %w", err)
	}
	if invalidExec.Receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("invalid exec tx reverted before inclusion")
	}

	invalidBlockNum := invalidExec.BlockNumber()
	invalidBlockHash := invalidExec.BlockHash()
	fmt.Fprintf(env.stderr, "    Invalid exec tx included: %s\n", invalidExec.Receipt.TxHash)
	fmt.Fprintf(env.stderr, "    Invalid exec message landed in block %d (%s)\n", invalidBlockNum, invalidBlockHash)

	if err := waitForReorgedOut(env.ctx, env.stderr, env.chainB, invalidBlockNum, invalidBlockHash, invalidExec.Receipt.TxHash); err != nil {
		return err
	}
	fmt.Fprintf(env.stderr, "    Invalid tx was reorged out after block %d was replaced\n", invalidBlockNum)
	return nil
}

func waitForRelaySuccess(env *smokeEnv, sendTx *txintent.IntentTx[*sendETHTrigger, *txintent.InteropOutput]) (*types.Receipt, error) {
	deadline := time.Now().Add(smokeWaitTimeout)
	for attempt := 0; ; attempt++ {
		relayTx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](env.userB.plan())
		relayTx.Content.DependOn(&sendTx.Result)
		relayTx.Content.Fn(txintent.RelayIndexed(
			predeploys.L2toL2CrossDomainMessengerAddr,
			&sendTx.Result,
			&sendTx.PlannedTx.Included,
			1,
		))

		relayReceipt, err := relayTx.PlannedTx.Included.Eval(env.ctx)
		if err == nil && relayReceipt.Status == types.ReceiptStatusSuccessful {
			return relayReceipt, nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return nil, fmt.Errorf("timed out waiting for relay: init tx not yet cross-unsafe on chain B after %s: %w", smokeWaitTimeout, err)
			}
			return nil, fmt.Errorf("timed out waiting for relay: relay tx kept reverting on chain B after %s", smokeWaitTimeout)
		}
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				// Tx submitted but never included: still pending in chain B's pool,
				// waiting for the init tx on chain A to become cross-unsafe on chain B.
				fmt.Fprintf(env.stderr, "    Relay attempt %d: tx pending in chain B pool (init tx not yet cross-unsafe on chain B)\n", attempt+1)
			} else if code, ok := rpcErrCode(err); ok && code == rpcErrOutOfScope {
				// op-reth hard-rejected the relay: init tx block not yet in scope on chain B.
				fmt.Fprintf(env.stderr, "    Relay attempt %d: out of scope (RPC %d): init tx block not yet in scope on chain B\n", attempt+1, code)
			} else {
				fmt.Fprintf(env.stderr, "    Relay attempt %d: chain B rejected relay tx: %v\n", attempt+1, err)
			}
		} else {
			fmt.Fprintf(env.stderr, "    Relay attempt %d: relay tx reverted (tx %s)\n", attempt+1, relayReceipt.TxHash)
		}
		time.Sleep(time.Second)
	}
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

func waitForReorgedOut(ctx context.Context, stderr io.Writer, chain *remoteChain, blockNum uint64, oldHash, txHash common.Hash) error {
	deadline := time.Now().Add(smokeWaitTimeout)
	replaced := false
	for attempt := 0; ; attempt++ {
		currentBlock, err := chain.ethClient.BlockRefByNumber(ctx, blockNum)
		if err == nil {
			if currentBlock.Hash != oldHash {
				fmt.Fprintf(stderr, "    Reorg detected at block %d: %s -> %s\n", blockNum, oldHash, currentBlock.Hash)
				replaced = true
				break
			}
			if attempt == 0 || attempt%10 == 0 {
				fmt.Fprintf(stderr, "    Waiting for reorg at block %d: still %s\n", blockNum, currentBlock.Hash)
			}
		} else if errors.Is(eth.MaybeAsNotFoundErr(err), ethereum.NotFound) {
			if attempt == 0 || attempt%10 == 0 {
				fmt.Fprintf(stderr, "    Waiting for reorg at block %d: block temporarily missing\n", blockNum)
			}
		} else if attempt == 0 || attempt%10 == 0 {
			fmt.Fprintf(stderr, "    Waiting for reorg at block %d: lookup error: %v\n", blockNum, err)
		}

		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for block %d (%s) to be reorged out", blockNum, oldHash)
		}
		time.Sleep(time.Second)
	}

	if !replaced {
		return fmt.Errorf("invalid reorg state")
	}
	if err := waitForHeadAtLeast(ctx, chain, blockNum+1); err != nil {
		return err
	}
	return assertTxNotInBlock(ctx, chain, blockNum, txHash)
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

// dialFilterAdmins creates JWT-authenticated RPC clients for each filter admin endpoint.
func dialFilterAdmins(ctx context.Context, logger log.Logger, adminRPCs, jwtSecrets []string) ([]opclient.RPC, func(), error) {
	if len(adminRPCs) != len(jwtSecrets) {
		return nil, nil, fmt.Errorf("--filter.admin-rpc and --filter.jwt-secret must have equal counts (got %d and %d)", len(adminRPCs), len(jwtSecrets))
	}
	clients := make([]opclient.RPC, 0, len(adminRPCs))
	for i, url := range adminRPCs {
		raw, err := hex.DecodeString(strings.TrimPrefix(jwtSecrets[i], "0x"))
		if err != nil || len(raw) != 32 {
			for _, c := range clients {
				c.Close()
			}
			return nil, nil, fmt.Errorf("filter jwt secret %d: must be a 32-byte hex string", i)
		}
		cl, err := opclient.NewRPC(ctx, logger, url,
			opclient.WithGethRPCOptions(gethrpc.WithHTTPAuth(gn.NewJWTAuth([32]byte(raw)))))
		if err != nil {
			for _, c := range clients {
				c.Close()
			}
			return nil, nil, fmt.Errorf("dial filter admin %d (%s): %w", i, url, err)
		}
		clients = append(clients, cl)
	}
	return clients, func() {
		for _, c := range clients {
			c.Close()
		}
	}, nil
}

// setFailsafeAll toggles failsafe on all filters and verifies each updated.
func setFailsafeAll(ctx context.Context, filters []opclient.RPC, enabled bool) error {
	for i, f := range filters {
		if err := f.CallContext(ctx, nil, "admin_setFailsafeEnabled", enabled); err != nil {
			return fmt.Errorf("filter %d: admin_setFailsafeEnabled(%v): %w", i, enabled, err)
		}
		var got bool
		if err := f.CallContext(ctx, &got, "admin_getFailsafeEnabled"); err != nil {
			return fmt.Errorf("filter %d: admin_getFailsafeEnabled: %w", i, err)
		}
		if got != enabled {
			return fmt.Errorf("filter %d: failsafe state mismatch: want %v got %v", i, enabled, got)
		}
	}
	return nil
}

// isFailsafeRejection reports whether err is a known interop-tx rejection from a failsafe.
func isFailsafeRejection(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "transaction filtered out") ||
		strings.Contains(msg, "interop failsafe is active") ||
		strings.Contains(msg, "failsafe is enabled") ||
		strings.Contains(msg, "failed to parse access entry")
}

// expectBridgeBlocked sends ETH from chain A and asserts the relay on chain B is rejected by the failsafe.
func expectBridgeBlocked(env *smokeEnv, amount eth.ETH) error {
	recipient := randomAddress()

	sendTx := txintent.NewIntent[*sendETHTrigger, *txintent.InteropOutput](
		env.userA.plan(),
		txplan.WithValue(amount),
	)
	sendTx.Content.Set(&sendETHTrigger{
		Recipient:   recipient,
		Destination: env.chainB.chainID,
	})
	sendReceipt, err := sendTx.PlannedTx.Included.Eval(env.ctx)
	if err != nil {
		return fmt.Errorf("sendETH: %w", err)
	}
	if sendReceipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("sendETH tx reverted")
	}
	fmt.Fprintf(env.stderr, "    sendETH tx included: %s\n", sendReceipt.TxHash)

	// Wait for propagation so rejection is attributable to failsafe, not timing.
	select {
	case <-time.After(failsafePropagationWait):
	case <-env.ctx.Done():
		return env.ctx.Err()
	}

	singlePlan := txplan.Combine(
		txplan.WithChainID(env.userB.chain.ethClient),
		txplan.WithPrivateKey(env.userB.privKey),
		txplan.WithPendingNonce(env.userB.chain.ethClient),
		txplan.WithAgainstLatestBlock(env.userB.chain.ethClient),
		txplan.WithGasLimit(failsafeSubmitGasLimit),
		txplan.WithTransactionSubmitter(env.userB.chain.ethClient),
	)
	for attempt := 1; attempt <= failsafeRelayAttempts; attempt++ {
		relayTx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](singlePlan)
		relayTx.Content.DependOn(&sendTx.Result)
		relayTx.Content.Fn(txintent.RelayIndexed(
			predeploys.L2toL2CrossDomainMessengerAddr,
			&sendTx.Result,
			&sendTx.PlannedTx.Included,
			1,
		))
		_, err = relayTx.PlannedTx.Submitted.Eval(env.ctx)
		if err == nil {
			return fmt.Errorf("relay attempt %d/%d accepted while failsafe enabled (chain %s → chain %s), expected rejection",
				attempt, failsafeRelayAttempts, env.chainA.chainID, env.chainB.chainID)
		}
		if !isFailsafeRejection(err) {
			return fmt.Errorf("relay attempt %d/%d (chain %s → chain %s) failed with unrecognized error:\n  %w",
				attempt, failsafeRelayAttempts, env.chainA.chainID, env.chainB.chainID, err)
		}
		fmt.Fprintf(env.stderr, "    relay attempt %d/%d (chain %s → chain %s) correctly rejected:\n      %v\n",
			attempt, failsafeRelayAttempts, env.chainA.chainID, env.chainB.chainID, err)
		if attempt < failsafeRelayAttempts {
			time.Sleep(time.Second)
		}
	}
	return nil
}

// smokeFailsafe runs the failsafe lifecycle in both directions:
// bridge A↔B succeeds, failsafe blocks both relays, bridge A↔B succeeds again.
func smokeFailsafe(envAB, envBA *smokeEnv, filters []opclient.RPC) error {
	stderr := envAB.stderr
	ctx := envAB.ctx
	amount := eth.OneHundredthEther

	// Ensure we start from a known-disabled state.
	if err := setFailsafeAll(ctx, filters, false); err != nil {
		return err
	}

	fmt.Fprintf(stderr, "\nStep 1: bridge A→B and B→A before failsafe (expect success)\n")
	if err := smokeBridge(envAB, amount); err != nil {
		return fmt.Errorf("step 1 A→B: %w", err)
	}
	if err := smokeBridge(envBA, amount); err != nil {
		return fmt.Errorf("step 1 B→A: %w", err)
	}

	fmt.Fprintf(stderr, "\nStep 2: enabling failsafe on %d filter(s)\n", len(filters))
	if err := setFailsafeAll(ctx, filters, true); err != nil {
		return err
	}

	fmt.Fprintf(stderr, "\nStep 3: relay A→B and B→A with failsafe enabled (expect rejection)\n")
	if err := expectBridgeBlocked(envAB, amount); err != nil {
		_ = setFailsafeAll(ctx, filters, false)
		return fmt.Errorf("step 3 A→B: %w", err)
	}
	if err := expectBridgeBlocked(envBA, amount); err != nil {
		_ = setFailsafeAll(ctx, filters, false)
		return fmt.Errorf("step 3 B→A: %w", err)
	}

	fmt.Fprintf(stderr, "\nStep 4: disabling failsafe on %d filter(s)\n", len(filters))
	if err := setFailsafeAll(ctx, filters, false); err != nil {
		return err
	}

	fmt.Fprintf(stderr, "\nStep 5: bridge A→B and B→A after failsafe disabled (expect success)\n")
	if err := smokeBridge(envAB, amount); err != nil {
		return fmt.Errorf("step 5 A→B: %w", err)
	}
	if err := smokeBridge(envBA, amount); err != nil {
		return fmt.Errorf("step 5 B→A: %w", err)
	}

	fmt.Fprintf(stderr, "\nPASS: failsafe lifecycle test\n")
	return nil
}

func randomAddress() common.Address {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	return crypto.PubkeyToAddress(privKey.PublicKey)
}
