package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/go/gas-oracle/bindings"
	"github.com/ethereum-optimism/optimism/go/gas-oracle/flags"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli"
)

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
}

// Import the contract binding
func (g *GasPriceOracle) Start() {
	fmt.Println("starting...")

	address := crypto.PubkeyToAddress(g.privateKey.PublicKey)
	owner, err := g.contract.Owner(&bind.CallOpts{
		Context: g.ctx,
	})

	if err != nil {
		fmt.Println(err)
	}
	if address != owner {
		fmt.Println("key mismatch")
	}

	go func() {
		timer := time.NewTicker(5 * time.Second)
		opts, err := bind.NewKeyedTransactorWithChainID(g.privateKey, g.chainID)
		if err != nil {
			fmt.Println(err)
		}
		opts.Context = g.ctx

		for {
			select {
			case <-timer.C:
				fmt.Println("Polling...")

				l2GasPrice, err := g.contract.GasPrice(&bind.CallOpts{
					Context: g.ctx,
				})
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println("got gas price:", l2GasPrice)

				if g.gasPrice == nil {
					gasPrice, err := g.client.SuggestGasPrice(g.ctx)
					if err == nil {
						fmt.Println(err)
					}
					opts.GasPrice = gasPrice
				} else {
					opts.GasPrice = g.gasPrice
				}

				// import tool and use here

				tx, err := g.contract.SetGasPrice(opts, new(big.Int))
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println("set gas price:", tx.Hash().Hex())

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

func NewGasPriceOracle(cfg *config) *GasPriceOracle {
	client, err := ethclient.Dial(cfg.ethereumHttpUrl)
	if err != nil {
		fmt.Println("cannot dial")
	}

	chainID := cfg.chainID
	if chainID == nil {
		fmt.Println("chainid is nil, fetching remote")
		chainID, err = client.ChainID(context.Background())
	}

	address := cfg.gasPriceOracleAddress
	contract, err := bindings.NewGasPriceOracle(address, client)
	if err != nil {
		fmt.Println(err)
	}

	privateKey := cfg.privateKey
	if privateKey == nil {
		fmt.Println("private key cannot be nil")
	}

	return &GasPriceOracle{
		signer:     types.NewEIP155Signer(chainID),
		chainID:    chainID,
		ctx:        context.Background(),
		stop:       make(chan struct{}),
		contract:   contract,
		privateKey: privateKey,
		client:     client,
		gasPrice:   cfg.gasPrice,
	}
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
	if ctx.GlobalIsSet(flags.TransactionGasPrice.Name) {
		gasPrice := ctx.GlobalUint64(flags.TransactionGasPrice.Name)
		cfg.gasPrice = new(big.Int).SetUint64(gasPrice)
	}
	return &cfg
}

func main() {
	app := cli.NewApp()
	app.Flags = flags.Flags

	app.Action = func(ctx *cli.Context) error {
		if args := ctx.Args(); len(args) > 0 {
			return fmt.Errorf("invalid command: %q", args[0])
		}

		config := newConfig(ctx)
		gpo := NewGasPriceOracle(config)
		gpo.Start()
		gpo.Wait()

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
