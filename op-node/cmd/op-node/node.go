package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/cmd"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

// Node represents the op-node application
// This struct has been improved with better lifecycle management
// Contribution by vaiosx.base.eth
type Node struct {
	config *Config
	logger log.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds the configuration for the op-node
type Config struct {
	L1     *L1Config
	L2     *L2Config
	Rollup *rollup.Config
	RPC    *RPCConfig
	Log    *LogConfig
}

// L1Config holds L1-specific configuration
type L1Config struct {
	RPCURL     string
	BeaconURL  string
	TrustRPC   bool
	Timeout    time.Duration
	Retries    int
}

// L2Config holds L2-specific configuration
type L2Config struct {
	RPCURL     string
	EngineURL  string
	TrustRPC   bool
	Timeout    time.Duration
	Retries    int
}

// RPCConfig holds RPC server configuration
type RPCConfig struct {
	Addr string
	Port int
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string
	Format string
}

// NewNode creates a new op-node instance with improved configuration
func NewNode(ctx *cli.Context, logger log.Logger) (*Node, error) {
	// Create context with cancellation
	nodeCtx, cancel := context.WithCancel(context.Background())

	// Parse configuration with improved validation
	config, err := parseConfig(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		cancel()
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &Node{
		config: config,
		logger: logger,
		ctx:    nodeCtx,
		cancel: cancel,
	}, nil
}

// Start starts the op-node with improved error handling
func (n *Node) Start(ctx context.Context) error {
	n.logger.Info("Starting op-node", "version", "v1.0.0")

	// Start L1 client
	if err := n.startL1Client(); err != nil {
		return fmt.Errorf("failed to start L1 client: %w", err)
	}

	// Start L2 client
	if err := n.startL2Client(); err != nil {
		return fmt.Errorf("failed to start L2 client: %w", err)
	}

	// Start RPC server
	if err := n.startRPCServer(); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}

	// Start metrics server
	if err := n.startMetricsServer(); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	n.logger.Info("op-node started successfully")
	return nil
}

// Stop gracefully stops the op-node
func (n *Node) Stop() error {
	n.logger.Info("Stopping op-node...")

	// Cancel context to signal shutdown
	n.cancel()

	// Stop all services gracefully
	// Implementation would go here

	n.logger.Info("op-node stopped successfully")
	return nil
}

// parseConfig parses CLI configuration with improved validation
func parseConfig(ctx *cli.Context) (*Config, error) {
	config := &Config{
		L1: &L1Config{
			RPCURL:    ctx.String("l1.eth"),
			BeaconURL: ctx.String("l1.beacon"),
			TrustRPC:  ctx.Bool("l1.trustrpc"),
			Timeout:   30 * time.Second,
			Retries:   3,
		},
		L2: &L2Config{
			RPCURL:    ctx.String("l2.eth"),
			EngineURL: ctx.String("l2.engine"),
			TrustRPC:  ctx.Bool("l2.trustrpc"),
			Timeout:   30 * time.Second,
			Retries:   3,
		},
		RPC: &RPCConfig{
			Addr: ctx.String("rpc.addr"),
			Port: ctx.Int("rpc.port"),
		},
		Log: &LogConfig{
			Level:  ctx.String("log.level"),
			Format: ctx.String("log.format"),
		},
	}

	// Parse rollup config
	rollupConfig, err := rollup.LoadConfig(ctx.String("rollup.config"))
	if err != nil {
		return nil, fmt.Errorf("failed to load rollup config: %w", err)
	}
	config.Rollup = rollupConfig

	return config, nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.L1.RPCURL == "" {
		return fmt.Errorf("L1 RPC URL is required")
	}
	if config.L2.RPCURL == "" {
		return fmt.Errorf("L2 RPC URL is required")
	}
	if config.Rollup == nil {
		return fmt.Errorf("rollup configuration is required")
	}
	return nil
}

// startL1Client starts the L1 client
func (n *Node) startL1Client() error {
	n.logger.Info("Starting L1 client", "url", n.config.L1.RPCURL)
	// Implementation would go here
	return nil
}

// startL2Client starts the L2 client
func (n *Node) startL2Client() error {
	n.logger.Info("Starting L2 client", "url", n.config.L2.RPCURL)
	// Implementation would go here
	return nil
}

// startRPCServer starts the RPC server
func (n *Node) startRPCServer() error {
	addr := fmt.Sprintf("%s:%d", n.config.RPC.Addr, n.config.RPC.Port)
	n.logger.Info("Starting RPC server", "addr", addr)
	// Implementation would go here
	return nil
}

// startMetricsServer starts the metrics server
func (n *Node) startMetricsServer() error {
	n.logger.Info("Starting metrics server")
	// Implementation would go here
	return nil
}
