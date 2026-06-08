package chain_container

// Random L1/L2 chain data for fuzzing the interop phases. A RandomChainManager
// generates valid chains; each RandomChain child implements
// virtual_node.VirtualNode and the engine_controller l2Provider set.

import (
	"context"
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

// RandomChain holds one chain's generated model and implements
// virtual_node.VirtualNode and the engine_controller l2Provider set over it.
type RandomChain struct {
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
	// bad label panics rather than reporting a zero-value head as real.
	return &eth.SyncStatus{
		CurrentL1:   rc.l1[rc.currentL1],
		FinalizedL1: rc.l1[rc.finalizedL1],
		UnsafeL2:    rc.l2[rc.unsafe].Ref,
		SafeL2:      rc.l2[rc.safe].Ref,
		LocalSafeL2: rc.l2[rc.safe].Ref,
		FinalizedL2: rc.l2[rc.finalized].Ref,
	}, nil
}
