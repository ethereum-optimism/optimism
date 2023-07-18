package processor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	EthAddress = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")
)

type StandardBridgeInitiatedEvent struct {
	// We hardcode to ERC20 since ETH can be pseudo-represented as an ERC20 utilizing
	// the hardcoded ETH address
	*bindings.L1StandardBridgeERC20BridgeInitiated

	CrossDomainMessengerNonce *big.Int
	RawEvent                  *database.ContractEvent
}

type StandardBridgeFinalizedEvent struct {
	// We hardcode to ERC20 since ETH can be pseudo-represented as an ERC20 utilizing
	// the hardcoded ETH address
	*bindings.L1StandardBridgeERC20BridgeFinalized

	CrossDomainMessengerNonce *big.Int
	RawEvent                  *database.ContractEvent
}

// StandardBridgeInitiatedEvents extracts all initiated bridge events from the contracts that follow the StandardBridge ABI. The
// correlated CrossDomainMessenger nonce is also parsed from the associated messenger events.
func StandardBridgeInitiatedEvents(events *ProcessedContractEvents) ([]StandardBridgeInitiatedEvent, error) {
	ethBridgeInitiatedEvents, err := _standardBridgeInitiatedEvents[bindings.L1StandardBridgeETHBridgeInitiated](events)
	if err != nil {
		return nil, err
	}

	erc20BridgeInitiatedEvents, err := _standardBridgeInitiatedEvents[bindings.L1StandardBridgeERC20BridgeInitiated](events)
	if err != nil {
		return nil, err
	}

	return append(ethBridgeInitiatedEvents, erc20BridgeInitiatedEvents...), nil
}

// StandardBridgeFinalizedEvents extracts all finalization bridge events from the contracts that follow the StandardBridge ABI. The
// correlated CrossDomainMessenger nonce is also parsed by looking at the parameters of the corresponding relayMessage transaction data.
func StandardBridgeFinalizedEvents(rawEthClient *ethclient.Client, events *ProcessedContractEvents) ([]StandardBridgeFinalizedEvent, error) {
	ethBridgeFinalizedEvents, err := _standardBridgeFinalizedEvents[bindings.L1StandardBridgeETHBridgeFinalized](rawEthClient, events)
	if err != nil {
		return nil, err
	}

	erc20BridgeFinalizedEvents, err := _standardBridgeFinalizedEvents[bindings.L1StandardBridgeERC20BridgeFinalized](rawEthClient, events)
	if err != nil {
		return nil, err
	}

	return append(ethBridgeFinalizedEvents, erc20BridgeFinalizedEvents...), nil
}

// parse out eth or erc20 bridge initiated events
func _standardBridgeInitiatedEvents[BridgeEvent bindings.L1StandardBridgeETHBridgeInitiated | bindings.L1StandardBridgeERC20BridgeInitiated](
	events *ProcessedContractEvents,
) ([]StandardBridgeInitiatedEvent, error) {
	l1StandardBridgeABI, err := bindings.L1StandardBridgeMetaData.GetAbi()
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
	case bindings.L1StandardBridgeETHBridgeInitiated:
		eventName = "ETHBridgeInitiated"
		finalizeMethodName = "finalizeBridgeETH"
	case bindings.L1StandardBridgeERC20BridgeInitiated:
		eventName = "ERC20BridgeInitiated"
		finalizeMethodName = "finalizeBridgeERC20"
	default:
		panic("should not be here")
	}

	processedInitiatedBridgeEvents := events.eventsBySignature[l1StandardBridgeABI.Events[eventName].ID]
	initiatedBridgeEvents := make([]StandardBridgeInitiatedEvent, len(processedInitiatedBridgeEvents))
	for i, bridgeInitiatedEvent := range processedInitiatedBridgeEvents {
		log := events.eventLog[bridgeInitiatedEvent.GUID]

		bridgeData := new(BridgeEvent)
		err := UnpackLog(bridgeData, log, eventName, l1StandardBridgeABI)
		if err != nil {
			return nil, err
		}

		// Look for the sent message event to extract the associated messager nonce
		//   - L1: BridgeInitiated -> Portal#DepositTransaction -> SentMessage ...
		//   - L1: BridgeInitiated -> L2ToL1MessagePasser#MessagePassed -> SentMessage ...
		var sentMsgData bindings.L1CrossDomainMessengerSentMessage
		sentMsgLog := events.eventLog[events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 2}].GUID]
		err = UnpackLog(&sentMsgData, sentMsgLog, sentMessageEventAbi.Name, l1CrossDomainMessengerABI)
		if err != nil {
			return nil, err
		}

		var erc20BridgeData *bindings.L1StandardBridgeERC20BridgeInitiated
		var expectedCrossDomainMessage []byte
		switch any(bridgeData).(type) {
		case *bindings.L1StandardBridgeETHBridgeInitiated:
			ethBridgeData := any(bridgeData).(*bindings.L1StandardBridgeETHBridgeInitiated)
			expectedCrossDomainMessage, err = l1StandardBridgeABI.Pack(finalizeMethodName, ethBridgeData.From, ethBridgeData.To, ethBridgeData.Amount, ethBridgeData.ExtraData)
			if err != nil {
				return nil, err
			}

			// represent eth bridge as an erc20
			erc20BridgeData = &bindings.L1StandardBridgeERC20BridgeInitiated{
				// Represent ETH using the hardcoded address
				LocalToken: EthAddress, RemoteToken: EthAddress,
				// Bridge data
				From: ethBridgeData.From, To: ethBridgeData.To, Amount: ethBridgeData.Amount, ExtraData: ethBridgeData.ExtraData,
			}

		case *bindings.L1StandardBridgeERC20BridgeInitiated:
			_temp := any(bridgeData).(bindings.L1StandardBridgeERC20BridgeInitiated)
			erc20BridgeData = &_temp
			expectedCrossDomainMessage, err = l1StandardBridgeABI.Pack(finalizeMethodName, erc20BridgeData.RemoteToken, erc20BridgeData.LocalToken, erc20BridgeData.From, erc20BridgeData.To, erc20BridgeData.Amount, erc20BridgeData.ExtraData)
			if err != nil {
				return nil, err
			}
		}

		if !bytes.Equal(sentMsgData.Message, expectedCrossDomainMessage) {
			return nil, errors.New("bridge cross domain message mismatch")
		}

		initiatedBridgeEvents[i] = StandardBridgeInitiatedEvent{erc20BridgeData, sentMsgData.MessageNonce, bridgeInitiatedEvent}
	}

	return initiatedBridgeEvents, nil
}

// parse out eth or erc20 bridge finalization events
func _standardBridgeFinalizedEvents[BridgeEvent bindings.L1StandardBridgeETHBridgeFinalized | bindings.L1StandardBridgeERC20BridgeFinalized](
	rawEthClient *ethclient.Client,
	events *ProcessedContractEvents,
) ([]StandardBridgeFinalizedEvent, error) {
	l1StandardBridgeABI, err := bindings.L1StandardBridgeMetaData.GetAbi()
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
	case bindings.L1StandardBridgeETHBridgeFinalized:
		eventName = "ETHBridgeFinalized"
	case bindings.L1StandardBridgeERC20BridgeFinalized:
		eventName = "ERC20BridgeFinalized"
	default:
		panic("should not be here")
	}

	processedFinalizedBridgeEvents := events.eventsBySignature[l1StandardBridgeABI.Events[eventName].ID]
	finalizedBridgeEvents := make([]StandardBridgeFinalizedEvent, len(processedFinalizedBridgeEvents))
	for i, bridgeFinalizedEvent := range processedFinalizedBridgeEvents {
		log := events.eventLog[bridgeFinalizedEvent.GUID]

		var bridgeData BridgeEvent
		err := UnpackLog(&bridgeData, log, eventName, l1StandardBridgeABI)
		if err != nil {
			return nil, err
		}

		// Look for the RelayedMessage event that follows right after the BridgeFinalized Event
		relayedMsgLog := events.eventLog[events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 1}].GUID]
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

		var erc20BridgeData *bindings.L1StandardBridgeERC20BridgeFinalized
		switch any(bridgeData).(type) {
		case bindings.L1StandardBridgeETHBridgeInitiated:
			ethBridgeData := any(bridgeData).(bindings.L1StandardBridgeETHBridgeFinalized)
			erc20BridgeData = &bindings.L1StandardBridgeERC20BridgeFinalized{
				// Represent ETH using the hardcoded address
				LocalToken: EthAddress, RemoteToken: EthAddress,
				// Bridge data
				From: ethBridgeData.From, To: ethBridgeData.To, Amount: ethBridgeData.Amount, ExtraData: ethBridgeData.ExtraData,
			}

		case bindings.L1StandardBridgeERC20BridgeInitiated:
			_temp := any(bridgeData).(bindings.L1StandardBridgeERC20BridgeFinalized)
			erc20BridgeData = &_temp
		}

		finalizedBridgeEvents[i] = StandardBridgeFinalizedEvent{erc20BridgeData, nonce, bridgeFinalizedEvent}
	}

	return finalizedBridgeEvents, nil
}
