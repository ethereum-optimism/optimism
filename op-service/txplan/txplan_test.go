package txplan

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestPlannedTx_Defaults(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	ptx := NewPlannedTx(WithPrivateKey(key), WithValue(eth.WeiU64(123)))
	t.Log("tx", ptx.Signed.String())

	block := types.NewBlock(&types.Header{BaseFee: big.NewInt(7e9)}, nil, nil, nil, types.DefaultBlockConfig)
	blockInfo := eth.BlockToInfo(block)
	ptx.AgainstBlock.Set(blockInfo)

	expectedAddr := crypto.PubkeyToAddress(key.PublicKey)
	signer := types.LatestSignerForChainID(big.NewInt(1))
	{
		tx, err := ptx.Signed.Eval(context.Background())
		require.NoError(t, err)

		sender, err := signer.Sender(tx)
		require.NoError(t, err)
		require.Equal(t, expectedAddr, sender)

		require.Equal(t, big.NewInt(123), tx.Value())
	}

	// Get a new signed tx
	ptx.Value.Set(big.NewInt(42))
	{
		tx, err := ptx.Signed.Eval(context.Background())
		require.NoError(t, err)

		sender, err := signer.Sender(tx)
		require.NoError(t, err)
		require.Equal(t, expectedAddr, sender)

		require.Equal(t, big.NewInt(42), tx.Value())
	}
}

type mockReader struct {
	msg         ethereum.CallMsg
	blockNumber rpc.BlockNumber
}

func (m *mockReader) Call(_ context.Context, msg ethereum.CallMsg, blockNumber rpc.BlockNumber) ([]byte, error) {
	m.msg = msg
	m.blockNumber = blockNumber
	return []byte{0x01}, nil
}

func TestWithReaderDoesNotUseDefaultFeeCaps(t *testing.T) {
	reader := new(mockReader)
	to := common.Address{0x01}
	from := common.Address{0x02}
	accessList := types.AccessList{
		{Address: common.Address{0x03}, StorageKeys: []common.Hash{{0x04}}},
	}
	block := types.NewBlock(&types.Header{Number: big.NewInt(7), BaseFee: big.NewInt(8)}, nil, nil, nil, types.DefaultBlockConfig)

	ptx := NewPlannedTx(
		WithReader(reader),
		WithSender(from),
		WithTo(&to),
		WithValue(eth.WeiU64(9)),
		WithData([]byte{0x0a}),
	)
	ptx.AccessList.Set(accessList)
	ptx.AgainstBlock.Set(eth.BlockToInfo(block))

	output, err := ptx.Read.Eval(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte{0x01}, output)

	require.Equal(t, from, reader.msg.From)
	require.Equal(t, &to, reader.msg.To)
	require.Zero(t, reader.msg.Gas)
	require.Nil(t, reader.msg.GasPrice)
	require.Nil(t, reader.msg.GasFeeCap)
	require.Nil(t, reader.msg.GasTipCap)
	require.Equal(t, big.NewInt(9), reader.msg.Value)
	require.Equal(t, []byte{0x0a}, reader.msg.Data)
	require.Equal(t, accessList, reader.msg.AccessList)
	require.Equal(t, rpc.BlockNumber(7), reader.blockNumber)
}
