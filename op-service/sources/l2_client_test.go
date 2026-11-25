package sources

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestL2ClientOutputV0MissingWithdrawalsRoot(t *testing.T) {
	hdr := &RPCHeader{
		UncleHash: types.EmptyUncleHash,
		Number:    123,
		Time:      1_700_000_000,
		BaseFee:   (*hexutil.Big)(big.NewInt(7)),
	}
	hdr.Hash = hdr.computeBlockHash()

	m := new(mockRPC)
	defer m.AssertExpectations(t)
	m.On("CallContext", mock.Anything, new(*RPCHeader), "eth_getBlockByNumber", []any{hexutil.EncodeUint64(123), false}).Run(func(args mock.Arguments) {
		*args[1].(**RPCHeader) = hdr
	}).Return([]error{nil}).Once()
	s, err := NewL2Client(m, testlog.Logger(t, log.LevelInfo), nil, L2ClientSimpleConfig(&rollup.Config{IsthmusTime: new(uint64)}, false, 10, 10))
	require.NoError(t, err)

	_, err = s.OutputV0AtBlockNumber(t.Context(), 123)
	require.ErrorContains(t, err, "missing withdrawals root")
}
