package main

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

type StateDB struct {
}

func (s *StateDB) AddAddressToAccessList(addr common.Address)                {}
func (s *StateDB) AddBalance(addr common.Address, amount *big.Int)           {}
func (s *StateDB) AddLog(log *types.Log)                                     {}
func (s *StateDB) AddPreimage(hash common.Hash, preimage []byte)             {}
func (s *StateDB) AddRefund(gas uint64)                                      {}
func (s *StateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {}
func (s *StateDB) AddressInAccessList(addr common.Address) bool              { return true }
func (s *StateDB) CreateAccount(addr common.Address)                         {}
func (s *StateDB) Empty(addr common.Address) bool                            { return false }
func (s *StateDB) Exist(addr common.Address) bool                            { return true }
func (b *StateDB) ForEachStorage(addr common.Address, cb func(key, value common.Hash) bool) error {
	return nil
}
func (s *StateDB) GetBalance(addr common.Address) *big.Int     { return common.Big0 }
func (s *StateDB) GetCode(addr common.Address) []byte          { return []byte{} }
func (s *StateDB) GetCodeHash(addr common.Address) common.Hash { return common.Hash{} }
func (s *StateDB) GetCodeSize(addr common.Address) int         { return 0 }
func (s *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	return common.Hash{}
}
func (s *StateDB) GetNonce(addr common.Address) uint64                        { return 0 }
func (s *StateDB) GetRefund() uint64                                          { return 0 }
func (s *StateDB) GetState(addr common.Address, hash common.Hash) common.Hash { return common.Hash{} }
func (s *StateDB) HasSuicided(addr common.Address) bool                       { return false }
func (s *StateDB) PrepareAccessList(sender common.Address, dst *common.Address, precompiles []common.Address, list types.AccessList) {
}
func (s *StateDB) RevertToSnapshot(revid int)                           {}
func (s *StateDB) SetCode(addr common.Address, code []byte)             {}
func (s *StateDB) SetNonce(addr common.Address, nonce uint64)           {}
func (s *StateDB) SetState(addr common.Address, key, value common.Hash) {}
func (s *StateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return true, true
}
func (s *StateDB) Snapshot() int                                   { return 0 }
func (s *StateDB) SubBalance(addr common.Address, amount *big.Int) {}
func (s *StateDB) SubRefund(gas uint64)                            {}
func (s *StateDB) Suicide(addr common.Address) bool                { return true }

func main() {
	fmt.Println("hello")

	/*var parent types.Header
	database := state.NewDatabase(parent)
	statedb, _ := state.New(parent.Root, database, nil)*/
	statedb := &StateDB{}

	config := vm.Config{}
	vm := vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, params.MainnetChainConfig, config)
	fmt.Println(vm)
}
