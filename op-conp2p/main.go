// Command op-conp2p is a thin P2P sidecar for the opql consensus nodes
// (op-con-node / op-con-ex-node), which have no built-in P2P. It reuses
// op-node's P2P stack to join the OP gossip network and receive unsafe
// ExecutionPayloadEnvelopes, then delegates the per-block signature verdict back
// to the consensus node over JSON-RPC (admin_verifyUnsafePayload). The node owns
// the sequencer-signer truth and decides accept/reject/ignore; the sidecar holds
// no signer state and maps the node's verdict onto its gossipsub validator
// result, which is the relay decision.
//
// See docs/sidecar-p2p-design.md in the opql repo.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	opnodeflags "github.com/ethereum-optimism/optimism/op-node/flags"
	opmetrics "github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

const envPrefix = "OP_CONP2P"

var (
	rollupConfigFlag = &cli.StringFlag{
		Name:     "rollup.config",
		Usage:    "Path to the standard op-node rollup config JSON (chain id + fork times drive gossip topic selection).",
		EnvVars:  []string{envPrefix + "_ROLLUP_CONFIG"},
		Required: true,
	}
	nodeRPCFlag = &cli.StringFlag{
		Name:     "node.rpc",
		Usage:    "Consensus node JSON-RPC endpoint (ws:// or http://) exposing admin_verifyUnsafePayload.",
		EnvVars:  []string{envPrefix + "_NODE_RPC"},
		Required: true,
	}
	nodeTimeoutFlag = &cli.DurationFlag{
		Name:    "node.rpc.timeout",
		Usage:   "Per-call timeout for the consensus node verdict RPC.",
		EnvVars: []string{envPrefix + "_NODE_RPC_TIMEOUT"},
		Value:   2 * time.Second,
	}
	signedPayloadWSFlag = &cli.StringFlag{
		Name: "signed-payload-ws",
		Usage: "Websocket URL of a sequencing op-con-node's signed-payload multicast " +
			"(--sequencer-payload-ws-addr). When set, the sidecar subscribes and PUBLISHES " +
			"each node-signed unsafe block to the OP gossip /blocks topics (never re-signing).",
		EnvVars: []string{envPrefix + "_SIGNED_PAYLOAD_WS"},
	}
)

func main() {
	logger := log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true))

	app := cli.NewApp()
	app.Name = "op-conp2p"
	app.Usage = "OP gossip P2P sidecar for the opql consensus nodes"
	app.Flags = append([]cli.Flag{rollupConfigFlag, nodeRPCFlag, nodeTimeoutFlag, signedPayloadWSFlag},
		opnodeflags.P2PFlags(envPrefix)...)
	app.Action = func(cliCtx *cli.Context) error {
		return run(cliCtx, logger)
	}

	if err := app.Run(os.Args); err != nil {
		logger.Crit("op-conp2p failed", "err", err)
	}
}

func run(cliCtx *cli.Context, logger log.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rollupCfg, err := loadRollupConfig(cliCtx.String(rollupConfigFlag.Name))
	if err != nil {
		return fmt.Errorf("failed to load rollup config: %w", err)
	}

	p2pConfig, err := p2pcli.NewConfig(cliCtx, rollupCfg.BlockTime)
	if err != nil {
		return fmt.Errorf("failed to load p2p config: %w", err)
	}
	if p2pConfig.Disabled() {
		return fmt.Errorf("p2p is disabled; the sidecar requires p2p to be enabled")
	}

	node, err := dialNode(ctx, cliCtx.String(nodeRPCFlag.Name), cliCtx.Duration(nodeTimeoutFlag.Name), logger)
	if err != nil {
		return err
	}

	runCfg := &delegatingRuntimeConfig{node: node, log: logger}
	gossipIn := &loggingGossipIn{log: logger}

	n, err := p2p.NewNodeP2P(
		ctx,
		rollupCfg,
		logger,
		p2pConfig,
		gossipIn,
		nil, // no L2Chain: gossip-only, no req-resp server
		runCfg,
		opmetrics.NoopMetrics,
		clock.SystemClock,
	)
	if err != nil {
		return fmt.Errorf("failed to start p2p node: %w", err)
	}
	defer n.Close()

	// Publish path: when a sequencing op-con-node's signed-payload feed is
	// configured, bridge it onto gossip. The node signed each block already; the
	// sidecar only re-encodes and publishes (see publish.go).
	if wsURL := cliCtx.String(signedPayloadWSFlag.Name); wsURL != "" {
		pub := &payloadPublisher{log: logger, url: wsURL, out: n.GossipOut()}
		go pub.run(ctx)
		logger.Info("signed-payload publish path enabled", "feed", wsURL)
	}

	logger.Info("op-conp2p sidecar started",
		"chain_id", rollupCfg.L2ChainID,
		"node_rpc", cliCtx.String(nodeRPCFlag.Name),
		"peer_id", n.Host().ID(),
	)

	<-ctx.Done()
	logger.Info("op-conp2p sidecar shutting down")
	return nil
}

func loadRollupConfig(path string) (*rollup.Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var cfg rollup.Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse rollup config %s: %w", path, err)
	}
	return &cfg, nil
}
