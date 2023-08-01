package disputegame

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum-optimism/optimism/op-service/client/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

// deployDisputeGameContracts deploys the DisputeGameFactory, AlphabetVM and FaultDisputeGame contracts
// It configures the alphabet fault game as game type 0 (faultGameType)
// If/when the dispute game factory becomes a predeployed contract this can be removed and just use the
// predeployed version
func deployDisputeGameContracts(require *require.Assertions, ctx context.Context, client *ethclient.Client, opts *bind.TransactOpts, gameDuration uint64) *bindings.DisputeGameFactory {
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
	_, err = proxy.UpgradeToAndCall(opts, factoryAddr, data)
	require.NoError(err)
	factory, err := bindings.NewDisputeGameFactory(proxyAddr, client)
	require.NoError(err)

	// Now setup the fault dispute game type
	// Start by deploying the AlphabetVM
	_, tx, _, err = bindings.DeployAlphabetVM(opts, client, alphabetVMAbsolutePrestateClaim)
	require.NoError(err)
	alphaVMAddr, err := bind.WaitDeployed(ctx, client, tx)
	require.NoError(err)

	// Deploy the fault dispute game implementation
	_, tx, _, err = bindings.DeployFaultDisputeGame(opts, client, alphabetVMAbsolutePrestateClaim, big.NewInt(alphabetGameDepth), gameDuration, alphaVMAddr, common.Address{0xBE, 0xEF})
	require.NoError(err)
	faultDisputeGameAddr, err := bind.WaitDeployed(ctx, client, tx)
	require.NoError(err)

	// Set the fault game type implementation
	tx, err = factory.SetImplementation(opts, faultGameType, faultDisputeGameAddr)
	require.NoError(err)

	_, err = utils.WaitReceiptOK(ctx, client, tx.Hash())
	require.NoError(err, "wait for final transaction to be included and OK")

	return factory
}
