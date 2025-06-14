package flags

import (
	"time"

	"github.com/urfave/cli/v2"
)

var (
	L2EngineAddrs = &cli.StringSliceFlag{
		Name:    "l2",
		Usage:   "Addresses of L2 Engine JSON-RPC endpoints to use (engine and eth RPC namespaces are required)",
		EnvVars: prefixEnvVars("L2_ENGINE_RPC"),
		// Note: originally in rollup-category, but non-breaking change.
		Category: L2RPCCategory,
		// Note: originally a single-value, now a slice in v2.
		// Now has a default in v2. Defaulting to a port different from the L1 default.
		Value: cli.NewStringSlice("ws://127.0.0.1:8651"),
	}
	L2EngineJWTSecrets = &cli.StringSliceFlag{
		Name: "l2.jwt-secret",
		Usage: "Path(s) to JWT secret key(s). Keys are 32 bytes, hex encoded in a file. " +
			"A new key will be generated if the file does not exist. " +
			"If multiple paths are specified, secrets are assumed to match l2 engine endpoints order.",
		EnvVars: prefixEnvVars("L2_ENGINE_AUTH"),
		// Note: originally in rollup-category, but non-breaking change.
		Category: L2RPCCategory,
		// Note: originally a single-value, now a slice in v2.
		Value: cli.NewStringSlice("jwt_secret.txt"), // now has a default in v2
	}
	L2EngineRpcTimeout = &cli.DurationFlag{
		Name:    "l2.engine-rpc-timeout",
		Usage:   "L2 engine client RPC request timeout.",
		EnvVars: prefixEnvVars("L2_ENGINE_RPC_TIMEOUT"),
		Value:   time.Second * 10,
		// Note: originally in rollup-category, but non-breaking change.
		Category: L2RPCCategory,
	}
	L2ReadAddrs = &cli.StringSliceFlag{
		Name:     "l2.read-rpc",
		Usage:    "Addresses of L2 read-only JSON-RPC endpoints to use as alternative to an execution engine.",
		EnvVars:  prefixEnvVars("L2_READ_RPC"),
		Category: L2RPCCategory,
		// empty by default
	}
)

var L2RPCFlags = []cli.Flag{
	L2EngineAddrs,
	L2EngineJWTSecrets,
	L2EngineRpcTimeout,
	L2ReadAddrs,
}
