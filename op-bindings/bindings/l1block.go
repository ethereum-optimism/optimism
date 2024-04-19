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
)

// L1BlockMetaData contains all meta data concerning the L1Block contract.
var L1BlockMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"DEPOSITOR_ACCOUNT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"baseFeeScalar\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"basefee\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"batcherHash\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"blobBaseFee\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"blobBaseFeeScalar\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gasPayingToken\",\"inputs\":[],\"outputs\":[{\"name\":\"addr_\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"decimals_\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gasPayingTokenName\",\"inputs\":[],\"outputs\":[{\"name\":\"name_\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gasPayingTokenSymbol\",\"inputs\":[],\"outputs\":[{\"name\":\"symbol_\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"hash\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isCustomGasToken\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l1FeeOverhead\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l1FeeScalar\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"number\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"sequenceNumber\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setGasPayingToken\",\"inputs\":[{\"name\":\"_token\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_decimals\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"_name\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_symbol\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setL1BlockValues\",\"inputs\":[{\"name\":\"_number\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"_timestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"_basefee\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_hash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_sequenceNumber\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"_batcherHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_l1FeeOverhead\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_l1FeeScalar\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setL1BlockValuesEcotone\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"timestamp\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"GasPayingTokenSet\",\"inputs\":[{\"name\":\"token\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"decimals\",\"type\":\"uint8\",\"indexed\":true,\"internalType\":\"uint8\"},{\"name\":\"name\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"symbol\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"NotDepositor\",\"inputs\":[]}]",
	Bin: "0x608060405234801561001057600080fd5b50610ae8806100206000396000f3fe608060405234801561001057600080fd5b506004361061016c5760003560e01c806371cfaa3f116100cd578063c598591811610081578063e591b28211610066578063e591b28214610331578063e81b2c6d14610371578063f82061401461037a57600080fd5b8063c598591814610309578063d84447151461032957600080fd5b80638b239f73116100b25780638b239f73146102d75780639e8c4966146102e0578063b80777ea146102e957600080fd5b806371cfaa3f146102b05780638381f58a146102c357600080fd5b806354fd4d50116101245780635cf24969116101095780635cf249691461024957806364ca23ef1461025257806368d5dca61461027f57600080fd5b806354fd4d50146101f8578063550fcdc91461024157600080fd5b8063213268491161015557806321326849146101a25780634397dfef146101ba578063440a5e20146101f057600080fd5b8063015d8eb91461017157806309bd5a6014610186575b600080fd5b61018461017f366004610953565b610383565b005b61018f60025481565b6040519081526020015b60405180910390f35b6101aa6104c2565b6040519015158152602001610199565b6101c2610501565b6040805173ffffffffffffffffffffffffffffffffffffffff909316835260ff909116602083015201610199565b610184610515565b6102346040518060400160405280601b81526020017f312e332e302d626574612b637573746f6d2d6761732d746f6b656e000000000081525081565b60405161019991906109c5565b61023461056a565b61018f60015481565b6003546102669067ffffffffffffffff1681565b60405167ffffffffffffffff9091168152602001610199565b60035461029b9068010000000000000000900463ffffffff1681565b60405163ffffffff9091168152602001610199565b6101846102be366004610a38565b610579565b6000546102669067ffffffffffffffff1681565b61018f60055481565b61018f60065481565b6000546102669068010000000000000000900467ffffffffffffffff1681565b60035461029b906c01000000000000000000000000900463ffffffff1681565b61023461062e565b61034c73deaddeaddeaddeaddeaddeaddeaddeaddead000181565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610199565b61018f60045481565b61018f60075481565b3373deaddeaddeaddeaddeaddeaddeaddeaddead00011461042a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603b60248201527f4c31426c6f636b3a206f6e6c7920746865206465706f7369746f72206163636f60448201527f756e742063616e20736574204c3120626c6f636b2076616c7565730000000000606482015260840160405180910390fd5b6000805467ffffffffffffffff98891668010000000000000000027fffffffffffffffffffffffffffffffff00000000000000000000000000000000909116998916999099179890981790975560019490945560029290925560038054919094167fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000009190911617909255600491909155600555600655565b6000806104cd610501565b5073ffffffffffffffffffffffffffffffffffffffff1673eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee141592915050565b60008061050c610638565b90939092509050565b3373deaddeaddeaddeaddeaddeaddeaddeaddead00011461053e57633cc50b456000526004601cfd5b60043560801c60035560143560801c600055602435600155604435600755606435600255608435600455565b60606105746106b9565b905090565b3373deaddeaddeaddeaddeaddeaddeaddeaddead0001146105c6576040517f3cc50b4500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6105d28484848461077a565b604080518381526020810183905260ff85169173ffffffffffffffffffffffffffffffffffffffff8716917f10e43c4d58f3ef4edae7c1ca2e7f02d46b2cadbcc046737038527ed8486ffeb0910160405180910390a350505050565b606061057461084c565b6000808061066e61066a60017f04adb1412b2ddc16fcc0d4538d5c8f07cf9c83abecc6b41f6f69037b708fbcec610a9d565b5490565b73ffffffffffffffffffffffffffffffffffffffff811693509050826106ad575073eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee92601292509050565b60a081901c9150509091565b606060006106c5610638565b5090507fffffffffffffffffffffffff111111111111111111111111111111111111111273ffffffffffffffffffffffffffffffffffffffff82160161073e57505060408051808201909152600381527f4554480000000000000000000000000000000000000000000000000000000000602082015290565b61077461076f61066a60017fa48b38a4b44951360fbdcbfaaeae5ed6ae92585412e9841b70ec72ed8cd05764610a9d565b610902565b91505090565b6107e06107a860017f04adb1412b2ddc16fcc0d4538d5c8f07cf9c83abecc6b41f6f69037b708fbcec610a9d565b74ff000000000000000000000000000000000000000060a086901b1673ffffffffffffffffffffffffffffffffffffffff8716179055565b61081361080e60017f657c3582c29b3176614e3a33ddd1ec48352696a04e92b3c0566d72010fa8863d610a9d565b839055565b61084661084160017fa48b38a4b44951360fbdcbfaaeae5ed6ae92585412e9841b70ec72ed8cd05764610a9d565b829055565b50505050565b60606000610858610638565b5090507fffffffffffffffffffffffff111111111111111111111111111111111111111273ffffffffffffffffffffffffffffffffffffffff8216016108d157505060408051808201909152600581527f4574686572000000000000000000000000000000000000000000000000000000602082015290565b61077461076f61066a60017f657c3582c29b3176614e3a33ddd1ec48352696a04e92b3c0566d72010fa8863d610a9d565b60405160005b82811a1561091857600101610908565b80825260208201838152600082820152505060408101604052919050565b803567ffffffffffffffff8116811461094e57600080fd5b919050565b600080600080600080600080610100898b03121561097057600080fd5b61097989610936565b975061098760208a01610936565b965060408901359550606089013594506109a360808a01610936565b979a969950949793969560a0850135955060c08501359460e001359350915050565b600060208083528351808285015260005b818110156109f2578581018301518582016040015282016109d6565b81811115610a04576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b60008060008060808587031215610a4e57600080fd5b843573ffffffffffffffffffffffffffffffffffffffff81168114610a7257600080fd5b9350602085013560ff81168114610a8857600080fd5b93969395505050506040820135916060013590565b600082821015610ad6577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b50039056fea164736f6c634300080f000a",
}

// L1BlockABI is the input ABI used to generate the binding from.
// Deprecated: Use L1BlockMetaData.ABI instead.
var L1BlockABI = L1BlockMetaData.ABI

// L1BlockBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1BlockMetaData.Bin instead.
var L1BlockBin = L1BlockMetaData.Bin

// DeployL1Block deploys a new Ethereum contract, binding an instance of L1Block to it.
func DeployL1Block(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L1Block, error) {
	parsed, err := L1BlockMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1BlockBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1Block{L1BlockCaller: L1BlockCaller{contract: contract}, L1BlockTransactor: L1BlockTransactor{contract: contract}, L1BlockFilterer: L1BlockFilterer{contract: contract}}, nil
}

// L1Block is an auto generated Go binding around an Ethereum contract.
type L1Block struct {
	L1BlockCaller     // Read-only binding to the contract
	L1BlockTransactor // Write-only binding to the contract
	L1BlockFilterer   // Log filterer for contract events
}

// L1BlockCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1BlockCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1BlockTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1BlockFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1BlockSession struct {
	Contract     *L1Block          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1BlockCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1BlockCallerSession struct {
	Contract *L1BlockCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// L1BlockTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1BlockTransactorSession struct {
	Contract     *L1BlockTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// L1BlockRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1BlockRaw struct {
	Contract *L1Block // Generic contract binding to access the raw methods on
}

// L1BlockCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1BlockCallerRaw struct {
	Contract *L1BlockCaller // Generic read-only contract binding to access the raw methods on
}

// L1BlockTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1BlockTransactorRaw struct {
	Contract *L1BlockTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1Block creates a new instance of L1Block, bound to a specific deployed contract.
func NewL1Block(address common.Address, backend bind.ContractBackend) (*L1Block, error) {
	contract, err := bindL1Block(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1Block{L1BlockCaller: L1BlockCaller{contract: contract}, L1BlockTransactor: L1BlockTransactor{contract: contract}, L1BlockFilterer: L1BlockFilterer{contract: contract}}, nil
}

// NewL1BlockCaller creates a new read-only instance of L1Block, bound to a specific deployed contract.
func NewL1BlockCaller(address common.Address, caller bind.ContractCaller) (*L1BlockCaller, error) {
	contract, err := bindL1Block(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlockCaller{contract: contract}, nil
}

// NewL1BlockTransactor creates a new write-only instance of L1Block, bound to a specific deployed contract.
func NewL1BlockTransactor(address common.Address, transactor bind.ContractTransactor) (*L1BlockTransactor, error) {
	contract, err := bindL1Block(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlockTransactor{contract: contract}, nil
}

// NewL1BlockFilterer creates a new log filterer instance of L1Block, bound to a specific deployed contract.
func NewL1BlockFilterer(address common.Address, filterer bind.ContractFilterer) (*L1BlockFilterer, error) {
	contract, err := bindL1Block(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1BlockFilterer{contract: contract}, nil
}

// bindL1Block binds a generic wrapper to an already deployed contract.
func bindL1Block(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1BlockABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1Block *L1BlockRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1Block.Contract.L1BlockCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1Block *L1BlockRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1Block.Contract.L1BlockTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1Block *L1BlockRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1Block.Contract.L1BlockTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1Block *L1BlockCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1Block.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1Block *L1BlockTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1Block.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1Block *L1BlockTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1Block.Contract.contract.Transact(opts, method, params...)
}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_L1Block *L1BlockCaller) DEPOSITORACCOUNT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "DEPOSITOR_ACCOUNT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_L1Block *L1BlockSession) DEPOSITORACCOUNT() (common.Address, error) {
	return _L1Block.Contract.DEPOSITORACCOUNT(&_L1Block.CallOpts)
}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_L1Block *L1BlockCallerSession) DEPOSITORACCOUNT() (common.Address, error) {
	return _L1Block.Contract.DEPOSITORACCOUNT(&_L1Block.CallOpts)
}

// BaseFeeScalar is a free data retrieval call binding the contract method 0xc5985918.
//
// Solidity: function baseFeeScalar() view returns(uint32)
func (_L1Block *L1BlockCaller) BaseFeeScalar(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "baseFeeScalar")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// BaseFeeScalar is a free data retrieval call binding the contract method 0xc5985918.
//
// Solidity: function baseFeeScalar() view returns(uint32)
func (_L1Block *L1BlockSession) BaseFeeScalar() (uint32, error) {
	return _L1Block.Contract.BaseFeeScalar(&_L1Block.CallOpts)
}

// BaseFeeScalar is a free data retrieval call binding the contract method 0xc5985918.
//
// Solidity: function baseFeeScalar() view returns(uint32)
func (_L1Block *L1BlockCallerSession) BaseFeeScalar() (uint32, error) {
	return _L1Block.Contract.BaseFeeScalar(&_L1Block.CallOpts)
}

// Basefee is a free data retrieval call binding the contract method 0x5cf24969.
//
// Solidity: function basefee() view returns(uint256)
func (_L1Block *L1BlockCaller) Basefee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "basefee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Basefee is a free data retrieval call binding the contract method 0x5cf24969.
//
// Solidity: function basefee() view returns(uint256)
func (_L1Block *L1BlockSession) Basefee() (*big.Int, error) {
	return _L1Block.Contract.Basefee(&_L1Block.CallOpts)
}

// Basefee is a free data retrieval call binding the contract method 0x5cf24969.
//
// Solidity: function basefee() view returns(uint256)
func (_L1Block *L1BlockCallerSession) Basefee() (*big.Int, error) {
	return _L1Block.Contract.Basefee(&_L1Block.CallOpts)
}

// BatcherHash is a free data retrieval call binding the contract method 0xe81b2c6d.
//
// Solidity: function batcherHash() view returns(bytes32)
func (_L1Block *L1BlockCaller) BatcherHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "batcherHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// BatcherHash is a free data retrieval call binding the contract method 0xe81b2c6d.
//
// Solidity: function batcherHash() view returns(bytes32)
func (_L1Block *L1BlockSession) BatcherHash() ([32]byte, error) {
	return _L1Block.Contract.BatcherHash(&_L1Block.CallOpts)
}

// BatcherHash is a free data retrieval call binding the contract method 0xe81b2c6d.
//
// Solidity: function batcherHash() view returns(bytes32)
func (_L1Block *L1BlockCallerSession) BatcherHash() ([32]byte, error) {
	return _L1Block.Contract.BatcherHash(&_L1Block.CallOpts)
}

// BlobBaseFee is a free data retrieval call binding the contract method 0xf8206140.
//
// Solidity: function blobBaseFee() view returns(uint256)
func (_L1Block *L1BlockCaller) BlobBaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "blobBaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BlobBaseFee is a free data retrieval call binding the contract method 0xf8206140.
//
// Solidity: function blobBaseFee() view returns(uint256)
func (_L1Block *L1BlockSession) BlobBaseFee() (*big.Int, error) {
	return _L1Block.Contract.BlobBaseFee(&_L1Block.CallOpts)
}

// BlobBaseFee is a free data retrieval call binding the contract method 0xf8206140.
//
// Solidity: function blobBaseFee() view returns(uint256)
func (_L1Block *L1BlockCallerSession) BlobBaseFee() (*big.Int, error) {
	return _L1Block.Contract.BlobBaseFee(&_L1Block.CallOpts)
}

// BlobBaseFeeScalar is a free data retrieval call binding the contract method 0x68d5dca6.
//
// Solidity: function blobBaseFeeScalar() view returns(uint32)
func (_L1Block *L1BlockCaller) BlobBaseFeeScalar(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "blobBaseFeeScalar")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// BlobBaseFeeScalar is a free data retrieval call binding the contract method 0x68d5dca6.
//
// Solidity: function blobBaseFeeScalar() view returns(uint32)
func (_L1Block *L1BlockSession) BlobBaseFeeScalar() (uint32, error) {
	return _L1Block.Contract.BlobBaseFeeScalar(&_L1Block.CallOpts)
}

// BlobBaseFeeScalar is a free data retrieval call binding the contract method 0x68d5dca6.
//
// Solidity: function blobBaseFeeScalar() view returns(uint32)
func (_L1Block *L1BlockCallerSession) BlobBaseFeeScalar() (uint32, error) {
	return _L1Block.Contract.BlobBaseFeeScalar(&_L1Block.CallOpts)
}

// GasPayingToken is a free data retrieval call binding the contract method 0x4397dfef.
//
// Solidity: function gasPayingToken() view returns(address addr_, uint8 decimals_)
func (_L1Block *L1BlockCaller) GasPayingToken(opts *bind.CallOpts) (struct {
	Addr     common.Address
	Decimals uint8
}, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "gasPayingToken")

	outstruct := new(struct {
		Addr     common.Address
		Decimals uint8
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Addr = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Decimals = *abi.ConvertType(out[1], new(uint8)).(*uint8)

	return *outstruct, err

}

// GasPayingToken is a free data retrieval call binding the contract method 0x4397dfef.
//
// Solidity: function gasPayingToken() view returns(address addr_, uint8 decimals_)
func (_L1Block *L1BlockSession) GasPayingToken() (struct {
	Addr     common.Address
	Decimals uint8
}, error) {
	return _L1Block.Contract.GasPayingToken(&_L1Block.CallOpts)
}

// GasPayingToken is a free data retrieval call binding the contract method 0x4397dfef.
//
// Solidity: function gasPayingToken() view returns(address addr_, uint8 decimals_)
func (_L1Block *L1BlockCallerSession) GasPayingToken() (struct {
	Addr     common.Address
	Decimals uint8
}, error) {
	return _L1Block.Contract.GasPayingToken(&_L1Block.CallOpts)
}

// GasPayingTokenName is a free data retrieval call binding the contract method 0xd8444715.
//
// Solidity: function gasPayingTokenName() view returns(string name_)
func (_L1Block *L1BlockCaller) GasPayingTokenName(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "gasPayingTokenName")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// GasPayingTokenName is a free data retrieval call binding the contract method 0xd8444715.
//
// Solidity: function gasPayingTokenName() view returns(string name_)
func (_L1Block *L1BlockSession) GasPayingTokenName() (string, error) {
	return _L1Block.Contract.GasPayingTokenName(&_L1Block.CallOpts)
}

// GasPayingTokenName is a free data retrieval call binding the contract method 0xd8444715.
//
// Solidity: function gasPayingTokenName() view returns(string name_)
func (_L1Block *L1BlockCallerSession) GasPayingTokenName() (string, error) {
	return _L1Block.Contract.GasPayingTokenName(&_L1Block.CallOpts)
}

// GasPayingTokenSymbol is a free data retrieval call binding the contract method 0x550fcdc9.
//
// Solidity: function gasPayingTokenSymbol() view returns(string symbol_)
func (_L1Block *L1BlockCaller) GasPayingTokenSymbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "gasPayingTokenSymbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// GasPayingTokenSymbol is a free data retrieval call binding the contract method 0x550fcdc9.
//
// Solidity: function gasPayingTokenSymbol() view returns(string symbol_)
func (_L1Block *L1BlockSession) GasPayingTokenSymbol() (string, error) {
	return _L1Block.Contract.GasPayingTokenSymbol(&_L1Block.CallOpts)
}

// GasPayingTokenSymbol is a free data retrieval call binding the contract method 0x550fcdc9.
//
// Solidity: function gasPayingTokenSymbol() view returns(string symbol_)
func (_L1Block *L1BlockCallerSession) GasPayingTokenSymbol() (string, error) {
	return _L1Block.Contract.GasPayingTokenSymbol(&_L1Block.CallOpts)
}

// Hash is a free data retrieval call binding the contract method 0x09bd5a60.
//
// Solidity: function hash() view returns(bytes32)
func (_L1Block *L1BlockCaller) Hash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "hash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Hash is a free data retrieval call binding the contract method 0x09bd5a60.
//
// Solidity: function hash() view returns(bytes32)
func (_L1Block *L1BlockSession) Hash() ([32]byte, error) {
	return _L1Block.Contract.Hash(&_L1Block.CallOpts)
}

// Hash is a free data retrieval call binding the contract method 0x09bd5a60.
//
// Solidity: function hash() view returns(bytes32)
func (_L1Block *L1BlockCallerSession) Hash() ([32]byte, error) {
	return _L1Block.Contract.Hash(&_L1Block.CallOpts)
}

// IsCustomGasToken is a free data retrieval call binding the contract method 0x21326849.
//
// Solidity: function isCustomGasToken() view returns(bool)
func (_L1Block *L1BlockCaller) IsCustomGasToken(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "isCustomGasToken")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsCustomGasToken is a free data retrieval call binding the contract method 0x21326849.
//
// Solidity: function isCustomGasToken() view returns(bool)
func (_L1Block *L1BlockSession) IsCustomGasToken() (bool, error) {
	return _L1Block.Contract.IsCustomGasToken(&_L1Block.CallOpts)
}

// IsCustomGasToken is a free data retrieval call binding the contract method 0x21326849.
//
// Solidity: function isCustomGasToken() view returns(bool)
func (_L1Block *L1BlockCallerSession) IsCustomGasToken() (bool, error) {
	return _L1Block.Contract.IsCustomGasToken(&_L1Block.CallOpts)
}

// L1FeeOverhead is a free data retrieval call binding the contract method 0x8b239f73.
//
// Solidity: function l1FeeOverhead() view returns(uint256)
func (_L1Block *L1BlockCaller) L1FeeOverhead(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "l1FeeOverhead")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L1FeeOverhead is a free data retrieval call binding the contract method 0x8b239f73.
//
// Solidity: function l1FeeOverhead() view returns(uint256)
func (_L1Block *L1BlockSession) L1FeeOverhead() (*big.Int, error) {
	return _L1Block.Contract.L1FeeOverhead(&_L1Block.CallOpts)
}

// L1FeeOverhead is a free data retrieval call binding the contract method 0x8b239f73.
//
// Solidity: function l1FeeOverhead() view returns(uint256)
func (_L1Block *L1BlockCallerSession) L1FeeOverhead() (*big.Int, error) {
	return _L1Block.Contract.L1FeeOverhead(&_L1Block.CallOpts)
}

// L1FeeScalar is a free data retrieval call binding the contract method 0x9e8c4966.
//
// Solidity: function l1FeeScalar() view returns(uint256)
func (_L1Block *L1BlockCaller) L1FeeScalar(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "l1FeeScalar")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L1FeeScalar is a free data retrieval call binding the contract method 0x9e8c4966.
//
// Solidity: function l1FeeScalar() view returns(uint256)
func (_L1Block *L1BlockSession) L1FeeScalar() (*big.Int, error) {
	return _L1Block.Contract.L1FeeScalar(&_L1Block.CallOpts)
}

// L1FeeScalar is a free data retrieval call binding the contract method 0x9e8c4966.
//
// Solidity: function l1FeeScalar() view returns(uint256)
func (_L1Block *L1BlockCallerSession) L1FeeScalar() (*big.Int, error) {
	return _L1Block.Contract.L1FeeScalar(&_L1Block.CallOpts)
}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint64)
func (_L1Block *L1BlockCaller) Number(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "number")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint64)
func (_L1Block *L1BlockSession) Number() (uint64, error) {
	return _L1Block.Contract.Number(&_L1Block.CallOpts)
}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint64)
func (_L1Block *L1BlockCallerSession) Number() (uint64, error) {
	return _L1Block.Contract.Number(&_L1Block.CallOpts)
}

// SequenceNumber is a free data retrieval call binding the contract method 0x64ca23ef.
//
// Solidity: function sequenceNumber() view returns(uint64)
func (_L1Block *L1BlockCaller) SequenceNumber(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "sequenceNumber")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// SequenceNumber is a free data retrieval call binding the contract method 0x64ca23ef.
//
// Solidity: function sequenceNumber() view returns(uint64)
func (_L1Block *L1BlockSession) SequenceNumber() (uint64, error) {
	return _L1Block.Contract.SequenceNumber(&_L1Block.CallOpts)
}

// SequenceNumber is a free data retrieval call binding the contract method 0x64ca23ef.
//
// Solidity: function sequenceNumber() view returns(uint64)
func (_L1Block *L1BlockCallerSession) SequenceNumber() (uint64, error) {
	return _L1Block.Contract.SequenceNumber(&_L1Block.CallOpts)
}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint64)
func (_L1Block *L1BlockCaller) Timestamp(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "timestamp")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint64)
func (_L1Block *L1BlockSession) Timestamp() (uint64, error) {
	return _L1Block.Contract.Timestamp(&_L1Block.CallOpts)
}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint64)
func (_L1Block *L1BlockCallerSession) Timestamp() (uint64, error) {
	return _L1Block.Contract.Timestamp(&_L1Block.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1Block *L1BlockCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1Block *L1BlockSession) Version() (string, error) {
	return _L1Block.Contract.Version(&_L1Block.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1Block *L1BlockCallerSession) Version() (string, error) {
	return _L1Block.Contract.Version(&_L1Block.CallOpts)
}

// SetGasPayingToken is a paid mutator transaction binding the contract method 0x71cfaa3f.
//
// Solidity: function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) returns()
func (_L1Block *L1BlockTransactor) SetGasPayingToken(opts *bind.TransactOpts, _token common.Address, _decimals uint8, _name [32]byte, _symbol [32]byte) (*types.Transaction, error) {
	return _L1Block.contract.Transact(opts, "setGasPayingToken", _token, _decimals, _name, _symbol)
}

// SetGasPayingToken is a paid mutator transaction binding the contract method 0x71cfaa3f.
//
// Solidity: function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) returns()
func (_L1Block *L1BlockSession) SetGasPayingToken(_token common.Address, _decimals uint8, _name [32]byte, _symbol [32]byte) (*types.Transaction, error) {
	return _L1Block.Contract.SetGasPayingToken(&_L1Block.TransactOpts, _token, _decimals, _name, _symbol)
}

// SetGasPayingToken is a paid mutator transaction binding the contract method 0x71cfaa3f.
//
// Solidity: function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) returns()
func (_L1Block *L1BlockTransactorSession) SetGasPayingToken(_token common.Address, _decimals uint8, _name [32]byte, _symbol [32]byte) (*types.Transaction, error) {
	return _L1Block.Contract.SetGasPayingToken(&_L1Block.TransactOpts, _token, _decimals, _name, _symbol)
}

// SetL1BlockValues is a paid mutator transaction binding the contract method 0x015d8eb9.
//
// Solidity: function setL1BlockValues(uint64 _number, uint64 _timestamp, uint256 _basefee, bytes32 _hash, uint64 _sequenceNumber, bytes32 _batcherHash, uint256 _l1FeeOverhead, uint256 _l1FeeScalar) returns()
func (_L1Block *L1BlockTransactor) SetL1BlockValues(opts *bind.TransactOpts, _number uint64, _timestamp uint64, _basefee *big.Int, _hash [32]byte, _sequenceNumber uint64, _batcherHash [32]byte, _l1FeeOverhead *big.Int, _l1FeeScalar *big.Int) (*types.Transaction, error) {
	return _L1Block.contract.Transact(opts, "setL1BlockValues", _number, _timestamp, _basefee, _hash, _sequenceNumber, _batcherHash, _l1FeeOverhead, _l1FeeScalar)
}

// SetL1BlockValues is a paid mutator transaction binding the contract method 0x015d8eb9.
//
// Solidity: function setL1BlockValues(uint64 _number, uint64 _timestamp, uint256 _basefee, bytes32 _hash, uint64 _sequenceNumber, bytes32 _batcherHash, uint256 _l1FeeOverhead, uint256 _l1FeeScalar) returns()
func (_L1Block *L1BlockSession) SetL1BlockValues(_number uint64, _timestamp uint64, _basefee *big.Int, _hash [32]byte, _sequenceNumber uint64, _batcherHash [32]byte, _l1FeeOverhead *big.Int, _l1FeeScalar *big.Int) (*types.Transaction, error) {
	return _L1Block.Contract.SetL1BlockValues(&_L1Block.TransactOpts, _number, _timestamp, _basefee, _hash, _sequenceNumber, _batcherHash, _l1FeeOverhead, _l1FeeScalar)
}

// SetL1BlockValues is a paid mutator transaction binding the contract method 0x015d8eb9.
//
// Solidity: function setL1BlockValues(uint64 _number, uint64 _timestamp, uint256 _basefee, bytes32 _hash, uint64 _sequenceNumber, bytes32 _batcherHash, uint256 _l1FeeOverhead, uint256 _l1FeeScalar) returns()
func (_L1Block *L1BlockTransactorSession) SetL1BlockValues(_number uint64, _timestamp uint64, _basefee *big.Int, _hash [32]byte, _sequenceNumber uint64, _batcherHash [32]byte, _l1FeeOverhead *big.Int, _l1FeeScalar *big.Int) (*types.Transaction, error) {
	return _L1Block.Contract.SetL1BlockValues(&_L1Block.TransactOpts, _number, _timestamp, _basefee, _hash, _sequenceNumber, _batcherHash, _l1FeeOverhead, _l1FeeScalar)
}

// SetL1BlockValuesEcotone is a paid mutator transaction binding the contract method 0x440a5e20.
//
// Solidity: function setL1BlockValuesEcotone() returns()
func (_L1Block *L1BlockTransactor) SetL1BlockValuesEcotone(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1Block.contract.Transact(opts, "setL1BlockValuesEcotone")
}

// SetL1BlockValuesEcotone is a paid mutator transaction binding the contract method 0x440a5e20.
//
// Solidity: function setL1BlockValuesEcotone() returns()
func (_L1Block *L1BlockSession) SetL1BlockValuesEcotone() (*types.Transaction, error) {
	return _L1Block.Contract.SetL1BlockValuesEcotone(&_L1Block.TransactOpts)
}

// SetL1BlockValuesEcotone is a paid mutator transaction binding the contract method 0x440a5e20.
//
// Solidity: function setL1BlockValuesEcotone() returns()
func (_L1Block *L1BlockTransactorSession) SetL1BlockValuesEcotone() (*types.Transaction, error) {
	return _L1Block.Contract.SetL1BlockValuesEcotone(&_L1Block.TransactOpts)
}

// L1BlockGasPayingTokenSetIterator is returned from FilterGasPayingTokenSet and is used to iterate over the raw logs and unpacked data for GasPayingTokenSet events raised by the L1Block contract.
type L1BlockGasPayingTokenSetIterator struct {
	Event *L1BlockGasPayingTokenSet // Event containing the contract specifics and raw log

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
func (it *L1BlockGasPayingTokenSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BlockGasPayingTokenSet)
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
		it.Event = new(L1BlockGasPayingTokenSet)
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
func (it *L1BlockGasPayingTokenSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BlockGasPayingTokenSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BlockGasPayingTokenSet represents a GasPayingTokenSet event raised by the L1Block contract.
type L1BlockGasPayingTokenSet struct {
	Token    common.Address
	Decimals uint8
	Name     [32]byte
	Symbol   [32]byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterGasPayingTokenSet is a free log retrieval operation binding the contract event 0x10e43c4d58f3ef4edae7c1ca2e7f02d46b2cadbcc046737038527ed8486ffeb0.
//
// Solidity: event GasPayingTokenSet(address indexed token, uint8 indexed decimals, bytes32 name, bytes32 symbol)
func (_L1Block *L1BlockFilterer) FilterGasPayingTokenSet(opts *bind.FilterOpts, token []common.Address, decimals []uint8) (*L1BlockGasPayingTokenSetIterator, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var decimalsRule []interface{}
	for _, decimalsItem := range decimals {
		decimalsRule = append(decimalsRule, decimalsItem)
	}

	logs, sub, err := _L1Block.contract.FilterLogs(opts, "GasPayingTokenSet", tokenRule, decimalsRule)
	if err != nil {
		return nil, err
	}
	return &L1BlockGasPayingTokenSetIterator{contract: _L1Block.contract, event: "GasPayingTokenSet", logs: logs, sub: sub}, nil
}

// WatchGasPayingTokenSet is a free log subscription operation binding the contract event 0x10e43c4d58f3ef4edae7c1ca2e7f02d46b2cadbcc046737038527ed8486ffeb0.
//
// Solidity: event GasPayingTokenSet(address indexed token, uint8 indexed decimals, bytes32 name, bytes32 symbol)
func (_L1Block *L1BlockFilterer) WatchGasPayingTokenSet(opts *bind.WatchOpts, sink chan<- *L1BlockGasPayingTokenSet, token []common.Address, decimals []uint8) (event.Subscription, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var decimalsRule []interface{}
	for _, decimalsItem := range decimals {
		decimalsRule = append(decimalsRule, decimalsItem)
	}

	logs, sub, err := _L1Block.contract.WatchLogs(opts, "GasPayingTokenSet", tokenRule, decimalsRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BlockGasPayingTokenSet)
				if err := _L1Block.contract.UnpackLog(event, "GasPayingTokenSet", log); err != nil {
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

// ParseGasPayingTokenSet is a log parse operation binding the contract event 0x10e43c4d58f3ef4edae7c1ca2e7f02d46b2cadbcc046737038527ed8486ffeb0.
//
// Solidity: event GasPayingTokenSet(address indexed token, uint8 indexed decimals, bytes32 name, bytes32 symbol)
func (_L1Block *L1BlockFilterer) ParseGasPayingTokenSet(log types.Log) (*L1BlockGasPayingTokenSet, error) {
	event := new(L1BlockGasPayingTokenSet)
	if err := _L1Block.contract.UnpackLog(event, "GasPayingTokenSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
