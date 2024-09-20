//go:build cannon32 || cannon64
// +build cannon32 cannon64

package cmd

import (
	"debug/elf"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/cannon/serialize"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
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
	elfPath := ctx.Path(LoadELFPathFlag.Name)
	elfProgram, err := elf.Open(elfPath)
	if err != nil {
		return fmt.Errorf("failed to open ELF file %q: %w", elfPath, err)
	}
	if elfProgram.Machine != elf.EM_MIPS {
		return fmt.Errorf("ELF is not big-endian MIPS R3000, but got %q", elfProgram.Machine.String())
	}

	var createInitialState func(f *elf.File) (mipsevm.FPVMState, error)

	var patcher = program.PatchStack
	if vmType, err := vmTypeFromString(ctx); err != nil {
		return err
	} else if vmType == cannonVMType {
		createInitialState = func(f *elf.File) (mipsevm.FPVMState, error) {
			return program.LoadELF(f, singlethreaded.CreateInitialState)
		}
		patcher = func(state mipsevm.FPVMState) error {
			err := program.PatchGoGC(elfProgram, state)
			if err != nil {
				return err
			}
			return program.PatchStack(state)
		}
	} else if vmType == mtVMType {
		createInitialState = func(f *elf.File) (mipsevm.FPVMState, error) {
			return program.LoadELF(f, multithreaded.CreateInitialState)
		}
	} else {
		return fmt.Errorf("invalid VM type: %q", vmType)
	}

	state, err := createInitialState(elfProgram)
	if err != nil {
		return fmt.Errorf("failed to load ELF data into VM state: %w", err)
	}
	err = patcher(state)
	if err != nil {
		return fmt.Errorf("failed to patch state: %w", err)
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

var LoadELFCommand = CreateLoadELFCommand(LoadELF)
