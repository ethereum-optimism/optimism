package cmd

import (
	"debug/elf"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
)

var (
	LoadELFPathFlag = &cli.PathFlag{
		Name:      "path",
		Usage:     "Path to 32-bit big-endian MIPS ELF file",
		TakesFile: true,
		Required:  true,
	}
	LoadELFPatchFlag = &cli.StringSliceFlag{
		Name:     "patch",
		Usage:    "Type of patching to do",
		Value:    cli.NewStringSlice("go", "stack"),
		Required: false,
	}
	LoadELFOutFlag = &cli.PathFlag{
		Name:     "out",
		Usage:    "Output path to write JSON state to. State is dumped to stdout if set to -. Not written if empty.",
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

func LoadELF(ctx *cli.Context) error {
	elfPath := ctx.Path(LoadELFPathFlag.Name)
	elfProgram, err := elf.Open(elfPath)
	if err != nil {
		return fmt.Errorf("failed to open ELF file %q: %w", elfPath, err)
	}
	if elfProgram.Machine != elf.EM_MIPS {
		return fmt.Errorf("ELF is not big-endian MIPS R3000, but got %q", elfProgram.Machine.String())
	}
	state, err := mipsevm.LoadELF(elfProgram)
	if err != nil {
		return fmt.Errorf("failed to load ELF data into VM state: %w", err)
	}
	for _, typ := range ctx.StringSlice(LoadELFPatchFlag.Name) {
		switch typ {
		case "stack":
			err = mipsevm.PatchStack(state)
		case "go":
			err = mipsevm.PatchGo(elfProgram, state)
		default:
			return fmt.Errorf("unrecognized form of patching: %q", typ)
		}
		if err != nil {
			return fmt.Errorf("failed to apply patch %s: %w", typ, err)
		}
	}
	meta, err := mipsevm.MakeMetadata(elfProgram)
	if err != nil {
		return fmt.Errorf("failed to compute program metadata: %w", err)
	}
	if err := writeJSON[*mipsevm.Metadata](ctx.Path(LoadELFMetaFlag.Name), meta); err != nil {
		return fmt.Errorf("failed to output metadata: %w", err)
	}
	return writeJSON[*mipsevm.State](ctx.Path(LoadELFOutFlag.Name), state)
}

var LoadELFCommand = &cli.Command{
	Name:        "load-elf",
	Usage:       "Load ELF file into Cannon JSON state",
	Description: "Load ELF file into Cannon JSON state, optionally patch out functions",
	Action:      LoadELF,
	Flags: []cli.Flag{
		LoadELFPathFlag,
		LoadELFPatchFlag,
		LoadELFOutFlag,
		LoadELFMetaFlag,
	},
}
