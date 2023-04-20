package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	cldr "github.com/ethereum-optimism/optimism/op-program/client/driver"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-program/host/l1"
	"github.com/ethereum-optimism/optimism/op-program/host/l2"
	"github.com/ethereum-optimism/optimism/op-program/host/prefetcher"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrClaimNotValid = errors.New("invalid claim")
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

	var preimageOracle preimage.OracleFn
	var hinter preimage.HinterFn
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
		preimageOracle = asOracleFn(func(key common.Hash) ([]byte, error) {
			return prefetch.GetPreimage(ctx, key)
		})
		hinter = asHinter(prefetch.Hint)
	} else {
		logger.Info("Using offline mode. All required pre-images must be pre-populated.")
		preimageOracle = asOracleFn(kv.Get)
		hinter = func(v preimage.Hint) {
			logger.Debug("ignoring prefetch hint", "hint", v)
		}
	}
	l1Source := l1.NewSource(logger, preimageOracle, hinter, cfg.L1Head)

	l2Source, err := l2.NewEngine(logger, preimageOracle, hinter, cfg)
	if err != nil {
		return fmt.Errorf("connect l2 oracle: %w", err)
	}

	logger.Info("Starting derivation")
	d := cldr.NewDriver(logger, cfg.Rollup, l1Source, l2Source, cfg.L2ClaimBlockNumber)
	for {
		if err = d.Step(ctx); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}
	}
	if !d.ValidateClaim(eth.Bytes32(cfg.L2Claim)) {
		return ErrClaimNotValid
	}
	return nil
}

func asOracleFn(getter func(key common.Hash) ([]byte, error)) preimage.OracleFn {
	return func(key preimage.Key) []byte {
		pre, err := getter(key.PreimageKey())
		if err != nil {
			panic(fmt.Errorf("preimage unavailable for key %v: %w", key, err))
		}
		return pre
	}
}

func asHinter(hint func(hint string) error) preimage.HinterFn {
	return func(v preimage.Hint) {
		err := hint(v.Hint())
		if err != nil {
			panic(fmt.Errorf("hint rejected %v: %w", v, err))
		}
	}
}
