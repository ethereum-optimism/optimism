package opcm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
)

// The MockSP1Verifier artifact is test-only: builds that skip test/, such as release builds, lack it.
const (
	MockSP1VerifierArtifact = "MockSP1Verifier.sol"
	MockSP1VerifierContract = "MockSP1Verifier"
)

// DeployMockSP1Verifier puts the accept-all MockSP1Verifier into a genesis state. Development
// environments pass it as DeployImplementationsInput.SP1Verifier.
func DeployMockSP1Verifier(genesis *GenesisState, af *foundry.ArtifactsFS) (common.Address, error) {
	artifact, err := af.ReadArtifact(MockSP1VerifierArtifact, MockSP1VerifierContract)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to read MockSP1Verifier artifact: %w", err)
	}
	// The verifier has no constructor and no immutables, so its runtime code is its genesis code.
	return genesis.Create(artifact.DeployedBytecode.Object), nil
}
