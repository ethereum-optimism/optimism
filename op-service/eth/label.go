package eth

import "fmt"

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

// CheckConfirmationLabel returns an error unless label is one that a confirmation depth can be counted from:
// latest, safe or finalized. An empty label means latest.
func CheckConfirmationLabel(label BlockLabel) error {
	switch label {
	case "", Unsafe, Safe, Finalized:
		return nil
	default:
		return fmt.Errorf("block label %q is not one of %q, %q or %q", label, Unsafe, Safe, Finalized)
	}
}
