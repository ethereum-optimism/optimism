package helpers

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/client/l2/engineapi"
	"github.com/ethereum-optimism/optimism/op-program/client/l2/engineapi/test"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestL2EngineAPI(gt *testing.T) {
	fmt.Printf("WHAT 000")

	t := NewDefaultTesting(gt)

	ctx := t.Ctx()

	defer func() {
		fmt.Printf("WHAT CANCELLING\n")
	}()

	defer func() {
		fmt.Println("WHAT DEFERRED")
		if x := recover(); x != nil {
			fmt.Printf("WHAT PANICKED 001 %v\n", x)
		}
	}()

	fmt.Printf("WHAT 001\n")

	jwtPath := e2eutils.WriteDefaultJWT(t)
	fmt.Printf("WHAT 002\n")
	dp := e2eutils.MakeDeployParams(t, DefaultRollupTestParams())
	fmt.Printf("WHAT 003\n")
	sd := e2eutils.Setup(t, dp, DefaultAlloc)
	fmt.Printf("WHAT 004\n")
	log := testlog.Logger(t, log.LevelDebug)
	fmt.Printf("WHAT 005\n")

	fmt.Printf("WHAT 005.0\n")
	genesisBlock := sd.L2Cfg.ToBlock()
	fmt.Printf("WHAT 005.1 %v\n", genesisBlock)
	fmt.Printf("WHAT 006\n")
	consensus := beacon.New(ethash.NewFaker())
	fmt.Printf("WHAT 007 %v\n", consensus)
	db := rawdb.NewMemoryDatabase()
	fmt.Printf("WHAT 008\n")
	tdb := triedb.NewDatabase(db, &triedb.Config{HashDB: hashdb.Defaults})
	fmt.Printf("WHAT 009\n")
	sd.L2Cfg.MustCommit(db, tdb)
	fmt.Printf("WHAT 010\n")

	engine := NewL2Engine(t, log.New("role", "engine"), sd.L2Cfg, jwtPath)
	fmt.Printf("WHAT 011")

	l2Cl, err := sources.NewEngineClient(engine.RPCClient(), log.New("role", "rpc"), nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
	fmt.Printf("WHAT 012")
	require.NoError(t, err)

	fmt.Printf("WHAT 013")

	// build an empty block
	chainA, _ := core.GenerateChain(sd.L2Cfg.Config, genesisBlock, consensus, db, 1, func(n int, gen *core.BlockGen) {
		gen.SetCoinbase(common.Address{'A'})
		if sd.L2Cfg.Config.IsCancun(gen.Number(), gen.Timestamp()) {
			root := crypto.Keccak256Hash([]byte("A"), binary.BigEndian.AppendUint64(nil, uint64(n)))
			gen.SetParentBeaconRoot(root)
		}
	})

	fmt.Printf("WHAT 014")

	payloadA, err := eth.BlockAsPayloadEnv(chainA[0], sd.L2Cfg.Config)
	fmt.Printf("WHAT 015")
	require.NoError(t, err)

	fmt.Printf("WHAT 016")

	// apply the payload
	status, err := l2Cl.NewPayload(ctx, payloadA.ExecutionPayload, payloadA.ParentBeaconBlockRoot)
	fmt.Printf("WHAT 017")
	require.NoError(t, err)
	fmt.Printf("WHAT 018")
	require.Equal(t, eth.ExecutionValid, status.Status)
	fmt.Printf("WHAT 019")
	require.Equal(t, genesisBlock.Hash(), engine.l2Chain.CurrentBlock().Hash(), "processed payloads are not immediately canonical")
	fmt.Printf("WHAT 020")

	// recognize the payload as canonical
	fcRes, err := l2Cl.ForkchoiceUpdate(ctx, &eth.ForkchoiceState{
		HeadBlockHash:      payloadA.ExecutionPayload.BlockHash,
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, nil)
	fmt.Printf("WHAT 021")
	require.NoError(t, err)

	fmt.Printf("WHAT 022")

	require.Equal(t, fcRes.PayloadStatus.Status, eth.ExecutionValid)
	fmt.Printf("WHAT 023")
	require.Equal(t, payloadA.ExecutionPayload.BlockHash, engine.l2Chain.CurrentBlock().Hash(), "now payload A is canonical")
	fmt.Printf("WHAT 024")

	// build an alternative block
	chainB, _ := core.GenerateChain(sd.L2Cfg.Config, genesisBlock, consensus, db, 1, func(n int, gen *core.BlockGen) {
		gen.SetCoinbase(common.Address{'B'})
		if sd.L2Cfg.Config.IsCancun(gen.Number(), gen.Timestamp()) {
			root := crypto.Keccak256Hash([]byte("A"), binary.BigEndian.AppendUint64(nil, uint64(n)))
			gen.SetParentBeaconRoot(root)
		}
	})
	fmt.Printf("WHAT 025")

	payloadB, err := eth.BlockAsPayloadEnv(chainB[0], sd.L2Cfg.Config)
	fmt.Printf("WHAT 026")
	require.NoError(t, err)
	fmt.Printf("WHAT 027")

	// apply the payload
	status, err = l2Cl.NewPayload(ctx, payloadB.ExecutionPayload, payloadB.ParentBeaconBlockRoot)
	fmt.Printf("WHAT 028")
	require.NoError(t, err)
	fmt.Printf("WHAT 029")
	require.Equal(t, status.Status, eth.ExecutionValid)
	fmt.Printf("WHAT 030")
	require.Equal(t, payloadA.ExecutionPayload.BlockHash, engine.l2Chain.CurrentBlock().Hash(), "processed payloads are not immediately canonical")
	fmt.Printf("WHAT 031")

	// reorg block A in favor of block B
	fcRes, err = l2Cl.ForkchoiceUpdate(ctx, &eth.ForkchoiceState{
		HeadBlockHash:      payloadB.ExecutionPayload.BlockHash,
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, nil)
	fmt.Printf("WHAT 032")
	require.NoError(t, err)
	fmt.Printf("WHAT 033")
	require.Equal(t, fcRes.PayloadStatus.Status, eth.ExecutionValid)
	fmt.Printf("WHAT 034")
	require.Equal(t, payloadB.ExecutionPayload.BlockHash, engine.l2Chain.CurrentBlock().Hash(), "now payload B is canonical")
	fmt.Printf("WHAT 035")

	t.Fail()

}

func TestL2EngineAPIBlockBuilding(gt *testing.T) {
	t := NewDefaultTesting(gt)
	jwtPath := e2eutils.WriteDefaultJWT(t)
	dp := e2eutils.MakeDeployParams(t, DefaultRollupTestParams())
	sd := e2eutils.Setup(t, dp, DefaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	genesisBlock := sd.L2Cfg.ToBlock()
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, &triedb.Config{HashDB: hashdb.Defaults})
	sd.L2Cfg.MustCommit(db, tdb)

	engine := NewL2Engine(t, log, sd.L2Cfg, jwtPath)
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
		GasFeeCap: new(big.Int).Add(engine.l2Chain.CurrentBlock().BaseFee, big.NewInt(2*params.GWei)),
		Gas:       params.TxGas,
		To:        &dp.Addresses.Bob,
		Value:     e2eutils.Ether(2),
	})
	require.NoError(gt, cl.SendTransaction(t.Ctx(), tx))

	buildBlock := func(includeAlice bool) {
		parent := engine.l2Chain.CurrentBlock()
		l2Cl, err := sources.NewEngineClient(engine.RPCClient(), log, nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
		require.NoError(t, err)

		nextBlockTime := eth.Uint64Quantity(parent.Time) + 2

		var w *types.Withdrawals
		if sd.RollupCfg.IsCanyon(uint64(nextBlockTime)) {
			w = &types.Withdrawals{}
		}

		var parentBeaconBlockRoot *common.Hash
		if sd.RollupCfg.IsEcotone(uint64(nextBlockTime)) {
			parentBeaconBlockRoot = &common.Hash{}
		}

		// Now let's ask the engine to build a block
		fcRes, err := l2Cl.ForkchoiceUpdate(t.Ctx(), &eth.ForkchoiceState{
			HeadBlockHash:      parent.Hash(),
			SafeBlockHash:      genesisBlock.Hash(),
			FinalizedBlockHash: genesisBlock.Hash(),
		}, &eth.PayloadAttributes{
			Timestamp:             nextBlockTime,
			PrevRandao:            eth.Bytes32{},
			SuggestedFeeRecipient: common.Address{'C'},
			Transactions:          nil,
			NoTxPool:              false,
			GasLimit:              (*eth.Uint64Quantity)(&sd.RollupCfg.Genesis.SystemConfig.GasLimit),
			Withdrawals:           w,
			ParentBeaconBlockRoot: parentBeaconBlockRoot,
		})
		require.NoError(t, err)
		require.Equal(t, fcRes.PayloadStatus.Status, eth.ExecutionValid)
		require.NotNil(t, fcRes.PayloadID, "building a block now")

		if includeAlice {
			engine.ActL2IncludeTx(dp.Addresses.Alice)(t)
		}

		envelope, err := l2Cl.GetPayload(t.Ctx(), eth.PayloadInfo{ID: *fcRes.PayloadID, Timestamp: uint64(nextBlockTime)})
		payload := envelope.ExecutionPayload
		require.NoError(t, err)
		require.Equal(t, parent.Hash(), payload.ParentHash, "block builds on parent block")

		// apply the payload
		status, err := l2Cl.NewPayload(t.Ctx(), payload, envelope.ParentBeaconBlockRoot)
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
	require.Zero(t, engine.l2Chain.GetBlockByHash(engine.l2Chain.CurrentBlock().Hash()).Transactions().Len(), "no tx included")
	buildBlock(true)
	require.Equal(gt, 1, engine.l2Chain.GetBlockByHash(engine.l2Chain.CurrentBlock().Hash()).Transactions().Len(), "tx from alice is included")
	buildBlock(false)
	require.Zero(t, engine.l2Chain.GetBlockByHash(engine.l2Chain.CurrentBlock().Hash()).Transactions().Len(), "no tx included")
	require.Equal(t, uint64(3), engine.l2Chain.CurrentBlock().Number.Uint64(), "built 3 blocks")
}

func TestL2EngineAPIFail(gt *testing.T) {
	t := NewDefaultTesting(gt)
	jwtPath := e2eutils.WriteDefaultJWT(t)
	dp := e2eutils.MakeDeployParams(t, DefaultRollupTestParams())
	sd := e2eutils.Setup(t, dp, DefaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	engine := NewL2Engine(t, log, sd.L2Cfg, jwtPath)
	// mock an RPC failure
	mockErr := errors.New("mock L2 RPC error")
	engine.ActL2RPCFail(t, mockErr)
	// check RPC failure
	l2Cl, err := sources.NewL2Client(engine.RPCClient(), log, nil, sources.L2ClientDefaultConfig(sd.RollupCfg, false))
	require.NoError(t, err)
	_, err = l2Cl.InfoByLabel(t.Ctx(), eth.Unsafe)
	require.ErrorIs(t, err, mockErr)
	head, err := l2Cl.InfoByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	require.Equal(gt, sd.L2Cfg.ToBlock().Hash(), head.Hash(), "expecting engine to start at genesis")
}

func TestEngineAPITests(t *testing.T) {
	test.RunEngineAPITests(t, func(t *testing.T) engineapi.EngineBackend {
		jwtPath := e2eutils.WriteDefaultJWT(t)
		dp := e2eutils.MakeDeployParams(t, DefaultRollupTestParams())
		sd := e2eutils.Setup(t, dp, DefaultAlloc)
		n, _, apiBackend := newBackend(t, sd.L2Cfg, jwtPath, nil)
		err := n.Start()
		require.NoError(t, err)
		return apiBackend
	})
}
