package flags

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const envVarPrefix = "OP_CHALLENGER"

// GameType is the type of dispute game
type GameType uint8

// DefaultGameType returns the default dispute game type.
func DefaultGameType() GameType {
	return AttestationDisputeGameType
}

const (
	// AttestationDisputeGameType is the uint8 enum value for the attestation dispute game
	AttestationDisputeGameType GameType = iota
	// FaultDisputeGameType is the uint8 enum value for the fault dispute game
	FaultDisputeGameType
	// ValidityDisputeGameType is the uint8 enum value for the validity dispute game
	ValidityDisputeGameType
)

// Valid returns true if the game type is within the valid range.
func (g GameType) Valid() bool {
	return g >= AttestationDisputeGameType && g <= ValidityDisputeGameType
}

// DisputeGameTypes is a list of dispute game types.
var DisputeGameTypes = []string{"attestation", "fault", "validity"}

// DisputeGameType is a custom flag type for dispute game type.
type DisputeGameType struct {
	Enum     []string
	Default  string
	selected string
}

// Set sets the dispute game type.
func (d *DisputeGameType) Set(value string) error {
	for _, enum := range d.Enum {
		if enum == value {
			d.selected = value
			return nil
		}
	}

	return fmt.Errorf("allowed values are %s", strings.Join(d.Enum, ", "))
}

// String returns the selected dispute game type.
func (d DisputeGameType) String() string {
	if d.selected == "" {
		return d.Default
	}
	return d.selected
}

// Type maps the [DisputeGameType] string value to a [GameType] enum value.
func (d DisputeGameType) Type() GameType {
	if d.selected == DisputeGameTypes[0] {
		return AttestationDisputeGameType
	} else if d.selected == DisputeGameTypes[1] {
		return FaultDisputeGameType
	} else if d.selected == DisputeGameTypes[2] {
		return ValidityDisputeGameType
	} else {
		return DefaultGameType()
	}
}

var (
	// Required Flags
	L1EthRpcFlag = cli.StringFlag{
		Name:   "l1-eth-rpc",
		Usage:  "HTTP provider URL for L1.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "L1_ETH_RPC"),
	}
	RollupRpcFlag = cli.StringFlag{
		Name:   "rollup-rpc",
		Usage:  "HTTP provider URL for the rollup node.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "ROLLUP_RPC"),
	}
	L2OOAddressFlag = cli.StringFlag{
		Name:   "l2oo-address",
		Usage:  "Address of the L2OutputOracle contract.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "L2OO_ADDRESS"),
	}
	DGFAddressFlag = cli.StringFlag{
		Name:   "dgf-address",
		Usage:  "Address of the DisputeGameFactory contract.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "DGF_ADDRESS"),
	}
	// Optional Flags
	DisputeGameTypeFlag = cli.GenericFlag{
		Name: "dispute-game-type",
		Value: &DisputeGameType{
			Enum:    DisputeGameTypes,
			Default: "attestation",
		},
		Usage:  "Type of dispute game: attestation, fault, or validity.",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "DISPUTE_GAME_TYPE"),
	}
)

// requiredFlags are checked by [CheckRequired]
var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	RollupRpcFlag,
	L2OOAddressFlag,
	DGFAddressFlag,
}

// optionalFlags is a list of unchecked cli flags
var optionalFlags = []cli.Flag{
	DisputeGameTypeFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, txmgr.CLIFlags(envVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.GlobalIsSet(f.GetName()) {
			return fmt.Errorf("flag %s is required", f.GetName())
		}
	}
	return nil
}
