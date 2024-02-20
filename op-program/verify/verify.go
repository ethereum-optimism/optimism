package verify

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-program/host"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

func Run(l1RpcUrl string, l1RpcKind string, l2RpcUrl string, l2OracleAddr common.Address, dataDir string, network string, chainCfg *params.ChainConfig) error {
	ctx := context.Background()
	logger := oplog.DefaultCLIConfig()
	logger.Level = log.LevelDebug

	setupLog := oplog.NewLogger(os.Stderr, logger)
	l1Client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, setupLog, l1RpcUrl)
	if err != nil {
		return fmt.Errorf("dial L1 client: %w", err)
	}
	l2Client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, setupLog, l2RpcUrl)
	if err != nil {
		return fmt.Errorf("dial L2 client: %w", err)
	}
	outputOracle, err := bindings.NewL2OutputOracle(l2OracleAddr, l1Client)
	if err != nil {
		return fmt.Errorf("create output oracle bindings: %w", err)
	}

	// Find L1 finalized block. Can't be re-orged.
	l1BlockNum := big.NewInt(int64(rpc.FinalizedBlockNumber))
	l1HeadBlock, err := retryOp(ctx, func() (*types.Block, error) {
		return l1Client.BlockByNumber(ctx, l1BlockNum)
	})
	if err != nil {
		return fmt.Errorf("find L1 head: %w", err)
	}
	fmt.Printf("Found l1 head block number: %v hash: %v\n", l1HeadBlock.NumberU64(), l1HeadBlock.Hash())

	l1CallOpts := &bind.CallOpts{Context: ctx, BlockNumber: l1BlockNum}

	// Find the latest output root published in this finalized block
	latestOutputIndex, err := retryOp(ctx, func() (*big.Int, error) {
		return outputOracle.LatestOutputIndex(l1CallOpts)
	})
	if err != nil {
		return fmt.Errorf("fetch latest output index: %w", err)
	}
	output, err := retryOp(ctx, func() (bindings.TypesOutputProposal, error) {
		return outputOracle.GetL2Output(l1CallOpts, latestOutputIndex)
	})
	if err != nil {
		return fmt.Errorf("fetch l2 output %v: %w", latestOutputIndex, err)
	}

	// Use the previous output as the agreed starting point
	agreedOutput, err := retryOp(ctx, func() (bindings.TypesOutputProposal, error) {
		return outputOracle.GetL2Output(l1CallOpts, new(big.Int).Sub(latestOutputIndex, common.Big1))
	})
	if err != nil {
		return fmt.Errorf("fetch l2 output before %v: %w", latestOutputIndex, err)
	}
	l2BlockAtOutput, err := retryOp(ctx, func() (*types.Block, error) { return l2Client.BlockByNumber(ctx, agreedOutput.L2BlockNumber) })
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
		"--network", network,
		"--exec", "./bin/op-program-client",
		"--datadir", dataDir,
		"--l1.head", l1Head.Hex(),
		"--l2.head", l2Head.Hex(),
		"--l2.outputroot", common.Bytes2Hex(agreedOutput.OutputRoot[:]),
		"--l2.claim", l2Claim.Hex(),
		"--l2.blocknumber", l2BlockNumber.String(),
	}
	argsStr := strings.Join(args, " ")
	// args.txt is used by the verify job for offline verification in CI
	if err := os.WriteFile(filepath.Join(dataDir, "args.txt"), []byte(argsStr), 0644); err != nil {
		fmt.Printf("Could not write args: %v", err)
		os.Exit(1)
	}
	fmt.Printf("Configuration: %s\n", argsStr)

	rollupCfg, err := rollup.LoadOPStackRollupConfig(chainCfg.ChainID.Uint64())
	if err != nil {
		return fmt.Errorf("failed to load rollup config: %w", err)
	}
	offlineCfg := config.Config{
		Rollup:             rollupCfg,
		DataDir:            dataDir,
		L2ChainConfig:      chainCfg,
		L2Head:             l2Head,
		L2OutputRoot:       agreedOutput.OutputRoot,
		L2Claim:            l2Claim,
		L2ClaimBlockNumber: l2BlockNumber.Uint64(),
		L1Head:             l1Head,
	}
	onlineCfg := offlineCfg
	onlineCfg.L1URL = l1RpcUrl
	onlineCfg.L2URL = l2RpcUrl
	onlineCfg.L1RPCKind = sources.RPCProviderKind(l1RpcKind)

	fmt.Println("Running in online mode")
	err = host.Main(oplog.NewLogger(os.Stderr, logger), &onlineCfg)
	if err != nil {
		return fmt.Errorf("online mode failed: %w", err)
	}

	fmt.Println("Running in offline mode")
	err = host.Main(oplog.NewLogger(os.Stderr, logger), &offlineCfg)
	if err != nil {
		return fmt.Errorf("offline mode failed: %w", err)
	}
	return nil
}

func retryOp[T any](ctx context.Context, op func() (T, error)) (T, error) {
	return retry.Do(ctx, 10, retry.Fixed(time.Second*2), op)
}
