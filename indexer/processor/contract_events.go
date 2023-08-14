package processor

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/indexer/database"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ProcessedContractEventLogIndexKey struct {
	blockHash common.Hash
	index     uint
}

type ProcessedContractEvents struct {
	events            []*database.ContractEvent
	eventsBySignature map[common.Hash][]*database.ContractEvent
	eventByLogIndex   map[ProcessedContractEventLogIndexKey]*database.ContractEvent
}

func NewProcessedContractEvents() *ProcessedContractEvents {
	return &ProcessedContractEvents{
		events:            []*database.ContractEvent{},
		eventsBySignature: make(map[common.Hash][]*database.ContractEvent),
		eventByLogIndex:   make(map[ProcessedContractEventLogIndexKey]*database.ContractEvent),
	}
}

func (p *ProcessedContractEvents) AddLog(log *types.Log, time uint64) *database.ContractEvent {
	event := database.ContractEventFromLog(log, time)
	emptyHash := common.Hash{}

	p.events = append(p.events, &event)
	p.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index}] = &event
	if event.EventSignature != emptyHash {
		p.eventsBySignature[event.EventSignature] = append(p.eventsBySignature[event.EventSignature], &event)
	}

	return &event
}

func UnpackLog(out interface{}, log *types.Log, name string, contractAbi *abi.ABI) error {
	eventAbi, ok := contractAbi.Events[name]
	if !ok {
		return fmt.Errorf("event %s not present in supplied ABI", name)
	} else if len(log.Topics) == 0 {
		return errors.New("anonymous events are not supported")
	} else if log.Topics[0] != eventAbi.ID {
		return errors.New("event signature mismatch")
	}

	err := contractAbi.UnpackIntoInterface(out, name, log.Data)
	if err != nil {
		return err
	}

	// handle topics if present
	if len(log.Topics) > 1 {
		var indexedArgs abi.Arguments
		for _, arg := range eventAbi.Inputs {
			if arg.Indexed {
				indexedArgs = append(indexedArgs, arg)
			}
		}

		// The first topic (event signature) is omitted
		err := abi.ParseTopics(out, indexedArgs, log.Topics[1:])
		if err != nil {
			return err
		}
	}

	return nil
}
