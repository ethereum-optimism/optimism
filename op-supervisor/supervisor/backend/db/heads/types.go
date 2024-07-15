package heads

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ChainHeads provides the serialization format for the current chain heads.
// The values here could be block numbers or just the index of entries in the log db.
// If they're log db entries, we can't detect if things changed because of a reorg though (if the logdb write succeeded and head update failed).
// So we probably need to store actual block IDs here... but then we don't have the block hash for every block in the log db.
// Only jumping the head forward on checkpoint blocks doesn't work though...
type ChainHeads struct {
	Unsafe         ChainHead `json:"localUnsafe"`
	CrossUnsafe    ChainHead `json:"crossUnsafe"`
	LocalSafe      ChainHead `json:"localSafe"`
	CrossSafe      ChainHead `json:"crossSafe"`
	LocalFinalized ChainHead `json:"localFinalized"`
	CrossFinalized ChainHead `json:"crossFinalized"`
}

type ChainHead struct {
	Index entrydb.EntryIdx `json:"index"`
	ID    entrydb.EntryID  `json:"id"`
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

func (h Heads) MarshalJSON() ([]byte, error) {
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
