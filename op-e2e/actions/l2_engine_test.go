package actions

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

func TestL2EngineAPI(gt *testing.T) {
	t := NewDefaultTesting(gt)
	jwtPath := e2eutils.WriteDefaultJWT(t)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	genesisBlock := sd.L2Cfg.ToBlock()
	consensus := beacon.New(ethash.NewFaker())
	db := rawdb.NewMemoryDatabase()
	sd.L2Cfg.MustCommit(db)

	engine := NewL2Engine(log, sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath)

	l2Cl, err := sources.NewEngineClient(engine.RPCClient(), log, nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
	require.NoError(t, err)

	// build an empty block
	chainA, _ := core.GenerateChain(sd.L2Cfg.Config, genesisBlock, consensus, db, 1, func(i int, gen *core.BlockGen) {
		gen.SetCoinbase(common.Address{'A'})
	})
	payloadA, err := eth.BlockAsPayload(chainA[0])
	require.NoError(t, err)

	// apply the payload
	status, err := l2Cl.NewPayload(t.Ctx(), payloadA)
	require.NoError(t, err)
	require.Equal(t, status.Status, eth.ExecutionValid)
	require.Equal(t, genesisBlock.Hash(), engine.l2Chain.CurrentBlock().Hash(), "processed payloads are not immediately canonical")

	// recognize the payload as canonical
	fcRes, err := l2Cl.ForkchoiceUpdate(t.Ctx(), &eth.ForkchoiceState{
		HeadBlockHash:      payloadA.BlockHash,
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, nil)
	require.NoError(t, err)
	require.Equal(t, fcRes.PayloadStatus.Status, eth.ExecutionValid)
	require.Equal(t, payloadA.BlockHash, engine.l2Chain.CurrentBlock().Hash(), "now payload A is canonical")

	// build an alternative block
	chainB, _ := core.GenerateChain(sd.L2Cfg.Config, genesisBlock, consensus, db, 1, func(i int, gen *core.BlockGen) {
		gen.SetCoinbase(common.Address{'B'})
	})
	payloadB, err := eth.BlockAsPayload(chainB[0])
	require.NoError(t, err)

	// apply the payload
	status, err = l2Cl.NewPayload(t.Ctx(), payloadB)
	require.NoError(t, err)
	require.Equal(t, status.Status, eth.ExecutionValid)
	require.Equal(t, payloadA.BlockHash, engine.l2Chain.CurrentBlock().Hash(), "processed payloads are not immediately canonical")

	// reorg block A in favor of block B
	fcRes, err = l2Cl.ForkchoiceUpdate(t.Ctx(), &eth.ForkchoiceState{
		HeadBlockHash:      payloadB.BlockHash,
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, nil)
	require.NoError(t, err)
	require.Equal(t, fcRes.PayloadStatus.Status, eth.ExecutionValid)
	require.Equal(t, payloadB.BlockHash, engine.l2Chain.CurrentBlock().Hash(), "now payload B is canonical")
}

func TestL2EngineAPIFail(gt *testing.T) {
	t := NewDefaultTesting(gt)
	jwtPath := e2eutils.WriteDefaultJWT(t)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	engine := NewL2Engine(log, sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath)
	// mock an RPC failure
	engine.ActL2RPCFail(t)
	// check RPC failure
	l2Cl, err := sources.NewL2Client(engine.RPCClient(), log, nil, sources.L2ClientDefaultConfig(sd.RollupCfg, false))
	require.NoError(t, err)
	_, err = l2Cl.InfoByLabel(t.Ctx(), eth.Unsafe)
	require.ErrorContains(t, err, "mock")
	head, err := l2Cl.InfoByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	require.Equal(gt, sd.L2Cfg.ToBlock().Hash(), head.Hash(), "expecting engine to start at genesis")
}
