package processor

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/common"
)

type L2ToL1MessagePasserMessagePassed struct {
	*bindings.L2ToL1MessagePasserMessagePassed
	Event *database.ContractEvent
}

func L2ToL1MessagePasserMessagePassedEvents(contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]L2ToL1MessagePasserMessagePassed, error) {
	l2ToL1MessagePasserAbi, err := bindings.L2ToL1MessagePasserMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	messagePassedAbi := l2ToL1MessagePasserAbi.Events["MessagePassed"]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: messagePassedAbi.ID}
	messagePassedEvents, err := db.ContractEvents.L2ContractEventsWithFilter(contractEventFilter, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	messagesPassed := make([]L2ToL1MessagePasserMessagePassed, len(messagePassedEvents))
	for i, messagePassedEvent := range messagePassedEvents {
		messagePassed := bindings.L2ToL1MessagePasserMessagePassed{Raw: *messagePassedEvent.RLPLog}
		err := UnpackLog(&messagePassed, messagePassedEvent.RLPLog, messagePassedAbi.Name, l2ToL1MessagePasserAbi)
		if err != nil {
			return nil, err
		}

		messagesPassed[i] = L2ToL1MessagePasserMessagePassed{
			L2ToL1MessagePasserMessagePassed: &messagePassed,
			Event:                            &messagePassedEvent.ContractEvent,
		}
	}

	return messagesPassed, nil
}
