package challenger

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

// Main is the entrypoint into the Challenger. This method executes the
// service and blocks until the service exits.
func Main(version string, cliCtx *cli.Context) error {
	cfg := NewConfig(cliCtx)
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid CLI flags: %w", err)
	}

	l := oplog.NewLogger(cfg.LogConfig)
	m := metrics.NewMetrics("default")
	l.Info("Initializing Challenger")

	challengerConfig, err := NewChallengerConfigFromCLIConfig(cfg, l, m)
	if err != nil {
		l.Error("Unable to create the Challenger", "error", err)
		return err
	}

	challenger, err := NewChallenger(*challengerConfig, l, m)
	if err != nil {
		l.Error("Unable to create the Challenger", "error", err)
		return err
	}

	l.Info("Starting Challenger")
	ctx, cancel := context.WithCancel(context.Background())
	if err := challenger.Start(); err != nil {
		cancel()
		l.Error("Unable to start Challenger", "error", err)
		return err
	}
	defer challenger.Stop()

	l.Info("Challenger started")
	pprofConfig := cfg.PprofConfig
	if pprofConfig.Enabled {
		l.Info("starting pprof", "addr", pprofConfig.ListenAddr, "port", pprofConfig.ListenPort)
		go func() {
			if err := oppprof.ListenAndServe(ctx, pprofConfig.ListenAddr, pprofConfig.ListenPort); err != nil {
				l.Error("error starting pprof", "err", err)
			}
		}()
	}

	metricsCfg := cfg.MetricsConfig
	if metricsCfg.Enabled {
		l.Info("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		go func() {
			if err := m.Serve(ctx, metricsCfg.ListenAddr, metricsCfg.ListenPort); err != nil {
				l.Error("error starting metrics server", err)
			}
		}()
		m.StartBalanceMetrics(ctx, l, challengerConfig.L1Client, challengerConfig.TxManager.From())
	}

	rpcCfg := cfg.RPCConfig
	server := oprpc.NewServer(rpcCfg.ListenAddr, rpcCfg.ListenPort, version, oprpc.WithLogger(l))
	if err := server.Start(); err != nil {
		cancel()
		return fmt.Errorf("error starting RPC server: %w", err)
	}

	m.RecordInfo(version)
	m.RecordUp()

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}...)
	<-interruptChannel
	cancel()

	return nil
}

// challenger contests invalid L2OutputOracle outputs
type Challenger struct {
	txMgr txmgr.TxManager
	wg    sync.WaitGroup
	done  chan struct{}

	log  log.Logger
	metr metrics.Metricer

	ctx    context.Context
	cancel context.CancelFunc

	l1Client *ethclient.Client

	rollupClient *sources.RollupClient

	// l2 Output Oracle contract
	l2ooContract     *bindings.L2OutputOracleCaller
	l2ooContractAddr common.Address
	l2ooABI          *abi.ABI

	// dispute game factory contract
	// TODO(@refcell): add a binding for this contract
	// dgfContract     *bindings.DisputeGameFactoryCaller
	dgfContractAddr common.Address
	// dgfABI          *abi.ABI

	networkTimeout time.Duration
}

// NewChallengerFromCLIConfig creates a new challenger given the CLI Config
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

	txManager, err := txmgr.NewSimpleTxManager("challenger", l, m, cfg.TxMgrConfig)
	if err != nil {
		return nil, err
	}

	// Connect to L1 and L2 providers. Perform these last since they are the most expensive.
	ctx := context.Background()
	l1Client, err := dialEthClientWithTimeout(ctx, cfg.L1EthRpc)
	if err != nil {
		return nil, err
	}

	rollupClient, err := dialRollupClientWithTimeout(ctx, cfg.RollupRpc)
	if err != nil {
		return nil, err
	}

	return &Config{
		L2OutputOracleAddr: l2ooAddress,
		DisputeGameFactory: dgfAddress,
		NetworkTimeout:     cfg.TxMgrConfig.NetworkTimeout,
		L1Client:           l1Client,
		RollupClient:       rollupClient,
		TxManager:          txManager,
	}, nil
}

// NewChallenger creates a new Challenger
func NewChallenger(cfg Config, l log.Logger, m metrics.Metricer) (*Challenger, error) {
	ctx, cancel := context.WithCancel(context.Background())

	l2ooContract, err := bindings.NewL2OutputOracleCaller(cfg.L2OutputOracleAddr, cfg.L1Client)
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

	parsed, err := bindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	return &Challenger{
		txMgr: cfg.TxManager,
		done:  make(chan struct{}),

		log:  l,
		metr: m,

		ctx:    ctx,
		cancel: cancel,

		rollupClient: cfg.RollupClient,

		l1Client: cfg.L1Client,

		l2ooContract:     l2ooContract,
		l2ooContractAddr: cfg.L2OutputOracleAddr,
		l2ooABI:          parsed,

		dgfContractAddr: cfg.DisputeGameFactory,

		networkTimeout: cfg.NetworkTimeout,
	}, nil
}

// Start runs the challenger in a goroutine.
func (c *Challenger) Start() error {
	c.log.Error("challenger not implemented.")
	return nil
}

// Stop closes the challenger and waits for spawned goroutines to exit.
func (c *Challenger) Stop() {
	c.cancel()
	close(c.done)
	c.wg.Wait()
}
