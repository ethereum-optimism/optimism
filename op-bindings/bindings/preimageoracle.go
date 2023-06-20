// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// PreimageOracleMetaData contains all meta data concerning the PreimageOracle contract.
var PreimageOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oracleWriter\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"PreimageKey\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"PreimageOffset\",\"name\":\"offset\",\"type\":\"uint256\"}],\"name\":\"MissingPreimage\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"PreimageOffset\",\"name\":\"partOffset\",\"type\":\"uint256\"},{\"internalType\":\"PreimageKey\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"PreimagePart\",\"name\":\"part\",\"type\":\"bytes32\"},{\"internalType\":\"PreimageLength\",\"name\":\"size\",\"type\":\"uint256\"}],\"name\":\"cheat\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"preimage\",\"type\":\"bytes\"}],\"name\":\"computePreimageKey\",\"outputs\":[{\"internalType\":\"PreimageKey\",\"name\":\"key\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oracleWriter\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PreimageOffset\",\"name\":\"partOffset\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"preimage\",\"type\":\"bytes\"}],\"name\":\"loadKeccak256PreimagePart\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"oracleWriter\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PreimageKey\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"preimageLengths\",\"outputs\":[{\"internalType\":\"PreimageLength\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PreimageKey\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"PreimageOffset\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"preimagePartOk\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PreimageKey\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"PreimageOffset\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"preimageParts\",\"outputs\":[{\"internalType\":\"PreimagePart\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PreimageKey\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"PreimageOffset\",\"name\":\"offset\",\"type\":\"uint256\"}],\"name\":\"readPreimage\",\"outputs\":[{\"internalType\":\"PreimagePart\",\"name\":\"dat\",\"type\":\"bytes32\"},{\"internalType\":\"PreimageLength\",\"name\":\"datLen\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50604051610a3a380380610a3a83398101604081905261002f9161018e565b6100388161003e565b506101be565b600054610100900460ff161580801561005e5750600054600160ff909116105b8061008957506100773061017f60201b6106651760201c565b158015610089575060005460ff166001145b6100f05760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b606482015260840160405180910390fd5b6000805460ff191660011790558015610113576000805461ff0019166101001790555b6000805462010000600160b01b031916620100006001600160a01b03851602179055801561017b576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050565b6001600160a01b03163b151590565b6000602082840312156101a057600080fd5b81516001600160a01b03811681146101b757600080fd5b9392505050565b61086d806101cd6000396000f3fe608060405234801561001057600080fd5b50600436106100a35760003560e01c8063c4d66de811610076578063e15926111161005b578063e1592611146101bf578063fe4ac08e146101d2578063fef2b4ed1461024957600080fd5b8063c4d66de814610182578063e03110e11461019757600080fd5b806361238bde146100a85780638542cf50146100e65780638ec4e64b14610124578063a57c202c1461016f575b600080fd5b6100d36100b6366004610681565b600260209081526000928352604080842090915290825290205481565b6040519081526020015b60405180910390f35b6101146100f4366004610681565b600360209081526000928352604080842090915290825290205460ff1681565b60405190151581526020016100dd565b60005461014a9062010000900473ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100dd565b6100d361017d3660046106ec565b610269565b61019561019036600461072e565b6102c8565b005b6101aa6101a5366004610681565b61049d565b604080519283526020830191909152016100dd565b6101956101cd36600461076b565b610566565b6101956101e03660046107b7565b6000838152600360209081526040808320878452825280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600190811790915586845260028352818420978452968252808320949094559381529390925290912055565b6100d36102573660046107e9565b60016020526000908152604090205481565b60243560c081901b608052600090608881858237207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f0200000000000000000000000000000000000000000000000000000000000000179392505050565b600054610100900460ff16158080156102e85750600054600160ff909116105b806103025750303b158015610302575060005460ff166001145b610393576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084015b60405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600117905580156103f157600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b600080547fffffffffffffffffffff0000000000000000000000000000000000000000ffff166201000073ffffffffffffffffffffffffffffffffffffffff851602179055801561049957600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050565b6000828152600360209081526040808320848452909152812054819060ff166104fc576040517f5b76e337000000000000000000000000000000000000000000000000000000008152600481018590526024810184905260440161038a565b5060008381526001602090815260409091205461051a816008610831565b610525856020610831565b106105435783610536826008610831565b6105409190610849565b91505b506000938452600260209081526040808620948652939052919092205492909150565b60443560008060088301861061057b57600080fd5b60c083901b6080526088838682378087017ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80151908490207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f02000000000000000000000000000000000000000000000000000000000000001760008181526003602090815260408083208b8452825280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001908117909155848452600283528184209b84529a8252808320949094559181529790529095209190915550505050565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b6000806040838503121561069457600080fd5b50508035926020909101359150565b60008083601f8401126106b557600080fd5b50813567ffffffffffffffff8111156106cd57600080fd5b6020830191508360208285010111156106e557600080fd5b9250929050565b600080602083850312156106ff57600080fd5b823567ffffffffffffffff81111561071657600080fd5b610722858286016106a3565b90969095509350505050565b60006020828403121561074057600080fd5b813573ffffffffffffffffffffffffffffffffffffffff8116811461076457600080fd5b9392505050565b60008060006040848603121561078057600080fd5b83359250602084013567ffffffffffffffff81111561079e57600080fd5b6107aa868287016106a3565b9497909650939450505050565b600080600080608085870312156107cd57600080fd5b5050823594602084013594506040840135936060013592509050565b6000602082840312156107fb57600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000821982111561084457610844610802565b500190565b60008282101561085b5761085b610802565b50039056fea164736f6c634300080f000a",
}

// PreimageOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use PreimageOracleMetaData.ABI instead.
var PreimageOracleABI = PreimageOracleMetaData.ABI

// PreimageOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use PreimageOracleMetaData.Bin instead.
var PreimageOracleBin = PreimageOracleMetaData.Bin

// DeployPreimageOracle deploys a new Ethereum contract, binding an instance of PreimageOracle to it.
func DeployPreimageOracle(auth *bind.TransactOpts, backend bind.ContractBackend, _oracleWriter common.Address) (common.Address, *types.Transaction, *PreimageOracle, error) {
	parsed, err := PreimageOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PreimageOracleBin), backend, _oracleWriter)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PreimageOracle{PreimageOracleCaller: PreimageOracleCaller{contract: contract}, PreimageOracleTransactor: PreimageOracleTransactor{contract: contract}, PreimageOracleFilterer: PreimageOracleFilterer{contract: contract}}, nil
}

// PreimageOracle is an auto generated Go binding around an Ethereum contract.
type PreimageOracle struct {
	PreimageOracleCaller     // Read-only binding to the contract
	PreimageOracleTransactor // Write-only binding to the contract
	PreimageOracleFilterer   // Log filterer for contract events
}

// PreimageOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type PreimageOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PreimageOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PreimageOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PreimageOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PreimageOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PreimageOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PreimageOracleSession struct {
	Contract     *PreimageOracle   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PreimageOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PreimageOracleCallerSession struct {
	Contract *PreimageOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// PreimageOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PreimageOracleTransactorSession struct {
	Contract     *PreimageOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// PreimageOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type PreimageOracleRaw struct {
	Contract *PreimageOracle // Generic contract binding to access the raw methods on
}

// PreimageOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PreimageOracleCallerRaw struct {
	Contract *PreimageOracleCaller // Generic read-only contract binding to access the raw methods on
}

// PreimageOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PreimageOracleTransactorRaw struct {
	Contract *PreimageOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPreimageOracle creates a new instance of PreimageOracle, bound to a specific deployed contract.
func NewPreimageOracle(address common.Address, backend bind.ContractBackend) (*PreimageOracle, error) {
	contract, err := bindPreimageOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PreimageOracle{PreimageOracleCaller: PreimageOracleCaller{contract: contract}, PreimageOracleTransactor: PreimageOracleTransactor{contract: contract}, PreimageOracleFilterer: PreimageOracleFilterer{contract: contract}}, nil
}

// NewPreimageOracleCaller creates a new read-only instance of PreimageOracle, bound to a specific deployed contract.
func NewPreimageOracleCaller(address common.Address, caller bind.ContractCaller) (*PreimageOracleCaller, error) {
	contract, err := bindPreimageOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PreimageOracleCaller{contract: contract}, nil
}

// NewPreimageOracleTransactor creates a new write-only instance of PreimageOracle, bound to a specific deployed contract.
func NewPreimageOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*PreimageOracleTransactor, error) {
	contract, err := bindPreimageOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PreimageOracleTransactor{contract: contract}, nil
}

// NewPreimageOracleFilterer creates a new log filterer instance of PreimageOracle, bound to a specific deployed contract.
func NewPreimageOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*PreimageOracleFilterer, error) {
	contract, err := bindPreimageOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PreimageOracleFilterer{contract: contract}, nil
}

// bindPreimageOracle binds a generic wrapper to an already deployed contract.
func bindPreimageOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PreimageOracleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PreimageOracle *PreimageOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PreimageOracle.Contract.PreimageOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PreimageOracle *PreimageOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PreimageOracle.Contract.PreimageOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PreimageOracle *PreimageOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PreimageOracle.Contract.PreimageOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PreimageOracle *PreimageOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PreimageOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PreimageOracle *PreimageOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PreimageOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PreimageOracle *PreimageOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PreimageOracle.Contract.contract.Transact(opts, method, params...)
}

// ComputePreimageKey is a free data retrieval call binding the contract method 0xa57c202c.
//
// Solidity: function computePreimageKey(bytes preimage) pure returns(bytes32 key)
func (_PreimageOracle *PreimageOracleCaller) ComputePreimageKey(opts *bind.CallOpts, preimage []byte) ([32]byte, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "computePreimageKey", preimage)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ComputePreimageKey is a free data retrieval call binding the contract method 0xa57c202c.
//
// Solidity: function computePreimageKey(bytes preimage) pure returns(bytes32 key)
func (_PreimageOracle *PreimageOracleSession) ComputePreimageKey(preimage []byte) ([32]byte, error) {
	return _PreimageOracle.Contract.ComputePreimageKey(&_PreimageOracle.CallOpts, preimage)
}

// ComputePreimageKey is a free data retrieval call binding the contract method 0xa57c202c.
//
// Solidity: function computePreimageKey(bytes preimage) pure returns(bytes32 key)
func (_PreimageOracle *PreimageOracleCallerSession) ComputePreimageKey(preimage []byte) ([32]byte, error) {
	return _PreimageOracle.Contract.ComputePreimageKey(&_PreimageOracle.CallOpts, preimage)
}

// OracleWriter is a free data retrieval call binding the contract method 0x8ec4e64b.
//
// Solidity: function oracleWriter() view returns(address)
func (_PreimageOracle *PreimageOracleCaller) OracleWriter(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "oracleWriter")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OracleWriter is a free data retrieval call binding the contract method 0x8ec4e64b.
//
// Solidity: function oracleWriter() view returns(address)
func (_PreimageOracle *PreimageOracleSession) OracleWriter() (common.Address, error) {
	return _PreimageOracle.Contract.OracleWriter(&_PreimageOracle.CallOpts)
}

// OracleWriter is a free data retrieval call binding the contract method 0x8ec4e64b.
//
// Solidity: function oracleWriter() view returns(address)
func (_PreimageOracle *PreimageOracleCallerSession) OracleWriter() (common.Address, error) {
	return _PreimageOracle.Contract.OracleWriter(&_PreimageOracle.CallOpts)
}

// PreimageLengths is a free data retrieval call binding the contract method 0xfef2b4ed.
//
// Solidity: function preimageLengths(bytes32 ) view returns(uint256)
func (_PreimageOracle *PreimageOracleCaller) PreimageLengths(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "preimageLengths", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PreimageLengths is a free data retrieval call binding the contract method 0xfef2b4ed.
//
// Solidity: function preimageLengths(bytes32 ) view returns(uint256)
func (_PreimageOracle *PreimageOracleSession) PreimageLengths(arg0 [32]byte) (*big.Int, error) {
	return _PreimageOracle.Contract.PreimageLengths(&_PreimageOracle.CallOpts, arg0)
}

// PreimageLengths is a free data retrieval call binding the contract method 0xfef2b4ed.
//
// Solidity: function preimageLengths(bytes32 ) view returns(uint256)
func (_PreimageOracle *PreimageOracleCallerSession) PreimageLengths(arg0 [32]byte) (*big.Int, error) {
	return _PreimageOracle.Contract.PreimageLengths(&_PreimageOracle.CallOpts, arg0)
}

// PreimagePartOk is a free data retrieval call binding the contract method 0x8542cf50.
//
// Solidity: function preimagePartOk(bytes32 , uint256 ) view returns(bool)
func (_PreimageOracle *PreimageOracleCaller) PreimagePartOk(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) (bool, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "preimagePartOk", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PreimagePartOk is a free data retrieval call binding the contract method 0x8542cf50.
//
// Solidity: function preimagePartOk(bytes32 , uint256 ) view returns(bool)
func (_PreimageOracle *PreimageOracleSession) PreimagePartOk(arg0 [32]byte, arg1 *big.Int) (bool, error) {
	return _PreimageOracle.Contract.PreimagePartOk(&_PreimageOracle.CallOpts, arg0, arg1)
}

// PreimagePartOk is a free data retrieval call binding the contract method 0x8542cf50.
//
// Solidity: function preimagePartOk(bytes32 , uint256 ) view returns(bool)
func (_PreimageOracle *PreimageOracleCallerSession) PreimagePartOk(arg0 [32]byte, arg1 *big.Int) (bool, error) {
	return _PreimageOracle.Contract.PreimagePartOk(&_PreimageOracle.CallOpts, arg0, arg1)
}

// PreimageParts is a free data retrieval call binding the contract method 0x61238bde.
//
// Solidity: function preimageParts(bytes32 , uint256 ) view returns(bytes32)
func (_PreimageOracle *PreimageOracleCaller) PreimageParts(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "preimageParts", arg0, arg1)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PreimageParts is a free data retrieval call binding the contract method 0x61238bde.
//
// Solidity: function preimageParts(bytes32 , uint256 ) view returns(bytes32)
func (_PreimageOracle *PreimageOracleSession) PreimageParts(arg0 [32]byte, arg1 *big.Int) ([32]byte, error) {
	return _PreimageOracle.Contract.PreimageParts(&_PreimageOracle.CallOpts, arg0, arg1)
}

// PreimageParts is a free data retrieval call binding the contract method 0x61238bde.
//
// Solidity: function preimageParts(bytes32 , uint256 ) view returns(bytes32)
func (_PreimageOracle *PreimageOracleCallerSession) PreimageParts(arg0 [32]byte, arg1 *big.Int) ([32]byte, error) {
	return _PreimageOracle.Contract.PreimageParts(&_PreimageOracle.CallOpts, arg0, arg1)
}

// ReadPreimage is a free data retrieval call binding the contract method 0xe03110e1.
//
// Solidity: function readPreimage(bytes32 key, uint256 offset) view returns(bytes32 dat, uint256 datLen)
func (_PreimageOracle *PreimageOracleCaller) ReadPreimage(opts *bind.CallOpts, key [32]byte, offset *big.Int) (struct {
	Dat    [32]byte
	DatLen *big.Int
}, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "readPreimage", key, offset)

	outstruct := new(struct {
		Dat    [32]byte
		DatLen *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Dat = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.DatLen = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// ReadPreimage is a free data retrieval call binding the contract method 0xe03110e1.
//
// Solidity: function readPreimage(bytes32 key, uint256 offset) view returns(bytes32 dat, uint256 datLen)
func (_PreimageOracle *PreimageOracleSession) ReadPreimage(key [32]byte, offset *big.Int) (struct {
	Dat    [32]byte
	DatLen *big.Int
}, error) {
	return _PreimageOracle.Contract.ReadPreimage(&_PreimageOracle.CallOpts, key, offset)
}

// ReadPreimage is a free data retrieval call binding the contract method 0xe03110e1.
//
// Solidity: function readPreimage(bytes32 key, uint256 offset) view returns(bytes32 dat, uint256 datLen)
func (_PreimageOracle *PreimageOracleCallerSession) ReadPreimage(key [32]byte, offset *big.Int) (struct {
	Dat    [32]byte
	DatLen *big.Int
}, error) {
	return _PreimageOracle.Contract.ReadPreimage(&_PreimageOracle.CallOpts, key, offset)
}

// Cheat is a paid mutator transaction binding the contract method 0xfe4ac08e.
//
// Solidity: function cheat(uint256 partOffset, bytes32 key, bytes32 part, uint256 size) returns()
func (_PreimageOracle *PreimageOracleTransactor) Cheat(opts *bind.TransactOpts, partOffset *big.Int, key [32]byte, part [32]byte, size *big.Int) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "cheat", partOffset, key, part, size)
}

// Cheat is a paid mutator transaction binding the contract method 0xfe4ac08e.
//
// Solidity: function cheat(uint256 partOffset, bytes32 key, bytes32 part, uint256 size) returns()
func (_PreimageOracle *PreimageOracleSession) Cheat(partOffset *big.Int, key [32]byte, part [32]byte, size *big.Int) (*types.Transaction, error) {
	return _PreimageOracle.Contract.Cheat(&_PreimageOracle.TransactOpts, partOffset, key, part, size)
}

// Cheat is a paid mutator transaction binding the contract method 0xfe4ac08e.
//
// Solidity: function cheat(uint256 partOffset, bytes32 key, bytes32 part, uint256 size) returns()
func (_PreimageOracle *PreimageOracleTransactorSession) Cheat(partOffset *big.Int, key [32]byte, part [32]byte, size *big.Int) (*types.Transaction, error) {
	return _PreimageOracle.Contract.Cheat(&_PreimageOracle.TransactOpts, partOffset, key, part, size)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _oracleWriter) returns()
func (_PreimageOracle *PreimageOracleTransactor) Initialize(opts *bind.TransactOpts, _oracleWriter common.Address) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "initialize", _oracleWriter)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _oracleWriter) returns()
func (_PreimageOracle *PreimageOracleSession) Initialize(_oracleWriter common.Address) (*types.Transaction, error) {
	return _PreimageOracle.Contract.Initialize(&_PreimageOracle.TransactOpts, _oracleWriter)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _oracleWriter) returns()
func (_PreimageOracle *PreimageOracleTransactorSession) Initialize(_oracleWriter common.Address) (*types.Transaction, error) {
	return _PreimageOracle.Contract.Initialize(&_PreimageOracle.TransactOpts, _oracleWriter)
}

// LoadKeccak256PreimagePart is a paid mutator transaction binding the contract method 0xe1592611.
//
// Solidity: function loadKeccak256PreimagePart(uint256 partOffset, bytes preimage) returns()
func (_PreimageOracle *PreimageOracleTransactor) LoadKeccak256PreimagePart(opts *bind.TransactOpts, partOffset *big.Int, preimage []byte) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "loadKeccak256PreimagePart", partOffset, preimage)
}

// LoadKeccak256PreimagePart is a paid mutator transaction binding the contract method 0xe1592611.
//
// Solidity: function loadKeccak256PreimagePart(uint256 partOffset, bytes preimage) returns()
func (_PreimageOracle *PreimageOracleSession) LoadKeccak256PreimagePart(partOffset *big.Int, preimage []byte) (*types.Transaction, error) {
	return _PreimageOracle.Contract.LoadKeccak256PreimagePart(&_PreimageOracle.TransactOpts, partOffset, preimage)
}

// LoadKeccak256PreimagePart is a paid mutator transaction binding the contract method 0xe1592611.
//
// Solidity: function loadKeccak256PreimagePart(uint256 partOffset, bytes preimage) returns()
func (_PreimageOracle *PreimageOracleTransactorSession) LoadKeccak256PreimagePart(partOffset *big.Int, preimage []byte) (*types.Transaction, error) {
	return _PreimageOracle.Contract.LoadKeccak256PreimagePart(&_PreimageOracle.TransactOpts, partOffset, preimage)
}

// PreimageOracleInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the PreimageOracle contract.
type PreimageOracleInitializedIterator struct {
	Event *PreimageOracleInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PreimageOracleInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PreimageOracleInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PreimageOracleInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PreimageOracleInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PreimageOracleInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PreimageOracleInitialized represents a Initialized event raised by the PreimageOracle contract.
type PreimageOracleInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_PreimageOracle *PreimageOracleFilterer) FilterInitialized(opts *bind.FilterOpts) (*PreimageOracleInitializedIterator, error) {

	logs, sub, err := _PreimageOracle.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &PreimageOracleInitializedIterator{contract: _PreimageOracle.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_PreimageOracle *PreimageOracleFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *PreimageOracleInitialized) (event.Subscription, error) {

	logs, sub, err := _PreimageOracle.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PreimageOracleInitialized)
				if err := _PreimageOracle.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_PreimageOracle *PreimageOracleFilterer) ParseInitialized(log types.Log) (*PreimageOracleInitialized, error) {
	event := new(PreimageOracleInitialized)
	if err := _PreimageOracle.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
