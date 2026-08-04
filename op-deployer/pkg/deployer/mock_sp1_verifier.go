package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

const (
	mockSP1VerifierArtifact = "MockSP1Verifier.sol"
	mockSP1VerifierContract = "MockSP1Verifier"
)

// DeployMockSP1Verifier deploys the test-only raw SP1 verifier used by development environments.
// The OPCM release wraps it in SP1PlonkAdapter; callers must not use it directly as an IZKVerifier.
func DeployMockSP1Verifier(
	ctx context.Context,
	client *ethclient.Client,
	key *ecdsa.PrivateKey,
	artifactsFS foundry.StatDirFs,
) (common.Address, error) {
	af := &foundry.ArtifactsFS{FS: artifactsFS}
	artifact, err := af.ReadArtifact(mockSP1VerifierArtifact, mockSP1VerifierContract)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to read MockSP1Verifier artifact (needs a local full contracts build): %w", err)
	}

	deployTx := txplan.NewPlannedTx(
		txplan.WithChainID(client),
		txplan.WithPrivateKey(key),
		txplan.WithPendingNonce(client),
		txplan.WithAgainstLatestBlockEthClient(client),
		txplan.WithData(artifact.Bytecode.Object),
		txplan.WithEstimator(client, true),
		txplan.WithRetrySubmission(client, 5, retry.Exponential()),
		txplan.WithRetryInclusion(client, 5, retry.Exponential()),
	)
	receipt, err := deployTx.Included.Eval(ctx)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to deploy MockSP1Verifier: %w", err)
	}
	if receipt.ContractAddress == (common.Address{}) {
		return common.Address{}, fmt.Errorf("MockSP1Verifier deploy produced no contract address")
	}
	return receipt.ContractAddress, nil
}
