package oracle

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/go/gas-oracle/bindings"
	"github.com/ethereum-optimism/optimism/go/gas-oracle/gasprices"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// errInvalidSigningKey represents the error when the signing key used
// is not the Owner of the contract and therefore cannot update the gasprice
var errInvalidSigningKey = errors.New("invalid signing key")

// errNoChainID represents the error when the chain id is not provided
// and it cannot be remotely fetched
var errNoChainID = errors.New("no chain id provided")

// errNoPrivateKey represents the error when the private key is not provided to
// the application
var errNoPrivateKey = errors.New("no private key provided")

// errWrongChainID represents the error when the configured chain id is not
// correct
var errWrongChainID = errors.New("wrong chain id provided")

// GasPriceOracle manages a hot key that can update the L2 Gas Price
type GasPriceOracle struct {
	chainID         *big.Int
	ctx             context.Context
	stop            chan struct{}
	contract        *bindings.GasPriceOracle
	backend         DeployContractBackend
	gasPriceUpdater *gasprices.GasPriceUpdater
	config          *Config
}

// Start runs the GasPriceOracle
func (g *GasPriceOracle) Start() error {
	if g.config.chainID == nil {
		return errNoChainID
	}
	if g.config.privateKey == nil {
		return errNoPrivateKey
	}

	address := crypto.PubkeyToAddress(g.config.privateKey.PublicKey)
	log.Info("Starting Gas Price Oracle", "chain-id", g.chainID, "address", address.Hex())

	price, err := g.contract.GasPrice(&bind.CallOpts{
		Context: context.Background(),
	})
	if err != nil {
		return err
	}
	gasPriceGauge.Update(int64(price.Uint64()))

	go g.Loop()

	return nil
}

func (g *GasPriceOracle) Stop() {
	close(g.stop)
}

func (g *GasPriceOracle) Wait() {
	<-g.stop
}

// ensure makes sure that the configured private key is the owner
// of the `OVM_GasPriceOracle`. If it is not the owner, then it will
// not be able to make updates to the L2 gas price.
func (g *GasPriceOracle) ensure() error {
	owner, err := g.contract.Owner(&bind.CallOpts{
		Context: g.ctx,
	})
	if err != nil {
		return err
	}
	address := crypto.PubkeyToAddress(g.config.privateKey.PublicKey)
	if address != owner {
		log.Error("Signing key does not match contract owner", "signer", address.Hex(), "owner", owner.Hex())
		return errInvalidSigningKey
	}
	return nil
}

// Loop is the main logic of the gas-oracle
func (g *GasPriceOracle) Loop() {
	timer := time.NewTicker(time.Duration(g.config.epochLengthSeconds) * time.Second)
	for {
		select {
		case <-timer.C:
			log.Trace("polling", "time", time.Now())
			if err := g.Update(); err != nil {
				log.Error("cannot update gas price", "message", err)
			}

		case <-g.ctx.Done():
			g.Stop()
		}
	}
}

// Update will update the gas price
func (g *GasPriceOracle) Update() error {
	l2GasPrice, err := g.contract.GasPrice(&bind.CallOpts{
		Context: g.ctx,
	})
	if err != nil {
		return fmt.Errorf("cannot get gas price: %w", err)
	}

	if err := g.gasPriceUpdater.UpdateGasPrice(); err != nil {
		return fmt.Errorf("cannot update gas price: %w", err)
	}

	newGasPrice, err := g.contract.GasPrice(&bind.CallOpts{
		Context: g.ctx,
	})
	if err != nil {
		return fmt.Errorf("cannot get gas price: %w", err)
	}

	local := g.gasPriceUpdater.GetGasPrice()
	log.Info("Update", "original", l2GasPrice, "current", newGasPrice, "local", local)
	return nil
}

// NewGasPriceOracle creates a new GasPriceOracle based on a Config
func NewGasPriceOracle(cfg *Config) (*GasPriceOracle, error) {
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
		log.Error("Unable to connect to remote node", "addr", cfg.ethereumHttpUrl)
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

	// Create a gas pricer for the gas price updater
	log.Info("Creating GasPricer", "currentPrice", currentPrice,
		"floorPrice", cfg.floorPrice, "targetGasPerSecond", cfg.targetGasPerSecond,
		"maxPercentChangePerEpoch", cfg.maxPercentChangePerEpoch)

	gasPricer, err := gasprices.NewGasPricer(
		currentPrice.Uint64(),
		cfg.floorPrice,
		func() float64 {
			return float64(cfg.targetGasPerSecond)
		},
		cfg.maxPercentChangePerEpoch,
	)
	if err != nil {
		return nil, err
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	// If the chainid is passed in, exit if the chain id is
	// not correct
	if cfg.chainID != nil {
		if cfg.chainID.Cmp(chainID) != 0 {
			return nil, fmt.Errorf("%w: configured with %d and got %d", errWrongChainID, cfg.chainID, chainID)
		}
	} else {
		cfg.chainID = chainID
	}

	if cfg.privateKey == nil {
		return nil, errNoPrivateKey
	}

	tip, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	// Start at the tip
	epochStartBlockNumber := tip.Number.Uint64()
	// getLatestBlockNumberFn is used by the GasPriceUpdater
	// to get the latest block number
	getLatestBlockNumberFn := wrapGetLatestBlockNumberFn(client)
	// updateL2GasPriceFn is used by the GasPriceUpdater to
	// update the gas price
	updateL2GasPriceFn, err := wrapUpdateL2GasPriceFn(client, cfg)
	if err != nil {
		return nil, err
	}

	log.Info("Creating GasPriceUpdater", "epochStartBlockNumber", epochStartBlockNumber,
		"averageBlockGasLimitPerEpoch", cfg.averageBlockGasLimitPerEpoch,
		"epochLengthSeconds", cfg.epochLengthSeconds)

	gasPriceUpdater, err := gasprices.NewGasPriceUpdater(
		gasPricer,
		epochStartBlockNumber,
		cfg.averageBlockGasLimitPerEpoch,
		cfg.epochLengthSeconds,
		getLatestBlockNumberFn,
		updateL2GasPriceFn,
	)

	if err != nil {
		return nil, err
	}

	gpo := GasPriceOracle{
		chainID:         chainID,
		ctx:             context.Background(),
		stop:            make(chan struct{}),
		contract:        contract,
		gasPriceUpdater: gasPriceUpdater,
		config:          cfg,
		backend:         client,
	}

	if err := gpo.ensure(); err != nil {
		return nil, err
	}

	return &gpo, nil
}
