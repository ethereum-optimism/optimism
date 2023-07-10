package processor

import (
	"bytes"
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	ethAddress = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")
)

type StandardBridgeInitiatedEvent struct {
	*bindings.L1StandardBridgeERC20BridgeInitiated

	CrossDomainMessengerNonce *big.Int
	RawEvent                  *database.ContractEvent
}

type StandardBridgeFinalizedEvent struct {
	*bindings.L1StandardBridgeERC20BridgeFinalized

	CrossDomainMessengerNonce *big.Int
	RawEvent                  *database.ContractEvent
}

// StandardBridgeInitiatedEvents extracts all initiated bridge events from the contracts that follow the StandardBridge ABI. The
// correlated CrossDomainMessenger nonce is also parsed from the associated messenger events.
func StandardBridgeInitiatedEvents(events *ProcessedContractEvents) ([]*StandardBridgeInitiatedEvent, error) {
	l1StandardBridgeABI, err := bindings.L1StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	l1CrossDomainMessengerABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	ethBridgeInitiatedEventAbi := l1StandardBridgeABI.Events["ETHBridgeInitiated"]
	erc20BridgeInitiatedEventAbi := l1StandardBridgeABI.Events["ERC20BridgeInitiated"]
	sentMessageEventAbi := l1CrossDomainMessengerABI.Events["SentMessage"]

	ethBridgeInitiatedEvents := events.eventsBySignature[ethBridgeInitiatedEventAbi.ID]
	erc20BridgeInitiatedEvents := events.eventsBySignature[erc20BridgeInitiatedEventAbi.ID]
	initiatedBridgeEvents := []*StandardBridgeInitiatedEvent{}

	// Handle ETH Bridge
	for _, bridgeInitiatedEvent := range ethBridgeInitiatedEvents {
		log := events.eventLog[bridgeInitiatedEvent.GUID]
		bridgeData, err := UnpackLog[bindings.L1StandardBridgeETHBridgeInitiated](log, ethBridgeInitiatedEventAbi.Name, l1StandardBridgeABI)
		if err != nil {
			return nil, err
		}

		// Look for the sent message event to extract the associated messager nonce
		//       - The `SentMessage` event is the second after the bridge initiated event. BridgeInitiated -> Portal#DepositTransaction -> SentMesage ...
		sentMsgLog := events.eventLog[events.eventByLogIndex[log.Index+2].GUID]
		sentMsgData, err := UnpackLog[bindings.L1CrossDomainMessengerSentMessage](sentMsgLog, sentMessageEventAbi.Name, l1CrossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		expectedMsg, err := l1StandardBridgeABI.Pack("finalizeBridgeETH", bridgeData.From, bridgeData.To, bridgeData.Amount, bridgeData.ExtraData)
		if err != nil {
			return nil, err
		} else if !bytes.Equal(sentMsgData.Message, expectedMsg) {
			return nil, errors.New("bridge cross domain message mismatch")
		}

		initiatedBridgeEvents = append(initiatedBridgeEvents, &StandardBridgeInitiatedEvent{
			&bindings.L1StandardBridgeERC20BridgeInitiated{
				// Default ETH
				LocalToken: ethAddress, RemoteToken: ethAddress,

				// BridgeDAta
				From: bridgeData.From, To: bridgeData.To, Amount: bridgeData.Amount, ExtraData: bridgeData.ExtraData,
			},
			sentMsgData.MessageNonce,
			bridgeInitiatedEvent,
		})
	}

	// Handle ERC20 Bridge
	for _, bridgeInitiatedEvent := range erc20BridgeInitiatedEvents {
		log := events.eventLog[bridgeInitiatedEvent.GUID]
		bridgeData, err := UnpackLog[bindings.L1StandardBridgeERC20BridgeInitiated](log, erc20BridgeInitiatedEventAbi.Name, l1StandardBridgeABI)
		if err != nil {
			return nil, err
		}

		// Look for the sent message event to extract the associated messager nonce
		//       - The `SentMessage` event is the second after the bridge initiated event. BridgeInitiated -> Portal#DepositTransaction -> SentMesage ...
		sentMsgLog := events.eventLog[events.eventByLogIndex[log.Index+2].GUID]
		sentMsgData, err := UnpackLog[bindings.L1CrossDomainMessengerSentMessage](sentMsgLog, sentMessageEventAbi.Name, l1CrossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		expectedMsg, err := l1StandardBridgeABI.Pack("finalizeBridgeERC20", bridgeData.RemoteToken, bridgeData.LocalToken, bridgeData.From, bridgeData.To, bridgeData.Amount, bridgeData.ExtraData)
		if err != nil {
			return nil, err
		} else if !bytes.Equal(sentMsgData.Message, expectedMsg) {
			return nil, errors.New("bridge cross domain message mismatch")
		}

		initiatedBridgeEvents = append(initiatedBridgeEvents, &StandardBridgeInitiatedEvent{bridgeData, sentMsgData.MessageNonce, bridgeInitiatedEvent})
	}

	return initiatedBridgeEvents, nil
}

// StandardBridgeFinalizedEvents extracts all finalization bridge events from the contracts that follow the StandardBridge ABI. The
// correlated CrossDomainMessenger nonce is also parsed by looking at the parameters of the corresponding relayMessage transaction data.
func StandardBridgeFinalizedEvents(rawEthClient *ethclient.Client, events *ProcessedContractEvents) ([]*StandardBridgeFinalizedEvent, error) {
	l1StandardBridgeABI, err := bindings.L1StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	l1CrossDomainMessengerABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	ethBridgeFinalizedEventAbi := l1StandardBridgeABI.Events["ETHBridgeFinalized"]
	erc20BridgeFinalizedEventAbi := l1StandardBridgeABI.Events["ERC20BridgeFinalized"]
	relayedMessageEventAbi := l1CrossDomainMessengerABI.Events["RelayedMessage"]
	relayMessageMethodAbi := l1CrossDomainMessengerABI.Methods["relayMessage"]

	ethBridgeFinalizedEvents := events.eventsBySignature[ethBridgeFinalizedEventAbi.ID]
	erc20BridgeFinalizedEvents := events.eventsBySignature[erc20BridgeFinalizedEventAbi.ID]
	finalizedBridgeEvents := []*StandardBridgeFinalizedEvent{}

	// Handle ETH Bridge
	for _, bridgeFinalizedEvent := range ethBridgeFinalizedEvents {
		log := events.eventLog[bridgeFinalizedEvent.GUID]
		bridgeData, err := UnpackLog[bindings.L1StandardBridgeETHBridgeFinalized](log, ethBridgeFinalizedEventAbi.Name, l1StandardBridgeABI)
		if err != nil {
			return nil, err
		}

		// Look for the RelayedMessage event that follows right after the BridgeFinalized Event
		relayedMsgLog := events.eventLog[events.eventByLogIndex[log.Index+1].GUID]
		if relayedMsgLog.Topics[0] != relayedMessageEventAbi.ID {
			return nil, errors.New("unexpected bridge event ordering")
		}

		// There's no way to extract the nonce on the relayed message event. we can extract
		// the nonce by unpacking the transaction input for the `relayMessage` transaction
		tx, isPending, err := rawEthClient.TransactionByHash(context.Background(), relayedMsgLog.TxHash)
		if err != nil || isPending {
			return nil, errors.New("unable to query relayMessage tx for bridge finalization event")
		}

		txData := tx.Data()
		if !bytes.Equal(txData[:4], relayMessageMethodAbi.ID) {
			return nil, errors.New("bridge finalization event does not match relayMessage tx invocation")
		}

		inputsMap := make(map[string]interface{})
		err = relayMessageMethodAbi.Inputs.UnpackIntoMap(inputsMap, txData[4:])
		if err != nil {
			return nil, err
		}

		nonce, ok := inputsMap["_nonce"].(*big.Int)
		if !ok {
			return nil, errors.New("unable to extract `_nonce` parameter from relayMessage transaction")
		}

		finalizedBridgeEvents = append(finalizedBridgeEvents, &StandardBridgeFinalizedEvent{
			&bindings.L1StandardBridgeERC20BridgeFinalized{
				// Default ETH
				LocalToken: ethAddress, RemoteToken: ethAddress,

				// BridgeDAta
				From: bridgeData.From, To: bridgeData.To, Amount: bridgeData.Amount, ExtraData: bridgeData.ExtraData,
			},
			nonce,
			bridgeFinalizedEvent,
		})
	}

	// Handle ERC20 Bridge
	for _, bridgeFinalizedEvent := range erc20BridgeFinalizedEvents {
		log := events.eventLog[bridgeFinalizedEvent.GUID]
		bridgeData, err := UnpackLog[bindings.L1StandardBridgeERC20BridgeFinalized](log, erc20BridgeFinalizedEventAbi.Name, l1StandardBridgeABI)
		if err != nil {
			return nil, err
		}

		// Look for the RelayedMessage event that follows right after the BridgeFinalized Event
		relayedMsgLog := events.eventLog[events.eventByLogIndex[log.Index+1].GUID]
		if relayedMsgLog.Topics[0] != relayedMessageEventAbi.ID {
			return nil, errors.New("unexpected bridge event ordering")
		}

		// There's no way to extract the nonce on the relayed message event. we can extract
		// the nonce by unpacking the transaction input for the `relayMessage` transaction
		tx, isPending, err := rawEthClient.TransactionByHash(context.Background(), relayedMsgLog.TxHash)
		if err != nil || isPending {
			return nil, errors.New("unable to query relayMessage tx for bridge finalization event")
		}

		txData := tx.Data()
		if !bytes.Equal(txData[:4], relayMessageMethodAbi.ID) {
			return nil, errors.New("bridge finalization event does not match relayMessage tx invocation")
		}

		inputsMap := make(map[string]interface{})
		err = relayMessageMethodAbi.Inputs.UnpackIntoMap(inputsMap, txData[4:])
		if err != nil {
			return nil, err
		}

		nonce, ok := inputsMap["_nonce"].(*big.Int)
		if !ok {
			return nil, errors.New("unable to extract `_nonce` parameter from relayMessage transaction")
		}

		finalizedBridgeEvents = append(finalizedBridgeEvents, &StandardBridgeFinalizedEvent{bridgeData, nonce, bridgeFinalizedEvent})
	}

	return finalizedBridgeEvents, nil
}
