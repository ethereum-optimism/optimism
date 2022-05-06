package oracle

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/gas-oracle/bindings"
	"github.com/ethereum-optimism/optimism/gas-oracle/gasprices"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// errInvalidSigningKey represents the error when the signing key used
	// is not the Owner of the contract and therefore cannot update the gasprice
	errInvalidSigningKey = errors.New("invalid signing key")
	// errNoChainID represents the error when the chain id is not provided
	// and it cannot be remotely fetched
	errNoChainID = errors.New("no chain id provided")
	// errNoPrivateKey represents the error when the private key is not provided to
	// the application
	errNoPrivateKey = errors.New("no private key provided")
	// errWrongChainID represents the error when the configured chain id is not
	// correct
	errWrongChainID = errors.New("wrong chain id provided")
	// errNoBaseFee represents the error when the base fee is not found on the
	// block. This means that the block being queried is pre eip1559
	errNoBaseFee = errors.New("base fee not found on block")
)

// GasPriceOracle manages a hot key that can update the L2 Gas Price
type GasPriceOracle struct {
	l1ChainID       *big.Int
	l2ChainID       *big.Int
	ctx             context.Context
	stop            chan struct{}
	contract        *bindings.GasPriceOracle
	l2Backend       DeployContractBackend
	l1Backend       bind.ContractTransactor
	gasPriceUpdater *gasprices.GasPriceUpdater
	config          *Config
}

// Start runs the GasPriceOracle
func (g *GasPriceOracle) Start() error {
	if g.config.l1ChainID == nil {
		return fmt.Errorf("layer-one: %w", errNoChainID)
	}
	if g.config.l2ChainID == nil {
		return fmt.Errorf("layer-two: %w", errNoChainID)
	}
	if g.config.privateKey == nil {
		return errNoPrivateKey
	}

	address := crypto.PubkeyToAddress(g.config.privateKey.PublicKey)
	log.Info("Starting Gas Price Oracle", "l1-chain-id", g.l1ChainID,
		"l2-chain-id", g.l2ChainID, "address", address.Hex())

	price, err := g.contract.GasPrice(&bind.CallOpts{
		Context: context.Background(),
	})
	if err != nil {
		return err
	}
	gasPriceGauge.Update(int64(price.Uint64()))

	if g.config.enableL1BaseFee {
		go g.BaseFeeLoop()
	}
	if g.config.enableL2GasPrice {
		go g.Loop()
	}

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
	defer timer.Stop()

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

func (g *GasPriceOracle) BaseFeeLoop() {
	timer := time.NewTicker(time.Duration(g.config.l1BaseFeeEpochLengthSeconds) * time.Second)
	defer timer.Stop()

	updateBaseFee, err := wrapUpdateBaseFee(g.l1Backend, g.l2Backend, g.config)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-timer.C:
			if err := updateBaseFee(); err != nil {
				log.Error("cannot update l1 base fee", "messgae", err)
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
	// Create the L2 client
	l2Client, err := ethclient.Dial(cfg.layerTwoHttpUrl)
	if err != nil {
		return nil, err
	}

	l1Client, err := ethclient.Dial(cfg.ethereumHttpUrl)
	if err != nil {
		return nil, err
	}

	// Ensure that we can actually connect to both backends
	log.Info("Connecting to layer two")
	if err := ensureConnection(l2Client); err != nil {
		log.Error("Unable to connect to layer two")
		return nil, err
	}
	log.Info("Connecting to layer one")
	if err := ensureConnection(l1Client); err != nil {
		log.Error("Unable to connect to layer one")
		return nil, err
	}

	address := cfg.gasPriceOracleAddress
	contract, err := bindings.NewGasPriceOracle(address, l2Client)
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

	l2ChainID, err := l2Client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	l1ChainID, err := l1Client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	if cfg.l2ChainID != nil {
		if cfg.l2ChainID.Cmp(l2ChainID) != 0 {
			return nil, fmt.Errorf("%w: L2: configured with %d and got %d",
				errWrongChainID, cfg.l2ChainID, l2ChainID)
		}
	} else {
		cfg.l2ChainID = l2ChainID
	}

	if cfg.l1ChainID != nil {
		if cfg.l1ChainID.Cmp(l1ChainID) != 0 {
			return nil, fmt.Errorf("%w: L1: configured with %d and got %d",
				errWrongChainID, cfg.l1ChainID, l1ChainID)
		}
	} else {
		cfg.l1ChainID = l1ChainID
	}

	if cfg.privateKey == nil {
		return nil, errNoPrivateKey
	}

	tip, err := l2Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	// Start at the tip
	epochStartBlockNumber := tip.Number.Uint64()
	// getLatestBlockNumberFn is used by the GasPriceUpdater
	// to get the latest block number
	getLatestBlockNumberFn := wrapGetLatestBlockNumberFn(l2Client)
	// updateL2GasPriceFn is used by the GasPriceUpdater to
	// update the gas price
	updateL2GasPriceFn, err := wrapUpdateL2GasPriceFn(l2Client, cfg)
	if err != nil {
		return nil, err
	}
	// getGasUsedByBlockFn is used by the GasPriceUpdater
	// to fetch the amount of gas that a block has used
	getGasUsedByBlockFn := wrapGetGasUsedByBlock(l2Client)

	log.Info("Creating GasPriceUpdater", "epochStartBlockNumber", epochStartBlockNumber,
		"averageBlockGasLimitPerEpoch", cfg.averageBlockGasLimitPerEpoch,
		"epochLengthSeconds", cfg.epochLengthSeconds)

	gasPriceUpdater, err := gasprices.NewGasPriceUpdater(
		gasPricer,
		epochStartBlockNumber,
		cfg.averageBlockGasLimitPerEpoch,
		cfg.epochLengthSeconds,
		getLatestBlockNumberFn,
		getGasUsedByBlockFn,
		updateL2GasPriceFn,
	)

	if err != nil {
		return nil, err
	}

	gpo := GasPriceOracle{
		l2ChainID:       l2ChainID,
		l1ChainID:       l1ChainID,
		ctx:             context.Background(),
		stop:            make(chan struct{}),
		contract:        contract,
		gasPriceUpdater: gasPriceUpdater,
		config:          cfg,
		l2Backend:       l2Client,
		l1Backend:       l1Client,
	}

	if err := gpo.ensure(); err != nil {
		return nil, err
	}

	return &gpo, nil
}

// Ensure that we can actually connect
func ensureConnection(client *ethclient.Client) error {
	t := time.NewTicker(1 * time.Second)
	retries := 0
	defer t.Stop()
	for ; true; <-t.C {
		_, err := client.ChainID(context.Background())
		if err == nil {
			break
		} else {
			retries += 1
			if retries > 90 {
				return err
			}
		}
	}
	return nil
}
