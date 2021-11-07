package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

func DeployChain(interpreter *vm.EVMInterpreter, statedb *StateDB) {
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

func getTrieNode(str common.Hash, interpreter *vm.EVMInterpreter, statedb *StateDB) []byte {
	from := common.Address{}
	to := common.HexToAddress("0xBd770416a3345F91E4B34576cb804a576fa48EB1")
	gas := uint64(100000000)

	input := crypto.Keccak256Hash([]byte("trie(bytes32)")).Bytes()[:4]
	input = append(input, str.Bytes()...)

	bytecode := statedb.Bytecodes[to]
	//fmt.Println("bytecode", len(bytecode))
	contract := vm.NewContract(vm.AccountRef(from), vm.AccountRef(to), common.Big0, gas)
	contract.SetCallCode(&to, crypto.Keccak256Hash(bytecode), bytecode)
	ret, err := interpreter.Run(contract, input, false)
	check(err)

	//fmt.Println("getTrieNode", str, ret)

	return ret[64:]
}

func AddTrieNode(str []byte, interpreter *vm.EVMInterpreter, statedb *StateDB) {
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

func RunWithInputAndGas(interpreter *vm.EVMInterpreter, statedb *StateDB, input []byte, gas uint64) ([]byte, uint64, error) {
	from := common.Address{}
	to := common.HexToAddress("0x1337")
	bytecode := statedb.Bytecodes[to]
	contract := vm.NewContract(vm.AccountRef(from), vm.AccountRef(to), common.Big0, gas)
	contract.SetCallCode(&to, crypto.Keccak256Hash(bytecode), bytecode)
	dat, err := interpreter.Run(contract, input, false)
	return dat, (gas - contract.Gas), err
}

func ZeroRegisters(ram map[uint32](uint32)) {
	for i := uint32(0xC0000000); i < 0xC0000000+36*4; i += 4 {
		WriteRam(ram, i, 0)
	}
}

func LoadData(dat []byte, ram map[uint32](uint32), base uint32) {
	for i := 0; i < len(dat); i += 4 {
		value := binary.BigEndian.Uint32(dat[i : i+4])
		if value != 0 {
			ram[base+uint32(i)] = value
		}
	}
}

func LoadMappedFile(fn string, ram map[uint32](uint32), base uint32) {
	dat, err := ioutil.ReadFile(fn)
	check(err)
	LoadData(dat, ram, base)
}

func RunFull() {
	interpreter, statedb := GetInterpreter(0, true, "")
	DeployChain(interpreter, statedb)

	ram := make(map[uint32](uint32))
	LoadMappedFile("../mipigo/test/test.bin", ram, 0)
	totalSteps := 1000

	//LoadMappedFile("test/bin/add.bin", ram, 0)
	//totalSteps := 12

	ZeroRegisters(ram)
	ram[0xC000007C] = 0x5EAD0000
	root := RamToTrie(ram)
	//ParseNode(root, 0)

	ioutil.WriteFile("/tmp/cannon/trie.json", TrieToJson(root, -1), 0644)

	for k, v := range Preimages {
		fmt.Println("AddTrieNode", k)
		AddTrieNode(v, interpreter, statedb)
	}
	fmt.Println("trie is ready, let's run")
	fmt.Println("state root", root, "nodes", len(Preimages))

	for step := 0; step < totalSteps; step++ {
		// it's run o clock
		from := common.Address{}
		to := common.HexToAddress("0x1337")
		bytecode := statedb.Bytecodes[to]
		gas := uint64(100000000)

		steps := 1
		input := crypto.Keccak256Hash([]byte("Steps(bytes32,uint256)")).Bytes()[:4]
		input = append(input, root.Bytes()...)
		input = append(input, common.BigToHash(big.NewInt(int64(steps))).Bytes()...)

		contract := vm.NewContract(vm.AccountRef(from), vm.AccountRef(to), common.Big0, gas)
		contract.SetCallCode(&to, crypto.Keccak256Hash(bytecode), bytecode)
		dat, err := interpreter.Run(contract, input, false)
		if err != nil {
			if len(dat) >= 0x24 {
				fmt.Println(string(dat[0x24:]))
			}
			log.Fatal(err)
		} else {
			root = common.BytesToHash(dat)
			fmt.Println("new state root", step, root, "gas used", (gas - contract.Gas))
		}
	}

	/*ParseNode(root, 0, func(t common.Hash) []byte {
		return getTrieNode(t, interpreter, statedb)
	})*/
}
