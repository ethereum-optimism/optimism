//go:build !go1.21
// +build !go1.21

package memsize

import "unsafe"

var _ = unsafe.Pointer(nil)

const stwReadMemStats string = "memsize scan"

//go:linkname stopTheWorld runtime.stopTheWorld
func stopTheWorld(reason string)
