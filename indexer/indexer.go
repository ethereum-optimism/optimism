package indexer

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/flags"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/indexer/processor"

	"github.com/ethereum/go-ethereum/log"

	"github.com/urfave/cli"
)

// Main is the entrypoint into the indexer service. This method returns
// a closure that executes the service and blocks until the service exits. The
// use of a closure allows the parameters bound to the top-level main package,
// e.g. GitVersion, to be captured and used once the function is executed.
func Main(gitVersion string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		log.Info("initializing indexer")
		indexer, err := NewIndexer(ctx)
		if err != nil {
			log.Error("unable to initialize indexer", "err", err)
			return err
		}

		log.Info("starting indexer")
		if err := indexer.Start(); err != nil {
			log.Error("unable to start indexer", "err", err)
		}

		defer indexer.Stop()
		log.Info("indexer started")

		// Never terminate
		<-(chan struct{})(nil)
		return nil
	}
}

// Indexer is a service that configures the necessary resources for
// running the Sync and BlockHandler sub-services.
type Indexer struct {
	db *database.DB

	l1Processor *processor.L1Processor
	l2Processor *processor.L2Processor
}

// NewIndexer initializes the Indexer, gathering any resources
// that will be needed by the TxIndexer and StateIndexer
// sub-services.
func NewIndexer(ctx *cli.Context) (*Indexer, error) {
	// TODO https://linear.app/optimism/issue/DX-55/api-implement-rest-api-with-mocked-data
	// do json format too
	// TODO https://linear.app/optimism/issue/DX-55/api-implement-rest-api-with-mocked-data

	logLevel, err := log.LvlFromString(ctx.GlobalString(flags.LogLevelFlag.Name))
	if err != nil {
		return nil, err
	}

	logHandler := log.StreamHandler(os.Stdout, log.TerminalFormat(true))
	log.Root().SetHandler(log.LvlFilterHandler(logLevel, logHandler))

	dsn := fmt.Sprintf("database=%s", ctx.GlobalString(flags.DBNameFlag.Name))
	db, err := database.NewDB(dsn)
	if err != nil {
		return nil, err
	}

	// L1 Processor
	l1EthClient, err := node.NewEthClient(ctx.GlobalString(flags.L1EthRPCFlag.Name))
	if err != nil {
		return nil, err
	}
	l1Processor, err := processor.NewL1Processor(l1EthClient, db)
	if err != nil {
		return nil, err
	}

	// L2Processor
	l2EthClient, err := node.NewEthClient(ctx.GlobalString(flags.L2EthRPCFlag.Name))
	if err != nil {
		return nil, err
	}
	l2Processor, err := processor.NewL2Processor(l2EthClient, db)
	if err != nil {
		return nil, err
	}

	indexer := &Indexer{
		db:          db,
		l1Processor: l1Processor,
		l2Processor: l2Processor,
	}

	return indexer, nil
}

// Serve spins up a REST API server at the given hostname and port.
func (b *Indexer) Serve() error {
	return nil
}

// Start starts the starts the indexing service on L1 and L2 chains and also
// starts the REST server.
func (b *Indexer) Start() error {
	go b.l1Processor.Start()
	go b.l2Processor.Start()

	return nil
}

// Stop stops the indexing service on L1 and L2 chains.
func (b *Indexer) Stop() {
}
