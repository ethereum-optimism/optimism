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

// replayNonceAttempts and replayNonceDelay bound the reads of the deployer nonce before the first
// send. A backend of a load-balanced RPC can lag the one that served the previous receipt, and
// nothing has been sent yet, so reading again is safe.
const (
	replayNonceAttempts = 5
	replayNonceDelay    = 2 * time.Second
)

// broadcastLiveScript keeps script execution separate from signing and transaction delivery.
// Each script dry-runs against the current L1 head, so it sees earlier scripts' confirmed
// deployments. Only forge's private transaction file is replayed; no unlocked RPC is exposed.
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
	head, nonce, err := retry.Do2(ctx, replayNonceAttempts, retry.Fixed(replayNonceDelay), func() (*types.Header, uint64, error) {
		head, err := client.HeaderByNumber(ctx, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read L1 head: %w", err)
		}
		nonce, err := client.NonceAt(ctx, from, head.Number)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read deployer nonce: %w", err)
		}
		return head, nonce, requireReplayNonce(ctx, client, from, nonce)
	})
	if err != nil {
		return out, err
	}

	transactions, err := env.Client.CaptureTransactions(func(client *forge.Client) error {
		var err error
		out, err = newCaller(client)(ctx, input, "--rpc-url", env.L1RPCUrl, "--sender", from.Hex(),
			"--fork-block-number", strconv.FormatUint(bigs.Uint64Strict(head.Number), 10), "--skip-simulation")
		return err
	})
	if err != nil {
		return out, fmt.Errorf("deployment simulation failed: %w", err)
	}
	bcasts := make([]broadcaster.Broadcast, 0, len(transactions))
	for i, tx := range transactions {
		bcast, err := replayBroadcast(tx, chainID, from, nonce+uint64(i))
		if err != nil {
			return out, err
		}
		bcasts = append(bcasts, bcast)
	}
	if len(bcasts) == 0 {
		return out, nil
	}
	// Revalidate after simulation, before the first live send. After that, txmgr owns nonce
	// sequencing: a lagging load-balanced RPC must not abort a partly completed script.
	if err := retry.Do0(ctx, replayNonceAttempts, retry.Fixed(replayNonceDelay), func() error {
		return requireReplayNonce(ctx, client, from, nonce)
	}); err != nil {
		return out, err
	}

	logger := env.Logger
	if logger == nil {
		logger = log.Root()
	}
	expectedNonce := nonce
	live, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:               logger,
		ChainID:              chainID,
		Client:               client,
		Signer:               nonceGuardSigner(key, chainID, &expectedNonce),
		From:                 from,
		ReceiptQueryInterval: env.ReceiptQueryInterval,
	})
	if err != nil {
		return out, fmt.Errorf("failed to create live broadcaster: %w", err)
	}
	for i, tx := range bcasts {
		logger.Info("sending forge transaction", "index", i+1, "total", len(bcasts), "nonce", expectedNonce)
		live.Hook(tx)
		if _, err := live.Broadcast(ctx); err != nil {
			return out, fmt.Errorf("failed to broadcast deployment transaction at nonce %d: %w", expectedNonce, err)
		}
		expectedNonce++
	}
	return out, nil
}

// nonceGuardSigner signs only a transaction at *expected, the nonce that forge simulated: another
// nonce would change the simulated CREATE addresses. txmgr signs again at the same nonce to bump a
// fee, which the guard allows.
func nonceGuardSigner(key *ecdsa.PrivateKey, chainID *big.Int, expected *uint64) opcrypto.SignerFn {
	sign := opcrypto.PrivateKeySignerFn(key, chainID)
	return func(_ context.Context, address common.Address, tx *types.Transaction) (*types.Transaction, error) {
		if tx.Nonce() != *expected {
			return nil, fmt.Errorf("deployment nonce changed: expected %d, got %d", *expected, tx.Nonce())
		}
		return sign(address, tx)
	}
}

// requireReplayNonce rejects pending transactions and unexpected account activity before
// replay starts: using another nonce would change the simulated CREATE addresses.
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
