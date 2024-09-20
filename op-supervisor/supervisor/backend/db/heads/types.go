package heads

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"

	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type HeadPointer struct {
	// LastSealedBlockHash is the last fully-processed block
	LastSealedBlockHash suptypes.TruncatedHash
	LastSealedBlockNum  uint64

	// Number of logs that have been verified since the LastSealedBlock.
	// These logs are contained in the block that builds on top of the LastSealedBlock.
	LogsSince uint32
}

// WithinRange checks if the given log, in the given block,
// is within range (i.e. before or equal to the head-pointer).
// This does not guarantee that the log exists.
func (ptr *HeadPointer) WithinRange(blockNum uint64, logIdx uint32) bool {
	if ptr.LastSealedBlockHash == (suptypes.TruncatedHash{}) {
		return false // no block yet
	}
	return blockNum <= ptr.LastSealedBlockNum ||
		(blockNum+1 == ptr.LastSealedBlockNum && logIdx < ptr.LogsSince)
}

func (ptr *HeadPointer) IsSealed(blockNum uint64) bool {
	if ptr.LastSealedBlockHash == (suptypes.TruncatedHash{}) {
		return false // no block yet
	}
	return blockNum <= ptr.LastSealedBlockNum
}

// ChainHeads provides the serialization format for the current chain heads.
type ChainHeads struct {
	Unsafe         HeadPointer `json:"localUnsafe"`
	CrossUnsafe    HeadPointer `json:"crossUnsafe"`
	LocalSafe      HeadPointer `json:"localSafe"`
	CrossSafe      HeadPointer `json:"crossSafe"`
	LocalFinalized HeadPointer `json:"localFinalized"`
	CrossFinalized HeadPointer `json:"crossFinalized"`
}

type Heads struct {
	Chains map[types.ChainID]ChainHeads
}

func NewHeads() *Heads {
	return &Heads{Chains: make(map[types.ChainID]ChainHeads)}
}

func (h *Heads) Get(id types.ChainID) ChainHeads {
	chain, ok := h.Chains[id]
	if !ok {
		return ChainHeads{}
	}
	return chain
}

func (h *Heads) Put(id types.ChainID, head ChainHeads) {
	h.Chains[id] = head
}

func (h *Heads) Copy() *Heads {
	c := &Heads{Chains: make(map[types.ChainID]ChainHeads)}
	for id, heads := range h.Chains {
		c.Chains[id] = heads
	}
	return c
}

func (h *Heads) MarshalJSON() ([]byte, error) {
	data := make(map[hexutil.U256]ChainHeads)
	for id, heads := range h.Chains {
		data[hexutil.U256(id)] = heads
	}
	return json.Marshal(data)
}

func (h *Heads) UnmarshalJSON(data []byte) error {
	hexData := make(map[hexutil.U256]ChainHeads)
	if err := json.Unmarshal(data, &hexData); err != nil {
		return err
	}
	h.Chains = make(map[types.ChainID]ChainHeads)
	for id, heads := range hexData {
		h.Put(types.ChainID(id), heads)
	}
	return nil
}

type Operation interface {
	Apply(head *Heads) error
}

type OperationFn func(heads *Heads) error

func (f OperationFn) Apply(heads *Heads) error {
	return f(heads)
}
