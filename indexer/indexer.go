package indexer

import (
	"context"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli"
)

const (
	// defaultDialTimeout is default duration the service will wait on
	// startup to make a connection to either the L1 or L2 backends.
	defaultDialTimeout = 5 * time.Second
)

// Main is the entrypoint into the indexer service. This method returns
// a closure that executes the service and blocks until the service exits. The
// use of a closure allows the parameters bound to the top-level main package,
// e.g. GitVersion, to be captured and used once the function is executed.
func Main(gitVersion string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		log.Info("Initializing indexer")
		return nil
	}
}

// Indexer is a service that configures the necessary resources for
// running the Sync and BlockHandler sub-services.
type Indexer struct {
	l1Client *ethclient.Client
	l2Client *ethclient.Client
}

// NewIndexer initializes the Indexer, gathering any resources
// that will be needed by the TxIndexer and StateIndexer
// sub-services.
func NewIndexer() (*Indexer, error) {
	ctx := context.Background()

	var logHandler log.Handler = log.StreamHandler(os.Stdout, log.TerminalFormat(true))
	// TODO https://linear.app/optimism/issue/DX-55/api-implement-rest-api-with-mocked-data
	// do json format too
	// TODO https://linear.app/optimism/issue/DX-55/api-implement-rest-api-with-mocked-data
	// pass in loglevel from config
	// logHandler = log.StreamHandler(os.Stdout, log.JSONFormat())
	logLevel, err := log.LvlFromString("info")
	if err != nil {
		return nil, err
	}

	log.Root().SetHandler(log.LvlFilterHandler(logLevel, logHandler))

	// Connect to L1 and L2 providers. Perform these last since they are the
	// most expensive.
	// TODO https://linear.app/optimism/issue/DX-55/api-implement-rest-api-with-mocked-data
	// pass in rpc url from config
	l1Client, _, err := dialEthClientWithTimeout(ctx, "http://localhost:8545")
	if err != nil {
		return nil, err
	}

	// TODO https://linear.app/optimism/issue/DX-55/api-implement-rest-api-with-mocked-data
	// pass in rpc url from config
	l2Client, _, err := dialEthClientWithTimeout(ctx, "http://localhost:9545")
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return &Indexer{
		l1Client: l1Client,
		l2Client: l2Client,
	}, nil
}

// Serve spins up a REST API server at the given hostname and port.
func (b *Indexer) Serve() error {
	return nil
}

// Start starts the starts the indexing service on L1 and L2 chains and also
// starts the REST server.
func (b *Indexer) Start() error {
	return nil
}

// Stop stops the indexing service on L1 and L2 chains.
func (b *Indexer) Stop() {
}

// dialL1EthClientWithTimeout attempts to dial the L1 provider using the
// provided URL. If the dial doesn't complete within defaultDialTimeout seconds,
// this method will return an error.
func dialEthClientWithTimeout(ctx context.Context, url string) (
	*ethclient.Client, *rpc.Client, error) {

	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	c, err := rpc.DialContext(ctxt, url)
	if err != nil {
		return nil, nil, err
	}
	return ethclient.NewClient(c), c, nil
}
