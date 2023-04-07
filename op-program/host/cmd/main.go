package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-program/client"
	cldr "github.com/ethereum-optimism/optimism/op-program/client/driver"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/flags"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	hl1 "github.com/ethereum-optimism/optimism/op-program/host/l1"
	hl2 "github.com/ethereum-optimism/optimism/op-program/host/l2"
	"github.com/ethereum-optimism/optimism/op-program/host/version"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

var (
	GitCommit = ""
	GitDate   = ""
)

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = func() string {
	v := version.Version
	if GitCommit != "" {
		v += "-" + GitCommit[:8]
	}
	if GitDate != "" {
		v += "-" + GitDate
	}
	if version.Meta != "" {
		v += "-" + version.Meta
	}
	return v
}()

func main() {
	args := os.Args
	err := run(args, FaultProofProgram)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

type ConfigAction func(log log.Logger, config *config.Config) error

// run parses the supplied args to create a config.Config instance, sets up logging
// then calls the supplied ConfigAction.
// This allows testing the translation from CLI arguments to Config
func run(args []string, action ConfigAction) error {
	// Set up logger with a default INFO level in case we fail to parse flags,
	// otherwise the final critical log won't show what the parsing error was.
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Version = VersionWithMeta
	app.Flags = flags.Flags
	app.Name = "op-program"
	app.Usage = "Optimism Fault Proof Program"
	app.Description = "The Optimism Fault Proof Program fault proof program that runs through the rollup state-transition to verify an L2 output from L1 inputs."
	app.Action = func(ctx *cli.Context) error {
		logger, err := setupLogging(ctx)
		if err != nil {
			return err
		}
		logger.Info("Starting fault proof program", "version", VersionWithMeta)

		cfg, err := config.NewConfigFromCLI(ctx)
		if err != nil {
			return err
		}
		return action(logger, cfg)
	}

	return app.Run(args)
}

func setupLogging(ctx *cli.Context) (log.Logger, error) {
	logCfg := oplog.ReadCLIConfig(ctx)
	if err := logCfg.Check(); err != nil {
		return nil, fmt.Errorf("log config error: %w", err)
	}
	logger := oplog.NewLogger(logCfg)
	return logger, nil
}

// FaultProofProgram is the programmatic entry-point for the fault proof program
func FaultProofProgram(logger log.Logger, cfg *config.Config) error {
	cfg.Rollup.LogDescription(logger, chaincfg.L2ChainIDToNetworkName)
	if !cfg.FetchingEnabled() {
		return errors.New("offline mode not supported")
	}

	ctx := context.Background()
	logger.Info("Connecting to L1 node", "l1", cfg.L1URL)
	l1Source, err := hl1.NewFetchingL1(ctx, logger, cfg)
	if err != nil {
		return fmt.Errorf("connect l1 oracle: %w", err)
	}

	l2Cfg, err := loadL2Genesis(cfg)
	if err != nil {
		return fmt.Errorf("failed to load L2 genesis: %w", err)
	}
	logger.Info("Connecting to L2 node", "l2", cfg.L2URL)
	l2Oracle, err := hl2.NewFetchingL2Oracle(ctx, logger, cfg.L2URL)
	if err != nil {
		return fmt.Errorf("connect l2 oracle: %w", err)
	}
	engineBackend, err := l2.NewOracleBackedL2Chain(logger, l2Oracle, l2Cfg, cfg.L2Head)
	if err != nil {
		return fmt.Errorf("failed to create oracle-backed L2 chain: %w", err)
	}
	l2Source := l2.NewOracleEngine(cfg.Rollup, logger, engineBackend)

	d := cldr.NewDriver(logger, cfg.Rollup, l1Source, l2Source)
	for {
		if err = d.Step(ctx); errors.Is(err, io.EOF) {
			break
		} else if cfg.FetchingEnabled() && errors.Is(err, derive.ErrTemporary) {
			// When in fetching mode, recover from temporary errors to allow us to keep fetching data
			// TODO(CLI-3780) Ideally the retry would happen in the fetcher so this is not needed
			logger.Warn("Temporary error in pipeline", "err", err)
			time.Sleep(5 * time.Second)
		} else if err != nil {
			return err
		}
	}
	logger.Info("Derivation complete", "head", d.SafeHead())
	return nil
}

type readWritePair struct {
	io.Reader
	io.Writer
}

func bidrectionalPipe() (a, b io.ReadWriter) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	return readWritePair{Reader: ar, Writer: aw}, readWritePair{Reader: br, Writer: bw}
}

func loadL2Genesis(cfg *config.Config) (*params.ChainConfig, error) {
	data, err := os.ReadFile(cfg.L2GenesisPath)
	if err != nil {
		return nil, fmt.Errorf("read l2 genesis file: %w", err)
	}
	var genesis core.Genesis
	err = json.Unmarshal(data, &genesis)
	if err != nil {
		return nil, fmt.Errorf("parse l2 genesis file: %w", err)
	}
	return genesis.Config, nil
}

// SeparatedFaultProofProgram is the programmatic entry-point for the fault proof program, using a pre-image oracle.
func SeparatedFaultProofProgram(logger log.Logger, cfg *config.Config) error {
	cfg.Rollup.LogDescription(logger, chaincfg.L2ChainIDToNetworkName)
	if !cfg.FetchingEnabled() {
		return errors.New("offline mode not supported")
	}

	ctx := context.Background()
	logger.Info("Connecting to L1 node", "l1", cfg.L1URL)
	l1Fetcher := hl1.NewPrefetcher(logger)

	logger.Info("Connecting to L2 node", "l2", cfg.L2URL)
	l2Fetcher, err := hl2.NewFetchingL2Oracle(ctx, logger, cfg.L2URL)
	if err != nil {
		return fmt.Errorf("connect l2 oracle: %w", err)
	}

	genesis, err := loadL2Genesis(cfg)
	if err != nil {
		return fmt.Errorf("failed to load L2 genesis: %w", err)
	}

	var kv kvstore.KV
	if cfg.PreimageDir == "" {
		kv = kvstore.NewMemKV()
	} else {
		kv = kvstore.NewDiskKV(cfg.PreimageDir)
	}

	pClientRW, pHostRW := bidrectionalPipe()
	pHost := preimage.NewOracleServer(pHostRW)
	hHostR, hClientW := io.Pipe()
	hHost := preimage.NewHintReader(hHostR)

	handleHint := func(req string) error {
		parts := strings.SplitN(req, " ", 2)
		switch parts[0] {
		case "l1-block":
			k := common.HexToHash(parts[1])
			bl, err := l1Fetcher.BlockByHash(k)
			if err != nil {
				return err
			}
			var buf bytes.Buffer
			if err := bl.EncodeRLP(&buf); err != nil {
				return err
			}
			return kv.Put(preimage.Keccak256Key(k).PreimageKey(), buf.Bytes())
		case "l2-block":
			k := common.HexToHash(parts[1])
			bl, err := l2Fetcher.BlockByHash(k)
			if err != nil {
				return err
			}
			var buf bytes.Buffer
			if err := bl.EncodeRLP(&buf); err != nil {
				return err
			}
			return kv.Put(preimage.Keccak256Key(k).PreimageKey(), buf.Bytes())
		case "l2-state-node":
			k := common.HexToHash(parts[1])
			p, err := l2Fetcher.NodeByHash(k)
			if err != nil {
				return err
			}
			return kv.Put(preimage.Keccak256Key(k).PreimageKey(), p)
		case "l2-code":
			k := common.HexToHash(parts[1])
			p, err := l2Fetcher.CodeByHash(k)
			if err != nil {
				return err
			}
			return kv.Put(preimage.Keccak256Key(k).PreimageKey(), p)
		}
		return nil
	}

	var hint string
	hintRouter := func(req string) error {
		logger.Info("received hint", "hint", req)
		hint = req
		return nil
	}

	// accept pre-image hints
	go func() {
		for {
			if err := hHost.NextHint(hintRouter); err != nil {
				if err == io.EOF {
					logger.Info("closing pre-image hint handler")
					return
				}
				logger.Error("pre-image hint handler hit an error", "err", err)
				return
			}
		}
	}()

	getPreimage := func(key common.Hash) ([]byte, error) {
		logger.Info("pre-image lookup", "key", key)
		value, err := kv.Get(key)

		// if the key is not found, try to execute our last pre-image hint,
		// to get missing pre-image(s) into the kv-store.
		if err == kvstore.NotFoundErr && hint != "" {
			logger.Info("executing last hint to get missing pre-image", "hint", hint)
			if err := handleHint(hint); err != nil {
				return nil, fmt.Errorf("failed to process hint %q: %w", hint, err)
			}
			hint = ""
			value, err = kv.Get(key)
		}

		if err != nil {
			return nil, fmt.Errorf("pre-image kv-store error: %w", err)
		} else {
			return value, nil
		}
	}

	// serve pre-images from the KV
	go func() {
		for {
			if err := pHost.NextPreimageRequest(getPreimage); err != nil {
				if err == io.EOF {
					logger.Info("closing pre-image server")
					return
				}
				logger.Error("pre-image server hit an error", "err", err)
				return
			}
		}
	}()

	if err := client.ClientProgram(logger, cfg.Rollup, genesis, cfg.L2Head, pClientRW, hClientW); err != nil {
		return fmt.Errorf("failed to execute client-side of the program: %w", err)
	}
	return nil
}
