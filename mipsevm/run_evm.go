package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var ministart time.Time

type jsoncontract struct {
	Bytecode         string `json:"bytecode"`
	DeployedBytecode string `json:"deployedBytecode"`
}

func GetBytecode(deployed bool) []byte {
	var jj jsoncontract
	mipsjson, err := ioutil.ReadFile("../artifacts/contracts/MIPS.sol/MIPS.json")
	check(err)
	json.NewDecoder(bytes.NewReader(mipsjson)).Decode(&jj)
	if deployed {
		return common.Hex2Bytes(jj.DeployedBytecode[2:])
	} else {
		return common.Hex2Bytes(jj.Bytecode[2:])
	}
}

func GetInterpreter(ldebug int, realState bool, root string) (*vm.EVMInterpreter, *StateDB) {
	statedb := NewStateDB(ldebug, realState, root)

	var header types.Header
	header.Number = big.NewInt(13284469)
	header.Difficulty = common.Big0
	bc := core.NewBlockChain(&header)
	author := common.Address{}
	blockContext := core.NewEVMBlockContext(&header, bc, &author)
	txContext := vm.TxContext{}
	config := vm.Config{}

	evm := vm.NewEVM(blockContext, txContext, statedb, params.MainnetChainConfig, config)

	interpreter := vm.NewEVMInterpreter(evm, config)
	return interpreter, statedb
}

func RunWithRam(lram map[uint32](uint32), steps int, debug int, root string, lcallback func(int, map[uint32](uint32))) (uint64, error) {
	interpreter, statedb := GetInterpreter(debug, false, root)
	statedb.Ram = lram

	callback = lcallback

	gas := 100000 * uint64(steps)

	// 0xdb7df598
	from := common.Address{}
	to := common.HexToAddress("0x1337")
	bytecode := GetBytecode(true)
	statedb.Bytecodes[to] = bytecode

	input := crypto.Keccak256Hash([]byte("Steps(bytes32,uint256)")).Bytes()[:4]
	input = append(input, common.BigToHash(common.Big0).Bytes()...)
	input = append(input, common.BigToHash(big.NewInt(int64(steps))).Bytes()...)

	ministart = time.Now()

	contract := vm.NewContract(vm.AccountRef(from), vm.AccountRef(to), common.Big0, gas)
	contract.SetCallCode(&to, crypto.Keccak256Hash(bytecode), bytecode)
	_, err := interpreter.Run(contract, input, false)

	return (gas - contract.Gas), err
}
