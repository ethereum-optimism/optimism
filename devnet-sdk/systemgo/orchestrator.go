package systemgo

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

type T interface {
	system2.T
	TempDir() string
	Cleanup(func()) // this orchestrator will shut down its components at the end of the test
}

type Orchestrator struct {
	t T

	keys devkeys.Keys

	// nil if no time travel is supported
	timeTravelClock *clock.AdvancingClock

	l1Nets      locks.RWMap[system2.L1NetworkID, *L1Network]
	l2Nets      locks.RWMap[system2.L2NetworkID, *L2Network]
	l1ELs       locks.RWMap[system2.L1ELNodeID, *L1ELNode]
	l1CLs       locks.RWMap[system2.L1CLNodeID, *L1CLNode]
	l2ELs       locks.RWMap[system2.L2ELNodeID, *L2ELNode]
	l2CLs       locks.RWMap[system2.L2CLNodeID, *L2CLNode]
	supervisors locks.RWMap[system2.SupervisorID, *Supervisor]
	batchers    locks.RWMap[system2.L2BatcherID, *L2Batcher]
	//challengers locks.RWMap[system2.L2ChallengerID, *L2Challenger] // TODO(#15057): op-challenger support
	proposers locks.RWMap[system2.L2ProposerID, *L2Proposer]

	jwtPath     string
	jwtSecret   [32]byte
	jwtPathOnce sync.Once
}

func NewOrchestrator(t T) *Orchestrator {
	return &Orchestrator{t: t}
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
