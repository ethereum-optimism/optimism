// Package testengine runs op-reth-test-engine, the Rust OP Stack execution engine that Go tests
// drive over a Unix domain socket: it locates the binary, spawns it over a genesis file behind a
// typed client, and builds the minimal chains and blocks its own tests drive it with.
package testengine

import "context"

// BinaryName is the name of the engine's binary and of the cargo package that builds it in the
// monorepo's rust/ workspace.
const BinaryName = "op-reth-test-engine"

// socketFlag is the flag the engine takes its socket path under.
const socketFlag = "--socket"

// ResolveBinary returns the path of the op-reth-test-engine binary. See resolveBinary for the
// environment it reads and the order it looks in.
func ResolveBinary(ctx context.Context) (string, error) {
	return resolveBinary(ctx, BinaryName)
}
