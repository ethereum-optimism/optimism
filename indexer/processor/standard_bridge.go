package processor

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"

	"github.com/ethereum/go-ethereum/common"
)

type StandardBridgeInitiatedEvent struct {
	// We use ERC20 since ETH can be pseudo-represented with the predeploy address
	*bindings.StandardBridgeERC20BridgeInitiated
	Event *database.ContractEvent
}

type StandardBridgeFinalizedEvent struct {
	// Weuse ERC20 since ETH can be pseudo-represented with the predeploy address
	*bindings.StandardBridgeERC20BridgeFinalized
	Event *database.ContractEvent
}

// StandardBridgeInitiatedEvents extracts all initiated bridge events from the contracts that follow the StandardBridge ABI. The
// correlated CrossDomainMessenger nonce is also parsed from the associated messenger events.
func StandardBridgeInitiatedEvents(contractAddress common.Address, chain string, db *database.DB, fromHeight, toHeight *big.Int) ([]StandardBridgeInitiatedEvent, error) {
	ethBridgeInitiatedEvents, err := _standardBridgeInitiatedEvents[bindings.StandardBridgeETHBridgeInitiated](contractAddress, chain, db, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	erc20BridgeInitiatedEvents, err := _standardBridgeInitiatedEvents[bindings.StandardBridgeERC20BridgeInitiated](contractAddress, chain, db, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	return append(ethBridgeInitiatedEvents, erc20BridgeInitiatedEvents...), nil
}

// StandardBridgeFinalizedEvents extracts all finalization bridge events from the contracts that follow the StandardBridge ABI. The
// correlated CrossDomainMessenger nonce is also parsed by looking at the parameters of the corresponding relayMessage transaction data.
func StandardBridgeFinalizedEvents(contractAddress common.Address, chain string, db *database.DB, fromHeight, toHeight *big.Int) ([]StandardBridgeFinalizedEvent, error) {
	ethBridgeFinalizedEvents, err := _standardBridgeFinalizedEvents[bindings.StandardBridgeETHBridgeFinalized](contractAddress, chain, db, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	erc20BridgeFinalizedEvents, err := _standardBridgeFinalizedEvents[bindings.StandardBridgeERC20BridgeFinalized](contractAddress, chain, db, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	return append(ethBridgeFinalizedEvents, erc20BridgeFinalizedEvents...), nil
}

// parse out eth or erc20 bridge initiated events
func _standardBridgeInitiatedEvents[BridgeEvent bindings.StandardBridgeETHBridgeInitiated | bindings.StandardBridgeERC20BridgeInitiated](
	contractAddress common.Address, chain string, db *database.DB, fromHeight, toHeight *big.Int,
) ([]StandardBridgeInitiatedEvent, error) {
	standardBridgeAbi, err := bindings.StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	var tmp BridgeEvent
	var eventName string
	switch any(tmp).(type) {
	case bindings.StandardBridgeETHBridgeInitiated:
		eventName = "ETHBridgeInitiated"
	case bindings.StandardBridgeERC20BridgeInitiated:
		eventName = "ERC20BridgeInitiated"
	default:
		panic("should not be here")
	}

	initiatedBridgeEventAbi := standardBridgeAbi.Events[eventName]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: initiatedBridgeEventAbi.ID}
	initiatedBridgeEvents, err := db.ContractEvents.ContractEventsWithFilter(contractEventFilter, chain, fromHeight, toHeight)
	standardBridgeInitiatedEvents := make([]StandardBridgeInitiatedEvent, len(initiatedBridgeEvents))
	for i := range initiatedBridgeEvents {
		erc20Bridge := bindings.StandardBridgeERC20BridgeInitiated{Raw: *initiatedBridgeEvents[i].RLPLog}
		err := UnpackLog(&erc20Bridge, initiatedBridgeEvents[i].RLPLog, eventName, standardBridgeAbi)
		if err != nil {
			return nil, err
		}

		// If an ETH bridge, lets fill in the needed fields
		switch any(tmp).(type) {
		case bindings.StandardBridgeETHBridgeInitiated:
			erc20Bridge.LocalToken = predeploys.LegacyERC20ETHAddr
			erc20Bridge.RemoteToken = predeploys.LegacyERC20ETHAddr
		}

		standardBridgeInitiatedEvents[i] = StandardBridgeInitiatedEvent{
			StandardBridgeERC20BridgeInitiated: &erc20Bridge,
			Event:                              &initiatedBridgeEvents[i],
		}
	}

	return standardBridgeInitiatedEvents, nil
}

// parse out eth or erc20 bridge finalization events
func _standardBridgeFinalizedEvents[BridgeEvent bindings.StandardBridgeETHBridgeFinalized | bindings.StandardBridgeERC20BridgeFinalized](
	contractAddress common.Address, chain string, db *database.DB, fromHeight, toHeight *big.Int,
) ([]StandardBridgeFinalizedEvent, error) {
	standardBridgeAbi, err := bindings.StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	var tmp BridgeEvent
	var eventName string
	switch any(tmp).(type) {
	case bindings.StandardBridgeETHBridgeFinalized:
		eventName = "ETHBridgeFinalized"
	case bindings.StandardBridgeERC20BridgeFinalized:
		eventName = "ERC20BridgeFinalized"
	default:
		panic("should not be here")
	}

	bridgeFinalizedEventAbi := standardBridgeAbi.Events[eventName]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: bridgeFinalizedEventAbi.ID}
	bridgeFinalizedEvents, err := db.ContractEvents.ContractEventsWithFilter(contractEventFilter, chain, fromHeight, toHeight)
	standardBridgeFinalizedEvents := make([]StandardBridgeFinalizedEvent, len(bridgeFinalizedEvents))
	for i := range bridgeFinalizedEvents {
		erc20Bridge := bindings.StandardBridgeERC20BridgeFinalized{Raw: *bridgeFinalizedEvents[i].RLPLog}
		err := UnpackLog(&erc20Bridge, bridgeFinalizedEvents[i].RLPLog, eventName, standardBridgeAbi)
		if err != nil {
			return nil, err
		}

		// If an ETH bridge, lets fill in the needed fields
		switch any(tmp).(type) {
		case bindings.StandardBridgeETHBridgeFinalized:
			erc20Bridge.LocalToken = predeploys.LegacyERC20ETHAddr
			erc20Bridge.RemoteToken = predeploys.LegacyERC20ETHAddr
		}

		standardBridgeFinalizedEvents[i] = StandardBridgeFinalizedEvent{
			StandardBridgeERC20BridgeFinalized: &erc20Bridge,
			Event:                              &bridgeFinalizedEvents[i],
		}
	}

	return standardBridgeFinalizedEvents, nil
}
