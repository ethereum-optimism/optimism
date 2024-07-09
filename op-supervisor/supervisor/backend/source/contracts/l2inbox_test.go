package contracts

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	backendTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestDecodeExecutingMessageEvent(t *testing.T) {
	inbox := NewCrossL2Inbox()
	payload := bytes.Repeat([]byte{0xaa, 0xbb}, 50)
	payloadHash := crypto.Keccak256Hash(payload)
	expected := backendTypes.ExecutingMessage{
		Chain:     42424,
		BlockNum:  12345,
		LogIdx:    98,
		Timestamp: 9578295,
		Hash:      backendTypes.TruncateHash(payloadHash),
	}
	contractIdent := contractIdentifier{
		Origin:      common.Address{0xbb, 0xcc},
		BlockNumber: new(big.Int).SetUint64(expected.BlockNum),
		LogIndex:    new(big.Int).SetUint64(uint64(expected.LogIdx)),
		Timestamp:   new(big.Int).SetUint64(expected.Timestamp),
		ChainId:     new(big.Int).SetUint64(uint64(expected.Chain)),
	}
	abi := snapshots.LoadCrossL2InboxABI()
	validData, err := abi.Events[eventExecutingMessage].Inputs.Pack(contractIdent, payload)
	require.NoError(t, err)
	createValidLog := func() *ethTypes.Log {
		return &ethTypes.Log{
			Address: predeploys.CrossL2InboxAddr,
			Topics:  []common.Hash{abi.Events[eventExecutingMessage].ID},
			Data:    validData,
		}
	}

	t.Run("ParseValid", func(t *testing.T) {
		l := createValidLog()
		result, err := inbox.DecodeExecutingMessageLog(l)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("IgnoreIncorrectContract", func(t *testing.T) {
		l := createValidLog()
		l.Address = common.Address{0xff}
		_, err := inbox.DecodeExecutingMessageLog(l)
		require.ErrorIs(t, err, ErrEventNotFound)
	})

	t.Run("IgnoreWrongEvent", func(t *testing.T) {
		l := createValidLog()
		l.Topics[0] = common.Hash{0xbb}
		_, err := inbox.DecodeExecutingMessageLog(l)
		require.ErrorIs(t, err, ErrEventNotFound)
	})

	t.Run("ErrorOnInvalidEvent", func(t *testing.T) {
		l := createValidLog()
		l.Data = []byte{0xbb, 0xcc}
		_, err := inbox.DecodeExecutingMessageLog(l)
		require.ErrorIs(t, err, batching.ErrInvalidEvent)
	})
}
