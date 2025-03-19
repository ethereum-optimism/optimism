package fetch

import (
	"context"
	"embed"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

//go:embed forge-artifacts
var forgeArtifacts embed.FS

func FetchChainInfoCLI() func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		fetcher, err := NewFetcherFromCli(cliCtx)
		if err != nil {
			return err
		}

		results, err := fetcher.FetchChainInfo(cliCtx.Context)
		if err != nil {
			return fmt.Errorf("failed to validate: %w", err)
		}
		for chainId, result := range results {
			if result.Error != nil {
				fetcher.lgr.Error("failed to fetch chain info", "chainId", chainId, "error", result.Error)
				continue
			}
			err = script.WriteChainConfigToFile(fetcher.OutputDir, result.Output, result.ChainName, chainId)
			if err != nil {
				return fmt.Errorf("failed to write chain info for %d: %w", chainId, err)
			}
			fetcher.lgr.Info("wrote chain info to file", "chainName", result.ChainName, "chainId", chainId)
		}
		fetcher.lgr.Info("completed fetching chain info")
		return nil
	}
}

type FetchChainInfoResult struct {
	ChainName string
	ChainId   uint64
	Output    script.FetchChainInfoOutput
	Error     error
}

func (f *Fetcher) FetchChainInfo(ctx context.Context) (map[uint64]FetchChainInfoResult, error) {
	f.lgr.Info("initializing fetcher", "numChains", len(f.ChainConfigs))

	l1RPC, err := rpc.Dial(f.L1RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}

	bcaster := broadcaster.NoopBroadcaster()
	deployerAddress := common.Address{0x01}
	artifactsFS := &foundry.EmbedFS{
		FS:      forgeArtifacts,
		RootDir: "forge-artifacts",
	}

	results := make(map[uint64]FetchChainInfoResult)
	sem := make(chan struct{}, 10) // Limit to 10 concurrent goroutines
	var wg sync.WaitGroup
	resultCh := make(chan FetchChainInfoResult, len(f.ChainConfigs))

	for chainId, chainConfig := range f.ChainConfigs {
		wg.Add(1)

		chainId := chainId
		chainConfig := chainConfig

		l1Host, err := env.DefaultForkedScriptHost(
			ctx,
			bcaster,
			f.lgr,
			deployerAddress,
			artifactsFS,
			l1RPC,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create script host: %w", err)
		}

		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }() // Release semaphore when done

			f.lgr.Info("fetching chain info", "chainName", chainConfig.ChainName, "chainId", chainId)
			scriptOutput, err := script.FetchChainInfo(l1Host, script.FetchChainInfoInput{
				SystemConfigProxy:     chainConfig.Addresses.SystemConfigProxy,
				L1StandardBridgeProxy: chainConfig.Addresses.L1StandardBridgeProxy,
			})
			resultCh <- FetchChainInfoResult{
				ChainName: chainConfig.ChainName,
				ChainId:   chainConfig.ChainId,
				Output:    scriptOutput,
				Error:     err,
			}
		}()
	}

	wg.Wait()
	close(resultCh)

	for result := range resultCh {
		results[result.ChainId] = result
	}

	return results, nil
}
