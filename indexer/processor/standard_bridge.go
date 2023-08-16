package processor

import (
	"bytes"
	"errors"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"

	"github.com/ethereum/go-ethereum/common"
)

type StandardBridgeInitiatedEvent struct {
	// We hardcode to ERC20 since ETH can be pseudo-represented as an ERC20 utilizing
	// the hardcoded ETH address
	*bindings.StandardBridgeERC20BridgeInitiated

	CrossDomainMessageHash common.Hash
	Event                  *database.ContractEvent
}

type StandardBridgeFinalizedEvent struct {
	// We hardcode to ERC20 since ETH can be pseudo-represented as an ERC20 utilizing
	// the hardcoded ETH address
	*bindings.StandardBridgeERC20BridgeFinalized

	CrossDomainMessageHash common.Hash
	Event                  *database.ContractEvent
}

// StandardBridgeInitiatedEvents extracts all initiated bridge events from the contracts that follow the StandardBridge ABI. The
// correlated CrossDomainMessenger nonce is also parsed from the associated messenger events.
func StandardBridgeInitiatedEvents(events *ProcessedContractEvents) ([]StandardBridgeInitiatedEvent, error) {
	ethBridgeInitiatedEvents, err := _standardBridgeInitiatedEvents[bindings.StandardBridgeETHBridgeInitiated](events)
	if err != nil {
		return nil, err
	}

	erc20BridgeInitiatedEvents, err := _standardBridgeInitiatedEvents[bindings.StandardBridgeERC20BridgeInitiated](events)
	if err != nil {
		return nil, err
	}

	return append(ethBridgeInitiatedEvents, erc20BridgeInitiatedEvents...), nil
}

// StandardBridgeFinalizedEvents extracts all finalization bridge events from the contracts that follow the StandardBridge ABI. The
// correlated CrossDomainMessenger nonce is also parsed by looking at the parameters of the corresponding relayMessage transaction data.
func StandardBridgeFinalizedEvents(events *ProcessedContractEvents) ([]StandardBridgeFinalizedEvent, error) {
	ethBridgeFinalizedEvents, err := _standardBridgeFinalizedEvents[bindings.StandardBridgeETHBridgeFinalized](events)
	if err != nil {
		return nil, err
	}

	erc20BridgeFinalizedEvents, err := _standardBridgeFinalizedEvents[bindings.StandardBridgeERC20BridgeFinalized](events)
	if err != nil {
		return nil, err
	}

	return append(ethBridgeFinalizedEvents, erc20BridgeFinalizedEvents...), nil
}

// parse out eth or erc20 bridge initiated events
func _standardBridgeInitiatedEvents[BridgeEvent bindings.StandardBridgeETHBridgeInitiated | bindings.StandardBridgeERC20BridgeInitiated](
	events *ProcessedContractEvents,
) ([]StandardBridgeInitiatedEvent, error) {
	standardBridgeABI, err := bindings.StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	crossDomainMessengerABI, err := bindings.CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	sentMessageEventAbi := crossDomainMessengerABI.Events["SentMessage"]
	sentMessageExtensionEventAbi := crossDomainMessengerABI.Events["SentMessageExtension1"]

	var tmp BridgeEvent
	var eventName string
	var finalizeMethodName string
	switch any(tmp).(type) {
	case bindings.StandardBridgeETHBridgeInitiated:
		eventName = "ETHBridgeInitiated"
		finalizeMethodName = "finalizeBridgeETH"
	case bindings.StandardBridgeERC20BridgeInitiated:
		eventName = "ERC20BridgeInitiated"
		finalizeMethodName = "finalizeBridgeERC20"
	default:
		panic("should not be here")
	}

	processedInitiatedBridgeEvents := events.eventsBySignature[standardBridgeABI.Events[eventName].ID]
	initiatedBridgeEvents := make([]StandardBridgeInitiatedEvent, len(processedInitiatedBridgeEvents))
	for i, bridgeInitiatedEvent := range processedInitiatedBridgeEvents {
		log := bridgeInitiatedEvent.RLPLog

		var bridgeData BridgeEvent
		err := UnpackLog(&bridgeData, log, eventName, standardBridgeABI)
		if err != nil {
			return nil, err
		}

		// Look for the sent message event to compute the message hash of the relayed tx
		//   - L1: BridgeInitiated -> Portal#DepositTransaction -> SentMessage ...
		//   - L1: BridgeInitiated -> L2ToL1MessagePasser#MessagePassed -> SentMessage ...
		var sentMsgData bindings.CrossDomainMessengerSentMessage
		sentMsgLog := events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 2}].RLPLog
		if sentMsgLog.Topics[0] != sentMessageEventAbi.ID {
			return nil, errors.New("unexpected bridge event ordering")
		}
		sentMsgData.Raw = *sentMsgLog
		err = UnpackLog(&sentMsgData, sentMsgLog, sentMessageEventAbi.Name, crossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		var sentMsgExtensionData bindings.CrossDomainMessengerSentMessageExtension1
		sentMsgExtensionLog := events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 3}].RLPLog
		if sentMsgExtensionLog.Topics[0] != sentMessageExtensionEventAbi.ID {
			return nil, errors.New("unexpected bridge event ordering")
		}
		sentMsgData.Raw = *sentMsgLog
		err = UnpackLog(&sentMsgExtensionData, sentMsgExtensionLog, sentMessageExtensionEventAbi.Name, crossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		msgHash, err := CrossDomainMessageHash(crossDomainMessengerABI, &sentMsgData, sentMsgExtensionData.Value)
		if err != nil {
			return nil, err
		}

		var erc20BridgeData *bindings.StandardBridgeERC20BridgeInitiated
		var expectedCrossDomainMessage []byte
		switch any(bridgeData).(type) {
		case bindings.StandardBridgeETHBridgeInitiated:
			ethBridgeData := any(bridgeData).(bindings.StandardBridgeETHBridgeInitiated)
			expectedCrossDomainMessage, err = standardBridgeABI.Pack(finalizeMethodName, ethBridgeData.From, ethBridgeData.To, ethBridgeData.Amount, ethBridgeData.ExtraData)
			if err != nil {
				return nil, err
			}

			// represent eth bridge as an erc20
			erc20BridgeData = &bindings.StandardBridgeERC20BridgeInitiated{
				Raw: *log,
				// Represent ETH using the hardcoded address
				LocalToken: predeploys.LegacyERC20ETHAddr, RemoteToken: predeploys.LegacyERC20ETHAddr,
				// Bridge data
				From: ethBridgeData.From, To: ethBridgeData.To, Amount: ethBridgeData.Amount, ExtraData: ethBridgeData.ExtraData,
			}

		case bindings.StandardBridgeERC20BridgeInitiated:
			_temp := any(bridgeData).(bindings.StandardBridgeERC20BridgeInitiated)
			erc20BridgeData = &_temp
			erc20BridgeData.Raw = *log
			expectedCrossDomainMessage, err = standardBridgeABI.Pack(finalizeMethodName, erc20BridgeData.RemoteToken, erc20BridgeData.LocalToken, erc20BridgeData.From, erc20BridgeData.To, erc20BridgeData.Amount, erc20BridgeData.ExtraData)
			if err != nil {
				return nil, err
			}
		}

		if !bytes.Equal(sentMsgData.Message, expectedCrossDomainMessage) {
			return nil, errors.New("bridge cross domain message mismatch")
		}

		initiatedBridgeEvents[i] = StandardBridgeInitiatedEvent{
			StandardBridgeERC20BridgeInitiated: erc20BridgeData,
			CrossDomainMessageHash:             msgHash,
			Event:                              bridgeInitiatedEvent,
		}
	}

	return initiatedBridgeEvents, nil
}

// parse out eth or erc20 bridge finalization events
func _standardBridgeFinalizedEvents[BridgeEvent bindings.StandardBridgeETHBridgeFinalized | bindings.StandardBridgeERC20BridgeFinalized](
	events *ProcessedContractEvents,
) ([]StandardBridgeFinalizedEvent, error) {
	standardBridgeABI, err := bindings.StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	crossDomainMessengerABI, err := bindings.CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	relayedMessageEventAbi := crossDomainMessengerABI.Events["RelayedMessage"]

	var bridgeData BridgeEvent
	var eventName string
	switch any(bridgeData).(type) {
	case bindings.StandardBridgeETHBridgeFinalized:
		eventName = "ETHBridgeFinalized"
	case bindings.StandardBridgeERC20BridgeFinalized:
		eventName = "ERC20BridgeFinalized"
	default:
		panic("should not be here")
	}

	processedFinalizedBridgeEvents := events.eventsBySignature[standardBridgeABI.Events[eventName].ID]
	finalizedBridgeEvents := make([]StandardBridgeFinalizedEvent, len(processedFinalizedBridgeEvents))
	for i, bridgeFinalizedEvent := range processedFinalizedBridgeEvents {
		log := bridgeFinalizedEvent.RLPLog

		var bridgeData BridgeEvent
		err := UnpackLog(&bridgeData, log, eventName, standardBridgeABI)
		if err != nil {
			return nil, err
		}

		// Look for the RelayedMessage event that follows right after the BridgeFinalized Event
		var relayedMsgData bindings.CrossDomainMessengerRelayedMessage
		relayedMsgLog := events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 1}].RLPLog
		if relayedMsgLog.Topics[0] != relayedMessageEventAbi.ID {
			return nil, errors.New("unexpected bridge event ordering")
		}
		err = UnpackLog(&relayedMsgData, relayedMsgLog, relayedMessageEventAbi.Name, crossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		var erc20BridgeData *bindings.StandardBridgeERC20BridgeFinalized
		switch any(bridgeData).(type) {
		case bindings.StandardBridgeETHBridgeFinalized:
			ethBridgeData := any(bridgeData).(bindings.StandardBridgeETHBridgeFinalized)
			erc20BridgeData = &bindings.StandardBridgeERC20BridgeFinalized{
				Raw: *log,
				// Represent ETH using the hardcoded address
				LocalToken: predeploys.LegacyERC20ETHAddr, RemoteToken: predeploys.LegacyERC20ETHAddr,
				// Bridge data
				From: ethBridgeData.From, To: ethBridgeData.To, Amount: ethBridgeData.Amount, ExtraData: ethBridgeData.ExtraData,
			}

		case bindings.StandardBridgeERC20BridgeFinalized:
			_temp := any(bridgeData).(bindings.StandardBridgeERC20BridgeFinalized)
			erc20BridgeData = &_temp
			erc20BridgeData.Raw = *log
		}

		finalizedBridgeEvents[i] = StandardBridgeFinalizedEvent{
			StandardBridgeERC20BridgeFinalized: erc20BridgeData,
			CrossDomainMessageHash:             relayedMsgData.MsgHash,
			Event:                              bridgeFinalizedEvent,
		}
	}

	return finalizedBridgeEvents, nil
}
