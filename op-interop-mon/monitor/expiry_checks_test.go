package monitor

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestL2ToL2MessageHash checks the Go message hash against vectors computed with
// `cast keccak $(cast abi-encode "f(uint256,uint256,uint256,address,address,bytes)" ...)`,
// the encoding of Hashing.hashL2toL2CrossDomainMessage.
func TestL2ToL2MessageHash(t *testing.T) {
	bridgeSend := &l2ToL2Message{
		Destination: eth.ChainIDFromUInt64(902),
		Source:      eth.ChainIDFromUInt64(901),
		Nonce:       big.NewInt(0),
		Sender:      predeploys.SuperchainETHBridgeAddr,
		Target:      predeploys.SuperchainETHBridgeAddr,
		Message: relayETHCalldata(common.HexToAddress("0x1111111111111111111111111111111111111111"),
			common.HexToAddress("0x2222222222222222222222222222222222222222"), big.NewInt(500_000_000_000_000_000)),
	}
	require.Equal(t,
		common.FromHex("0x4f0edcc90000000000000000000000001111111111111111111111111111111111111111000000000000000000000000222222222222222222222222222222222222222200000000000000000000000000000000000000000000000006f05b59d3b20000"),
		bridgeSend.Message, "relayETH calldata")
	require.Equal(t, common.HexToHash("0x688e2e5ba73c4dcb671c018b4a8578efbcce3728f02027cfde24bfcc11997bcb"), bridgeSend.Hash())

	empty := &l2ToL2Message{
		Destination: eth.ChainIDFromUInt64(10),
		Source:      eth.ChainIDFromUInt64(8453),
		Nonce:       big.NewInt(7),
		Sender:      common.HexToAddress("0xaa"),
		Target:      common.HexToAddress("0xbb"),
		Message:     []byte{},
	}
	require.Equal(t, common.HexToHash("0xe227942a742fddd04cd4c9a3d9a04dd83ca96b6ee92044deb3799db2dbbab4ed"), empty.Hash())

	// The messenger's SENT_MESSAGE_EVENT_SELECTOR constant.
	require.Equal(t, common.HexToHash("0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f320"), sentMessageTopic)
}

// TestMappingSlot checks a mapping slot against `cast index bytes32 0x...abc 3`.
func TestMappingSlot(t *testing.T) {
	require.Equal(t, common.HexToHash("0x0d25afa9aa2d9502df46ff124ae1e83a8b4c1908d456f3f2cc12ac5cd4abd2b1"),
		mappingSlot(common.HexToHash("0xabc"), sentMessageTimestampsSlot))
}

func TestDecodeSentMessageRoundTrip(t *testing.T) {
	s := newExpiryScenario()
	got, err := decodeSentMessage(sentMessageLog(s.msg), s.a.id)
	require.NoError(t, err)
	require.Equal(t, s.hash, got.Hash())
}

func reasons(vs []violation) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.reason)
	}
	return out
}

func verifyExpiry(t *testing.T, s *expiryScenario) []string {
	vs, err := s.verifier().verifyExpiry(context.Background(), s.a.id, s.expiredLog)
	require.NoError(t, err)
	return reasons(vs)
}

func verifyRefund(t *testing.T, s *expiryScenario, l *types.Log) []string {
	vs, err := s.verifier().verifyRefund(context.Background(), s.a.id, l)
	require.NoError(t, err)
	return reasons(vs)
}

func TestVerifyExpiry(t *testing.T) {
	t.Run("valid expiry", func(t *testing.T) {
		require.Empty(t, verifyExpiry(t, newExpiryScenario()))
	})

	t.Run("expiry with no export", func(t *testing.T) {
		s := newExpiryScenario()
		s.b.removeLogs(undeliveredExpTopic)
		require.Equal(t, []string{reasonNoExport}, verifyExpiry(t, s))
	})

	t.Run("export for another source", func(t *testing.T) {
		s := newExpiryScenario()
		s.b.removeLogs(undeliveredExpTopic)
		s.b.addLog(s.exportBlock, txHash("export"), exportLog(s.hash, eth.ChainIDFromUInt64(999), common.HexToAddress("0xa1"), s.exportT))
		require.Equal(t, []string{reasonNoExport}, verifyExpiry(t, s))
	})

	t.Run("relay within the window", func(t *testing.T) {
		s := newExpiryScenario()
		relayBlock := s.b.blockAt(s.sentAt + 4)
		s.b.addLog(relayBlock, txHash("relay"), relayedMessageLog(s.a.id, s.msg.Nonce, s.hash))
		s.b.setStorage(predeploys.L2toL2CrossDomainMessengerAddr, mappingSlot(s.hash, successfulMessagesSlot), relayBlock, big.NewInt(1))
		require.Equal(t, []string{reasonRelayedInWindow, reasonRelayedAtHead}, verifyExpiry(t, s))
	})

	t.Run("relay after the expiry", func(t *testing.T) {
		// Outside the window, so only the destination's storage shows it; the window check of the
		// executing message (check 1) flags the relay itself as expired.
		s := newExpiryScenario()
		relayBlock := s.b.blockAt(s.a.headers[s.expireBlock].Time) + 5
		s.b.addLog(relayBlock, txHash("relay"), relayedMessageLog(s.a.id, s.msg.Nonce, s.hash))
		s.b.setStorage(predeploys.L2toL2CrossDomainMessengerAddr, mappingSlot(s.hash, successfulMessagesSlot), relayBlock, big.NewInt(1))
		require.Equal(t, []string{reasonRelayedAtHead}, verifyExpiry(t, s))
	})

	t.Run("hash mismatch", func(t *testing.T) {
		s := newExpiryScenario()
		s.a.removeLogs(sentMessageTopic)
		wrong := *s.msg
		wrong.Message = relayETHCalldata(s.sender, s.to, big.NewInt(1))
		s.a.addLog(s.sendBlock, txHash("send"), sentMessageLog(&wrong))
		require.Equal(t, []string{reasonHashMismatch}, verifyExpiry(t, s))
	})

	t.Run("message never sent", func(t *testing.T) {
		s := newExpiryScenario()
		s.a.storage[[2]common.Hash{common.BytesToHash(predeploys.L2toL2CrossDomainMessengerAddr.Bytes()),
			mappingSlot(s.hash, sentMessageTimestampsSlot)}] = nil
		require.Equal(t, []string{reasonUnknownMessage}, verifyExpiry(t, s))
	})

	t.Run("expiry before the expiry period passed", func(t *testing.T) {
		s := newExpiryScenario()
		early := s.sentAt + scenarioPeriod
		s.b.removeLogs(undeliveredExpTopic)
		s.b.addLog(s.b.blockAt(early), txHash("export"), exportLog(s.hash, s.a.id, common.HexToAddress("0xa1"), early))
		s.expiredLog.Data = word(new(big.Int).SetUint64(early))
		require.Equal(t, []string{reasonEarlyExpiry}, verifyExpiry(t, s))
	})

	t.Run("expiry on a fact from the future", func(t *testing.T) {
		s := newExpiryScenario()
		future := s.a.headers[len(s.a.headers)-1].Time + 1000
		s.expiredLog.Data = word(new(big.Int).SetUint64(future))
		require.Equal(t, []string{reasonUndeliveredAtInFuture}, verifyExpiry(t, s))
	})

	t.Run("message to an unmonitored chain", func(t *testing.T) {
		s := newExpiryScenario()
		v := s.verifier()
		delete(v.clients, s.b.id)
		vs, err := v.verifyExpiry(context.Background(), s.a.id, s.expiredLog)
		require.NoError(t, err)
		require.Equal(t, []string{reasonUnknownDestination}, reasons(vs))
	})

	t.Run("waits for the export to be safe", func(t *testing.T) {
		s := newExpiryScenario()
		s.b.safe = s.exportBlock - 1
		_, err := s.verifier().verifyExpiry(context.Background(), s.a.id, s.expiredLog)
		require.ErrorIs(t, err, errNotReady)
	})
}

func TestVerifyRefund(t *testing.T) {
	t.Run("valid refund", func(t *testing.T) {
		s := newExpiryScenario()
		require.Empty(t, verifyRefund(t, s, s.refundLog))
	})

	t.Run("refund with no expiry", func(t *testing.T) {
		s := newExpiryScenario()
		s.a.storage[[2]common.Hash{common.BytesToHash(predeploys.L2toL2CrossDomainMessengerAddr.Bytes()),
			mappingSlot(s.hash, expiredMessagesSlot)}] = nil
		require.Equal(t, []string{reasonRefundWithoutExpiry}, verifyRefund(t, s, s.refundLog))
	})

	t.Run("second refund", func(t *testing.T) {
		s := newExpiryScenario()
		second := s.a.addLog(s.refundBlock+5, txHash("refund again"), refundETHLog(s.sender, s.amount, s.hash))
		require.Empty(t, verifyRefund(t, s, s.refundLog), "the first refund is valid")
		require.Equal(t, []string{reasonDuplicateRefund}, verifyRefund(t, s, second))
	})

	t.Run("refund of another amount", func(t *testing.T) {
		s := newExpiryScenario()
		s.refundLog.Data = word(new(big.Int).Mul(s.amount, big.NewInt(2)))
		require.Equal(t, []string{reasonRefundMismatch}, verifyRefund(t, s, s.refundLog))
	})

	t.Run("refund to another sender", func(t *testing.T) {
		s := newExpiryScenario()
		s.refundLog.Topics[1] = addrTopic(common.HexToAddress("0xbad"))
		require.Equal(t, []string{reasonRefundMismatch}, verifyRefund(t, s, s.refundLog))
	})

	t.Run("send that burned nothing", func(t *testing.T) {
		s := newExpiryScenario()
		s.a.removeLogs(liquidityBurnedTopic)
		require.Equal(t, []string{reasonNoBurn}, verifyRefund(t, s, s.refundLog))
	})

	t.Run("refund of a message that is not a bridge send", func(t *testing.T) {
		s := newExpiryScenario()
		s.a.removeLogs(sentMessageTopic)
		s.msg.Sender = common.HexToAddress("0xc0de")
		s.hash = s.msg.Hash()
		s.a.addLog(s.sendBlock, txHash("send"), sentMessageLog(s.msg))
		s.a.setStorage(predeploys.L2toL2CrossDomainMessengerAddr, mappingSlot(s.hash, sentMessageTimestampsSlot), s.sendBlock, new(big.Int).SetUint64(s.sentAt))
		s.a.setStorage(predeploys.L2toL2CrossDomainMessengerAddr, mappingSlot(s.hash, expiredMessagesSlot), s.expireBlock, big.NewInt(1))
		s.refundLog.Topics[2] = s.hash
		require.Equal(t, []string{reasonNotBridgeSend}, verifyRefund(t, s, s.refundLog))
	})

	t.Run("hash mismatch", func(t *testing.T) {
		s := newExpiryScenario()
		s.a.removeLogs(sentMessageTopic)
		require.Equal(t, []string{reasonHashMismatch}, verifyRefund(t, s, s.refundLog))
	})
}
