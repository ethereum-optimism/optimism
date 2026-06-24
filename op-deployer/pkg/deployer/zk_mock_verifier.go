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

// DeployZKMockVerifier deploys the test-only ZKMockVerifier contract on L1 and returns its address.
//
// DEV/DEVNET ONLY. The mock accepts every proof, so its only purpose is to give the ZK dispute
// game's verifier on-chain code (satisfying the ZKDG-80 verifier.code.length > 0 check) when
// spinning up a devnet for e2e testing without a real prover. The contract lives in test/ and is
// therefore absent from released artifacts, so artifactsFS must come from a local (full) contracts
// build; otherwise the artifact read fails. It must never be used as a verifier in production.
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
