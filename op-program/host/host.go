package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	cl "github.com/ethereum-optimism/optimism/op-program/client"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-program/host/prefetcher"
	oppio "github.com/ethereum-optimism/optimism/op-program/io"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type L2Source struct {
	*sources.L2Client
	*sources.DebugClient
}

const opProgramChildEnvName = "OP_PROGRAM_CHILD"

func RunningProgramInClient() bool {
	value, _ := os.LookupEnv(opProgramChildEnvName)
	return value == "true"
}

// FaultProofProgram is the programmatic entry-point for the fault proof program
func FaultProofProgram(logger log.Logger, cfg *config.Config) error {
	if RunningProgramInClient() {
		cl.Main(logger)
		panic("Client main should have exited process")
	}

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

	var (
		getPreimage func(key common.Hash) ([]byte, error)
		hinter      func(hint string) error
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

	// TODO(CLI-3751: Load local preimages
	localPreimageSource := kvstore.NewLocalPreimageSource(cfg)
	splitter := kvstore.NewPreimageSourceSplitter(localPreimageSource.Get, getPreimage)

	// Setup client I/O for preimage oracle interaction
	pClientRW, pHostRW, err := oppio.CreateBidirectionalChannel()
	if err != nil {
		return fmt.Errorf("failed to create preimage pipe: %w", err)
	}
	oracleServer := preimage.NewOracleServer(pHostRW)
	launchOracleServer(logger, oracleServer, splitter.Get)
	defer pHostRW.Close()

	// Setup client I/O for hint comms
	hClientRW, hHostRW, err := oppio.CreateBidirectionalChannel()
	if err != nil {
		return fmt.Errorf("failed to create hints pipe: %w", err)
	}
	defer hHostRW.Close()
	hHost := preimage.NewHintReader(hHostRW)
	routeHints(logger, hHost, hinter)

	bootClientR, bootHostW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create boot info pipe: %w", err)
	}

	var cmd *exec.Cmd
	if cfg.Detached {
		cmd = exec.Command(os.Args[0], os.Args[1:]...)
		cmd.ExtraFiles = make([]*os.File, cl.MaxFd-3) // not including stdin, stdout and stderr
		cmd.ExtraFiles[cl.HClientRFd-3] = hClientRW.Reader()
		cmd.ExtraFiles[cl.HClientWFd-3] = hClientRW.Writer()
		cmd.ExtraFiles[cl.PClientRFd-3] = pClientRW.Reader()
		cmd.ExtraFiles[cl.PClientWFd-3] = pClientRW.Writer()
		cmd.ExtraFiles[cl.BootRFd-3] = bootClientR
		cmd.Stdout = os.Stdout // for debugging
		cmd.Stderr = os.Stderr // for debugging
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=true", opProgramChildEnvName))

		err := cmd.Start()
		if err != nil {
			return fmt.Errorf("program cmd failed to start: %w", err)
		}
	}

	bootInfo := cl.BootInfo{
		Rollup:             cfg.Rollup,
		L2ChainConfig:      cfg.L2ChainConfig,
		L1Head:             cfg.L1Head,
		L2Head:             cfg.L2Head,
		L2Claim:            cfg.L2Claim,
		L2ClaimBlockNumber: cfg.L2ClaimBlockNumber,
	}
	// Spawn a goroutine to write the boot info to avoid blocking this host's goroutine
	// if we're running in detached mode
	bootInitErrorCh := initializeBootInfoAsync(&bootInfo, bootHostW)
	if !cfg.Detached {
		return cl.RunProgram(logger, bootClientR, pClientRW, hClientRW)
	}
	if err := <-bootInitErrorCh; err != nil {
		// return early as a detached client is blocked waiting for the boot info
		return fmt.Errorf("failed to write boot info: %w", err)
	}
	if cfg.Detached {
		err := cmd.Wait()
		if err != nil {
			return fmt.Errorf("failed to wait for child program: %w", err)
		}
	}
	return nil
}

func makePrefetcher(ctx context.Context, logger log.Logger, kv kvstore.KV, cfg *config.Config) (*prefetcher.Prefetcher, error) {
	logger.Info("Connecting to L1 node", "l1", cfg.L1URL)
	l1RPC, err := client.NewRPC(ctx, logger, cfg.L1URL)
	if err != nil {
		return nil, fmt.Errorf("failed to setup L1 RPC: %w", err)
	}

	logger.Info("Connecting to L2 node", "l2", cfg.L2URL)
	l2RPC, err := client.NewRPC(ctx, logger, cfg.L2URL)
	if err != nil {
		return nil, fmt.Errorf("failed to setup L2 RPC: %w", err)
	}

	l1ClCfg := sources.L1ClientDefaultConfig(cfg.Rollup, cfg.L1TrustRPC, cfg.L1RPCKind)
	l2ClCfg := sources.L2ClientDefaultConfig(cfg.Rollup, true)
	l1Cl, err := sources.NewL1Client(l1RPC, logger, nil, l1ClCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 client: %w", err)
	}
	l2Cl, err := sources.NewL2Client(l2RPC, logger, nil, l2ClCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 client: %w", err)
	}
	l2DebugCl := &L2Source{L2Client: l2Cl, DebugClient: sources.NewDebugClient(l2RPC.CallContext)}
	return prefetcher.NewPrefetcher(logger, l1Cl, l2DebugCl, kv), nil
}

func initializeBootInfoAsync(bootInfo *cl.BootInfo, bootOracle *os.File) <-chan error {
	bootWriteErr := make(chan error, 1)
	go func() {
		bootOracleWriter := cl.NewBootstrapOracleWriter(bootOracle)
		bootWriteErr <- bootOracleWriter.WriteBootInfo(bootInfo)
		close(bootWriteErr)
	}()
	return bootWriteErr
}

func routeHints(logger log.Logger, hintReader *preimage.HintReader, hinter func(hint string) error) {
	go func() {
		for {
			if err := hintReader.NextHint(hinter); err != nil {
				if err == io.EOF || errors.Is(err, fs.ErrClosed) {
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
				if err == io.EOF || errors.Is(err, fs.ErrClosed) {
					logger.Debug("closing pre-image server")
					return
				}
				logger.Error("pre-image server error", "error", err)
				return
			}
		}
	}()
}
