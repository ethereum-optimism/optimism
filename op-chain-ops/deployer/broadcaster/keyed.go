package broadcaster

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/go-multierror"
	"github.com/holiman/uint256"
)

const (
	GasPadFactor = 1.5
)

type KeyedBroadcaster struct {
	lgr    log.Logger
	mgr    txmgr.TxManager
	bcasts []script.Broadcast
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
	}

	minTipCap, err := eth.GweiToWei(1.0)
	if err != nil {
		panic(err)
	}
	minBaseFee, err := eth.GweiToWei(1.0)
	if err != nil {
		panic(err)
	}

	mgrCfg.ResubmissionTimeout.Store(int64(48 * time.Second))
	mgrCfg.FeeLimitMultiplier.Store(5)
	mgrCfg.FeeLimitThreshold.Store(big.NewInt(100))
	mgrCfg.MinTipCap.Store(minTipCap)
	mgrCfg.MinTipCap.Store(minBaseFee)

	mgr, err := txmgr.NewSimpleTxManagerFromConfig(
		"transactor",
		log.NewLogger(log.DiscardHandler()),
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
	t.bcasts = append(t.bcasts, bcast)
}

func (t *KeyedBroadcaster) Broadcast(ctx context.Context) ([]BroadcastResult, error) {
	results := make([]BroadcastResult, len(t.bcasts))
	futures := make([]<-chan txmgr.SendResponse, len(t.bcasts))
	ids := make([]common.Hash, len(t.bcasts))

	for i, bcast := range t.bcasts {
		futures[i], ids[i] = t.broadcast(ctx, bcast)
		t.lgr.Info(
			"transaction broadcasted",
			"id", ids[i],
			"nonce", bcast.Nonce,
		)
	}

	var err *multierror.Error
	var completed int
	for i, fut := range futures {
		bcastRes := <-fut
		completed++
		outRes := BroadcastResult{
			Broadcast: t.bcasts[i],
		}

		if bcastRes.Err == nil {
			outRes.Receipt = bcastRes.Receipt
			outRes.TxHash = bcastRes.Receipt.TxHash

			if bcastRes.Receipt.Status == 0 {
				failErr := fmt.Errorf("transaction failed: %s", outRes.Receipt.TxHash.String())
				err = multierror.Append(err, failErr)
				outRes.Err = failErr
				t.lgr.Error(
					"transaction failed on chain",
					"id", ids[i],
					"completed", completed,
					"total", len(t.bcasts),
					"hash", outRes.Receipt.TxHash.String(),
					"nonce", outRes.Broadcast.Nonce,
				)
			} else {
				t.lgr.Info(
					"transaction confirmed",
					"id", ids[i],
					"completed", completed,
					"total", len(t.bcasts),
					"hash", outRes.Receipt.TxHash.String(),
					"nonce", outRes.Broadcast.Nonce,
					"creation", outRes.Receipt.ContractAddress,
				)
			}
		} else {
			err = multierror.Append(err, bcastRes.Err)
			outRes.Err = bcastRes.Err
			t.lgr.Error(
				"transaction failed",
				"id", ids[i],
				"completed", completed,
				"total", len(t.bcasts),
				"err", bcastRes.Err,
			)
		}

		results = append(results, outRes)
	}
	return results, err.ErrorOrNil()
}

func (t *KeyedBroadcaster) broadcast(ctx context.Context, bcast script.Broadcast) (<-chan txmgr.SendResponse, common.Hash) {
	id := bcast.ID()

	value := ((*uint256.Int)(bcast.Value)).ToBig()
	var candidate txmgr.TxCandidate
	switch bcast.Type {
	case script.BroadcastCall:
		to := &bcast.To
		candidate = txmgr.TxCandidate{
			TxData:   bcast.Input,
			To:       to,
			Value:    value,
			GasLimit: padGasLimit(bcast.Input, bcast.GasUsed, false),
		}
	case script.BroadcastCreate:
		candidate = txmgr.TxCandidate{
			TxData:   bcast.Input,
			To:       nil,
			GasLimit: padGasLimit(bcast.Input, bcast.GasUsed, true),
		}
	}

	ch := make(chan txmgr.SendResponse, 1)
	t.mgr.SendAsync(ctx, candidate, ch)
	return ch, id
}

func padGasLimit(data []byte, gasUsed uint64, creation bool) uint64 {
	intrinsicGas, err := core.IntrinsicGas(data, nil, creation, true, true, false)
	// This method never errors - we should look into it if it does.
	if err != nil {
		panic(err)
	}

	return uint64(float64(intrinsicGas+gasUsed) * GasPadFactor)
}
