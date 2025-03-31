package dsl

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
)

type System struct {
	sys stack.System
}

// TODO decide what to expose publicly in DSL

func Hydrate(sys stack.System) *System {
	return &System{
		sys: sys,
	}
}
