package fetch

import (
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

type Fetcher struct {
	L1RPCURL              string
	SystemConfigProxy     common.Address
	L1StandardBridgeProxy common.Address
	OutputFile            string
	lgr                   log.Logger
}

func NewFetcherFromCli(cliCtx *cli.Context) (*Fetcher, error) {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	systemConfigProxy := cliCtx.String(SystemConfigProxyFlag.Name)
	l1StandardBridge := cliCtx.String(L1StandardBridgeProxyFlag.Name)

	outputFile := cliCtx.String(OutputFileFlag.Name)

	return &Fetcher{
		L1RPCURL:              cliCtx.String(L1RPCURLFlag.Name),
		SystemConfigProxy:     common.HexToAddress(systemConfigProxy),
		L1StandardBridgeProxy: common.HexToAddress(l1StandardBridge),
		OutputFile:            outputFile,
		lgr:                   lgr,
	}, nil
}
