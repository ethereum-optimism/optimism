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
	zkMockVerifierArtifact = "ZKMockVerifier.sol"
	zkMockVerifierContract = "ZKMockVerifier"
)

// DeployZKMockVerifier deploys the test-only ZKMockVerifier and returns its address. DEV ONLY: the
// mock accepts every proof; it just gives the ZK game's verifier on-chain code (ZKDG-80) for devnet
// e2e without a real prover. Lives in test/, so artifactsFS must be a local (full) build.
func DeployZKMockVerifier(
	ctx context.Context,
	client *ethclient.Client,
	key *ecdsa.PrivateKey,
	artifactsFS foundry.StatDirFs,
) (common.Address, error) {
	af := &foundry.ArtifactsFS{FS: artifactsFS}
	artifact, err := af.ReadArtifact(zkMockVerifierArtifact, zkMockVerifierContract)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to read ZKMockVerifier artifact (needs a local full contracts build): %w", err)
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
		return common.Address{}, fmt.Errorf("failed to deploy ZKMockVerifier: %w", err)
	}
	if receipt.ContractAddress == (common.Address{}) {
		return common.Address{}, fmt.Errorf("ZKMockVerifier deploy produced no contract address")
	}
	return receipt.ContractAddress, nil
}
