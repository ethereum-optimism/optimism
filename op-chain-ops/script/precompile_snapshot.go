package script

// This file is part of the op-geth-decoupling Rust script-engine spike (#20257 §16). It exposes
// the reflection a Precompile already builds so the Go side can drive the OPCM RunScript* path
// through the out-of-process Rust engine using the unidirectional design (design §4): the input
// precompile's getters are snapshotted here and shipped to the engine, and the output precompile's
// field-getter selectors configure the engine's setter-capture.

// Snapshot returns the ABI-encoded return value of every zero-argument getter of this precompile
// (struct fields and no-argument methods), keyed by 4-byte selector. It is the Go-side getter
// snapshot of the unidirectional transport: the Rust engine answers the script's input-getter
// CALLs from this map instead of calling back into Go.
//
// Getters that require arguments (and the field setters) are skipped: they return an error when
// invoked with empty calldata, which is exactly the set this snapshot must exclude.
func (p *Precompile[E]) Snapshot() map[[4]byte][]byte {
	out := make(map[[4]byte][]byte, len(p.abiMethods))
	for sel, fn := range p.abiMethods {
		data, err := fn.fn(nil)
		if err != nil {
			continue
		}
		out[sel] = data
	}
	return out
}

// SettableSelectors returns the 4-byte field-getter selectors of this precompile's settable fields
// (populated only when the precompile was built WithFieldSetter). These configure the Rust output
// precompile's set of valid getters, so it can answer the script's read-backs and reject unknown
// selectors exactly as the Go Precompile.Run does.
func (p *Precompile[E]) SettableSelectors() [][4]byte {
	out := make([][4]byte, 0, len(p.settable))
	for sel := range p.settable {
		out = append(out, sel)
	}
	return out
}
