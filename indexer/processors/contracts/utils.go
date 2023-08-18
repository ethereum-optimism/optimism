package contracts

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

// DecodeVersionNonce is an re-implementation of Encoding.sol#decodeVersionedNonce.
// If the nonce is greater than 32 bytes (solidity uint256), bytes [32:] are ignored
func DecodeVersionedNonce(nonce *big.Int) (uint16, *big.Int) {
	nonceBytes := nonce.Bytes()
	nonceByteLen := len(nonceBytes)
	if nonceByteLen < 30 {
		// version is 0x0000
		return 0, nonce
	} else if nonceByteLen == 31 {
		// version is 0x00[01..ff]
		return uint16(nonceBytes[0]), new(big.Int).SetBytes(nonceBytes[1:])
	} else {
		// fully specified
		version := binary.BigEndian.Uint16(nonceBytes[:2])
		return version, new(big.Int).SetBytes(nonceBytes[2:])
	}
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
