package main

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var (
	BackTestStart = &cli.Uint64Flag{
		Name:  "start",
		Usage: "L2 block number to start submitting data from",
	}
	BackTestEnd = &cli.Uint64Flag{
		Name:  "end",
		Usage: "L2 block number to stop submitting data at",
	}
	BackTestL2 = &cli.StringFlag{
		Name:  "l2",
		Usage: "L2 RPC endpoint to retrieve block data from",
	}
)

func BackTest(cliCtx *cli.Context) error {
	logger, err := setupLogging(cliCtx)
	if err != nil {
		return err
	}
	ctx := cliCtx.Context

	if !cliCtx.IsSet(BackTestL2.Name) {
		return fmt.Errorf("--%s must be set", BackTestStart.Name)
	}
	if !cliCtx.IsSet(BackTestStart.Name) {
		return fmt.Errorf("--%s must be set", BackTestStart.Name)
	}
	if !cliCtx.IsSet(BackTestEnd.Name) {
		return fmt.Errorf("--%s must be set", BackTestStart.Name)
	}
	l2URL := cliCtx.String(BackTestL2.Name)
	start := cliCtx.Uint64(BackTestStart.Name)
	end := cliCtx.Uint64(BackTestEnd.Name)

	logger.Info("Starting", "l2", l2URL, "start", start, "end", end)

	client, err := dial.DialEthClientWithTimeout(ctx, time.Minute, logger, l2URL)
	if err != nil {
		return fmt.Errorf("failed to dial L2: %w", err)
	}

	rollupCfg, err := rollup.LoadOPStackRollupConfig(10)
	if err != nil {
		return fmt.Errorf("failed to get mainnet rollup config: %w", err)
	}
	configProvider := ConfigProviderFn(func() batcher.ChannelConfig {
		targetNumFrames := 5
		maxFrameSize := uint64(eth.MaxBlobDataSize - 1)
		return batcher.ChannelConfig{
			SeqWindowSize:      rollupCfg.SeqWindowSize,
			ChannelTimeout:     rollupCfg.ChannelTimeoutGranite,
			MaxChannelDuration: 150,
			SubSafetyMargin:    40,
			MaxFrameSize:       maxFrameSize,
			TargetNumFrames:    targetNumFrames,
			CompressorConfig: compressor.Config{
				// Compressor output size needs to account for frame encoding overhead
				TargetOutputSize: batcher.MaxDataSize(targetNumFrames, maxFrameSize),
				ApproxComprRatio: 0.6,
				Kind:             compressor.ShadowKind,
				CompressionAlgo:  derive.Brotli10,
			},
			//BatchType: 0, // Singular batches
			BatchType: 1, // Span batches
			UseBlobs:  true,
		}
	})
	manager := batcher.NewChannelManager(logger, metrics.NoopMetrics, configProvider, rollupCfg)

	txCount := 0
	totalDataLen := 0

	fakeL1Block := eth.BlockID{
		Number: 0,
		Hash:   common.Hash{0xaa},
	}
	submitData := func(blockNum uint64) error {
		data, err := manager.TxData(fakeL1Block)
		if errors.Is(err, io.EOF) {
			logger.Info("No tx data to submit yet", "block", blockNum, "txs", txCount, "data", totalDataLen, "remaining", end-blockNum)
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to get tx data: %w", err)
		}
		logger.Info("Got tx data", "len", data.Len())
		txCount++
		totalDataLen += data.Len()
		manager.TxConfirmed(data.ID(), fakeL1Block)
		fakeL1Block.Number++
		return nil
	}
	for blockNum := start; blockNum <= end; blockNum++ {
		block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		if err != nil {
			return fmt.Errorf("failed to retrieve block %d: %w", blockNum, err)
		}
		if err := manager.AddL2Block(block); err != nil {
			return fmt.Errorf("failed to add L2 block %d: %w", blockNum, err)
		}

		if err := submitData(blockNum); err != nil {
			return fmt.Errorf("failed to submit data: %w", err)
		}
	}
	if err := manager.Close(); err != nil {
		return fmt.Errorf("failed to close manager: %w", err)
	}
	if err := submitData(end); err != nil {
		return fmt.Errorf("failed to submit data: %w", err)
	}
	logger.Info("All blocks submitted", "txs", txCount, "data", totalDataLen)
	return nil
}

type ConfigProviderFn func() batcher.ChannelConfig

func (f ConfigProviderFn) ChannelConfig() batcher.ChannelConfig {
	return f()
}

func backTestFlags() []cli.Flag {
	return []cli.Flag{
		BackTestStart,
		BackTestEnd,
		BackTestL2,
	}
}

var BackTestCommand = &cli.Command{
	Name:        "back-test",
	Usage:       "Compute the amount data that would be required to submit a range of L2 blocks to L1",
	Description: "Runs trace providers against real chain data to confirm compatibility",
	Action:      Interruptible(BackTest),
	Flags:       backTestFlags(),
}

func setupLogging(ctx *cli.Context) (log.Logger, error) {
	logCfg := oplog.ReadCLIConfig(ctx)
	logger := oplog.NewLogger(oplog.AppOut(ctx), logCfg)
	oplog.SetGlobalLogHandler(logger.Handler())
	return logger, nil
}

func Interruptible(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		ctx.Context = opio.CancelOnInterrupt(ctx.Context)
		return action(ctx)
	}
}
