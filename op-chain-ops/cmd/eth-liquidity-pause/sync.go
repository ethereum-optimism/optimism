package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// pollInterval is how often receipts and L2 state are re-read while waiting.
var pollInterval = 2 * time.Second

func syncAction(c *cli.Context) error {
	logger, ctx := setup(c, c.App.ErrWriter)
	key, err := parseKey(Key.Name, c.String(Key.Name))
	if err != nil {
		return err
	}
	conn, err := connectWithL1(ctx, c)
	if err != nil {
		return err
	}
	defer conn.Close()
	return syncChains(ctx, logger, c.App.Writer, conn, key, c.Duration(Timeout.Name), c.Bool(Force.Name))
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
// for all of them together: for each L1 receipt, and for L2 to apply the message that sync sent.
//
// Without force, a chain is skipped when the two sides match right now. That compares current
// state only: a sync already in flight may still change L2.
func syncChains(ctx context.Context, logger log.Logger, out io.Writer, conn *connection, key *ecdsa.PrivateKey, timeout time.Duration, force bool) error {
	sender := crypto.PubkeyToAddress(key.PublicKey)
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
	nonce, err := conn.L1.PendingNonceAt(ctx, sender)
	if err != nil {
		return fmt.Errorf("L1 nonce of %s: %w", sender, err)
	}
	errs := make([]error, len(todo))
	sent := make([]*types.Transaction, len(todo))
	for i, ch := range todo {
		tx, err := sendTx(ctx, logger, conn.L1, conn.L1ChainID, key, nonce, ch.L1.Messenger, syncData)
		if err != nil {
			errs[i] = fmt.Errorf("chain %d: %s: %w", ch.L2.ChainID, syncSignature, err)
			// A failed send may still have reached the node, so don't reuse its nonce blindly.
			if n, err := conn.L1.PendingNonceAt(ctx, sender); err == nil {
				nonce = n
			}
			continue
		}
		nonce++
		sent[i] = tx
		p.printf("  chain %d: sent %s (tx %s)\n", ch.L2.ChainID, syncSignature, tx.Hash())
	}

	var wg sync.WaitGroup
	for i, ch := range todo {
		if sent[i] == nil {
			continue
		}
		wg.Go(func() {
			if err := awaitSync(ctx, logger, p, conn.L1, ch, sent[i], timeout); err != nil {
				errs[i] = fmt.Errorf("chain %d: %w", ch.L2.ChainID, err)
			}
		})
	}
	wg.Wait()
	return errors.Join(errs...)
}

// awaitSync waits for one chain's sync transaction to be included on L1 and for L2 to apply
// the message it sent.
func awaitSync(ctx context.Context, logger log.Logger, p *syncPrinter, l1 l1Backend, ch chain, tx *types.Transaction, timeout time.Duration) error {
	receipt, err := waitReceipt(ctx, logger, l1, tx, timeout)
	if err != nil {
		return err
	}
	msg, err := parseSyncMessage(receipt, ch.L1.Messenger)
	if err != nil {
		return fmt.Errorf("tx %s: %w", tx.Hash(), err)
	}
	p.printf("  chain %d: sync included in L1 block %d with %s, carrying L1 paused=%t; waiting for L2 to apply it\n",
		ch.L2.ChainID, receipt.BlockNumber, fmtMessageNonce(msg.Nonce), msg.Paused)
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

// sendTx signs and sends an EIP-1559 call from key with the given account nonce. Gas
// estimation runs first, so a call that would revert fails before anything is sent.
func sendTx(ctx context.Context, logger log.Logger, cl l1Backend, chainID uint64, key *ecdsa.PrivateKey, nonce uint64, to common.Address, data []byte) (*types.Transaction, error) {
	from := crypto.PubkeyToAddress(key.PublicKey)
	gas, err := cl.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &to, Data: data})
	if err != nil {
		if r := decodeRevert(err); r != nil {
			return nil, fmt.Errorf("would revert: %s", r)
		}
		return nil, fmt.Errorf("estimate gas: %w", err)
	}
	tip, err := cl.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("gas tip: %w", err)
	}
	head, err := cl.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("head: %w", err)
	}
	if head.BaseFee == nil {
		return nil, fmt.Errorf("L1 head %d has no base fee; the chain must support EIP-1559", head.Number)
	}
	feeCap := new(big.Int).Add(tip, new(big.Int).Mul(head.BaseFee, big.NewInt(2)))
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(chainID))
	tx, err := types.SignNewTx(key, signer, &types.DynamicFeeTx{
		ChainID:   new(big.Int).SetUint64(chainID),
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: feeCap,
		Gas:       gas * 6 / 5,
		To:        &to,
		Data:      data,
	})
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	if err := cl.SendTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}
	logger.Info("sent L1 transaction", "tx", tx.Hash(), "to", to, "nonce", nonce)
	return tx, nil
}

// waitReceipt waits up to timeout for tx's receipt and fails if the tx reverted.
func waitReceipt(ctx context.Context, logger log.Logger, cl l1Backend, tx *types.Transaction, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		receipt, err := cl.TransactionReceipt(ctx, tx.Hash())
		if err == nil {
			if receipt.Status != types.ReceiptStatusSuccessful {
				return nil, fmt.Errorf("tx %s reverted in block %d", tx.Hash(), receipt.BlockNumber)
			}
			return receipt, nil
		}
		if !errors.Is(err, ethereum.NotFound) {
			logger.Debug("receipt not available yet", "tx", tx.Hash(), "err", err)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("tx %s not included: %w", tx.Hash(), ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// sendAndWait sends one call from key at its pending nonce and waits for the receipt.
func sendAndWait(ctx context.Context, logger log.Logger, cl l1Backend, chainID uint64, key *ecdsa.PrivateKey, to common.Address, data []byte, timeout time.Duration) (*types.Receipt, error) {
	nonce, err := cl.PendingNonceAt(ctx, crypto.PubkeyToAddress(key.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	tx, err := sendTx(ctx, logger, cl, chainID, key, nonce, to, data)
	if err != nil {
		return nil, err
	}
	return waitReceipt(ctx, logger, cl, tx, timeout)
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
	guardianKey, err := parseKey(GuardianKey.Name, c.String(GuardianKey.Name))
	if err != nil {
		return err
	}
	syncKey := guardianKey
	if c.String(Key.Name) != "" {
		if syncKey, err = parseKey(Key.Name, c.String(Key.Name)); err != nil {
			return err
		}
	}
	conn, err := connectWithL1(ctx, c)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := refuseProduction(conn.L1ChainID, conn.l2ChainIDs()); err != nil {
		return err
	}
	return devnetSetPause(ctx, logger, c.App.Writer, conn, guardianKey, syncKey, pause, c.Duration(Timeout.Name))
}

// devnetSetPause moves each chain's ETHLockbox identifier to the wanted pause as the
// SuperchainConfig guardian, then force-syncs every chain whose identifier got there. The sync
// is forced because L2 can already show the wanted value while an older sync is in flight.
func devnetSetPause(ctx context.Context, logger log.Logger, out io.Writer, conn *connection, guardianKey, syncKey *ecdsa.PrivateKey, pause bool, timeout time.Duration) error {
	guardian := crypto.PubkeyToAddress(guardianKey.PublicKey)
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
		if err := applyGuardianStep(ctx, logger, out, conn, guardianKey, id, steps[i].Call, timeout); err != nil {
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
		failed = append(failed, syncChains(ctx, logger, out, &toSync, syncKey, timeout, true))
	}
	if err := errors.Join(failed...); err != nil {
		return fmt.Errorf("%w\nrerun the command to retry: it skips identifiers already in the wanted state", err)
	}
	return nil
}

// applyGuardianStep sends call ("pause", "extend", "unpause"; "" does nothing) for one
// identifier and waits for it.
func applyGuardianStep(ctx context.Context, logger log.Logger, out io.Writer, conn *connection, guardianKey *ecdsa.PrivateKey, id pauseIdentifier, call string, timeout time.Duration) error {
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
	receipt, err := sendAndWait(ctx, logger, conn.L1, conn.L1ChainID, guardianKey, id.SuperchainConfig, data, timeout)
	if err != nil {
		return fmt.Errorf("SuperchainConfig %s %s(%s): %w", id.SuperchainConfig, call, id.Lockbox, err)
	}
	_, _ = fmt.Fprintf(out, "SuperchainConfig %s: %s(%s) included in L1 block %d (tx %s)\n",
		id.SuperchainConfig, call, id.Lockbox, receipt.BlockNumber, receipt.TxHash)
	return nil
}
