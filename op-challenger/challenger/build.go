package challenger

import (
	"context"
	_ "net/http/pprof"

	bind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/ethereum/go-ethereum/log"

	bindings "github.com/refcell/op-challenger/contracts/bindings"
	flags "github.com/refcell/op-challenger/flags"
	metrics "github.com/refcell/op-challenger/metrics"

	opBindings "github.com/ethereum-optimism/optimism/op-bindings/bindings"
	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
)

// NewChallengerFromCLIConfig creates a new Challenger given the CLI Config
func NewChallengerFromCLIConfig(cfg CLIConfig, l log.Logger, m metrics.Metricer) (*Challenger, error) {
	challengerConfig, err := NewChallengerConfigFromCLIConfig(cfg, l, m)
	if err != nil {
		return nil, err
	}
	return NewChallenger(*challengerConfig, l, m)
}

// NewChallengerConfigFromCLIConfig creates the challenger config from the CLI config.
func NewChallengerConfigFromCLIConfig(cfg CLIConfig, l log.Logger, m metrics.Metricer) (*Config, error) {
	l2ooAddress, err := parseAddress(cfg.L2OOAddress)
	if err != nil {
		return nil, err
	}

	dgfAddress, err := parseAddress(cfg.DGFAddress)
	if err != nil {
		return nil, err
	}

	// Connect to L1 and L2 providers. Perform these last since they are the most expensive.
	ctx := context.Background()
	l1Client, err := dialEthClientWithTimeout(ctx, cfg.L1EthRpc)
	if err != nil {
		return nil, err
	}

	txManagerConfig, err := flags.NewTxManagerConfig(cfg.TxMgrConfig, l)
	if err != nil {
		return nil, err
	}
	txManager := txmgr.NewSimpleTxManager("challenger", l, txManagerConfig, l1Client)

	rollupClient, err := dialRollupClientWithTimeout(ctx, cfg.RollupRpc)
	if err != nil {
		return nil, err
	}

	return &Config{
		L2OutputOracleAddr: l2ooAddress,
		DisputeGameFactory: dgfAddress,
		NetworkTimeout:     txManagerConfig.NetworkTimeout,
		L1Client:           l1Client,
		RollupClient:       rollupClient,
		TxManager:          txManager,
		From:               txManagerConfig.From,
		privateKey:         cfg.PrivateKey,
	}, nil
}

// NewChallenger creates a new Challenger
func NewChallenger(cfg Config, l log.Logger, m metrics.Metricer) (*Challenger, error) {
	ctx, cancel := context.WithCancel(context.Background())

	l2ooContract, err := opBindings.NewL2OutputOracleCaller(cfg.L2OutputOracleAddr, cfg.L1Client)
	if err != nil {
		cancel()
		return nil, err
	}

	cCtx, cCancel := context.WithTimeout(ctx, cfg.NetworkTimeout)
	defer cCancel()
	version, err := l2ooContract.Version(&bind.CallOpts{Context: cCtx})
	if err != nil {
		cancel()
		return nil, err
	}
	log.Info("Connected to L2OutputOracle", "address", cfg.L2OutputOracleAddr, "version", version)

	parsed, err := opBindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	dgfContract, err := bindings.NewMockDisputeGameFactoryCaller(cfg.DisputeGameFactory, cfg.L1Client)
	if err != nil {
		cancel()
		return nil, err
	}

	dgfAbi, err := bindings.MockDisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	adgAbi, err := bindings.MockAttestationDisputeGameMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	privateKey, err := crypto.HexToECDSA(cfg.privateKey)
	if err != nil {
		cancel()
		return nil, err
	}

	return &Challenger{
		txMgr:      cfg.TxManager,
		done:       make(chan struct{}),
		log:        l,
		ctx:        ctx,
		cancel:     cancel,
		metr:       m,
		privateKey: privateKey,

		from: cfg.From,

		l1Client: cfg.L1Client,

		rollupClient: cfg.RollupClient,

		l2ooContract:     l2ooContract,
		l2ooContractAddr: cfg.L2OutputOracleAddr,
		l2ooABI:          parsed,

		dgfContract:     dgfContract,
		dgfContractAddr: cfg.DisputeGameFactory,
		dgfABI:          dgfAbi,

		adgABI: adgAbi,

		networkTimeout: cfg.NetworkTimeout,
	}, nil
}
