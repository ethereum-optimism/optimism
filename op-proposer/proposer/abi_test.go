package proposer

import (
	"crypto/ecdsa"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-proposer/bindings"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func simulatedBackend() (privateKey *ecdsa.PrivateKey, address common.Address, opts *bind.TransactOpts, backend *backends.SimulatedBackend, err error) {
	privateKey, err = crypto.GenerateKey()
	if err != nil {
		return nil, common.Address{}, nil, nil, err
	}
	from := crypto.PubkeyToAddress(privateKey.PublicKey)
	opts, err = bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1337))
	if err != nil {
		return nil, common.Address{}, nil, nil, err
	}
	backend = backends.NewSimulatedBackend(types.GenesisAlloc{from: {Balance: big.NewInt(params.Ether)}}, 50_000_000) // nolint:staticcheck

	return privateKey, from, opts, backend, nil
}

// setupL2OutputOracle deploys the L2 Output Oracle contract to a simulated backend
func setupL2OutputOracle() (common.Address, *bind.TransactOpts, *backends.SimulatedBackend, *bindings.L2OutputOracle, error) {
	_, from, opts, backend, err := simulatedBackend()
	if err != nil {
		return common.Address{}, nil, nil, nil, err
	}
	_, _, contract, err := bindings.DeployL2OutputOracle(opts, backend)
	if err != nil {
		return common.Address{}, nil, nil, nil, err
	}
	return from, opts, backend, contract, nil
}

// setupDisputeGameFactory deploys the DisputeGameFactory contract to a simulated backend
func setupDisputeGameFactory() (common.Address, *bind.TransactOpts, *backends.SimulatedBackend, *bindings.DisputeGameFactory, error) {
	_, from, opts, backend, err := simulatedBackend()
	if err != nil {
		return common.Address{}, nil, nil, nil, err
	}

	_, _, contract, err := bindings.DeployDisputeGameFactory(
		opts,
		backend,
	)
	if err != nil {
		return common.Address{}, nil, nil, nil, err
	}
	return from, opts, backend, contract, nil
}

// TestManualABIPacking ensure that the manual ABI packing is the same as going through the bound contract.
// We don't use the contract to transact because it does not fit our transaction management scheme, but
// we want to make sure that we don't incorrectly create the transaction data.
func TestManualABIPacking(t *testing.T) {
	// L2OO
	_, opts, _, l2oo, err := setupL2OutputOracle()
	require.NoError(t, err)
	rng := rand.New(rand.NewSource(1234))

	l2ooAbi, err := bindings.L2OutputOracleMetaData.GetAbi()
	require.NoError(t, err)

	output := testutils.RandomOutputResponse(rng)

	txData, err := proposeL2OutputTxData(l2ooAbi, output)
	require.NoError(t, err)

	// set a gas limit to disable gas estimation. The invariants that the L2OO tries to uphold
	// are not maintained in this test.
	opts.GasLimit = 100_000
	tx, err := l2oo.ProposeL2Output(
		opts,
		output.OutputRoot,
		new(big.Int).SetUint64(output.BlockRef.Number),
		output.Status.CurrentL1.Hash,
		new(big.Int).SetUint64(output.Status.CurrentL1.Number))
	require.NoError(t, err)

	require.Equal(t, txData, tx.Data())

	// DGF
	_, opts, _, dgf, err := setupDisputeGameFactory()
	require.NoError(t, err)
	rng = rand.New(rand.NewSource(1234))

	dgfAbi, err := bindings.DisputeGameFactoryMetaData.GetAbi()
	require.NoError(t, err)

	output = testutils.RandomOutputResponse(rng)

	txData, err = proposeL2OutputDGFTxData(dgfAbi, uint32(0), output)
	require.NoError(t, err)

	opts.GasLimit = 100_000
	dgfTx, err := dgf.Create(
		opts,
		uint32(0),
		output.OutputRoot,
		math.U256Bytes(new(big.Int).SetUint64(output.BlockRef.Number)),
	)
	require.NoError(t, err)

	require.Equal(t, txData, dgfTx.Data())
}
