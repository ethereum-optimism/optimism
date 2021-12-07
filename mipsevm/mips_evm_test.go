package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"
)

func TestMinigethUnicorn(t *testing.T) {
	uniram := make(map[uint32](uint32))
	RunUnicorn("../mipigo/minigeth.bin", uniram, true, nil)
}

func TestSimpleEVM(t *testing.T) {
	files, err := ioutil.ReadDir("test/bin")
	if err != nil {
		log.Fatal(err)
	}
	good := true
	gas := uint64(0)
	for _, f := range files {
		ram := make(map[uint32](uint32))
		ram[0xC000007C] = 0x5EAD0000
		fn := "test/bin/" + f.Name()
		LoadMappedFile(fn, ram, 0)

		start := time.Now()
		remainingGas, err := RunWithRam(ram, 100, 0, "testoracle/", nil)
		elapsed := time.Now().Sub(start)

		fmt.Println(err, remainingGas, elapsed,
			ram[0xbffffff4], ram[0xbffffff8], fmt.Sprintf("%x", ram[0xc0000080]), fn)
		if err != nil {
			log.Fatal(err)
		}
		good = good && ((ram[0xbffffff4] & ram[0xbffffff8]) == 1)
		gas += remainingGas
	}
	if !good {
		panic("some tests failed")
	}
	fmt.Println("used", gas, "gas")
}
