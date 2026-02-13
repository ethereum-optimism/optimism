package gnosis

import (
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

const EnvVarPrefix = "GNOSIS"

var GlobalFlags = append([]cli.Flag{}, oplog.CLIFlags(EnvVarPrefix)...)

var (
	L1RpcUrlFlag = &cli.StringFlag{
		Name:     "l1-rpc-url",
		Usage:    "L1 RPC URL for network to send tx to",
		Required: true,
	}
	CalldataFlag = &cli.StringFlag{
		Name:  "calldata",
		Usage: "calldata to be sent",
	}
	PrivateKeysFlag = &cli.StringSliceFlag{
		Name:  "private-keys",
		Usage: "comma-separated list of private keys to sign the tx with",
	}
	SafeAddressFlag = &cli.StringFlag{
		Name:  "safe-address",
		Usage: "safe address to be used for the transaction",
	}
	ToAddressFlag = &cli.StringFlag{
		Name:  "to-address",
		Usage: "to address to be used for the transaction",
	}
	OperationFlag = &cli.StringFlag{
		Name:  "operation",
		Usage: "operation to be used for the transaction: 'call' (standard contract call), 'delegate', or 'create' (when deploying a new contract)",
		Value: "call",
	}
)

var SendGnosisTxFlags = []cli.Flag{
	CalldataFlag,
	PrivateKeysFlag,
	SafeAddressFlag,
	L1RpcUrlFlag,
	ToAddressFlag,
	OperationFlag,
}
