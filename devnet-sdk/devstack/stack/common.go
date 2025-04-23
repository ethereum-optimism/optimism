package stack

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
)

type Common interface {
	T() devtest.T
	Logger() log.Logger

	// Label retrieves a label by key.
	// If the label does not exist, it returns an empty string.
	Label(key string) string

	// SetLabel sets a label by key.
	// Note that labels added by tests are not visible to other tests against the same backend.
	SetLabel(key, value string)
}
