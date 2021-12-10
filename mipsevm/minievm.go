package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var callback func(int, map[uint32](uint32))

func bytesTo32(a []byte) uint32 {
	//return uint32(common.BytesToHash(a).Big().Uint64())
	return binary.BigEndian.Uint32(a[28:])
}

type StateDB struct {
	Bytecodes    map[common.Address]([]byte)
	RealState    map[common.Hash](common.Hash)
	Ram          map[uint32](uint32)
	Debug        int
	PcCount      int
	seenWrite    bool
	useRealState bool
	root         string
}

func NewStateDB(debug int, realState bool, root string) *StateDB {
	statedb := &StateDB{}
	statedb.Bytecodes = make(map[common.Address]([]byte))
	statedb.RealState = make(map[common.Hash](common.Hash))
	statedb.Debug = debug
	statedb.seenWrite = true
	statedb.useRealState = realState
	statedb.root = root
	return statedb
}

func (s *StateDB) AddAddressToAccessList(addr common.Address)      {}
func (s *StateDB) AddBalance(addr common.Address, amount *big.Int) {}
func (s *StateDB) AddLog(log *types.Log) {
	if s.Debug >= 1 {
		if log.Topics[0] == common.HexToHash("0x70c25ce54e55d181671946b6120c8147a91806a3620c981a355f3ae5b11deb13") {
			fmt.Printf("R: %x\n", bytesTo32(log.Data[0:32]))
		} else if log.Topics[0] == common.HexToHash("0x7b1a2ade00e6a076351ef8a0f302b160b7fd0c65c18234dfe8218c4fa4fa10ab") {
			//fmt.Printf("R: %x -> %x\n", bytesTo32(log.Data[0:32]), bytesTo32(log.Data[32:]))
		} else if log.Topics[0] == common.HexToHash("0x486ca368095cbbef9046ac7858bec943e866422cc388f49da1aa3aa77c10aa35") {
			fmt.Printf("W: %x <- %x\n", bytesTo32(log.Data[0:32]), bytesTo32(log.Data[32:]))
		} else if log.Topics[0] == common.HexToHash("0x86b89b5c9818dbbf520dd979a5f250d357508fe11b9511d4a43fd9bc6aa1be70") {
			fmt.Println("state", common.BytesToHash(log.Data))
		} else {
			fmt.Println("AddLog", log.Topics, log.Data)
		}
	}
}
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
	if s.Debug >= 2 {
		fmt.Println("GetCode", addr)
	}
	return s.Bytecodes[addr]
}
func (s *StateDB) GetCodeHash(addr common.Address) common.Hash { return common.Hash{} }
func (s *StateDB) GetCodeSize(addr common.Address) int         { return 100 }
func (s *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	return common.Hash{}
}
func (s *StateDB) GetNonce(addr common.Address) uint64 { return 0 }
func (s *StateDB) GetRefund() uint64                   { return 0 }

func (s *StateDB) GetState(fakeaddr common.Address, hash common.Hash) common.Hash {
	if s.useRealState {
		// TODO: fakeaddr?
		if s.Debug >= 2 {
			fmt.Println("GetState", fakeaddr, hash)
		}
		return s.RealState[hash]
	}
	ram := s.Ram
	//fmt.Println("GetState", addr, hash)
	addr := bytesTo32(hash.Bytes())
	nret := ram[addr]

	mret := make([]byte, 32)
	binary.BigEndian.PutUint32(mret[0x1c:], nret)

	if s.Debug >= 2 {
		fmt.Println("HOOKED READ!   ", fmt.Sprintf("%x = %x", addr, nret))
	}

	if addr == 0xc0000080 && s.seenWrite {
		if s.Debug >= 1 {
			fmt.Printf("%7d %8X %08X : %08X %08X %08X %08X %08X %08X %08X %08X %08X\n",
				s.PcCount, nret&0x7FFFFFFF, ram[nret],
				ram[0xc0000004],
				ram[0xc0000008], ram[0xc000000c], ram[0xc0000010], ram[0xc0000014],
				ram[0xc0000018], ram[0xc000001c], ram[0xc0000020], ram[0xc0000024])
		}
		if (s.PcCount%100000) == 0 && false {
			steps_per_sec := float64(s.PcCount) * 1e9 / float64(time.Now().Sub(ministart).Nanoseconds())
			os.Stderr.WriteString(fmt.Sprintf("%10d pc: %x steps per s %f ram entries %d\n", s.PcCount, nret&0x7FFFFFFF, steps_per_sec, len(ram)))
		}
		if callback != nil {
			callback(s.PcCount, ram)
		}
		if ram[nret] == 0xC {
			syscall := ram[0xc0000008]
			if syscall == 4020 {
				oracle_hash := make([]byte, 0x20)
				for i := uint32(0); i < 0x20; i += 4 {
					binary.BigEndian.PutUint32(oracle_hash[i:i+4], ram[0x30001000+i])
				}
				hash := common.BytesToHash(oracle_hash)

				if s.root == "" {
					log.Fatal("need root if using oracle for ", hash)
				}
				key := fmt.Sprintf("%s/%s", s.root, hash)
				value, err := ioutil.ReadFile(key)
				if err != nil {
					log.Fatal(err)
				}

				WriteRam(ram, 0x31000000, uint32(len(value)))
				value = append(value, 0, 0, 0)
				for i := uint32(0); i < ram[0x31000000]; i += 4 {
					WriteRam(ram, 0x31000004+i, binary.BigEndian.Uint32(value[i:i+4]))
				}
			} else if syscall == 4004 {
				len := ram[0xc0000018]
				buf := make([]byte, len+0x10)
				addr := ram[0xc0000014]
				offset := addr & 3
				for i := uint32(0); i < offset+len; i += 4 {
					binary.BigEndian.PutUint32(buf[i:i+4], ram[(addr&0xFFFFFFFC)+uint32(i)])
				}
				WriteBytes(int(ram[0xc0000010]), buf[offset:offset+len])
				//fmt.Printf("write %x %x %x\n", ram[0xc0000010], ram[0xc0000014], ram[0xc0000018])
			} else {
				//os.Stderr.WriteString(fmt.Sprintf("syscall %d at %x (step %d)\n", syscall, nret, pcCount))
			}
		}
		s.PcCount += 1
		s.seenWrite = false
	}

	return common.BytesToHash(mret)
}
func (s *StateDB) HasSuicided(addr common.Address) bool { return false }
func (s *StateDB) PrepareAccessList(sender common.Address, dst *common.Address, precompiles []common.Address, list types.AccessList) {
}
func (s *StateDB) RevertToSnapshot(revid int) {}
func (s *StateDB) SetCode(addr common.Address, code []byte) {
	fmt.Println("SetCode", addr, len(code))
	s.Bytecodes[addr] = code
}
func (s *StateDB) SetNonce(addr common.Address, nonce uint64) {}
func (s *StateDB) SetState(fakeaddr common.Address, key, value common.Hash) {
	if s.useRealState {
		if s.Debug >= 2 {
			fmt.Println("SetState", fakeaddr, key, value)
		}
		s.RealState[key] = value
		return
	}

	//fmt.Println("SetState", addr, key, value)
	addr := bytesTo32(key.Bytes())
	dat := bytesTo32(value.Bytes())

	if addr == 0xc0000080 {
		s.seenWrite = true
	}

	if s.Debug >= 2 {
		fmt.Println("HOOKED WRITE!  ", fmt.Sprintf("%x = %x (at step %d)", addr, dat, s.PcCount))
	}

	WriteRam(s.Ram, addr, dat)
}
func (s *StateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return true, true
}
func (s *StateDB) Snapshot() int                                   { return 0 }
func (s *StateDB) SubBalance(addr common.Address, amount *big.Int) {}
func (s *StateDB) SubRefund(gas uint64)                            {}
func (s *StateDB) Suicide(addr common.Address) bool                { return true }
