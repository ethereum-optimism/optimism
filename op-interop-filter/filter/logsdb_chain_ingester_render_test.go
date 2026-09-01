package filter

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// These tests drive the real ingestion path — the production constructor, a real HTTP JSON-RPC
// listener, the real op-service/sources client with verification fully ON — against an ordinary,
// self-consistent private execution client. Nothing about the source is unusual: the header hashes
// to the hash it reports, the receipts derive to the header's receipts root, and the block-level log
// indexes are dense. The only thing under test is what the ingester does with those verified logs
// between the fetch and the logs DB.

const (
	renderChainID   = uint64(901)
	renderBlockNum  = uint64(100)
	renderBlockTime = uint64(1200)
)

var renderPrivateAddr = common.HexToAddress("0x00000000000000000000000000000000000f00d1")

// privateChainFixture is one block of an ordinary L2. Its logs are, in block order:
//
//	tx0: [private]
//	tx1: [SentMessage, private, ExecutingMessage]
//	tx2: []
//
// so the two exported logs sit at RAW block-level indexes 1 and 3, and their RENDERED indexes are 0
// and 1. Those two numberings disagree for both logs, which is what lets the assertions below tell
// the transformed path from the stock one.
type privateChainFixture struct {
	header   *gethTypes.Header
	txs      gethTypes.Transactions
	receipts optypes.Receipts
	hash     common.Hash
	sent     *gethTypes.Log // the export: raw index 1, rendered index 0
	exec     *gethTypes.Log // the import: raw index 3, rendered index 1
}

func newPrivateChainFixture(t *testing.T) *privateChainFixture {
	t.Helper()

	key, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)
	signer := gethTypes.LatestSignerForChainID(new(big.Int).SetUint64(renderChainID))
	mkTx := func(nonce uint64) *gethTypes.Transaction {
		tx, err := gethTypes.SignNewTx(key, signer, &gethTypes.DynamicFeeTx{
			ChainID: new(big.Int).SetUint64(renderChainID), Nonce: nonce,
			GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2),
			Gas: 100_000, To: &renderPrivateAddr, Value: big.NewInt(0),
		})
		require.NoError(t, err)
		return tx
	}

	f := &privateChainFixture{
		sent: renderSentMessageLog(),
		exec: renderExecutingMessageLog(),
		header: &gethTypes.Header{
			ParentHash:      common.Hash{0x99},
			UncleHash:       gethTypes.EmptyUncleHash,
			Root:            common.Hash{0x11},
			Difficulty:      new(big.Int),
			Number:          new(big.Int).SetUint64(renderBlockNum),
			GasLimit:        30_000_000,
			Time:            renderBlockTime,
			BaseFee:         big.NewInt(7),
			WithdrawalsHash: &gethTypes.EmptyWithdrawalsHash,
		},
	}
	groups := [][]*gethTypes.Log{
		{renderPrivateLog(0x01)},
		{f.sent, renderPrivateLog(0x03), f.exec},
		{},
	}

	logIndex := uint(0)
	cumulative := uint64(0)
	for i, group := range groups {
		tx := mkTx(uint64(i))
		f.txs = append(f.txs, tx)
		cumulative += 21_000
		rec := &gethTypes.Receipt{
			Type:              gethTypes.DynamicFeeTxType,
			Status:            gethTypes.ReceiptStatusSuccessful,
			CumulativeGasUsed: cumulative,
			GasUsed:           21_000,
			TxHash:            tx.Hash(),
			TransactionIndex:  uint(i),
			BlockNumber:       new(big.Int).SetUint64(renderBlockNum),
			Logs:              []*gethTypes.Log{},
		}
		for _, l := range group {
			l.BlockNumber = renderBlockNum
			l.TxHash = tx.Hash()
			l.TxIndex = uint(i)
			l.Index = logIndex
			logIndex++
			rec.Logs = append(rec.Logs, l)
		}
		rec.Bloom = gethTypes.CreateBloom(rec)
		f.receipts = append(f.receipts, optypes.FromGethReceipt(rec))
	}

	// A completely ordinary block: every root derives from the data it commits to, and the block
	// hash is the hash of this header. The ingestion client verifies all of it.
	f.header.TxHash = gethTypes.DeriveSha(f.txs, trie.NewStackTrie(nil))
	f.header.ReceiptHash = gethTypes.DeriveSha(f.receipts, trie.NewStackTrie(nil))
	for _, r := range f.receipts {
		for i := range f.header.Bloom {
			f.header.Bloom[i] |= r.Bloom[i]
		}
	}
	f.hash = f.header.Hash()
	for _, r := range f.receipts {
		r.BlockHash = f.hash
		for _, l := range r.Logs {
			l.BlockHash = f.hash
		}
	}
	require.Equal(t, uint(1), f.sent.Index, "the export sits at raw block-level log index 1")
	require.Equal(t, uint(3), f.exec.Index, "the import sits at raw block-level log index 3")
	return f
}

// renderSentMessageLog is a well-formed messenger SentMessage: an exported message.
func renderSentMessageLog() *gethTypes.Log {
	payload := []byte{0xde, 0xad, 0xbe}
	data := make([]byte, 0, 128)
	data = append(data, common.LeftPadBytes(renderPrivateAddr.Bytes(), 32)...)
	data = append(data, common.BigToHash(big.NewInt(64)).Bytes()...)
	data = append(data, common.BigToHash(big.NewInt(int64(len(payload)))).Bytes()...)
	data = append(data, payload...)
	data = append(data, make([]byte, 32-len(payload))...)
	return &gethTypes.Log{
		Address: predeploys.L2toL2CrossDomainMessengerAddr,
		Topics: []common.Hash{
			render.SentMessageEventTopic,
			common.BigToHash(big.NewInt(902)),
			common.BytesToHash(renderPrivateAddr.Bytes()),
			common.BigToHash(big.NewInt(7)),
		},
		Data: data,
	}
}

// renderExecutingMessageLog is a well-formed CrossL2Inbox ExecutingMessage: an imported message,
// which the ingester decodes and stores against its block-level log index.
func renderExecutingMessageLog() *gethTypes.Log {
	id := messages.Identifier{
		Origin:      predeploys.L2toL2CrossDomainMessengerAddr,
		BlockNumber: 10,
		LogIndex:    2,
		Timestamp:   1_199,
		ChainID:     eth.ChainIDFromUInt64(902),
	}
	data := make([]byte, 0, 32*5)
	data = append(data, common.LeftPadBytes(id.Origin.Bytes(), 32)...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(id.BlockNumber)).Bytes()...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(uint64(id.LogIndex))).Bytes()...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(id.Timestamp)).Bytes()...)
	chainID := id.ChainID.Bytes32()
	data = append(data, chainID[:]...)
	return &gethTypes.Log{
		Address: predeploys.CrossL2InboxAddr,
		Topics:  []common.Hash{messages.ExecutingMessageEventTopic, {0x77}},
		Data:    data,
	}
}

// renderPrivateLog is private business: outside the emitter set, so it is never part of a rendering.
func renderPrivateLog(tag byte) *gethTypes.Log {
	return &gethTypes.Log{Address: renderPrivateAddr, Topics: []common.Hash{{tag}, {tag, tag}}, Data: nil}
}

// privateChainRPC is the fixture served as an ordinary L2 execution client.
type privateChainRPC struct{ f *privateChainFixture }

func (e *privateChainRPC) ChainId(ctx context.Context) (string, error) {
	return "0x" + new(big.Int).SetUint64(renderChainID).Text(16), nil
}

func (e *privateChainRPC) GetBlockByNumber(ctx context.Context, number json.RawMessage, fullTx bool) (map[string]json.RawMessage, error) {
	return e.blockJSON(), nil
}

func (e *privateChainRPC) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]json.RawMessage, error) {
	return e.blockJSON(), nil
}

func (e *privateChainRPC) GetBlockReceipts(ctx context.Context, blockNrOrHash json.RawMessage) (optypes.Receipts, error) {
	return e.f.receipts, nil
}

func (e *privateChainRPC) blockJSON() map[string]json.RawMessage {
	raw, err := json.Marshal(e.f.header)
	if err != nil {
		panic(err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		panic(err)
	}
	obj["hash"], _ = json.Marshal(e.f.hash)
	obj["transactions"], _ = json.Marshal(e.f.txs)
	obj["uncles"], _ = json.Marshal([]common.Hash{})
	// A post-Canyon node serves the empty withdrawals list its withdrawalsRoot commits to. The
	// ingestion client verifies that the two agree, which is one of the checks that only exists
	// because this source is a real, self-consistent node.
	obj["withdrawals"], _ = json.Marshal(gethTypes.Withdrawals{})
	return obj
}

// startPrivateChain serves the fixture over HTTP and returns the endpoint.
func startPrivateChain(t *testing.T, f *privateChainFixture) string {
	t.Helper()
	rpcSrv := gethRPC.NewServer()
	require.NoError(t, rpcSrv.RegisterName("eth", &privateChainRPC{f: f}))
	t.Cleanup(rpcSrv.Stop)
	httpSrv := httptest.NewServer(rpcSrv)
	t.Cleanup(httpSrv.Close)
	return httpSrv.URL
}

// newRenderIngester builds an ingester through the production constructor, pointed at the node.
func newRenderIngester(t *testing.T, rpcURL string, renderTransform *render.EmitterSet) *LogsDBChainIngester {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ing, err := NewLogsDBChainIngester(
		ctx,
		testlog.Logger(t, log.LevelError),
		metrics.NoopMetrics,
		eth.ChainIDFromUInt64(renderChainID),
		rpcURL,
		t.TempDir(),
		renderBlockTime,
		time.Second,
		100*time.Millisecond,
		testRollupConfig(renderChainID, renderBlockNum, renderBlockTime),
		10,
		4,
		renderTransform,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ing.Stop() })
	require.NoError(t, ing.initLogsDB())
	return ing
}

func renderChecksum(l *gethTypes.Log, logIdx uint32) messages.MessageChecksum {
	return messages.ChecksumArgs{
		BlockNumber: renderBlockNum,
		LogIndex:    logIdx,
		Timestamp:   renderBlockTime,
		ChainID:     eth.ChainIDFromUInt64(renderChainID),
		LogHash:     messages.LogToLogHash(l),
	}.Checksum()
}

func renderContains(ing *LogsDBChainIngester, l *gethTypes.Log, logIdx uint32) (messages.BlockSeal, error) {
	return ing.Contains(messages.ContainsQuery{
		Timestamp: renderBlockTime,
		BlockNum:  renderBlockNum,
		LogIdx:    logIdx,
		Checksum:  renderChecksum(l, logIdx),
	})
}

// TestRenderTransform_OffIsStockIngestion pins that an unflagged chain is untouched: every log the
// block emitted is stored, at the index it actually had. This is the behaviour every counterparty
// chain gets, and the transform must not perturb it.
func TestRenderTransform_OffIsStockIngestion(t *testing.T) {
	t.Parallel()
	f := newPrivateChainFixture(t)
	ing := newRenderIngester(t, startPrivateChain(t, f), nil)

	require.NoError(t, ing.ingestBlock(renderBlockNum))

	ref, logCount, execMsgs, err := ing.logsDB.OpenBlock(renderBlockNum)
	require.NoError(t, err)
	require.Equal(t, f.hash, ref.Hash)
	require.Equal(t, renderBlockTime, ref.Time)
	require.Equal(t, uint32(4), logCount, "all four logs are stored, private business included")

	require.Len(t, execMsgs, 1)
	require.Contains(t, execMsgs, uint32(3), "the import keeps its raw block-level index")

	seal, err := renderContains(ing, f.sent, 1)
	require.NoError(t, err, "the export answers at its raw index")
	require.Equal(t, f.hash, seal.Hash)

	_, err = renderContains(ing, f.sent, 0)
	require.Error(t, err, "index 0 holds a different log entirely")
}

// TestRenderTransform_OnStoresRenderedPositions is the deliverable: a flagged chain is fetched and
// verified exactly as above, and the transformation is then applied in-process, so the logs DB holds
// only the emitter-set logs, at their rendered indexes, under the block's real hash.
func TestRenderTransform_OnStoresRenderedPositions(t *testing.T) {
	t.Parallel()
	f := newPrivateChainFixture(t)
	ing := newRenderIngester(t, startPrivateChain(t, f), &render.EmitterSet{})

	require.NoError(t, ing.ingestBlock(renderBlockNum),
		"the source is an ordinary node and passes verification unchanged")

	// The block's identity is its own. Only the log sequence was transformed.
	latest, sealed := ing.logsDB.LatestSealedBlock()
	require.True(t, sealed)
	require.Equal(t, eth.BlockID{Hash: f.hash, Number: renderBlockNum}, latest,
		"the real block hash is what gets sealed")

	ref, logCount, execMsgs, err := ing.logsDB.OpenBlock(renderBlockNum)
	require.NoError(t, err)
	require.Equal(t, f.hash, ref.Hash)
	require.Equal(t, renderBlockTime, ref.Time)
	require.Equal(t, uint32(2), logCount,
		"four logs emitted, two in the emitter set: private business never reaches the logs DB")

	// The load-bearing assertion. The import was at raw log index 3; it is stored at RENDERED index
	// 1, which is the position a counterparty will reference.
	require.Len(t, execMsgs, 1)
	require.Contains(t, execMsgs, uint32(1), "the import is stored at its rendered index, not raw index 3")
	require.NotContains(t, execMsgs, uint32(3))
	require.Equal(t, eth.ChainIDFromUInt64(902), execMsgs[uint32(1)].ChainID)

	// The export answers at rendered index 0 (raw index 1) and at no other index.
	seal, err := renderContains(ing, f.sent, 0)
	require.NoError(t, err, "the export is queryable at its rendered position")
	require.Equal(t, f.hash, seal.Hash)

	_, err = renderContains(ing, f.sent, 1)
	require.Error(t, err, "the export's raw index is not a position anyone can reference")
}

// TestRenderTransform_ExtraEmitterIsCarried pins that the configured extra emitters reach the same
// predicate the rendering builder uses: with the private address configured as an extra emitter, its
// logs render too, and every index shifts accordingly. This is why the flag must match the builder's
// configuration.
func TestRenderTransform_ExtraEmitterIsCarried(t *testing.T) {
	t.Parallel()
	f := newPrivateChainFixture(t)
	set := render.NewEmitterSet(renderPrivateAddr)
	ing := newRenderIngester(t, startPrivateChain(t, f), &set)

	require.NoError(t, ing.ingestBlock(renderBlockNum))

	_, logCount, execMsgs, err := ing.logsDB.OpenBlock(renderBlockNum)
	require.NoError(t, err)
	require.Equal(t, uint32(4), logCount, "an extra emitter renders at any topic, so all four render")
	require.Contains(t, execMsgs, uint32(3),
		"a different emitter set is a different numbering: the same import is now at index 3")
}

func TestRenderingEmitterSetComesFromRollupConfig(t *testing.T) {
	t.Parallel()

	ordinary := testRollupConfig(renderChainID, renderBlockNum, renderBlockTime)
	require.Nil(t, renderingEmitterSet(ordinary))

	rendering := testRollupConfig(renderChainID, renderBlockNum, renderBlockTime)
	rendering.PrivateInterop = &rollup.PrivateInteropConfig{ExtraEmitters: []common.Address{renderPrivateAddr}}
	set := renderingEmitterSet(rendering)
	require.NotNil(t, set)
	require.True(t, set.Renders(&gethTypes.Log{Address: renderPrivateAddr}))
	require.False(t, set.Renders(&gethTypes.Log{Address: common.HexToAddress("0x00000000000000000000000000000000000abcde")}))
}
