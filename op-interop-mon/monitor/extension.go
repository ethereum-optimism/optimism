package monitor

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// Extension is an additional check that runs inside the monitor service. The service builds it
// once its clients are ready, starts it before the finders and stops it after them.
type Extension interface {
	Start(ctx context.Context) error
	Stop() error
}

// BlockObserver is an Extension that also sees the receipts of every block the finders process.
// ObserveBlock is called from the finders' goroutines, one per chain, so it must not block.
type BlockObserver interface {
	ObserveBlock(chainID eth.ChainID, receipts []*types.Receipt)
}

// ExtensionEnv is what an Extension is built from.
type ExtensionEnv struct {
	Log     log.Logger
	Clients map[eth.ChainID]*sources.EthClient
	// MessageExpiryWindow is the dependency set's message expiry window, in seconds.
	MessageExpiryWindow uint64
	// Registry is the registry the metrics server serves, or nil when metrics are disabled.
	Registry *prometheus.Registry
}

// ExtensionFactory builds an Extension.
type ExtensionFactory func(ctx context.Context, env ExtensionEnv) (Extension, error)

// Option configures the monitor service.
type Option func(*options)

type options struct {
	extensions []ExtensionFactory
	extraFlags []cli.Flag
}

// WithExtension adds an Extension to the service.
func WithExtension(factory ExtensionFactory) Option {
	return func(o *options) { o.extensions = append(o.extensions, factory) }
}

// WithExtraFlags declares the flags an extension's binary adds, so that Main does not warn about
// their environment variables.
func WithExtraFlags(extra []cli.Flag) Option {
	return func(o *options) { o.extraFlags = append(o.extraFlags, extra...) }
}

func applyOptions(opts []Option) options {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// observingJobFilter wraps a JobFilter so that each observer sees every block's receipts before
// the filter turns them into jobs.
func observingJobFilter(filter JobFilter, observers []BlockObserver) JobFilter {
	if len(observers) == 0 {
		return filter
	}
	return func(receipts []*types.Receipt, executingChain eth.ChainID) []*Job {
		for _, o := range observers {
			o.ObserveBlock(executingChain, receipts)
		}
		return filter(receipts, executingChain)
	}
}
