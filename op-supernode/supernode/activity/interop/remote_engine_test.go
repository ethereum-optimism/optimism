package interop

import (
	"context"
	"encoding/binary"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/remote"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/remote/remotetest"
)

// encodeExecutingMessageLog builds a CrossL2Inbox ExecutingMessage log identifying the
// given initiating message — the inverse of messages.DecodeExecutingMessageLog. The 160-byte
// data layout is: [12 zero][20 origin] [24 zero][8 blockNumber] [28 zero][4 logIndex]
// [24 zero][8 timestamp] [32 chainID].
func encodeExecutingMessageLog(id messages.Identifier, payloadHash common.Hash) *types.Log {
	data := make([]byte, 32*5)
	copy(data[12:32], id.Origin.Bytes())
	binary.BigEndian.PutUint64(data[56:64], id.BlockNumber)
	binary.BigEndian.PutUint32(data[92:96], id.LogIndex)
	binary.BigEndian.PutUint64(data[120:128], id.Timestamp)
	cid := id.ChainID.Bytes32()
	copy(data[128:160], cid[:])
	return &types.Log{
		Address: params.InteropCrossL2InboxAddress,
		Topics:  []common.Hash{messages.ExecutingMessageEventTopic, payloadHash},
		Data:    data,
	}
}

// TestEngineVerifiesExecutingMessageFromRemoteNode is an engine-level end-to-end test: a
// driven chain emits a real CrossL2Inbox executing-message log that references an
// initiating message served by the mock remote node over HTTP. The log is sealed through
// the production processBlockLogs path (real DecodeExecutingMessageLog) and validated by
// the real verifyInteropMessages engine — proving an actual executing message, sourced by
// a remote node, is accepted, and that a tampered reference is rejected.
func TestEngineVerifiesExecutingMessageFromRemoteNode(t *testing.T) {
	t.Parallel()

	const (
		activation = uint64(1000)
		blockTime  = uint64(1)
	)
	remoteID := eth.ChainIDFromUInt64(8453)
	drivenID := eth.ChainIDFromUInt64(10)

	// Remote node: serve a deterministic chain over HTTP and ingest its block 1 into a real
	// logsDB via the real HTTPAdapter. Block 1's timestamp = activation + blockTime = 1001.
	chain := remotetest.New(remotetest.Config{
		ChainID: remoteID, BlockTime: blockTime, MsgsPerBlock: 1, StartTimestamp: activation,
	})
	srv := httptest.NewServer(chain.Handler())
	defer srv.Close()

	remoteDB, err := openLogsDB(testLogger(), remoteID, t.TempDir())
	require.NoError(t, err)
	defer func() { _ = remoteDB.Close() }()
	node := &remoteNode{
		log:     testLogger(),
		adapter: remote.NewHTTPAdapter(remoteID, srv.URL, srv.Client()),
		db:      remoteDB,
		poll:    defaultRemotePollInterval,
	}
	ingested, err := node.ingestOnce(context.Background())
	require.NoError(t, err)
	require.True(t, ingested)

	const initBlock, initLogIdx = uint64(1), uint32(0)
	initTimestamp := activation + blockTime // 1001

	// The executing message identifies remote block 1, log 0.
	initID := messages.Identifier{
		Origin:      chain.Origin(initBlock, initLogIdx),
		BlockNumber: initBlock,
		LogIndex:    initLogIdx,
		Timestamp:   initTimestamp,
		ChainID:     remoteID,
	}

	// runRound seals a driven block carrying the given executing-message log through the
	// production decode+seal path, runs the real verifyInteropMessages engine, and reports
	// whether the driven chain was flagged invalid.
	runRound := func(t *testing.T, logEntry *types.Log) (Result, bool) {
		t.Helper()
		drivenDB, err := openLogsDB(testLogger(), drivenID, t.TempDir())
		require.NoError(t, err)
		defer func() { _ = drivenDB.Close() }()

		const drivenBlockNum = uint64(5)
		drivenHash := common.HexToHash("0xd0e5")
		l1Block := eth.BlockID{Number: 40, Hash: common.HexToHash("0x1")}
		drivenChain := newMockChainWithL1(drivenID, l1Block, eth.BlockID{Number: drivenBlockNum, Hash: drivenHash})

		i := &Interop{
			log:                 testLogger(),
			ctx:                 context.Background(),
			activationTimestamp: activation,
			messageExpiryWindow: defaultMessageExpiryWindow,
			logsDBs:             map[eth.ChainID]LogsDB{drivenID: drivenDB, remoteID: remoteDB},
			chains:              map[eth.ChainID]cc.InteropChain{drivenID: drivenChain},
			remoteNodes:         map[eth.ChainID]*remoteNode{remoteID: node},
		}

		// Seal the driven block through the production decode+seal path.
		blockInfo := &mockBlockInfo{
			hash:       drivenHash,
			parentHash: common.HexToHash("0xd0e4"),
			number:     drivenBlockNum,
			timestamp:  initTimestamp, // the executing block's timestamp
		}
		require.NoError(t, i.processBlockLogs(drivenDB, blockInfo, types.Receipts{
			&types.Receipt{Logs: []*types.Log{logEntry}},
		}))

		// The log must actually decode to one executing message, so "accepted" below
		// reflects a validated message rather than an empty (vacuously valid) block.
		_, _, execMsgs, err := drivenDB.OpenBlock(drivenBlockNum)
		require.NoError(t, err)
		require.Len(t, execMsgs, 1, "driven block must carry exactly one decoded executing message")

		blocks := map[eth.ChainID]eth.BlockID{drivenID: {Number: drivenBlockNum, Hash: drivenHash}}
		result, err := i.verifyInteropMessages(initTimestamp, blocks, l1HeadsFromMocks(i.chains, blocks), nil)
		require.NoError(t, err)
		_, invalid := result.InvalidHeads[drivenID]
		return result, invalid
	}

	t.Run("valid executing message is accepted", func(t *testing.T) {
		logEntry := encodeExecutingMessageLog(initID, chain.PayloadHash(initBlock, initLogIdx))
		result, invalid := runRound(t, logEntry)
		require.False(t, invalid, "executing message referencing the remote node must be accepted")
		require.Contains(t, result.L2Heads, drivenID)
	})

	t.Run("tampered reference is rejected", func(t *testing.T) {
		// Wrong payload hash → the derived checksum no longer matches what the remote node
		// sealed → conflict → invalid head.
		logEntry := encodeExecutingMessageLog(initID, common.HexToHash("0xbadbad"))
		_, invalid := runRound(t, logEntry)
		require.True(t, invalid, "executing message with a tampered reference must be rejected")
	})
}
