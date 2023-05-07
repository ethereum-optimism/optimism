package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

const agreedBlockTrailingDistance = 100

func main() {
	if len(os.Args) != 3 {
		_, _ = fmt.Fprintln(os.Stderr, "Must specify L1 RPC URL and L2 RPC URL as arguments")
		os.Exit(2)
	}
	l1RpcUrl := os.Args[1]
	l2RpcUrl := os.Args[2]
	goerliOutputAddress := common.HexToAddress("0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0")
	err := Run(l1RpcUrl, l2RpcUrl, goerliOutputAddress)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed: %v\n", err.Error())
		os.Exit(1)
	}
}

func Run(l1RpcUrl string, l2RpcUrl string, l2OracleAddr common.Address) error {
	ctx := context.Background()
	l1RpcClient, err := rpc.Dial(l1RpcUrl)
	if err != nil {
		return fmt.Errorf("dial L1 client: %w", err)
	}
	l1Client := ethclient.NewClient(l1RpcClient)

	l2RpcClient, err := rpc.Dial(l2RpcUrl)
	if err != nil {
		return fmt.Errorf("dial L2 client: %w", err)
	}
	l2Client := ethclient.NewClient(l2RpcClient)

	outputOracle, err := bindings.NewL2OutputOracle(l2OracleAddr, l1Client)
	if err != nil {
		return fmt.Errorf("create output oracle bindings: %w", err)
	}

	// Find L2 finalized head. This is far enough back that we know it's submitted to L1 and won't be re-orged
	l2FinalizedHead, err := l2Client.BlockByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return fmt.Errorf("get l2 safe head: %w", err)
	}

	// Find L1 finalized block. Can't be re-orged and must contain all batches for the L2 finalized block
	l1BlockNum := big.NewInt(int64(rpc.FinalizedBlockNumber))
	l1HeadBlock, err := l1Client.BlockByNumber(ctx, l1BlockNum)
	if err != nil {
		return fmt.Errorf("find L1 head: %w", err)
	}

	// Get the most published L2 output from before the finalized block
	callOpts := &bind.CallOpts{Context: ctx}
	outputIndex, err := outputOracle.GetL2OutputIndexAfter(callOpts, l2FinalizedHead.Number())
	if err != nil {
		return fmt.Errorf("get output index after finalized block: %w", err)
	}
	outputIndex = outputIndex.Sub(outputIndex, big.NewInt(1))
	output, err := outputOracle.GetL2Output(callOpts, outputIndex)
	if err != nil {
		return fmt.Errorf("retrieve latest output: %w", err)
	}

	l1Head := l1HeadBlock.Hash()
	l2Claim := common.Hash(output.OutputRoot)
	l2BlockNumber := output.L2BlockNumber

	// Use an agreed starting L2 block some distance before the block the output claim is from
	agreedBlockNumber := uint64(0)
	if l2BlockNumber.Uint64() > agreedBlockTrailingDistance {
		agreedBlockNumber = l2BlockNumber.Uint64() - agreedBlockTrailingDistance
	}
	l2AgreedBlock, err := l2Client.BlockByNumber(ctx, big.NewInt(int64(agreedBlockNumber)))
	if err != nil {
		return fmt.Errorf("retrieve agreed l2 block: %w", err)
	}
	l2Head := l2AgreedBlock.Hash()

	temp, err := os.MkdirTemp("", "oracledata")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		err := os.RemoveAll(temp)
		if err != nil {
			println("Failed to remove temp dir:" + err.Error())
		}
	}()
	fmt.Printf("Using temp dir: %s\n", temp)
	args := []string{
		"--network", "goerli",
		"--exec", "./bin/op-program-client",
		"--datadir", temp,
		"--l1.head", l1Head.Hex(),
		"--l2.head", l2Head.Hex(),
		"--l2.claim", l2Claim.Hex(),
		"--l2.blocknumber", l2BlockNumber.String(),
	}
	fmt.Printf("Configuration: %s\n", args)
	fmt.Println("Running in online mode")
	err = runFaultProofProgram(ctx, append(args, "--l1", l1RpcUrl, "--l2", l2RpcUrl))
	if err != nil {
		return fmt.Errorf("online mode failed: %w", err)
	}

	fmt.Println("Running in offline mode")
	err = runFaultProofProgram(ctx, args)
	if err != nil {
		return fmt.Errorf("offline mode failed: %w", err)
	}
	return nil
}

func runFaultProofProgram(ctx context.Context, args []string) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "./bin/op-program", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
