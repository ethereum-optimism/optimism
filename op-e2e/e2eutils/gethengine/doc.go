// Package gethengine hosts an in-process op-geth L2 execution engine — the Engine API
// backend together with direct *core.BlockChain access — used exclusively by the kona
// fault-proof action tests in rust/kona/tests/proofs, which read L2 blocks and state
// straight from the chain object to build execution witnesses for the fault-proof program.
// The op-e2e action tests run on the out-of-process op-reth-test-engine and do not use this
// package.
package gethengine
