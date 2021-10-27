package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func LoadData(dat []byte, ram map[uint32](uint32), base uint32) {
	for i := 0; i < len(dat); i += 4 {
		value := binary.BigEndian.Uint32(dat[i : i+4])
		if value != 0 {
			ram[base+uint32(i)] = value
		}
	}
}

func LoadMappedFile(fn string, ram map[uint32](uint32), base uint32) {
	dat, _ := ioutil.ReadFile(fn)
	LoadData(dat, ram, base)
}

func ZeroRegisters(ram map[uint32](uint32)) {
	for i := uint32(0xC0000000); i < 0xC0000000+36*4; i += 4 {
		WriteRam(ram, i, 0)
	}
}

func RunMinigeth(fn string, steps int, debug int) {
	ram := make(map[uint32](uint32))
	LoadMappedFile(fn, ram, 0)
	LoadMappedFile(fmt.Sprintf("/tmp/eth/%d", 13284469), ram, 0xB0000000)
	RunWithRam(ram, steps, debug, nil)
}

func runTest(fn string, steps int, debug int) (uint32, uint64) {
	ram := make(map[uint32](uint32))
	ram[0xC000007C] = 0x5EAD0000
	LoadMappedFile(fn, ram, 0)

	start := time.Now()
	remainingGas, err := RunWithRam(ram, steps, debug, nil)
	elapsed := time.Now().Sub(start)

	fmt.Println(err, remainingGas, elapsed,
		ram[0xbffffff4], ram[0xbffffff8], fmt.Sprintf("%x", ram[0xc0000080]), fn)
	if err != nil {
		log.Fatal(err)
	}
	return ram[0xbffffff4] & ram[0xbffffff8], remainingGas
}

func main() {
	steps, _ := strconv.Atoi(os.Getenv("STEPS"))
	if steps == 0 {
		steps = 1000000
	}
	if len(os.Args) > 1 {
		if strings.HasPrefix(os.Args[1], "../mipigo/") {
			debug, _ := strconv.Atoi(os.Getenv("DEBUG"))
			RunMinigeth(os.Args[1], steps, debug)
		} else if os.Args[1] == "unicorn" {
			uniram := make(map[uint32](uint32))
			RunUnicorn(os.Args[2], uniram, steps, false, nil)
		} else {
			runTest(os.Args[1], 20, 2)
		}
	} else {
		files, err := ioutil.ReadDir("test/bin")
		if err != nil {
			log.Fatal(err)
		}
		good := true
		gas := uint64(0)
		for _, f := range files {
			ret, lgas := runTest("test/bin/"+f.Name(), 100, 0)
			good = good && (ret == 1)
			gas += lgas
		}
		if !good {
			panic("some tests failed")
		}
		fmt.Println("used", gas, "gas")
	}
}
