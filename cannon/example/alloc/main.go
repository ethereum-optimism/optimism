package main

import (
	"encoding/binary"
	"fmt"
	"runtime"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

func main() {
	var mem []byte
	po := preimage.NewOracleClient(preimage.ClientPreimageChannel())
	numAllocs := binary.LittleEndian.Uint64(po.Get(preimage.LocalIndexKey(0)))

	fmt.Printf("alloc program. numAllocs=%d\n", numAllocs)
	var alloc int
	for i := 0; i < int(numAllocs); i++ {
		mem = make([]byte, 32*1024*1024)
		alloc += len(mem)
		// touch a couple pages to prevent the runtime from overcommitting memory
		for j := 0; j < len(mem); j += 1024 {
			mem[j] = 0xFF
		}
		printGCStats(alloc)
	}

	fmt.Println("alloc program exit")
	printGCStats(alloc)
}

func printGCStats(alloc int) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("allocated %d bytes. memstats: heap_alloc=%d next_gc=%d frees=%d mallocs=%d num_gc=%d\n",
		alloc, m.HeapAlloc, m.NextGC, m.Frees, m.Mallocs, m.NumGC)
}
