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
	signer     types.Signer
	chainID    *big.Int
	ctx        context.Context
	stop       chan struct{}
	contract   *bindings.GasPriceOracle
	privateKey *ecdsa.PrivateKey
	client     *ethclient.Client
	gasPrice   *big.Int
	gasPricer  *gasprices.L2GasPricer
}

func (g *GasPriceOracle) ensureClient() {
	t := time.NewTicker(5 * time.Second)
	for ; true; <-t.C {
		chainID, err := g.client.ChainID()
	}

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

	// ensureClient()

	// Fetch the owner of the contract to check that the
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
	go func() {
		timer := time.NewTicker(5 * time.Second)
		// There should never be an error here as long as a chain id is passed in
		opts, err := bind.NewKeyedTransactorWithChainID(g.privateKey, g.chainID)
		if err != nil {
			fmt.Println(err)
		}
		opts.Context = g.ctx

		tip, err := g.client.HeaderByNumber(g.ctx, nil)
		if err != nil {
			fmt.Println(err)
		}

		// TODO: handle types correctly - they are cast to uint64 below
		epochStartBlockNumber := float64(tip.Number.Uint64())
		averageBlockGasLimit := float64(11_000_000)

		// TODO: don't define inline
		getLatestBlockNumberFn := func() (uint64, error) {
			tip, err := g.client.HeaderByNumber(g.ctx, nil)
			if err != nil {
				return 0, err
			}
			return tip.Number.Uint64(), nil
		}

		// TODO: don't define inline
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

			fmt.Println("tx sent:", tx.Hash().Hex())
			ticker := time.NewTicker(100 * time.Millisecond)
		loop:
			for range ticker.C {
				_, err := g.client.TransactionReceipt(g.ctx, tx.Hash())
				if errors.Is(err, ethereum.NotFound) {
					continue
				}
				if err == nil {
					break loop
				}
			}
			fmt.Println("tx confirmed: ", tx.Hash().Hex())
			return nil
		}

		gasPriceUpdater := gasprices.NewGasPriceUpdater(
			g.gasPricer,
			uint64(epochStartBlockNumber),
			uint64(averageBlockGasLimit),
			getLatestBlockNumberFn,
			updateL2GasPriceFn,
		)

		for {
			select {
			case <-timer.C:
				fmt.Println("Polling")

				l2GasPrice, err := g.contract.GasPrice(&bind.CallOpts{
					Context: g.ctx,
				})
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println("got gas price:", l2GasPrice)

				if err := gasPriceUpdater.UpdateGasPrice(); err != nil {
					fmt.Println(err)
				}

			case <-g.ctx.Done():
				g.Stop()
			}
		}
	}()
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

	// TODO: parse from config
	currentPrice := float64(0)
	floorPrice := float64(0)
	getTargetGasPerSecond := func() float64 {
		return float64(0)
	}
	maxPercentChangePerEpoch := float64(0)
	gasPricer := gasprices.NewGasPricer(currentPrice, floorPrice, getTargetGasPerSecond, maxPercentChangePerEpoch)

	chainID := cfg.chainID
	if chainID == nil {
		log.Info("ChainID unset, fetching remote")
		chainID, err = client.ChainID(context.Background())
		if err != nil {
			return nil, err
		}
	}

	address := cfg.gasPriceOracleAddress
	contract, err := bindings.NewGasPriceOracle(address, client)
	if err != nil {
		return nil, err
	}

	privateKey := cfg.privateKey
	if privateKey == nil {
		return nil, errNoPrivateKey
	}

	// these error checks should go in here instead of in Start

	return &GasPriceOracle{
		signer:     types.NewEIP155Signer(chainID),
		chainID:    chainID,
		ctx:        context.Background(),
		stop:       make(chan struct{}),
		contract:   contract,
		privateKey: privateKey,
		client:     client,
		gasPrice:   cfg.gasPrice,
		gasPricer:  gasPricer,
	}, nil
}

type config struct {
	chainID               *big.Int
	ethereumHttpUrl       string
	gasPriceOracleAddress common.Address
	privateKey            *ecdsa.PrivateKey
	gasPrice              *big.Int
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

		// This shouldn't return an error ... ?
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
