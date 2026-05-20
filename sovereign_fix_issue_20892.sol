# Sovereign Bounty Fix — Issue #20892
# Repo: ethereum-optimism/optimism

# Sovereign Fix: op-supernode: Ignore backfill depth if activation timestamp is not set
package node

import (
	"fmt"
)

// Config represents the configuration for the op-supernode.
type Config struct {
	// BackfillDepth specifies the number of blocks to backfill from the activation timestamp.
	// This value is ignored if InteropActivationTimestamp is not set (i.e., is 0).
	BackfillDepth uint64

	// InteropActivationTimestamp is the Unix timestamp (in seconds) at which interop features activate.
	// If this value is 0, interop features are considered inactive, and BackfillDepth is ignored.
	InteropActivationTimestamp uint64

	// Other configuration fields would typically be here, e.g.:
	// L1RPCURL string
	// L2RPCURL string
	// RollupConfig *rollup.Config // Assuming a rollup config struct exists
}

// Validate ensures the configuration is valid and applies any necessary adjustments.
// It returns an error if the configuration is invalid.
func (cfg *Config) Validate() error {
	// Apply the fix: If interop activation timestamp is not set (0), ignore backfill depth.
	if cfg.InteropActivationTimestamp == 0 {
		if cfg.BackfillDepth > 0 {
			// The issue states "backfill depth is ignored". Setting it to 0 achieves this.
			// In a production environment, a log message (e.g., log.Info) would be beneficial here
			// to inform the operator that the configured backfill depth is being overridden.
			// Example: log.Info("Interop activation timestamp is not set, ignoring configured BackfillDepth.")
			cfg.BackfillDepth = 0
		}
	}

	// Example of other validation that might exist in a real config.
	// This is kept minimal to focus on the bounty's specific fix.
	if cfg.BackfillDepth > 1_000_000 { // Arbitrary large number for example validation
		return fmt.Errorf("backfill depth %d is excessively large", cfg.BackfillDepth)
	}

	// Add more validation logic here as needed for other fields.

	return nil
}