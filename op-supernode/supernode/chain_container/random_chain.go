package chain_container

// Random L1/L2 chain data for fuzzing the interop phases. A RandomChainManager
// generates valid chains; each RandomChain child implements
// virtual_node.VirtualNode and the engine_controller l2Provider set.

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"sort"
	"sync"
	"sync/atomic"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type L2Block struct {
	Ref      eth.L2BlockRef
	Payload  *eth.ExecutionPayloadEnvelope
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

// Receipts encodes the executing messages as logs the interop decoder accepts.
func (b *L2Block) Receipts() gethtypes.Receipts {
	keys := make([]uint32, 0, len(b.ExecMsgs))
	for k := range b.ExecMsgs {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	logs := make([]*gethtypes.Log, 0, len(keys))
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

// SafeHeadEntry is one SafeDB row: the L1 block at which an L2 safe head became
// safe. Sparse, ascending by L1 number.
type SafeHeadEntry struct {
	L1 eth.BlockID
	L2 eth.BlockID
}

// ---------------------------------------------------------------------------
// RandomChainManager
// ---------------------------------------------------------------------------

type RandomChainManager struct {
	chains map[eth.ChainID]*RandomChain
	order  []eth.ChainID // deterministic iteration

	l1Source *RandomL1Source
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

// RandomL1Source feeds the Phase-1 l1ConsistencyChecker.
type RandomL1Source struct {
	parent *RandomChainManager
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
