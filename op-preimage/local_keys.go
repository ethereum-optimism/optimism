package preimage

// Local key indices for the fault-proof program's bootstrap inputs. These are
// provided by the host via the local preimage key space and read by the client
// program during boot. The values are part of the fault-proof ABI and must not
// change.
const (
	L1HeadLocalIndex LocalIndexKey = iota + 1
	L2OutputRootLocalIndex
	L2ClaimLocalIndex
	L2ClaimBlockNumberLocalIndex
	L2ChainIDLocalIndex

	// These local keys are only used for custom chains
	L2ChainConfigLocalIndex
	RollupConfigLocalIndex
	DependencySetLocalIndex
	L1ChainConfigLocalIndex
)
