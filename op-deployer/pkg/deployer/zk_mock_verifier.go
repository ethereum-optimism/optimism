package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	deployZKMockVerifierArtifact = "DeployZKMockVerifier.s.sol"
	deployZKMockVerifierContract = "DeployZKMockVerifier"
)

type deployZKMockVerifierScript struct {
	Run func() (common.Address, error)
}

// DeployZKMockVerifier deploys the test-only ZKMockVerifier and returns its address. DEV ONLY: the
// mock accepts every proof; it just gives the ZK game's verifier on-chain code (ZKDG-80) for devnet
// e2e without a real prover.
func DeployZKMockVerifier(
	ctx context.Context,
	client *ethclient.Client,
	key *ecdsa.PrivateKey,
	artifactsFS foundry.StatDirFs,
) (common.Address, error) {
	deployer := crypto.PubkeyToAddress(key.PublicKey)
	deployData, err := zkMockVerifierDeployData(artifactsFS, deployer)
	if err != nil {
		return common.Address{}, err
	}

	deployTx := txplan.NewPlannedTx(
		txplan.WithChainID(client),
		txplan.WithPrivateKey(key),
		txplan.WithPendingNonce(client),
		txplan.WithAgainstLatestBlockEthClient(client),
		txplan.WithData(deployData),
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

func zkMockVerifierDeployData(artifactsFS foundry.StatDirFs, deployer common.Address) ([]byte, error) {
	bcaster := new(broadcaster.CalldataBroadcaster)
	host, err := env.DefaultScriptHost(
		bcaster,
		log.NewLogger(log.DiscardHandler()),
		deployer,
		artifactsFS,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create script host: %w", err)
	}

	deployScript, cleanup, err := script.WithScript[deployZKMockVerifierScript](
		host,
		deployZKMockVerifierArtifact,
		deployZKMockVerifierContract,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployZKMockVerifier script: %w", err)
	}
	defer cleanup()

	verifier, err := deployScript.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run DeployZKMockVerifier script: %w", err)
	}
	if verifier == (common.Address{}) {
		return nil, fmt.Errorf("DeployZKMockVerifier script returned no contract address")
	}

	txs, err := bcaster.Dump()
	if err != nil {
		return nil, fmt.Errorf("failed to dump DeployZKMockVerifier transactions: %w", err)
	}
	if len(txs) != 1 {
		return nil, fmt.Errorf("DeployZKMockVerifier script produced %d transactions, expected 1", len(txs))
	}
	if txs[0].To != nil {
		return nil, fmt.Errorf("DeployZKMockVerifier script did not produce a contract creation")
	}
	if len(txs[0].Data) == 0 {
		return nil, fmt.Errorf("DeployZKMockVerifier script produced empty contract creation data")
	}
	return txs[0].Data, nil
}
