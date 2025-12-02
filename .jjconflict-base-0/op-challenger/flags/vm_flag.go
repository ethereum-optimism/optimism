package flags

import (
	"fmt"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
)

type FlagCreator func(name string, envVars []string, gameTypeInfo string) cli.Flag

// VMFlag defines a set of flags to set a VM specific option. Provides a flag to set the default plus flags to
// override the default on a per VM basis.
type VMFlag struct {
	vms          []gameTypes.GameType
	name         string
	envVarPrefix string
	flagCreator  FlagCreator
}

func NewVMFlag(name string, envVarPrefix string, vms []gameTypes.GameType, flagCreator FlagCreator) *VMFlag {
	return &VMFlag{
		name:         name,
		envVarPrefix: envVarPrefix,
		flagCreator:  flagCreator,
		vms:          vms,
	}
}

func (f *VMFlag) Flags() []cli.Flag {
	flags := make([]cli.Flag, 0, len(f.vms))
	// Default
	defaultEnvVar := opservice.FlagNameToEnvVarName(f.name, f.envVarPrefix)
	flags = append(flags, f.flagCreator(f.name, []string{defaultEnvVar}, ""))
	for _, vm := range f.vms {
		name := f.TraceSpecificFlagName(vm)
		envVar := opservice.FlagNameToEnvVarName(name, f.envVarPrefix)
		flags = append(flags, f.flagCreator(name, []string{envVar}, fmt.Sprintf("(%v game type only)", vm)))
	}
	return flags
}

func (f *VMFlag) DefaultName() string {
	return f.name
}

func (f *VMFlag) IsSet(ctx *cli.Context, vm gameTypes.GameType) bool {
	return ctx.IsSet(f.TraceSpecificFlagName(vm)) || ctx.IsSet(f.name)
}

func (f *VMFlag) String(ctx *cli.Context, vm gameTypes.GameType) string {
	val := ctx.String(f.TraceSpecificFlagName(vm))
	if val == "" {
		val = ctx.String(f.name)
	}
	return val
}

func (f *VMFlag) StringSlice(ctx *cli.Context, vm gameTypes.GameType) []string {
	val := ctx.StringSlice(f.TraceSpecificFlagName(vm))
	if len(val) == 0 {
		val = ctx.StringSlice(f.name)
	}
	return val
}

func (f *VMFlag) SourceFlagName(ctx *cli.Context, vm gameTypes.GameType) string {
	vmFlag := f.TraceSpecificFlagName(vm)
	if ctx.IsSet(vmFlag) {
		return vmFlag
	}
	return f.name
}

func (f *VMFlag) EitherFlagName(vm gameTypes.GameType) string {
	return fmt.Sprintf("%s/%s", f.DefaultName(), f.TraceSpecificFlagName(vm))
}

func (f *VMFlag) TraceSpecificFlagName(vm gameTypes.GameType) string {
	return fmt.Sprintf("%v-%v", vm, f.name)
}
