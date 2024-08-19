package testutil

import "math/rand"

func RandomRegisters(seed int64) [32]uint32 {
	r := rand.New(rand.NewSource(seed))
	var registers [32]uint32
	for i := 0; i < 32; i++ {
		registers[i] = r.Uint32()
	}
	return registers
}
