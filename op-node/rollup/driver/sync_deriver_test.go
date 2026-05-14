package driver

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type discardDriverEmitter struct{}

func (discardDriverEmitter) Emit(context.Context, event.Event) {}

func TestELSyncUnsafePayloadRetriesTemporaryInsertFailure(t *testing.T) {
	ctx := context.Background()
	logger := testlog.Logger(t, log.LevelError)
	cfg, genesis := testELSyncRollupConfig()
	syncCfg := &sync.Config{SyncMode: sync.ELSync}
	mockEngine := &testutils.MockEngine{}

	ec := engine.NewEngineController(ctx, mockEngine, logger, metrics.NoopMetrics, cfg, syncCfg, false, &testutils.MockL1Source{}, discardDriverEmitter{}, nil)
	ec.SetUnsafeHead(genesis)
	ec.SetLocalSafeHead(genesis)
	ec.SetFinalizedHead(genesis)

	sd := &SyncDeriver{
		Engine:  ec,
		SyncCfg: syncCfg,
		Config:  cfg,
		Ctx:     ctx,
		Log:     logger,
	}
	ec.SyncDeriver = sd

	ref1, payload1 := testELSyncPayload(t, cfg, genesis)
	ref2, payload2 := testELSyncPayload(t, cfg, ref1)
	ref3, payload3 := testELSyncPayload(t, cfg, ref2)
	tempErr := derive.NewTemporaryError(errors.New("connection refused"))

	mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, genesis, nil)
	mockEngine.ExpectNewPayload(payload1.ExecutionPayload, nil, &eth.PayloadStatusV1{}, tempErr)
	sd.OnUnsafeL2Payload(ctx, payload1)
	require.Equal(t, genesis, ec.UnsafeL2Head())

	mockEngine.ExpectNewPayload(payload1.ExecutionPayload, nil, &eth.PayloadStatusV1{}, tempErr)
	sd.OnUnsafeL2Payload(ctx, payload2)
	require.Equal(t, genesis, ec.UnsafeL2Head())

	expectELSyncInsert(t, mockEngine, genesis, ref1, payload1)
	expectELSyncInsert(t, mockEngine, genesis, ref2, payload2)
	expectELSyncInsert(t, mockEngine, genesis, ref3, payload3)
	sd.OnUnsafeL2Payload(ctx, payload3)

	require.Equal(t, ref3, ec.UnsafeL2Head())
	mockEngine.AssertExpectations(t)
}

func expectELSyncInsert(t *testing.T, mockEngine *testutils.MockEngine, safe eth.L2BlockRef, ref eth.L2BlockRef, payload *eth.ExecutionPayloadEnvelope) {
	t.Helper()
	mockEngine.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, nil)
	mockEngine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
		HeadBlockHash: ref.Hash,
		SafeBlockHash: safe.Hash,
	}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}}, nil)
}

func testELSyncRollupConfig() (*rollup.Config, eth.L2BlockRef) {
	l1Genesis := eth.BlockID{Hash: common.Hash{0x11}, Number: 100}
	l2Genesis := eth.L2BlockRef{
		Hash:       common.Hash{0x22},
		Number:     0,
		Time:       1_000,
		L1Origin:   l1Genesis,
		ParentHash: common.Hash{},
	}
	return &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     l1Genesis,
			L2:     l2Genesis.ID(),
			L2Time: l2Genesis.Time,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.Address{0x42},
				GasLimit:    30_000_000,
			},
		},
		BlockTime:     2,
		SeqWindowSize: 2,
	}, l2Genesis
}

func testELSyncPayload(t *testing.T, cfg *rollup.Config, parent eth.L2BlockRef) (eth.L2BlockRef, *eth.ExecutionPayloadEnvelope) {
	t.Helper()
	number := parent.Number + 1
	timestamp := parent.Time + cfg.BlockTime
	l1Origin := eth.BlockID{Hash: common.Hash{byte(number), 0xaa}, Number: cfg.Genesis.L1.Number + number}
	l1Info := &testutils.MockBlockInfo{
		InfoHash:        l1Origin.Hash,
		InfoParentHash:  cfg.Genesis.L1.Hash,
		InfoNum:         l1Origin.Number,
		InfoTime:        timestamp,
		InfoBaseFee:     big.NewInt(1),
		InfoBlobBaseFee: big.NewInt(1),
		InfoReceiptRoot: gethtypes.EmptyRootHash,
		InfoRoot:        common.Hash{byte(number), 0xbb},
	}
	l1InfoTx, err := derive.L1InfoDepositBytes(cfg, params.MergedTestChainConfig, cfg.Genesis.SystemConfig, parent.SequenceNumber+1, l1Info, timestamp)
	require.NoError(t, err)

	ref := eth.L2BlockRef{
		Hash:           common.Hash{byte(number), 0xcc},
		Number:         number,
		ParentHash:     parent.Hash,
		Time:           timestamp,
		L1Origin:       l1Origin,
		SequenceNumber: parent.SequenceNumber + 1,
	}
	payload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockHash:    ref.Hash,
			ParentHash:   parent.Hash,
			BlockNumber:  hexutil.Uint64(ref.Number),
			Timestamp:    hexutil.Uint64(ref.Time),
			Transactions: []eth.Data{l1InfoTx},
		},
	}
	return ref, payload
}
