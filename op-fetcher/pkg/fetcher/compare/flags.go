package compare

import (
	"github.com/urfave/cli/v2"
)

var (
	FetchOutputDirFlag = &cli.StringFlag{
		Name:     "fetch-output-dir",
		Usage:    "directory containing fetch output files",
		Required: true,
	}
	AddressesFileFlag = &cli.StringFlag{
		Name:     "addresses-file",
		Usage:    "json file containing map of chainId to addresses for all chains",
		Required: true,
	}
)

var CompareFlags = []cli.Flag{
	FetchOutputDirFlag,
	AddressesFileFlag,
}
