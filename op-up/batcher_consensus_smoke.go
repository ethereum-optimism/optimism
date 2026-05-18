package main

import (
	"context"
	"fmt"
	"io"
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/urfave/cli/v2"
)

var batcherConsensusSmokeL2URLFlag = &cli.StringFlag{
	Name:    "l2-rpc",
	Usage:   "RPC URL for the consensus-enabled L2.",
	EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_BATCHER_CONSENSUS_L2_RPC"),
	Value:   "http://localhost:8545",
}
var batcherConsensusSmokeValidL2URLFlag = &cli.StringFlag{
	Name:    "valid-l2-rpc",
	Usage:   "RPC URL for the consensus-enabled L2 with valid Commonware proofs.",
	EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_BATCHER_CONSENSUS_VALID_L2_RPC"),
	Value:   "http://localhost:8545",
}
var batcherConsensusSmokeInvalidL2URLFlag = &cli.StringFlag{
	Name:    "invalid-l2-rpc",
	Usage:   "RPC URL for the consensus-enabled L2 with invalid Commonware proofs.",
	EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_BATCHER_CONSENSUS_INVALID_L2_RPC"),
	Value:   "http://localhost:8546",
}

const batcherConsensusRejectTimeout = 20 * time.Second

type batcherConsensusSmokeEnv struct {
	ctx    context.Context
	stderr io.Writer
	chain  *remoteChain
	user   *remoteUser
}

func batcherConsensusSmokeCommand() *cli.Command {
	smokeFlags := cliapp.ProtectFlags([]cli.Flag{batcherConsensusSmokeL2URLFlag, smokePrivateKeyFlag})
	smokeAllFlags := cliapp.ProtectFlags([]cli.Flag{
		batcherConsensusSmokeValidL2URLFlag,
		batcherConsensusSmokeInvalidL2URLFlag,
		smokePrivateKeyFlag,
	})
	return &cli.Command{
		Name:  "smoke-batcher-consensus",
		Usage: "run batcher consensus smoke tests against a remote chain RPC",
		Subcommands: []*cli.Command{
			{
				Name:  "all",
				Usage: "run valid and invalid batcher consensus smoke tests sequentially",
				Flags: smokeAllFlags,
				Action: func(cliCtx *cli.Context) error {
					return smokeBatcherConsensusAll(cliCtx)
				},
			},
			{
				Name:  "safe-advance",
				Usage: "send an L2 transfer and verify the safe head advances to include it",
				Flags: smokeFlags,
				Action: func(cliCtx *cli.Context) error {
					return withBatcherConsensusSmokeEnv(cliCtx, "Batcher Consensus Safe Advance", smokeBatcherConsensusSafeAdvance)
				},
			},
			{
				Name:  "invalid-rejection",
				Usage: "send an L2 transfer and verify invalid consensus proofs keep it out of the safe head",
				Flags: smokeFlags,
				Action: func(cliCtx *cli.Context) error {
					return withBatcherConsensusSmokeEnv(cliCtx, "Batcher Consensus Invalid Rejection", smokeBatcherConsensusInvalidRejection)
				},
			},
		},
	}
}

func newBatcherConsensusSmokeEnv(ctx context.Context, stderr io.Writer, name, url, privateKey string) (*batcherConsensusSmokeEnv, func(), error) {
	logger := newLogger(ctx, stderr)
	chain, err := connectRemoteChain(ctx, logger, name, url)
	if err != nil {
		return nil, nil, err
	}
	privKey, address, err := resolveSmokeKey(privateKey)
	if err != nil {
		chain.ethClient.Close()
		return nil, nil, err
	}
	env := &batcherConsensusSmokeEnv{
		ctx:    ctx,
		stderr: stderr,
		chain:  chain,
		user:   &remoteUser{chain: chain, privKey: privKey, address: address},
	}
	cleanup := func() {
		chain.ethClient.Close()
	}
	return env, cleanup, nil
}

func withBatcherConsensusSmokeEnv(cliCtx *cli.Context, name string, fn func(env *batcherConsensusSmokeEnv) error) error {
	ctx := cliCtx.Context
	stderr := cliCtx.App.ErrWriter
	env, cleanup, err := newBatcherConsensusSmokeEnv(
		ctx,
		stderr,
		"L2",
		cliCtx.String(batcherConsensusSmokeL2URLFlag.Name),
		cliCtx.String(smokePrivateKeyFlag.Name),
	)
	if err != nil {
		return err
	}
	defer cleanup()
	fmt.Fprintf(stderr, "\n=== %s ===\n", name)
	fmt.Fprintf(stderr, "%s RPC: %s (chain ID %s)\n", env.chain.name, env.chain.url, env.chain.chainID)
	fmt.Fprintf(stderr, "Smoke Sender Address: %s\n\n", env.user.address)
	if err := fn(env); err != nil {
		fmt.Fprintf(stderr, "\nFAIL: %s (%v)\n", name, err)
		return err
	}
	fmt.Fprintf(stderr, "\nPASS: %s\n", name)
	return nil
}

func smokeBatcherConsensusAll(cliCtx *cli.Context) error {
	ctx := cliCtx.Context
	stderr := cliCtx.App.ErrWriter
	validEnv, validCleanup, err := newBatcherConsensusSmokeEnv(
		ctx,
		stderr,
		"Valid L2",
		cliCtx.String(batcherConsensusSmokeValidL2URLFlag.Name),
		cliCtx.String(smokePrivateKeyFlag.Name),
	)
	if err != nil {
		return err
	}
	defer validCleanup()

	invalidEnv, invalidCleanup, err := newBatcherConsensusSmokeEnv(
		ctx,
		stderr,
		"Invalid L2",
		cliCtx.String(batcherConsensusSmokeInvalidL2URLFlag.Name),
		cliCtx.String(smokePrivateKeyFlag.Name),
	)
	if err != nil {
		return err
	}
	defer invalidCleanup()

	fmt.Fprintf(stderr, "\n=== Batcher Consensus All ===\n")
	fmt.Fprintf(stderr, "Valid L2 RPC: %s (chain ID %s)\n", validEnv.chain.url, validEnv.chain.chainID)
	fmt.Fprintf(stderr, "Invalid L2 RPC: %s (chain ID %s)\n", invalidEnv.chain.url, invalidEnv.chain.chainID)
	fmt.Fprintf(stderr, "Smoke Sender Address: %s\n\n", validEnv.user.address)

	if err := smokeBatcherConsensusSafeAdvance(validEnv); err != nil {
		fmt.Fprintf(stderr, "\nFAIL: Batcher Consensus Safe Advance (%v)\n", err)
		return err
	}
	if err := smokeBatcherConsensusInvalidRejection(invalidEnv); err != nil {
		fmt.Fprintf(stderr, "\nFAIL: Batcher Consensus Invalid Rejection (%v)\n", err)
		return err
	}
	fmt.Fprintf(stderr, "\nPASS: Batcher Consensus All\n")
	return nil
}

func smokeBatcherConsensusSafeAdvance(env *batcherConsensusSmokeEnv) error {
	recipient := randomAddress()
	tx, err := env.user.transfer(env.ctx, recipient, eth.OneWei)
	if err != nil {
		return fmt.Errorf("send L2 transfer: %w", err)
	}
	receipt, err := tx.Included.Eval(env.ctx)
	if err != nil {
		return fmt.Errorf("wait for L2 transfer inclusion: %w", err)
	}
	targetSafe := receipt.BlockNumber.Uint64()
	fmt.Fprintf(env.stderr, "    Included transfer in L2 block %d, waiting for safe head\n", targetSafe)
	deadline := time.Now().Add(2 * smokeWaitTimeout)
	for time.Now().Before(deadline) {
		safe, err := env.chain.ethClient.BlockRefByLabel(env.ctx, eth.Safe)
		if err == nil && safe.Number >= targetSafe {
			fmt.Fprintf(env.stderr, "    Safe head reached L2 block %d\n", safe.Number)
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("safe head did not reach L2 block %d before timeout", targetSafe)
}

func smokeBatcherConsensusInvalidRejection(env *batcherConsensusSmokeEnv) error {
	startSafe, err := env.chain.ethClient.BlockRefByLabel(env.ctx, eth.Safe)
	if err != nil {
		return fmt.Errorf("read starting safe head: %w", err)
	}
	recipient := randomAddress()
	tx, err := env.user.transfer(env.ctx, recipient, eth.OneWei)
	if err != nil {
		return fmt.Errorf("send L2 transfer: %w", err)
	}
	receipt, err := tx.Included.Eval(env.ctx)
	if err != nil {
		return fmt.Errorf("wait for L2 transfer inclusion: %w", err)
	}
	targetSafe := receipt.BlockNumber.Uint64()
	fmt.Fprintf(env.stderr, "    Starting safe head: L2 block %d\n", startSafe.Number)
	fmt.Fprintf(env.stderr, "    Included transfer in unsafe L2 block %d, checking it is not marked safe\n", targetSafe)
	deadline := time.Now().Add(batcherConsensusRejectTimeout)
	for time.Now().Before(deadline) {
		safe, err := env.chain.ethClient.BlockRefByLabel(env.ctx, eth.Safe)
		if err == nil && safe.Number >= targetSafe {
			return fmt.Errorf("safe head unexpectedly reached L2 block %d with invalid consensus proofs", safe.Number)
		}
		time.Sleep(time.Second)
	}
	fmt.Fprintf(env.stderr, "    Safe head stayed below L2 block %d for %s\n", targetSafe, batcherConsensusRejectTimeout)
	return nil
}
