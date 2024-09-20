package cmd

import (
	"github.com/urfave/cli/v2"
)

var (
	LoadELFVMTypeFlag = &cli.StringFlag{
		Name:     "type",
		Usage:    "VM type to create state for. Options are 'cannon' (default), 'cannon-mt'",
		Value:    "cannon",
		Required: false,
	}
	LoadELFPathFlag = &cli.PathFlag{
		Name:      "path",
		Usage:     "Path to 32-bit big-endian MIPS ELF file",
		TakesFile: true,
		Required:  true,
	}
	LoadELFOutFlag = &cli.PathFlag{
		Name:     "out",
		Usage:    "Output path to write state to. State is dumped to stdout if set to '-'. Not written if empty. Use file extension '.bin', '.bin.gz', or '.json' for binary, compressed binary, or JSON formats.",
		Value:    "state.json",
		Required: false,
	}
	LoadELFMetaFlag = &cli.PathFlag{
		Name:     "meta",
		Usage:    "Write metadata file, for symbol lookup during program execution. None if empty.",
		Value:    "meta.json",
		Required: false,
	}
)

func CreateLoadELFCommand(action cli.ActionFunc) *cli.Command {
	return &cli.Command{
		Name:        "load-elf",
		Usage:       "Load ELF file into Cannon state",
		Description: "Load ELF file into Cannon state",
		Action:      LoadELF,
		Flags: []cli.Flag{
			LoadELFVMTypeFlag,
			LoadELFPathFlag,
			LoadELFOutFlag,
			LoadELFMetaFlag,
		},
	}
}
