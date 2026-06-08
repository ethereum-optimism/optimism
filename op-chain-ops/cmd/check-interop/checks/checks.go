package checks

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/lmittmann/w3"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

const (
	// maxRelayRetries caps submission/inclusion retry attempts. The effective
	// bound is the per-leg context deadline (RelayTimeout); this is just large
	// enough not to give up before that deadline ends the retries.
	maxRelayRetries = 1024
	// execGasLimit is a fixed gas limit for the failsafe-blocked relay, so we
	// skip eth_estimateGas and let the txpool rejection surface on submission.
	execGasLimit = 300_000
)

// bridgeAmount is the ETH moved across chains on each leg.
var bridgeAmount = eth.OneHundredthEther

// sendETHFn is the SuperchainETHBridge.sendETH(to, chainId) entrypoint.
var sendETHFn = w3.MustNewFunc("sendETH(address,uint256)", "bytes32")

// CheckInteropConfig holds the dependencies shared by the interop checks.
type CheckInteropConfig struct {
	Log log.Logger
	L2A *sources.EthClient
	L2B *sources.EthClient
	Key *ecdsa.PrivateKey

	// L2AChainID and L2BChainID are the chain IDs of L2A and L2B, used as the
	// SuperchainETHBridge destination chain when bridging in each direction.
	L2AChainID eth.ChainID
	L2BChainID eth.ChainID

	// RelayTimeout bounds each bridge leg (the send and its self-relay).
	RelayTimeout time.Duration

	// FilterAdmin is the JWT-authenticated admin RPC of the op-interop-filter,
	// required only by CheckFailsafe.
	FilterAdmin client.RPC
	// PropagationWait is how long to wait after the initiating send before the
	// expected-to-be-blocked relay, so that absent failsafe it would be admittable.
	PropagationWait time.Duration
}

// Close releases the underlying RPC clients.
func (cfg *CheckInteropConfig) Close() {
	if cfg.L2A != nil {
		cfg.L2A.Close()
	}
	if cfg.L2B != nil {
		cfg.L2B.Close()
	}
	if cfg.FilterAdmin != nil {
		cfg.FilterAdmin.Close()
	}
}

// chain returns the client and chain ID for "A" or "B".
func (cfg *CheckInteropConfig) chain(name string) (*sources.EthClient, eth.ChainID) {
	if name == "A" {
		return cfg.L2A, cfg.L2AChainID
	}
	return cfg.L2B, cfg.L2BChainID
}

// sendETHTrigger initiates a cross-chain ETH transfer through the
// SuperchainETHBridge predeploy. The bridged amount is the tx value.
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

// CheckRoundTrip bridges ETH A -> B and then B -> A through the
// SuperchainETHBridge predeploy, relaying each message itself, repeated for the
// given number of iterations.
func CheckRoundTrip(ctx context.Context, cfg *CheckInteropConfig, iterations int) error {
	if iterations < 1 {
		return fmt.Errorf("iterations must be >= 1, got %d", iterations)
	}
	recipient := randomRecipient()
	for i := 1; i <= iterations; i++ {
		cfg.Log.Info("round-trip iteration", "iteration", i, "of", iterations)
		if err := cfg.bridgeETH(ctx, "A", "B", recipient); err != nil {
			return fmt.Errorf("iteration %d/%d: %w", i, iterations, err)
		}
		if err := cfg.bridgeETH(ctx, "B", "A", recipient); err != nil {
			return fmt.Errorf("iteration %d/%d: %w", i, iterations, err)
		}
	}
	cfg.Log.Info("interop ETH round-trip check passed", "iterations", iterations)
	return nil
}

// CheckFailsafe verifies the interop filter failsafe lifecycle: an interop ETH
// bridge succeeds, its relay is rejected while failsafe is enabled, and bridging
// succeeds again once failsafe is disabled.
func CheckFailsafe(ctx context.Context, cfg *CheckInteropConfig) error {
	if cfg.FilterAdmin == nil {
		return errors.New("failsafe check requires --filter.admin-rpc and --filter.jwt-secret")
	}
	recipient := randomRecipient()

	// Start from a known-disabled state in case a previous run left it enabled.
	if err := cfg.setFailsafe(ctx, false); err != nil {
		return err
	}

	cfg.Log.Info("step 1: bridge before failsafe (expect success)")
	if err := cfg.bridgeETH(ctx, "A", "B", recipient); err != nil {
		return err
	}

	cfg.Log.Info("step 2: enabling failsafe")
	if err := cfg.setFailsafe(ctx, true); err != nil {
		return err
	}

	cfg.Log.Info("step 3: relay with failsafe enabled (expect rejection)")
	if err := cfg.expectBlocked(ctx, "A", "B", recipient); err != nil {
		_ = cfg.setFailsafe(ctx, false) // best-effort: leave the filter disabled
		return err
	}

	cfg.Log.Info("step 4: disabling failsafe")
	if err := cfg.setFailsafe(ctx, false); err != nil {
		return err
	}

	// The retrying plan tolerates the brief window during which the execution
	// client's cached failsafe flag is still refreshing.
	cfg.Log.Info("step 5: bridge after failsafe disabled (expect success)")
	if err := cfg.bridgeETH(ctx, "A", "B", recipient); err != nil {
		return err
	}

	cfg.Log.Info("interop failsafe lifecycle check passed")
	return nil
}

// bridgeETH sends bridgeAmount of ETH from the test account on src to recipient
// on dst via the SuperchainETHBridge predeploy, relays the message itself, and
// asserts recipient's dst-chain balance grew by exactly bridgeAmount.
func (cfg *CheckInteropConfig) bridgeETH(ctx context.Context, src, dst string, recipient common.Address) error {
	ctx, cancel := context.WithTimeout(ctx, cfg.RelayTimeout)
	defer cancel()

	srcCl, _ := cfg.chain(src)
	dstCl, dstChainID := cfg.chain(dst)

	before, err := dstCl.BalanceAt(ctx, recipient, nil)
	if err != nil {
		return fmt.Errorf("read recipient balance on %s: %w", dst, err)
	}

	send := txintent.NewIntent[*sendETHTrigger, *txintent.InteropOutput](retryPlan(srcCl, cfg.Key), txplan.WithValue(bridgeAmount))
	send.Content.Set(&sendETHTrigger{Recipient: recipient, Destination: dstChainID})
	if _, err := send.PlannedTx.Success.Eval(ctx); err != nil {
		return fmt.Errorf("send ETH %s -> %s: %w", src, dst, err)
	}
	cfg.Log.Info("sent ETH", "from", src, "to", dst, "tx", send.PlannedTx.Included.Value().TxHash)

	relay := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](retryPlan(dstCl, cfg.Key))
	relay.Content.DependOn(&send.Result)
	relay.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &send.Result, &send.PlannedTx.Included, 1))
	if _, err := relay.PlannedTx.Success.Eval(ctx); err != nil {
		return fmt.Errorf("relay ETH on %s: %w", dst, err)
	}
	cfg.Log.Info("relayed ETH", "chain", dst, "tx", relay.PlannedTx.Included.Value().TxHash)

	after, err := dstCl.BalanceAt(ctx, recipient, nil)
	if err != nil {
		return fmt.Errorf("read recipient balance on %s: %w", dst, err)
	}
	if credited := new(big.Int).Sub(after, before); credited.Cmp(bridgeAmount.ToBig()) != 0 {
		return fmt.Errorf("recipient on %s credited %s wei, want %s wei", dst, credited, bridgeAmount.ToBig())
	}
	cfg.Log.Info("bridged ETH", "from", src, "to", dst, "amount", bridgeAmount, "recipient", recipient)
	return nil
}

// expectBlocked sends ETH on src, then submits the relay on dst once, asserting
// it is rejected by the interop filter (failsafe).
func (cfg *CheckInteropConfig) expectBlocked(ctx context.Context, src, dst string, recipient common.Address) error {
	srcCl, _ := cfg.chain(src)
	dstCl, dstChainID := cfg.chain(dst)

	send := txintent.NewIntent[*sendETHTrigger, *txintent.InteropOutput](retryPlan(srcCl, cfg.Key), txplan.WithValue(bridgeAmount))
	send.Content.Set(&sendETHTrigger{Recipient: recipient, Destination: dstChainID})
	sendCtx, cancel := context.WithTimeout(ctx, cfg.RelayTimeout)
	_, err := send.PlannedTx.Success.Eval(sendCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("send ETH %s -> %s: %w", src, dst, err)
	}

	// Let the destination ingest the message so that, absent failsafe, the relay
	// would have been admittable — making the rejection attributable to failsafe.
	select {
	case <-time.After(cfg.PropagationWait):
	case <-ctx.Done():
		return ctx.Err()
	}

	relay := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](singleSubmitPlan(dstCl, cfg.Key))
	relay.Content.DependOn(&send.Result)
	relay.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &send.Result, &send.PlannedTx.Included, 1))
	if _, err := relay.PlannedTx.Submitted.Eval(ctx); err == nil {
		return fmt.Errorf("relay on %s was accepted while failsafe enabled, expected rejection", dst)
	} else if !interopTxRejected(err) {
		return fmt.Errorf("relay on %s failed but not with a recognized interop filter rejection: %w", dst, err)
	}
	cfg.Log.Info("relay correctly rejected while failsafe enabled", "chain", dst)
	return nil
}

// setFailsafe toggles failsafe mode on the interop filter and confirms it took effect.
func (cfg *CheckInteropConfig) setFailsafe(ctx context.Context, enabled bool) error {
	if err := cfg.FilterAdmin.CallContext(ctx, nil, "admin_setFailsafeEnabled", enabled); err != nil {
		return fmt.Errorf("admin_setFailsafeEnabled(%v): %w", enabled, err)
	}
	var got bool
	if err := cfg.FilterAdmin.CallContext(ctx, &got, "admin_getFailsafeEnabled"); err != nil {
		return fmt.Errorf("admin_getFailsafeEnabled: %w", err)
	}
	if got != enabled {
		return fmt.Errorf("failsafe state did not update: wanted %v, got %v", enabled, got)
	}
	return nil
}

// retryPlan estimates gas and retries submission and inclusion to ride out
// cross-chain propagation; the overall wait is capped by the caller's context.
func retryPlan(cl *sources.EthClient, key *ecdsa.PrivateKey) txplan.Option {
	return txplan.Combine(
		txplan.WithChainID(cl),
		txplan.WithPrivateKey(key),
		txplan.WithPendingNonce(cl),
		txplan.WithAgainstLatestBlock(cl),
		txplan.WithEstimator(cl, true),
		txplan.WithRetrySubmission(cl, maxRelayRetries, retry.Exponential()),
		txplan.WithRetryInclusion(cl, maxRelayRetries, retry.Exponential()),
		txplan.WithBlockInclusionInfo(cl),
	)
}

// singleSubmitPlan submits once with a fixed gas limit (skipping eth_estimateGas)
// so a txpool rejection surfaces immediately instead of being retried.
func singleSubmitPlan(cl *sources.EthClient, key *ecdsa.PrivateKey) txplan.Option {
	return txplan.Combine(
		txplan.WithChainID(cl),
		txplan.WithPrivateKey(key),
		txplan.WithPendingNonce(cl),
		txplan.WithAgainstLatestBlock(cl),
		txplan.WithGasLimit(execGasLimit),
		txplan.WithTransactionSubmitter(cl),
	)
}

// randomRecipient returns a fresh address that only ever receives ETH, so its
// destination-chain balance delta over a bridge leg is exactly the bridged amount.
func randomRecipient() common.Address {
	var b [20]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("failed to read random recipient address: %w", err))
	}
	return common.Address(b)
}

// interopTxRejected reports whether err is a known interop-transaction rejection
// from op-geth, op-reth, or the interop filter.
func interopTxRejected(err error) bool {
	msg := err.Error()
	switch {
	// op-geth: generic filter rejection wrapping all causes
	case strings.Contains(msg, "transaction filtered out"):
		return true
	// op-interop-filter: malformed or unrecognized access list entry
	case strings.Contains(msg, "failed to parse access entry"):
		return true
	// op-reth fast-path: cached failsafe state rejects before calling the filter
	case strings.Contains(msg, "interop failsafe is active"):
		return true
	// op-interop-filter: failsafe enabled at the filter level
	case strings.Contains(msg, "failsafe is enabled"):
		return true
	default:
		return false
	}
}
