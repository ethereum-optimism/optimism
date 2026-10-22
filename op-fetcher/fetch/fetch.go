package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-fetcher/chaininfo"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

func FetchChainInfoCLI() func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		outputFile := cliCtx.String(OutputFileFlag.Name)
		systemConfigProxy := common.HexToAddress(cliCtx.String(SystemConfigProxyFlag.Name))
		l1StandardBridge := common.HexToAddress(cliCtx.String(L1StandardBridgeProxyFlag.Name))
		l1RPCURL := cliCtx.String(L1RPCURLFlag.Name)

		logCfg := logcli.ReadCLIConfig(cliCtx)
		lgr := logcli.NewLogger(logcli.AppOut(cliCtx), logCfg)

		fetcher, err := NewFetcher(lgr, l1RPCURL, systemConfigProxy, l1StandardBridge)
		if err != nil {
			return err
		}

		result, err := fetcher.FetchChainInfo(cliCtx.Context)
		if err != nil {
			return err
		}

		fileData := result.Config()
		if outputFile == "" {
			// Write to stdout when no output file is specified
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(fileData); err != nil {
				return fmt.Errorf("failed to write output to stdout: %w", err)
			}
		} else {
			// Write to the specified file
			jsonData, err := json.MarshalIndent(fileData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal output: %w", err)
			}

			err = os.WriteFile(outputFile, jsonData, 0o644)
			if err != nil {
				return fmt.Errorf("failed to write output to file: %w", err)
			}

			fetcher.lgr.Info("completed fetching chain info", "outputFile", outputFile)
		}
		return nil
	}
}

type Fetcher struct {
	L1RPCURL              string
	SystemConfigProxy     common.Address
	L1StandardBridgeProxy common.Address
	lgr                   log.Logger
}

func NewFetcher(lgr log.Logger, l1RPCURL string, systemConfigProxy, l1StandardBridge common.Address) (*Fetcher, error) {
	return &Fetcher{
		L1RPCURL:              l1RPCURL,
		SystemConfigProxy:     systemConfigProxy,
		L1StandardBridgeProxy: l1StandardBridge,
		lgr:                   lgr,
	}, nil
}

// FetchChainInfo reads the chain's L1 contracts, roles and fault proof status at the
// latest L1 block.
func (f *Fetcher) FetchChainInfo(ctx context.Context) (chaininfo.Info, error) {
	client, err := ethclient.DialContext(ctx, f.L1RPCURL)
	if err != nil {
		return chaininfo.Info{}, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	l1, head, err := newRPCL1State(ctx, client, nil)
	if err != nil {
		return chaininfo.Info{}, err
	}
	f.lgr.Info("fetching chain info", "systemConfigProxy", f.SystemConfigProxy, "l1Block", head.Number, "l1BlockHash", head.Hash())

	out, err := fetchChainInfo(ctx, l1, f.SystemConfigProxy, f.L1StandardBridgeProxy)
	if err != nil {
		return chaininfo.Info{}, fmt.Errorf("failed to fetch chain info of SystemConfig %s at L1 block %v (%s): %w",
			f.SystemConfigProxy, head.Number, head.Hash(), err)
	}
	return out, nil
}
