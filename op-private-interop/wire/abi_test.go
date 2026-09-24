package wire

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestEncodeSentMessageDataMatchesABI(t *testing.T) {
	for _, n := range []int{0, 1, 31, 32, 33, 64, 1000} {
		msg := bytes.Repeat([]byte{0xab}, n)
		want, err := sentMessageDataArgs.Pack(common.Address{0x12}, msg)
		require.NoError(t, err)
		require.Equal(t, want, EncodeSentMessageData(common.Address{0x12}, msg), "len %d", n)
	}
}

func TestSentMessageLogRoundTrips(t *testing.T) {
	m := &SentMessage{Destination: big.NewInt(902), Nonce: new(big.Int).Lsh(big.NewInt(1), 200), Sender: common.Address{1}, Target: common.Address{2}, Message: []byte("hello")}
	topics, data := SentMessageLog(m)
	require.Equal(t, SentMessageEventTopic, topics[0])
	require.Equal(t, common.BigToHash(m.Destination), topics[1])
	require.Equal(t, common.BytesToHash(m.Target[:]), topics[2])
	require.Equal(t, common.BigToHash(m.Nonce), topics[3])
	decoded, err := DecodeSentMessage(topics, data)
	require.NoError(t, err)
	require.Equal(t, m, decoded)
	// The replay calldata carries the same fields.
	calldata, err := EncodeReplaySentMessage(m)
	require.NoError(t, err)
	replayed, err := DecodeReplaySentMessage(calldata)
	require.NoError(t, err)
	t2, d2 := SentMessageLog(replayed)
	require.Equal(t, topics, t2)
	require.Equal(t, data, d2)
}
