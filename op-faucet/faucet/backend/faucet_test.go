package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-faucet/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/mocks"
)

func TestFaucet(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	m := &metrics.NoopMetrics{}

	chainID := eth.ChainIDFromUInt64(123)
	fID := ftypes.FaucetID("foo")
	faucetAddr := common.HexToAddress("0x742d35Cc6634C0532925a3b8D0C9964E5Bade11E")
	intendedFaucetBalance := eth.HundredEther.Add(eth.Ether(1)) // 101 eth

	txMgr := mocks.NewTxManager(t)
	txMgr.On("From").Return(faucetAddr)
	txMgr.On("ChainID").Return(chainID)

	simulatedEthClient := testutils.NewSimulatedEthClient(testutils.WithAccountBalance(txMgr.From(), intendedFaucetBalance.ToBig()))
	f := faucetWithTxManager(logger, m, fID, txMgr, simulatedEthClient.Client)

	observedFaucetBalance, err := f.Balance()
	require.NoError(t, err, "faucet balance should be returned successfully")
	require.Equal(t, intendedFaucetBalance, observedFaucetBalance, "faucet balance should be 100 ether")

	require.Equal(t, chainID, f.ChainID())

	req := &ftypes.FaucetRequest{
		Target: common.HexToAddress("0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"),
		Amount: eth.Ether(50),
	}

	txMgr.On("Send", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			candidate := args.Get(1).(txmgr.TxCandidate)
			require.Nil(t, candidate.To, "must not do naive eth send")
			require.Equal(t, req.Amount, eth.WeiBig(candidate.Value))
		}).
		Return(&types.Receipt{Status: types.ReceiptStatusSuccessful, TxHash: common.Hash{}}, nil)

	require.NoError(t, f.RequestETH(context.Background(), req))

	req.Amount = intendedFaucetBalance.Add(eth.Ether(1)) // making a transfer beyond the faucet's balance
	require.ErrorContains(t, f.RequestETH(context.Background(), req), "insufficient balance")

	f.Disable()
	require.ErrorContains(t, f.RequestETH(context.Background(), req), "disabled")
	f.Enable()

	txMgr.On("Close").Once()
	f.Close()

	txMgr.AssertExpectations(t)
}
