package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

func main() {
	app := cli.NewApp()
	app.Name = "check-jovian"
	app.Usage = "Check Jovian upgrade results."
	app.Description = "Check Jovian upgrade results."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = []*cli.Command{
		{
			Name: "contracts",
			Subcommands: []*cli.Command{
				makeCommand("gpo", checkGPO),
				makeCommand("l1block", checkL1Block),
			},
		},
		makeCommand("block", checkBlock),
		makeCommand("extra-data", checkExtraData),
		makeCommand("all", checkAll),
	}

	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}

type actionEnv struct {
	log        log.Logger
	l2         *ethclient.Client
	l2endpoint string
	secretKey  string
}

type CheckAction func(ctx context.Context, env *actionEnv) error

var (
	prefix     = "CHECK_JOVIAN"
	EndpointL2 = &cli.StringFlag{
		Name:    "l2",
		Usage:   "L2 execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2"),
		Value:   "http://localhost:9545",
	}
	SecretKeyFlag = &cli.StringFlag{
		Name:    "secret-key",
		Usage:   "hex encoded secret key for sending a test tx (optional)",
		EnvVars: op_service.PrefixEnvVar(prefix, "SECRET_KEY"),
		Value:   "",
	}
)

func makeFlags() []cli.Flag {
	flags := []cli.Flag{
		EndpointL2,
		SecretKeyFlag,
	}
	return append(flags, oplog.CLIFlags(prefix)...)
}

func makeCommand(name string, fn CheckAction) *cli.Command {
	return &cli.Command{
		Name:   name,
		Action: makeCommandAction(fn),
		Flags:  cliapp.ProtectFlags(makeFlags()),
	}
}

func makeCommandAction(fn CheckAction) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(c)
		logger := oplog.NewLogger(c.App.Writer, logCfg)

		c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
		l2Cl, err := ethclient.DialContext(c.Context, c.String(EndpointL2.Name))
		if err != nil {
			return fmt.Errorf("failed to dial L2 RPC: %w", err)
		}
		secretKey := c.String(SecretKeyFlag.Name)
		if secretKey != "" {
			// Normalize possible 0x prefix
			secretKey = strings.TrimPrefix(secretKey, "0x")
		}
		if err := fn(c.Context, &actionEnv{
			log:        logger,
			l2:         l2Cl,
			l2endpoint: c.String(EndpointL2.Name),
			secretKey:  secretKey,
		}); err != nil {
			return fmt.Errorf("command error: %w", err)
		}
		return nil
	}
}

// checkGPO checks that GasPriceOracle.isJovian returns true
func checkGPO(ctx context.Context, env *actionEnv) error {
	cl, err := bindings.NewGasPriceOracle(predeploys.GasPriceOracleAddr, env.l2)
	if err != nil {
		return fmt.Errorf("failed to create bindings around GasPriceOracle contract: %w", err)
	}
	isJovian, err := cl.IsJovian(nil)
	if err != nil {
		return fmt.Errorf("failed to get jovian status: %w", err)
	}
	if !isJovian {
		return fmt.Errorf("GPO is not set to jovian")
	}
	env.log.Info("GasPriceOracle test: success", "isJovian", isJovian)
	return nil
}

// checkL1Block checks that L1Block.DAFootprintGasScalar returns a number
func checkL1Block(ctx context.Context, env *actionEnv) error {
	cl, err := bindings.NewL1Block(predeploys.L1BlockAddr, env.l2)
	if err != nil {
		return fmt.Errorf("failed to create bindings around L1Block contract: %w", err)
	}
	daFootprintGasScalar, err := cl.DaFootprintGasScalar(nil)
	if err != nil {
		return fmt.Errorf("failed to get DA footprint gas scalar from L1Block contract: %w", err)
	}
	if daFootprintGasScalar == 0 {
		return fmt.Errorf("DA footprint gas scalar is set to 0 in L1Block contract, which should not be possible with Jovian.")
	}
	env.log.Info("L1Block test: success", "daFootprintGasScalar", daFootprintGasScalar)
	return nil
}

// checkBlock checks that a block for correct use of a the blobgasused field. It can be inconclusive if
// there are no user transactions in the block.
// If a secret key is provided, it will attempt to send a tx-to-self on L2, wait for it to be mined,
// then use the block containing that tx as the block to check.
func checkBlock(ctx context.Context, env *actionEnv) error {
	var err error
	var latest *types.Block

	// If a secret key was provided, attempt to send a tx-to-self and wait for it to be mined.
	if env.secretKey != "" {
		env.log.Info("secret key provided - attempting to send tx-to-self and wait for inclusion")

		cfg := txmgr.NewCLIConfig(env.l2endpoint, txmgr.DefaultBatcherFlagValues)
		cfg.PrivateKey = env.secretKey
		t, err := txmgr.NewSimpleTxManager("check-jovian", env.log.With("component", "txmgr"), new(metrics.NoopTxMetrics), cfg)
		if err != nil {
			return fmt.Errorf("failed to create tx manager: %w", err)
		}
		defer t.Close()
		fromAddr := t.From()

		receipt, err := t.Send(ctx, txmgr.TxCandidate{
			To:    &fromAddr, // Send to self
			Value: big.NewInt(0),
		})
		if err != nil {
			return fmt.Errorf("error waiting for tx to be mined: %w", err)
		}
		if receipt == nil {
			return fmt.Errorf("tx mined receipt was nil")
		}

		env.log.Info("tx mined", "txHash", receipt.TxHash.Hex(), "blockNumber", receipt.BlockNumber.Uint64(), "blobGasUsed", receipt.BlobGasUsed)

		if receipt.BlobGasUsed == 0 {
			return fmt.Errorf("receipt.BlobGasUsed was zero (required with Jovian)")
		}

		// Fetch the block that contained the receipt
		blk, err := env.l2.BlockByNumber(ctx, receipt.BlockNumber)
		if err != nil {
			return fmt.Errorf("failed to fetch block containing tx: %w", err)
		}
		latest = blk
	} else {
		latest, err = env.l2.BlockByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get latest block: %w", err)
		}
	}

	bguPtr := latest.BlobGasUsed()
	if bguPtr == nil {
		return fmt.Errorf("block %d has nil BlobGasUsed field", latest.Number())
	}
	bgu := *bguPtr

	txs := latest.Body().Transactions
	switch len(txs) {
	case 0:
		return fmt.Errorf("block %d has no transactions at all", latest.Number())
	case 1:
		env.log.Warn("Block has no user txs - inconclusive for Jovian activation",
			"blockNumber", latest.Number(),
			"note", "Zero could indicate an empty block or pre-Jovian state")
	default:
		expectedDAFootprint, err := types.CalcDAFootprint(txs)
		if err != nil {
			return fmt.Errorf("failed to calculate DA footprint for block %d: %w", latest.Number(), err)
		}
		if expectedDAFootprint != bgu {
			return fmt.Errorf("expected DA footprint %d stored in header.blobGasUsed but got %d", expectedDAFootprint, bgu)
		}
		env.log.Info("Block header test: success - non-zero BlobGasUsed is hard evidence of Jovian being active",
			"blockNumber", latest.Number,
			"blobGasUsed", bgu,
			"expectedDAFootprint", expectedDAFootprint)
	}
	return nil
}

// checkExtraData validates that the block header has the correct Jovian extra data format
func checkExtraData(ctx context.Context, env *actionEnv) error {
	latest, err := env.l2.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	extra := latest.Extra

	// Validate using op-geth's validation function
	if err := eip1559.ValidateJovianExtraData(extra); err != nil {
		return fmt.Errorf("invalid extraData format: %w", err)
	}

	// Decode the validated extra data using op-geth's decode function
	denominator, elasticity, minBaseFee := eip1559.DecodeJovianExtraData(extra)

	env.log.Info("ExtraData format test: success",
		"blockNumber", latest.Number,
		"version", extra[0],
		"denominator", denominator,
		"elasticity", elasticity,
		"minBaseFee", *minBaseFee)
	return nil
}

// checkAll runs all Jovian checks
func checkAll(ctx context.Context, env *actionEnv) error {
	env.log.Info("starting Jovian checks")

	if err := checkGPO(ctx, env); err != nil {
		return fmt.Errorf("failed: GPO contract error: %w", err)
	}
	if err := checkL1Block(ctx, env); err != nil {
		return fmt.Errorf("failed: L1Block contract error: %w", err)
	}
	if err := checkBlock(ctx, env); err != nil {
		return fmt.Errorf("failed: block header error: %w", err)
	}
	if err := checkExtraData(ctx, env); err != nil {
		return fmt.Errorf("failed: extra data format error: %w", err)
	}

	env.log.Info("completed all tests successfully!")

	return nil
}
