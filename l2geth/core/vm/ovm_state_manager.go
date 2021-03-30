package vm

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type stateManagerFunction func(*EVM, *Contract, map[string]interface{}) ([]interface{}, error)

var funcs = map[string]stateManagerFunction{
	"owner":                                    owner,
	"setAccountNonce":                          setAccountNonce,
	"getAccountNonce":                          getAccountNonce,
	"getAccountEthAddress":                     getAccountEthAddress,
	"getContractStorage":                       getContractStorage,
	"putContractStorage":                       putContractStorage,
	"isAuthenticated":                          nativeFunctionTrue,
	"hasAccount":                               nativeFunctionTrue,
	"hasEmptyAccount":                          hasEmptyAccount,
	"hasContractStorage":                       nativeFunctionTrue,
	"testAndSetAccountLoaded":                  testAndSetAccount,
	"testAndSetAccountChanged":                 testAndSetAccount,
	"testAndSetContractStorageLoaded":          testAndSetContractStorageLoaded,
	"testAndSetContractStorageChanged":         testAndSetContractStorageChanged,
	"incrementTotalUncommittedAccounts":        nativeFunctionVoid,
	"incrementTotalUncommittedContractStorage": nativeFunctionVoid,
	"initPendingAccount":                       nativeFunctionVoid,
	"commitPendingAccount":                     nativeFunctionVoid,
}

func callStateManager(input []byte, evm *EVM, contract *Contract) (ret []byte, err error) {
	rawabi := evm.Context.OvmStateManager.ABI
	abi := &rawabi

	method, err := abi.MethodById(input)
	if err != nil {
		return nil, fmt.Errorf("cannot find method id %s: %w", input, err)
	}

	var inputArgs = make(map[string]interface{})
	err = method.Inputs.UnpackIntoMap(inputArgs, input[4:])
	if err != nil {
		return nil, err
	}

	fn, exist := funcs[method.RawName]
	if !exist {
		return nil, fmt.Errorf("Native OVM_StateManager function not found for method '%s'", method.RawName)
	}

	outputArgs, err := fn(evm, contract, inputArgs)
	if err != nil {
		return nil, fmt.Errorf("cannot execute state manager function: %w", err)
	}

	returndata, err := method.Outputs.PackValues(outputArgs)
	if err != nil {
		return nil, fmt.Errorf("cannot pack returndata: %w", err)
	}

	return returndata, nil
}

func owner(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	origin := evm.Context.Origin
	return []interface{}{origin}, nil
}

func setAccountNonce(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	address, ok := args["_address"].(common.Address)
	if !ok {
		return nil, errors.New("Could not parse address arg in setAccountNonce")
	}
	nonce, ok := args["_nonce"].(*big.Int)
	if !ok {
		return nil, errors.New("Could not parse nonce arg in setAccountNonce")
	}
	evm.StateDB.SetNonce(address, nonce.Uint64())
	return []interface{}{}, nil
}

func getAccountNonce(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	address, ok := args["_address"].(common.Address)
	if !ok {
		return nil, errors.New("Could not parse address arg in getAccountNonce")
	}
	nonce := evm.StateDB.GetNonce(address)
	return []interface{}{new(big.Int).SetUint64(reflect.ValueOf(nonce).Uint())}, nil
}

func getAccountEthAddress(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	address, ok := args["_address"].(common.Address)
	if !ok {
		return nil, errors.New("Could not parse address arg in getAccountEthAddress")
	}
	return []interface{}{address}, nil
}

func getContractStorage(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	address, ok := args["_contract"].(common.Address)
	if !ok {
		return nil, errors.New("Could not parse contract arg in getContractStorage")
	}
	_key, ok := args["_key"]
	if !ok {
		return nil, errors.New("Could not parse key arg in getContractStorage")
	}
	key := toHash(_key)
	val := evm.StateDB.GetState(address, key)
	if evm.Context.EthCallSender == nil {
		log.Debug("Got contract storage", "address", address.Hex(), "key", key.Hex(), "val", val.Hex())
	}
	return []interface{}{val}, nil
}

func putContractStorage(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	address, ok := args["_contract"].(common.Address)
	if !ok {
		return nil, errors.New("Could not parse address arg in putContractStorage")
	}
	_key, ok := args["_key"]
	if !ok {
		return nil, errors.New("Could not parse key arg in putContractStorage")
	}
	key := toHash(_key)
	_value, ok := args["_value"]
	if !ok {
		return nil, errors.New("Could not parse value arg in putContractStorage")
	}
	val := toHash(_value)

	// save the block number and address with modified key if it's not an eth_call
	if evm.Context.EthCallSender == nil {
		// save the value before
		before := evm.StateDB.GetState(address, key)
		evm.StateDB.SetState(address, key, val)
		err := evm.StateDB.SetDiffKey(
			evm.Context.BlockNumber,
			address,
			key,
			before != val,
		)
		if err != nil {
			log.Error("Cannot set diff key", "err", err)
		}
		log.Debug("Put contract storage", "address", address.Hex(), "key", key.Hex(), "val", val.Hex())
	} else {
		// otherwise just do the db update
		evm.StateDB.SetState(address, key, val)
	}
	return []interface{}{}, nil
}

func testAndSetAccount(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	address, ok := args["_address"].(common.Address)
	if !ok {
		return nil, errors.New("Could not parse address arg in putContractStorage")
	}

	if evm.Context.EthCallSender == nil {
		err := evm.StateDB.SetDiffAccount(
			evm.Context.BlockNumber,
			address,
		)

		if err != nil {
			log.Error("Cannot set account diff", err)
		}
	}

	return []interface{}{true}, nil
}

func testAndSetContractStorageLoaded(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	return testAndSetContractStorage(evm, contract, args, false)
}

func testAndSetContractStorageChanged(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	return testAndSetContractStorage(evm, contract, args, true)
}

func testAndSetContractStorage(evm *EVM, contract *Contract, args map[string]interface{}, changed bool) ([]interface{}, error) {
	address, ok := args["_contract"].(common.Address)
	if !ok {
		return nil, errors.New("Could not parse address arg in putContractStorage")
	}
	_key, ok := args["_key"]
	if !ok {
		return nil, errors.New("Could not parse key arg in putContractStorage")
	}
	key := toHash(_key)

	if evm.Context.EthCallSender == nil {
		err := evm.StateDB.SetDiffKey(
			evm.Context.BlockNumber,
			address,
			key,
			changed,
		)
		if err != nil {
			log.Error("Cannot set diff key", "err", err)
		}
		log.Debug("Test and Set Contract Storage", "address", address.Hex(), "key", key.Hex(), "changed", changed)
	}

	return []interface{}{true}, nil
}

func hasEmptyAccount(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	address, ok := args["_address"].(common.Address)
	if !ok {
		return nil, errors.New("Could not parse address arg in hasEmptyAccount")
	}

	contractHash := evm.StateDB.GetCodeHash(address)
	if evm.StateDB.GetNonce(address) != 0 || (contractHash != (common.Hash{}) && contractHash != emptyCodeHash) {
		return []interface{}{false}, nil
	}

	return []interface{}{true}, nil
}

func nativeFunctionTrue(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	return []interface{}{true}, nil
}

func nativeFunctionVoid(evm *EVM, contract *Contract, args map[string]interface{}) ([]interface{}, error) {
	return []interface{}{}, nil
}

func toHash(arg interface{}) common.Hash {
	b := arg.([32]uint8)
	return common.BytesToHash(b[:])
}
