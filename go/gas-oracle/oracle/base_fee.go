package oracle

import (
	"context"
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/go/gas-oracle/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

func wrapUpdateBaseFee(l1Backend bind.ContractTransactor, l2Backend DeployContractBackend, cfg *Config) (func() error, error) {
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
	contract, err := bindings.NewGasPriceOracle(cfg.gasPriceOracleAddress, l2Backend)
	if err != nil {
		return nil, err
	}

	return func() error {
		baseFee, err := contract.L1BaseFee(&bind.CallOpts{
			Context: context.Background(),
		})
		tip, err := l1Backend.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return err
		}
		if tip.BaseFee == nil {
			return errNoBaseFee
		}
		if !isDifferenceSignificant(baseFee.Uint64(), tip.BaseFee.Uint64(), cfg.l1BaseFeeSignificanceFactor) {
			log.Debug("non significant base fee update", "tip", tip.BaseFee, "current", baseFee)
			return nil
		}

		latestBaseFee := tip.BaseFee.Uint64()
		// If the max percent change is configured, don't allow the value to
		// change more than the configured value
		if cfg.maxPercentChangeL1BaseFee != 0.0 {
			maxValue := uint64(math.Ceil((1 + cfg.maxPercentChangeL1BaseFee) * math.Max(float64(baseFee.Uint64()), 1)))
			minValue := uint64(math.Ceil((1 - cfg.maxPercentChangeL1BaseFee) * math.Max(float64(baseFee.Uint64()), 1)))
			if latestBaseFee > maxValue {
				log.Info("Base fee adjusted upwards too quickly", "tip.baseFee", latestBaseFee, "using", maxValue)
				latestBaseFee = maxValue
			}
			if latestBaseFee < minValue {
				log.Info("Base fee adjusted downwards too quickly", "tip.baseFee", latestBaseFee, "using", minValue)
				latestBaseFee = minValue
			}
		}

		// Use the configured gas price if it is set,
		// otherwise use gas estimation
		if cfg.gasPrice != nil {
			opts.GasPrice = cfg.gasPrice
		} else {
			gasPrice, err := l2Backend.SuggestGasPrice(opts.Context)
			if err != nil {
				return err
			}
			opts.GasPrice = gasPrice
		}

		tx, err := contract.SetL1BaseFee(opts, tip.BaseFee)
		if err != nil {
			return err
		}
		log.Debug("updating L1 base fee", "tx.gasPrice", tx.GasPrice(), "tx.gasLimit", tx.Gas(),
			"tx.data", hexutil.Encode(tx.Data()), "tx.to", tx.To().Hex(), "tx.nonce", tx.Nonce())
		if err := l2Backend.SendTransaction(context.Background(), tx); err != nil {
			return fmt.Errorf("cannot update base fee: %w", err)
		}
		log.Info("L1 base fee transaction sent", "hash", tx.Hash().Hex())

		if cfg.waitForReceipt {
			// Wait for the receipt
			receipt, err := waitForReceipt(l2Backend, tx)
			if err != nil {
				return err
			}

			log.Info("base-fee transaction confirmed", "hash", tx.Hash().Hex(),
				"gas-used", receipt.GasUsed, "blocknumber", receipt.BlockNumber)
		}
		return nil
	}, nil
}
