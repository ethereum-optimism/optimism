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
		fmt.Printf("allocated %d bytes\n", alloc)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("alloc program exit. memstats: heap_alloc=%d frees=%d mallocs=%d\n", m.HeapAlloc, m.Frees, m.Mallocs)
}
