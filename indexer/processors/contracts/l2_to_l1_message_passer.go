package contracts

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/common"
)

type L2ToL1MessagePasserMessagePassed struct {
	Event          *database.ContractEvent
	WithdrawalHash common.Hash
	GasLimit       *big.Int
	Nonce          *big.Int
	Tx             database.Transaction
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
	for i := range messagePassedEvents {
		messagePassed := bindings.L2ToL1MessagePasserMessagePassed{Raw: *messagePassedEvents[i].RLPLog}
		err := UnpackLog(&messagePassed, messagePassedEvents[i].RLPLog, messagePassedAbi.Name, l2ToL1MessagePasserAbi)
		if err != nil {
			return nil, err
		}

		messagesPassed[i] = L2ToL1MessagePasserMessagePassed{
			Event:          &messagePassedEvents[i].ContractEvent,
			WithdrawalHash: messagePassed.WithdrawalHash,
			Nonce:          messagePassed.Nonce,
			GasLimit:       messagePassed.GasLimit,
			Tx: database.Transaction{
				FromAddress: messagePassed.Sender,
				ToAddress:   messagePassed.Target,
				Amount:      messagePassed.Value,
				Data:        messagePassed.Data,
				Timestamp:   messagePassedEvents[i].Timestamp,
			},
		}
	}

	return messagesPassed, nil
}
