package op_e2e

import "testing"

// [Category: Initial Setup]
// In this test, we test that we can successfully setup a working cluster.
func TestSequencerFailver_SetupCluster(t *testing.T) {
	sys, _ := setupSequencerFailoverTest(t)
	defer sys.Close()
}
