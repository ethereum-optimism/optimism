package eth

type BlockLabel string

const (
	// Unsafe is:
	// - L1: absolute head of the chain
	// - L2: absolute head of the chain, not confirmed on L1
	Unsafe = "latest"
	// Safe is:
	// - L1: Justified checkpoint, beacon chain: 1 epoch of 2/3 of the validators attesting the epoch.
	// - L2: Derived chain tip from L1 data
	Safe = "safe"
	// Finalized is:
	// - L1: Finalized checkpoint, beacon chain: 2+ justified epochs with "supermajority link" (see FFG docs).
	//       More about FFG: https://ethereum.org/en/developers/docs/consensus-mechanisms/pos/gasper/
	// - L2: Derived chain tip from finalized L1 data
	Finalized = "finalized"
)

func (label BlockLabel) Arg() any { return string(label) }

func (BlockLabel) CheckID(id BlockID) error {
	return nil
}
