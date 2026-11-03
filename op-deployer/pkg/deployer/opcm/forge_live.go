package opcm

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// l1WaitTimeout bounds each wait for the L1 RPC: for the deployer's pending transactions to be
// mined, and for the head to reach the block of the last send. A transaction that another program
// sends with the deployer key, such as a funding transfer, is mined within a few blocks. Variable
// to allow tests to shorten it.
var l1WaitTimeout = 2 * time.Minute

// l1PollInterval is how often a wait reads the L1 RPC again.
const l1PollInterval = 2 * time.Second

// maxSimulations bounds the runs of a script whose deployer nonce moves between its simulation and
// its first send.
const maxSimulations = 3

// forkAttempts bounds the runs of a script on a fork, before anything is sent: a backend of a
// load-balanced RPC that has not seen the fork block fails the run, and a later run can reach another
// backend. A script that reverts runs that often too.
const forkAttempts = 3

// broadcastLiveScript runs a script with forge on a fork of the L1 head, without a key, and sends
// the transactions that forge recorded with txmgr. Only forge's private transaction file is
// replayed, and no unlocked RPC is exposed. The simulated CREATE addresses depend on the deployer
// nonce, so the script runs again when the nonce moves before the first send, and the signer
// refuses any other nonce.
func broadcastLiveScript[I, O any](ctx context.Context, env *ForgeEnv, newCaller newScriptCaller[I, O], input I) (O, error) {
	var out O
	if env.PrivateKey == "" {
		return out, errors.New("private key is required for Forge deployments")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(env.PrivateKey, "0x"))
	if err != nil {
		return out, fmt.Errorf("invalid private key: %w", err)
	}
	from := crypto.PubkeyToAddress(key.PublicKey)
	client, err := ethclient.DialContext(ctx, env.L1RPCUrl)
	if err != nil {
		return out, fmt.Errorf("failed to connect to L1: %w", err)
	}
	defer client.Close()
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return out, fmt.Errorf("failed to read L1 chain ID: %w", err)
	}
	logger := env.logger()

	for run := 1; ; run++ {
		head, nonce, err := waitForSettledNonce(ctx, logger, client, from, env.sentBlock)
		if err != nil {
			return out, err
		}
		fork := strconv.FormatUint(bigs.Uint64Strict(head.Number), 10)
		var transactions []forge.Transaction
		out, transactions, err = retry.Do2(ctx, forkAttempts, retry.Fixed(l1PollInterval), func() (O, []forge.Transaction, error) {
			return newCaller(env.Client)(ctx, input, "--rpc-url", env.L1RPCUrl, "--sender", from.Hex(),
				"--fork-block-number", fork, "--skip-simulation")
		})
		if err != nil {
			return out, fmt.Errorf("deployment simulation failed: %w", err)
		}
		bcasts, err := replayBroadcasts(transactions, chainID, from, nonce)
		if err != nil {
			return out, err
		}
		if len(bcasts) == 0 {
			return out, nil
		}
		// Checked once, before the first send. After it, txmgr owns the nonce, so that a lagging
		// backend of a load-balanced RPC cannot abort a script that has sent a part.
		if err := requireReplayNonce(ctx, client, from, nonce); err != nil {
			if run == maxSimulations {
				return out, fmt.Errorf("deployer nonce check failed after %d simulations: %w", maxSimulations, err)
			}
			logger.Warn("deployer nonce check failed after the simulation; simulating again", "err", err)
			continue
		}
		sentBlock, err := sendReplay(ctx, logger, client, key, chainID, nonce, bcasts, env.ReceiptQueryInterval)
		env.sentBlock = max(env.sentBlock, sentBlock)
		return out, err
	}
}

// readLiveScript runs a script that only reads, on a fork of an L1 head that includes the block of
// the last send.
func readLiveScript[I, O any](ctx context.Context, env *ForgeEnv, newCaller newScriptCaller[I, O], input I) (O, error) {
	var out O
	client, err := ethclient.DialContext(ctx, env.L1RPCUrl)
	if err != nil {
		return out, fmt.Errorf("failed to connect to L1: %w", err)
	}
	defer client.Close()
	logger := env.logger()
	head, err := retry.Do(ctx, l1WaitAttempts(), retry.Fixed(l1PollInterval), func() (*types.Header, error) {
		head, err := l1Head(ctx, client, env.sentBlock)
		if err != nil {
			logger.Info("waiting for the L1 head", "err", err)
		}
		return head, err
	})
	if err != nil {
		return out, err
	}
	fork := strconv.FormatUint(bigs.Uint64Strict(head.Number), 10)
	return retry.Do(ctx, forkAttempts, retry.Fixed(l1PollInterval), func() (O, error) {
		out, _, err := newCaller(env.Client)(ctx, input, "--rpc-url", env.L1RPCUrl, "--fork-block-number", fork)
		return out, err
	})
}

// waitForSettledNonce waits until the L1 head includes the block of the last send and the deployer
// has no pending transaction, and returns the head and the deployer's nonce at it.
func waitForSettledNonce(ctx context.Context, logger log.Logger, client *ethclient.Client, from common.Address, sentBlock uint64) (*types.Header, uint64, error) {
	return retry.Do2(ctx, l1WaitAttempts(), retry.Fixed(l1PollInterval), func() (*types.Header, uint64, error) {
		head, err := l1Head(ctx, client, sentBlock)
		if err != nil {
			logger.Info("waiting for the L1 head", "err", err)
			return nil, 0, err
		}
		nonce, err := client.NonceAt(ctx, from, head.Number)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read deployer nonce: %w", err)
		}
		if err := requireReplayNonce(ctx, client, from, nonce); err != nil {
			logger.Info("waiting for the deployer's pending transactions", "err", err)
			return nil, 0, err
		}
		return head, nonce, nil
	})
}

func l1WaitAttempts() int {
	return max(1, int(l1WaitTimeout/l1PollInterval))
}

// l1Head returns the L1 head, or an error when it is behind sentBlock: a backend of a load-balanced
// RPC can lag the one that served the receipt of the last send.
func l1Head(ctx context.Context, client *ethclient.Client, sentBlock uint64) (*types.Header, error) {
	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read L1 head: %w", err)
	}
	if number := bigs.Uint64Strict(head.Number); number < sentBlock {
		return nil, fmt.Errorf("L1 head %d is behind block %d of the last deployment transaction", number, sentBlock)
	}
	return head, nil
}

// sendReplay sends bcasts with txmgr, one at a time from nonce on, and returns the block of the last
// receipt.
func sendReplay(ctx context.Context, logger log.Logger, client *ethclient.Client, key *ecdsa.PrivateKey, chainID *big.Int, nonce uint64, bcasts []broadcaster.Broadcast, receiptQueryInterval time.Duration) (uint64, error) {
	// The signer stops the send when it refuses a nonce: txmgr would retry the signing for a minute.
	sendCtx, stop := context.WithCancelCause(ctx)
	defer stop(nil)
	expectedNonce := nonce
	live, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:               logger,
		ChainID:              chainID,
		Client:               client,
		Signer:               nonceGuardSigner(key, chainID, &expectedNonce, stop),
		From:                 crypto.PubkeyToAddress(key.PublicKey),
		ReceiptQueryInterval: receiptQueryInterval,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create live broadcaster: %w", err)
	}
	var sentBlock uint64
	for i, tx := range bcasts {
		logger.Info("sending forge transaction", "index", i+1, "total", len(bcasts), "nonce", expectedNonce)
		live.Hook(tx)
		results, err := live.Broadcast(sendCtx)
		if err != nil {
			if cause := context.Cause(sendCtx); cause != nil && ctx.Err() == nil {
				err = cause
			}
			return sentBlock, fmt.Errorf("failed to broadcast deployment transaction at nonce %d: %w", expectedNonce, err)
		}
		sentBlock = bigs.Uint64Strict(results[len(results)-1].Receipt.BlockNumber)
		expectedNonce++
	}
	return sentBlock, nil
}

// nonceGuardSigner signs only a transaction at *expected, the nonce that forge simulated, and calls
// stop with its error when it refuses one. txmgr signs again at the same nonce to bump a fee, which
// the guard allows.
func nonceGuardSigner(key *ecdsa.PrivateKey, chainID *big.Int, expected *uint64, stop context.CancelCauseFunc) opcrypto.SignerFn {
	sign := opcrypto.PrivateKeySignerFn(key, chainID)
	return func(_ context.Context, address common.Address, tx *types.Transaction) (*types.Transaction, error) {
		if tx.Nonce() != *expected {
			err := fmt.Errorf("deployment nonce changed: expected %d, got %d", *expected, tx.Nonce())
			stop(err)
			return nil, err
		}
		return sign(address, tx)
	}
}

// requireReplayNonce checks that the deployer's latest and pending nonces are both expected: that it
// has no pending transaction and no other activity.
func requireReplayNonce(ctx context.Context, client *ethclient.Client, from common.Address, expected uint64) error {
	latest, err := client.NonceAt(ctx, from, nil)
	if err != nil {
		return fmt.Errorf("failed to read deployer nonce: %w", err)
	}
	pending, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return fmt.Errorf("failed to read pending deployer nonce: %w", err)
	}
	if latest != expected || pending != expected {
		return fmt.Errorf("deployer %s nonce mismatch: expected latest and pending %d, got latest %d and pending %d; resolve pending transactions or concurrent account activity, or use an RPC endpoint whose backends agree, before retrying", from, expected, latest, pending)
	}
	return nil
}

// replayBroadcasts checks the transactions that forge recorded and returns them as broadcasts, from
// nonce on.
func replayBroadcasts(transactions []forge.Transaction, chainID *big.Int, from common.Address, nonce uint64) ([]broadcaster.Broadcast, error) {
	bcasts := make([]broadcaster.Broadcast, 0, len(transactions))
	for i, tx := range transactions {
		bcast, err := replayBroadcast(tx, chainID, from, nonce+uint64(i))
		if err != nil {
			return nil, err
		}
		bcasts = append(bcasts, bcast)
	}
	return bcasts, nil
}

func replayBroadcast(tx forge.Transaction, chainID *big.Int, from common.Address, nonce uint64) (broadcaster.Broadcast, error) {
	if tx.From != from {
		return broadcaster.Broadcast{}, fmt.Errorf("simulated transaction sender %s does not match deployer %s", tx.From, from)
	}
	if uint64(tx.Nonce) != nonce {
		return broadcaster.Broadcast{}, fmt.Errorf("simulated transaction nonce mismatch: expected %d, got %d", nonce, tx.Nonce)
	}
	if (tx.Type != types.LegacyTxType && tx.Type != types.DynamicFeeTxType) || len(tx.AccessList) != 0 {
		return broadcaster.Broadcast{}, fmt.Errorf("unsupported simulated transaction type or access list: %d", tx.Type)
	}
	if tx.ChainID == nil || (*big.Int)(tx.ChainID).Cmp(chainID) != 0 {
		return broadcaster.Broadcast{}, fmt.Errorf("simulated transaction chain ID does not match L1 %s", chainID)
	}
	if tx.Value == nil {
		return broadcaster.Broadcast{}, errors.New("invalid simulated transaction value")
	}
	out := broadcaster.Broadcast{
		From:  tx.From,
		Input: tx.Input,
		Value: tx.Value,
		Type:  broadcaster.BroadcastCall,
		Nonce: uint64(tx.Nonce),
	}
	if tx.To == nil {
		out.Type = broadcaster.BroadcastCreate
	} else {
		out.To = *tx.To
	}
	return out, nil
}
