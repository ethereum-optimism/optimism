package api

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/teleportr/bindings/deposit"
	"github.com/ethereum-optimism/optimism/teleportr/bindings/disburse"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ChainDataReader interface {
	Get(ctx context.Context) (*ChainData, error)
}

type ChainData struct {
	MaxBalance             *big.Int
	DisburserBalance       *big.Int
	NextDisbursementID     uint64
	DepositContractBalance *big.Int
	NextDepositID          uint64
	MaxDepositAmount       *big.Int
	MinDepositAmount       *big.Int
}

type chainDataReaderImpl struct {
	l1Client            *ethclient.Client
	l2Client            *ethclient.Client
	depositContract     *deposit.TeleportrDeposit
	depositContractAddr common.Address
	disburserContract   *disburse.TeleportrDisburser
	disburserWalletAddr common.Address
}

func NewChainDataReader(
	l1Client, l2Client *ethclient.Client,
	depositContractAddr, disburserWalletAddr common.Address,
	depositContract *deposit.TeleportrDeposit,
	disburserContract *disburse.TeleportrDisburser,
) ChainDataReader {
	return &chainDataReaderImpl{
		l1Client:            l1Client,
		l2Client:            l2Client,
		depositContract:     depositContract,
		depositContractAddr: depositContractAddr,
		disburserContract:   disburserContract,
		disburserWalletAddr: disburserWalletAddr,
	}
}

func (c *chainDataReaderImpl) maxDepositBalance(ctx context.Context) (*big.Int, error) {
	return c.depositContract.MaxBalance(&bind.CallOpts{
		Context: ctx,
	})
}

func (c *chainDataReaderImpl) disburserBalance(ctx context.Context) (*big.Int, error) {
	return c.l2Client.BalanceAt(ctx, c.disburserWalletAddr, nil)
}

func (c *chainDataReaderImpl) nextDisbursementID(ctx context.Context) (uint64, error) {
	total, err := c.disburserContract.TotalDisbursements(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return 0, err
	}
	return total.Uint64(), nil
}

func (c *chainDataReaderImpl) depositContractBalance(ctx context.Context) (*big.Int, error) {
	return c.l1Client.BalanceAt(ctx, c.depositContractAddr, nil)
}

func (c *chainDataReaderImpl) nextDepositID(ctx context.Context) (uint64, error) {
	total, err := c.depositContract.TotalDeposits(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return 0, err
	}
	return total.Uint64(), nil
}

func (c *chainDataReaderImpl) maxDepositAmount(ctx context.Context) (*big.Int, error) {
	return c.depositContract.MaxDepositAmount(&bind.CallOpts{
		Context: ctx,
	})
}

func (c *chainDataReaderImpl) minDepositAmount(ctx context.Context) (*big.Int, error) {
	return c.depositContract.MinDepositAmount(&bind.CallOpts{
		Context: ctx,
	})
}

func (c *chainDataReaderImpl) Get(ctx context.Context) (*ChainData, error) {
	maxBalance, err := c.maxDepositBalance(ctx)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("max_balance").Inc()
		return nil, err
	}

	disburserBal, err := c.disburserBalance(ctx)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("disburser_wallet_balance_at").Inc()
		return nil, err
	}
	nextDisbursementID, err := c.nextDisbursementID(ctx)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("next_disbursement_id").Inc()
		return nil, err
	}
	depositContractBalance, err := c.depositContractBalance(ctx)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("deposit_balance_at").Inc()
		return nil, err
	}
	nextDepositID, err := c.nextDepositID(ctx)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("next_deposit_id").Inc()
		return nil, err
	}
	maxDepositAmount, err := c.maxDepositAmount(ctx)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("max_deposit_amount").Inc()
		return nil, err
	}
	minDepositAmount, err := c.minDepositAmount(ctx)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("min_deposit_amount").Inc()
		return nil, err
	}

	return &ChainData{
		MaxBalance:             maxBalance,
		DisburserBalance:       disburserBal,
		NextDisbursementID:     nextDisbursementID,
		DepositContractBalance: depositContractBalance,
		NextDepositID:          nextDepositID,
		MaxDepositAmount:       maxDepositAmount,
		MinDepositAmount:       minDepositAmount,
	}, nil
}

type cachingChainDataReader struct {
	inner    ChainDataReader
	interval time.Duration
	last     time.Time
	data     *ChainData
	mu       sync.Mutex
}

func NewCachingChainDataReader(inner ChainDataReader, interval time.Duration) ChainDataReader {
	return &cachingChainDataReader{
		inner:    inner,
		interval: interval,
	}
}

func (c *cachingChainDataReader) Get(ctx context.Context) (*ChainData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data != nil && time.Since(c.last) < c.interval {
		return c.data, nil
	}

	data, err := c.inner.Get(ctx)
	if err != nil {
		return nil, err
	}
	c.data = data
	c.last = time.Now()
	return c.data, nil
}
