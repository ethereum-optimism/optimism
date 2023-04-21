package prefetcher

import (
	"context"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const maxAttempts = math.MaxInt // Succeed or die trying

type RetryingL1Source struct {
	logger   log.Logger
	source   L1Source
	strategy backoff.Strategy
}

func NewRetryingL1Source(logger log.Logger, source L1Source) *RetryingL1Source {
	return &RetryingL1Source{
		logger:   logger,
		source:   source,
		strategy: backoff.Exponential(),
	}
}

func (s *RetryingL1Source) InfoByHash(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, error) {
	var info eth.BlockInfo
	err := backoff.DoCtx(ctx, maxAttempts, s.strategy, func() error {
		res, err := s.source.InfoByHash(ctx, blockHash)
		if err != nil {
			s.logger.Warn("Failed to retrieve info", "hash", blockHash, "err", err)
			return err
		}
		info = res
		return nil
	})
	return info, err
}

func (s *RetryingL1Source) InfoAndTxsByHash(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	var info eth.BlockInfo
	var txs types.Transactions
	err := backoff.DoCtx(ctx, maxAttempts, s.strategy, func() error {
		i, t, err := s.source.InfoAndTxsByHash(ctx, blockHash)
		if err != nil {
			s.logger.Warn("Failed to retrieve info and txs", "hash", blockHash, "err", err)
			return err
		}
		info = i
		txs = t
		return nil
	})
	return info, txs, err
}

func (s *RetryingL1Source) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	var info eth.BlockInfo
	var rcpts types.Receipts
	err := backoff.DoCtx(ctx, maxAttempts, s.strategy, func() error {
		i, r, err := s.source.FetchReceipts(ctx, blockHash)
		if err != nil {
			s.logger.Warn("Failed to fetch receipts", "hash", blockHash, "err", err)
			return err
		}
		info = i
		rcpts = r
		return nil
	})
	return info, rcpts, err
}

var _ L1Source = (*RetryingL1Source)(nil)

type RetryingL2Source struct {
	logger   log.Logger
	source   L2Source
	strategy backoff.Strategy
}

func (s *RetryingL2Source) InfoAndTxsByHash(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	var info eth.BlockInfo
	var txs types.Transactions
	err := backoff.DoCtx(ctx, maxAttempts, s.strategy, func() error {
		i, t, err := s.source.InfoAndTxsByHash(ctx, blockHash)
		if err != nil {
			s.logger.Warn("Failed to retrieve info and txs", "hash", blockHash, "err", err)
			return err
		}
		info = i
		txs = t
		return nil
	})
	return info, txs, err
}

func (s *RetryingL2Source) NodeByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	var node []byte
	err := backoff.DoCtx(ctx, maxAttempts, s.strategy, func() error {
		n, err := s.source.NodeByHash(ctx, hash)
		if err != nil {
			s.logger.Warn("Failed to retrieve node", "hash", hash, "err", err)
			return err
		}
		node = n
		return nil
	})
	return node, err
}

func (s *RetryingL2Source) CodeByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	var code []byte
	err := backoff.DoCtx(ctx, maxAttempts, s.strategy, func() error {
		c, err := s.source.CodeByHash(ctx, hash)
		if err != nil {
			s.logger.Warn("Failed to retrieve code", "hash", hash, "err", err)
			return err
		}
		code = c
		return nil
	})
	return code, err
}

func NewRetryingL2Source(logger log.Logger, source L2Source) *RetryingL2Source {
	return &RetryingL2Source{
		logger:   logger,
		source:   source,
		strategy: backoff.Exponential(),
	}
}

var _ L2Source = (*RetryingL2Source)(nil)
