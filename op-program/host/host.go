package host

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-preimage/kvstore"
	"github.com/ethereum-optimism/optimism/op-preimage/stack"
	cl "github.com/ethereum-optimism/optimism/op-program/client"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/local"
	"github.com/ethereum-optimism/optimism/op-program/host/prefetcher"
	oppio "github.com/ethereum-optimism/optimism/op-program/io"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L2Source struct {
	*L2Client
	*sources.DebugClient
}

type HostService struct {
	logger     log.Logger
	cfg        *config.Config
	onComplete context.CancelCauseFunc

	stopper stack.Stoppable
	stopped atomic.Bool
}

func (h *HostService) Start(ctx context.Context) error {
	if h.cfg.ServerMode {
		return h.startServerMode(ctx)
	} else {
		return h.startFullMode(ctx)
	}
}

// startFullMode runs both the client and the server side of the program.
// I.e. it boths runs the state-transition and handles the preimage requests/hints of the state-transition.
func (h *HostService) startFullMode(ctx context.Context) error {
	cl, err := FaultProofProgram(h.logger, h.cfg, h.onComplete)
	if err != nil {
		return err
	}
	h.stopper = cl
	return nil
}

// startServerMode makes the HostService act like a preimage server attached to the standard file-descriptors.
// When running Cannon or another FP-VM, the op-program can be executed as child-process,
// to handle the preimage requests/hints for the FP-VM.
func (h *HostService) startServerMode(ctx context.Context) error {
	preimageChan := preimage.CreatePreimageChannel()
	hinterChan := preimage.CreateHinterChannel()

	serverCtx, serverCancel := context.WithCancel(context.Background())
	var errGrp errgroup.Group
	errGrp.Go(func() error {
		err := PreimageServer(serverCtx, h.logger, h.cfg, preimageChan, hinterChan)
		serverCancel()
		h.onComplete(err)
		return err
	})

	h.stopper = stack.StopFn(func(ctx context.Context) error {
		serverCancel()
		var result error
		// close the pipes first
		result = errors.Join(result, preimageChan.Close())
		result = errors.Join(result, hinterChan.Close())
		// wait for server to shut down
		result = errors.Join(result, errGrp.Wait())
		return result
	})
	return nil
}

func (h *HostService) Stop(ctx context.Context) error {
	defer h.stopped.Store(true)
	return h.stopper.Stop(ctx)
}

func (h *HostService) Stopped() bool {
	return h.stopped.Load()
}

var _ cliapp.Lifecycle = (*HostService)(nil)

func Main(logger log.Logger, cfg *config.Config, onComplete context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	if err := cfg.Check(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	cfg.Rollup.LogDescription(logger, chaincfg.L2ChainIDToNetworkDisplayName)

	return &HostService{
		logger:     logger,
		cfg:        cfg,
		onComplete: onComplete,
	}, nil
}

// FaultProofProgram is the programmatic entry-point for the fault proof program.
// It runs both the host and client. The client may be in a sub-process.
func FaultProofProgram(logger log.Logger, cfg *config.Config, onComplete context.CancelCauseFunc) (stack.Stoppable, error) {
	// The driver.ErrClaimNotValid error is propagated through the onComplete(err) call,
	// if the program completes before interruption.

	pClientRW, pHostRW, hClientRW, hHostRW, stopPipes, err := stack.MiddlewarePipes()
	if err != nil {
		return nil, err
	}

	serverCtx, serverCancel := context.WithCancel(context.Background())

	// Start the server. The server will stop when its communication channels stop.
	var errGrp errgroup.Group
	errGrp.Go(func() error {
		err := PreimageServer(serverCtx, logger, cfg, pHostRW, hHostRW)
		if err != nil {
			err = fmt.Errorf("preimage server failed: %w", err)
		}
		onComplete(err)
		return err
	})

	// Create the client (sub-process or in-process)
	sink := CreateClient(logger, cfg, onComplete)

	// Start the client. The client will stop when we signal it to stop.
	stopClient, err := sink(pClientRW, hClientRW)
	if err != nil {
		err = fmt.Errorf("failed to start client: %w", err)
		serverCancel()
		if closeErr := stopPipes.Stop(serverCtx); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close client-server communication: %w", closeErr))
		}
		if closeErr := errGrp.Wait(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close server upon client error: %w", closeErr))
		}
		return nil, err
	}

	// Upon closing the client we stop the communication channels to the server.
	// This will cause the server to stop. Which we can then just await.

	stopper := stack.StopFn(func(ctx context.Context) error {
		serverCancel() // don't try to finish any fetching on the server-side
		var result error
		result = errors.Join(result, stopClient.Stop(ctx))
		result = errors.Join(result, stopPipes.Stop(ctx))
		result = errors.Join(result, errGrp.Wait())
		return result
	})

	return stopper, nil
}

func CreateClient(logger log.Logger, cfg *config.Config, onComplete context.CancelCauseFunc) stack.Sink {
	if cfg.ExecCmd != "" {
		return stack.ExecSink(cfg.ExecCmd, os.Stdout, os.Stderr, onComplete)
	} else {
		return func(preimageRW oppio.FileChannel, hintRW oppio.FileChannel) (stack.Stoppable, error) {
			// context to signal to the in-process client when we want it stop
			ctx, cancel := context.WithCancel(context.Background())

			// track client-side completion with a wait-group
			var wg sync.WaitGroup
			wg.Add(1)

			// Start running the program, and signal to the caller when it ends.
			go func() {
				defer func() { // client is expected to panic when the server closes before the client can get its preimage data.
					if r := recover(); r != nil {
						err := fmt.Errorf("client panic: %v", r)
						onComplete(err)
					}
					wg.Done()
				}()
				err := cl.RunProgram(ctx, logger, preimageRW, hintRW)
				onComplete(err) // signal any client-side error as completion result
			}()

			return stack.StopFn(func(_ context.Context) error {
				// If user asks the in-process client to stop, and we terminate the stop-context,
				// we can't kill the client, since it's in-process. So unfortunately we just have to wait.
				cancel()   // signal in-process client program to stop
				wg.Wait()  // wait for it to stop
				return nil // there is no case where we fail to stop the in-process program.
			}), nil
		}
	}
}

// PreimageServer reads hints and preimage requests from the provided channels and processes those requests.
// This method will block until both the hinter and preimage handlers complete.
// If either returns an error both handlers are stopped.
// The supplied preimageChannel and hintChannel will be closed before this function returns.
func PreimageServer(ctx context.Context, logger log.Logger, cfg *config.Config, preimageChannel oppio.FileChannel, hintChannel oppio.FileChannel) error {
	defer func() {
		_ = preimageChannel.Close()
		_ = hintChannel.Close()
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

	localPreimageSource := local.NewLocalPreimageSource(cfg)
	splitter := kvstore.NewPreimageSourceSplitter(localPreimageSource.Get, getPreimage)
	preimageGetter := preimage.WithVerification(splitter.Get)

	var errGrp errgroup.Group
	errGrp.Go(func() error {
		err := stack.HandlePreimages(logger, preimageChannel, preimageGetter)
		_ = hintChannel.Close() // stop the other err-group member
		return err
	})
	errGrp.Go(func() error {
		err := stack.HandleHints(logger, hintChannel, hinter)
		_ = preimageChannel.Close() // stop the other err-group member
		return err
	})
	return errGrp.Wait()
}

func makePrefetcher(ctx context.Context, logger log.Logger, kv kvstore.KV, cfg *config.Config) (*prefetcher.Prefetcher, error) {
	logger.Info("Connecting to L1 node", "l1", cfg.L1URL)
	l1RPC, err := client.NewRPC(ctx, logger, cfg.L1URL, client.WithDialBackoff(10))
	if err != nil {
		return nil, fmt.Errorf("failed to setup L1 RPC: %w", err)
	}

	logger.Info("Connecting to L2 node", "l2", cfg.L2URL)
	l2RPC, err := client.NewRPC(ctx, logger, cfg.L2URL, client.WithDialBackoff(10))
	if err != nil {
		return nil, fmt.Errorf("failed to setup L2 RPC: %w", err)
	}

	l1ClCfg := sources.L1ClientDefaultConfig(cfg.Rollup, cfg.L1TrustRPC, cfg.L1RPCKind)
	l2ClCfg := sources.L2ClientDefaultConfig(cfg.Rollup, true)
	l1Cl, err := sources.NewL1Client(l1RPC, logger, nil, l1ClCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 client: %w", err)
	}
	l2Cl, err := NewL2Client(l2RPC, logger, nil, &L2ClientConfig{L2ClientConfig: l2ClCfg, L2Head: cfg.L2Head})
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 client: %w", err)
	}
	l2DebugCl := &L2Source{L2Client: l2Cl, DebugClient: sources.NewDebugClient(l2RPC.CallContext)}
	return prefetcher.NewPrefetcher(logger, l1Cl, l2DebugCl, kv), nil
}
