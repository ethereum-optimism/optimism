package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"

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
	if c.String(L1RPC.Name) == "" {
		return nil, errors.New("--l1 is required")
	}
	return connect(ctx, c.String(L1RPC.Name), pairs)
}

// syncChains calls the permissionless sync on each chain's L1CrossDomainMessenger whose L2
// does not already match L1 (all of them with force), then waits for each L2 to match the L1
// pause as of the sync's L1 block.
func syncChains(ctx context.Context, logger log.Logger, out io.Writer, conn *connection, key *ecdsa.PrivateKey, timeout time.Duration, force bool) error {
	sender := crypto.PubkeyToAddress(key.PublicKey)
	_, _ = fmt.Fprintf(out, "Syncing the ETHLiquidity pause from L1 (chain %d), sender %s:\n", conn.L1ChainID, sender)
	type pending struct {
		ch   chain
		l1   bool
		skip bool
	}
	plan := make([]pending, 0, len(conn.Chains))
	for _, ch := range conn.Chains {
		l1p, err := l1Paused(ctx, conn.L1, ch.L1.Messenger, nil)
		if err != nil {
			return fmt.Errorf("chain %d: %w", ch.L2.ChainID, err)
		}
		l2p, err := l2Paused(ctx, ch.L2)
		if err != nil {
			return fmt.Errorf("chain %d: %w", ch.L2.ChainID, err)
		}
		skip := l1p == l2p && !force
		action := "sync"
		if skip {
			action = "already in sync, skipping (use --force to sync anyway)"
		}
		_, _ = fmt.Fprintf(out, "  chain %d: L1 paused=%t, L2 paused=%t: %s\n", ch.L2.ChainID, l1p, l2p, action)
		plan = append(plan, pending{ch: ch, l1: l1p, skip: skip})
	}

	syncData, err := syncFn.EncodeArgs()
	if err != nil {
		return err
	}
	var failed []error
	for _, p := range plan {
		if p.skip {
			continue
		}
		id := p.ch.L2.ChainID
		receipt, err := sendTx(ctx, logger, conn.L1, conn.L1ChainID, key, p.ch.L1.Messenger, syncData, timeout)
		if err != nil {
			failed = append(failed, fmt.Errorf("chain %d: %s: %w", id, syncSignature, err))
			continue
		}
		// The deposit carries the L1 pause as of the sync's block.
		want, err := l1Paused(ctx, conn.L1, p.ch.L1.Messenger, receipt.BlockNumber)
		if err != nil {
			failed = append(failed, fmt.Errorf("chain %d: %w", id, err))
			continue
		}
		_, _ = fmt.Fprintf(out, "  chain %d: sync included in L1 block %d (tx %s); waiting for L2 paused=%t\n",
			id, receipt.BlockNumber, receipt.TxHash, want)
		if err := waitL2Paused(ctx, p.ch.L2, want, timeout); err != nil {
			failed = append(failed, fmt.Errorf("chain %d: %w", id, err))
			continue
		}
		_, _ = fmt.Fprintf(out, "  chain %d: L2 paused=%t\n", id, want)
	}
	return errors.Join(failed...)
}

// waitL2Paused polls ETHLiquidity.paused() until it equals want.
func waitL2Paused(ctx context.Context, l2 *l2Client, want bool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var lastErr error
	for {
		got, err := l2Paused(ctx, l2)
		if err == nil && got == want {
			return nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("L2 paused() did not become %t: %w (last error: %w)", want, ctx.Err(), lastErr)
			}
			return fmt.Errorf("L2 paused() did not become %t: %w", want, ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// sendTx signs and sends an EIP-1559 call from key and waits for a successful receipt.
// Gas estimation runs first, so a call that would revert fails before anything is sent.
func sendTx(ctx context.Context, logger log.Logger, cl l1Backend, chainID uint64, key *ecdsa.PrivateKey, to common.Address, data []byte, timeout time.Duration) (*types.Receipt, error) {
	from := crypto.PubkeyToAddress(key.PublicKey)
	gas, err := cl.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &to, Data: data})
	if err != nil {
		if r := decodeRevert(err); r != nil {
			return nil, fmt.Errorf("would revert: %s", r)
		}
		return nil, fmt.Errorf("estimate gas: %w", err)
	}
	nonce, err := cl.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	tip, err := cl.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("gas tip: %w", err)
	}
	head, err := cl.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("head: %w", err)
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

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		receipt, err := cl.TransactionReceipt(waitCtx, tx.Hash())
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
		case <-waitCtx.Done():
			return nil, fmt.Errorf("tx %s not included: %w", tx.Hash(), waitCtx.Err())
		case <-time.After(pollInterval):
		}
	}
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
// renewed with extend().
func planGuardianStep(pause, pausable, active bool) guardianStep {
	switch {
	case pause && pausable:
		return guardianStep{Call: "pause", Reason: "not paused"}
	case pause && active:
		return guardianStep{Reason: "already paused"}
	case pause:
		return guardianStep{Call: "extend", Reason: "pause expired"}
	case pausable:
		return guardianStep{Reason: "not paused"}
	default:
		return guardianStep{Call: "unpause", Reason: "paused"}
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

	out := c.App.Writer
	guardian := crypto.PubkeyToAddress(guardianKey.PublicKey)
	ids := pauseIdentifiers(conn.Chains)
	type step struct {
		id   pauseIdentifier
		step guardianStep
	}
	var steps []step
	for _, id := range ids {
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
		st := planGuardianStep(pause, pausable, active)
		action := st.Call
		if action == "" {
			action = "nothing to do"
		}
		_, _ = fmt.Fprintf(out, "SuperchainConfig %s, ETHLockbox %s (chains %v): %s; %s\n",
			id.SuperchainConfig, id.Lockbox, id.ChainIDs, st.Reason, action)
		steps = append(steps, step{id: id, step: st})
	}

	timeout := c.Duration(Timeout.Name)
	for _, s := range steps {
		var data []byte
		switch s.step.Call {
		case "":
			continue
		case "pause":
			data, err = scPauseFn.EncodeArgs(s.id.Lockbox)
		case "extend":
			data, err = scExtendFn.EncodeArgs(s.id.Lockbox)
		case "unpause":
			data, err = scUnpauseFn.EncodeArgs(s.id.Lockbox)
		}
		if err != nil {
			return err
		}
		receipt, err := sendTx(ctx, logger, conn.L1, conn.L1ChainID, guardianKey, s.id.SuperchainConfig, data, timeout)
		if err != nil {
			return fmt.Errorf("SuperchainConfig %s %s(%s): %w", s.id.SuperchainConfig, s.step.Call, s.id.Lockbox, err)
		}
		_, _ = fmt.Fprintf(out, "SuperchainConfig %s: %s(%s) included in L1 block %d (tx %s)\n",
			s.id.SuperchainConfig, s.step.Call, s.id.Lockbox, receipt.BlockNumber, receipt.TxHash)
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
	return syncChains(ctx, logger, out, conn, syncKey, timeout, false)
}
