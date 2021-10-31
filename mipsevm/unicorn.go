package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// SHOULD BE GO BUILTIN
func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var steps int = 0
var heap_start uint64 = 0

func WriteBytes(fd int, bytes []byte) {
	printer := color.New(color.FgWhite).SprintFunc()
	if fd == 1 {
		printer = color.New(color.FgGreen).SprintFunc()
	} else if fd == 2 {
		printer = color.New(color.FgRed).SprintFunc()
	}
	os.Stderr.WriteString(printer(string(bytes)))
}

func WriteRam(ram map[uint32](uint32), addr uint32, value uint32) {
	// we no longer delete from ram, since deleting from tries is hard
	if value == 0 && false {
		delete(ram, addr)
	} else {
		/*if addr < 0xc0000000 {
			fmt.Printf("store %x = %x\n", addr, value)
		}*/
		ram[addr] = value
	}
}

var REG_OFFSET uint32 = 0xc0000000
var REG_PC uint32 = REG_OFFSET + 0x20*4
var REG_HEAP uint32 = REG_OFFSET + 0x23*4

func SyncRegs(mu uc.Unicorn, ram map[uint32](uint32)) {
	pc, _ := mu.RegRead(uc.MIPS_REG_PC)
	//fmt.Printf("%d uni %x\n", step, pc)
	WriteRam(ram, 0xc0000080, uint32(pc))

	addr := uint32(0xc0000000)
	for i := uc.MIPS_REG_ZERO; i < uc.MIPS_REG_ZERO+32; i++ {
		reg, _ := mu.RegRead(i)
		WriteRam(ram, addr, uint32(reg))
		addr += 4
	}

	reg_hi, _ := mu.RegRead(uc.MIPS_REG_HI)
	reg_lo, _ := mu.RegRead(uc.MIPS_REG_LO)
	WriteRam(ram, REG_OFFSET+0x21*4, uint32(reg_hi))
	WriteRam(ram, REG_OFFSET+0x22*4, uint32(reg_lo))

	WriteRam(ram, REG_HEAP, uint32(heap_start))
}

func GetHookedUnicorn(root string, ram map[uint32](uint32), callback func(int, uc.Unicorn, map[uint32](uint32))) uc.Unicorn {
	mu, err := uc.NewUnicorn(uc.ARCH_MIPS, uc.MODE_32|uc.MODE_BIG_ENDIAN)
	check(err)

	mu.HookAdd(uc.HOOK_INTR, func(mu uc.Unicorn, intno uint32) {
		if intno != 17 {
			log.Fatal("invalid interrupt ", intno, " at step ", steps)
		}
		syscall_no, _ := mu.RegRead(uc.MIPS_REG_V0)
		v0 := uint64(0)
		if syscall_no == 4020 {
			oracle_hash, _ := mu.MemRead(0xB0001000, 0x20)
			hash := common.BytesToHash(oracle_hash)
			key := fmt.Sprintf("%s/%s", root, hash)
			value, _ := ioutil.ReadFile(key)

			tmp := []byte{0, 0, 0, 0}
			binary.BigEndian.PutUint32(tmp, uint32(len(value)))
			mu.MemWrite(0xB1000000, tmp)
			mu.MemWrite(0xB1000004, value)

			WriteRam(ram, 0xB1000000, uint32(len(value)))
			value = append(value, 0, 0, 0)
			for i := uint32(0); i < ram[0xB1000000]; i += 4 {
				WriteRam(ram, 0xB1000004+i, binary.BigEndian.Uint32(value[i:i+4]))
			}
		} else if syscall_no == 4004 {
			fd, _ := mu.RegRead(uc.MIPS_REG_A0)
			buf, _ := mu.RegRead(uc.MIPS_REG_A1)
			count, _ := mu.RegRead(uc.MIPS_REG_A2)
			bytes, _ := mu.MemRead(buf, count)
			WriteBytes(int(fd), bytes)
		} else if syscall_no == 4090 {
			a0, _ := mu.RegRead(uc.MIPS_REG_A0)
			sz, _ := mu.RegRead(uc.MIPS_REG_A1)
			if a0 == 0 {
				v0 = 0x20000000 + heap_start
				heap_start += sz
			} else {
				v0 = a0
			}
		} else if syscall_no == 4045 {
			v0 = 0x40000000
		} else if syscall_no == 4120 {
			v0 = 1
		} else if syscall_no == 4246 {
			// exit group
			mu.RegWrite(uc.MIPS_REG_PC, 0x5ead0000)
		} else {
			//fmt.Println("syscall", syscall_no)
		}
		mu.RegWrite(uc.MIPS_REG_V0, v0)
		mu.RegWrite(uc.MIPS_REG_A3, 0)
	}, 0, 0)

	if callback != nil {
		mu.HookAdd(uc.HOOK_MEM_WRITE, func(mu uc.Unicorn, access int, addr64 uint64, size int, value int64) {
			rt := value
			rs := addr64 & 3
			addr := uint32(addr64 & 0xFFFFFFFC)
			//fmt.Printf("%X(%d) = %x (at step %d)\n", addr, size, value, steps)
			if size == 1 {
				mem := ram[addr]
				val := uint32((rt & 0xFF) << (24 - (rs&3)*8))
				mask := 0xFFFFFFFF ^ uint32(0xFF<<(24-(rs&3)*8))
				WriteRam(ram, uint32(addr), (mem&mask)|val)
			} else if size == 2 {
				mem := ram[addr]
				val := uint32((rt & 0xFFFF) << (16 - (rs&2)*8))
				mask := 0xFFFFFFFF ^ uint32(0xFFFF<<(16-(rs&2)*8))
				WriteRam(ram, uint32(addr), (mem&mask)|val)
			} else if size == 4 {
				WriteRam(ram, uint32(addr), uint32(rt))
			} else {
				log.Fatal("bad size write to ram")
			}

		}, 0, 0x80000000)

		mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
			SyncRegs(mu, ram)
			callback(steps, mu, ram)
			steps += 1
		}, 0, 0x80000000)
	}

	check(mu.MemMap(0, 0x80000000))
	return mu
}

func LoadMappedFileUnicorn(mu uc.Unicorn, fn string, ram map[uint32](uint32), base uint32) {
	dat, _ := ioutil.ReadFile(fn)
	LoadData(dat, ram, base)
	mu.MemWrite(uint64(base), dat)
}

// reimplement simple.py in go
func RunUnicorn(fn string, ram map[uint32](uint32), checkIO bool, callback func(int, uc.Unicorn, map[uint32](uint32))) {
	root := "/tmp/eth/13284469"
	mu := GetHookedUnicorn(root, ram, callback)

	// loop forever to match EVM
	//mu.MemMap(0x5ead0000, 0x1000)
	//mu.MemWrite(0xdead0000, []byte{0x08, 0x10, 0x00, 0x00})

	// program
	dat, _ := ioutil.ReadFile(fn)
	mu.MemWrite(0, dat)

	// inputs
	inputFile := fmt.Sprintf("%s/input", root)
	inputs, _ := ioutil.ReadFile(inputFile)
	mu.MemWrite(0xB0000000, inputs[0:0xc0])

	// load into ram
	LoadData(dat, ram, 0)
	if checkIO {
		LoadData(inputs[0:0xc0], ram, 0xB0000000)
	}

	mu.Start(0, 0x5ead0004)

	if checkIO {
		real := append([]byte{0x13, 0x37, 0xf0, 0x0d}, inputs[0xc0:]...)
		output, _ := mu.MemRead(0xB0000800, 0x44)
		if bytes.Compare(real, output) != 0 {
			log.Fatal("mismatch output")
		} else {
			fmt.Println("output match")
		}
	}
}
