package sysgo

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

type Supervisor interface {
	stack.Lifecycle
	UserRPC() string
}
