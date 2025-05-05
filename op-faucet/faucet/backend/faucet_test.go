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
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/mocks"
)

func TestFaucet(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	m := &metrics.NoopMetrics{}
	fID := ftypes.FaucetID("foo")
	txMgr := mocks.NewTxManager(t)
	chainID := eth.ChainIDFromUInt64(123)
	txMgr.On("ChainID").Return(chainID)
	f := faucetWithTxManager(logger, m, fID, txMgr)
	require.Equal(t, chainID, f.ChainID())

	req := &ftypes.FaucetRequest{
		Target: common.HexToAddress("0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"),
		Amount: eth.Ether(123),
	}

	txMgr.On("Send", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			candidate := args.Get(1).(txmgr.TxCandidate)
			require.Nil(t, candidate.To, "must not do naive eth send")
			require.Equal(t, req.Amount, eth.WeiBig(candidate.Value))
		}).
		Return(&types.Receipt{Status: types.ReceiptStatusSuccessful, TxHash: common.Hash{}}, nil).
		Once()

	require.NoError(t, f.RequestETH(context.Background(), req))

	f.Disable()
	require.ErrorContains(t, f.RequestETH(context.Background(), req), "disabled")
	f.Enable()

	txMgr.On("Close").Once()
	f.Close()

	txMgr.AssertExpectations(t)
}
