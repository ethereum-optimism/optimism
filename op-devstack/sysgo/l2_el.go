package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
)

type L2ELNode interface {
	stack.Lifecycle
	UserRPC() string
	EngineRPC() string
	JWTPath() string
}

// l2EngineKind is the engine kind an op-node is configured for, whichever L2ELNode it drives:
// op-reth, a CLI-superset build of it, or the sync-tester mock, which EL-syncs past a finalized
// block as op-reth does.
const l2EngineKind = engine.Reth
