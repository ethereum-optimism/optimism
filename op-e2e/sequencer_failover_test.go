package op_e2e

import "testing"

func TestSequencerFailver(t *testing.T) {
	sys, _ := setupSequencerFailoverTest(t)
	defer sys.Close()
}
