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
	RawEvent    *database.ContractEvent
}

type CrossDomainMessengerRelayedMessageEvent struct {
	*bindings.CrossDomainMessengerRelayedMessage
	RawEvent *database.ContractEvent
}

func CrossDomainMessengerSentMessageEvents(events *ProcessedContractEvents) ([]CrossDomainMessengerSentMessageEvent, error) {
	crossDomainMessengerABI, err := bindings.CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	sentMessageEventAbi := crossDomainMessengerABI.Events["SentMessage"]
	sentMessageEventExtensionAbi := crossDomainMessengerABI.Events["SentMessageExtension1"]

	processedSentMessageEvents := events.eventsBySignature[sentMessageEventAbi.ID]
	crossDomainMessageEvents := make([]CrossDomainMessengerSentMessageEvent, len(processedSentMessageEvents))
	for i, sentMessageEvent := range processedSentMessageEvents {
		log := sentMessageEvent.GethLog

		var sentMsgData bindings.CrossDomainMessengerSentMessage
		sentMsgData.Raw = *log
		err = UnpackLog(&sentMsgData, log, sentMessageEventAbi.Name, crossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		var sentMsgExtensionData bindings.CrossDomainMessengerSentMessageExtension1
		extensionLog := events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 1}].GethLog
		sentMsgExtensionData.Raw = *extensionLog
		err = UnpackLog(&sentMsgExtensionData, extensionLog, sentMessageEventExtensionAbi.Name, crossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		msgHash, err := CrossDomainMessageHash(crossDomainMessengerABI, &sentMsgData, sentMsgExtensionData.Value)
		if err != nil {
			return nil, err
		}

		crossDomainMessageEvents[i] = CrossDomainMessengerSentMessageEvent{
			CrossDomainMessengerSentMessage: &sentMsgData,
			Value:                           sentMsgExtensionData.Value,
			MessageHash:                     msgHash,
			RawEvent:                        sentMessageEvent,
		}
	}

	return crossDomainMessageEvents, nil
}

func CrossDomainMessengerRelayedMessageEvents(events *ProcessedContractEvents) ([]CrossDomainMessengerRelayedMessageEvent, error) {
	crossDomainMessengerABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	relayedMessageEventAbi := crossDomainMessengerABI.Events["RelayedMessage"]
	processedRelayedMessageEvents := events.eventsBySignature[relayedMessageEventAbi.ID]
	crossDomainMessageEvents := make([]CrossDomainMessengerRelayedMessageEvent, len(processedRelayedMessageEvents))
	for i, relayedMessageEvent := range processedRelayedMessageEvents {
		log := relayedMessageEvent.GethLog

		var relayedMsgData bindings.CrossDomainMessengerRelayedMessage
		relayedMsgData.Raw = *log
		err = UnpackLog(&relayedMsgData, log, relayedMessageEventAbi.Name, crossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		crossDomainMessageEvents[i] = CrossDomainMessengerRelayedMessageEvent{
			CrossDomainMessengerRelayedMessage: &relayedMsgData,
			RawEvent:                           relayedMessageEvent,
		}
	}

	return crossDomainMessageEvents, nil
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
