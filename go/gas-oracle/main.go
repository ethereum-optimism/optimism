package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/go/gas-oracle/bindings"
	"github.com/ethereum-optimism/optimism/go/gas-oracle/flags"
	"github.com/ethereum-optimism/optimism/go/gas-oracle/gasprices"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
)

var errInvalidSigningKey = errors.New("invalid signing key")
var errNoChainID = errors.New("no chain id provided")
var errNoPrivateKey = errors.New("no private key provided")

// GasPriceOracle manages a hot key that can update the L2 Gas Price
type GasPriceOracle struct {
	signer                       types.Signer
	chainID                      *big.Int
	ctx                          context.Context
	stop                         chan struct{}
	contract                     *bindings.GasPriceOracle
	privateKey                   *ecdsa.PrivateKey
	client                       *ethclient.Client
	gasPrice                     *big.Int
	gasPricer                    *gasprices.L2GasPricer
	averageBlockGasLimitPerEpoch float64
}

// Import the contract binding
func (g *GasPriceOracle) Start() error {
	if g.chainID == nil {
		return errNoChainID
	}
	if g.privateKey == nil {
		return errNoPrivateKey
	}

	address := crypto.PubkeyToAddress(g.privateKey.PublicKey)
	log.Info("Starting Gas Price Oracle", "chain-id", g.chainID, "address", address.Hex())

	// Fetch the owner of the contract to check that the local key matches
	// the owner of the contract. If it doesn't match then nothing can be
	// accomplished.
	owner, err := g.contract.Owner(&bind.CallOpts{
		Context: g.ctx,
	})
	if err != nil {
		return err
	}

	if address != owner {
		log.Error("Signing key does not match contract owner", "signer", address.Hex(), "owner", owner.Hex())
		return errInvalidSigningKey
	}

	// TODO: break this up into smaller functions
	// TODO: Errors in this goroutine should write to an error channel
	// and be handled externally
	go func() {
		timer := time.NewTicker(5 * time.Second)
		// There should never be an error here as long as a chain id is passed in
		opts, _ := bind.NewKeyedTransactorWithChainID(g.privateKey, g.chainID)
		// Once https://github.com/ethereum/go-ethereum/pull/23062 is released
		// then we can remove setting the context here
		opts.Context = g.ctx

		// getLatestBlockNumberFn is used by the GasPriceUpdater
		// to get the latest block number
		getLatestBlockNumberFn := func() (uint64, error) {
			tip, err := g.client.HeaderByNumber(g.ctx, nil)
			if err != nil {
				return 0, err
			}
			return tip.Number.Uint64(), nil
		}

		// updateL2GasPriceFn is used by the GasPriceUpdater
		// to update the L2 gas price
		updateL2GasPriceFn := func(num float64) error {
			if g.gasPrice == nil {
				gasPrice, err := g.client.SuggestGasPrice(g.ctx)
				if err == nil {
					fmt.Println(err)
				}
				opts.GasPrice = gasPrice
			} else {
				opts.GasPrice = g.gasPrice
			}

			updatedGasPrice := uint64(num)
			updatedGasPrice = 0

			tx, err := g.contract.SetGasPrice(opts, new(big.Int).SetUint64(updatedGasPrice))
			if err != nil {
				return err
			}
			log.Info("transaction sent", "hash", tx.Hash().Hex())

			// Wait for the receipt
			ticker := time.NewTicker(100 * time.Millisecond)
			receipt := new(types.Receipt)
		loop:
			for range ticker.C {
				receipt, err = g.client.TransactionReceipt(g.ctx, tx.Hash())
				if errors.Is(err, ethereum.NotFound) {
					continue
				}
				if err == nil {
					break loop
				}
			}
			log.Info("transaction confirmed", "hash", tx.Hash().Hex(),
				"gas-used", receipt.GasUsed, "blocknumber", receipt.BlockNumber)
			return nil
		}

		tip, err := g.client.HeaderByNumber(g.ctx, nil)
		if err != nil {
			log.Crit("Cannot fetch tip", "message", err)
		}
		epochStartBlockNumber := float64(tip.Number.Uint64())

		gasPriceUpdater := gasprices.NewGasPriceUpdater(
			g.gasPricer,
			epochStartBlockNumber,
			g.averageBlockGasLimitPerEpoch,
			getLatestBlockNumberFn,
			updateL2GasPriceFn,
		)

		for {
			select {
			case <-timer.C:
				log.Debug("polling", "time", time.Now())

				l2GasPrice, err := g.contract.GasPrice(&bind.CallOpts{
					Context: g.ctx,
				})
				if err != nil {
					log.Error("cannot get gas price", "message", err)
					continue
				}

				if err := gasPriceUpdater.UpdateGasPrice(); err != nil {
					log.Error("cannot update gas price", "message", err)
					continue
				}

				newGasPrice := gasPriceUpdater.GetGasPrice()
				log.Info("Updated gas price", "previous", l2GasPrice, "current", newGasPrice)
			case <-g.ctx.Done():
				g.Stop()
			}
		}
	}()

	return nil
}

func (g *GasPriceOracle) Stop() {
	close(g.stop)
}

func (g *GasPriceOracle) Wait() {
	<-g.stop
}

func NewGasPriceOracle(cfg *config) (*GasPriceOracle, error) {
	client, err := ethclient.Dial(cfg.ethereumHttpUrl)
	if err != nil {
		return nil, err
	}

	// Ensure that we can actually connect
	t := time.NewTicker(5 * time.Second)
	for ; true; <-t.C {
		_, err := client.ChainID(context.Background())
		if err == nil {
			t.Stop()
			break
		}
	}

	address := cfg.gasPriceOracleAddress
	contract, err := bindings.NewGasPriceOracle(address, client)
	if err != nil {
		return nil, err
	}

	// Fetch the current gas price to use as the current price
	currentPrice, err := contract.GasPrice(&bind.CallOpts{
		Context: context.Background(),
	})
	if err != nil {
		return nil, err
	}

	// TODO: ?
	maxPercentChangePerEpoch := float64(0)
	//

	gasPricer := gasprices.NewGasPricer(
		float64(currentPrice.Uint64()),
		cfg.floorPrice,
		func() float64 {
			return cfg.targetGasPerSecond
		},
		maxPercentChangePerEpoch,
	)

	chainID := cfg.chainID
	if chainID == nil {
		log.Info("ChainID unset, fetching remote")
		chainID, err = client.ChainID(context.Background())
		if err != nil {
			return nil, err
		}
	}

	privateKey := cfg.privateKey
	if privateKey == nil {
		return nil, errNoPrivateKey
	}

	return &GasPriceOracle{
		signer:                       types.NewEIP155Signer(chainID),
		chainID:                      chainID,
		ctx:                          context.Background(),
		stop:                         make(chan struct{}),
		contract:                     contract,
		privateKey:                   privateKey,
		client:                       client,
		gasPrice:                     cfg.gasPrice,
		gasPricer:                    gasPricer,
		averageBlockGasLimitPerEpoch: cfg.averageBlockGasLimitPerEpoch,
	}, nil
}

type config struct {
	chainID                      *big.Int
	ethereumHttpUrl              string
	gasPriceOracleAddress        common.Address
	privateKey                   *ecdsa.PrivateKey
	gasPrice                     *big.Int
	floorPrice                   float64
	targetGasPerSecond           float64
	maxPercentChangePerSecond    float64
	averageBlockGasLimitPerEpoch float64
}

func newConfig(ctx *cli.Context) *config {
	cfg := config{
		gasPriceOracleAddress: common.HexToAddress("0x420000000000000000000000000000000000000F"),
	}
	if ctx.GlobalIsSet(flags.EthereumHttpUrlFlag.Name) {
		cfg.ethereumHttpUrl = ctx.GlobalString(flags.EthereumHttpUrlFlag.Name)
	}
	if ctx.GlobalIsSet(flags.ChainIDFlag.Name) {
		chainID := ctx.GlobalUint64(flags.ChainIDFlag.Name)
		cfg.chainID = new(big.Int).SetUint64(chainID)
	}
	if ctx.GlobalIsSet(flags.GasPriceOracleAddressFlag.Name) {
		addr := ctx.GlobalString(flags.GasPriceOracleAddressFlag.Name)
		cfg.gasPriceOracleAddress = common.HexToAddress(addr)
	}
	if ctx.GlobalIsSet(flags.PrivateKeyFlag.Name) {
		hex := ctx.GlobalString(flags.PrivateKeyFlag.Name)
		if strings.HasPrefix(hex, "0x") {
			hex = hex[2:]
		}
		key, err := crypto.HexToECDSA(hex)
		if err != nil {
			fmt.Printf("Option %q: %v", flags.PrivateKeyFlag.Name, err)
		}
		cfg.privateKey = key
	}
	if ctx.GlobalIsSet(flags.TransactionGasPriceFlag.Name) {
		gasPrice := ctx.GlobalUint64(flags.TransactionGasPriceFlag.Name)
		cfg.gasPrice = new(big.Int).SetUint64(gasPrice)
	}
	if ctx.GlobalIsSet(flags.FloorPriceFlag.Name) {
		cfg.floorPrice = ctx.GlobalFloat64(flags.FloorPriceFlag.Name)
	}
	if ctx.GlobalIsSet(flags.TargetGasPerSecondFlag.Name) {
		cfg.targetGasPerSecond = ctx.GlobalFloat64(flags.TargetGasPerSecondFlag.Name)
	} else {
		log.Crit("Missing config option", "option", flags.TargetGasPerSecondFlag.Name)
	}
	if ctx.GlobalIsSet(flags.MaxPercentChangePerSecondFlag.Name) {
		cfg.maxPercentChangePerSecond = ctx.GlobalFloat64(flags.MaxPercentChangePerSecondFlag.Name)
	} else {
		log.Crit("Missing config option", "option", flags.MaxPercentChangePerSecondFlag.Name)
	}
	if ctx.GlobalIsSet(flags.AverageBlockGasLimitPerEpochFlag.Name) {
		cfg.averageBlockGasLimitPerEpoch = ctx.GlobalFloat64(flags.AverageBlockGasLimitPerEpochFlag.Name)
	} else {
		log.Crit("Missing config option", "option", flags.AverageBlockGasLimitPerEpochFlag.Name)
	}
	return &cfg
}

func main() {
	app := cli.NewApp()
	app.Flags = flags.Flags

	app.Before = func(ctx *cli.Context) error {
		loglevel := ctx.GlobalUint64(flags.LogLevelFlag.Name)
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(loglevel), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
		return nil
	}

	app.Action = func(ctx *cli.Context) error {
		if args := ctx.Args(); len(args) > 0 {
			return fmt.Errorf("invalid command: %q", args[0])
		}

		config := newConfig(ctx)
		gpo, err := NewGasPriceOracle(config)
		if err != nil {
			return err
		}

		if err := gpo.Start(); err != nil {
			return err
		}
		gpo.Wait()

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("application failed", "message", err)
	}
}
