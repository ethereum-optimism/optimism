package presets

import "github.com/ethereum-optimism/optimism/op-devstack/devtest"

// TestSetup is a function that initializes a desired presentation of the system
type TestSetup[V any] func(t devtest.T) V
