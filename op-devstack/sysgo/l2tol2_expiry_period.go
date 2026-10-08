package sysgo

import "fmt"

// checkL2ToL2MessageExpiryPeriod checks that a configured L2ToL2CrossDomainMessenger expiry period
// keeps an expired message unrelayable: it must exceed the dependency set's message expiry window,
// and it only takes effect when interop is active at genesis, since a later activation installs
// the production period.
func checkL2ToL2MessageExpiryPeriod(cfg PresetConfig, enableInterop bool, delaySeconds uint64) error {
	if cfg.L2ToL2MessageExpiryPeriod == 0 {
		return nil
	}
	if !enableInterop || delaySeconds != 0 {
		return fmt.Errorf("an L2ToL2 message expiry period needs interop at genesis: a later activation installs the production period")
	}
	if cfg.MessageExpiryWindow == nil {
		return fmt.Errorf("an L2ToL2 message expiry period needs a message expiry window, so expired messages stay unrelayable")
	}
	if cfg.L2ToL2MessageExpiryPeriod <= *cfg.MessageExpiryWindow {
		return fmt.Errorf("the L2ToL2 message expiry period %ds must exceed the message expiry window %ds, or an expired message could still be relayed",
			cfg.L2ToL2MessageExpiryPeriod, *cfg.MessageExpiryWindow)
	}
	return nil
}
