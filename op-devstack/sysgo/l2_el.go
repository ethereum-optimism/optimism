package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type L2ELNode interface {
	stack.Lifecycle
	UserRPC() string
	EngineRPC() string
	JWTPath() string
}
