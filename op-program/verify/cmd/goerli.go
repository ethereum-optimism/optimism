package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func main() {
	var l1RpcUrl string
	var l1RpcKind string
	var l2RpcUrl string
	var dataDir string
	flag.StringVar(&l1RpcUrl, "l1", "", "L1 RPC URL to use")
	flag.StringVar(&l1RpcKind, "l1-rpckind", "alchemy", "L1 RPC kind")
	flag.StringVar(&l2RpcUrl, "l2", "", "L2 RPC URL to use")
	flag.StringVar(&dataDir, "datadir", "",
		"Directory to use for storing pre-images. If not set a temporary directory will be used.")
	flag.Parse()

	if l1RpcUrl == "" || l2RpcUrl == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Must specify --l1 and --l2 RPC URLs")
		os.Exit(2)
	}

	goerliOutputAddress := common.HexToAddress("0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0")
	err := Run(l1RpcUrl, l1RpcKind, l2RpcUrl, goerliOutputAddress, dataDir)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed: %v\n", err.Error())
		os.Exit(1)
	}
}

func Run(l1RpcUrl string, l1RpcKind string, l2RpcUrl string, l2OracleAddr common.Address, dataDir string) error {
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

	// Find L1 finalized block. Can't be re-orged.
	l1BlockNum := big.NewInt(int64(rpc.FinalizedBlockNumber))
	l1HeadBlock, err := l1Client.BlockByNumber(ctx, l1BlockNum)
	if err != nil {
		return fmt.Errorf("find L1 head: %w", err)
	}
	fmt.Printf("Found l1 head block number: %v hash: %v\n", l1HeadBlock.NumberU64(), l1HeadBlock.Hash())

	l1CallOpts := &bind.CallOpts{Context: ctx, BlockNumber: l1BlockNum}

	// Find the latest output root published in this finalized block
	latestOutputIndex, err := outputOracle.LatestOutputIndex(l1CallOpts)
	if err != nil {
		return fmt.Errorf("fetch latest output index: %w", err)
	}
	output, err := outputOracle.GetL2Output(l1CallOpts, latestOutputIndex)
	if err != nil {
		return fmt.Errorf("fetch l2 output %v: %w", latestOutputIndex, err)
	}

	// Use the previous output as the agreed starting point
	agreedOutput, err := outputOracle.GetL2Output(l1CallOpts, new(big.Int).Sub(latestOutputIndex, common.Big1))
	if err != nil {
		return fmt.Errorf("fetch l2 output before %v: %w", latestOutputIndex, err)
	}
	l2BlockAtOutput, err := l2Client.BlockByNumber(ctx, agreedOutput.L2BlockNumber)
	if err != nil {
		return fmt.Errorf("retrieve agreed block: %w", err)
	}

	l2Head := l2BlockAtOutput.Hash()
	l2BlockNumber := output.L2BlockNumber
	l2Claim := common.Hash(output.OutputRoot)
	l1Head := l1HeadBlock.Hash()

	if dataDir == "" {
		dataDir, err = os.MkdirTemp("", "oracledata")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
		defer func() {
			err := os.RemoveAll(dataDir)
			if err != nil {
				fmt.Println("Failed to remove temp dir:" + err.Error())
			}
		}()
	} else {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			fmt.Printf("Could not create data directory %v: %v", dataDir, err)
			os.Exit(1)
		}
	}
	fmt.Printf("Using dir: %s\n", dataDir)
	args := []string{
		"--log.level", "DEBUG",
		"--network", "goerli",
		"--exec", "./bin/op-program-client",
		"--datadir", dataDir,
		"--l1.head", l1Head.Hex(),
		"--l2.head", l2Head.Hex(),
		"--l2.outputroot", common.Bytes2Hex(agreedOutput.OutputRoot[:]),
		"--l2.claim", l2Claim.Hex(),
		"--l2.blocknumber", l2BlockNumber.String(),
	}
	argsStr := strings.Join(args, " ")
	if err := os.WriteFile(filepath.Join(dataDir, "args.txt"), []byte(argsStr), 0644); err != nil {
		fmt.Printf("Could not write args: %v", err)
		os.Exit(1)
	}
	fmt.Printf("Configuration: %s\n", argsStr)
	fmt.Println("Running in online mode")
	err = runFaultProofProgram(ctx, append(args, "--l1", l1RpcUrl, "--l2", l2RpcUrl, "--l1.rpckind", l1RpcKind))
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
