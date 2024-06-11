package sender

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var ErrTransactionReverted = errors.New("transaction published but reverted")

type TxSender struct {
	log log.Logger

	txMgr txmgr.TxManager
	queue *txmgr.Queue[int]
}

func NewTxSender(ctx context.Context, logger log.Logger, txMgr txmgr.TxManager, maxPending uint64) *TxSender {
	queue := txmgr.NewQueue[int](ctx, txMgr, maxPending)
	return &TxSender{
		log:   logger,
		txMgr: txMgr,
		queue: queue,
	}
}

func (s *TxSender) From() common.Address {
	return s.txMgr.From()
}

func (s *TxSender) SendAndWaitDetailed(txPurpose string, txs ...txmgr.TxCandidate) []error {
	receiptsCh := make(chan txmgr.TxReceipt[int], len(txs))
	for i, tx := range txs {
		s.queue.Send(i, tx, receiptsCh)
	}
	completed := 0
	errs := make([]error, len(txs))
	for completed < len(txs) {
		rcpt := <-receiptsCh
		completed++
		if rcpt.Err != nil {
			errs[rcpt.ID] = rcpt.Err
		} else if rcpt.Receipt != nil {
			if rcpt.Receipt.Status != types.ReceiptStatusSuccessful {
				errs[rcpt.ID] = fmt.Errorf("%w purpose: %v hash: %v", ErrTransactionReverted, txPurpose, rcpt.Receipt.TxHash)
			} else {
				s.log.Debug("Transaction successfully published", "tx_hash", rcpt.Receipt.TxHash, "purpose", txPurpose)
			}
		}
	}
	return errs
}

func (s *TxSender) SendAndWaitSimple(txPurpose string, txs ...txmgr.TxCandidate) error {
	errs := s.SendAndWaitDetailed(txPurpose, txs...)
	return errors.Join(errs...)
}
