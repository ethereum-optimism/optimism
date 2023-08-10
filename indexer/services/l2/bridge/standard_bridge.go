package bridge

import (
	"context"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type StandardBridge struct {
	name      string
	address   common.Address
	client    *ethclient.Client
	l2SB      *bindings.L2StandardBridge
	l2L1MP    *bindings.L2ToL1MessagePasser
	isBedrock bool
}

func (s *StandardBridge) Address() common.Address {
	return s.address
}

func (s *StandardBridge) GetWithdrawalsByBlockRange(ctx context.Context, start, end uint64) (WithdrawalsMap, error) {
	withdrawalsByBlockhash := make(map[common.Hash][]db.Withdrawal)
	opts := &bind.FilterOpts{
		Context: ctx,
		Start:   start,
		End:     &end,
	}

	iter, err := backoff.Do(ctx, 3, backoff.Exponential(), func() (*bindings.L2StandardBridgeWithdrawalInitiatedIterator, error) {
		return s.l2SB.FilterWithdrawalInitiated(opts, nil, nil, nil)
	})
	if err != nil {
		return nil, err
	}

	receipts := make(map[common.Hash]*types.Receipt)
	defer iter.Close()
	for iter.Next() {
		ev := iter.Event
		if s.isBedrock {
			receipt := receipts[ev.Raw.TxHash]
			if receipt == nil {
				receipt, err = s.client.TransactionReceipt(ctx, ev.Raw.TxHash)
				if err != nil {
					return nil, err
				}
				receipts[ev.Raw.TxHash] = receipt
			}

			var withdrawalInitiated *bindings.L2ToL1MessagePasserMessagePassed
			for _, eLog := range receipt.Logs {
				if len(eLog.Topics) == 0 || eLog.Topics[0] != withdrawals.MessagePassedTopic {
					continue
				}

				if withdrawalInitiated != nil {
					logger.Warn("detected multiple withdrawal initiated events! ignoring", "tx_hash", ev.Raw.TxHash)
					continue
				}

				withdrawalInitiated, err = s.l2L1MP.ParseMessagePassed(*eLog)
				if err != nil {
					return nil, err
				}
			}

			hash, err := withdrawals.WithdrawalHash(withdrawalInitiated)
			if err != nil {
				return nil, err
			}

			withdrawalsByBlockhash[ev.Raw.BlockHash] = append(
				withdrawalsByBlockhash[ev.Raw.BlockHash], db.Withdrawal{
					TxHash:      ev.Raw.TxHash,
					L1Token:     ev.L1Token,
					L2Token:     ev.L2Token,
					FromAddress: ev.From,
					ToAddress:   ev.To,
					Amount:      ev.Amount,
					Data:        ev.ExtraData,
					LogIndex:    ev.Raw.Index,
					BedrockHash: &hash,
				},
			)
		} else {
			withdrawalsByBlockhash[ev.Raw.BlockHash] = append(
				withdrawalsByBlockhash[ev.Raw.BlockHash], db.Withdrawal{
					TxHash:      ev.Raw.TxHash,
					L1Token:     ev.L1Token,
					L2Token:     ev.L2Token,
					FromAddress: ev.From,
					ToAddress:   ev.To,
					Amount:      ev.Amount,
					Data:        ev.ExtraData,
					LogIndex:    ev.Raw.Index,
				},
			)
		}
	}

	return withdrawalsByBlockhash, iter.Error()
}

func (s *StandardBridge) String() string {
	return s.name
}
