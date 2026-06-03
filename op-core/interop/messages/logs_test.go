package messages

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestDecodeExecutingMessageLog(t *testing.T) {
	data := `
{
  "address": "0x4200000000000000000000000000000000000022",
  "topics": [
    "0x5c37832d2e8d10e346e55ad62071a6a2f9fa5130614ef2ec6617555c6f467ba7",
    "0xc3f57e1f0dd62a4f77787d834029bfeaab8894022c47edbe13b044fb658c9190"
  ],
  "data": "0x0000000000000000000000005fbdb2315678afecb367f032d93f642f64180aa3000000000000000000000000000000000000000000000000000000000000119d0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006724d56300000000000000000000000000000000000000000000000000000000000dbc68",
  "blockHash": "0x355b82e9db9105fe3e7b7ed1897878ecbba8be7f30f94aca9dc55b6296a624cf",
  "blockNumber": "0x13a8",
  "transactionHash": "0x6eb22bb67562ac6a8fdbf60d6227c0b1f3f9d1d15ead1b0de055358f4fb9fa69",
  "transactionIndex": "0x2",
  "logIndex": "0x0",
  "removed": false
}
`
	var logEvent ethTypes.Log
	require.NoError(t, json.Unmarshal([]byte(data), &logEvent))

	msg, err := DecodeExecutingMessageLog(&logEvent)
	require.NoError(t, err)
	require.NotNil(t, msg)

	originAddr := common.HexToAddress("0x5fbdb2315678afecb367f032d93f642f64180aa3")
	payloadHash := common.HexToHash("0xc3f57e1f0dd62a4f77787d834029bfeaab8894022c47edbe13b044fb658c9190")
	logHash := PayloadHashToLogHash(payloadHash, originAddr)
	args := ChecksumArgs{
		BlockNumber: uint64(4509),
		LogIndex:    uint32(0),
		Timestamp:   uint64(1730467171),
		ChainID:     eth.ChainIDFromUInt64(900200),
		LogHash:     logHash,
	}
	checksum := args.Checksum()

	require.Equal(t, checksum, msg.Checksum)
	require.Equal(t, uint64(4509), msg.BlockNum)
	require.Equal(t, uint32(0), msg.LogIdx)
	require.Equal(t, uint64(1730467171), msg.Timestamp)
	require.Equal(t, eth.ChainIDFromUInt64(900200), msg.ChainID)
}

func TestLogToLogHash(t *testing.T) {
	mkLog := func() *ethTypes.Log {
		return &ethTypes.Log{
			Address: common.Address{0xaa, 0xbb},
			Topics: []common.Hash{
				{0xcc},
				{0xdd},
			},
			Data:        []byte{0xee, 0xff, 0x00},
			BlockNumber: 12345,
			TxHash:      common.Hash{0x11, 0x22, 0x33},
			TxIndex:     4,
			BlockHash:   common.Hash{0x44, 0x55},
			Index:       8,
			Removed:     false,
		}
	}
	relevantMods := []func(l *ethTypes.Log){
		func(l *ethTypes.Log) { l.Address = common.Address{0xab, 0xcd} },
		func(l *ethTypes.Log) { l.Topics = append(l.Topics, common.Hash{0x12, 0x34}) },
		func(l *ethTypes.Log) { l.Topics = l.Topics[:len(l.Topics)-1] },
		func(l *ethTypes.Log) { l.Topics[0] = common.Hash{0x12, 0x34} },
		func(l *ethTypes.Log) { l.Data = append(l.Data, 0x56) },
		func(l *ethTypes.Log) { l.Data = l.Data[:len(l.Data)-1] },
		func(l *ethTypes.Log) { l.Data[0] = 0x45 },
	}
	irrelevantMods := []func(l *ethTypes.Log){
		func(l *ethTypes.Log) { l.BlockNumber = 987 },
		func(l *ethTypes.Log) { l.TxHash = common.Hash{0xab, 0xcd} },
		func(l *ethTypes.Log) { l.TxIndex = 99 },
		func(l *ethTypes.Log) { l.BlockHash = common.Hash{0xab, 0xcd} },
		func(l *ethTypes.Log) { l.Index = 98 },
		func(l *ethTypes.Log) { l.Removed = true },
	}
	refHash := LogToLogHash(mkLog())
	expectedRefHash := common.HexToHash("0x4e1dc08fddeb273275f787762cdfe945cf47bb4e80a1fabbc7a825801e81b73f")
	require.Equal(t, expectedRefHash, refHash, "reference hash changed, check that database compatibility is not broken")

	for i, mod := range relevantMods {
		l := mkLog()
		mod(l)
		hash := LogToLogHash(l)
		require.NotEqualf(t, refHash, hash, "expected relevant modification %v to affect the hash but it did not", i)
	}
	for i, mod := range irrelevantMods {
		l := mkLog()
		mod(l)
		hash := LogToLogHash(l)
		require.Equal(t, refHash, hash, "expected irrelevant modification %v to not affect the hash but it did", i)
	}
}
