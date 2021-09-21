package state

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

type StateDB struct {
	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects map[common.Address]*stateObject
}

// AddAddressToAccessList adds the given address to the access list
func (s *StateDB) AddAddressToAccessList(addr common.Address) {
}

// AddBalance adds amount to the account associated with addr.
func (s *StateDB) AddBalance(addr common.Address, amount *big.Int) {
	fmt.Println("AddBalance", addr, amount)
}

// SubBalance subtracts amount from the account associated with addr.
func (s *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	fmt.Println("SubBalance", addr, amount)
}

func (s *StateDB) AddLog(log *types.Log) {
	fmt.Println("AddLog", log)
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// hopefully we don't have to implement this
	// hmm, if we want the right receipt hash we do
	// but for stateRoot we don't
	fmt.Println("IntermediateRoot")
	return common.HexToHash("0x0")
}

func (s *StateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	fmt.Println("GetLogs", hash, blockHash)
	return nil
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (s *StateDB) AddPreimage(hash common.Hash, preimage []byte) {
	fmt.Println("AddPreimage", hash)
}

// AddRefund adds gas to the refund counter
func (s *StateDB) AddRefund(gas uint64) {
	fmt.Println("AddRefund", gas)
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (s *StateDB) SubRefund(gas uint64) {
	fmt.Println("SubRefund", gas)
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (s *StateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
}

// AddressInAccessList returns true if the given address is in the access list.
func (s *StateDB) AddressInAccessList(addr common.Address) bool {
	return false
}

func (s *StateDB) CreateAccount(addr common.Address) {
}

// Finalise finalises the state by removing the s destructed objects and clears
// the journal as well as the refunds. Finalise, however, will not push any updates
// into the tries just yet. Only IntermediateRoot or Commit will do that.
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
}

// TxIndex returns the current transaction index set by Prepare.
func (s *StateDB) TxIndex() int {
	return 0
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (s *StateDB) Empty(addr common.Address) bool {
	return true
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for suicided accounts.
func (s *StateDB) Exist(addr common.Address) bool {
	return true
}

func (db *StateDB) ForEachStorage(addr common.Address, cb func(key, value common.Hash) bool) error {
	return nil
}

// GetBalance retrieves the balance from the given address or 0 if object not found
func (s *StateDB) GetBalance(addr common.Address) *big.Int {
	fmt.Println("GetBalance", addr)
	return big.NewInt(1e18)
}

func (s *StateDB) GetCode(addr common.Address) []byte {
	fmt.Println("GetCode", addr)
	return nil
}

func (s *StateDB) GetCodeSize(addr common.Address) int {
	fmt.Println("GetCodeSize", addr)
	return 0
}

func (s *StateDB) GetCodeHash(addr common.Address) common.Hash {
	fmt.Println("GetCodeHash", addr)
	return common.HexToHash("0x0")
}

// GetCommittedState retrieves a value from the given account's committed storage trie.
func (s *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	fmt.Println("GetCommittedState", addr, hash)
	return common.Hash{}
}

// GetState retrieves a value from the given account's storage trie.
func (s *StateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	fmt.Println("GetState", addr, hash)
	return common.Hash{}
}

func (s *StateDB) GetNonce(addr common.Address) uint64 {
	fmt.Println("GetNonce", addr)
	//return 2122

	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}

	return 0
}

// GetRefund returns the current value of the refund counter.
func (s *StateDB) GetRefund() uint64 {
	fmt.Println("GetRefund")
	return 0
}

func (s *StateDB) Suicide(addr common.Address) bool {
	fmt.Println("Suicide", addr)
	return true
}

func (s *StateDB) HasSuicided(addr common.Address) bool {
	fmt.Println("HasSuicided", addr)
	return false
}

func (s *StateDB) PrepareAccessList(sender common.Address, dst *common.Address, precompiles []common.Address, list types.AccessList) {
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (s *StateDB) RevertToSnapshot(revid int) {
}

func (s *StateDB) SetCode(addr common.Address, code []byte) {
	fmt.Println("SetCode", addr, code)
}

func (s *StateDB) SetNonce(addr common.Address, nonce uint64) {
	fmt.Println("SetNonce", addr, nonce)
}

func (s *StateDB) SetState(addr common.Address, key, value common.Hash) {
	fmt.Println("SetState", addr, key)
}

// SlotInAccessList returns true if the given (address, slot)-tuple is in the access list.
func (s *StateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return true, true
}

func (s *StateDB) Snapshot() int {
	return 0
}

// Prepare sets the current transaction hash and index which are
// used when the EVM emits new state logs.
func (s *StateDB) Prepare(thash common.Hash, ti int) {
}

// lower level

func (s *StateDB) setStateObject(object *stateObject) {
	s.stateObjects[object.Address()] = object
}

// getStateObject retrieves a state object given by the address, returning nil if
// the object is not found or was deleted in this execution context. If you need
// to differentiate between non-existent/just-deleted, use getDeletedStateObject.
func (s *StateDB) getStateObject(addr common.Address) *stateObject {
	if obj := s.getDeletedStateObject(addr); obj != nil && !obj.deleted {
		return obj
	}
	return nil
}

// getDeletedStateObject is similar to getStateObject, but instead of returning
// nil for a deleted state object, it returns the actual object with the deleted
// flag set. This is needed by the state journal to revert to the correct s-
// destructed object instead of wiping all knowledge about the state object.
func (s *StateDB) getDeletedStateObject(addr common.Address) *stateObject {
	// Prefer live objects if any is available
	if obj := s.stateObjects[addr]; obj != nil {
		return obj
	}

	fmt.Println("getDeletedStateObject:", addr)
	// If snapshot unavailable or reading from it failed, load from the database
	/*enc, err := s.trie.TryGet(addr.Bytes())
	if err != nil {
		fmt.Printf("getDeleteStateObject (%x) error: %v\n", addr.Bytes(), err)
		return nil
	}*/
	enc := []byte("12")
	if len(enc) == 0 {
		return nil
	}
	data := new(Account)
	if err := rlp.DecodeBytes(enc, data); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	// Insert into the live set
	obj := newObject(s, addr, *data)
	s.setStateObject(obj)
	return obj
}
