package eth

type BlockLabel string

const (
	// Unsafe is:
	// - L1: absolute head of the chain
	// - L2: absolute head of the chain, not confirmed on L1
	Unsafe = "latest"
	// Safe is:
	// - L1: Justified checkpoint
	// - L2: Derived chain tip from L1 data
	Safe = "safe"
	// Finalized is:
	// - L1: Finalized checkpoint
	// - L2: Derived chain tip from finalized L1 data
	Finalized = "finalized"
)
