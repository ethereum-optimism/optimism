package main

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

func deploy(interpreter *vm.EVMInterpreter, statedb *StateDB) {
	bytecode := GetBytecode(false)

	from := common.Address{}
	to := common.Address{}
	gas := uint64(10000000)
	input := make([]byte, 0)

	contract := vm.NewContract(vm.AccountRef(from), vm.AccountRef(to), common.Big0, gas)
	contract.SetCallCode(&to, crypto.Keccak256Hash(bytecode), bytecode)
	ret, err := interpreter.Run(contract, input, false)
	check(err)
	fmt.Println("returned", len(ret))
	statedb.Bytecodes[common.HexToAddress("0x1337")] = ret
}

func addTrieNode(str []byte, interpreter *vm.EVMInterpreter, statedb *StateDB) {
	from := common.Address{}
	to := common.HexToAddress("0xBd770416a3345F91E4B34576cb804a576fa48EB1")
	gas := uint64(100000000)

	input := crypto.Keccak256Hash([]byte("AddTrieNode(bytes)")).Bytes()[:4]
	// offset
	input = append(input, common.BigToHash(big.NewInt(int64(0x20))).Bytes()...)
	// length
	input = append(input, common.BigToHash(big.NewInt(int64(len(str)))).Bytes()...)
	input = append(input, str...)
	input = append(input, make([]byte, 0x20-(len(input)%0x20))...)

	bytecode := statedb.Bytecodes[to]
	//fmt.Println("bytecode", len(bytecode))
	contract := vm.NewContract(vm.AccountRef(from), vm.AccountRef(to), common.Big0, gas)
	contract.SetCallCode(&to, crypto.Keccak256Hash(bytecode), bytecode)
	_, err := interpreter.Run(contract, input, false)
	check(err)
}

func RunFull() {
	interpreter, statedb := GetInterpreter(0, true)
	deploy(interpreter, statedb)

	ram := make(map[uint32](uint32))
	//LoadMappedFile("../mipigo/test/test.bin", ram, 0)
	LoadMappedFile("test/bin/add.bin", ram, 0)
	ZeroRegisters(ram)
	ram[0xC000007C] = 0x5EAD0000
	root := RamToTrie(ram)
	fmt.Println("state root", root, "nodes", len(Preimages))
	//ParseNode(root, 0)

	for _, v := range Preimages {
		addTrieNode(v, interpreter, statedb)
	}
}
