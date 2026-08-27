// Command op-silhouette-el runs the proof-backed, in-memory execution-layer shim used by
// silhouette verifier nodes. It executes no EVM code and persists no chain state.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gn "github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supernode/silhouette"
	"github.com/ethereum/go-ethereum/log"
)

var (
	Version   = "v0.1.0"
	GitCommit = ""
	GitDate   = ""
)

const envPrefix = "OP_SILHOUETTE_EL"

func main() {
	oplog.SetupDefaults()
	app := cli.NewApp()
	app.Name = "op-silhouette-el"
	app.Usage = "Proof-backed in-memory EL for silhouette verifier nodes"
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Flags = cliapp.ProtectFlags(append([]cli.Flag{
		&cli.StringFlag{Name: "l1", Required: true, EnvVars: []string{envPrefix + "_L1"}},
		&cli.StringFlag{Name: "l1.beacon", Required: true, EnvVars: []string{envPrefix + "_L1_BEACON"}},
		&cli.Uint64Flag{Name: "l1.beacon.slot-duration-override", EnvVars: []string{envPrefix + "_L1_BEACON_SLOT_DURATION_OVERRIDE"}},
		&cli.StringFlag{Name: "supernode", Required: true, EnvVars: []string{envPrefix + "_SUPERNODE"}},
		&cli.PathFlag{Name: "rollup-config", Required: true, TakesFile: true, EnvVars: []string{envPrefix + "_ROLLUP_CONFIG"}},
		&cli.PathFlag{Name: "verifier-config", Required: true, TakesFile: true, EnvVars: []string{envPrefix + "_VERIFIER_CONFIG"}},
		&cli.StringFlag{Name: "replacement-engine", Required: true, EnvVars: []string{envPrefix + "_REPLACEMENT_ENGINE"}},
		&cli.PathFlag{Name: "replacement-engine.jwt-secret", Required: true, TakesFile: true, EnvVars: []string{envPrefix + "_REPLACEMENT_ENGINE_JWT_SECRET"}},
		&cli.StringFlag{Name: "rpc.addr", Value: "127.0.0.1", EnvVars: []string{envPrefix + "_RPC_ADDR"}},
		&cli.IntFlag{Name: "rpc.port", Value: 9545, EnvVars: []string{envPrefix + "_RPC_PORT"}},
		&cli.DurationFlag{Name: "poll-interval", Value: 2 * time.Second, EnvVars: []string{envPrefix + "_POLL_INTERVAL"}},
	}, oplog.CLIFlags(envPrefix)...))
	app.Action = run

	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Crit("Application failed", "message", err)
	}
}

func run(cliCtx *cli.Context) error {
	logger := oplog.NewLogger(oplog.AppOut(cliCtx), oplog.ReadCLIConfig(cliCtx))
	oplog.SetGlobalLogHandler(logger.Handler())

	verifierCfg, err := silhouette.LoadConfig(cliCtx.Path("verifier-config"))
	if err != nil {
		return err
	}
	rollupCfg, err := loadRollupConfig(cliCtx.Path("rollup-config"))
	if err != nil {
		return err
	}
	l1Chain, err := silhouette.L1ChainConfig(verifierCfg)
	if err != nil {
		return err
	}

	l1RPC, err := client.NewRPC(cliCtx.Context, logger, cliCtx.String("l1"),
		client.WithDialAttempts(10), client.WithHttpPollInterval(cliCtx.Duration("poll-interval")))
	if err != nil {
		return fmt.Errorf("dial L1: %w", err)
	}
	defer l1RPC.Close()
	l1Client, err := sources.NewL1Client(l1RPC, logger, nil,
		sources.L1ClientSimpleConfig(false, sources.RPCKindStandard, 100))
	if err != nil {
		return fmt.Errorf("create L1 client: %w", err)
	}
	defer l1Client.Close()

	beaconHTTP := sources.NewBeaconHTTPClient(
		client.NewBasicHTTPClient(cliCtx.String("l1.beacon"), logger),
		sources.WithSlotDurationOverride(cliCtx.Uint64("l1.beacon.slot-duration-override")),
	)
	beacon := sources.NewL1BeaconClient(beaconHTTP, sources.L1BeaconClientConfig{FetchAllSidecars: false})

	proofVerifier, err := verifierCfg.NewVerifier()
	if err != nil {
		return fmt.Errorf("create proof verifier: %w", err)
	}
	facts := &silhouette.FactStore{}
	supernodeRPC, err := client.NewRPC(cliCtx.Context, logger, cliCtx.String("supernode"),
		client.WithLazyDial(), client.WithCallTimeout(10*time.Second))
	if err != nil {
		return fmt.Errorf("configure supernode client: %w", err)
	}
	defer supernodeRPC.Close()
	supernodeClient := sources.NewSuperNodeClient(supernodeRPC)
	chainID := eth.ChainIDFromBig(rollupCfg.L2ChainID)
	facts.SetDeniedChecker(func(number uint64, hash common.Hash) (bool, error) {
		ctx, cancel := context.WithTimeout(cliCtx.Context, 10*time.Second)
		defer cancel()
		return supernodeClient.IsDenied(ctx, chainID, number, hash)
	})
	source := silhouette.NewDataSource(logger.New("component", "proof-source"), verifierCfg, rollupCfg,
		l1Chain, rollupCfg.Genesis.SystemConfig, l1Client, beacon, proofVerifier, facts)
	shim := silhouette.NewShim(logger.New("component", "engine"), rollupCfg, l1Chain,
		rollupCfg.Genesis.SystemConfig, l1Client, facts)
	replacementJWT, err := oprpc.ObtainJWTSecret(logger, cliCtx.Path("replacement-engine.jwt-secret"), false)
	if err != nil {
		return fmt.Errorf("read replacement engine JWT: %w", err)
	}
	replacementRPC, err := client.NewRPC(cliCtx.Context, logger, cliCtx.String("replacement-engine"),
		client.WithGethRPCOptions(gethrpc.WithHTTPAuth(gn.NewJWTAuth([32]byte(replacementJWT)))),
		client.WithCallTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("dial replacement engine: %w", err)
	}
	defer replacementRPC.Close()
	replacementEngine, err := sources.NewEngineClient(replacementRPC, logger.New("component", "replacement-engine"),
		nil, sources.EngineClientDefaultConfig(rollupCfg))
	if err != nil {
		return fmt.Errorf("create replacement engine client: %w", err)
	}
	shim.SetReplacementBuilder(silhouette.NewEngineReplacementBuilder(replacementEngine))

	start := verifierCfg.L1StartBlock
	if start == 0 {
		start = rollupCfg.Genesis.L1.Number
	}
	tracker := silhouette.NewProvenHeadTracker(logger.New("component", "proof-walker"), source,
		l1Client, start, cliCtx.Duration("poll-interval"))
	server := shim.Standalone(cliCtx.String("rpc.addr"), cliCtx.Int("rpc.port"))
	if err := server.Start(); err != nil {
		return fmt.Errorf("start RPC server: %w", err)
	}
	defer server.Stop()

	logger.Info("op-silhouette-el started",
		"chain", rollupCfg.L2ChainID, "rpc", server.Endpoint(), "l1_start", start)
	tracker.Run(cliCtx.Context)
	return nil
}

func loadRollupConfig(path string) (*rollup.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open rollup config %q: %w", path, err)
	}
	defer f.Close()
	var cfg rollup.Config
	if err := cfg.ParseRollupConfig(f); err != nil {
		return nil, fmt.Errorf("parse rollup config %q: %w", path, err)
	}
	if err := cfg.Check(); err != nil {
		return nil, fmt.Errorf("invalid rollup config %q: %w", path, err)
	}
	return &cfg, nil
}
