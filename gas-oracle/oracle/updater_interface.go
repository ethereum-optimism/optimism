package oracle

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/gas-oracle/bindings"
	ometrics "github.com/ethereum-optimism/optimism/gas-oracle/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	txSendCounter           = metrics.NewRegisteredCounter("tx/send", ometrics.DefaultRegistry)
	txNotSignificantCounter = metrics.NewRegisteredCounter("tx/not_significant", ometrics.DefaultRegistry)
	gasPriceGauge           = metrics.NewRegisteredGauge("gas_price", ometrics.DefaultRegistry)
	txConfTimer             = metrics.NewRegisteredTimer("tx/confirmed", ometrics.DefaultRegistry)
	txSendTimer             = metrics.NewRegisteredTimer("tx/send", ometrics.DefaultRegistry)
)

// getLatestBlockNumberFn is used by the GasPriceUpdater
// to get the latest block number. The outer function binds the
// inner function to a `bind.ContractBackend` which is implemented
// by the `ethclient.Client`
func wrapGetLatestBlockNumberFn(backend bind.ContractBackend) func() (uint64, error) {
	return func() (uint64, error) {
		tip, err := backend.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return 0, err
		}
		return tip.Number.Uint64(), nil
	}
}

// wrapGetGasUsedByBlock is used by the GasPriceUpdater to get
// the amount of gas used by a particular block. This is used to
// track gas usage over time
func wrapGetGasUsedByBlock(backend bind.ContractBackend) func(*big.Int) (uint64, error) {
	return func(number *big.Int) (uint64, error) {
		block, err := backend.HeaderByNumber(context.Background(), number)
		if err != nil {
			return 0, err
		}
		return block.GasUsed, nil
	}
}

// DeployContractBackend represents the union of the
// DeployBackend and the ContractBackend
type DeployContractBackend interface {
	bind.DeployBackend
	bind.ContractBackend
}

// updateL2GasPriceFn is used by the GasPriceUpdater
// to update the L2 gas price
// perhaps this should take an options struct along with the backend?
// how can this continue to be decomposed?
func wrapUpdateL2GasPriceFn(backend DeployContractBackend, cfg *Config) (func(uint64) error, error) {
	if cfg.privateKey == nil {
		return nil, errNoPrivateKey
	}
	if cfg.l2ChainID == nil {
		return nil, errNoChainID
	}

	opts, err := bind.NewKeyedTransactorWithChainID(cfg.privateKey, cfg.l2ChainID)
	if err != nil {
		return nil, err
	}
	// Once https://github.com/ethereum/go-ethereum/pull/23062 is released
	// then we can remove setting the context here
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	// Don't send the transaction using the `contract` so that we can inspect
	// it beforehand
	opts.NoSend = true

	// Create a new contract bindings in scope of the updateL2GasPriceFn
	// that is returned from this function
	contract, err := bindings.NewGasPriceOracle(cfg.gasPriceOracleAddress, backend)
	if err != nil {
		return nil, err
	}

	return func(updatedGasPrice uint64) error {
		log.Trace("UpdateL2GasPriceFn", "gas-price", updatedGasPrice)
		if cfg.gasPrice == nil {
			// Set the gas price manually to use legacy transactions
			gasPrice, err := backend.SuggestGasPrice(context.Background())
			if err != nil {
				log.Error("cannot fetch gas price", "message", err)
				return err
			}
			log.Trace("fetched L2 tx.gasPrice", "gas-price", gasPrice)
			opts.GasPrice = gasPrice
		} else {
			// Allow a configurable gas price to be set
			opts.GasPrice = cfg.gasPrice
		}

		// Query the current L2 gas price
		currentPrice, err := contract.GasPrice(&bind.CallOpts{
			Context: context.Background(),
		})
		if err != nil {
			log.Error("cannot fetch current gas price", "message", err)
			return err
		}

		// no need to update when they are the same
		if currentPrice.Uint64() == updatedGasPrice {
			log.Info("gas price did not change", "gas-price", updatedGasPrice)
			txNotSignificantCounter.Inc(1)
			return nil
		}

		// Only update the gas price when it must be changed by at least
		// a paramaterizable amount.
		if !isDifferenceSignificant(currentPrice.Uint64(), updatedGasPrice, cfg.l2GasPriceSignificanceFactor) {
			log.Info("gas price did not significantly change", "min-factor", cfg.l2GasPriceSignificanceFactor,
				"current-price", currentPrice, "next-price", updatedGasPrice)
			txNotSignificantCounter.Inc(1)
			return nil
		}

		// Set the gas price by sending a transaction
		tx, err := contract.SetGasPrice(opts, new(big.Int).SetUint64(updatedGasPrice))
		if err != nil {
			return err
		}

		log.Debug("updating L2 gas price", "tx.gasPrice", tx.GasPrice(), "tx.gasLimit", tx.Gas(),
			"tx.data", hexutil.Encode(tx.Data()), "tx.to", tx.To().Hex(), "tx.nonce", tx.Nonce())
		pre := time.Now()
		if err := backend.SendTransaction(context.Background(), tx); err != nil {
			return err
		}
		txSendTimer.Update(time.Since(pre))
		log.Info("L2 gas price transaction sent", "hash", tx.Hash().Hex())

		gasPriceGauge.Update(int64(updatedGasPrice))
		txSendCounter.Inc(1)

		if cfg.waitForReceipt {
			// Keep track of the time it takes to confirm the transaction
			pre := time.Now()
			// Wait for the receipt
			receipt, err := waitForReceipt(backend, tx)
			if err != nil {
				return err
			}
			txConfTimer.Update(time.Since(pre))

			log.Info("L2 gas price transaction confirmed", "hash", tx.Hash().Hex(),
				"gas-used", receipt.GasUsed, "blocknumber", receipt.BlockNumber)
		}
		return nil
	}, nil
}

// Only update the gas price when it must be changed by at least
// a paramaterizable amount. If the param is greater than the result
// of 1 - (min/max) where min and max are the gas prices then do not
// update the gas price
func isDifferenceSignificant(a, b uint64, c float64) bool {
	max := max(a, b)
	min := min(a, b)
	factor := 1 - (float64(min) / float64(max))
	return c <= factor
}

// Wait for the receipt by polling the backend
func waitForReceipt(backend DeployContractBackend, tx *types.Transaction) (*types.Receipt, error) {
	t := time.NewTicker(300 * time.Millisecond)
	receipt := new(types.Receipt)
	var err error
	for range t.C {
		receipt, err = backend.TransactionReceipt(context.Background(), tx.Hash())
		if errors.Is(err, ethereum.NotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if receipt != nil {
			t.Stop()
			break
		}
	}
	return receipt, nil
}

func max(a, b uint64) uint64 {
	if a >= b {
		return a
	}
	return b
}

func min(a, b uint64) uint64 {
	if a >= b {
		return b
	}
	return a
}
