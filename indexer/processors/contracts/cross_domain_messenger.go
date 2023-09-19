package contracts

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
)

var (
	// Standard ABI types copied from golang ABI tests
	uint256Type, _ = abi.NewType("uint256", "", nil)
	bytesType, _   = abi.NewType("bytes", "", nil)
	addressType, _ = abi.NewType("address", "", nil)

	CrossDomainMessengerLegacyRelayMessageEncoding = abi.NewMethod(
		"relayMessage",
		"relayMessage",
		abi.Function,
		"external", // mutability
		false,      // isConst
		true,       // payable
		abi.Arguments{ // inputs
			{Name: "target", Type: addressType},
			{Name: "sender", Type: addressType},
			{Name: "data", Type: bytesType},
			{Name: "nonce", Type: uint256Type},
			// The actual transaction on the legacy L1CrossDomainMessenger has a trailing
			// proof argument but is ignored for the `XDomainCallData` encoding
		},
		abi.Arguments{}, // outputs
	)
)

type CrossDomainMessengerSentMessageEvent struct {
	Event           *database.ContractEvent
	MessageCalldata []byte
	BridgeMessage   database.BridgeMessage
}

type CrossDomainMessengerRelayedMessageEvent struct {
	Event       *database.ContractEvent
	MessageHash common.Hash
}

func CrossDomainMessengerSentMessageEvents(chainSelector string, contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]CrossDomainMessengerSentMessageEvent, error) {
	crossDomainMessengerAbi, err := bindings.CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	sentMessageEventAbi := crossDomainMessengerAbi.Events["SentMessage"]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: sentMessageEventAbi.ID}
	sentMessageEvents, err := db.ContractEvents.ContractEventsWithFilter(contractEventFilter, chainSelector, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}
	if len(sentMessageEvents) == 0 {
		return nil, nil
	}

	sentMessageExtensionEventAbi := crossDomainMessengerAbi.Events["SentMessageExtension1"]
	contractEventFilter = database.ContractEvent{ContractAddress: contractAddress, EventSignature: sentMessageExtensionEventAbi.ID}
	sentMessageExtensionEvents, err := db.ContractEvents.ContractEventsWithFilter(contractEventFilter, chainSelector, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}
	if len(sentMessageExtensionEvents) > len(sentMessageEvents) {
		return nil, fmt.Errorf("mismatch. %d sent messages & %d sent message extensions", len(sentMessageEvents), len(sentMessageExtensionEvents))
	}

	// We handle version zero messages uniquely since the first version of cross domain messages
	// do not have the SentMessageExtension1 event emitted, introduced in version 1.
	numVersionZeroMessages := len(sentMessageEvents) - len(sentMessageExtensionEvents)
	crossDomainSentMessages := make([]CrossDomainMessengerSentMessageEvent, len(sentMessageEvents))
	for i := range sentMessageEvents {
		sentMessage := bindings.CrossDomainMessengerSentMessage{Raw: *sentMessageEvents[i].RLPLog}
		err = UnpackLog(&sentMessage, sentMessageEvents[i].RLPLog, sentMessageEventAbi.Name, crossDomainMessengerAbi)
		if err != nil {
			return nil, err
		}

		version, _ := DecodeVersionedNonce(sentMessage.MessageNonce)
		if i < numVersionZeroMessages && version != 0 {
			return nil, fmt.Errorf("expected version zero nonce. nonce %d tx_hash %s", sentMessage.MessageNonce, sentMessage.Raw.TxHash)
		}

		// In version zero, to value is bridged through the cross domain messenger.
		value := big.NewInt(0)
		if version > 0 {
			sentMessageExtension := bindings.CrossDomainMessengerSentMessageExtension1{Raw: *sentMessageExtensionEvents[i].RLPLog}
			err = UnpackLog(&sentMessageExtension, sentMessageExtensionEvents[i].RLPLog, sentMessageExtensionEventAbi.Name, crossDomainMessengerAbi)
			if err != nil {
				return nil, err
			}
			value = sentMessageExtension.Value
		}

		messageCalldata, err := CrossDomainMessageCalldata(crossDomainMessengerAbi, &sentMessage, value)
		if err != nil {
			return nil, err
		}

		crossDomainSentMessages[i] = CrossDomainMessengerSentMessageEvent{
			Event:           &sentMessageEvents[i],
			MessageCalldata: messageCalldata,
			BridgeMessage: database.BridgeMessage{
				MessageHash:          crypto.Keccak256Hash(messageCalldata),
				Nonce:                sentMessage.MessageNonce,
				SentMessageEventGUID: sentMessageEvents[i].GUID,
				GasLimit:             sentMessage.GasLimit,
				Tx: database.Transaction{
					FromAddress: sentMessage.Sender,
					ToAddress:   sentMessage.Target,
					Amount:      value,
					Data:        sentMessage.Message,
					Timestamp:   sentMessageEvents[i].Timestamp,
				},
			},
		}
	}

	return crossDomainSentMessages, nil
}

func CrossDomainMessengerRelayedMessageEvents(chainSelector string, contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]CrossDomainMessengerRelayedMessageEvent, error) {
	crossDomainMessengerAbi, err := bindings.CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	relayedMessageEventAbi := crossDomainMessengerAbi.Events["RelayedMessage"]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: relayedMessageEventAbi.ID}
	relayedMessageEvents, err := db.ContractEvents.ContractEventsWithFilter(contractEventFilter, chainSelector, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	crossDomainRelayedMessages := make([]CrossDomainMessengerRelayedMessageEvent, len(relayedMessageEvents))
	for i := range relayedMessageEvents {
		relayedMessage := bindings.CrossDomainMessengerRelayedMessage{Raw: *relayedMessageEvents[i].RLPLog}
		err = UnpackLog(&relayedMessage, relayedMessageEvents[i].RLPLog, relayedMessageEventAbi.Name, crossDomainMessengerAbi)
		if err != nil {
			return nil, err
		}

		crossDomainRelayedMessages[i] = CrossDomainMessengerRelayedMessageEvent{
			Event:       &relayedMessageEvents[i],
			MessageHash: relayedMessage.MsgHash,
		}
	}

	return crossDomainRelayedMessages, nil
}

// Replica of `Encoding.sol#encodeCrossDomainMessage` solidity implementation
func CrossDomainMessageCalldata(abi *abi.ABI, sentMsg *bindings.CrossDomainMessengerSentMessage, value *big.Int) ([]byte, error) {
	version, _ := DecodeVersionedNonce(sentMsg.MessageNonce)
	switch version {
	case 0:
		// Legacy Message
		inputBytes, err := CrossDomainMessengerLegacyRelayMessageEncoding.Inputs.Pack(sentMsg.Target, sentMsg.Sender, sentMsg.Message, sentMsg.MessageNonce)
		if err != nil {
			return nil, err
		}
		return append(CrossDomainMessengerLegacyRelayMessageEncoding.ID, inputBytes...), nil
	case 1:
		// Current Message
		msgBytes, err := abi.Pack("relayMessage", sentMsg.MessageNonce, sentMsg.Sender, sentMsg.Target, value, sentMsg.GasLimit, sentMsg.Message)
		if err != nil {
			return nil, err
		}
		return msgBytes, nil
	}

	return nil, fmt.Errorf("unsupported cross domain messenger version: %d", version)
}
