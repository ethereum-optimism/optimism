package disputegame

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum-optimism/optimism/op-service/client/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

// deployDisputeGameContracts deploys the DisputeGameFactory, AlphabetVM and FaultDisputeGame contracts
// It configures the alphabet fault game as game type 0 (faultGameType)
// If/when the dispute game factory becomes a predeployed contract this can be removed and just use the
// predeployed version
func deployDisputeGameContracts(require *require.Assertions, ctx context.Context, client *ethclient.Client, opts *bind.TransactOpts, gameDuration uint64) (*bindings.DisputeGameFactory, uint64) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	// Deploy the proxy
	_, tx, proxy, err := bindings.DeployProxy(opts, client, deployer.TestAddress)
	require.NoError(err)
	proxyAddr, err := bind.WaitDeployed(ctx, client, tx)
	require.NoError(err)

	// Deploy the dispute game factory implementation
	_, tx, _, err = bindings.DeployDisputeGameFactory(opts, client)
	require.NoError(err)
	factoryAddr, err := bind.WaitDeployed(ctx, client, tx)
	require.NoError(err)

	// Point the proxy at the implementation and create bindings going via the proxy
	disputeGameFactoryAbi, err := bindings.DisputeGameFactoryMetaData.GetAbi()
	require.NoError(err)
	data, err := disputeGameFactoryAbi.Pack("initialize", deployer.TestAddress)
	require.NoError(err)
	tx, err = proxy.UpgradeToAndCall(opts, factoryAddr, data)
	require.NoError(err)
	_, err = utils.WaitReceiptOK(ctx, client, tx.Hash())
	require.NoError(err)
	factory, err := bindings.NewDisputeGameFactory(proxyAddr, client)
	require.NoError(err)

	// Now setup the fault dispute game type
	// Start by deploying the AlphabetVM
	_, tx, _, err = bindings.DeployAlphabetVM(opts, client, alphabetVMAbsolutePrestateClaim)
	require.NoError(err)
	alphaVMAddr, err := bind.WaitDeployed(ctx, client, tx)
	require.NoError(err)

	// Deploy the block hash oracle
	_, tx, _, err = bindings.DeployBlockOracle(opts, client)
	require.NoError(err)
	blockHashOracleAddr, err := bind.WaitDeployed(ctx, client, tx)
	require.NoError(err)
	blockHashOracle, err := bindings.NewBlockOracle(blockHashOracleAddr, client)
	require.NoError(err)

	// Deploy the L2 output oracle
	_, tx, _, err = bindings.DeployL2OutputOracle(
		opts,
		client,
		big.NewInt(1800),
		big.NewInt(2),
		big.NewInt(0),
		big.NewInt(0),
		opts.From,
		opts.From,
		big.NewInt(3600),
	)
	require.NoError(err)
	l2OutputOracleAddr, err := bind.WaitDeployed(ctx, client, tx)
	require.NoError(err)
	l2OutputOracle, err := bindings.NewL2OutputOracle(l2OutputOracleAddr, client)
	require.NoError(err)

	// Deploy the fault dispute game implementation
	_, tx, _, err = bindings.DeployFaultDisputeGame(opts, client, alphabetVMAbsolutePrestateClaim, big.NewInt(alphabetGameDepth), gameDuration, alphaVMAddr, l2OutputOracleAddr, blockHashOracleAddr)
	require.NoError(err)
	faultDisputeGameAddr, err := bind.WaitDeployed(ctx, client, tx)
	require.NoError(err)

	// Propose 2 outputs
	for i := uint8(0); i < 2; i++ {
		nextBlockNumber, err := l2OutputOracle.NextBlockNumber(nil)
		require.NoError(err)
		block, err := client.BlockByNumber(ctx, big.NewInt(int64(i)))
		require.NoError(err)

		tx, err = l2OutputOracle.ProposeL2Output(opts, [32]byte{i + 1}, nextBlockNumber, block.Hash(), block.Number())
		require.NoError(err)
		_, err = utils.WaitReceiptOK(ctx, client, tx.Hash())
		require.NoError(err)
	}

	// Store the current block in the oracle
	blockNo, err := client.BlockNumber(ctx)
	require.NoError(err)
	tx, err = blockHashOracle.Store(opts, big.NewInt(int64(blockNo)))
	require.NoError(err)
	_, err = utils.WaitReceiptOK(ctx, client, tx.Hash())
	require.NoError(err, "failed to store block in blockoracle")

	// Set the fault game type implementation
	tx, err = factory.SetImplementation(opts, faultGameType, faultDisputeGameAddr)
	require.NoError(err)
	_, err = utils.WaitReceiptOK(ctx, client, tx.Hash())
	require.NoError(err, "wait for final transaction to be included and OK")

	return factory, blockNo
}
