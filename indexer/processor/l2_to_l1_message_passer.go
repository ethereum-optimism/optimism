package processor

import (
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
)

type L2ToL1MessagePasserMessagePassed struct {
	*bindings.L2ToL1MessagePasserMessagePassed
	RawEvent *database.ContractEvent
}

func L2ToL1MessagePasserMessagesPassed(events *ProcessedContractEvents) ([]L2ToL1MessagePasserMessagePassed, error) {
	l2ToL1MessagePasserAbi, err := bindings.L2ToL1MessagePasserMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	eventName := "MessagePassed"
	processedMessagePassedEvents := events.eventsBySignature[l2ToL1MessagePasserAbi.Events[eventName].ID]
	messagesPassed := make([]L2ToL1MessagePasserMessagePassed, len(processedMessagePassedEvents))
	for i, messagePassedEvent := range processedMessagePassedEvents {
		log := events.eventLog[messagePassedEvent.GUID]

		var messagePassed bindings.L2ToL1MessagePasserMessagePassed
		err := UnpackLog(&messagePassed, log, eventName, l2ToL1MessagePasserAbi)
		if err != nil {
			return nil, err
		}

		messagesPassed[i] = L2ToL1MessagePasserMessagePassed{&messagePassed, messagePassedEvent}
	}

	return messagesPassed, nil
}
