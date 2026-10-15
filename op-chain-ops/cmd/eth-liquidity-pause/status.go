package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

const (
	expectPaused   = "paused"
	expectUnpaused = "unpaused"
)

// chainStatus is the pause state of one chain.
type chainStatus struct {
	ChainID uint64 `json:"chainId"`
	RPC     string `json:"rpc"`
	// L2Paused is ETHLiquidity.paused(); nil if the call failed (L2PausedError says why, e.g.
	// an ETHLiquidity without the pause).
	L2Paused      *bool  `json:"l2Paused"`
	L2PausedError string `json:"l2PausedError,omitempty"`
	// L2PausedMessageNonce is ETHLiquidity.pausedMessageNonce(), the versioned
	// L1CrossDomainMessenger nonce of the last applied sync (0 if never synced); nil if paused()
	// failed.
	L2PausedMessageNonce *hexutil.Big `json:"l2PausedMessageNonce,omitempty"`
	// L2PausedMessageNonceError is set when pausedMessageNonce() reverts, e.g. on an
	// ETHLiquidity that predates the nonce.
	L2PausedMessageNonceError string `json:"l2PausedMessageNonceError,omitempty"`
	// SendETHReverts reports whether the SuperchainETHBridge.sendETH eth_call reverted.
	SendETHReverts bool        `json:"sendETHReverts"`
	SendETHRevert  *revertInfo `json:"sendETHRevert,omitempty"`
	// L1 fields are set only with --l1 and --l1-messenger. L1Paused is
	// L1CrossDomainMessenger.paused(), the ETHLockbox pause the sync mirrors.
	L1            *l1Contracts `json:"l1,omitempty"`
	L1Paused      *bool        `json:"l1Paused,omitempty"`
	L1PausedError string       `json:"l1PausedError,omitempty"`
	// InSync is set when both sides were read.
	InSync *bool `json:"inSync,omitempty"`
}

func statusAction(c *cli.Context) error {
	logger, ctx := setup(c, c.App.ErrWriter)
	expect := c.String(Expect.Name)
	if expect != "" && expect != expectPaused && expect != expectUnpaused {
		return fmt.Errorf("--expect must be %q or %q, got %q", expectPaused, expectUnpaused, expect)
	}
	fromStr := c.String(From.Name)
	if !common.IsHexAddress(fromStr) {
		return fmt.Errorf("--from %q is not an address", fromStr)
	}
	messengers := c.StringSlice(L1Messengers.Name)
	pairs, err := pairChains(c.StringSlice(L2RPCs.Name), messengers, false)
	if err != nil {
		return err
	}
	conn, err := connect(ctx, c.String(L1RPC.Name), pairs, len(messengers) > 0)
	if err != nil {
		return err
	}
	defer conn.Close()

	statuses := make([]chainStatus, 0, len(conn.Chains))
	for _, ch := range conn.Chains {
		st, err := readStatus(ctx, conn, ch, common.HexToAddress(fromStr), c.Uint64(Destination.Name))
		if err != nil {
			return fmt.Errorf("chain %d: %w", ch.L2.ChainID, err)
		}
		logger.Debug("read chain status", "chain", st.ChainID, "l2Paused", st.L2Paused, "l1Paused", st.L1Paused)
		statuses = append(statuses, st)
	}

	// In JSON mode stdout carries only the JSON document, so the --expect verdict goes to stderr.
	verdictOut := c.App.Writer
	if c.Bool(JSON.Name) {
		verdictOut = c.App.ErrWriter
		enc := json.NewEncoder(c.App.Writer)
		enc.SetIndent("", "  ")
		if err := enc.Encode(statuses); err != nil {
			return err
		}
	} else {
		printStatus(c.App.Writer, statuses)
	}

	if expect == "" {
		return nil
	}
	var problems []string
	for _, st := range statuses {
		problems = append(problems, checkExpectation(st, expect)...)
	}
	if len(problems) > 0 {
		return fmt.Errorf("expected every chain %s:\n  %s", expect, strings.Join(problems, "\n  "))
	}
	_, _ = fmt.Fprintf(verdictOut, "OK: every chain is %s\n", expect)
	return nil
}

// readStatus reads both sides of one chain's pause. Contract calls that revert are recorded in
// the status; only transport failures are returned as errors.
func readStatus(ctx context.Context, conn *connection, ch chain, from common.Address, destination uint64) (chainStatus, error) {
	st := chainStatus{ChainID: ch.L2.ChainID, RPC: ch.L2.URL, L1: ch.L1}

	paused, err := l2Paused(ctx, ch.L2, nil)
	switch {
	case err == nil:
		st.L2Paused = &paused
		nonce, err := l2PausedMessageNonce(ctx, ch.L2, nil)
		switch {
		case err == nil:
			st.L2PausedMessageNonce = (*hexutil.Big)(nonce)
		case decodeRevert(err) != nil:
			st.L2PausedMessageNonceError = decodeRevert(err).String()
		default:
			return st, err
		}
	case decodeRevert(err) != nil:
		st.L2PausedError = decodeRevert(err).String()
	default:
		return st, err
	}

	revert, err := simulateSendETH(ctx, ch.L2.Geth, ch.L2.ChainID, from, destination)
	if err != nil {
		return st, err
	}
	st.SendETHReverts = revert != nil
	st.SendETHRevert = revert

	if ch.L1 != nil {
		p, err := l1Paused(ctx, conn.L1, ch.L1.Messenger, nil)
		switch {
		case err == nil:
			st.L1Paused = &p
		case decodeRevert(err) != nil:
			st.L1PausedError = decodeRevert(err).String()
		default:
			return st, err
		}
		if st.L1Paused != nil && st.L2Paused != nil {
			inSync := *st.L1Paused == *st.L2Paused
			st.InSync = &inSync
		}
	}
	return st, nil
}

// simulateSendETH eth_calls SuperchainETHBridge.sendETH(from, destination) with 1 wei from
// from, overriding from's balance so any address works. It returns the revert, or nil if the
// call succeeded.
func simulateSendETH(ctx context.Context, l2 *gethclient.Client, chainID uint64, from common.Address, destination uint64) (*revertInfo, error) {
	if destination == 0 {
		// Any chain but our own gets past L2ToL2CrossDomainMessenger.sendMessage.
		destination = chainID + 1
	}
	data, err := sendETHFn.EncodeArgs(from, new(big.Int).SetUint64(destination))
	if err != nil {
		return nil, err
	}
	bridge := predeploys.SuperchainETHBridgeAddr
	msg := ethereum.CallMsg{From: from, To: &bridge, Value: big.NewInt(1), Data: data}
	overrides := map[common.Address]gethclient.OverrideAccount{from: {Balance: oneETH}}
	_, err = l2.CallContract(ctx, msg, nil, &overrides)
	if err == nil {
		return nil, nil
	}
	if r := decodeRevert(err); r != nil {
		return r, nil
	}
	return nil, fmt.Errorf("sendETH eth_call: %w", err)
}

// checkExpectation lists how st differs from expect. paused means ETHLiquidity.paused() is
// true, sendETH reverts with ETHLiquidity_Paused() and, when read, L1 is paused. unpaused means
// paused() is false, sendETH succeeds and, when read, L1 is unpaused.
func checkExpectation(st chainStatus, expect string) []string {
	want := expect == expectPaused
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf("chain %d: ", st.ChainID)+fmt.Sprintf(format, args...))
	}
	switch {
	case st.L2Paused == nil:
		add("ETHLiquidity.paused() failed: %s", st.L2PausedError)
	case *st.L2Paused != want:
		add("ETHLiquidity.paused() is %t", *st.L2Paused)
	}
	if want {
		switch {
		case !st.SendETHReverts:
			add("sendETH succeeds")
		case st.SendETHRevert.Name != pausedErrorName:
			add("sendETH reverts with %s, not %s", st.SendETHRevert, pausedErrorName)
		}
	} else if st.SendETHReverts {
		add("sendETH reverts with %s", st.SendETHRevert)
	}
	if st.L1 != nil {
		switch {
		case st.L1Paused == nil:
			add("L1CrossDomainMessenger.paused() failed: %s", st.L1PausedError)
		case *st.L1Paused != want:
			add("L1 paused() is %t", *st.L1Paused)
		}
	}
	return problems
}

func fmtBool(b *bool, errMsg string) string {
	if b == nil {
		return "error: " + errMsg
	}
	return fmt.Sprintf("%t", *b)
}

func printStatus(w io.Writer, statuses []chainStatus) {
	for _, st := range statuses {
		_, _ = fmt.Fprintf(w, "chain %d (%s)\n", st.ChainID, st.RPC)
		_, _ = fmt.Fprintf(w, "  L2 ETHLiquidity.paused(): %s", fmtBool(st.L2Paused, st.L2PausedError))
		switch {
		case st.L2PausedMessageNonceError != "":
			_, _ = fmt.Fprintf(w, " (pausedMessageNonce() failed: %s)", st.L2PausedMessageNonceError)
		case st.L2PausedMessageNonce == nil:
		case st.L2PausedMessageNonce.ToInt().Sign() == 0:
			_, _ = fmt.Fprintf(w, " (never synced)")
		default:
			_, _ = fmt.Fprintf(w, " (synced by %s)", fmtMessageNonce(st.L2PausedMessageNonce.ToInt()))
		}
		_, _ = fmt.Fprintln(w)
		if st.SendETHReverts {
			_, _ = fmt.Fprintf(w, "  sendETH eth_call:         reverts %s\n", st.SendETHRevert)
		} else {
			_, _ = fmt.Fprintf(w, "  sendETH eth_call:         succeeds\n")
		}
		if st.L1 == nil {
			continue
		}
		_, _ = fmt.Fprintf(w, "  L1 paused():              %s (ETHLockbox %s, SuperchainConfig %s)\n",
			fmtBool(st.L1Paused, st.L1PausedError), st.L1.Lockbox, st.L1.SuperchainConfig)
		if st.InSync != nil && !*st.InSync {
			_, _ = fmt.Fprintf(w, "  OUT OF SYNC: L1 paused=%t but L2 paused=%t; run `eth-liquidity-pause sync`\n",
				*st.L1Paused, *st.L2Paused)
		}
	}
}
