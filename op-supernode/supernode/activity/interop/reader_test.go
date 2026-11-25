package interop

// Compile-time interface conformance assertions. If *Interop or
// NoopVerifiedResultReader stops satisfying VerifiedResultReader (e.g. because
// the method signature drifts), this file fails to compile.
var (
	_ VerifiedResultReader = (*Interop)(nil)
	_ VerifiedResultReader = NoopVerifiedResultReader{}
)
