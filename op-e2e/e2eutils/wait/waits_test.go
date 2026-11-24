package wait

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type failedTxEth struct{ hash common.Hash }

func (b *failedTxEth) GetTransactionReceipt(_ context.Context, hash common.Hash) (*optypes.Receipt, error) {
	return &optypes.Receipt{Receipt: types.Receipt{
		Status: types.ReceiptStatusFailed,
		TxHash: b.hash,
		Logs:   []*types.Log{},
	}}, nil
}

type revertTracer struct{}

func (revertTracer) TraceTransaction(_ context.Context, _ hexutil.Bytes, _ map[string]any) (*TxTrace, error) {
	return &TxTrace{CallTrace: CallTrace{Error: "execution reverted"}}, nil
}

// TestForOPReceiptAttachesTrace checks that a receipt with an unexpected status comes back with
// the tx's call trace, as ForReceipt does for go-ethereum receipts.
func TestForOPReceiptAttachesTrace(t *testing.T) {
	t.Parallel()
	hash := common.HexToHash("0x01")
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	require.NoError(t, srv.RegisterName("eth", &failedTxEth{hash: hash}))
	require.NoError(t, srv.RegisterName("debug", revertTracer{}))
	cl, err := sources.NewEthClient(client.NewBaseRPCClient(rpc.DialInProc(srv)),
		testlog.Logger(t, log.LevelError), nil, sources.DefaultEthClientConfig(10))
	require.NoError(t, err)
	t.Cleanup(cl.Close)

	receipt, err := ForOPReceiptOK(context.Background(), cl, hash)
	require.NotNil(t, receipt)
	var statusErr *ReceiptStatusError
	require.ErrorAs(t, err, &statusErr)
	require.Equal(t, types.ReceiptStatusFailed, statusErr.Status)
	require.Equal(t, "execution reverted", statusErr.TxTrace.Error)
}
