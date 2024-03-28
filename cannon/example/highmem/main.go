package main

var mem [2][]byte
var memsiz int32 = 512 * 1024 * 1024

func main() {
	for i := range mem {
		mem[i] = make([]byte, memsiz)
		// dirty memory to ensure allocated memory is flushed
		for j := range mem[i] {
			mem[i][j] = 0x12
		}
	}
}
