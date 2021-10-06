package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

type StateDB struct {
	Bytecode []byte
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
func (s *StateDB) GetBalance(addr common.Address) *big.Int { return common.Big0 }
func (s *StateDB) GetCode(addr common.Address) []byte {
	fmt.Println("GetCode", addr)
	return s.Bytecode
}
func (s *StateDB) GetCodeHash(addr common.Address) common.Hash { return common.Hash{} }
func (s *StateDB) GetCodeSize(addr common.Address) int         { return 100 }
func (s *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	return common.Hash{}
}
func (s *StateDB) GetNonce(addr common.Address) uint64 { return 0 }
func (s *StateDB) GetRefund() uint64                   { return 0 }

var seenWrite bool = true

func (s *StateDB) GetState(fakeaddr common.Address, hash common.Hash) common.Hash {
	//fmt.Println("GetState", addr, hash)
	addr := bytesTo32(hash.Bytes())
	nret := ram[addr]

	mret := make([]byte, 32)
	binary.BigEndian.PutUint32(mret[0x1c:], nret)

	if debug >= 2 {
		fmt.Println("HOOKED READ!   ", fmt.Sprintf("%x = %x", addr, nret))
	}

	if addr == 0xc0000080 && seenWrite {
		if debug >= 1 {
			fmt.Printf("%7d %8X %08X : %08X %08X %08X %08X %08X %08X %08X %08X %08X\n",
				pcCount, nret&0x7FFFFFFF, ram[nret],
				ram[0xc0000004],
				ram[0xc0000008], ram[0xc000000c], ram[0xc0000010], ram[0xc0000014],
				ram[0xc0000018], ram[0xc000001c], ram[0xc0000020], ram[0xc0000024])
		}
		if ram[nret] == 0xC {
			syscall := ram[0xc0000008]
			os.Stderr.WriteString(fmt.Sprintf("syscall %d at %x (step %d)\n", syscall, nret, pcCount))
		}
		if (pcCount % 10000) == 0 {
			steps_per_sec := float64(pcCount) * 1e9 / float64(time.Now().Sub(ministart).Nanoseconds())
			os.Stderr.WriteString(fmt.Sprintf("step %7d steps per s %f ram entries %d\n", pcCount, steps_per_sec, len(ram)))
		}
		pcCount += 1
		seenWrite = false
	}

	return common.BytesToHash(mret)
}
func (s *StateDB) HasSuicided(addr common.Address) bool { return false }
func (s *StateDB) PrepareAccessList(sender common.Address, dst *common.Address, precompiles []common.Address, list types.AccessList) {
}
func (s *StateDB) RevertToSnapshot(revid int)                 {}
func (s *StateDB) SetCode(addr common.Address, code []byte)   {}
func (s *StateDB) SetNonce(addr common.Address, nonce uint64) {}
func (s *StateDB) SetState(fakeaddr common.Address, key, value common.Hash) {
	//fmt.Println("SetState", addr, key, value)
	addr := bytesTo32(key.Bytes())
	dat := bytesTo32(value.Bytes())

	if addr == 0xc0000080 {
		seenWrite = true
	}

	if debug >= 2 {
		fmt.Println("HOOKED WRITE!  ", fmt.Sprintf("%x = %x", addr, dat))
	}

	if dat == 0 {
		delete(ram, addr)
	} else {
		ram[addr] = dat
	}
}
func (s *StateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return true, true
}
func (s *StateDB) Snapshot() int                                   { return 0 }
func (s *StateDB) SubBalance(addr common.Address, amount *big.Int) {}
func (s *StateDB) SubRefund(gas uint64)                            {}
func (s *StateDB) Suicide(addr common.Address) bool                { return true }

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

//var ram []byte
//var regs [4096]byte

var ministart time.Time
var pcCount int = 0
var debug int = 0

// TODO: why is ram uint64 -> uint32 and not uint32 -> uint32
var ram map[uint32](uint32)

func bytesTo32(a []byte) uint32 {
	//return uint32(common.BytesToHash(a).Big().Uint64())
	return binary.BigEndian.Uint32(a[28:])
}

func LoadMappedFile(fn string, ram map[uint32](uint32)) {
	dat, _ := ioutil.ReadFile(fn)
	for i := 0; i < len(dat); i += 4 {
		ram[uint32(i)] = binary.BigEndian.Uint32(dat[i : i+4])
	}
}

func RunMinigeth(fn string, interpreter *vm.EVMInterpreter, bytecode []byte, steps int) {
	ram = make(map[uint32](uint32))
	LoadMappedFile(fn, ram)

	gas := 10000 * uint64(steps)

	// 0xdb7df598
	from := common.Address{}
	to := common.HexToAddress("0x1337")
	input := []byte{0xdb, 0x7d, 0xf5, 0x98} // Steps(bytes32, uint256)
	input = append(input, common.BigToHash(common.Big0).Bytes()...)
	input = append(input, common.BigToHash(big.NewInt(int64(steps))).Bytes()...)

	ministart = time.Now()
	for i := 0; i < 1; i++ {
		contract := vm.NewContract(vm.AccountRef(from), vm.AccountRef(to), common.Big0, gas)
		contract.SetCallCode(&to, crypto.Keccak256Hash(bytecode), bytecode)
		_, err := interpreter.Run(contract, input, false)
		fmt.Printf("step:%d pc:0x%x v0:%d ", i, ram[0xc0000080], ram[0xc0000008])
		fmt.Println(err)

		ram[0xc0000008] = 0
		ram[0xc0000000+7*4] = 0
	}
	fmt.Println("evm ins count", evmInsCount)
}

func runTest(fn string, steps int, interpreter *vm.EVMInterpreter, bytecode []byte, gas uint64) (uint32, uint64) {
	ram = make(map[uint32](uint32))
	ram[0xC000007C] = 0xDEAD0000
	LoadMappedFile(fn, ram)

	// 0xdb7df598
	from := common.Address{}
	to := common.HexToAddress("0x1337")
	input := []byte{0xdb, 0x7d, 0xf5, 0x98} // Steps(bytes32, uint256)
	input = append(input, common.BigToHash(common.Big0).Bytes()...)
	input = append(input, common.BigToHash(big.NewInt(int64(steps))).Bytes()...)
	contract := vm.NewContract(vm.AccountRef(from), vm.AccountRef(to), common.Big0, gas)
	//fmt.Println(bytecodehash, bytecode)
	contract.SetCallCode(&to, crypto.Keccak256Hash(bytecode), bytecode)

	start := time.Now()
	_, err := interpreter.Run(contract, input, false)
	elapsed := time.Now().Sub(start)

	fmt.Println(err, contract.Gas, elapsed,
		ram[0xbffffff4], ram[0xbffffff8], fmt.Sprintf("%x", ram[0xc0000080]), fn)
	if err != nil {
		log.Fatal(err)
	}
	return ram[0xbffffff4] & ram[0xbffffff8], (gas - contract.Gas)
}

func GetInterpreterAndBytecode() (*vm.EVMInterpreter, []byte) {
	var jj jsoncontract
	mipsjson, _ := ioutil.ReadFile("../artifacts/contracts/MIPS.sol/MIPS.json")
	json.NewDecoder(bytes.NewReader(mipsjson)).Decode(&jj)
	bytecode := common.Hex2Bytes(jj.DeployedBytecode[2:])

	statedb := &StateDB{Bytecode: bytecode}

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
	return interpreter, bytecode
}

func main() {
	//fmt.Println("hello")

	/*var parent types.Header
	database := state.NewDatabase(parent)
	statedb, _ := state.New(parent.Root, database, nil)*/

	//fmt.Println(bytecode, jj.Bytecode)

	/*input := []byte{0x69, 0x37, 0x33, 0x72} // Step(bytes32)
	input = append(input, common.Hash{}.Bytes()...)*/

	// 1.26s for 100000 steps
	//steps := 20
	// 19.100079097s for 1_000_000 new steps
	//steps := 1000000
	//debug = true

	interpreter, bytecode := GetInterpreterAndBytecode()

	if len(os.Args) > 1 {
		if os.Args[1] == "../mipigeth/minigeth.bin" {
			debug, _ = strconv.Atoi(os.Getenv("DEBUG"))
			steps, _ := strconv.Atoi(os.Getenv("STEPS"))
			if steps == 0 {
				steps = 100000
			}
			RunMinigeth(os.Args[1], interpreter, bytecode, steps)
		} else {
			debug = 2
			runTest(os.Args[1], 20, interpreter, bytecode, 1000000)
		}
	} else {
		files, err := ioutil.ReadDir("test/bin")
		if err != nil {
			log.Fatal(err)
		}
		good := true
		gas := uint64(0)
		for _, f := range files {
			ret, lgas := runTest("test/bin/"+f.Name(), 100, interpreter, bytecode, 1000000)
			good = good && (ret == 1)
			gas += lgas
		}
		if !good {
			panic("some tests failed")
		}
		fmt.Println("used", gas, "gas")
	}
}
