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

	legacyCrossDomainMessengerRelayMessageMethod = abi.NewMethod(
		"relayMessage",
		"relayMessage",
		abi.Function,
		"external", // mutability
		false,      // isConst
		true,       // payable
		abi.Arguments{ // inputs
			{Name: "sender", Type: addressType},
			{Name: "target", Type: addressType},
			{Name: "data", Type: bytesType},
			{Name: "nonce", Type: uint256Type},
		},
		abi.Arguments{}, // outputs
	)
)

type CrossDomainMessengerSentMessageEvent struct {
	Event         *database.ContractEvent
	BridgeMessage database.BridgeMessage
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
		// prevent the following db queries if we dont need them
		return nil, nil
	}

	sentMessageExtensionEventAbi := crossDomainMessengerAbi.Events["SentMessageExtension1"]
	contractEventFilter = database.ContractEvent{ContractAddress: contractAddress, EventSignature: sentMessageExtensionEventAbi.ID}
	sentMessageExtensionEvents, err := db.ContractEvents.ContractEventsWithFilter(contractEventFilter, chainSelector, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}
	if len(sentMessageEvents) != len(sentMessageExtensionEvents) {
		return nil, fmt.Errorf("mismatch in SentMessage events. %d sent messages & %d sent message extensions", len(sentMessageEvents), len(sentMessageExtensionEvents))
	}

	crossDomainSentMessages := make([]CrossDomainMessengerSentMessageEvent, len(sentMessageEvents))
	for i := range sentMessageEvents {
		sentMessage := bindings.CrossDomainMessengerSentMessage{Raw: *sentMessageEvents[i].RLPLog}
		err = UnpackLog(&sentMessage, sentMessageEvents[i].RLPLog, sentMessageEventAbi.Name, crossDomainMessengerAbi)
		if err != nil {
			return nil, err
		}
		sentMessageExtension := bindings.CrossDomainMessengerSentMessageExtension1{Raw: *sentMessageExtensionEvents[i].RLPLog}
		err = UnpackLog(&sentMessageExtension, sentMessageExtensionEvents[i].RLPLog, sentMessageExtensionEventAbi.Name, crossDomainMessengerAbi)
		if err != nil {
			return nil, err
		}

		messageHash, err := CrossDomainMessageHash(crossDomainMessengerAbi, &sentMessage, sentMessageExtension.Value)
		if err != nil {
			return nil, err
		}

		crossDomainSentMessages[i] = CrossDomainMessengerSentMessageEvent{
			Event: &sentMessageEvents[i],
			BridgeMessage: database.BridgeMessage{
				MessageHash:          messageHash,
				Nonce:                sentMessage.MessageNonce,
				SentMessageEventGUID: sentMessageEvents[i].GUID,
				GasLimit:             sentMessage.GasLimit,
				Tx: database.Transaction{
					FromAddress: sentMessage.Sender,
					ToAddress:   sentMessage.Target,
					Amount:      sentMessageExtension.Value,
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

// Replica of `Hashing.sol#hashCrossDomainMessage` solidity implementation
func CrossDomainMessageHash(abi *abi.ABI, sentMsg *bindings.CrossDomainMessengerSentMessage, value *big.Int) (common.Hash, error) {
	version, _ := DecodeVersionedNonce(sentMsg.MessageNonce)
	switch version {
	case 0:
		// Legacy Message
		inputBytes, err := legacyCrossDomainMessengerRelayMessageMethod.Inputs.Pack(sentMsg.Sender, sentMsg.Target, sentMsg.Message, sentMsg.MessageNonce)
		if err != nil {
			return common.Hash{}, err
		}
		msgBytes := append(legacyCrossDomainMessengerRelayMessageMethod.ID, inputBytes...)
		return crypto.Keccak256Hash(msgBytes), nil
	case 1:
		// Current Message
		msgBytes, err := abi.Pack("relayMessage", sentMsg.MessageNonce, sentMsg.Sender, sentMsg.Target, value, sentMsg.GasLimit, sentMsg.Message)
		if err != nil {
			return common.Hash{}, err
		}
		return crypto.Keccak256Hash(msgBytes), nil
	}

	return common.Hash{}, fmt.Errorf("unsupported cross domain messenger version: %d", version)
}
