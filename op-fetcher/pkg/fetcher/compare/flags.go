package compare

import (
	"github.com/urfave/cli/v2"
)

var (
	FetchedDir = &cli.StringFlag{
		Name:     "fetched-dir",
		Usage:    "directory containing chain info files created by fetch command",
		Required: true,
	}
	AddressesFileFlag = &cli.StringFlag{
		Name:     "addresses-file",
		Usage:    "json file containing map of chainId to addresses for all chains",
		Required: true,
	}
	ChainListFileFlag = &cli.StringFlag{
		Name:     "chain-list-file",
		Usage:    "json file containing array of chain list entries",
		Required: true,
	}
	OutputDirFlag = &cli.StringFlag{
		Name:  "output-dir",
		Usage: "directory to write comparison results to",
		Value: "./.fetcher/compare_results",
	}
)

var CompareFlags = []cli.Flag{
	FetchedDir,
	AddressesFileFlag,
	ChainListFileFlag,
	OutputDirFlag,
}
