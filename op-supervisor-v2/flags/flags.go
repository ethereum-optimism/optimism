package flags

import (
	"time"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

// EnvVarPrefix mirrors the style of other OP components; reserved for future use.
const EnvVarPrefix = "SV2_"

// Flags defines the CLI flags for op-supervisor-v2.
// Keep names stable to avoid breaking existing invocations.
var Flags = []cli.Flag{
	&cli.StringFlag{Name: "sv2.config", Value: "", Usage: "Path to multi-chain JSON config (overrides per-chain flags)"},
	&cli.StringFlag{Name: "http.addr", Value: "127.0.0.1", Usage: "HTTP listen address"},
	&cli.IntFlag{Name: "http.port", Value: 9750, Usage: "HTTP listen port"},
	&cli.BoolFlag{Name: "proxy.opnode", Value: true, Usage: "Expose virtual op-node RPC under /opnode/"},
	&cli.BoolFlag{Name: "disable.p2p", Value: false, Usage: "Disable P2P networking for virtual op-nodes"},
	&cli.StringFlag{Name: "sv2.data-dir", Value: "", Usage: "SV2 data dir (denylist.json and chain DBs). Default is a unique temp dir."},

	// Single-chain bootstrap (Milestone 1): retained for backwards-compatibility.
	&cli.StringFlag{Name: "l1.rpc", Usage: "L1 execution RPC endpoint"},
	&cli.StringFlag{Name: "beacon.addr", Usage: "L1 beacon endpoint for blobs (e.g. http://localhost:5052)"},
	&cli.StringFlag{Name: "l2.authrpc", Usage: "L2 execution Engine API (auth RPC) endpoint"},
	&cli.StringFlag{Name: "l2.userrpc", Usage: "L2 execution user RPC endpoint (for reads)"},
	&cli.StringFlag{Name: "jwt.secret", Usage: "Path to 32-byte hex JWT secret file for L2 Engine API"},
	&cli.StringFlag{Name: "rollup.config", Usage: "Path to rollup config JSON"},

	&cli.DurationFlag{Name: "poll.interval", Value: 1 * time.Second, Usage: "Polling interval for rollup status"},
	&cli.UintFlag{Name: "confirm.depth", Value: 40, Usage: "L1 confirmation depth for cross-safety gating"},
}

// init adds logging flags to the Flags array
func init() {
	Flags = append(Flags, oplog.CLIFlags(EnvVarPrefix)...)
}
