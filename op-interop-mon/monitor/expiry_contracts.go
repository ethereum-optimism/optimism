package monitor

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Event topics of the contracts the expiry checks read. Each matches the event declared in
// packages/contracts-bedrock.
var (
	sentMessageTopic     = crypto.Keccak256Hash([]byte("SentMessage(uint256,address,uint256,address,bytes)"))
	relayedMessageTopic  = crypto.Keccak256Hash([]byte("RelayedMessage(uint256,uint256,bytes32,bytes32)"))
	messageExpiredTopic  = crypto.Keccak256Hash([]byte("MessageExpired(bytes32,uint256)"))
	undeliveredExpTopic  = crypto.Keccak256Hash([]byte("UndeliveredMessageExported(bytes32,uint256,address,uint256)"))
	sendETHTopic         = crypto.Keccak256Hash([]byte("SendETH(address,address,uint256,uint256)"))
	relayETHTopic        = crypto.Keccak256Hash([]byte("RelayETH(address,address,uint256,uint256)"))
	refundETHTopic       = crypto.Keccak256Hash([]byte("RefundETH(address,uint256,bytes32)"))
	liquidityBurnedTopic = crypto.Keccak256Hash([]byte("LiquidityBurned(address,uint256)"))
	liquidityMintedTopic = crypto.Keccak256Hash([]byte("LiquidityMinted(address,uint256)"))
	liquidityFundedTopic = crypto.Keccak256Hash([]byte("LiquidityFunded(address,uint256)"))

	relayETHSelector = crypto.Keccak256([]byte("relayETH(address,address,uint256)"))[:4]
)

// Storage slots of the L2ToL2CrossDomainMessenger, from its storage layout.
const (
	successfulMessagesSlot    = 0
	sentMessageTimestampsSlot = 3
	expiredMessagesSlot       = 4
	expiryPeriodSlot          = 5
)

var errMalformedLog = errors.New("malformed log")

var (
	uint256Type, _ = abi.NewType("uint256", "", nil)
	addressType, _ = abi.NewType("address", "", nil)
	bytesType, _   = abi.NewType("bytes", "", nil)

	l2ToL2MessageArgs = abi.Arguments{
		{Type: uint256Type}, {Type: uint256Type}, {Type: uint256Type},
		{Type: addressType}, {Type: addressType}, {Type: bytesType},
	}
	sentMessageDataArgs = abi.Arguments{{Type: addressType}, {Type: bytesType}}
	relayETHArgs        = abi.Arguments{{Type: addressType}, {Type: addressType}, {Type: uint256Type}}
	twoWordArgs         = abi.Arguments{{Type: uint256Type}, {Type: uint256Type}}
	exportDataArgs      = abi.Arguments{{Type: addressType}, {Type: uint256Type}}
)

// l2ToL2Message is an L2ToL2CrossDomainMessenger message, as read from the SentMessage event that
// initiated it on its source chain.
type l2ToL2Message struct {
	Source      eth.ChainID
	Destination eth.ChainID
	Nonce       *big.Int
	Sender      common.Address
	Target      common.Address
	Message     []byte
}

// Hash computes the message hash the same way as Hashing.hashL2toL2CrossDomainMessage:
// keccak256(abi.encode(destination, source, nonce, sender, target, message)).
func (m *l2ToL2Message) Hash() common.Hash {
	encoded, err := l2ToL2MessageArgs.Pack(m.Destination.ToBig(), m.Source.ToBig(), m.Nonce, m.Sender, m.Target, m.Message)
	if err != nil {
		// Pack only fails on a type mismatch, which the fixed argument types rule out.
		panic(fmt.Errorf("encode L2ToL2 message: %w", err))
	}
	return crypto.Keccak256Hash(encoded)
}

// decodeSentMessage decodes an L2ToL2CrossDomainMessenger SentMessage log emitted on `source`.
func decodeSentMessage(l *types.Log, source eth.ChainID) (*l2ToL2Message, error) {
	if l.Address != predeploys.L2toL2CrossDomainMessengerAddr || len(l.Topics) != 4 || l.Topics[0] != sentMessageTopic {
		return nil, fmt.Errorf("%w: not a SentMessage", errMalformedLog)
	}
	data, err := sentMessageDataArgs.Unpack(l.Data)
	if err != nil {
		return nil, fmt.Errorf("%w: SentMessage data: %w", errMalformedLog, err)
	}
	return &l2ToL2Message{
		Source:      source,
		Destination: eth.ChainIDFromBig(l.Topics[1].Big()),
		Target:      common.BytesToAddress(l.Topics[2].Bytes()),
		Nonce:       l.Topics[3].Big(),
		Sender:      data[0].(common.Address),
		Message:     data[1].([]byte),
	}, nil
}

// decodeRelayETH decodes SuperchainETHBridge.relayETH(from, to, amount) calldata.
func decodeRelayETH(calldata []byte) (from common.Address, to common.Address, amount *big.Int, err error) {
	if len(calldata) < 4 || string(calldata[:4]) != string(relayETHSelector) {
		return common.Address{}, common.Address{}, nil, errors.New("not a relayETH call")
	}
	args, err := relayETHArgs.Unpack(calldata[4:])
	if err != nil {
		return common.Address{}, common.Address{}, nil, fmt.Errorf("decode relayETH: %w", err)
	}
	return args[0].(common.Address), args[1].(common.Address), args[2].(*big.Int), nil
}

// messageExpiredEvent is a MessageExpired(messageHash, undeliveredAt) log.
type messageExpiredEvent struct {
	MessageHash   common.Hash
	UndeliveredAt uint64
}

func decodeMessageExpired(l *types.Log) (messageExpiredEvent, error) {
	if l.Address != predeploys.L2toL2CrossDomainMessengerAddr || len(l.Topics) != 2 || l.Topics[0] != messageExpiredTopic || len(l.Data) != 32 {
		return messageExpiredEvent{}, fmt.Errorf("%w: not a MessageExpired", errMalformedLog)
	}
	undeliveredAt := new(big.Int).SetBytes(l.Data)
	if !undeliveredAt.IsUint64() {
		return messageExpiredEvent{}, fmt.Errorf("%w: undeliveredAt %s overflows uint64", errMalformedLog, undeliveredAt)
	}
	return messageExpiredEvent{MessageHash: l.Topics[1], UndeliveredAt: bigs.Uint64Strict(undeliveredAt)}, nil
}

// refundETHEvent is a RefundETH(from, amount, messageHash) log.
type refundETHEvent struct {
	From        common.Address
	Amount      *big.Int
	MessageHash common.Hash
}

func decodeRefundETH(l *types.Log) (refundETHEvent, error) {
	if l.Address != predeploys.SuperchainETHBridgeAddr || len(l.Topics) != 3 || l.Topics[0] != refundETHTopic || len(l.Data) != 32 {
		return refundETHEvent{}, fmt.Errorf("%w: not a RefundETH", errMalformedLog)
	}
	return refundETHEvent{
		From:        common.BytesToAddress(l.Topics[1].Bytes()),
		Amount:      new(big.Int).SetBytes(l.Data),
		MessageHash: l.Topics[2],
	}, nil
}

// undeliveredExport is an UndeliveredMessageExported(messageHash, source, sourceMessenger,
// undeliveredAt) log.
type undeliveredExport struct {
	MessageHash     common.Hash
	Source          eth.ChainID
	SourceMessenger common.Address
	UndeliveredAt   *big.Int
}

func decodeUndeliveredExport(l *types.Log) (undeliveredExport, error) {
	if l.Address != predeploys.UndeliveredMessageExporterAddr || len(l.Topics) != 3 || l.Topics[0] != undeliveredExpTopic {
		return undeliveredExport{}, fmt.Errorf("%w: not an UndeliveredMessageExported", errMalformedLog)
	}
	data, err := exportDataArgs.Unpack(l.Data)
	if err != nil {
		return undeliveredExport{}, fmt.Errorf("%w: UndeliveredMessageExported data: %w", errMalformedLog, err)
	}
	return undeliveredExport{
		MessageHash:     l.Topics[1],
		Source:          eth.ChainIDFromBig(l.Topics[2].Big()),
		SourceMessenger: data[0].(common.Address),
		UndeliveredAt:   data[1].(*big.Int),
	}, nil
}

// sendETHEvent is a SendETH(from, to, amount, destination) log.
type sendETHEvent struct {
	From        common.Address
	To          common.Address
	Amount      *big.Int
	Destination *big.Int
}

func decodeSendETH(l *types.Log) (sendETHEvent, error) {
	if l.Address != predeploys.SuperchainETHBridgeAddr || len(l.Topics) != 3 || l.Topics[0] != sendETHTopic {
		return sendETHEvent{}, fmt.Errorf("%w: not a SendETH", errMalformedLog)
	}
	data, err := twoWordArgs.Unpack(l.Data)
	if err != nil {
		return sendETHEvent{}, fmt.Errorf("%w: SendETH data: %w", errMalformedLog, err)
	}
	return sendETHEvent{
		From:        common.BytesToAddress(l.Topics[1].Bytes()),
		To:          common.BytesToAddress(l.Topics[2].Bytes()),
		Amount:      data[0].(*big.Int),
		Destination: data[1].(*big.Int),
	}, nil
}

// liquidityAmount decodes the amount of an ETHLiquidity LiquidityBurned or LiquidityFunded log.
func liquidityAmount(l *types.Log, topic common.Hash) (*big.Int, bool) {
	if l.Address != predeploys.ETHLiquidityAddr || len(l.Topics) != 2 || l.Topics[0] != topic || len(l.Data) != 32 {
		return nil, false
	}
	return new(big.Int).SetBytes(l.Data), true
}

// mappingSlot returns the storage slot of mapping[key] for a mapping declared at `slot`.
func mappingSlot(key common.Hash, slot uint64) common.Hash {
	return crypto.Keccak256Hash(key.Bytes(), common.BigToHash(new(big.Int).SetUint64(slot)).Bytes())
}
