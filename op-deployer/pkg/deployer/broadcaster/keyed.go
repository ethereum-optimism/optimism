package broadcaster

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type KeyedBroadcaster struct {
	lgr    log.Logger
	mgr    txmgr.TxManager
	bcasts []script.Broadcast
	mtx    sync.Mutex
}

type KeyedBroadcasterOpts struct {
	Logger  log.Logger
	ChainID *big.Int
	Client  *ethclient.Client
	Signer  opcrypto.SignerFn
	From    common.Address
}

func NewKeyedBroadcaster(cfg KeyedBroadcasterOpts) (*KeyedBroadcaster, error) {
	mgrCfg := &txmgr.Config{
		Backend:                   cfg.Client,
		ChainID:                   cfg.ChainID,
		TxSendTimeout:             5 * time.Minute,
		TxNotInMempoolTimeout:     time.Minute,
		NetworkTimeout:            10 * time.Second,
		ReceiptQueryInterval:      time.Second,
		NumConfirmations:          1,
		SafeAbortNonceTooLowCount: 3,
		Signer:                    cfg.Signer,
		From:                      cfg.From,
		GasPriceEstimatorFn:       DeployerGasPriceEstimator,
	}

	minTipCap, err := eth.GweiToWei(1.0)
	if err != nil {
		panic(err)
	}
	minBaseFee, err := eth.GweiToWei(1.0)
	if err != nil {
		panic(err)
	}

	mgrCfg.RebroadcastInterval.Store(int64(12 * time.Second))
	mgrCfg.ResubmissionTimeout.Store(int64(48 * time.Second))
	mgrCfg.FeeLimitMultiplier.Store(5)
	mgrCfg.FeeLimitThreshold.Store(big.NewInt(100))
	mgrCfg.MinTipCap.Store(minTipCap)
	mgrCfg.MinBaseFee.Store(minBaseFee)

	mgr, err := txmgr.NewSimpleTxManagerFromConfig(
		"transactor",
		cfg.Logger,
		&metrics.NoopTxMetrics{},
		mgrCfg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tx manager: %w", err)
	}

	return &KeyedBroadcaster{
		lgr: cfg.Logger,
		mgr: mgr,
	}, nil
}

func (t *KeyedBroadcaster) Hook(bcast script.Broadcast) {
	if bcast.Type != script.BroadcastCreate2 && bcast.From != t.mgr.From() {
		panic(fmt.Sprintf("invalid from for broadcast:%v, expected:%v", bcast.From, t.mgr.From()))
	}
	t.mtx.Lock()
	t.bcasts = append(t.bcasts, bcast)
	t.mtx.Unlock()
}

func (t *KeyedBroadcaster) Broadcast(ctx context.Context) ([]BroadcastResult, error) {
	// Empty the internal broadcast buffer as soon as this method is called.
	t.mtx.Lock()
	bcasts := t.bcasts
	t.bcasts = nil
	t.mtx.Unlock()

	if len(bcasts) == 0 {
		return nil, nil
	}

	results := make([]BroadcastResult, 0, len(bcasts))
	for i, bcast := range bcasts {
		id := bcast.ID()
		t.lgr.Info("broadcasting transaction", "id", id, "nonce", bcast.Nonce)

		// A zero gas limit makes txmgr estimate against the target chain. Waiting
		// for confirmation ensures the next estimate sees this transaction's state.
		receipt, err := t.mgr.Send(ctx, asTxCandidate(bcast, 0))
		outRes := BroadcastResult{Broadcast: bcast, Receipt: receipt, Err: err}
		if err != nil {
			t.lgr.Error(
				"transaction failed",
				"id", id,
				"completed", i+1,
				"total", len(bcasts),
				"err", err,
			)
			results = append(results, outRes)
			return results, err
		}

		outRes.TxHash = receipt.TxHash
		if receipt.Status == 0 {
			failErr := fmt.Errorf("transaction failed: %s", receipt.TxHash)
			outRes.Err = failErr
			t.lgr.Error(
				"transaction failed on chain",
				"id", id,
				"completed", i+1,
				"total", len(bcasts),
				"hash", receipt.TxHash,
				"nonce", bcast.Nonce,
			)
			results = append(results, outRes)
			return results, failErr
		}

		t.lgr.Info(
			"transaction confirmed",
			"id", id,
			"completed", i+1,
			"total", len(bcasts),
			"hash", receipt.TxHash,
			"nonce", bcast.Nonce,
			"creation", receipt.ContractAddress,
		)
		results = append(results, outRes)
	}
	return results, nil
}

func asTxCandidate(bcast script.Broadcast, gasLimit uint64) txmgr.TxCandidate {
	value := ((*uint256.Int)(bcast.Value)).ToBig()
	var candidate txmgr.TxCandidate
	switch bcast.Type {
	case script.BroadcastCall:
		to := &bcast.To
		candidate = txmgr.TxCandidate{
			TxData:   bcast.Input,
			To:       to,
			Value:    value,
			GasLimit: gasLimit,
		}
	case script.BroadcastCreate:
		candidate = txmgr.TxCandidate{
			TxData:   bcast.Input,
			To:       nil,
			GasLimit: gasLimit,
		}
	case script.BroadcastCreate2:
		txData := make([]byte, len(bcast.Salt)+len(bcast.Input))
		copy(txData, bcast.Salt[:])
		copy(txData[len(bcast.Salt):], bcast.Input)

		candidate = txmgr.TxCandidate{
			TxData:   txData,
			To:       &script.DeterministicDeployerAddress,
			Value:    value,
			GasLimit: gasLimit,
		}
	default:
		panic(fmt.Sprintf("unrecognized broadcast type: '%s'", bcast.Type))
	}
	return candidate
}
