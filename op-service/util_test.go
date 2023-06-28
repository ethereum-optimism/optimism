package op_service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCLIFlagsToEnvVars(t *testing.T) {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "test",
			EnvVars: []string{"OP_NODE_TEST_VAR"},
		},
		&cli.IntFlag{
			Name: "no env var",
		},
	}
	res := cliFlagsToEnvVars(flags)
	require.Contains(t, res, "OP_NODE_TEST_VAR")
}

func TestValidateEnvVars(t *testing.T) {
	provided := []string{"OP_BATCHER_CONFIG=true", "OP_BATCHER_FAKE=false", "LD_PRELOAD=/lib/fake.so"}
	defined := map[string]struct{}{
		"OP_BATCHER_CONFIG": {},
		"OP_BATCHER_OTHER":  {},
	}
	invalids := validateEnvVars("OP_BATCHER", provided, defined)
	require.ElementsMatch(t, invalids, []string{"OP_BATCHER_FAKE=false"})
}
