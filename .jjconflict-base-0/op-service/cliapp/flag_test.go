package cliapp

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

type testGeneric string

func (t *testGeneric) Clone() any {
	cpy := *t
	return &cpy
}

func (t *testGeneric) Set(value string) error {
	*t = testGeneric(value)
	return nil
}

func (t *testGeneric) String() string {
	if t == nil {
		return "nil"
	}
	return string(*t)
}

var _ CloneableGeneric = (*testGeneric)(nil)

func TestProtectFlags(t *testing.T) {
	foo := &cli.StringFlag{
		Name:     "foo",
		Value:    "123",
		Required: true,
	}
	barValue := testGeneric("original")
	bar := &cli.GenericFlag{
		Name:     "bar",
		Value:    &barValue,
		Required: true,
	}
	var originalFlags = []cli.Flag{
		foo,
		bar,
	}
	// if we ran with the original flags instead, we see mutation issues:
	//outFlags := originalFlags
	outFlags := ProtectFlags(originalFlags)
	app := &cli.App{
		Name:  "test",
		Flags: outFlags,
		Action: func(ctx *cli.Context) error {
			require.Equal(t, "a", ctx.String(foo.Name))
			require.Equal(t, "changed", ctx.Generic(bar.Name).(*testGeneric).String())
			return nil
		},
	}
	require.NoError(t, app.Run([]string{"test", "--foo=a", "--bar=changed"}))
	// check that the original flag definitions are still untouched
	require.Equal(t, "123", foo.Value)
	require.Equal(t, "original", bar.Value.String())
}
