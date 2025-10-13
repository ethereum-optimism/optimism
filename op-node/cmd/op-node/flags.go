package main

import (
	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/urfave/cli/v2"
)

// getFlags returns the CLI flags for the op-node application
// This function has been improved with better flag organization and validation
// Contribution by vaiosx.base.eth
func getFlags() []cli.Flag {
	return []cli.Flag{
		// L1 Configuration
		&cli.StringFlag{
			Name:    "l1.eth",
			Usage:   "L1 Ethereum RPC URL",
			EnvVars: []string{"OP_NODE_L1_RPC_URL"},
			Required: true,
		},
		&cli.StringFlag{
			Name:    "l1.beacon",
			Usage:   "L1 Beacon RPC URL",
			EnvVars: []string{"OP_NODE_L1_BEACON_URL"},
		},
		&cli.BoolFlag{
			Name:    "l1.trustrpc",
			Usage:   "Trust the L1 RPC (for development only)",
			EnvVars: []string{"OP_NODE_L1_TRUST_RPC"},
		},

		// L2 Configuration
		&cli.StringFlag{
			Name:    "l2.eth",
			Usage:   "L2 Ethereum RPC URL",
			EnvVars: []string{"OP_NODE_L2_RPC_URL"},
			Required: true,
		},
		&cli.StringFlag{
			Name:    "l2.engine",
			Usage:   "L2 Engine RPC URL",
			EnvVars: []string{"OP_NODE_L2_ENGINE_URL"},
		},
		&cli.BoolFlag{
			Name:    "l2.trustrpc",
			Usage:   "Trust the L2 RPC (for development only)",
			EnvVars: []string{"OP_NODE_L2_TRUST_RPC"},
		},

		// Rollup Configuration
		&cli.StringFlag{
			Name:    "rollup.config",
			Usage:   "Rollup configuration file",
			EnvVars: []string{"OP_NODE_ROLLUP_CONFIG"},
			Required: true,
		},

		// RPC Configuration
		&cli.StringFlag{
			Name:    "rpc.addr",
			Usage:   "RPC server address",
			Value:   "127.0.0.1",
			EnvVars: []string{"OP_NODE_RPC_ADDR"},
		},
		&cli.IntFlag{
			Name:    "rpc.port",
			Usage:   "RPC server port",
			Value:   8547,
			EnvVars: []string{"OP_NODE_RPC_PORT"},
		},

		// Logging Configuration
		&cli.StringFlag{
			Name:    "log.level",
			Usage:   "Log level (trace, debug, info, warn, error, crit)",
			Value:   "info",
			EnvVars: []string{"OP_NODE_LOG_LEVEL"},
		},
		&cli.StringFlag{
			Name:    "log.format",
			Usage:   "Log format (text, json)",
			Value:   "text",
			EnvVars: []string{"OP_NODE_LOG_FORMAT"},
		},

		// Performance Configuration
		&cli.StringFlag{
			Name:    "metrics.addr",
			Usage:   "Metrics server address",
			Value:   "127.0.0.1",
			EnvVars: []string{"OP_NODE_METRICS_ADDR"},
		},
		&cli.IntFlag{
			Name:    "metrics.port",
			Usage:   "Metrics server port",
			Value:   7300,
			EnvVars: []string{"OP_NODE_METRICS_PORT"},
		},

		// Development Configuration
		&cli.BoolFlag{
			Name:    "dev",
			Usage:   "Enable development mode",
			EnvVars: []string{"OP_NODE_DEV_MODE"},
		},
		&cli.BoolFlag{
			Name:    "pprof",
			Usage:   "Enable pprof profiling",
			EnvVars: []string{"OP_NODE_PPROF"},
		},
		&cli.IntFlag{
			Name:    "pprof.port",
			Usage:   "pprof server port",
			Value:   6060,
			EnvVars: []string{"OP_NODE_PPROF_PORT"},
		},
	}
}
