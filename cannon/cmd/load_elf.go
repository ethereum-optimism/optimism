package cmd

import (
	"debug/elf"
	"fmt"

	program32 "github.com/ethereum-optimism/optimism/cannon/mipsevm32/program"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm64/multithreaded"
	program64 "github.com/ethereum-optimism/optimism/cannon/mipsevm64/program"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm32/singlethreaded"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
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

type Patcher interface {
	LoadELF(f *elf.File) error
	PatchStack() error
	PatchGo(f *elf.File) error
	MakeMetadata(f *elf.File) error
	WriteState(path string) error
	WriteMetadata(path string) error
}

type Patcher64 struct {
	state *multithreaded.State
	meta  *program64.Metadata
}

func (p *Patcher64) LoadELF(f *elf.File) error {
	state, err := program64.LoadELF(f, multithreaded.CreateInitialState)
	if err != nil {
		return err
	}
	p.state = state
	return nil
}

func (p *Patcher64) PatchStack() error {
	return program64.PatchStack(p.state)
}

func (p *Patcher64) PatchGo(f *elf.File) error {
	return program64.PatchGo(f, p.state)
}

func (p *Patcher64) MakeMetadata(f *elf.File) error {
	metadata, err := program64.MakeMetadata(f)
	if err != nil {
		return err
	}
	p.meta = metadata
	return nil
}

func (p *Patcher64) WriteMetadata(path string) error {
	return jsonutil.WriteJSON[*program64.Metadata](path, p.meta, OutFilePerm)
}

func (p *Patcher64) WriteState(path string) error {
	return jsonutil.WriteJSON[*multithreaded.State](path, p.state, OutFilePerm)
}

var _ Patcher = (*Patcher64)(nil)

type Patcher32 struct {
	state *singlethreaded.State
	meta  *program32.Metadata
}

func (p *Patcher32) LoadELF(f *elf.File) error {
	state, err := program32.LoadELF(f, singlethreaded.CreateInitialState)
	if err != nil {
		return err
	}
	p.state = state
	return nil
}

func (p *Patcher32) PatchStack() error {
	return program32.PatchStack(p.state)
}

func (p *Patcher32) PatchGo(f *elf.File) error {
	return program32.PatchGo(f, p.state)
}

func (p *Patcher32) MakeMetadata(f *elf.File) error {
	metadata, err := program32.MakeMetadata(f)
	if err != nil {
		return err
	}
	p.meta = metadata
	return nil
}

func (p *Patcher32) WriteMetadata(path string) error {
	return jsonutil.WriteJSON[*program32.Metadata](path, p.meta, OutFilePerm)
}

func (p *Patcher32) WriteState(path string) error {
	return jsonutil.WriteJSON[*singlethreaded.State](path, p.state, OutFilePerm)
}

var _ Patcher = (*Patcher32)(nil)

func LoadELF(ctx *cli.Context) error {
	var patcher Patcher
	if vmType, err := vmTypeFromString(ctx); err != nil {
		return err
	} else if vmType == cannonVMType {
		patcher = &Patcher32{}
	} else if vmType == mtVMType {
		patcher = &Patcher64{}
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
	err = patcher.LoadELF(elfProgram)
	if err != nil {
		return fmt.Errorf("failed to load ELF data into VM state: %w", err)
	}
	for _, typ := range ctx.StringSlice(LoadELFPatchFlag.Name) {
		switch typ {
		case "stack":
			err = patcher.PatchStack()
		case "go":
			err = patcher.PatchGo(elfProgram)
		default:
			return fmt.Errorf("unrecognized form of patching: %q", typ)
		}
		if err != nil {
			return fmt.Errorf("failed to apply patch %s: %w", typ, err)
		}
	}
	err = patcher.MakeMetadata(elfProgram)
	if err != nil {
		return fmt.Errorf("failed to compute program metadata: %w", err)
	}

	if err := patcher.WriteMetadata(ctx.Path(LoadELFMetaFlag.Name)); err != nil {
		return fmt.Errorf("failed to output metadata: %w", err)
	}
	return patcher.WriteState(ctx.Path(LoadELFOutFlag.Name))
}

var LoadELFCommand = &cli.Command{
	Name:        "load-elf",
	Usage:       "Load ELF file into Cannon JSON state",
	Description: "Load ELF file into Cannon JSON state, optionally patch out functions",
	Action:      LoadELF,
	Flags: []cli.Flag{
		VMTypeFlag,
		LoadELFPathFlag,
		LoadELFPatchFlag,
		LoadELFOutFlag,
		LoadELFMetaFlag,
	},
}
