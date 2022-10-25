package state

import (
	"errors"
	"math/big"

	lcommon "github.com/ethereum-optimism/optimism/l2geth/common"
	lstate "github.com/ethereum-optimism/optimism/l2geth/core/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

// WrappedStateDB wraps both the StateDB types from l2geth and upstream geth.
// This allows for a l2geth StateDB to be passed to functions that expect an
// upstream geth StateDB.
type WrappedStateDB struct {
	statedb       *state.StateDB
	legacyStatedb *lstate.StateDB
}

// NewWrappedStateDB will create a WrappedStateDB. It can wrap either an
// upstream geth database or a legacy l2geth database.
func NewWrappedStateDB(statedb *state.StateDB, legacyStatedb *lstate.StateDB) (*WrappedStateDB, error) {
	if statedb == nil && legacyStatedb == nil {
		return nil, errors.New("must pass at least 1 database")
	}
	if statedb != nil && legacyStatedb != nil {
		return nil, errors.New("cannot pass both databases")
	}

	return &WrappedStateDB{
		statedb:       statedb,
		legacyStatedb: legacyStatedb,
	}, nil
}

func (w *WrappedStateDB) CreateAccount(addr common.Address) {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		w.legacyStatedb.CreateAccount(address)
	} else {
		w.statedb.CreateAccount(addr)
	}
}

func (w *WrappedStateDB) SubBalance(addr common.Address, value *big.Int) {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		w.legacyStatedb.SubBalance(address, value)
	} else {
		w.statedb.SubBalance(addr, value)
	}
}

func (w *WrappedStateDB) AddBalance(addr common.Address, value *big.Int) {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		w.legacyStatedb.AddBalance(address, value)
	} else {
		w.statedb.AddBalance(addr, value)
	}
}

func (w *WrappedStateDB) GetBalance(addr common.Address) *big.Int {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		return w.legacyStatedb.GetBalance(address)
	} else {
		return w.statedb.GetBalance(addr)
	}
}

func (w *WrappedStateDB) GetNonce(addr common.Address) uint64 {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		return w.legacyStatedb.GetNonce(address)
	} else {
		return w.statedb.GetNonce(addr)
	}
}

func (w *WrappedStateDB) SetNonce(addr common.Address, nonce uint64) {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		w.legacyStatedb.SetNonce(address, nonce)
	} else {
		w.statedb.SetNonce(addr, nonce)
	}
}

func (w *WrappedStateDB) GetCodeHash(addr common.Address) common.Hash {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		hash := w.legacyStatedb.GetCodeHash(address)
		return common.BytesToHash(hash.Bytes())
	} else {
		return w.statedb.GetCodeHash(addr)
	}
}

func (w *WrappedStateDB) GetCode(addr common.Address) []byte {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		return w.legacyStatedb.GetCode(address)
	} else {
		return w.statedb.GetCode(addr)
	}
}

func (w *WrappedStateDB) SetCode(addr common.Address, code []byte) {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		w.legacyStatedb.SetCode(address, code)
	} else {
		w.statedb.SetCode(addr, code)
	}
}

func (w *WrappedStateDB) GetCodeSize(addr common.Address) int {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		return w.legacyStatedb.GetCodeSize(address)
	} else {
		return w.statedb.GetCodeSize(addr)
	}
}

func (w *WrappedStateDB) AddRefund(refund uint64) {
	if w.legacyStatedb != nil {
		w.legacyStatedb.AddRefund(refund)
	} else {
		w.statedb.AddRefund(refund)
	}
}

func (w *WrappedStateDB) SubRefund(refund uint64) {
	if w.legacyStatedb != nil {
		w.legacyStatedb.SubRefund(refund)
	} else {
		w.statedb.SubRefund(refund)
	}
}

func (w *WrappedStateDB) GetRefund() uint64 {
	if w.legacyStatedb != nil {
		return w.legacyStatedb.GetRefund()
	} else {
		return w.statedb.GetRefund()
	}
}

func (w *WrappedStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		lkey := lcommon.BytesToHash(key.Bytes())
		value := w.legacyStatedb.GetCommittedState(address, lkey)
		return common.BytesToHash(value.Bytes())
	} else {
		return w.statedb.GetCommittedState(addr, key)
	}
}

func (w *WrappedStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		lkey := lcommon.BytesToHash(key.Bytes())
		value := w.legacyStatedb.GetState(address, lkey)
		return common.BytesToHash(value.Bytes())
	} else {
		return w.statedb.GetState(addr, key)
	}
}

func (w *WrappedStateDB) SetState(addr common.Address, key, value common.Hash) {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		lkey := lcommon.BytesToHash(key.Bytes())
		lvalue := lcommon.BytesToHash(value.Bytes())
		w.legacyStatedb.SetState(address, lkey, lvalue)
	} else {
		w.statedb.SetState(addr, key, value)
	}
}

func (w *WrappedStateDB) Suicide(addr common.Address) bool {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		return w.legacyStatedb.Suicide(address)
	} else {
		return w.statedb.Suicide(addr)
	}
}

func (w *WrappedStateDB) HasSuicided(addr common.Address) bool {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		return w.legacyStatedb.HasSuicided(address)
	} else {
		return w.statedb.HasSuicided(addr)
	}
}

func (w *WrappedStateDB) Exist(addr common.Address) bool {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		return w.legacyStatedb.Exist(address)
	} else {
		return w.statedb.Exist(addr)
	}
}

func (w *WrappedStateDB) Empty(addr common.Address) bool {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		return w.legacyStatedb.Empty(address)
	} else {
		return w.statedb.Empty(addr)
	}
}

func (w *WrappedStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	if w.legacyStatedb != nil {
		panic("PrepareAccessList unimplemented")
	} else {
		w.statedb.PrepareAccessList(sender, dest, precompiles, txAccesses)
	}
}

func (w *WrappedStateDB) AddressInAccessList(addr common.Address) bool {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		return w.legacyStatedb.AddressInAccessList(address)
	} else {
		return w.statedb.AddressInAccessList(addr)
	}
}

func (w *WrappedStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		lslot := lcommon.BytesToHash(slot.Bytes())
		return w.legacyStatedb.SlotInAccessList(address, lslot)
	} else {
		return w.statedb.SlotInAccessList(addr, slot)
	}
}

func (w *WrappedStateDB) AddAddressToAccessList(addr common.Address) {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		w.legacyStatedb.AddAddressToAccessList(address)
	} else {
		w.statedb.AddAddressToAccessList(addr)
	}
}

func (w *WrappedStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		lslot := lcommon.BytesToHash(slot.Bytes())
		w.legacyStatedb.AddSlotToAccessList(address, lslot)
	} else {
		w.statedb.AddSlotToAccessList(addr, slot)
	}
}

func (w *WrappedStateDB) RevertToSnapshot(snapshot int) {
	if w.legacyStatedb != nil {
		w.legacyStatedb.RevertToSnapshot(snapshot)
	} else {
		w.statedb.RevertToSnapshot(snapshot)
	}
}

func (w *WrappedStateDB) Snapshot() int {
	if w.legacyStatedb != nil {
		return w.legacyStatedb.Snapshot()
	} else {
		return w.statedb.Snapshot()
	}
}

func (w *WrappedStateDB) AddLog(log *types.Log) {
	if w.legacyStatedb != nil {
		panic("AddLog unimplemented")
	} else {
		w.statedb.AddLog(log)
	}
}

func (w *WrappedStateDB) AddPreimage(hash common.Hash, preimage []byte) {
	if w.legacyStatedb != nil {
		lhash := lcommon.BytesToHash(hash.Bytes())
		w.legacyStatedb.AddPreimage(lhash, preimage)
	} else {
		w.statedb.AddPreimage(hash, preimage)
	}

}

func (w *WrappedStateDB) ForEachStorage(addr common.Address, cb func(common.Hash, common.Hash) bool) error {
	if w.legacyStatedb != nil {
		address := lcommon.BytesToAddress(addr.Bytes())
		return w.legacyStatedb.ForEachStorage(address, func(lkey, lvalue lcommon.Hash) bool {
			key := common.BytesToHash(lkey.Bytes())
			value := common.BytesToHash(lvalue.Bytes())
			return cb(key, value)
		})
	} else {
		return w.statedb.ForEachStorage(addr, cb)
	}
}
