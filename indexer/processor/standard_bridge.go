package processor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"

	"github.com/ethereum/go-ethereum/ethclient"
)

type StandardBridgeInitiatedEvent struct {
	// We hardcode to ERC20 since ETH can be pseudo-represented as an ERC20 utilizing
	// the hardcoded ETH address
	*bindings.StandardBridgeERC20BridgeInitiated

	CrossDomainMessengerNonce *big.Int
	RawEvent                  *database.ContractEvent
}

type StandardBridgeFinalizedEvent struct {
	// We hardcode to ERC20 since ETH can be pseudo-represented as an ERC20 utilizing
	// the hardcoded ETH address
	*bindings.StandardBridgeERC20BridgeFinalized

	CrossDomainMessengerNonce *big.Int
	RawEvent                  *database.ContractEvent
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
func StandardBridgeFinalizedEvents(rawEthClient *ethclient.Client, events *ProcessedContractEvents) ([]StandardBridgeFinalizedEvent, error) {
	ethBridgeFinalizedEvents, err := _standardBridgeFinalizedEvents[bindings.StandardBridgeETHBridgeFinalized](rawEthClient, events)
	if err != nil {
		return nil, err
	}

	erc20BridgeFinalizedEvents, err := _standardBridgeFinalizedEvents[bindings.StandardBridgeERC20BridgeFinalized](rawEthClient, events)
	if err != nil {
		return nil, err
	}

	return append(ethBridgeFinalizedEvents, erc20BridgeFinalizedEvents...), nil
}

// parse out eth or erc20 bridge initiated events
func _standardBridgeInitiatedEvents[BridgeEvent bindings.StandardBridgeETHBridgeInitiated | bindings.StandardBridgeERC20BridgeInitiated](
	events *ProcessedContractEvents,
) ([]StandardBridgeInitiatedEvent, error) {
	StandardBridgeABI, err := bindings.StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	l1CrossDomainMessengerABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	sentMessageEventAbi := l1CrossDomainMessengerABI.Events["SentMessage"]

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

	processedInitiatedBridgeEvents := events.eventsBySignature[StandardBridgeABI.Events[eventName].ID]
	initiatedBridgeEvents := make([]StandardBridgeInitiatedEvent, len(processedInitiatedBridgeEvents))
	for i, bridgeInitiatedEvent := range processedInitiatedBridgeEvents {
		log := bridgeInitiatedEvent.GethLog

		var bridgeData BridgeEvent
		err := UnpackLog(&bridgeData, log, eventName, StandardBridgeABI)
		if err != nil {
			return nil, err
		}

		// Look for the sent message event to extract the associated messager nonce
		//   - L1: BridgeInitiated -> Portal#DepositTransaction -> SentMessage ...
		//   - L1: BridgeInitiated -> L2ToL1MessagePasser#MessagePassed -> SentMessage ...
		var sentMsgData bindings.L1CrossDomainMessengerSentMessage
		sentMsgLog := events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 2}].GethLog
		sentMsgData.Raw = *sentMsgLog
		err = UnpackLog(&sentMsgData, sentMsgLog, sentMessageEventAbi.Name, l1CrossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		var erc20BridgeData *bindings.StandardBridgeERC20BridgeInitiated
		var expectedCrossDomainMessage []byte
		switch any(bridgeData).(type) {
		case bindings.StandardBridgeETHBridgeInitiated:
			ethBridgeData := any(bridgeData).(bindings.StandardBridgeETHBridgeInitiated)
			expectedCrossDomainMessage, err = StandardBridgeABI.Pack(finalizeMethodName, ethBridgeData.From, ethBridgeData.To, ethBridgeData.Amount, ethBridgeData.ExtraData)
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
			expectedCrossDomainMessage, err = StandardBridgeABI.Pack(finalizeMethodName, erc20BridgeData.RemoteToken, erc20BridgeData.LocalToken, erc20BridgeData.From, erc20BridgeData.To, erc20BridgeData.Amount, erc20BridgeData.ExtraData)
			if err != nil {
				return nil, err
			}
		}

		if !bytes.Equal(sentMsgData.Message, expectedCrossDomainMessage) {
			return nil, errors.New("bridge cross domain message mismatch")
		}

		initiatedBridgeEvents[i] = StandardBridgeInitiatedEvent{
			StandardBridgeERC20BridgeInitiated: erc20BridgeData,
			CrossDomainMessengerNonce:          sentMsgData.MessageNonce,
			RawEvent:                           bridgeInitiatedEvent,
		}
	}

	return initiatedBridgeEvents, nil
}

// parse out eth or erc20 bridge finalization events
func _standardBridgeFinalizedEvents[BridgeEvent bindings.StandardBridgeETHBridgeFinalized | bindings.StandardBridgeERC20BridgeFinalized](
	rawEthClient *ethclient.Client,
	events *ProcessedContractEvents,
) ([]StandardBridgeFinalizedEvent, error) {
	StandardBridgeABI, err := bindings.StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	l1CrossDomainMessengerABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	optimismPortalAbi, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	relayedMessageEventAbi := l1CrossDomainMessengerABI.Events["RelayedMessage"]
	relayMessageMethodAbi := l1CrossDomainMessengerABI.Methods["relayMessage"]
	finalizeWithdrawalTransactionMethodAbi := optimismPortalAbi.Methods["finalizeWithdrawalTransaction"]

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

	processedFinalizedBridgeEvents := events.eventsBySignature[StandardBridgeABI.Events[eventName].ID]
	finalizedBridgeEvents := make([]StandardBridgeFinalizedEvent, len(processedFinalizedBridgeEvents))
	for i, bridgeFinalizedEvent := range processedFinalizedBridgeEvents {
		log := bridgeFinalizedEvent.GethLog

		var bridgeData BridgeEvent
		err := UnpackLog(&bridgeData, log, eventName, StandardBridgeABI)
		if err != nil {
			return nil, err
		}

		// Look for the RelayedMessage event that follows right after the BridgeFinalized Event
		relayedMsgLog := events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 1}].GethLog
		if relayedMsgLog.Topics[0] != relayedMessageEventAbi.ID {
			return nil, errors.New("unexpected bridge event ordering")
		}

		// There's no way to extract the nonce on the relayed message event. we can extract the nonce by
		// by unpacking the transaction input for the `relayMessage` transaction. Since bedrock has OptimismPortal
		// as on L1 as an intermediary for finalization, we have to check both scenarios
		tx, isPending, err := rawEthClient.TransactionByHash(context.Background(), relayedMsgLog.TxHash)
		if err != nil || isPending {
			return nil, errors.New("unable to query relayMessage tx for bridge finalization event")
		}

		// If this is a finalization step with the optimism portal, the calldata for relayMessage invocation can be
		// extracted from the withdrawal transaction.

		// NOTE: the L2CrossDomainMessenger nonce may not match the L2ToL1MessagePasser nonce, hence the additional
		// layer of decoding vs reading the nocne of the withdrawal transaction. Both nonces have a similar but
		// different lifeycle that might not match (i.e L2ToL1MessagePasser can be invoced directly)
		var relayMsgCallData []byte
		switch {
		case bytes.Equal(tx.Data()[:4], relayMessageMethodAbi.ID):
			relayMsgCallData = tx.Data()[4:]
		case bytes.Equal(tx.Data()[:4], finalizeWithdrawalTransactionMethodAbi.ID):
			data, err := finalizeWithdrawalTransactionMethodAbi.Inputs.Unpack(tx.Data()[4:])
			if err != nil {
				return nil, err
			}

			finalizeWithdrawTransactionInput := new(struct {
				Tx bindings.TypesWithdrawalTransaction
			})
			err = finalizeWithdrawalTransactionMethodAbi.Inputs.Copy(finalizeWithdrawTransactionInput, data)
			if err != nil {
				return nil, fmt.Errorf("unable extract withdrawal tx input from finalizeWithdrawalTransaction calldata: %w", err)
			} else if !bytes.Equal(finalizeWithdrawTransactionInput.Tx.Data[:4], relayMessageMethodAbi.ID) {
				return nil, errors.New("finalizeWithdrawalTransaction calldata does not match relayMessage invocation")
			}
			relayMsgCallData = finalizeWithdrawTransactionInput.Tx.Data[4:]
		default:
			return nil, errors.New("bridge finalization event does not correlate with a relayMessage tx invocation")
		}

		inputsMap := make(map[string]interface{})
		err = relayMessageMethodAbi.Inputs.UnpackIntoMap(inputsMap, relayMsgCallData)
		if err != nil {
			return nil, err
		}
		nonce, ok := inputsMap["_nonce"].(*big.Int)
		if !ok {
			return nil, errors.New("unable to extract `_nonce` parameter from relayMessage calldata")
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
			CrossDomainMessengerNonce:          nonce,
			RawEvent:                           bridgeFinalizedEvent,
		}
	}

	return finalizedBridgeEvents, nil
}
