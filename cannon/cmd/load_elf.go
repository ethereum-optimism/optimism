package cmd

import (
	"debug/elf"
	"fmt"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/cannon/serialize"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
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

type VMType string

var (
	cannonVMType VMType = "cannon"
	mtVMType     VMType = "cannon-mt"
)

func vmTypeFromString(ctx *cli.Context) (VMType, error) {
	if vmTypeStr := ctx.String(LoadELFVMTypeFlag.Name); vmTypeStr == string(cannonVMType) {
		return cannonVMType, nil
	} else if vmTypeStr == string(mtVMType) {
		return mtVMType, nil
	} else {
		return "", fmt.Errorf("unknown VM type %q", vmTypeStr)
	}
}

func LoadELF(ctx *cli.Context) error {
	var createInitialState func(f *elf.File) (mipsevm.FPVMState, error)

	if vmType, err := vmTypeFromString(ctx); err != nil {
		return err
	} else if vmType == cannonVMType {
		createInitialState = func(f *elf.File) (mipsevm.FPVMState, error) {
			return program.LoadELF(f, singlethreaded.CreateInitialState)
		}
	} else if vmType == mtVMType {
		createInitialState = func(f *elf.File) (mipsevm.FPVMState, error) {
			return program.LoadELF(f, multithreaded.CreateInitialState)
		}
	} else {
		return fmt.Errorf("invalid VM type: %q", vmType)
	}
	elfPath := ctx.Path(LoadELFPathFlag.Name)
	elfProgram, err := elf.Open(elfPath)
	if err != nil {
		return fmt.Errorf("failed to open ELF file %q: %w", elfPath, err)
	}
	if elfProgram.Machine != elf.EM_MIPS {
		return fmt.Errorf("ELF is not big-endian MIPS R3000, but got %q", elfProgram.Machine.String())
	}
	state, err := createInitialState(elfProgram)
	if err != nil {
		return fmt.Errorf("failed to load ELF data into VM state: %w", err)
	}
	for _, typ := range ctx.StringSlice(LoadELFPatchFlag.Name) {
		switch typ {
		case "stack":
			err = program.PatchStack(state)
		case "go":
			err = program.PatchGo(elfProgram, state)
		default:
			return fmt.Errorf("unrecognized form of patching: %q", typ)
		}
		if err != nil {
			return fmt.Errorf("failed to apply patch %s: %w", typ, err)
		}
	}
	meta, err := program.MakeMetadata(elfProgram)
	if err != nil {
		return fmt.Errorf("failed to compute program metadata: %w", err)
	}
	if err := jsonutil.WriteJSON[*program.Metadata](meta, ioutil.ToStdOutOrFileOrNoop(ctx.Path(LoadELFMetaFlag.Name), OutFilePerm)); err != nil {
		return fmt.Errorf("failed to output metadata: %w", err)
	}

	// Ensure the state is written with appropriate version information
	versionedState, err := versions.NewFromState(state)
	if err != nil {
		return fmt.Errorf("failed to create versioned state: %w", err)
	}
	return serialize.Write(ctx.Path(LoadELFOutFlag.Name), versionedState, OutFilePerm)
}

var LoadELFCommand = &cli.Command{
	Name:        "load-elf",
	Usage:       "Load ELF file into Cannon JSON state",
	Description: "Load ELF file into Cannon JSON state, optionally patch out functions",
	Action:      LoadELF,
	Flags: []cli.Flag{
		LoadELFVMTypeFlag,
		LoadELFPathFlag,
		LoadELFPatchFlag,
		LoadELFOutFlag,
		LoadELFMetaFlag,
	},
}
