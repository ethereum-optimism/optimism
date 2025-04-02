package sysgo

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

type Orchestrator struct {
	t   stack.T
	log log.Logger

	keys devkeys.Keys

	// nil if no time travel is supported
	timeTravelClock *clock.AdvancingClock

	l1Nets      locks.RWMap[stack.L1NetworkID, *L1Network]
	l2Nets      locks.RWMap[stack.L2NetworkID, *L2Network]
	l1ELs       locks.RWMap[stack.L1ELNodeID, *L1ELNode]
	l1CLs       locks.RWMap[stack.L1CLNodeID, *L1CLNode]
	l2ELs       locks.RWMap[stack.L2ELNodeID, *L2ELNode]
	l2CLs       locks.RWMap[stack.L2CLNodeID, *L2CLNode]
	supervisors locks.RWMap[stack.SupervisorID, *Supervisor]
	batchers    locks.RWMap[stack.L2BatcherID, *L2Batcher]
	//challengers locks.RWMap[stack.L2ChallengerID, *L2Challenger] // TODO(#15057): op-challenger support
	proposers locks.RWMap[stack.L2ProposerID, *L2Proposer]

	jwtPath     string
	jwtSecret   [32]byte
	jwtPathOnce sync.Once
}

func NewOrchestrator(t stack.T, log log.Logger) *Orchestrator {
	return &Orchestrator{t: t, log: log}
}

func (o *Orchestrator) T() stack.T {
	return o.t
}

func (o *Orchestrator) Log() log.Logger {
	return o.log
}

func (o *Orchestrator) writeDefaultJWT() (jwtPath string, secret [32]byte) {
	o.jwtPathOnce.Do(func() {
		// Sadly the geth node config cannot load JWT secret from memory, it has to be a file
		o.jwtPath = filepath.Join(o.t.TempDir(), "jwt_secret")
		o.jwtSecret = [32]byte{123}
		err := os.WriteFile(o.jwtPath, []byte(hexutil.Encode(o.jwtSecret[:])), 0o600)
		require.NoError(o.t, err, "failed to prepare jwt file")
	})
	return o.jwtPath, o.jwtSecret
}
