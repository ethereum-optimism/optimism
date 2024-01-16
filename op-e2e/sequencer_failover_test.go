package op_e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// [Category: Initial Setup]
// In this test, we test that we can successfully setup a working cluster.
func TestSequencerFailover_SetupCluster(t *testing.T) {
	sys, conductors := setupSequencerFailoverTest(t)
	defer sys.Close()

	require.Equal(t, 3, len(conductors), "Expected 3 conductors")
	for _, con := range conductors {
		require.NotNil(t, con, "Expected conductor to be non-nil")
	}
}
