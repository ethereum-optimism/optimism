package projection

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// testVerifiersCompiled is set by gate_tag.go under the private_interop_test_verifiers build tag.
var testVerifiersCompiled = false

// testGateChainIDs are the only projection chain IDs on which a test-gated verifier mode may run
// (devstack DefaultL2AID/DefaultL2BID; every shared vector uses 901). An array, and unexported,
// so no importer can widen the allowlist at run time.
var testGateChainIDs = [...]uint64{901, 902}

// gateAllows is the pure §B.5 gate: the insecure modes must be compiled in (build tag or a go test
// binary) AND the projection chain must be allowlisted.
func gateAllows(compiledOrTesting bool, chainID *big.Int) bool {
	if !compiledOrTesting || chainID == nil || !chainID.IsUint64() {
		return false
	}
	for _, id := range testGateChainIDs {
		if bigs.Uint64Strict(chainID) == id {
			return true
		}
	}
	return false
}

// testVerifiersCompiledOrTesting is the compile half of the gate for this binary: the build tag,
// or a go test binary. It is false in every production binary built without the tag.
func testVerifiersCompiledOrTesting() bool {
	return testVerifiersCompiled || testing.Testing()
}

// testGated reports whether the config selects a verifier mode that provides no private-execution
// soundness: the stub, the execution mock, or SP1 with mock envelopes.
func (c *Config) testGated() bool {
	return c.Verifier == InsecureStub || c.Verifier == ExecutionMock || (c.Verifier == SP1PrivateProjectionV1 && c.MockProofs)
}
