package op_e2e

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func SendDepositTx(t *testing.T, cfg SystemConfig, l1Client *ethclient.Client, l2Client *ethclient.Client, l1Opts *bind.TransactOpts, applyL2Opts DepositTxOptsFn) {
	l2Opts := defaultDepositTxOpts(l1Opts)
	applyL2Opts(l2Opts)
	// Find deposit contract
	depositContract, err := bindings.NewOptimismPortal(predeploys.DevOptimismPortalAddr, l1Client)
	require.Nil(t, err)

	// Finally send TX
	tx, err := depositContract.DepositTransaction(l1Opts, l2Opts.ToAddr, l2Opts.Value, l2Opts.GasLimit, l2Opts.IsCreation, l2Opts.Data)
	require.Nil(t, err, "with deposit tx")

	// Wait for transaction on L1
	receipt, err := waitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for deposit tx on L1")

	// Wait for transaction to be included on L2
	reconstructedDep, err := derive.UnmarshalDepositLogEvent(receipt.Logs[0])
	require.NoError(t, err, "Could not reconstruct L2 Deposit")
	tx = types.NewTx(reconstructedDep)
	receipt, err = waitForTransaction(tx.Hash(), l2Client, 6*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.NoError(t, err)
	require.Equal(t, l2Opts.ExpectedStatus, receipt.Status)
}

type DepositTxOptsFn func(l2Opts *DepositTxOpts)

type DepositTxOpts struct {
	ToAddr         common.Address
	Value          *big.Int
	GasLimit       uint64
	IsCreation     bool
	Data           []byte
	ExpectedStatus uint64
}

func defaultDepositTxOpts(opts *bind.TransactOpts) *DepositTxOpts {
	return &DepositTxOpts{
		ToAddr:         opts.From,
		Value:          opts.Value,
		GasLimit:       1_000_000,
		IsCreation:     false,
		Data:           nil,
		ExpectedStatus: types.ReceiptStatusSuccessful,
	}
}
