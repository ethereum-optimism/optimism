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

// **** stub Tracer ****
type Tracer struct{}

// CaptureStart implements the Tracer interface to initialize the tracing operation.
func (jst *Tracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

// CaptureState implements the Tracer interface to trace a single step of VM execution.
var evmInsCount uint64 = 0

func (jst *Tracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	//fmt.Println(pc, op, gas)
	evmInsCount += 1
}

// CaptureFault implements the Tracer interface to trace an execution fault
func (jst *Tracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (jst *Tracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
}

type jsoncontract struct {
	Bytecode         string `json:"bytecode"`
	DeployedBytecode string `json:"deployedBytecode"`
}

func GetBytecode(deployed bool) []byte {
	var jj jsoncontract
	mipsjson, _ := ioutil.ReadFile("../artifacts/contracts/MIPS.sol/MIPS.json")
	json.NewDecoder(bytes.NewReader(mipsjson)).Decode(&jj)
	if deployed {
		return common.Hex2Bytes(jj.DeployedBytecode[2:])
	} else {
		return common.Hex2Bytes(jj.Bytecode[2:])
	}
}

func GetInterpreter(ldebug int, realState bool) (*vm.EVMInterpreter, *StateDB) {
	statedb := NewStateDB(ldebug, realState)

	var header types.Header
	header.Number = big.NewInt(13284469)
	header.Difficulty = common.Big0
	bc := core.NewBlockChain(&header)
	author := common.Address{}
	blockContext := core.NewEVMBlockContext(&header, bc, &author)
	txContext := vm.TxContext{}
	config := vm.Config{}

	/*config.Debug = true
	tracer := Tracer{}
	config.Tracer = &tracer*/

	evm := vm.NewEVM(blockContext, txContext, statedb, params.MainnetChainConfig, config)

	interpreter := vm.NewEVMInterpreter(evm, config)
	return interpreter, statedb
}

func RunWithRam(lram map[uint32](uint32), steps int, debug int, lcallback func(int, map[uint32](uint32))) (uint64, error) {
	interpreter, statedb := GetInterpreter(debug, false)
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
