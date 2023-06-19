package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	cl "github.com/ethereum-optimism/optimism/op-program/client"
	"github.com/ethereum-optimism/optimism/op-program/client/driver"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/flags"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-program/host/prefetcher"
	oppio "github.com/ethereum-optimism/optimism/op-program/io"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const maxRPCRetries = math.MaxInt

type L2Source struct {
	*sources.L2Client
	*sources.DebugClient
}

func Main(logger log.Logger, cfg *config.Config) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, logger)
	cfg.Rollup.LogDescription(logger, chaincfg.L2ChainIDToNetworkName)

	ctx := context.Background()
	if cfg.ServerMode {
		preimageChan := cl.CreatePreimageChannel()
		hinterChan := cl.CreateHinterChannel()
		return PreimageServer(ctx, logger, cfg, preimageChan, hinterChan)
	}

	if err := FaultProofProgram(ctx, logger, cfg); errors.Is(err, driver.ErrClaimNotValid) {
		log.Crit("Claim is invalid", "err", err)
	} else if err != nil {
		return err
	} else {
		log.Info("Claim successfully verified")
	}
	return nil
}

// FaultProofProgram is the programmatic entry-point for the fault proof program
func FaultProofProgram(ctx context.Context, logger log.Logger, cfg *config.Config) error {
	var (
		serverErr chan error
		pClientRW oppio.FileChannel
		hClientRW oppio.FileChannel
	)
	defer func() {
		if pClientRW != nil {
			_ = pClientRW.Close()
		}
		if hClientRW != nil {
			_ = hClientRW.Close()
		}
		if serverErr != nil {
			err := <-serverErr
			if err != nil {
				logger.Error("preimage server failed", "err", err)
			}
			logger.Debug("Preimage server stopped")
		}
	}()
	// Setup client I/O for preimage oracle interaction
	pClientRW, pHostRW, err := oppio.CreateBidirectionalChannel()
	if err != nil {
		return fmt.Errorf("failed to create preimage pipe: %w", err)
	}

	// Setup client I/O for hint comms
	hClientRW, hHostRW, err := oppio.CreateBidirectionalChannel()
	if err != nil {
		return fmt.Errorf("failed to create hints pipe: %w", err)
	}

	// Use a channel to receive the server result so we can wait for it to complete before returning
	serverErr = make(chan error)
	go func() {
		defer close(serverErr)
		serverErr <- PreimageServer(ctx, logger, cfg, pHostRW, hHostRW)
	}()

	var cmd *exec.Cmd
	if cfg.ExecCmd != "" {
		cmd = exec.CommandContext(ctx, cfg.ExecCmd)
		cmd.ExtraFiles = make([]*os.File, cl.MaxFd-3) // not including stdin, stdout and stderr
		cmd.ExtraFiles[cl.HClientRFd-3] = hClientRW.Reader()
		cmd.ExtraFiles[cl.HClientWFd-3] = hClientRW.Writer()
		cmd.ExtraFiles[cl.PClientRFd-3] = pClientRW.Reader()
		cmd.ExtraFiles[cl.PClientWFd-3] = pClientRW.Writer()
		cmd.Stdout = os.Stdout // for debugging
		cmd.Stderr = os.Stderr // for debugging

		err := cmd.Start()
		if err != nil {
			return fmt.Errorf("program cmd failed to start: %w", err)
		}
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("failed to wait for child program: %w", err)
		}
		logger.Debug("Client program completed successfully")
		return nil
	} else {
		return cl.RunProgram(logger, pClientRW, hClientRW)
	}
}

// PreimageServer reads hints and preimage requests from the provided channels and processes those requests.
// This method will block until both the hinter and preimage handlers complete.
// If either returns an error both handlers are stopped.
// The supplied preimageChannel and hintChannel will be closed before this function returns.
func PreimageServer(ctx context.Context, logger log.Logger, cfg *config.Config, preimageChannel oppio.FileChannel, hintChannel oppio.FileChannel) error {
	var serverDone chan error
	var hinterDone chan error
	defer func() {
		preimageChannel.Close()
		hintChannel.Close()
		if serverDone != nil {
			// Wait for pre-image server to complete
			<-serverDone
		}
		if hinterDone != nil {
			// Wait for hinter to complete
			<-hinterDone
		}
	}()
	logger.Info("Starting preimage server")
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

	var (
		getPreimage kvstore.PreimageSource
		hinter      preimage.HintHandler
	)
	if cfg.FetchingEnabled() {
		prefetch, err := makePrefetcher(ctx, logger, kv, cfg)
		if err != nil {
			return fmt.Errorf("failed to create prefetcher: %w", err)
		}
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
	preimageGetter := splitter.Get

	serverDone = launchOracleServer(logger, preimageChannel, preimageGetter)
	hinterDone = routeHints(logger, hintChannel, hinter)
	select {
	case err := <-serverDone:
		return err
	case err := <-hinterDone:
		return err
	}
}

func makePrefetcher(ctx context.Context, logger log.Logger, kv kvstore.KV, cfg *config.Config) (*prefetcher.Prefetcher, error) {
	logger.Info("Connecting to L1 node", "l1", cfg.L1URL)
	l1RPC, err := createRetryingRPC(ctx, logger, cfg.L1URL)
	if err != nil {
		return nil, fmt.Errorf("failed to setup L1 RPC: %w", err)
	}

	logger.Info("Connecting to L2 node", "l2", cfg.L2URL)
	l2RPC, err := createRetryingRPC(ctx, logger, cfg.L2URL)
	if err != nil {
		return nil, fmt.Errorf("failed to setup L2 RPC: %w", err)
	}

	l1ClCfg := sources.L1ClientDefaultConfig(cfg.Rollup, cfg.L1TrustRPC, cfg.L1RPCKind)
	l1Cl, err := sources.NewL1Client(l1RPC, logger, nil, l1ClCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 client: %w", err)
	}

	l2ClCfg := sources.L2ClientDefaultConfig(cfg.Rollup, true)
	l2Cl, err := sources.NewL2Client(l2RPC, logger, nil, l2ClCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 client: %w", err)
	}

	l2DebugCl := &L2Source{L2Client: l2Cl, DebugClient: sources.NewDebugClient(l2RPC.CallContext)}
	return prefetcher.NewPrefetcher(logger, l1Cl, l2DebugCl, kv), nil
}

func createRetryingRPC(ctx context.Context, logger log.Logger, url string) (client.RPC, error) {
	rpc, err := client.NewRPC(ctx, logger, url)
	if err != nil {
		return nil, err
	}
	return opclient.NewRetryingClient(rpc, maxRPCRetries), nil
}

func routeHints(logger log.Logger, hHostRW io.ReadWriter, hinter preimage.HintHandler) chan error {
	chErr := make(chan error)
	hintReader := preimage.NewHintReader(hHostRW)
	go func() {
		defer close(chErr)
		for {
			if err := hintReader.NextHint(hinter); err != nil {
				if err == io.EOF || errors.Is(err, fs.ErrClosed) {
					logger.Debug("closing pre-image hint handler")
					return
				}
				logger.Error("pre-image hint router error", "err", err)
				chErr <- err
				return
			}
		}
	}()
	return chErr
}

func launchOracleServer(logger log.Logger, pHostRW io.ReadWriteCloser, getter preimage.PreimageGetter) chan error {
	chErr := make(chan error)
	server := preimage.NewOracleServer(pHostRW)
	go func() {
		defer close(chErr)
		for {
			if err := server.NextPreimageRequest(getter); err != nil {
				if err == io.EOF || errors.Is(err, fs.ErrClosed) {
					logger.Debug("closing pre-image server")
					return
				}
				logger.Error("pre-image server error", "error", err)
				chErr <- err
				return
			}
		}
	}()
	return chErr
}
