package cliapp

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

type CloneableGeneric interface {
	cli.Generic
	Clone() any
}

// ProtectFlags ensures that no flags are safe to Apply() flag sets to without accidental flag-value mutations.
// ProtectFlags panics if any of the flag definitions cannot be protected.
func ProtectFlags(flags []cli.Flag) []cli.Flag {
	out := make([]cli.Flag, 0, len(flags))
	for _, f := range flags {
		fCopy, err := cloneFlag(f)
		if err != nil {
			panic(fmt.Errorf("failed to clone flag %q: %w", f.Names()[0], err))
		}
		out = append(out, fCopy)
	}
	return out
}

func cloneFlag(f cli.Flag) (cli.Flag, error) {
	switch typedFlag := f.(type) {
	case *cli.GenericFlag:
		if genValue, ok := typedFlag.Value.(CloneableGeneric); ok {
			cpy := *typedFlag
			cpyVal, ok := genValue.Clone().(cli.Generic)
			if !ok {
				return nil, fmt.Errorf("cloned Generic value is not Generic: %T", typedFlag)
			}
			cpy.Value = cpyVal
			return &cpy, nil
		} else {
			return nil, fmt.Errorf("cannot clone Generic value: %T", typedFlag)
		}
	case *cli.StringFlag:
		cpy := *typedFlag
		return &cpy, nil
	case *cli.BoolFlag:
		cpy := *typedFlag
		return &cpy, nil
	case *cli.IntFlag:
		cpy := *typedFlag
		return &cpy, nil
	case *cli.UintFlag:
		cpy := *typedFlag
		return &cpy, nil
	case *cli.Uint64Flag:
		cpy := *typedFlag
		return &cpy, nil
	case *cli.Float64Flag:
		cpy := *typedFlag
		return &cpy, nil
	case *cli.DurationFlag:
		cpy := *typedFlag
		return &cpy, nil
	case *cli.PathFlag:
		cpy := *typedFlag
		return &cpy, nil
	case *cli.StringSliceFlag:
		cpy := *typedFlag
		if typedFlag.Value != nil {
			orig := typedFlag.Value.Value()
			vals := make([]string, len(orig))
			copy(vals, orig)
			cpy.Value = cli.NewStringSlice(vals...)
		}
		return &cpy, nil
	case *cli.Uint64SliceFlag:
		cpy := *typedFlag
		if typedFlag.Value != nil {
			orig := typedFlag.Value.Value()
			vals := make([]uint64, len(orig))
			copy(vals, orig)
			cpy.Value = cli.NewUint64Slice(vals...)
		}
		return &cpy, nil
	case *cli.IntSliceFlag:
		cpy := *typedFlag
		if typedFlag.Value != nil {
			orig := typedFlag.Value.Value()
			vals := make([]int, len(orig))
			copy(vals, orig)
			cpy.Value = cli.NewIntSlice(vals...)
		}
		return &cpy, nil
	default:
		return f, nil
	}
}
