package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	levelopt "github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/urfave/cli/v2"

	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
)

func Main(cliCtx *cli.Context) error {
	if err := CheckRequired(cliCtx); err != nil {
		return err
	}

	cfg := ReadCLIConfig(cliCtx)
	if err := cfg.Check(); err != nil {
		return err
	}

	logCfg := oplog.ReadCLIConfig(cliCtx)

	l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(l.Handler())

	l.Info("Initializing Plasma DA server...")

	var store plasma.KVStore

	if cfg.LevelDBEnabled() {
		l.Info("Using LevelDB storage", "path", cfg.LevelDBPath)
		levelStore, err := leveldb.NewCustom(cfg.LevelDBPath, "plasma", func(options *levelopt.Options) {
			// TODO, not that crucial for now
		})
		if err != nil {
			return fmt.Errorf("failed to create LevelDB store: %w", err)
		}
		store = levelStore
	} else if cfg.S3Enabled() {
		s3, err := NewS3Store(cfg.S3Bucket)
		if err != nil {
			return fmt.Errorf("failed to create S3 store: %w", err)
		}
		store = s3
	}

	server := plasma.NewDAServer(cliCtx.String(ListenAddrFlagName), cliCtx.Int(PortFlagName), store)

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start the DA server")
	}

	defer func() {
		if err := server.Stop(); err != nil {
			l.Error("failed to stop DA server", "err", err)
		}
	}()

	opio.BlockOnInterrupts()

	return nil
}
