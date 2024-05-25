package contracts

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

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
