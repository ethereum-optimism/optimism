package safego

// NoCopy is a super simple safety util taken from the Go atomic lib.
//
// NoCopy may be added to structs which must not be copied
// after the first use.
//
// The NoCopy struct is empty, so should be a zero-cost util at runtime.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
//
// Note that it must not be embedded, due to the Lock and Unlock methods.
//
// Like:
// ```
//
//	type Example {
//		   V uint64
//		   _ NoCopy
//	}
//
// Then run: `go vet -copylocks .`
// ```
type NoCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*NoCopy) Lock()   {}
func (*NoCopy) Unlock() {}
