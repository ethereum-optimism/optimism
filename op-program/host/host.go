package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	cl "github.com/ethereum-optimism/optimism/op-program/client"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-program/host/prefetcher"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type L2Source struct {
	*sources.L2Client
	*sources.DebugClient
}

// FaultProofProgram is the programmatic entry-point for the fault proof program
func FaultProofProgram(logger log.Logger, cfg *config.Config) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	cfg.Rollup.LogDescription(logger, chaincfg.L2ChainIDToNetworkName)

	ctx := context.Background()
	var kv kvstore.KV
	if cfg.DataDir == "" {
		logger.Info("Using in-memory storage")
		kv = kvstore.NewMemKV()
	} else {
		logger.Info("Creating disk storage", "datadir", cfg.DataDir)
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			return fmt.Errorf("creating datadir: %w", err)
		}
		kv = kvstore.NewDiskKV(cfg.DataDir)
	}

	var getPreimage func(key common.Hash) ([]byte, error)
	var hinter func(hint string) error
	if cfg.FetchingEnabled() {
		logger.Info("Connecting to L1 node", "l1", cfg.L1URL)
		l1RPC, err := client.NewRPC(ctx, logger, cfg.L1URL)
		if err != nil {
			return fmt.Errorf("failed to setup L1 RPC: %w", err)
		}

		logger.Info("Connecting to L2 node", "l2", cfg.L2URL)
		l2RPC, err := client.NewRPC(ctx, logger, cfg.L2URL)
		if err != nil {
			return fmt.Errorf("failed to setup L2 RPC: %w", err)
		}

		l1ClCfg := sources.L1ClientDefaultConfig(cfg.Rollup, cfg.L1TrustRPC, cfg.L1RPCKind)
		l2ClCfg := sources.L2ClientDefaultConfig(cfg.Rollup, true)
		l1Cl, err := sources.NewL1Client(l1RPC, logger, nil, l1ClCfg)
		if err != nil {
			return fmt.Errorf("failed to create L1 client: %w", err)
		}
		l2Cl, err := sources.NewL2Client(l2RPC, logger, nil, l2ClCfg)
		if err != nil {
			return fmt.Errorf("failed to create L2 client: %w", err)
		}
		l2DebugCl := &L2Source{L2Client: l2Cl, DebugClient: sources.NewDebugClient(l2RPC.CallContext)}

		logger.Info("Setting up pre-fetcher")
		prefetch := prefetcher.NewPrefetcher(logger, l1Cl, l2DebugCl, kv)
		getPreimage = func(key common.Hash) ([]byte, error) { return prefetch.GetPreimage(ctx, key) }
		hinter = prefetch.Hint
	} else {
		logger.Info("Using offline mode. All required pre-images must be pre-populated.")
		getPreimage = kv.Get
		hinter = func(hint string) error {
			logger.Debug("ignoring prefetch hint", "hint", hint)
			return nil
		}
	}

	localPreimageSource := kvstore.NewLocalPreimageSource(cfg)
	splitter := kvstore.NewPreimageSourceSplitter(localPreimageSource.Get, getPreimage)

	// Setup pipe for preimage oracle interaction
	pClientRW, pHostRW := bidirectionalPipe()
	oracleServer := preimage.NewOracleServer(pHostRW)
	// Setup pipe for hint comms
	hClientRW, hHostRW := bidirectionalPipe()
	hHost := preimage.NewHintReader(hHostRW)
	defer pHostRW.Close()
	defer hHostRW.Close()
	routeHints(logger, hHost, hinter)
	launchOracleServer(logger, oracleServer, splitter.Get)

	return cl.ClientProgram(
		logger,
		cfg.Rollup,
		cfg.L2ChainConfig,
		cfg.L1Head,
		cfg.L2Head,
		cfg.L2Claim,
		cfg.L2ClaimBlockNumber,
		pClientRW,
		hClientRW,
	)
}

type readWritePair struct {
	io.ReadCloser
	io.WriteCloser
}

func (rw *readWritePair) Close() error {
	if err := rw.ReadCloser.Close(); err != nil {
		return err
	}
	return rw.WriteCloser.Close()
}

func bidirectionalPipe() (a, b io.ReadWriteCloser) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	return &readWritePair{ReadCloser: ar, WriteCloser: aw}, &readWritePair{ReadCloser: br, WriteCloser: bw}
}

func routeHints(logger log.Logger, hintReader *preimage.HintReader, hinter func(hint string) error) {
	go func() {
		for {
			if err := hintReader.NextHint(hinter); err != nil {
				if err == io.EOF || errors.Is(err, io.ErrClosedPipe) {
					logger.Debug("closing pre-image hint handler")
					return
				}
				logger.Error("pre-image hint router error", "err", err)
				return
			}
		}
	}()
}

func launchOracleServer(logger log.Logger, server *preimage.OracleServer, getter func(key common.Hash) ([]byte, error)) {
	go func() {
		for {
			if err := server.NextPreimageRequest(getter); err != nil {
				if err == io.EOF || errors.Is(err, io.ErrClosedPipe) {
					logger.Debug("closing pre-image server")
					return
				}
				logger.Error("pre-image server error", "error", err)
				return
			}
		}
	}()
}
