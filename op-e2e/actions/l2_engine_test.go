package actions

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
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

	engine := NewL2Engine(t, log, sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath)

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

func TestL2EngineAPIBlockBuilding(gt *testing.T) {
	t := NewDefaultTesting(gt)
	jwtPath := e2eutils.WriteDefaultJWT(t)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	genesisBlock := sd.L2Cfg.ToBlock()
	db := rawdb.NewMemoryDatabase()
	sd.L2Cfg.MustCommit(db)

	engine := NewL2Engine(t, log, sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath)
	t.Cleanup(func() {
		_ = engine.Close()
	})

	cl := engine.EthClient()
	signer := types.LatestSigner(sd.L2Cfg.Config)

	// send a tx to the miner
	tx := types.MustSignNewTx(dp.Secrets.Alice, signer, &types.DynamicFeeTx{
		ChainID:   sd.L2Cfg.Config.ChainID,
		Nonce:     0,
		GasTipCap: big.NewInt(2 * params.GWei),
		GasFeeCap: new(big.Int).Add(engine.l2Chain.CurrentBlock().BaseFee(), big.NewInt(2*params.GWei)),
		Gas:       params.TxGas,
		To:        &dp.Addresses.Bob,
		Value:     e2eutils.Ether(2),
	})
	require.NoError(gt, cl.SendTransaction(t.Ctx(), tx))

	buildBlock := func(includeAlice bool) {
		parent := engine.l2Chain.CurrentBlock()
		l2Cl, err := sources.NewEngineClient(engine.RPCClient(), log, nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
		require.NoError(t, err)

		// Now let's ask the engine to build a block
		fcRes, err := l2Cl.ForkchoiceUpdate(t.Ctx(), &eth.ForkchoiceState{
			HeadBlockHash:      parent.Hash(),
			SafeBlockHash:      genesisBlock.Hash(),
			FinalizedBlockHash: genesisBlock.Hash(),
		}, &eth.PayloadAttributes{
			Timestamp:             eth.Uint64Quantity(parent.Time()) + 2,
			PrevRandao:            eth.Bytes32{},
			SuggestedFeeRecipient: common.Address{'C'},
			Transactions:          nil,
			NoTxPool:              false,
			GasLimit:              (*eth.Uint64Quantity)(&sd.RollupCfg.Genesis.SystemConfig.GasLimit),
		})
		require.NoError(t, err)
		require.Equal(t, fcRes.PayloadStatus.Status, eth.ExecutionValid)
		require.NotNil(t, fcRes.PayloadID, "building a block now")

		if includeAlice {
			engine.ActL2IncludeTx(dp.Addresses.Alice)(t)
		}

		payload, err := l2Cl.GetPayload(t.Ctx(), *fcRes.PayloadID)
		require.NoError(t, err)
		require.Equal(t, parent.Hash(), payload.ParentHash, "block builds on parent block")

		// apply the payload
		status, err := l2Cl.NewPayload(t.Ctx(), payload)
		require.NoError(t, err)
		require.Equal(t, status.Status, eth.ExecutionValid)
		require.Equal(t, parent.Hash(), engine.l2Chain.CurrentBlock().Hash(), "processed payloads are not immediately canonical")

		// recognize the payload as canonical
		fcRes, err = l2Cl.ForkchoiceUpdate(t.Ctx(), &eth.ForkchoiceState{
			HeadBlockHash:      payload.BlockHash,
			SafeBlockHash:      genesisBlock.Hash(),
			FinalizedBlockHash: genesisBlock.Hash(),
		}, nil)
		require.NoError(t, err)
		require.Equal(t, fcRes.PayloadStatus.Status, eth.ExecutionValid)
		require.Equal(t, payload.BlockHash, engine.l2Chain.CurrentBlock().Hash(), "now payload is canonical")
	}
	buildBlock(false)
	require.Zero(t, engine.l2Chain.CurrentBlock().Transactions().Len(), "no tx included")
	buildBlock(true)
	require.Equal(gt, 1, engine.l2Chain.CurrentBlock().Transactions().Len(), "tx from alice is included")
	buildBlock(false)
	require.Zero(t, engine.l2Chain.CurrentBlock().Transactions().Len(), "no tx included")
	require.Equal(t, uint64(3), engine.l2Chain.CurrentBlock().NumberU64(), "built 3 blocks")
}

func TestL2EngineAPIFail(gt *testing.T) {
	t := NewDefaultTesting(gt)
	jwtPath := e2eutils.WriteDefaultJWT(t)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	engine := NewL2Engine(t, log, sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath)
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
