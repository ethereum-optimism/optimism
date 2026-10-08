package sysgo

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
)

// checkL2ToL2MessageExpiryPeriod checks that a configured L2ToL2CrossDomainMessenger expiry period
// keeps an expired message unrelayable: it must exceed the dependency set's effective message
// expiry window, and it only takes effect when interop is active at genesis, since a later
// activation installs the production period.
func checkL2ToL2MessageExpiryPeriod(cfg PresetConfig, enableInterop bool, delaySeconds uint64) error {
	if cfg.L2ToL2MessageExpiryPeriod == 0 {
		return nil
	}
	if !enableInterop || delaySeconds != 0 {
		return fmt.Errorf("an L2ToL2 message expiry period needs interop at genesis: a later activation installs the production period")
	}
	// An unset or zero window override means the protocol's window.
	window := depset.MessageExpiryTimeSecondsInterop
	if cfg.MessageExpiryWindow != nil && *cfg.MessageExpiryWindow != 0 {
		window = *cfg.MessageExpiryWindow
	}
	if cfg.L2ToL2MessageExpiryPeriod <= window {
		return fmt.Errorf("the L2ToL2 message expiry period %ds must exceed the message expiry window %ds, or an expired message could still be relayed",
			cfg.L2ToL2MessageExpiryPeriod, window)
	}
	return nil
}

// withL2ToL2MessageExpiryPeriod sets the L2ToL2CrossDomainMessenger's expiry period, in seconds, in
// every L2 genesis. The runtime applies it after checkL2ToL2MessageExpiryPeriod, last among the
// deployer options.
func withL2ToL2MessageExpiryPeriod(seconds uint64) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		builder.WithGlobalOverride("l2ToL2MessageExpiryPeriod", seconds)
	}
}
