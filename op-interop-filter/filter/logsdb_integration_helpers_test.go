package filter

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

const (
	defaultChainID      uint64 = 901
	defaultBlockTime    uint64 = 2
	defaultGenesisTime  uint64 = 1000
	defaultAnchorNumber uint64 = 99
	defaultAnchorTime   uint64 = 1198
	defaultFirstBlock   uint64 = 100
	defaultStartTs      uint64 = 1 // below typical seed-block timestamps so Ready() flips true
)

// seedLog declares one log on a seeded block.
//
// Either Log is set (use as-is) or it's nil and a `simpleLog` placeholder is
// synthesized. ExecMsg, if non-nil, replaces the default with an executing
// message targeting the given coordinates.
type seedLog struct {
	Log     *gethTypes.Log
	ExecMsg *seedExecMsg
}

type seedExecMsg struct {
	TargetChainID eth.ChainID
	TargetBlock   uint64
	TargetLogIdx  uint32
	TargetTs      uint64
	Origin        common.Address
	PayloadHash   common.Hash
}

// seedBlock declares one block to seal into the ingester's logsdb.
type seedBlock struct {
	Num            uint64
	Ts             uint64
	ParentOverride *common.Hash
	Logs           []seedLog
}

// seedSpec configures a single-chain seeded ingester.
type seedSpec struct {
	ChainID      uint64 // defaults to defaultChainID
	AnchorNumber uint64 // defaults to defaultAnchorNumber
	AnchorTime   uint64 // defaults to defaultAnchorTime
	Blocks       []seedBlock

	// Optional overrides — zero values fall through to test defaults.
	DataDir          string
	StartTimestamp   uint64
	BackfillDuration time.Duration
	PollInterval     time.Duration
	FetchConcurrency int
	NoSealAnchor     bool // omit sealParentBlock; used by init tests
	NoIngest         bool // omit per-block ingest; used by init tests
}

// seededIngester bundles a real LogsDBChainIngester with the artefacts a test
// needs to query and assert against it.
type seededIngester struct {
	*LogsDBChainIngester
	t         *testing.T
	chainID   eth.ChainID
	eth       *MockEthClient
	metrics   *capturingMetrics
	blockInfo map[uint64]*mockBlockInfo
	receipts  map[uint64]gethTypes.Receipts
}

// newSeededIngester builds a LogsDBChainIngester with a real on-disk logsdb in
// t.TempDir(), feeds it the spec's blocks (anchor + each ingestBlock), and
// returns it ready for assertions. Safe under t.Parallel().
func newSeededIngester(t *testing.T, spec seedSpec) *seededIngester {
	t.Helper()
	applySpecDefaults(&spec)

	si := &seededIngester{
		t:         t,
		chainID:   eth.ChainIDFromUInt64(spec.ChainID),
		eth:       NewMockEthClient(),
		metrics:   newCapturingMetrics(),
		blockInfo: map[uint64]*mockBlockInfo{},
		receipts:  map[uint64]gethTypes.Receipts{},
	}

	si.seedEthClient(spec)
	si.LogsDBChainIngester = buildIngester(t, spec, si.eth, si.metrics)

	require.NoError(t, si.initLogsDB(), "initLogsDB")
	t.Cleanup(func() {
		if si.logsDB != nil {
			_ = si.logsDB.Close()
		}
	})

	if !spec.NoSealAnchor {
		require.NoError(t, si.sealParentBlock(spec.AnchorNumber), "sealParentBlock")
	}
	if !spec.NoIngest {
		for _, b := range spec.Blocks {
			require.NoErrorf(t, si.ingestBlock(b.Num), "ingestBlock %d", b.Num)
		}
	}
	return si
}

func applySpecDefaults(spec *seedSpec) {
	if spec.ChainID == 0 {
		spec.ChainID = defaultChainID
	}
	if spec.AnchorNumber == 0 && spec.AnchorTime == 0 {
		spec.AnchorNumber = defaultAnchorNumber
		spec.AnchorTime = defaultAnchorTime
	}
	if spec.StartTimestamp == 0 {
		// Explicit zero -> use the default; a test that wants Ready=false should
		// override to a value above the seeded block timestamps.
		spec.StartTimestamp = defaultStartTs
	}
	if spec.PollInterval == 0 {
		spec.PollInterval = 100 * time.Millisecond
	}
	if spec.FetchConcurrency == 0 {
		spec.FetchConcurrency = 4
	}
}

func (si *seededIngester) seedEthClient(spec seedSpec) {
	anchor := makeBlockInfo(spec.AnchorNumber, spec.AnchorTime, common.Hash{})
	si.blockInfo[anchor.number] = anchor
	si.eth.AddBlock(anchor, nil)

	prevHash := anchor.hash
	prevNum := spec.AnchorNumber
	for _, b := range spec.Blocks {
		parent := prevHash
		if b.ParentOverride != nil {
			parent = *b.ParentOverride
		}
		info := makeBlockInfo(b.Num, b.Ts, parent)
		receipts := materialiseReceipts(b)
		si.blockInfo[b.Num] = info
		si.receipts[b.Num] = receipts
		si.eth.AddBlock(info, receipts)
		prevHash = info.hash
		prevNum = b.Num
	}
	if len(spec.Blocks) > 0 {
		si.eth.SetHeadBlock(si.blockInfo[prevNum])
		si.eth.SetLabelBlock(eth.Finalized, si.blockInfo[prevNum])
	} else {
		si.eth.SetHeadBlock(anchor)
		si.eth.SetLabelBlock(eth.Finalized, anchor)
	}
}

func buildIngester(t *testing.T, spec seedSpec, ethClient EthClient, m metrics.Metricer) *LogsDBChainIngester {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dataDir := spec.DataDir
	if dataDir == "" {
		dataDir = t.TempDir()
	}

	rollupCfg := testRollupConfig(spec.ChainID, 0, defaultGenesisTime)
	return &LogsDBChainIngester{
		log:              testlog.Logger(t, log.LevelError),
		metrics:          m,
		chainID:          eth.ChainIDFromUInt64(spec.ChainID),
		ethClient:        ethClient,
		dataDir:          dataDir,
		startTimestamp:   spec.StartTimestamp,
		backfillDuration: spec.BackfillDuration,
		pollInterval:     spec.PollInterval,
		rollupCfg:        rollupCfg,
		fetchConcurrency: spec.FetchConcurrency,
		ctx:              ctx,
		cancel:           cancel,
	}
}

func makeBlockInfo(number, timestamp uint64, parentHash common.Hash) *mockBlockInfo {
	var hash common.Hash
	binary.BigEndian.PutUint64(hash[24:], number)
	// Mix the parent into the hash so reorged branches get distinct hashes.
	copy(hash[:24], crypto.Keccak256(parentHash[:], hash[24:])[:24])
	return &mockBlockInfo{
		number:     number,
		hash:       hash,
		parentHash: parentHash,
		timestamp:  timestamp,
	}
}

func materialiseReceipts(b seedBlock) gethTypes.Receipts {
	if len(b.Logs) == 0 {
		return nil
	}
	logs := make([]*gethTypes.Log, 0, len(b.Logs))
	for i, sl := range b.Logs {
		switch {
		case sl.Log != nil:
			cp := *sl.Log
			cp.Index = uint(i)
			logs = append(logs, &cp)
		case sl.ExecMsg != nil:
			logs = append(logs, buildExecMsgLog(uint(i), *sl.ExecMsg))
		default:
			logs = append(logs, plainLog(uint(i), b.Num))
		}
	}
	return gethTypes.Receipts{{
		TxHash: common.Hash{byte(b.Num)},
		Logs:   logs,
	}}
}

func plainLog(index uint, blockNum uint64) *gethTypes.Log {
	return &gethTypes.Log{
		Address: common.Address{byte(blockNum), byte(index)},
		Topics:  []common.Hash{{0x01, 0x02, 0x03}},
		Data:    []byte{0x00},
		Index:   index,
	}
}

func buildExecMsgLog(index uint, em seedExecMsg) *gethTypes.Log {
	id := messages.Identifier{
		Origin:      em.Origin,
		BlockNumber: em.TargetBlock,
		LogIndex:    em.TargetLogIdx,
		Timestamp:   em.TargetTs,
		ChainID:     em.TargetChainID,
	}
	data := encodeExecMsgIdentifier(id)
	payloadHash := em.PayloadHash
	if payloadHash == (common.Hash{}) {
		payloadHash = common.Hash{0xee, byte(index), byte(em.TargetBlock)}
	}
	return &gethTypes.Log{
		Address: params.InteropCrossL2InboxAddress,
		Topics: []common.Hash{
			messages.ExecutingMessageEventTopic,
			payloadHash,
		},
		Data:  data,
		Index: index,
	}
}

// encodeExecMsgIdentifier is the inverse of Message.DecodeEvent's data section.
func encodeExecMsgIdentifier(id messages.Identifier) []byte {
	out := make([]byte, 32*5)
	copy(out[12:32], id.Origin[:])
	binary.BigEndian.PutUint64(out[32+24:64], id.BlockNumber)
	binary.BigEndian.PutUint32(out[64+28:96], id.LogIndex)
	binary.BigEndian.PutUint64(out[96+24:128], id.Timestamp)
	cid := id.ChainID.Bytes32()
	copy(out[128:160], cid[:])
	return out
}

// accessForLog returns an Access referencing the given (blockNum, logIdx) on
// this chain. The checksum is computed from the actual stored log content so
// the access entry is accepted by a happy-path CheckAccessList.
func (si *seededIngester) accessForLog(blockNum uint64, logIdx uint32) messages.Access {
	si.t.Helper()
	receipts, ok := si.receipts[blockNum]
	require.Truef(si.t, ok, "no receipts seeded for block %d", blockNum)
	require.Lessf(si.t, int(logIdx), len(receipts[0].Logs), "log index %d out of range for block %d", logIdx, blockNum)
	log := receipts[0].Logs[logIdx]
	info := si.blockInfo[blockNum]
	args := messages.ChecksumArgs{
		BlockNumber: blockNum,
		LogIndex:    logIdx,
		Timestamp:   info.timestamp,
		ChainID:     si.chainID,
		LogHash:     messages.LogToLogHash(log),
	}
	return args.Access()
}

// twoChainBackend builds a backend with a source chain (chainID 901) and an
// executing chain (chainID 902), both seeded with blocks 100 and 101 at
// timestamps 1200 and 1202. The source-chain block 100 carries the given
// number of placeholder logs.
func twoChainBackend(t *testing.T, sourceLogCount int) *seededBackend {
	t.Helper()
	sourceLogs := make([]seedLog, sourceLogCount)
	return newSeededBackend(t, backendOpts{
		Specs: []seedSpec{
			{ChainID: 901, AnchorNumber: 99, AnchorTime: 1198,
				Blocks: []seedBlock{
					{Num: 100, Ts: 1200, Logs: sourceLogs},
					{Num: 101, Ts: 1202},
				}},
			{ChainID: 902, AnchorNumber: 99, AnchorTime: 1198,
				Blocks: []seedBlock{
					{Num: 100, Ts: 1200},
					{Num: 101, Ts: 1202},
				}},
		},
	})
}

// sourceAccess returns an access for the first source-chain block's log.
func (sb *seededBackend) sourceAccess(blockNum uint64, logIdx uint32) messages.Access {
	return sb.ingesters[eth.ChainIDFromUInt64(901)].accessForLog(blockNum, logIdx)
}

const (
	executingChainID uint64 = 902
	sourceLogTs      uint64 = 1200
	inclusionTs      uint64 = 1300 // safely above latest sealed timestamps on both chains
)

func executingChain() eth.ChainID { return eth.ChainIDFromUInt64(executingChainID) }

// reopenSeededIngester builds a fresh ingester sharing the given ingester's
// dataDir and eth client, then runs initIngestion. Use this for restart /
// resume tests.
func reopenSeededIngester(t *testing.T, prev *seededIngester) *seededIngester {
	t.Helper()
	if prev.logsDB != nil {
		require.NoError(t, prev.logsDB.Close())
		prev.logsDB = nil
	}
	dataDir := prev.LogsDBChainIngester.dataDir

	si := &seededIngester{
		t:         t,
		chainID:   prev.chainID,
		eth:       prev.eth,
		metrics:   prev.metrics,
		blockInfo: prev.blockInfo,
		receipts:  prev.receipts,
	}
	chainIDu64, _ := prev.chainID.Uint64()
	si.LogsDBChainIngester = buildIngester(t, seedSpec{
		ChainID:          chainIDu64,
		DataDir:          dataDir,
		StartTimestamp:   defaultStartTs,
		PollInterval:     100 * time.Millisecond,
		FetchConcurrency: 4,
	}, si.eth, si.metrics)
	require.NoError(t, si.initLogsDB())
	t.Cleanup(func() {
		if si.logsDB != nil {
			_ = si.logsDB.Close()
		}
	})

	latest, ok := si.logsDB.LatestSealedBlock()
	require.True(t, ok, "reopened DB has no sealed blocks")
	require.NoError(t, si.findAndSetEarliestBlock(latest.Number+1))
	return si
}

// addBlock seeds an additional block on the eth client *after* construction.
// Returns the block info so tests can reference its hash.
func (si *seededIngester) addBlock(num, ts uint64, parent common.Hash, logs []seedLog) *mockBlockInfo {
	info := makeBlockInfo(num, ts, parent)
	receipts := materialiseReceipts(seedBlock{Num: num, Logs: logs})
	si.blockInfo[num] = info
	if receipts != nil {
		si.receipts[num] = receipts
	}
	si.eth.AddBlock(info, receipts)
	si.eth.SetHeadBlock(info)
	return info
}

// withChecksum returns a copy of acc with a synthetic checksum (typed prefix
// preserved so it survives access-list encoding).
func withChecksum(acc messages.Access, raw [32]byte) messages.Access {
	raw[0] = messages.PrefixChecksum
	acc.Checksum = messages.MessageChecksum(raw)
	return acc
}

// seededBackend wires a fully-real Backend (real LockstepCrossValidator, real
// LogsDBChainIngester, real on-disk logsdb) for end-to-end CheckAccessList
// tests.
type seededBackend struct {
	*Backend
	t         *testing.T
	ingesters map[eth.ChainID]*seededIngester
	metrics   *capturingMetrics
}

type backendOpts struct {
	Specs               []seedSpec
	MessageExpiryWindow uint64
	ValidationInterval  time.Duration
	Passthrough         bool
}

func newSeededBackend(t *testing.T, opts backendOpts) *seededBackend {
	t.Helper()
	if len(opts.Specs) == 0 {
		opts.Specs = []seedSpec{{Blocks: []seedBlock{{Num: defaultFirstBlock, Ts: defaultAnchorTime + defaultBlockTime}}}}
	}
	if opts.MessageExpiryWindow == 0 {
		opts.MessageExpiryWindow = 1 << 30
	}
	if opts.ValidationInterval == 0 {
		opts.ValidationInterval = 100 * time.Millisecond
	}

	chains := map[eth.ChainID]ChainIngester{}
	ingesters := map[eth.ChainID]*seededIngester{}
	mtr := newCapturingMetrics()

	for _, spec := range opts.Specs {
		si := newSeededIngester(t, spec)
		si.metrics = mtr
		si.LogsDBChainIngester.metrics = mtr
		chains[si.chainID] = si.LogsDBChainIngester
		ingesters[si.chainID] = si
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	logger := testlog.Logger(t, log.LevelError)
	cv := NewLockstepCrossValidator(ctx, logger, mtr, opts.MessageExpiryWindow, defaultStartTs,
		opts.ValidationInterval, chains)
	backend := NewBackend(ctx, BackendParams{
		Logger:               logger,
		Metrics:              mtr,
		Chains:               chains,
		CrossValidator:       cv,
		Passthrough:          opts.Passthrough,
		ReorgRecoveryEnabled: false,
	})
	return &seededBackend{Backend: backend, t: t, ingesters: ingesters, metrics: mtr}
}

// checkAccessList encodes the given access entries into the inbox-entry form
// CheckAccessList expects and forwards the call.
func (sb *seededBackend) checkAccessList(execChain eth.ChainID, execTs uint64, accesses ...messages.Access) error {
	entries := messages.EncodeAccessList(accesses)
	return sb.CheckAccessList(context.Background(), entries, safety.LocalUnsafe,
		messages.ExecutingDescriptor{Timestamp: execTs, ChainID: execChain})
}

// requireRejection asserts CheckAccessList rejects the given accesses with the
// expected classification label.
func (sb *seededBackend) requireRejection(execChain eth.ChainID, execTs uint64, expectedReason string, accesses ...messages.Access) {
	sb.t.Helper()
	before := sb.metrics.rejectionCount(expectedReason)
	err := sb.checkAccessList(execChain, execTs, accesses...)
	require.Errorf(sb.t, err, "expected rejection (reason=%s) but got nil", expectedReason)
	require.Equal(sb.t, expectedReason, classifyRejectionReason(err),
		"unexpected classification for err=%v", err)
	require.Equal(sb.t, before+1, sb.metrics.rejectionCount(expectedReason),
		"rejection metric not incremented for reason=%s", expectedReason)
}

// requireAccepted asserts CheckAccessList accepts the given accesses.
func (sb *seededBackend) requireAccepted(execChain eth.ChainID, execTs uint64, accesses ...messages.Access) {
	sb.t.Helper()
	err := sb.checkAccessList(execChain, execTs, accesses...)
	require.NoErrorf(sb.t, err, "expected accept, got err=%v", err)
}

// =============================================================================
// capturingMetrics — records the calls tests need to assert on.
// =============================================================================

type capturingMetrics struct {
	mu          sync.Mutex
	rejections  map[string]int
	reorgs      map[uint64]int
	blockSealed map[uint64]int64
	logsAdded   map[uint64]int64
	chainHead   map[uint64]uint64
}

func newCapturingMetrics() *capturingMetrics {
	return &capturingMetrics{
		rejections:  map[string]int{},
		reorgs:      map[uint64]int{},
		blockSealed: map[uint64]int64{},
		logsAdded:   map[uint64]int64{},
		chainHead:   map[uint64]uint64{},
	}
}

func (m *capturingMetrics) rejectionCount(reason string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rejections[reason]
}

func (m *capturingMetrics) reorgCount(chainID uint64) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reorgs[chainID]
}

func (m *capturingMetrics) sealedCount(chainID uint64) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.blockSealed[chainID]
}

func (m *capturingMetrics) RecordInfo(version string)          {}
func (m *capturingMetrics) RecordUp()                          {}
func (m *capturingMetrics) RecordFailsafeEnabled(enabled bool) {}
func (m *capturingMetrics) RecordChainHead(chainID uint64, blockNum uint64) {
	m.locked(func() { m.chainHead[chainID] = blockNum })
}
func (m *capturingMetrics) RecordCheckAccessList(success bool)             {}
func (m *capturingMetrics) RecordCheckAccessListDuration(duration float64) {}
func (m *capturingMetrics) RecordCheckAccessListRejection(reason string) {
	m.locked(func() { m.rejections[reason]++ })
}
func (m *capturingMetrics) RecordBackfillProgress(chainID uint64, p float64) {}
func (m *capturingMetrics) RecordReorgDetected(chainID uint64) {
	m.locked(func() { m.reorgs[chainID]++ })
}
func (m *capturingMetrics) RecordLogsAdded(chainID uint64, count int64) {
	m.locked(func() { m.logsAdded[chainID] += count })
}
func (m *capturingMetrics) RecordBlocksSealed(chainID uint64, count int64) {
	m.locked(func() { m.blockSealed[chainID] += count })
}
func (m *capturingMetrics) RecordCrossUnsafeValidatedTimestamp(timestamp uint64) {}

func (m *capturingMetrics) locked(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn()
}

var _ metrics.Metricer = (*capturingMetrics)(nil)
