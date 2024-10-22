package inspect

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/urfave/cli/v2"
)

var versionSelector = []byte{0x54, 0xfd, 0x4d, 0x50}

func L2SemversCLI(cliCtx *cli.Context) error {
	cliCfg, err := readConfig(cliCtx)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cliCtx.Context, time.Minute)
	defer cancel()

	logCfg := oplog.ReadCLIConfig(cliCtx)
	l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(l.Handler())

	globalState, err := pipeline.ReadState(cliCfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read intent: %w", err)
	}
	chainState, err := globalState.Chain(cliCfg.ChainID)
	if err != nil {
		return fmt.Errorf("failed to find chain state: %w", err)
	}

	intent := globalState.AppliedIntent
	if intent == nil {
		return fmt.Errorf("can only run this command following a full apply")
	}
	if chainState.Allocs == nil {
		return fmt.Errorf("chain state does not have allocs")
	}

	artifactsFS, cleanup, err := pipeline.DownloadArtifacts(ctx, intent.L2ContractsLocator, pipeline.LogProgressor(l))
	if err != nil {
		return fmt.Errorf("failed to download L2 artifacts: %w", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			l.Warn("failed to clean up L2 artifacts", "err", err)
		}
	}()

	host, err := pipeline.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		l,
		common.Address{19: 0x01},
		artifactsFS,
		0,
	)
	if err != nil {
		return fmt.Errorf("failed to create script host: %w", err)
	}
	host.ImportState(chainState.Allocs.Data)

	addr := common.Address{19: 0x01}

	type contractToCheck struct {
		Address common.Address
		Name    string
	}

	contractsOutput := make(map[string]string)

	// The gov token and the proxy admin do not have semvers.
	contracts := []contractToCheck{
		{predeploys.L2ToL1MessagePasserAddr, "L2ToL1MessagePasser"},
		{predeploys.DeployerWhitelistAddr, "DeployerWhitelist"},
		{predeploys.WETHAddr, "WETH"},
		{predeploys.L2CrossDomainMessengerAddr, "L2CrossDomainMessenger"},
		{predeploys.L2StandardBridgeAddr, "L2StandardBridge"},
		{predeploys.SequencerFeeVaultAddr, "SequencerFeeVault"},
		{predeploys.OptimismMintableERC20FactoryAddr, "OptimismMintableERC20Factory"},
		{predeploys.L1BlockNumberAddr, "L1BlockNumber"},
		{predeploys.GasPriceOracleAddr, "GasPriceOracle"},
		{predeploys.L1BlockAddr, "L1Block"},
		{predeploys.LegacyMessagePasserAddr, "LegacyMessagePasser"},
		{predeploys.L2ERC721BridgeAddr, "L2ERC721Bridge"},
		{predeploys.OptimismMintableERC721FactoryAddr, "OptimismMintableERC721Factory"},
		{predeploys.BaseFeeVaultAddr, "BaseFeeVault"},
		{predeploys.L1FeeVaultAddr, "L1FeeVault"},
		{predeploys.SchemaRegistryAddr, "SchemaRegistry"},
		{predeploys.EASAddr, "EAS"},
		{predeploys.WETHAddr, "WETH"},
	}
	for _, contract := range contracts {
		data, _, err := host.Call(
			addr,
			contract.Address,
			bytes.Clone(versionSelector),
			1_000_000_000,
			uint256.NewInt(0),
		)
		if err != nil {
			return fmt.Errorf("failed to call version on %s: %w", contract.Name, err)
		}

		// The second 32 bytes contain the length of the string
		length := new(big.Int).SetBytes(data[32:64]).Int64()
		// Start of the string data (after offset and length)
		stringStart := 64
		stringEnd := int64(stringStart) + length

		// Bounds check
		if stringEnd > int64(len(data)) {
			return fmt.Errorf("string data out of bounds")
		}

		contractsOutput[contract.Name] = string(data[stringStart:stringEnd])
	}

	if err := jsonutil.WriteJSON(contractsOutput, ioutil.ToStdOutOrFileOrNoop(cliCfg.Outfile, 0o666)); err != nil {
		return fmt.Errorf("failed to write rollup config: %w", err)
	}

	return nil
}
