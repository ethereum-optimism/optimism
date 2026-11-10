package opcm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

// The MockSP1Verifier contract lives under test/. op-deployer/scripts/build-embedded-contracts.sh
// compiles it into the embedded artifacts; other contracts builds that skip test/ lack it.
const (
	MockSP1VerifierArtifact = "MockSP1Verifier.sol"
	MockSP1VerifierContract = "MockSP1Verifier"
)

// DeployMockSP1Verifier deploys the accept-all MockSP1Verifier into a script host and returns its
// address and init code. Development environments pass it as DeployImplementationsInput.SP1Verifier.
func DeployMockSP1Verifier(host *script.Host, deployer common.Address) (common.Address, []byte, error) {
	artifact, err := host.Artifacts().ReadArtifact(MockSP1VerifierArtifact, MockSP1VerifierContract)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to read MockSP1Verifier artifact: %w", err)
	}
	initCode := artifact.Bytecode.Object
	verifier, err := host.Create(deployer, initCode)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to deploy MockSP1Verifier: %w", err)
	}
	if verifier == (common.Address{}) {
		return common.Address{}, nil, fmt.Errorf("MockSP1Verifier deployment produced no contract address")
	}
	return verifier, initCode, nil
}
