package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

func LoadMappedFile(fn string, ram map[uint32](uint32)) {
	dat, _ := ioutil.ReadFile(fn)
	for i := 0; i < len(dat); i += 4 {
		ram[uint32(i)] = binary.BigEndian.Uint32(dat[i : i+4])
	}
}

func RunMinigeth(fn string, steps int, debug int) {
	ram := make(map[uint32](uint32))
	LoadMappedFile(fn, ram)
	RunWithRam(ram, steps, debug)
}

func runTest(fn string, steps int, debug int) (uint32, uint64) {
	ram := make(map[uint32](uint32))
	ram[0xC000007C] = 0xDEAD0000
	LoadMappedFile(fn, ram)

	start := time.Now()
	remainingGas, err := RunWithRam(ram, steps, debug)
	elapsed := time.Now().Sub(start)

	fmt.Println(err, remainingGas, elapsed,
		ram[0xbffffff4], ram[0xbffffff8], fmt.Sprintf("%x", ram[0xc0000080]), fn)
	if err != nil {
		log.Fatal(err)
	}
	return ram[0xbffffff4] & ram[0xbffffff8], remainingGas
}

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "../mipigeth/minigeth.bin" {
			debug, _ := strconv.Atoi(os.Getenv("DEBUG"))
			steps, _ := strconv.Atoi(os.Getenv("STEPS"))
			if steps == 0 {
				steps = 100000
			}
			RunMinigeth(os.Args[1], steps, debug)
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
