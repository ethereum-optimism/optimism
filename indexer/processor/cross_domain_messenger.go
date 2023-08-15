package processor

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
	Uint256Type, _ = abi.NewType("uint256", "", nil)
	BytesType, _   = abi.NewType("bytes", "", nil)
	AddressType, _ = abi.NewType("address", "", nil)

	LegacyCrossDomainMessengerRelayMessageMethod = abi.NewMethod(
		"relayMessage",
		"relayMessage",
		abi.Function,
		"external", // mutability
		false,      // isConst
		true,       // payable
		abi.Arguments{ // inputs
			{Name: "sender", Type: AddressType},
			{Name: "target", Type: AddressType},
			{Name: "data", Type: BytesType},
			{Name: "nonce", Type: Uint256Type},
		},
		abi.Arguments{}, // outputs
	)
)

type CrossDomainMessengerSentMessageEvent struct {
	*bindings.CrossDomainMessengerSentMessage

	Value       *big.Int
	MessageHash common.Hash
	Event       *database.ContractEvent
}

type CrossDomainMessengerRelayedMessageEvent struct {
	*bindings.CrossDomainMessengerRelayedMessage
	Event *database.ContractEvent
}

func CrossDomainMessengerSentMessageEvents(contractAddress common.Address, chain string, db *database.DB, fromHeight, toHeight *big.Int) ([]CrossDomainMessengerSentMessageEvent, error) {
	crossDomainMessengerAbi, err := bindings.CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	sentMessageEventAbi := crossDomainMessengerAbi.Events["SentMessage"]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: sentMessageEventAbi.ID}
	sentMessageEvents, err := db.ContractEvents.ContractEventsWithFilter(contractEventFilter, chain, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}
	if len(sentMessageEvents) == 0 {
		// prevent the following db queries if we dont need them
		return nil, nil
	}

	sentMessageExtensionEventAbi := crossDomainMessengerAbi.Events["SentMessageExtension1"]
	contractEventFilter = database.ContractEvent{ContractAddress: contractAddress, EventSignature: sentMessageExtensionEventAbi.ID}
	sentMessageExtensionEvents, err := db.ContractEvents.ContractEventsWithFilter(contractEventFilter, chain, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}
	if len(sentMessageEvents) != len(sentMessageExtensionEvents) {
		return nil, fmt.Errorf("mismatch in SentMessage events. %d sent messages & %d sent message extensions", len(sentMessageEvents), len(sentMessageExtensionEvents))
	}

	crossDomainSentMessages := make([]CrossDomainMessengerSentMessageEvent, len(sentMessageEvents))
	for i := range sentMessageEvents {
		var sentMessage bindings.CrossDomainMessengerSentMessage
		err = UnpackLog(&sentMessage, sentMessageEvents[i].RLPLog, sentMessageEventAbi.Name, crossDomainMessengerAbi)
		if err != nil {
			return nil, err
		}
		var sentMessageExtension bindings.CrossDomainMessengerSentMessageExtension1
		err = UnpackLog(&sentMessageExtension, sentMessageExtensionEvents[i].RLPLog, sentMessageExtensionEventAbi.Name, crossDomainMessengerAbi)
		if err != nil {
			return nil, err
		}

		msgHash, err := CrossDomainMessageHash(crossDomainMessengerAbi, &sentMessage, sentMessageExtension.Value)
		if err != nil {
			return nil, err
		}

		crossDomainSentMessages[i] = CrossDomainMessengerSentMessageEvent{
			CrossDomainMessengerSentMessage: &sentMessage,
			Value:                           sentMessageExtension.Value,
			MessageHash:                     msgHash,
			Event:                           &sentMessageEvents[i],
		}

	}

	return crossDomainSentMessages, nil
}

func CrossDomainMessengerRelayedMessageEvents(contractAddress common.Address, chain string, db *database.DB, fromHeight, toHeight *big.Int) ([]CrossDomainMessengerRelayedMessageEvent, error) {
	crossDomainMessengerABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	relayedMessageEventAbi := crossDomainMessengerABI.Events["RelayedMessage"]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: relayedMessageEventAbi.ID}
	relayedMessageEvents, err := db.ContractEvents.ContractEventsWithFilter(contractEventFilter, chain, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	crossDomainRelayedMessages := make([]CrossDomainMessengerRelayedMessageEvent, len(relayedMessageEvents))
	for i := range relayedMessageEvents {
		relayedMsgData := bindings.CrossDomainMessengerRelayedMessage{Raw: *relayedMessageEvents[i].RLPLog}
		err = UnpackLog(&relayedMsgData, relayedMessageEvents[i].RLPLog, relayedMessageEventAbi.Name, crossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		crossDomainRelayedMessages[i] = CrossDomainMessengerRelayedMessageEvent{
			CrossDomainMessengerRelayedMessage: &relayedMsgData,
			Event:                              &relayedMessageEvents[i],
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
		inputBytes, err := LegacyCrossDomainMessengerRelayMessageMethod.Inputs.Pack(sentMsg.Sender, sentMsg.Target, sentMsg.Message, sentMsg.MessageNonce)
		if err != nil {
			return common.Hash{}, err
		}
		msgBytes := append(LegacyCrossDomainMessengerRelayMessageMethod.ID, inputBytes...)
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
