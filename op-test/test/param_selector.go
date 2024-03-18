package test

import (
	"context"
	"fmt"
	"slices"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-test/test/flags"
)

type testParameters struct {
	L1Fork []string
	L2Fork []string
}

func paramsFromCLI(cliCtx *cli.Context) *testParameters {
	return &testParameters{
		L1Fork: cliCtx.StringSlice(flags.L1ForksFlag.Name),
		L2Fork: cliCtx.StringSlice(flags.L2ForksFlag.Name),
	}
}

func filterParams(candidates, options []string) (out []string) {
	if slices.Contains(options, "*") {
		return candidates
	}
	for _, el := range candidates {
		if slices.Contains(options, el) {
			out = append(out, el)
		}
	}
	return out
}

func (t *testParameters) Select(name string, options []string) []string {
	switch name {
	case "l1_fork":
		return filterParams(t.L1Fork, options)
	case "l2_fork":
		return filterParams(t.L2Fork, options)
	default:
		fmt.Printf("WARNING: unknown parameter %q\n", name)
		return options
	}
}

var _ ParameterSelector = (*testParameters)(nil)

// We make the ParameterSelector available through the ctx, instead of a global,
// so the parameter-selector logic itself can be overridden and tested
type parameterSelectorCtxKey struct{}

func GetParameterSelector(ctx context.Context) ParameterSelector {
	sel := ctx.Value(parameterSelectorCtxKey{})
	if sel == nil {
		return nil
	}
	v, ok := sel.(ParameterSelector)
	if !ok {
		panic(fmt.Errorf("bad parameter selector: %v", v))
	}
	return v
}
