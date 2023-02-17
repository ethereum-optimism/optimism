package op_e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErigonBuildPath(t *testing.T) {
	binPath := BuildErigon(t)
	require.FileExists(t, binPath)
}

func TestErigonRunner(t *testing.T) {
	er := &ErigonRunner{}
	es := er.Run(t)
	es.Shutdown()
}
