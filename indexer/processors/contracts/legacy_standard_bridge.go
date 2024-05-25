package contracts

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum/common"
)

type LegacyBridgeEvent struct {
	Event          *database.ContractEvent
	BridgeTransfer database.BridgeTransfer
}

func L1StandardBridgeLegacyDepositInitiatedEvents(contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]LegacyBridgeEvent, error) {
	// The L1StandardBridge ABI contains the legacy events
	l1StandardBridgeAbi, err := bindings.L1StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	ethDepositEventAbi := l1StandardBridgeAbi.Events["ETHDepositInitiated"]
	erc20DepositEventAbi := l1StandardBridgeAbi.Events["ERC20DepositInitiated"]

	// Grab both ETH & ERC20 Events
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: ethDepositEventAbi.ID}
	ethDepositEvents, err := db.ContractEvents.L1ContractEventsWithFilter(contractEventFilter, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}
	contractEventFilter.EventSignature = erc20DepositEventAbi.ID
	erc20DepositEvents, err := db.ContractEvents.L1ContractEventsWithFilter(contractEventFilter, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	// Represent the ETH deposits via the ETH ERC20 predeploy address
	deposits := make([]LegacyBridgeEvent, len(ethDepositEvents)+len(erc20DepositEvents))
	for i := range ethDepositEvents {
		bridgeEvent := bindings.L1StandardBridgeETHDepositInitiated{Raw: *ethDepositEvents[i].RLPLog}
		err := UnpackLog(&bridgeEvent, &bridgeEvent.Raw, ethDepositEventAbi.Name, l1StandardBridgeAbi)
		if err != nil {
			return nil, err
		}
		deposits[i] = LegacyBridgeEvent{
			Event: &ethDepositEvents[i].ContractEvent,
			BridgeTransfer: database.BridgeTransfer{
				TokenPair: database.ETHTokenPair,
				Tx: database.Transaction{
					FromAddress: bridgeEvent.From,
					ToAddress:   bridgeEvent.To,
					Amount:      bridgeEvent.Amount,
					Data:        bridgeEvent.ExtraData,
					Timestamp:   ethDepositEvents[i].Timestamp,
				},
			},
		}
	}
	for i := range erc20DepositEvents {
		bridgeEvent := bindings.L1StandardBridgeERC20DepositInitiated{Raw: *erc20DepositEvents[i].RLPLog}
		err := UnpackLog(&bridgeEvent, &bridgeEvent.Raw, erc20DepositEventAbi.Name, l1StandardBridgeAbi)
		if err != nil {
			return nil, err
		}
		deposits[len(ethDepositEvents)+i] = LegacyBridgeEvent{
			Event: &erc20DepositEvents[i].ContractEvent,
			BridgeTransfer: database.BridgeTransfer{
				TokenPair: database.TokenPair{LocalTokenAddress: bridgeEvent.L1Token, RemoteTokenAddress: bridgeEvent.L2Token},
				Tx: database.Transaction{
					FromAddress: bridgeEvent.From,
					ToAddress:   bridgeEvent.To,
					Amount:      bridgeEvent.Amount,
					Data:        bridgeEvent.ExtraData,
					Timestamp:   erc20DepositEvents[i].Timestamp,
				},
			},
		}
	}

	return deposits, nil
}

func L2StandardBridgeLegacyWithdrawalInitiatedEvents(contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]LegacyBridgeEvent, error) {
	// The L2StandardBridge ABI contains the legacy events
	l2StandardBridgeAbi, err := bindings.L2StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	withdrawalInitiatedEventAbi := l2StandardBridgeAbi.Events["WithdrawalInitiated"]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: withdrawalInitiatedEventAbi.ID}
	withdrawalEvents, err := db.ContractEvents.L2ContractEventsWithFilter(contractEventFilter, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	withdrawals := make([]LegacyBridgeEvent, len(withdrawalEvents))
	for i := range withdrawalEvents {
		bridgeEvent := bindings.L2StandardBridgeWithdrawalInitiated{Raw: *withdrawalEvents[i].RLPLog}
		err := UnpackLog(&bridgeEvent, &bridgeEvent.Raw, withdrawalInitiatedEventAbi.Name, l2StandardBridgeAbi)
		if err != nil {
			return nil, err
		}

		withdrawals[i] = LegacyBridgeEvent{
			Event: &withdrawalEvents[i].ContractEvent,
			BridgeTransfer: database.BridgeTransfer{
				TokenPair: database.ETHTokenPair,
				Tx: database.Transaction{
					FromAddress: bridgeEvent.From,
					ToAddress:   bridgeEvent.To,
					Amount:      bridgeEvent.Amount,
					Data:        bridgeEvent.ExtraData,
					Timestamp:   withdrawalEvents[i].Timestamp,
				},
			},
		}
	}

	return withdrawals, nil
}
