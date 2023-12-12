package dial

import (
	"testing"
)

// TestActiveSequencerFailoverBehavior tests the behavior of the ActiveSequencerProvider when the active sequencer fails
func TestActiveSequencerFailoverBehavior(t *testing.T) {
	// set up a few mock ethclients
	// set up a few mock rollup clients
	// make the first mock rollup client return Active, then Inactive
	// make the second mock rollup client return Inactive, then Active
	// make ActiveL2EndpointProvider, probably manually
	// ask it for a rollup client, should get the first one
	// ask it for a rollup client, should get the second one
}
