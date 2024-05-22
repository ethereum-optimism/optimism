package memsize

import "unsafe"

var _ = unsafe.Pointer(nil)

//go:linkname startTheWorld runtime.startTheWorld
func startTheWorld()

//go:linkname chanbuf runtime.chanbuf
func chanbuf(ch unsafe.Pointer, i uint) unsafe.Pointer
