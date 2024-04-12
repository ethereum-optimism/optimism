package state

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var _ vm.StateDB = (*MemoryStateDB)(nil)

var (
	emptyCodeHash = crypto.Keccak256(nil)
	zeroAddr      = common.Address{}
)

// MemoryStateDB implements geth's StateDB interface
// but operates on a core.Genesis so that a genesis.json
// can easily be created.
type MemoryStateDB struct {
	rw      sync.RWMutex
	genesis *core.Genesis
}

func NewMemoryStateDB(genesis *core.Genesis) *MemoryStateDB {
	if genesis == nil {
		genesis = core.DeveloperGenesisBlock(15_000_000, &zeroAddr)
	}

	return &MemoryStateDB{
		genesis: genesis,
		rw:      sync.RWMutex{},
	}
}

// Genesis is a getter for the underlying core.Genesis
func (db *MemoryStateDB) Genesis() *core.Genesis {
	return db.genesis
}

// GetAccount is a getter for a core.GenesisAccount found in
// the core.Genesis
func (db *MemoryStateDB) GetAccount(addr common.Address) *core.GenesisAccount {
	db.rw.RLock()
	defer db.rw.RUnlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		return nil
	}
	return &account
}

// StateDB interface implemented below

func (db *MemoryStateDB) CreateAccount(addr common.Address) {
	db.rw.Lock()
	defer db.rw.Unlock()

	db.createAccount(addr)
}

func (db *MemoryStateDB) createAccount(addr common.Address) {
	if _, ok := db.genesis.Alloc[addr]; !ok {
		db.genesis.Alloc[addr] = core.GenesisAccount{
			Code:    []byte{},
			Storage: make(map[common.Hash]common.Hash),
			Balance: big.NewInt(0),
			Nonce:   0,
		}
	}
}

func (db *MemoryStateDB) SubBalance(addr common.Address, amount *uint256.Int) {
	db.rw.Lock()
	defer db.rw.Unlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		panic(fmt.Sprintf("%s not in state", addr))
	}
	if account.Balance.Sign() == 0 {
		return
	}
	account.Balance.Sub(account.Balance, amount.ToBig())
	db.genesis.Alloc[addr] = account
}

func (db *MemoryStateDB) AddBalance(addr common.Address, amount *uint256.Int) {
	db.rw.Lock()
	defer db.rw.Unlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		panic(fmt.Sprintf("%s not in state", addr))
	}
	account.Balance.Add(account.Balance, amount.ToBig())
	db.genesis.Alloc[addr] = account
}

func (db *MemoryStateDB) GetBalance(addr common.Address) *uint256.Int {
	db.rw.RLock()
	defer db.rw.RUnlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		return common.U2560
	}
	return uint256.MustFromBig(account.Balance)
}

func (db *MemoryStateDB) GetNonce(addr common.Address) uint64 {
	db.rw.RLock()
	defer db.rw.RUnlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		return 0
	}
	return account.Nonce
}

func (db *MemoryStateDB) SetNonce(addr common.Address, value uint64) {
	db.rw.Lock()
	defer db.rw.Unlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		return
	}
	account.Nonce = value
	db.genesis.Alloc[addr] = account
}

func (db *MemoryStateDB) GetCodeHash(addr common.Address) common.Hash {
	db.rw.RLock()
	defer db.rw.RUnlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		return common.Hash{}
	}
	if len(account.Code) == 0 {
		return common.BytesToHash(emptyCodeHash)
	}
	return common.BytesToHash(crypto.Keccak256(account.Code))
}

func (db *MemoryStateDB) GetCode(addr common.Address) []byte {
	db.rw.RLock()
	defer db.rw.RUnlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		return nil
	}
	if bytes.Equal(crypto.Keccak256(account.Code), emptyCodeHash) {
		return nil
	}
	return account.Code
}

func (db *MemoryStateDB) SetCode(addr common.Address, code []byte) {
	db.rw.Lock()
	defer db.rw.Unlock()

	db.createAccount(addr)

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		return
	}
	account.Code = code
	db.genesis.Alloc[addr] = account
}

func (db *MemoryStateDB) GetCodeSize(addr common.Address) int {
	db.rw.RLock()
	defer db.rw.RUnlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		return 0
	}
	if bytes.Equal(crypto.Keccak256(account.Code), emptyCodeHash) {
		return 0
	}
	return len(account.Code)
}

func (db *MemoryStateDB) AddRefund(uint64) {
	panic("AddRefund unimplemented")
}

func (db *MemoryStateDB) SubRefund(uint64) {
	panic("SubRefund unimplemented")
}

func (db *MemoryStateDB) GetRefund() uint64 {
	panic("GetRefund unimplemented")
}

func (db *MemoryStateDB) GetCommittedState(common.Address, common.Hash) common.Hash {
	panic("GetCommittedState unimplemented")
}

func (db *MemoryStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	db.rw.RLock()
	defer db.rw.RUnlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		return common.Hash{}
	}
	return account.Storage[key]
}

func (db *MemoryStateDB) SetState(addr common.Address, key, value common.Hash) {
	db.rw.Lock()
	defer db.rw.Unlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		panic(fmt.Sprintf("%s not in state", addr))
	}
	account.Storage[key] = value
	db.genesis.Alloc[addr] = account
}

func (db *MemoryStateDB) DeleteState(addr common.Address, key common.Hash) {
	db.rw.Lock()
	defer db.rw.Unlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		panic(fmt.Sprintf("%s not in state", addr))
	}
	delete(account.Storage, key)
	db.genesis.Alloc[addr] = account
}

func (db *MemoryStateDB) SelfDestruct(common.Address) {
	panic("SelfDestruct unimplemented")
}

func (db *MemoryStateDB) HasSelfDestructed(common.Address) bool {
	panic("HasSelfDestructed unimplemented")
}

func (db *MemoryStateDB) Selfdestruct6780(common.Address) {
	panic("Selfdestruct6780 unimplemented")
}

// Exist reports whether the given account exists in state.
// Notably this should also return true for suicided accounts.
func (db *MemoryStateDB) Exist(addr common.Address) bool {
	db.rw.RLock()
	defer db.rw.RUnlock()

	_, ok := db.genesis.Alloc[addr]
	return ok
}

// Empty returns whether the given account is empty. Empty
// is defined according to EIP161 (balance = nonce = code = 0).
func (db *MemoryStateDB) Empty(addr common.Address) bool {
	db.rw.RLock()
	defer db.rw.RUnlock()

	account, ok := db.genesis.Alloc[addr]
	isZeroNonce := account.Nonce == 0
	isZeroValue := account.Balance.Sign() == 0
	isEmptyCode := bytes.Equal(crypto.Keccak256(account.Code), emptyCodeHash)

	return ok || (isZeroNonce && isZeroValue && isEmptyCode)
}

func (db *MemoryStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	panic("PrepareAccessList unimplemented")
}

func (db *MemoryStateDB) AddressInAccessList(addr common.Address) bool {
	panic("AddressInAccessList unimplemented")
}

func (db *MemoryStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	panic("SlotInAccessList unimplemented")
}

// AddAddressToAccessList adds the given address to the access list. This operation is safe to perform
// even if the feature/fork is not active yet
func (db *MemoryStateDB) AddAddressToAccessList(addr common.Address) {
	panic("AddAddressToAccessList unimplemented")
}

// AddSlotToAccessList adds the given (address,slot) to the access list. This operation is safe to perform
// even if the feature/fork is not active yet
func (db *MemoryStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	panic("AddSlotToAccessList unimplemented")
}

func (db *MemoryStateDB) RevertToSnapshot(int) {
	panic("RevertToSnapshot unimplemented")
}

func (db *MemoryStateDB) Snapshot() int {
	panic("Snapshot unimplemented")
}

func (db *MemoryStateDB) AddLog(*types.Log) {
	panic("AddLog unimplemented")
}

func (db *MemoryStateDB) AddPreimage(common.Hash, []byte) {
	panic("AddPreimage unimplemented")
}

func (db *MemoryStateDB) ForEachStorage(addr common.Address, cb func(common.Hash, common.Hash) bool) error {
	db.rw.RLock()
	defer db.rw.RUnlock()

	account, ok := db.genesis.Alloc[addr]
	if !ok {
		return nil
	}
	for key, value := range account.Storage {
		if !cb(key, value) {
			return nil
		}
	}
	return nil
}

func (db *MemoryStateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	panic("transient state is unsupported")
}

func (db *MemoryStateDB) SetTransientState(addr common.Address, key, value common.Hash) {
	panic("transient state is unsupported")
}

func (db *MemoryStateDB) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	// no-op, no transient state to prepare, nor any access-list to set/prepare
}
