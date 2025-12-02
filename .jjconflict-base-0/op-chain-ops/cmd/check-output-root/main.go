package main

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const (
	// L2EthRpcFlagname defines the flag name for the l2 eth RPC endpoint.
	L2EthRpcFlagName = "l2-eth-rpc"
	// BlockNumberFlagName defines the flag name for the L2 block number.
	BlockNumberFlagName = "block-num"
)

// Flags contains the list of configuration options available to the binary.
var Flags = []cli.Flag{
	&cli.StringFlag{
		Name:     L2EthRpcFlagName,
		Usage:    "Required: L2 execution client RPC endpoint (e.g., http://host:port).",
		Required: true,
		EnvVars:  []string{"CHECK_OUTPUT_ROOT_L2_ETH_RPC"},
	},
	&cli.Uint64Flag{
		Name:    BlockNumberFlagName,
		Usage:   "Required: L2 block number to calculate the output root for.",
		EnvVars: []string{"CHECK_OUTPUT_ROOT_BLOCK_NUM"},
	},
}

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Name = "check-output-root"
	app.Usage = "Calculates a output root from an L2 EL endpoint."
	// Combine specific flags with log flags
	app.Flags = append(Flags, oplog.CLIFlags("CHECK_OUTPUT_ROOT")...)

	app.Action = func(c *cli.Context) error {
		ctx := ctxinterrupt.WithCancelOnInterrupt(c.Context)
		rpcUrl := c.String(L2EthRpcFlagName)
		blockNum := c.Uint64(BlockNumberFlagName)
		root, err := CalculateOutputRoot(ctx, rpcUrl, blockNum)
		if err != nil {
			return err
		}
		fmt.Println(root.Hex())
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("Application failed", "err", err)
	}
}

func CalculateOutputRoot(ctx context.Context, rpcUrl string, blockNum uint64) (common.Hash, error) {
	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to connect to L2 RPC endpoint: %w", err)
	}
	header, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNum))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get L2 block header: %w", err)
	}
	// Isthmus assumes WithdrawalsHash is present in the header.
	if header.WithdrawalsHash == nil {
		return common.Hash{}, fmt.Errorf("target block %d (%s) is missing withdrawals hash, required for Isthmus output root calculation",
			header.Number.Uint64(), header.Hash())
	}

	// Construct OutputV0 using StateRoot, WithdrawalsHash (as MessagePasserStorageRoot), and BlockHash
	output := &eth.OutputV0{
		StateRoot:                eth.Bytes32(header.Root),
		MessagePasserStorageRoot: eth.Bytes32(*header.WithdrawalsHash),
		BlockHash:                header.Hash(),
	}

	// Calculate the output root hash
	return common.Hash(eth.OutputRoot(output)), nil
}
