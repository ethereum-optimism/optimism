package chain_container

// Random L1/L2 chain data for fuzzing the interop phases. A RandomChainManager
// generates valid chains; each RandomChain child implements
// virtual_node.VirtualNode and the engine_controller l2Provider set.

import (
	"context"
	"errors"
	"hash/fnv"
	"math/big"
	"math/rand"
	"slices"
	"sort"
	"sync"
	"sync/atomic"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type L2Block struct {
	Ref     eth.L2BlockRef
	Payload *eth.ExecutionPayloadEnvelope
	// InitLog is a planted plain log other chains can reference as an
	// initiating message. Always emitted at log index 0.
	InitLog *gethtypes.Log
	// ExecMsgs holds executing messages keyed by their flat log index in the
	// block's receipts.
	ExecMsgs map[uint32]*supervisortypes.Message
}

// Output mirrors engine_controller.OutputV0AtBlockNumber's payload branch.
func (b *L2Block) Output() *eth.OutputV0 {
	p := b.Payload.ExecutionPayload
	var msgPasser eth.Bytes32
	if p.WithdrawalsRoot != nil {
		msgPasser = eth.Bytes32(*p.WithdrawalsRoot)
	}
	return &eth.OutputV0{
		StateRoot:                p.StateRoot,
		MessagePasserStorageRoot: msgPasser,
		BlockHash:                p.BlockHash,
	}
}

// Receipts encodes the init log followed by the executing messages as logs
// the interop decoder accepts.
func (b *L2Block) Receipts() gethtypes.Receipts {
	keys := make([]uint32, 0, len(b.ExecMsgs))
	for k := range b.ExecMsgs {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	logs := make([]*gethtypes.Log, 0, 1+len(keys))
	if b.InitLog != nil {
		logs = append(logs, b.InitLog)
	}
	for _, k := range keys {
		topics, data := b.ExecMsgs[k].EncodeEvent()
		logs = append(logs, &gethtypes.Log{
			Address: params.InteropCrossL2InboxAddress,
			Topics:  topics,
			Data:    data,
		})
	}
	return gethtypes.Receipts{{
		Type:   gethtypes.DynamicFeeTxType,
		Status: gethtypes.ReceiptStatusSuccessful,
		Logs:   logs,
	}}
}

// addExecMsg appends msg at the next flat log index (the init log occupies 0).
func (b *L2Block) addExecMsg(msg *supervisortypes.Message) {
	if b.ExecMsgs == nil {
		b.ExecMsgs = make(map[uint32]*supervisortypes.Message)
	}
	b.ExecMsgs[uint32(1+len(b.ExecMsgs))] = msg
}

// SafeHeadEntry is one SafeDB row: the L1 block at which an L2 safe head became
// safe. Sparse, ascending by L1 number.
type SafeHeadEntry struct {
	L1 eth.BlockID
	L2 eth.BlockID
}

// ---------------------------------------------------------------------------
// RandomChainManager
// ---------------------------------------------------------------------------

var ErrUnknownChain = errors.New("random chain: unknown chain id")

type RandomChainManager struct {
	rng *rand.Rand

	l1     []eth.L1BlockRef // canonical L1, shared by all chains
	chains map[eth.ChainID]*RandomChain
	order  []eth.ChainID // deterministic iteration

	l1Source *RandomL1Source
}

// NewRandomChainManager seeds the generator deterministically from the fuzz
// input bytes. Call Generate to build the chains.
func NewRandomChainManager(seed []byte) *RandomChainManager {
	h := fnv.New64a()
	_, _ = h.Write(seed)
	m := &RandomChainManager{rng: rand.New(rand.NewSource(int64(h.Sum64())))}
	m.l1Source = &RandomL1Source{parent: m}
	return m
}

const (
	genNumChains   = 2
	genL2Depth     = 8
	genL1Depth     = 4
	genL2BlockTime = 2
	genL1BlockTime = 12
	genGenesisTime = 1000
)

// Generate builds a fixed-shape set of valid, internally-consistent chains.
// Topology is constant; the fuzz input only varies block contents.
func (m *RandomChainManager) Generate() {
	m.l1 = m.generateL1()
	m.chains = make(map[eth.ChainID]*RandomChain, genNumChains)
	m.order = make([]eth.ChainID, 0, genNumChains)
	for i := 0; i < genNumChains; i++ {
		id := uint64(900 + i)
		rc := m.generateChain(id)
		m.chains[rc.chainID] = rc
		m.order = append(m.order, rc.chainID)
	}
	m.wireExecutingMessages()
}

// wireExecutingMessages plants one executing message in every verifiable block
// after the first on each chain, referencing the planted init log of an
// earlier verifiable block on another chain.
func (m *RandomChainManager) wireExecutingMessages() {
	if len(m.order) < 2 {
		return
	}
	for bi, id := range m.order {
		b := m.chains[id]
		for i := b.firstVerifiable() + 1; i <= b.safe; i++ {
			src := m.randomChainExcept(bi)
			j, ok := src.randomInitBlockBefore(b.l2[i].Ref.Time)
			if !ok {
				continue
			}
			b.l2[i].addExecMsg(src.initMessage(j))
		}
	}
}

// randomChainExcept returns a uniformly chosen chain other than m.order[i],
// by drawing from the n-1 other indices and shifting past i.
func (m *RandomChainManager) randomChainExcept(i int) *RandomChain {
	j := m.rng.Intn(len(m.order) - 1)
	if j >= i {
		j++
	}
	return m.chains[m.order[j]]
}

func (m *RandomChainManager) generateL1() []eth.L1BlockRef {
	l1 := make([]eth.L1BlockRef, genL1Depth)
	var parent common.Hash
	for i := range l1 {
		h := m.randHash()
		l1[i] = eth.L1BlockRef{
			Hash:       h,
			Number:     uint64(i),
			ParentHash: parent,
			Time:       genGenesisTime + uint64(i)*genL1BlockTime,
		}
		parent = h
	}
	return l1
}

func (m *RandomChainManager) randHash() common.Hash {
	var h common.Hash
	_, _ = m.rng.Read(h[:])
	return h
}

func (m *RandomChainManager) generateChain(idNum uint64) *RandomChain {
	l1 := m.l1

	l2 := make([]L2Block, genL2Depth)
	var l2Parent common.Hash
	for i := range l2 {
		h := m.randHash()
		stateRoot := m.randHash()
		withdrawals := m.randHash()
		l2[i] = L2Block{
			Ref: eth.L2BlockRef{
				Hash:       h,
				Number:     uint64(i),
				ParentHash: l2Parent,
				Time:       genGenesisTime + uint64(i)*genL2BlockTime,
				L1Origin:   l1[i*genL1Depth/genL2Depth].ID(),
			},
			Payload: &eth.ExecutionPayloadEnvelope{
				ExecutionPayload: &eth.ExecutionPayload{
					BlockHash:       h,
					BlockNumber:     eth.Uint64Quantity(i),
					StateRoot:       eth.Bytes32(stateRoot),
					WithdrawalsRoot: &withdrawals,
				},
			},
			InitLog: &gethtypes.Log{
				Address: common.Address(m.randHash().Bytes()[:20]),
				Topics:  []common.Hash{m.randHash()},
				Data:    m.randHash().Bytes(),
			},
		}
		l2Parent = h
	}

	safeDB := []SafeHeadEntry{
		{L1: l1[1].ID(), L2: l2[2].Ref.ID()},
		{L1: l1[2].ID(), L2: l2[4].Ref.ID()},
		{L1: l1[3].ID(), L2: l2[6].Ref.ID()},
	}

	return &RandomChain{
		parent:  m,
		chainID: eth.ChainIDFromUInt64(idNum),
		cfg: &rollup.Config{
			Genesis: rollup.Genesis{
				L1:     l1[0].ID(),
				L2:     l2[0].Ref.ID(),
				L2Time: l2[0].Ref.Time,
			},
			BlockTime: genL2BlockTime,
			L2ChainID: new(big.Int).SetUint64(idNum),
		},
		l2:          l2,
		l1:          l1,
		safeDB:      safeDB,
		unsafe:      genL2Depth - 1,
		safe:        6,
		finalized:   4,
		currentL1:   genL1Depth - 1,
		finalizedL1: 2,
	}
}

func (m *RandomChainManager) Chain(id eth.ChainID) *RandomChain { return m.chains[id] }

func (m *RandomChainManager) Chains() []*RandomChain {
	out := make([]*RandomChain, 0, len(m.order))
	for _, id := range m.order {
		out = append(out, m.chains[id])
	}
	return out
}

func (m *RandomChainManager) L1Source() *RandomL1Source { return m.l1Source }

// MinSafeTimestamp returns the lowest safe-head timestamp across chains — the
// last timestamp a verification loop over the generated data can reach.
func (m *RandomChainManager) MinSafeTimestamp() uint64 {
	min := uint64(0)
	for i, id := range m.order {
		rc := m.chains[id]
		ts := rc.l2[rc.safe].Ref.Time
		if i == 0 || ts < min {
			min = ts
		}
	}
	return min
}

// ChainContainer wires a simpleChainContainer: RandomChain as the VirtualNode
// and an EngineController wrapping the same RandomChain as l2Provider.
func (m *RandomChainManager) ChainContainer(id eth.ChainID) (*simpleChainContainer, error) {
	rc := m.chains[id]
	if rc == nil {
		return nil, ErrUnknownChain
	}
	return &simpleChainContainer{
		chainID: id,
		vn:      rc,
		engine:  engine_controller.NewEngineControllerWithL2AndRollup(rc, rc.cfg),
		vncfg:   &opnodecfg.Config{Rollup: *rc.cfg},
		log:     gethlog.New(),
		stopped: make(chan struct{}, 1),
		metrics: resources.NewSupernodeMetrics(),
	}, nil
}

// ChainContainers wires every generated chain, keyed by id — the shape the
// interop activity's constructor consumes.
func (m *RandomChainManager) ChainContainers() (map[eth.ChainID]InteropChain, error) {
	out := make(map[eth.ChainID]InteropChain, len(m.order))
	for _, id := range m.order {
		cc, err := m.ChainContainer(id)
		if err != nil {
			return nil, err
		}
		out[id] = cc
	}
	return out, nil
}

// RandomL1Source feeds the Phase-1 l1ConsistencyChecker from the canonical L1.
type RandomL1Source struct {
	parent *RandomChainManager
}

func (s *RandomL1Source) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	l1 := s.parent.l1
	if num >= uint64(len(l1)) {
		return eth.L1BlockRef{}, ethereum.NotFound
	}
	return l1[num], nil
}

// ---------------------------------------------------------------------------
// RandomChain
// ---------------------------------------------------------------------------

// RandomChain holds one chain's generated model and implements
// virtual_node.VirtualNode and the engine_controller l2Provider set over it.
type RandomChain struct {
	parent  *RandomChainManager
	chainID eth.ChainID
	cfg     *rollup.Config
	vncfg   *opnodecfg.Config

	mu     sync.RWMutex
	l2     []L2Block        // index == block number
	l1     []eth.L1BlockRef // index == block number
	safeDB []SafeHeadEntry  // sparse, ascending by L1 number

	// head labels as block numbers (indices into l2 / l1)
	unsafe, safe, finalized uint64 // L2 block numbers
	currentL1, finalizedL1  uint64 // L1 block numbers

	running atomic.Bool
}

// firstVerifiable returns the first block verification can reach: the lowest
// SafeDB-covered block. Earlier blocks are below SafeDB history, so they are
// never verified and never sealed into a logsDB.
func (rc *RandomChain) firstVerifiable() uint64 { return rc.safeDB[0].L2.Number }

// randomInitBlockBefore picks a uniform verifiable block with timestamp
// strictly below ts, to serve as an initiating block. ok is false when no such
// block exists.
func (rc *RandomChain) randomInitBlockBefore(ts uint64) (blockNum uint64, ok bool) {
	if ts == 0 {
		return 0, false
	}
	hi, err := rc.cfg.TargetBlockNumber(ts - 1)
	if err != nil {
		return 0, false
	}
	hi = min(hi, rc.safe)
	lo := rc.firstVerifiable()
	if hi < lo {
		return 0, false
	}
	return lo + uint64(rc.parent.rng.Int63n(int64(hi-lo+1))), true
}

// initMessage builds the executing-message reference to this chain's planted
// init log at blockNum.
func (rc *RandomChain) initMessage(blockNum uint64) *supervisortypes.Message {
	il := rc.l2[blockNum].InitLog
	return &supervisortypes.Message{
		Identifier: supervisortypes.Identifier{
			Origin:      il.Address,
			ChainID:     rc.chainID,
			BlockNumber: blockNum,
			LogIndex:    0,
			Timestamp:   rc.l2[blockNum].Ref.Time,
		},
		PayloadHash: crypto.Keccak256Hash(supervisortypes.LogToMessagePayload(il)),
	}
}

// --- virtual_node.VirtualNode ----------------------------------------------

func (rc *RandomChain) Start(ctx context.Context) error {
	rc.running.Store(true)
	return nil
}

func (rc *RandomChain) Stop(ctx context.Context) error {
	rc.running.Store(false)
	return nil
}

// SafeHeadAtL1 returns the highest entry whose L1.Number <= l1BlockNum.
func (rc *RandomChain) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	if !rc.running.Load() {
		return eth.BlockID{}, eth.BlockID{}, virtual_node.ErrVirtualNodeNotRunning
	}
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	i := sort.Search(len(rc.safeDB), func(i int) bool {
		return rc.safeDB[i].L1.Number > l1BlockNum
	})
	if i == 0 {
		return eth.BlockID{}, eth.BlockID{}, safedb.ErrNotFound
	}
	e := rc.safeDB[i-1]
	return e.L1, e.L2, nil
}

func (rc *RandomChain) L1AtSafeHead(ctx context.Context, target eth.BlockID) (eth.BlockID, error) {
	if !rc.running.Load() {
		return eth.BlockID{}, virtual_node.ErrVirtualNodeNotRunning
	}
	// Genesis is safe at L1 0 (the real VN uses 0, not cfg.Genesis.L1).
	if rc.cfg != nil && target == rc.cfg.Genesis.L2 {
		return eth.BlockID{Number: 0}, nil
	}
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.safeDBLookup(target)
}

// safeDBLookup returns the L1 of the first entry whose L2.Number >= l2.Number.
// Above the latest entry is transient (ErrL1AtSafeHeadNotFound); below the
// first is permanent (ErrL1AtSafeHeadUnavailable).
func (rc *RandomChain) safeDBLookup(l2 eth.BlockID) (eth.BlockID, error) {
	n := len(rc.safeDB)
	if n == 0 {
		return eth.BlockID{}, safedb.ErrL1AtSafeHeadNotFound
	}
	target := l2.Number
	if target > rc.safeDB[n-1].L2.Number {
		return eth.BlockID{}, safedb.ErrL1AtSafeHeadNotFound
	}
	if target < rc.safeDB[0].L2.Number {
		return eth.BlockID{}, safedb.ErrL1AtSafeHeadUnavailable
	}
	i := sort.Search(n, func(i int) bool {
		return rc.safeDB[i].L2.Number >= target
	})
	return rc.safeDB[i].L1, nil
}

func (rc *RandomChain) FirstSafeHeadEntry(ctx context.Context) (eth.BlockID, eth.BlockID, error) {
	if !rc.running.Load() {
		return eth.BlockID{}, eth.BlockID{}, virtual_node.ErrVirtualNodeNotRunning
	}
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if len(rc.safeDB) == 0 {
		return eth.BlockID{}, eth.BlockID{}, safedb.ErrNotFound
	}
	e := rc.safeDB[0]
	return e.L1, e.L2, nil
}

func (rc *RandomChain) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	if !rc.running.Load() {
		return nil, virtual_node.ErrVirtualNodeNotRunning
	}
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// Generation guarantees the labels index real blocks; index directly so a
	// bad label panics rather than reporting a zero-value head as real. Values
	// describe a fully-synced node (zero confirmation distance) so every field
	// is internally consistent, not just the ones currently read.
	curL1 := rc.l1[rc.currentL1]
	finL1 := rc.l1[rc.finalizedL1]
	unsafeL2 := rc.l2[rc.unsafe].Ref
	safeL2 := rc.l2[rc.safe].Ref
	return &eth.SyncStatus{
		CurrentL1:          curL1,
		CurrentL1Finalized: finL1,
		HeadL1:             curL1,
		SafeL1:             curL1,
		FinalizedL1:        finL1,
		UnsafeL2:           unsafeL2,
		CrossUnsafeL2:      unsafeL2,
		SafeL2:             safeL2,
		LocalSafeL2:        safeL2,
		PendingSafeL2:      safeL2,
		FinalizedL2:        rc.l2[rc.finalized].Ref,
	}, nil
}

// --- engine_controller l2Provider set --------------------------------------

func (rc *RandomChain) headNum(label eth.BlockLabel) uint64 {
	switch label {
	case eth.Unsafe:
		return rc.unsafe
	case eth.Finalized:
		return rc.finalized
	default:
		return rc.safe
	}
}

// blockByHash finds the block with the given hash. Assumes rc.mu held.
func (rc *RandomChain) blockByHash(h common.Hash) (*L2Block, bool) {
	for i := range rc.l2 {
		if rc.l2[i].Ref.Hash == h {
			return &rc.l2[i], true
		}
	}
	return nil, false
}

func (rc *RandomChain) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.l2[rc.headNum(label)].Ref, nil
}

func (rc *RandomChain) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if num >= uint64(len(rc.l2)) {
		return eth.L2BlockRef{}, ethereum.NotFound
	}
	return rc.l2[num].Ref, nil
}

func (rc *RandomChain) OutputV0AtBlockNumber(ctx context.Context, blockNum uint64) (*eth.OutputV0, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if blockNum >= uint64(len(rc.l2)) {
		return nil, ethereum.NotFound
	}
	return rc.l2[blockNum].Output(), nil
}

func (rc *RandomChain) OutputV0AtBlock(ctx context.Context, blockHash common.Hash) (*eth.OutputV0, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	blk, ok := rc.blockByHash(blockHash)
	if !ok {
		return nil, ethereum.NotFound
	}
	return blk.Output(), nil
}

func (rc *RandomChain) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if number >= uint64(len(rc.l2)) {
		return nil, ethereum.NotFound
	}
	return rc.l2[number].Payload, nil
}

func (rc *RandomChain) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	blk, ok := rc.blockByHash(hash)
	if !ok {
		return nil, ethereum.NotFound
	}
	return blk.Payload, nil
}

var errUnexpectedEngineCall = errors.New("random chain: unexpected engine call")

func (rc *RandomChain) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return nil, errUnexpectedEngineCall
}

func (rc *RandomChain) NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	return nil, errUnexpectedEngineCall
}

func (rc *RandomChain) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, gethtypes.Receipts, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	blk, ok := rc.blockByHash(blockHash)
	if !ok {
		return nil, nil, ethereum.NotFound
	}
	return blockInfoFor(blk), blk.Receipts(), nil
}

// blockInfoFor returns trusted info so Ref.Hash is used verbatim; the interop
// seal path reads only Hash/NumberU64/ParentHash/Time.
func blockInfoFor(blk *L2Block) eth.BlockInfo {
	h := &gethtypes.Header{
		ParentHash: blk.Ref.ParentHash,
		Number:     new(big.Int).SetUint64(blk.Ref.Number),
		Time:       blk.Ref.Time,
	}
	return eth.HeaderBlockInfoTrusted(blk.Ref.Hash, h)
}

// Close releases resources; RandomChain holds none.
func (rc *RandomChain) Close() {}

// l2Provider is unexported; assert conformance via the real constructor.
var _ virtual_node.VirtualNode = (*RandomChain)(nil)
var _ = engine_controller.NewEngineControllerWithL2AndRollup((*RandomChain)(nil), nil)
