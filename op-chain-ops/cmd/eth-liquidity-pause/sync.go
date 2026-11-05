package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/log"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

// pollInterval is how often receipts and L2 state are re-read while waiting.
var pollInterval = 2 * time.Second

func syncAction(c *cli.Context) error {
	logger, ctx := setup(c, c.App.ErrWriter)
	cfg := readTxMgrConfig(c)
	if !hasSigner(cfg) {
		return errors.New("a sender is required: set --private-key, --mnemonic or --signer.endpoint")
	}
	conn, err := connectWithL1(ctx, c)
	if err != nil {
		return err
	}
	defer conn.Close()
	mgr, err := newTxMgr(logger, cfg)
	if err != nil {
		return err
	}
	defer mgr.Close()
	return syncChains(ctx, c.App.Writer, conn, mgr, c.Duration(Timeout.Name), c.Bool(Force.Name))
}

// txSender is the part of txmgr.TxManager that the tool uses.
type txSender interface {
	From() common.Address
	Send(ctx context.Context, candidate txmgr.TxCandidate) (*types.Receipt, error)
	SendAsync(ctx context.Context, candidate txmgr.TxCandidate, ch chan txmgr.SendResponse)
}

// readTxMgrConfig reads the txmgr flags. The L1 endpoint comes from --l1.
func readTxMgrConfig(c *cli.Context) txmgr.CLIConfig {
	cfg := txmgr.ReadCLIConfig(c)
	cfg.L1RPCURL = c.String(L1RPC.Name)
	return cfg
}

func hasSigner(cfg txmgr.CLIConfig) bool {
	return cfg.PrivateKey != "" || cfg.Mnemonic != "" || cfg.SignerCLIConfig.Enabled()
}

func newTxMgr(logger log.Logger, cfg txmgr.CLIConfig) (*txmgr.SimpleTxManager, error) {
	mgr, err := txmgr.NewSimpleTxManager("eth-liquidity-pause", logger, &metrics.NoopTxMetrics{}, cfg)
	if err != nil {
		return nil, fmt.Errorf("create the transaction manager: %w", err)
	}
	return mgr, nil
}

// checkCall fails if a call from from to to with data would revert. It runs before each send,
// because txmgr retries a failed candidate for a minute and then resets its nonce, which can
// clash with sends still in flight. Other txmgr prepare failures still reset the nonce; a rerun
// recovers from that.
func checkCall(ctx context.Context, l1 l1Backend, from, to common.Address, data []byte) error {
	if _, err := l1.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &to, Data: data}); err != nil {
		if r := decodeRevert(err); r != nil {
			return fmt.Errorf("would revert: %s", r)
		}
		return fmt.Errorf("estimate gas: %w", err)
	}
	return nil
}

func connectWithL1(ctx context.Context, c *cli.Context) (*connection, error) {
	pairs, err := pairChains(c.StringSlice(L2RPCs.Name), c.StringSlice(L1Messengers.Name), true)
	if err != nil {
		return nil, err
	}
	return connect(ctx, c.String(L1RPC.Name), pairs, true)
}

// syncPrinter serializes progress lines from the per-chain waits.
type syncPrinter struct {
	mu  sync.Mutex
	out io.Writer
}

func (p *syncPrinter) printf(format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, _ = fmt.Fprintf(p.out, format, args...)
}

// syncChains calls the permissionless sync on each chain's L1CrossDomainMessenger whose L2
// does not currently match L1 (all of them with force). It sends every sync first, then waits
// for all of them together: for each L1 receipt, and then for L2 to apply the message that sync
// sent. timeout limits each L2 wait; mgr's own timeouts limit each L1 wait.
//
// Without force, a chain is skipped when the two sides match right now. That compares current
// state only: a sync already in flight may still change L2.
func syncChains(ctx context.Context, out io.Writer, conn *connection, mgr txSender, timeout time.Duration, force bool) error {
	sender := mgr.From()
	p := &syncPrinter{out: out}
	p.printf("Syncing the ETHLiquidity pause from L1 (chain %d), sender %s:\n", conn.L1ChainID, sender)
	var todo []chain
	for _, ch := range conn.Chains {
		l1p, err := l1Paused(ctx, conn.L1, ch.L1.Messenger, nil)
		if err != nil {
			return fmt.Errorf("chain %d: %w", ch.L2.ChainID, err)
		}
		l2p, err := l2Paused(ctx, ch.L2, nil)
		if err != nil {
			return fmt.Errorf("chain %d: %w", ch.L2.ChainID, err)
		}
		action := "sync"
		if l1p == l2p && !force {
			action = "in sync now, skipping (this compares current state only, not syncs still in flight; " +
				"use --force to sync anyway)"
		} else {
			todo = append(todo, ch)
		}
		p.printf("  chain %d: L1 paused=%t, L2 paused=%t: %s\n", ch.L2.ChainID, l1p, l2p, action)
	}
	if len(todo) == 0 {
		return nil
	}

	syncData, err := syncFn.EncodeArgs()
	if err != nil {
		return err
	}
	errs := make([]error, len(todo))
	sent := make([]chan txmgr.SendResponse, len(todo))
	for i, ch := range todo {
		if err := checkCall(ctx, conn.L1, sender, ch.L1.Messenger, syncData); err != nil {
			errs[i] = fmt.Errorf("chain %d: %s: %w", ch.L2.ChainID, syncSignature, err)
			continue
		}
		sent[i] = make(chan txmgr.SendResponse, 1)
		mgr.SendAsync(ctx, txmgr.TxCandidate{TxData: syncData, To: &ch.L1.Messenger}, sent[i])
		p.printf("  chain %d: sending %s\n", ch.L2.ChainID, syncSignature)
	}

	var wg sync.WaitGroup
	for i, ch := range todo {
		if sent[i] == nil {
			continue
		}
		wg.Go(func() {
			if err := awaitSync(ctx, p, ch, <-sent[i], timeout); err != nil {
				errs[i] = fmt.Errorf("chain %d: %w", ch.L2.ChainID, err)
			}
		})
	}
	wg.Wait()
	return errors.Join(errs...)
}

// awaitSync takes one chain's L1 send result and waits for L2 to apply the message it sent.
func awaitSync(ctx context.Context, p *syncPrinter, ch chain, res txmgr.SendResponse, timeout time.Duration) error {
	receipt, err := successfulReceipt(res)
	if err != nil {
		return fmt.Errorf("%s: %w", syncSignature, err)
	}
	msg, err := parseSyncMessage(receipt, ch.L1.Messenger)
	if err != nil {
		return fmt.Errorf("tx %s: %w", receipt.TxHash, err)
	}
	p.printf("  chain %d: sync included in L1 block %d (tx %s) with %s, carrying L1 paused=%t; waiting for L2 to apply it\n",
		ch.L2.ChainID, receipt.BlockNumber, receipt.TxHash, fmtMessageNonce(msg.Nonce), msg.Paused)
	st, err := waitL2Applied(ctx, ch.L2, msg, timeout)
	if err != nil {
		return err
	}
	if st.Nonce.Cmp(msg.Nonce) > 0 {
		p.printf("  chain %d: L2 already applied a later sync (%s); L2 paused=%t\n",
			ch.L2.ChainID, fmtMessageNonce(st.Nonce), st.Paused)
		return nil
	}
	p.printf("  chain %d: L2 applied the sync; L2 paused=%t\n", ch.L2.ChainID, st.Paused)
	return nil
}

// syncMessage is the L2 message one syncETHLiquidityPause() call sent.
type syncMessage struct {
	// Nonce is the versioned L1CrossDomainMessenger nonce, which ETHLiquidity records as
	// pausedMessageNonce once it applies the message.
	Nonce *big.Int
	// Paused is the L1 pause the sync read, and so the value L2 takes on.
	Paused bool
	// Hash is the cross-domain message hash: L2CrossDomainMessenger.failedMessages records it
	// if the message reverts on L2.
	Hash common.Hash
}

// parseSyncMessage reads the SentMessage event that messenger emitted for a sync and decodes
// the ETHLiquidity.setPaused call it carries.
func parseSyncMessage(receipt *types.Receipt, messenger common.Address) (*syncMessage, error) {
	for _, l := range receipt.Logs {
		if l.Address != messenger || len(l.Topics) == 0 || l.Topics[0] != sentMessageEvt.Topic0 {
			continue
		}
		var (
			target, sender  common.Address
			data            []byte
			nonce, gasLimit big.Int
		)
		if err := sentMessageEvt.DecodeArgs(l, &target, &sender, &data, &nonce, &gasLimit); err != nil {
			return nil, fmt.Errorf("decode SentMessage: %w", err)
		}
		if target != predeploys.ETHLiquidityAddr {
			continue
		}
		msg := &syncMessage{Nonce: &nonce}
		var setNonce big.Int
		if err := setPausedFn.DecodeArgs(data, &msg.Paused, &setNonce); err != nil {
			return nil, fmt.Errorf("decode ETHLiquidity.setPaused message: %w", err)
		}
		if setNonce.Cmp(&nonce) != 0 {
			return nil, fmt.Errorf("setPaused carries message nonce %s, but the message was sent with %s", &setNonce, &nonce)
		}
		hash, err := crossdomain.HashCrossDomainMessageV1(&nonce, sender, target, new(big.Int), &gasLimit, data)
		if err != nil {
			return nil, fmt.Errorf("hash the sync message: %w", err)
		}
		msg.Hash = hash
		return msg, nil
	}
	return nil, fmt.Errorf("no SentMessage to ETHLiquidity from L1CrossDomainMessenger %s", messenger)
}

// l2PauseState is ETHLiquidity's pause as of one L2 block.
type l2PauseState struct {
	// Nonce is pausedMessageNonce(): the versioned nonce of the last sync L2 applied.
	Nonce  *big.Int
	Paused bool
}

var (
	errMessageFailed   = errors.New("the sync message failed on L2")
	errAppliedMismatch = errors.New("L2 applied the sync message but paused() differs from it")
)

// waitL2Applied polls L2 until ETHLiquidity.pausedMessageNonce() reaches msg's nonce. At msg's
// own nonce, paused() must equal msg.Paused; a higher nonce means a later sync superseded msg.
// It fails early if the message failed on L2, and otherwise times out.
func waitL2Applied(ctx context.Context, l2 *l2Client, msg *syncMessage, timeout time.Duration) (*l2PauseState, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var (
		lastErr error
		last    *big.Int // pausedMessageNonce() at the last successful read
	)
	for {
		st, err := readL2Applied(ctx, l2, msg)
		switch {
		case err == nil && st.Nonce.Cmp(msg.Nonce) >= 0:
			return st, nil
		case errors.Is(err, errMessageFailed), errors.Is(err, errAppliedMismatch):
			return nil, err
		case err == nil:
			last, lastErr = st.Nonce, nil
		default:
			lastErr = err
		}
		select {
		case <-ctx.Done():
			applied := "pausedMessageNonce() never read"
			switch {
			case last == nil:
			case last.Sign() == 0:
				applied = "L2 never synced"
			default:
				applied = "L2 is at " + fmtMessageNonce(last)
			}
			err := fmt.Errorf("L2 did not apply %s (%s): %w; "+
				"L2 may not have derived the sync's L1 block yet, rerun `sync` to check again",
				fmtMessageNonce(msg.Nonce), applied, ctx.Err())
			if lastErr != nil {
				return nil, fmt.Errorf("%w (last error: %w)", err, lastErr)
			}
			return nil, err
		case <-time.After(pollInterval):
		}
	}
}

// readL2Applied reads ETHLiquidity's pause at the L2 head. Paused is read only once msg (or a
// later sync) is applied; before that it checks whether msg failed on L2.
func readL2Applied(ctx context.Context, l2 *l2Client, msg *syncMessage) (*l2PauseState, error) {
	head, err := l2.Eth.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("L2 block number: %w", err)
	}
	block := new(big.Int).SetUint64(head)
	nonce, err := l2PausedMessageNonce(ctx, l2, block)
	if err != nil {
		return nil, err
	}
	st := &l2PauseState{Nonce: nonce}
	if nonce.Cmp(msg.Nonce) < 0 {
		var failed bool
		if err := callView(ctx, l2.Eth, predeploys.L2CrossDomainMessengerAddr, block, failedMessagesFn, &failed, msg.Hash); err != nil {
			return nil, fmt.Errorf("L2CrossDomainMessenger.failedMessages(): %w", err)
		}
		if failed {
			return nil, fmt.Errorf("%w: L2CrossDomainMessenger.failedMessages(%s) is true; rerun `sync` to send a new one",
				errMessageFailed, msg.Hash)
		}
		return st, nil
	}
	if st.Paused, err = l2Paused(ctx, l2, block); err != nil {
		return nil, err
	}
	if nonce.Cmp(msg.Nonce) == 0 && st.Paused != msg.Paused {
		return nil, fmt.Errorf("%w: the message carries paused=%t, L2 paused=%t at block %d",
			errAppliedMismatch, msg.Paused, st.Paused, head)
	}
	return st, nil
}

// successfulReceipt returns res's receipt, or an error if the send failed or the tx reverted.
// txmgr returns the receipt of a reverted tx without an error.
func successfulReceipt(res txmgr.SendResponse) (*types.Receipt, error) {
	if res.Err != nil {
		return nil, res.Err
	}
	if err := checkStatus(res.Receipt); err != nil {
		return nil, err
	}
	return res.Receipt, nil
}

func checkStatus(r *types.Receipt) error {
	if r.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("tx %s reverted in L1 block %d", r.TxHash, r.BlockNumber)
	}
	return nil
}

// guardianStep is the SuperchainConfig call (if any) needed to move one identifier to the
// wanted state.
type guardianStep struct {
	// Call is "pause", "extend", "unpause" or "" when the identifier is already there.
	Call string
	// Reason explains the choice for the confirmation print.
	Reason string
}

// planGuardianStep picks the SuperchainConfig call for one identifier. pausable is
// SuperchainConfig.pausable(id) (no pause timestamp set) and active is paused(id) (set and not
// expired). pause() reverts once a timestamp is set, even if expired, so an expired pause is
// renewed with extend(), and unpause() clears it.
func planGuardianStep(pause, pausable, active bool) guardianStep {
	switch {
	case pause && pausable:
		return guardianStep{Call: "pause", Reason: "not paused"}
	case pause && active:
		return guardianStep{Reason: "already paused"}
	case pause:
		return guardianStep{Call: "extend", Reason: "pause expired but still set, so pause() would revert"}
	case pausable:
		return guardianStep{Reason: "not paused"}
	case active:
		return guardianStep{Call: "unpause", Reason: "paused"}
	default:
		return guardianStep{Call: "unpause", Reason: "pause expired but still set"}
	}
}

func devnetAction(c *cli.Context, pause bool) error {
	logger, ctx := setup(c, c.App.ErrWriter)
	if c.String(GuardianKey.Name) == "" {
		return fmt.Errorf("--%s is required", GuardianKey.Name)
	}
	syncCfg := readTxMgrConfig(c)
	guardianCfg := syncCfg
	guardianCfg.PrivateKey = c.String(GuardianKey.Name)
	guardianCfg.Mnemonic = ""
	guardianCfg.SignerCLIConfig = opsigner.NewCLIConfig()

	conn, err := connectWithL1(ctx, c)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := refuseProduction(conn.L1ChainID, conn.l2ChainIDs()); err != nil {
		return err
	}
	guardianMgr, err := newTxMgr(logger, guardianCfg)
	if err != nil {
		return err
	}
	defer guardianMgr.Close()
	// Without a separate sender, the guardian's txmgr also sends the sync, so one txmgr tracks
	// the guardian's nonce.
	var syncMgr txSender = guardianMgr
	if hasSigner(syncCfg) {
		mgr, err := newTxMgr(logger, syncCfg)
		if err != nil {
			return err
		}
		defer mgr.Close()
		syncMgr = mgr
	}
	return devnetSetPause(ctx, c.App.Writer, conn, guardianMgr, syncMgr, pause, c.Duration(Timeout.Name))
}

// devnetSetPause moves each chain's ETHLockbox identifier to the wanted pause as the
// SuperchainConfig guardian, then force-syncs every chain whose identifier got there. The sync
// is forced because L2 can already show the wanted value while an older sync is in flight.
func devnetSetPause(ctx context.Context, out io.Writer, conn *connection, guardianMgr, syncMgr txSender, pause bool, timeout time.Duration) error {
	guardian := guardianMgr.From()
	ids := pauseIdentifiers(conn.Chains)
	steps := make([]guardianStep, len(ids))
	for i, id := range ids {
		var onchainGuardian common.Address
		if err := callView(ctx, conn.L1, id.SuperchainConfig, nil, guardianFn, &onchainGuardian); err != nil {
			return fmt.Errorf("SuperchainConfig %s guardian(): %w", id.SuperchainConfig, err)
		}
		if onchainGuardian != guardian {
			return fmt.Errorf("--guardian-key is %s but SuperchainConfig %s guardian is %s", guardian, id.SuperchainConfig, onchainGuardian)
		}
		var pausable, active bool
		if err := callView(ctx, conn.L1, id.SuperchainConfig, nil, pausableFn, &pausable, id.Lockbox); err != nil {
			return fmt.Errorf("SuperchainConfig %s pausable(): %w", id.SuperchainConfig, err)
		}
		if err := callView(ctx, conn.L1, id.SuperchainConfig, nil, pausedForFn, &active, id.Lockbox); err != nil {
			return fmt.Errorf("SuperchainConfig %s paused(address): %w", id.SuperchainConfig, err)
		}
		steps[i] = planGuardianStep(pause, pausable, active)
		action := steps[i].Call
		if action == "" {
			action = "nothing to do"
		}
		_, _ = fmt.Fprintf(out, "SuperchainConfig %s, ETHLockbox %s (chains %v): %s; %s\n",
			id.SuperchainConfig, id.Lockbox, id.ChainIDs, steps[i].Reason, action)
	}

	// A failed guardian call leaves its chains out of the sync; the others still sync.
	var failed []error
	skip := make(map[uint64]bool)
	for i, id := range ids {
		if err := applyGuardianStep(ctx, out, conn, guardianMgr, id, steps[i].Call); err != nil {
			failed = append(failed, fmt.Errorf("%w; chains %v not synced", err, id.ChainIDs))
			for _, cid := range id.ChainIDs {
				skip[cid] = true
			}
		}
	}
	if !pause {
		for _, id := range ids {
			var global bool
			if err := callView(ctx, conn.L1, id.SuperchainConfig, nil, pausedForFn, &global, globalPauseAddress); err != nil {
				return fmt.Errorf("SuperchainConfig %s paused(address(0)): %w", id.SuperchainConfig, err)
			}
			if global {
				_, _ = fmt.Fprintf(out, "WARNING: SuperchainConfig %s global pause is active, so L1 stays paused for chains %v\n",
					id.SuperchainConfig, id.ChainIDs)
			}
		}
	}
	toSync := *conn
	toSync.Chains = nil
	for _, ch := range conn.Chains {
		if !skip[ch.L2.ChainID] {
			toSync.Chains = append(toSync.Chains, ch)
		}
	}
	if len(toSync.Chains) > 0 {
		failed = append(failed, syncChains(ctx, out, &toSync, syncMgr, timeout, true))
	}
	if err := errors.Join(failed...); err != nil {
		return fmt.Errorf("%w\nrerun the command to retry: it skips identifiers already in the wanted state", err)
	}
	return nil
}

// applyGuardianStep sends call ("pause", "extend", "unpause"; "" does nothing) for one
// identifier and waits for it.
func applyGuardianStep(ctx context.Context, out io.Writer, conn *connection, mgr txSender, id pauseIdentifier, call string) error {
	var (
		data []byte
		err  error
	)
	switch call {
	case "":
		return nil
	case "pause":
		data, err = scPauseFn.EncodeArgs(id.Lockbox)
	case "extend":
		data, err = scExtendFn.EncodeArgs(id.Lockbox)
	case "unpause":
		data, err = scUnpauseFn.EncodeArgs(id.Lockbox)
	default:
		return fmt.Errorf("unknown SuperchainConfig call %q", call)
	}
	if err != nil {
		return err
	}
	wrap := func(err error) error {
		return fmt.Errorf("SuperchainConfig %s %s(%s): %w", id.SuperchainConfig, call, id.Lockbox, err)
	}
	if err := checkCall(ctx, conn.L1, mgr.From(), id.SuperchainConfig, data); err != nil {
		return wrap(err)
	}
	receipt, err := mgr.Send(ctx, txmgr.TxCandidate{TxData: data, To: &id.SuperchainConfig})
	if err == nil {
		err = checkStatus(receipt)
	}
	if err != nil {
		return wrap(err)
	}
	_, _ = fmt.Fprintf(out, "SuperchainConfig %s: %s(%s) included in L1 block %d (tx %s)\n",
		id.SuperchainConfig, call, id.Lockbox, receipt.BlockNumber, receipt.TxHash)
	return nil
}
