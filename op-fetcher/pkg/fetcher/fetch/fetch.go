package fetch

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/scriptbackend"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

//go:embed forge-artifacts
var forgeArtifacts embed.FS

func FetchChainInfoCLI() func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		outputFile := cliCtx.String(OutputFileFlag.Name)
		systemConfigProxy := common.HexToAddress(cliCtx.String(SystemConfigProxyFlag.Name))
		l1StandardBridge := common.HexToAddress(cliCtx.String(L1StandardBridgeProxyFlag.Name))
		l1RPCURL := cliCtx.String(L1RPCURLFlag.Name)

		logCfg := oplog.ReadCLIConfig(cliCtx)
		lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)

		fetcher, err := NewFetcher(lgr, l1RPCURL, systemConfigProxy, l1StandardBridge)
		if err != nil {
			return err
		}
		engineKind, err := env.ParseScriptEngine(cliCtx.String(ScriptEngineFlag.Name))
		if err != nil {
			return err
		}
		fetcher.ScriptEngine = engineKind

		result, err := fetcher.FetchChainInfo(cliCtx.Context)
		if err != nil {
			return fmt.Errorf("failed to validate: %w", err)
		}

		fileData := script.CreateChainConfig(result)
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
	// ScriptEngine selects the forge-script backend for the forked L1 read. The empty value
	// resolves to the default (rust); set it to env.ScriptEngineGo to fall back to the in-process
	// Go script.Host.
	ScriptEngine env.ScriptEngineKind
	lgr          log.Logger
}

func NewFetcher(lgr log.Logger, l1RPCURL string, systemConfigProxy, l1StandardBridge common.Address) (*Fetcher, error) {
	return &Fetcher{
		L1RPCURL:              l1RPCURL,
		SystemConfigProxy:     systemConfigProxy,
		L1StandardBridgeProxy: l1StandardBridge,
		lgr:                   lgr,
	}, nil
}

func (f *Fetcher) FetchChainInfo(ctx context.Context) (script.FetchChainInfoOutput, error) {
	f.lgr.Info("initializing fetcher", "systemConfigProxy", f.SystemConfigProxy)

	artifactsFS, cleanup, err := extractedArtifacts()
	if err != nil {
		return script.FetchChainInfoOutput{}, fmt.Errorf("failed to extract forge artifacts: %w", err)
	}
	defer cleanup()

	bcaster := broadcaster.NoopBroadcaster()
	deployerAddress := common.Address{0x01}

	fl1, err := scriptbackend.NewForkedL1(
		ctx,
		f.ScriptEngine,
		f.lgr,
		deployerAddress,
		artifactsFS,
		f.L1RPCURL,
		bcaster,
	)
	if err != nil {
		return script.FetchChainInfoOutput{}, fmt.Errorf("failed to create forked script backend: %w", err)
	}
	defer fl1.Close()

	return script.FetchChainInfoWithBackend(fl1.Backend, script.FetchChainInfoInput{
		SystemConfigProxy:     f.SystemConfigProxy,
		L1StandardBridgeProxy: f.L1StandardBridgeProxy,
	})
}

// extractedArtifacts materializes the embedded forge-artifacts onto disk. Both backends read them
// from there: the Rust engine takes an on-disk --artifacts directory, and serving the Go host from
// the same directory keeps the two paths identical.
func extractedArtifacts() (foundry.StatDirFs, func(), error) {
	dir, err := os.MkdirTemp("", "op-fetcher-artifacts-")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create artifacts temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	sub, err := fs.Sub(forgeArtifacts, "forge-artifacts")
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to root embedded artifacts: %w", err)
	}
	if err := os.CopyFS(dir, sub); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to extract embedded artifacts: %w", err)
	}
	dirFS, ok := os.DirFS(dir).(foundry.StatDirFs)
	if !ok {
		cleanup()
		return nil, nil, fmt.Errorf("os.DirFS does not implement StatDirFs")
	}
	return dirFS, cleanup, nil
}
