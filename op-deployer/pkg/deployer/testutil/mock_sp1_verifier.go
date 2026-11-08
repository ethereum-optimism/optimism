package testutil

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// DeployMockSP1Verifier deploys the accept-all MockSP1Verifier test artifact to a live L1 and
// returns its address, for test deployments to pass as the raw SP1 verifier.
func DeployMockSP1Verifier(t *testing.T, client *ethclient.Client, key *ecdsa.PrivateKey, artifactsFS foundry.StatDirFs) common.Address {
	t.Helper()
	af := &foundry.ArtifactsFS{FS: artifactsFS}
	artifact, err := af.ReadArtifact("MockSP1Verifier.sol", "MockSP1Verifier")
	require.NoError(t, err, "MockSP1Verifier artifact needs a local contracts build")

	deployTx := txplan.NewPlannedTx(
		txplan.WithChainID(client),
		txplan.WithPrivateKey(key),
		txplan.WithPendingNonce(client),
		txplan.WithAgainstLatestBlockEthClient(client),
		txplan.WithData(artifact.Bytecode.Object),
		txplan.WithEstimator(client, true),
		txplan.WithRetrySubmission(client, 5, retry.Exponential()),
		txplan.WithRetryInclusion(txplan.FromGethReceipts(client), 5, retry.Exponential()),
	)
	receipt, err := deployTx.Included.Eval(t.Context())
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, receipt.ContractAddress, "MockSP1Verifier deploy produced no contract address")
	return receipt.ContractAddress
}
