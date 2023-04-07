package l1

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type OracleL1Client struct {
	oracle Oracle
}

func NewOracleL1Client(oracle Oracle) *OracleL1Client {
	return &OracleL1Client{oracle: oracle}
}

func (o OracleL1Client) L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error) {
	//TODO implement me
	panic("implement me")
}

func (o OracleL1Client) L1BlockRefByNumber(ctx context.Context, u uint64) (eth.L1BlockRef, error) {
	//TODO implement me
	panic("implement me")
}

func (o OracleL1Client) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	//TODO implement me
	panic("implement me")
}

func (o OracleL1Client) L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error) {
	//TODO implement me
	panic("implement me")
}

func (o OracleL1Client) InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (o OracleL1Client) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	//TODO implement me
	panic("implement me")
}

var _ derive.L1Fetcher = (*OracleL1Client)(nil)
