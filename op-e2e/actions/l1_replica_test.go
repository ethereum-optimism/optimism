package actions

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

var defaultRollupTestParams = &e2eutils.TestParams{
	MaxSequencerDrift:   40,
	SequencerWindowSize: 120,
	ChannelTimeout:      120,
	L1BlockTime:         15,
}

var defaultAlloc = &e2eutils.AllocParams{PrefundTestUsers: true}

// Test if we can mock an RPC failure
func TestL1Replica_ActL1RPCFail(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	replica := NewL1Replica(t, log, sd.L1Cfg)
	t.Cleanup(func() {
		_ = replica.Close()
	})
	// mock an RPC failure
	replica.ActL1RPCFail(t)
	// check RPC failure
	l1Cl, err := sources.NewL1Client(replica.RPCClient(), log, nil, sources.L1ClientDefaultConfig(sd.RollupCfg, false, sources.RPCKindBasic))
	require.NoError(t, err)
	_, err = l1Cl.InfoByLabel(t.Ctx(), eth.Unsafe)
	require.ErrorContains(t, err, "mock")
	head, err := l1Cl.InfoByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	require.Equal(gt, sd.L1Cfg.ToBlock().Hash(), head.Hash(), "expecting replica to start at genesis")
}

// Test if we can make the replica sync an artificial L1 chain, rewind it, and reorg it
func TestL1Replica_ActL1Sync(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	genesisBlock := sd.L1Cfg.ToBlock()
	consensus := beacon.New(ethash.NewFaker())
	db := rawdb.NewMemoryDatabase()
	sd.L1Cfg.MustCommit(db)

	chainA, _ := core.GenerateChain(sd.L1Cfg.Config, genesisBlock, consensus, db, 10, func(n int, g *core.BlockGen) {
		g.SetCoinbase(common.Address{'A'})
	})
	chainA = append(append([]*types.Block{}, genesisBlock), chainA...)
	chainB, _ := core.GenerateChain(sd.L1Cfg.Config, chainA[3], consensus, db, 10, func(n int, g *core.BlockGen) {
		g.SetCoinbase(common.Address{'B'})
	})
	chainB = append(append([]*types.Block{}, chainA[:4]...), chainB...)
	require.NotEqual(t, chainA[9], chainB[9], "need different chains")
	canonL1 := func(blocks []*types.Block) func(num uint64) *types.Block {
		return func(num uint64) *types.Block {
			if num >= uint64(len(blocks)) {
				return nil
			}
			return blocks[num]
		}
	}

	// Enough setup, create the test actor and run the actual actions
	replica1 := NewL1Replica(t, log, sd.L1Cfg)
	t.Cleanup(func() {
		_ = replica1.Close()
	})
	syncFromA := replica1.ActL1Sync(canonL1(chainA))
	// sync canonical chain A
	for replica1.l1Chain.CurrentBlock().Number.Uint64()+1 < uint64(len(chainA)) {
		syncFromA(t)
	}
	require.Equal(t, replica1.l1Chain.CurrentBlock().Hash(), chainA[len(chainA)-1].Hash(), "sync replica1 to head of chain A")
	replica1.ActL1RewindToParent(t)
	require.Equal(t, replica1.l1Chain.CurrentBlock().Hash(), chainA[len(chainA)-2].Hash(), "rewind replica1 to parent of chain A")

	// sync new canonical chain B
	syncFromB := replica1.ActL1Sync(canonL1(chainB))
	for replica1.l1Chain.CurrentBlock().Number.Uint64()+1 < uint64(len(chainB)) {
		syncFromB(t)
	}
	require.Equal(t, replica1.l1Chain.CurrentBlock().Hash(), chainB[len(chainB)-1].Hash(), "sync replica1 to head of chain B")

	// Adding and syncing a new replica
	replica2 := NewL1Replica(t, log, sd.L1Cfg)
	t.Cleanup(func() {
		_ = replica2.Close()
	})
	syncFromOther := replica2.ActL1Sync(replica1.CanonL1Chain())
	for replica2.l1Chain.CurrentBlock().Number.Uint64()+1 < uint64(len(chainB)) {
		syncFromOther(t)
	}
	require.Equal(t, replica2.l1Chain.CurrentBlock().Hash(), chainB[len(chainB)-1].Hash(), "sync replica2 to head of chain B")
}
