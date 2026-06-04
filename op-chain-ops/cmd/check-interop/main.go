// Command check-interop verifies the interop failsafe kill-switch against a
// devnet with two interop-connected L2 chains.
//
// It runs the full failsafe lifecycle:
//  1. Enable failsafe  -> an executing (interop) message is rejected, not included.
//  2. Disable failsafe -> an executing message is accepted again.
//
// The original failsafe state is restored on exit.
//
// Toggling failsafe requires the op-interop-filter's JWT-protected admin RPC
// (its admin.rpc.addr endpoint) plus the JWT secret. The public/gateway filter
// RPC only exposes the read-only getter, so point --filter-admin-rpc at the admin
// endpoint (e.g. via `kubectl port-forward` to the filter's admin port).
//
// Usage:
//
//	check-interop \
//	  -l2a-rpc https://chain-a.example \
//	  -l2b-rpc https://chain-b.example \
//	  -private-key <hex> \
//	  -filter-admin-rpc http://127.0.0.1:8560 \
//	  -filter-jwt-secret ./filter-jwt.txt
package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	gethnode "github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/mattn/go-isatty"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	txintentbindings "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

const waitTimeout = 90 * time.Second

func main() {
	color := isatty.IsTerminal(os.Stderr.Fd())
	handler := log.NewTerminalHandler(os.Stderr, color)
	oplog.SetGlobalLogHandler(handler)
	logger := log.NewLogger(handler)

	var (
		l2aRPC          string
		l2bRPC          string
		privateKey      string
		filterRPC       string
		publicFilterRPC string
		jwtPath         string
	)
	flag.StringVar(&l2aRPC, "l2a-rpc", "http://localhost:8545", "RPC URL for chain A (interop source).")
	flag.StringVar(&l2bRPC, "l2b-rpc", "http://localhost:8546", "RPC URL for chain B (interop destination).")
	flag.StringVar(&privateKey, "private-key", "", "Hex private key funding the test transactions (required).")
	flag.StringVar(&filterRPC, "filter-admin-rpc", "", "JWT-protected admin RPC URL of op-interop-filter (required).")
	flag.StringVar(&publicFilterRPC, "filter-rpc", "", "Public RPC URL of op-interop-filter (interop_checkAccessList), used to assert the dedicated failsafe error code (required).")
	flag.StringVar(&jwtPath, "filter-jwt-secret", "", "Path to the JWT secret file for the filter admin RPC (required).")
	flag.Parse()

	if err := run(context.Background(), logger, l2aRPC, l2bRPC, privateKey, filterRPC, publicFilterRPC, jwtPath); err != nil {
		logger.Error("interop failsafe check FAILED", "err", err)
		os.Exit(1)
	}
	logger.Info("interop failsafe check PASSED")
}

func run(ctx context.Context, logger log.Logger, l2aRPC, l2bRPC, privateKey, filterRPC, publicFilterRPC, jwtPath string) error {
	if privateKey == "" {
		return fmt.Errorf("-private-key is required")
	}
	if filterRPC == "" {
		return fmt.Errorf("-filter-admin-rpc is required (the JWT admin endpoint; the public filter RPC cannot set failsafe)")
	}
	if publicFilterRPC == "" {
		return fmt.Errorf("-filter-rpc is required (the public filter RPC for interop_checkAccessList)")
	}
	if jwtPath == "" {
		return fmt.Errorf("-filter-jwt-secret is required")
	}

	priv, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	addr := crypto.PubkeyToAddress(priv.PublicKey)

	chainA, err := connectChain(ctx, logger, "L2A", l2aRPC)
	if err != nil {
		return err
	}
	defer chainA.ethClient.Close()
	chainB, err := connectChain(ctx, logger, "L2B", l2bRPC)
	if err != nil {
		return err
	}
	defer chainB.ethClient.Close()
	if chainA.chainID == chainB.chainID {
		return fmt.Errorf("chain A and B have the same chain ID %s", chainA.chainID)
	}

	fs, err := connectFilter(ctx, logger, filterRPC, jwtPath)
	if err != nil {
		return err
	}
	defer fs.close()

	filterQuery, err := connectFilterQuery(ctx, logger, publicFilterRPC)
	if err != nil {
		return err
	}
	defer filterQuery.Close()

	userA := &user{chain: chainA, priv: priv, addr: addr}
	userB := &user{chain: chainB, priv: priv, addr: addr}
	logger.Info("connected", "chainA", chainA.chainID, "chainB", chainB.chainID, "sender", addr)

	// Record the original failsafe state and restore it on exit so the devnet is
	// left as we found it.
	original, err := fs.get(ctx)
	if err != nil {
		return fmt.Errorf("read initial failsafe state: %w", err)
	}
	logger.Info("initial failsafe state", "enabled", original)
	defer func() {
		if err := fs.set(ctx, original); err != nil {
			logger.Warn("failed to restore failsafe", "want", original, "err", err)
			return
		}
		logger.Info("restored failsafe", "enabled", original)
	}()

	eventLogger, err := userA.deployEventLogger(ctx)
	if err != nil {
		return fmt.Errorf("deploy EventLogger: %w", err)
	}
	logger.Info("deployed EventLogger on chain A", "addr", eventLogger)

	rng := rand.New(rand.NewSource(7))

	// Start from a known-good baseline: failsafe disabled.
	if err := setFailsafeAndWait(ctx, logger, fs, chainB, false); err != nil {
		return err
	}

	// --- Failsafe ON: an executing message must be rejected ---
	if err := setFailsafeAndWait(ctx, logger, fs, chainB, true); err != nil {
		return err
	}
	logger.Info("failsafe ENABLED; expecting executing message to be rejected")

	initMsg, err := userA.sendInitMessage(ctx, rng, eventLogger)
	if err != nil {
		return fmt.Errorf("send init message (failsafe on): %w", err)
	}
	if _, err := waitForNextBlock(ctx, chainB); err != nil {
		return err
	}
	accessList, err := execAccessList(ctx, initMsg)
	if err != nil {
		return err
	}
	rejErr := userB.submitRawAccessListTx(ctx, accessList)
	if rejErr == nil {
		return fmt.Errorf("executing message was accepted while failsafe is enabled; expected rejection")
	}
	if !interopTxRejectedError(rejErr) {
		return fmt.Errorf("executing message failed but not with a filter/failsafe rejection: %w", rejErr)
	}
	logger.Info("executing message correctly rejected while failsafe on", "err", rejErr)

	// The end-to-end rejection above proves the stack enforces failsafe, but op-reth/proxyd
	// translate the filter's error before it reaches us. Query the filter directly to assert it
	// returns the dedicated failsafe error code (op-interop-filter #21205).
	if err := assertFilterFailsafeCode(ctx, logger, filterQuery, chainB, accessList); err != nil {
		return err
	}

	// --- Failsafe OFF: an executing message must succeed again ---
	if err := setFailsafeAndWait(ctx, logger, fs, chainB, false); err != nil {
		return err
	}
	logger.Info("failsafe DISABLED; waiting for interop to recover, then expecting executing message to succeed")

	// op-reth re-marks its interop/supervisor backend healthy asynchronously after
	// failsafe is disabled, so retry the full init+exec round until it lands.
	recoveryDeadline := time.Now().Add(3 * time.Minute)
	for attempt := 1; ; attempt++ {
		initMsg2, err := userA.sendInitMessage(ctx, rng, eventLogger)
		if err != nil {
			return fmt.Errorf("send init message (failsafe off): %w", err)
		}
		if _, err := waitForNextBlock(ctx, chainB); err != nil {
			return err
		}
		execMsg, err := userB.sendExecMessage(ctx, initMsg2)
		if err == nil && execMsg.receipt.Status == types.ReceiptStatusSuccessful {
			logger.Info("executing message included after failsafe disabled", "block", execMsg.blockNumber(), "attempt", attempt)
			return nil
		}
		if time.Now().After(recoveryDeadline) {
			if err != nil {
				return fmt.Errorf("executing message still failing after failsafe disabled: %w", err)
			}
			return fmt.Errorf("executing tx reverted after failsafe disabled")
		}
		logger.Info("interop not recovered yet, retrying", "attempt", attempt, "err", err)
		time.Sleep(3 * time.Second)
	}
}

type chain struct {
	name      string
	ethClient apis.EthClient
	chainID   eth.ChainID
}

type user struct {
	chain *chain
	priv  *ecdsa.PrivateKey
	addr  common.Address
}

type initMessage struct {
	tx      *txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput]
	receipt *types.Receipt
}

type execMessage struct {
	tx      *txintent.IntentTx[*txintent.ExecTrigger, *txintent.InteropOutput]
	receipt *types.Receipt
}

func (m *execMessage) blockNumber() uint64 {
	return bigs.Uint64Strict(m.receipt.BlockNumber)
}

func connectChain(ctx context.Context, logger log.Logger, name, url string) (*chain, error) {
	chainLogger := logger.New("chain", name, "rpc", url)
	rpcCl, err := opclient.NewRPC(ctx, chainLogger, url,
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
	return &chain{name: name, ethClient: ethCl, chainID: eth.ChainIDFromBig(chainIDBig)}, nil
}

func (u *user) plan() txplan.Option {
	return txplan.Combine(
		txplan.WithChainID(u.chain.ethClient),
		txplan.WithPrivateKey(u.priv),
		txplan.WithPendingNonce(u.chain.ethClient),
		txplan.WithAgainstLatestBlock(u.chain.ethClient),
		txplan.WithEstimator(u.chain.ethClient, true),
		txplan.WithRetrySubmission(u.chain.ethClient, 5, retry.Exponential()),
		txplan.WithRetryInclusion(u.chain.ethClient, 5, retry.Exponential()),
		txplan.WithBlockInclusionInfo(u.chain.ethClient),
	)
}

func (u *user) deployEventLogger(ctx context.Context) (common.Address, error) {
	tx := txplan.NewPlannedTx(u.plan(), txplan.WithData(common.FromHex(txintentbindings.EventloggerBin)))
	receipt, err := tx.Included.Eval(ctx)
	if err != nil {
		return common.Address{}, err
	}
	return receipt.ContractAddress, nil
}

func (u *user) sendInitMessage(ctx context.Context, rng *rand.Rand, eventLogger common.Address) (*initMessage, error) {
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
	return &initMessage{tx: tx, receipt: receipt}, nil
}

func (u *user) sendExecMessage(ctx context.Context, initMsg *initMessage) (*execMessage, error) {
	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](u.plan())
	tx.Content.DependOn(&initMsg.tx.Result)
	tx.Content.Fn(txintent.ExecuteIndexed(predeploys.CrossL2InboxAddr, &initMsg.tx.Result, 0))
	receipt, err := tx.PlannedTx.Included.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return &execMessage{tx: tx, receipt: receipt}, nil
}

// submitRawAccessListTx submits a self-transfer carrying the given CrossL2Inbox
// access list using a single-attempt submitter, returning the submission error
// (nil if the node accepted it). With failsafe enabled, the interop filter rejects
// the access list at ingress, so submission fails immediately.
func (u *user) submitRawAccessListTx(ctx context.Context, accessList types.AccessList) error {
	to := u.addr
	tx := txplan.NewPlannedTx(
		u.plan(),
		// Override the retrying submitter so a filter rejection surfaces
		// immediately instead of being retried until the context expires.
		txplan.WithTransactionSubmitter(u.chain.ethClient),
		txplan.WithTo(&to),
		txplan.WithValue(eth.GWei(1)),
		txplan.WithAccessList(accessList),
		txplan.WithGasLimit(100_000),
	)
	_, err := tx.Submitted.Eval(ctx)
	return err
}

// execAccessList builds the CrossL2Inbox access list for the first interop entry
// emitted by an init message -- the access list an executing message carries.
func execAccessList(ctx context.Context, initMsg *initMessage) (types.AccessList, error) {
	result, err := initMsg.tx.Result.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("eval init message result: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("init message produced no interop entries")
	}
	msg := result.Entries[0]
	return types.AccessList{{
		Address:     predeploys.CrossL2InboxAddr,
		StorageKeys: messages.EncodeAccessList([]messages.Access{msg.Access()}),
	}}, nil
}

// failsafe toggles and reads the op-interop-filter failsafe state via its
// JWT-protected admin RPC (admin_setFailsafeEnabled / admin_getFailsafeEnabled).
type failsafe struct {
	rpc opclient.RPC
}

func connectFilter(ctx context.Context, logger log.Logger, url, jwtPath string) (*failsafe, error) {
	secret, err := oprpc.ObtainJWTSecret(logger, jwtPath, false)
	if err != nil {
		return nil, fmt.Errorf("load filter JWT secret %q: %w", jwtPath, err)
	}
	rpcCl, err := opclient.NewRPC(ctx, logger.New("chain", "interop-filter-admin", "rpc", url), url,
		opclient.WithGethRPCOptions(gethrpc.WithHTTPAuth(gethnode.NewJWTAuth([32]byte(secret)))),
		opclient.WithFixedDialBackoff(time.Second),
		opclient.WithDialAttempts(5),
	)
	if err != nil {
		return nil, fmt.Errorf("dial filter admin RPC %s: %w", url, err)
	}
	return &failsafe{rpc: rpcCl}, nil
}

func (f *failsafe) get(ctx context.Context) (bool, error) {
	var enabled bool
	if err := f.rpc.CallContext(ctx, &enabled, "admin_getFailsafeEnabled"); err != nil {
		return false, fmt.Errorf("admin_getFailsafeEnabled: %w", err)
	}
	return enabled, nil
}

func (f *failsafe) set(ctx context.Context, enabled bool) error {
	if err := f.rpc.CallContext(ctx, nil, "admin_setFailsafeEnabled", enabled); err != nil {
		return fmt.Errorf("admin_setFailsafeEnabled(%v): %w", enabled, err)
	}
	return nil
}

func (f *failsafe) close() {
	f.rpc.Close()
}

// failsafeErrorCode is the dedicated JSON-RPC error code op-interop-filter returns from
// interop_checkAccessList when failsafe is enabled. See ethereum-optimism/optimism#21205
// (previously failsafe shared the generic invalid-params code).
const failsafeErrorCode = -320602

// connectFilterQuery dials the filter's public RPC for interop_checkAccessList. Unlike the admin
// endpoint it needs no JWT.
func connectFilterQuery(ctx context.Context, logger log.Logger, url string) (*sources.InteropFilterClient, error) {
	rpcCl, err := opclient.NewRPC(ctx, logger.New("chain", "interop-filter", "rpc", url), url,
		opclient.WithFixedDialBackoff(time.Second),
		opclient.WithDialAttempts(5),
	)
	if err != nil {
		return nil, fmt.Errorf("dial filter RPC %s: %w", url, err)
	}
	return sources.NewInteropFilterClient(rpcCl), nil
}

// assertFilterFailsafeCode queries interop_checkAccessList directly (the filter must already be in
// failsafe) and requires it to fail with the dedicated failsafe error code. Failsafe short-circuits
// all validation, so the executing descriptor's timestamp does not affect the outcome.
func assertFilterFailsafeCode(ctx context.Context, logger log.Logger, fc *sources.InteropFilterClient, execChain *chain, accessList types.AccessList) error {
	if len(accessList) == 0 {
		return fmt.Errorf("empty executing access list; cannot query the filter")
	}
	desc := messages.ExecutingDescriptor{
		ChainID:   execChain.chainID,
		Timestamp: uint64(time.Now().Unix()),
	}
	err := fc.CheckAccessList(ctx, accessList[0].StorageKeys, safety.CrossUnsafe, desc)
	if err == nil {
		return fmt.Errorf("interop_checkAccessList unexpectedly succeeded while failsafe is enabled")
	}
	var rpcErr gethrpc.Error
	if !errors.As(err, &rpcErr) {
		return fmt.Errorf("interop_checkAccessList failed without a JSON-RPC error code: %w", err)
	}
	if rpcErr.ErrorCode() != failsafeErrorCode {
		return fmt.Errorf("interop_checkAccessList returned code %d, want dedicated failsafe code %d (#21205): %w",
			rpcErr.ErrorCode(), failsafeErrorCode, err)
	}
	logger.Info("filter returned dedicated failsafe error code", "code", rpcErr.ErrorCode())
	return nil
}

// setFailsafeAndWait sets the failsafe flag, confirms the filter reflects it, then
// waits one Chain B block so op-reth's failsafe poller can pick up the new state
// before we depend on tx-pool behavior.
func setFailsafeAndWait(ctx context.Context, logger log.Logger, fs *failsafe, chainB *chain, enabled bool) error {
	if err := fs.set(ctx, enabled); err != nil {
		return err
	}
	deadline := time.Now().Add(waitTimeout)
	for {
		got, err := fs.get(ctx)
		if err == nil && got == enabled {
			break
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for filter failsafe state to become %v", enabled)
		}
		time.Sleep(500 * time.Millisecond)
	}
	logger.Debug("failsafe set on filter", "enabled", enabled)
	if _, err := waitForNextBlock(ctx, chainB); err != nil {
		return err
	}
	return nil
}

func waitForNextBlock(ctx context.Context, c *chain) (eth.BlockRef, error) {
	head, err := c.ethClient.BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return eth.BlockRef{}, fmt.Errorf("fetch latest block on %s: %w", c.name, err)
	}
	target := head.Number + 1
	deadline := time.Now().Add(waitTimeout)
	for {
		head, err = c.ethClient.BlockRefByLabel(ctx, eth.Unsafe)
		if err == nil && head.Number >= target {
			return head, nil
		}
		if err := ctx.Err(); err != nil {
			return eth.BlockRef{}, err
		}
		if time.Now().After(deadline) {
			return eth.BlockRef{}, fmt.Errorf("timed out waiting for %s head >= %d", c.name, target)
		}
		time.Sleep(time.Second)
	}
}

// interopTxRejectedError reports whether err is a known interop-tx rejection from
// op-geth, op-reth, or the interop filter (including failsafe).
func interopTxRejectedError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "transaction filtered out") || // op-geth: generic filter rejection
		strings.Contains(msg, "failed to parse access entry") || // op-interop-filter: bad access entry
		strings.Contains(msg, "interop failsafe is active") || // op-reth: cached failsafe fast-path
		strings.Contains(msg, "failsafe is enabled") || // op-interop-filter: failsafe at filter level
		strings.Contains(msg, "no healthy supervisor backends found") // proxyd interop_validation: maps the filter's ErrFailsafeEnabled (-320602, #21205) to HTTP 503 and marks the (single) filter backend unhealthy
}
