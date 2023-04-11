package l1

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrUnknownLabel = errors.New("unknown label")
)

type OracleL1Client struct {
	oracle Oracle
	head   eth.L1BlockRef
}

func NewOracleL1Client(oracle Oracle, l1Head common.Hash) *OracleL1Client {
	head := eth.InfoToL1BlockRef(oracle.HeaderByHash(l1Head))
	return &OracleL1Client{
		oracle: oracle,
		head:   head,
	}
}

func (o OracleL1Client) L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error) {
	switch label {
	case eth.Unsafe:
		return o.head, nil
	case eth.Safe:
		return o.head, nil
	case eth.Finalized:
		return o.head, nil
	}
	return eth.L1BlockRef{}, fmt.Errorf("%w: %s", ErrUnknownLabel, label)
}

func (o OracleL1Client) L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error) {
	if number > o.head.Number {
		return eth.L1BlockRef{}, fmt.Errorf("%w: block number %d", ErrNotFound, number)
	}
	head := o.head
	for head.Number > number {
		head = eth.InfoToL1BlockRef(o.oracle.HeaderByHash(head.ParentHash))
	}
	return head, nil
}

func (o OracleL1Client) L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error) {
	return eth.InfoToL1BlockRef(o.oracle.HeaderByHash(hash)), nil
}

func (o OracleL1Client) InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	return o.oracle.HeaderByHash(hash), nil
}

func (o OracleL1Client) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	info, rcpts := o.oracle.ReceiptsByHash(blockHash)
	return info, rcpts, nil
}

func (o OracleL1Client) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	info, txs := o.oracle.TransactionsByHash(hash)
	return info, txs, nil
}
