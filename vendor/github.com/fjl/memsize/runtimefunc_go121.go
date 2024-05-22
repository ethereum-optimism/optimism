//go:build go1.21
// +build go1.21

package memsize

import "unsafe"

var _ = unsafe.Pointer(nil)

//go:linkname stwReason runtime.stwReason
type stwReason uint8

//go:linkname stwReadMemStats runtime.stwReadMemStats
const stwReadMemStats stwReason = 7

//go:linkname stopTheWorld runtime.stopTheWorld
func stopTheWorld(reason stwReason)
