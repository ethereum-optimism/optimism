package cmd

import (
	"debug/elf"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v2"

	"cannon/mipsevm"
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
		Usage:    "Output path to write JSON state to. State is dumped to stdout if set to empty string.",
		Value:    "state.json",
		Required: false,
	}
)

func LoadELF(ctx *cli.Context) error {
	elfPath := ctx.Path(LoadELFPathFlag.Name)
	elfProgram, err := elf.Open(elfPath)
	if err != nil {
		return fmt.Errorf("failed to open ELF file %q: %w", elfPath, err)
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
	p := ctx.Path(LoadELFOutFlag.Name)
	var out io.Writer
	if p != "" {
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("failed to open output file: %w", err)
		}
		defer f.Close()
		out = f
	} else {
		out = os.Stdout
	}
	enc := json.NewEncoder(out)
	if err := enc.Encode(state); err != nil {
		return fmt.Errorf("failed to encode state to JSON: %w", err)
	}
	return nil
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
	},
}
