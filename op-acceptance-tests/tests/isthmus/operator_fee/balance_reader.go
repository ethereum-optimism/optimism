package operatorfee

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// BalanceReader provides methods to read balances from the chain
type BalanceReader struct {
	client *ethclient.Client
	t      devtest.T
	logger log.Logger
}

// NewBalanceReader creates a new BalanceReader instance
func NewBalanceReader(t devtest.T, client *ethclient.Client, logger log.Logger) *BalanceReader {
	return &BalanceReader{
		client: client,
		t:      t,
		logger: logger,
	}
}

// SampleBalances reads all the relevant balances at the given block number
// and returns a BalanceSnapshot containing the results
func (br *BalanceReader) SampleBalances(ctx context.Context, blockNumber *big.Int, walletAddr common.Address) *BalanceSnapshot {
	br.logger.Debug("Sampling balances",
		"block", blockNumber,
		"wallet", walletAddr.Hex())

	// Read all balances
	baseFeeVaultBalance, err := br.client.BalanceAt(ctx, predeploys.BaseFeeVaultAddr, blockNumber)
	require.NoError(br.t, err)

	l1FeeVaultBalance, err := br.client.BalanceAt(ctx, predeploys.L1FeeVaultAddr, blockNumber)
	require.NoError(br.t, err)

	sequencerFeeVaultBalance, err := br.client.BalanceAt(ctx, predeploys.SequencerFeeVaultAddr, blockNumber)
	require.NoError(br.t, err)

	operatorFeeVaultBalance, err := br.client.BalanceAt(ctx, predeploys.OperatorFeeVaultAddr, blockNumber)
	require.NoError(br.t, err)

	walletBalance, err := br.client.BalanceAt(ctx, walletAddr, blockNumber)
	require.NoError(br.t, err)

	br.logger.Debug("Sampled balances",
		"baseFee", baseFeeVaultBalance,
		"l1Fee", l1FeeVaultBalance,
		"sequencerFee", sequencerFeeVaultBalance,
		"operatorFee", operatorFeeVaultBalance,
		"wallet", walletBalance)

	return &BalanceSnapshot{
		BlockNumber:         blockNumber,
		BaseFeeVaultBalance: baseFeeVaultBalance,
		L1FeeVaultBalance:   l1FeeVaultBalance,
		SequencerFeeVault:   sequencerFeeVaultBalance,
		OperatorFeeVault:    operatorFeeVaultBalance,
		FromBalance:         walletBalance,
	}
}
