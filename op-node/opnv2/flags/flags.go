package flags

import (
	"github.com/urfave/cli/v2"

	opNodeFlags "github.com/ethereum-optimism/optimism/op-node/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	RollupCategory     = "1. ROLLUP"
	L1RPCCategory      = "2. L1 RPC"
	L2RPCCategory      = "3. L2 RPC"
	SequencerCategory  = "4. SEQUENCER"
	OperationsCategory = "5. LOGGING, METRICS, DEBUGGING, AND API"
	P2PCategory        = "6. PEER-TO-PEER"
	AltDACategory      = "7. ALT-DA (EXPERIMENTAL)" // TODO: omitted from op-node v2 for now
	MiscCategory       = "8. MISC"
)

const EnvVarPrefixOpnode = "OP_NODE"

func prefixEnvVars(name string) (out []string) {
	out = append(out, opservice.PrefixEnvVar(EnvVarPrefixOpnode, name)...)
	return out
}

// Flags contains the list of configuration options available to the binary.
var Flags = []cli.Flag{}

func init() {
	Flags = append(Flags, RollupFlags...)
	Flags = append(Flags, L1RPCFlags...)
	Flags = append(Flags, L2RPCFlags...)
	Flags = append(Flags, SequencerFlags...)
	Flags = append(Flags, OperationsFlags...)
	// For backwards compat, use the op-node namespace
	Flags = append(Flags, opNodeFlags.P2PFlags("OP_NODE")...)
	Flags = append(Flags, DeprecatedFlags...)
	Flags = append(Flags, MiscFlags...)
}
