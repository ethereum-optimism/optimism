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

// L2OutputOracleOutputProposal is an auto generated low-level Go binding around an user-defined struct.
type L2OutputOracleOutputProposal struct {
	OutputRoot [32]byte
	Timestamp  *big.Int
}

// L2OutputOracleMetaData contains all meta data concerning the L2OutputOracle contract.
var L2OutputOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_submissionInterval\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_genesisL2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_historicalTotalBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_startingBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_sequencer\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l1Timestamp\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"l2OutputAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l1Timestamp\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"l2OutputDeleted\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"HISTORICAL_TOTAL_BLOCKS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"STARTING_BLOCK_NUMBER\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SUBMISSION_INTERVAL\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_l1Blockhash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l1BlockNumber\",\"type\":\"uint256\"}],\"name\":\"appendL2Output\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"internalType\":\"structL2OutputOracle.OutputProposal\",\"name\":\"_proposal\",\"type\":\"tuple\"}],\"name\":\"deleteL2Output\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"getL2Output\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"internalType\":\"structL2OutputOracle.OutputProposal\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60e060405234801561001057600080fd5b50604051610d7c380380610d7c83398101604081905261002f916100dc565b6100383361008c565b608085905260a083905260c082905260408051808201825285815242602080830191825260008681526002909152929092209051815590516001918201558290556100828161008c565b5050505050610137565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b600080600080600060a086880312156100f457600080fd5b855160208701516040880151606089015160808a0151939850919650945092506001600160a01b038116811461012957600080fd5b809150509295509295909350565b60805160a05160c051610c086101746000396000610131015260006102570152600081816101650152818161052401526108c90152610c086000f3fe6080604052600436106100bc5760003560e01c8063715018a611610074578063a4771aad1161004e578063a4771aad14610245578063dcec334814610279578063f2fde38b1461028e57600080fd5b8063715018a6146101875780638da5cb5b1461019c578063a25ae557146101d157600080fd5b80634599c788116100a55780634599c788146100f65780634ab65d731461011f578063529933df1461015357600080fd5b8063093b3d90146100c157806325188104146100e3575b600080fd5b3480156100cd57600080fd5b506100e16100dc366004610a9f565b6102ae565b005b6100e16100f1366004610b15565b610550565b34801561010257600080fd5b5061010c60015481565b6040519081526020015b60405180910390f35b34801561012b57600080fd5b5061010c7f000000000000000000000000000000000000000000000000000000000000000081565b34801561015f57600080fd5b5061010c7f000000000000000000000000000000000000000000000000000000000000000081565b34801561019357600080fd5b506100e1610838565b3480156101a857600080fd5b5060005460405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610116565b3480156101dd57600080fd5b5061022a6101ec366004610b47565b604080518082019091526000808252602082015250600090815260026020908152604091829020825180840190935280548352600101549082015290565b60408051825181526020928301519281019290925201610116565b34801561025157600080fd5b5061010c7f000000000000000000000000000000000000000000000000000000000000000081565b34801561028557600080fd5b5061010c6108c5565b34801561029a57600080fd5b506100e16102a9366004610b60565b6108fa565b60005473ffffffffffffffffffffffffffffffffffffffff163314610334576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064015b60405180910390fd5b6001805460009081526002602090815260409182902082518084019093528054808452930154908201528251909114610415576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605260248201527f4f75747075744f7261636c653a20546865206f757470757420726f6f7420746f60448201527f2064656c65746520646f6573206e6f74206d6174636820746865206c6174657360648201527f74206f75747075742070726f706f73616c2e0000000000000000000000000000608482015260a40161032b565b80602001518260200151146104d2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605060248201527f4f75747075744f7261636c653a205468652074696d657374616d7020746f206460448201527f656c65746520646f6573206e6f74206d6174636820746865206c61746573742060648201527f6f75747075742070726f706f73616c2e00000000000000000000000000000000608482015260a40161032b565b600154602082015182516040517f6897e92e2fea3b89bf0d45ed867487716da71bfa624c6878569d227d736c700990600090a460018054600090815260026020526040812081815582015554610549907f000000000000000000000000000000000000000000000000000000000000000090610bcc565b6001555050565b60005473ffffffffffffffffffffffffffffffffffffffff1633146105d1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161032b565b6105d96108c5565b831461068d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604760248201527f4f75747075744f7261636c653a20426c6f636b206e756d626572206d7573742060448201527f626520657175616c20746f206e65787420657870656374656420626c6f636b2060648201527f6e756d6265722e00000000000000000000000000000000000000000000000000608482015260a40161032b565b8361071a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f4f75747075744f7261636c653a2043616e6e6f74207375626d697420656d707460448201527f79204c32206f75747075742e0000000000000000000000000000000000000000606482015260840161032b565b81156107d657818140146107d6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604760248201527f4f75747075744f7261636c653a20426c6f636b6861736820646f6573206e6f7460448201527f206d61746368207468652068617368206174207468652065787065637465642060648201527f6865696768742e00000000000000000000000000000000000000000000000000608482015260a40161032b565b60408051808201825285815242602080830182815260008881526002909252848220935184555160019384015591869055915185929187917f54fbf9b58db0b6543a3a0cb9fbd4e98a7c7b88878978fa125b84a2c7f51d34ba9190a450505050565b60005473ffffffffffffffffffffffffffffffffffffffff1633146108b9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161032b565b6108c36000610a2a565b565b60007f00000000000000000000000000000000000000000000000000000000000000006001546108f59190610be3565b905090565b60005473ffffffffffffffffffffffffffffffffffffffff16331461097b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161032b565b73ffffffffffffffffffffffffffffffffffffffff8116610a1e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015260840161032b565b610a2781610a2a565b50565b6000805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b600060408284031215610ab157600080fd5b6040516040810181811067ffffffffffffffff82111715610afb577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604052823581526020928301359281019290925250919050565b60008060008060808587031215610b2b57600080fd5b5050823594602084013594506040840135936060013592509050565b600060208284031215610b5957600080fd5b5035919050565b600060208284031215610b7257600080fd5b813573ffffffffffffffffffffffffffffffffffffffff81168114610b9657600080fd5b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015610bde57610bde610b9d565b500390565b60008219821115610bf657610bf6610b9d565b50019056fea164736f6c634300080a000a",
}

// L2OutputOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use L2OutputOracleMetaData.ABI instead.
var L2OutputOracleABI = L2OutputOracleMetaData.ABI

// L2OutputOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2OutputOracleMetaData.Bin instead.
var L2OutputOracleBin = L2OutputOracleMetaData.Bin

// DeployL2OutputOracle deploys a new Ethereum contract, binding an instance of L2OutputOracle to it.
func DeployL2OutputOracle(auth *bind.TransactOpts, backend bind.ContractBackend, _submissionInterval *big.Int, _genesisL2Output [32]byte, _historicalTotalBlocks *big.Int, _startingBlockNumber *big.Int, _sequencer common.Address) (common.Address, *types.Transaction, *L2OutputOracle, error) {
	parsed, err := L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2OutputOracleBin), backend, _submissionInterval, _genesisL2Output, _historicalTotalBlocks, _startingBlockNumber, _sequencer)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2OutputOracle{L2OutputOracleCaller: L2OutputOracleCaller{contract: contract}, L2OutputOracleTransactor: L2OutputOracleTransactor{contract: contract}, L2OutputOracleFilterer: L2OutputOracleFilterer{contract: contract}}, nil
}

// L2OutputOracle is an auto generated Go binding around an Ethereum contract.
type L2OutputOracle struct {
	L2OutputOracleCaller     // Read-only binding to the contract
	L2OutputOracleTransactor // Write-only binding to the contract
	L2OutputOracleFilterer   // Log filterer for contract events
}

// L2OutputOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2OutputOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2OutputOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2OutputOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2OutputOracleSession struct {
	Contract     *L2OutputOracle   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L2OutputOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2OutputOracleCallerSession struct {
	Contract *L2OutputOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// L2OutputOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2OutputOracleTransactorSession struct {
	Contract     *L2OutputOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// L2OutputOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2OutputOracleRaw struct {
	Contract *L2OutputOracle // Generic contract binding to access the raw methods on
}

// L2OutputOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2OutputOracleCallerRaw struct {
	Contract *L2OutputOracleCaller // Generic read-only contract binding to access the raw methods on
}

// L2OutputOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2OutputOracleTransactorRaw struct {
	Contract *L2OutputOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2OutputOracle creates a new instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracle(address common.Address, backend bind.ContractBackend) (*L2OutputOracle, error) {
	contract, err := bindL2OutputOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracle{L2OutputOracleCaller: L2OutputOracleCaller{contract: contract}, L2OutputOracleTransactor: L2OutputOracleTransactor{contract: contract}, L2OutputOracleFilterer: L2OutputOracleFilterer{contract: contract}}, nil
}

// NewL2OutputOracleCaller creates a new read-only instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleCaller(address common.Address, caller bind.ContractCaller) (*L2OutputOracleCaller, error) {
	contract, err := bindL2OutputOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleCaller{contract: contract}, nil
}

// NewL2OutputOracleTransactor creates a new write-only instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*L2OutputOracleTransactor, error) {
	contract, err := bindL2OutputOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleTransactor{contract: contract}, nil
}

// NewL2OutputOracleFilterer creates a new log filterer instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*L2OutputOracleFilterer, error) {
	contract, err := bindL2OutputOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleFilterer{contract: contract}, nil
}

// bindL2OutputOracle binds a generic wrapper to an already deployed contract.
func bindL2OutputOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2OutputOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2OutputOracle *L2OutputOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2OutputOracle.Contract.L2OutputOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2OutputOracle *L2OutputOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.L2OutputOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2OutputOracle *L2OutputOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.L2OutputOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2OutputOracle *L2OutputOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2OutputOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2OutputOracle *L2OutputOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2OutputOracle *L2OutputOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.contract.Transact(opts, method, params...)
}

// HISTORICALTOTALBLOCKS is a free data retrieval call binding the contract method 0xa4771aad.
//
// Solidity: function HISTORICAL_TOTAL_BLOCKS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) HISTORICALTOTALBLOCKS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "HISTORICAL_TOTAL_BLOCKS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HISTORICALTOTALBLOCKS is a free data retrieval call binding the contract method 0xa4771aad.
//
// Solidity: function HISTORICAL_TOTAL_BLOCKS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) HISTORICALTOTALBLOCKS() (*big.Int, error) {
	return _L2OutputOracle.Contract.HISTORICALTOTALBLOCKS(&_L2OutputOracle.CallOpts)
}

// HISTORICALTOTALBLOCKS is a free data retrieval call binding the contract method 0xa4771aad.
//
// Solidity: function HISTORICAL_TOTAL_BLOCKS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) HISTORICALTOTALBLOCKS() (*big.Int, error) {
	return _L2OutputOracle.Contract.HISTORICALTOTALBLOCKS(&_L2OutputOracle.CallOpts)
}

// STARTINGBLOCKNUMBER is a free data retrieval call binding the contract method 0x4ab65d73.
//
// Solidity: function STARTING_BLOCK_NUMBER() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) STARTINGBLOCKNUMBER(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "STARTING_BLOCK_NUMBER")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// STARTINGBLOCKNUMBER is a free data retrieval call binding the contract method 0x4ab65d73.
//
// Solidity: function STARTING_BLOCK_NUMBER() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) STARTINGBLOCKNUMBER() (*big.Int, error) {
	return _L2OutputOracle.Contract.STARTINGBLOCKNUMBER(&_L2OutputOracle.CallOpts)
}

// STARTINGBLOCKNUMBER is a free data retrieval call binding the contract method 0x4ab65d73.
//
// Solidity: function STARTING_BLOCK_NUMBER() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) STARTINGBLOCKNUMBER() (*big.Int, error) {
	return _L2OutputOracle.Contract.STARTINGBLOCKNUMBER(&_L2OutputOracle.CallOpts)
}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) SUBMISSIONINTERVAL(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "SUBMISSION_INTERVAL")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) SUBMISSIONINTERVAL() (*big.Int, error) {
	return _L2OutputOracle.Contract.SUBMISSIONINTERVAL(&_L2OutputOracle.CallOpts)
}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) SUBMISSIONINTERVAL() (*big.Int, error) {
	return _L2OutputOracle.Contract.SUBMISSIONINTERVAL(&_L2OutputOracle.CallOpts)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2BlockNumber) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleCaller) GetL2Output(opts *bind.CallOpts, _l2BlockNumber *big.Int) (L2OutputOracleOutputProposal, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "getL2Output", _l2BlockNumber)

	if err != nil {
		return *new(L2OutputOracleOutputProposal), err
	}

	out0 := *abi.ConvertType(out[0], new(L2OutputOracleOutputProposal)).(*L2OutputOracleOutputProposal)

	return out0, err

}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2BlockNumber) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleSession) GetL2Output(_l2BlockNumber *big.Int) (L2OutputOracleOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2BlockNumber) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleCallerSession) GetL2Output(_l2BlockNumber *big.Int) (L2OutputOracleOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) LatestBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "latestBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) LatestBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestBlockNumber(&_L2OutputOracle.CallOpts)
}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) LatestBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestBlockNumber(&_L2OutputOracle.CallOpts)
}

// NextBlockNumber is a free data retrieval call binding the contract method 0xdcec3348.
//
// Solidity: function nextBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) NextBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "nextBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextBlockNumber is a free data retrieval call binding the contract method 0xdcec3348.
//
// Solidity: function nextBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) NextBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextBlockNumber(&_L2OutputOracle.CallOpts)
}

// NextBlockNumber is a free data retrieval call binding the contract method 0xdcec3348.
//
// Solidity: function nextBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) NextBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextBlockNumber(&_L2OutputOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleSession) Owner() (common.Address, error) {
	return _L2OutputOracle.Contract.Owner(&_L2OutputOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleCallerSession) Owner() (common.Address, error) {
	return _L2OutputOracle.Contract.Owner(&_L2OutputOracle.CallOpts)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2BlockNumber, bytes32 _l1Blockhash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactor) AppendL2Output(opts *bind.TransactOpts, _l2Output [32]byte, _l2BlockNumber *big.Int, _l1Blockhash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "appendL2Output", _l2Output, _l2BlockNumber, _l1Blockhash, _l1BlockNumber)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2BlockNumber, bytes32 _l1Blockhash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleSession) AppendL2Output(_l2Output [32]byte, _l2BlockNumber *big.Int, _l1Blockhash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.AppendL2Output(&_L2OutputOracle.TransactOpts, _l2Output, _l2BlockNumber, _l1Blockhash, _l1BlockNumber)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2BlockNumber, bytes32 _l1Blockhash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) AppendL2Output(_l2Output [32]byte, _l2BlockNumber *big.Int, _l1Blockhash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.AppendL2Output(&_L2OutputOracle.TransactOpts, _l2Output, _l2BlockNumber, _l1Blockhash, _l1BlockNumber)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) DeleteL2Output(opts *bind.TransactOpts, _proposal L2OutputOracleOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "deleteL2Output", _proposal)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleSession) DeleteL2Output(_proposal L2OutputOracleOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Output(&_L2OutputOracle.TransactOpts, _proposal)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) DeleteL2Output(_proposal L2OutputOracleOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Output(&_L2OutputOracle.TransactOpts, _proposal)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleSession) RenounceOwnership() (*types.Transaction, error) {
	return _L2OutputOracle.Contract.RenounceOwnership(&_L2OutputOracle.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _L2OutputOracle.Contract.RenounceOwnership(&_L2OutputOracle.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.TransferOwnership(&_L2OutputOracle.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.TransferOwnership(&_L2OutputOracle.TransactOpts, newOwner)
}

// L2OutputOracleOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the L2OutputOracle contract.
type L2OutputOracleOwnershipTransferredIterator struct {
	Event *L2OutputOracleOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleOwnershipTransferred)
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
		it.Event = new(L2OutputOracleOwnershipTransferred)
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
func (it *L2OutputOracleOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleOwnershipTransferred represents a OwnershipTransferred event raised by the L2OutputOracle contract.
type L2OutputOracleOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*L2OutputOracleOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleOwnershipTransferredIterator{contract: _L2OutputOracle.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *L2OutputOracleOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleOwnershipTransferred)
				if err := _L2OutputOracle.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseOwnershipTransferred(log types.Log) (*L2OutputOracleOwnershipTransferred, error) {
	event := new(L2OutputOracleOwnershipTransferred)
	if err := _L2OutputOracle.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleL2OutputAppendedIterator is returned from FilterL2OutputAppended and is used to iterate over the raw logs and unpacked data for L2OutputAppended events raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputAppendedIterator struct {
	Event *L2OutputOracleL2OutputAppended // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleL2OutputAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleL2OutputAppended)
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
		it.Event = new(L2OutputOracleL2OutputAppended)
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
func (it *L2OutputOracleL2OutputAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleL2OutputAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleL2OutputAppended represents a L2OutputAppended event raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputAppended struct {
	L2Output      [32]byte
	L1Timestamp   *big.Int
	L2BlockNumber *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterL2OutputAppended is a free log retrieval operation binding the contract event 0x54fbf9b58db0b6543a3a0cb9fbd4e98a7c7b88878978fa125b84a2c7f51d34ba.
//
// Solidity: event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterL2OutputAppended(opts *bind.FilterOpts, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2BlockNumber []*big.Int) (*L2OutputOracleL2OutputAppendedIterator, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2BlockNumberRule []interface{}
	for _, _l2BlockNumberItem := range _l2BlockNumber {
		_l2BlockNumberRule = append(_l2BlockNumberRule, _l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "l2OutputAppended", _l2OutputRule, _l1TimestampRule, _l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleL2OutputAppendedIterator{contract: _L2OutputOracle.contract, event: "l2OutputAppended", logs: logs, sub: sub}, nil
}

// WatchL2OutputAppended is a free log subscription operation binding the contract event 0x54fbf9b58db0b6543a3a0cb9fbd4e98a7c7b88878978fa125b84a2c7f51d34ba.
//
// Solidity: event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchL2OutputAppended(opts *bind.WatchOpts, sink chan<- *L2OutputOracleL2OutputAppended, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2BlockNumber []*big.Int) (event.Subscription, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2BlockNumberRule []interface{}
	for _, _l2BlockNumberItem := range _l2BlockNumber {
		_l2BlockNumberRule = append(_l2BlockNumberRule, _l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "l2OutputAppended", _l2OutputRule, _l1TimestampRule, _l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleL2OutputAppended)
				if err := _L2OutputOracle.contract.UnpackLog(event, "l2OutputAppended", log); err != nil {
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

// ParseL2OutputAppended is a log parse operation binding the contract event 0x54fbf9b58db0b6543a3a0cb9fbd4e98a7c7b88878978fa125b84a2c7f51d34ba.
//
// Solidity: event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseL2OutputAppended(log types.Log) (*L2OutputOracleL2OutputAppended, error) {
	event := new(L2OutputOracleL2OutputAppended)
	if err := _L2OutputOracle.contract.UnpackLog(event, "l2OutputAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleL2OutputDeletedIterator is returned from FilterL2OutputDeleted and is used to iterate over the raw logs and unpacked data for L2OutputDeleted events raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputDeletedIterator struct {
	Event *L2OutputOracleL2OutputDeleted // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleL2OutputDeletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleL2OutputDeleted)
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
		it.Event = new(L2OutputOracleL2OutputDeleted)
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
func (it *L2OutputOracleL2OutputDeletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleL2OutputDeletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleL2OutputDeleted represents a L2OutputDeleted event raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputDeleted struct {
	L2Output      [32]byte
	L1Timestamp   *big.Int
	L2BlockNumber *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterL2OutputDeleted is a free log retrieval operation binding the contract event 0x6897e92e2fea3b89bf0d45ed867487716da71bfa624c6878569d227d736c7009.
//
// Solidity: event l2OutputDeleted(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterL2OutputDeleted(opts *bind.FilterOpts, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2BlockNumber []*big.Int) (*L2OutputOracleL2OutputDeletedIterator, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2BlockNumberRule []interface{}
	for _, _l2BlockNumberItem := range _l2BlockNumber {
		_l2BlockNumberRule = append(_l2BlockNumberRule, _l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "l2OutputDeleted", _l2OutputRule, _l1TimestampRule, _l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleL2OutputDeletedIterator{contract: _L2OutputOracle.contract, event: "l2OutputDeleted", logs: logs, sub: sub}, nil
}

// WatchL2OutputDeleted is a free log subscription operation binding the contract event 0x6897e92e2fea3b89bf0d45ed867487716da71bfa624c6878569d227d736c7009.
//
// Solidity: event l2OutputDeleted(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchL2OutputDeleted(opts *bind.WatchOpts, sink chan<- *L2OutputOracleL2OutputDeleted, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2BlockNumber []*big.Int) (event.Subscription, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2BlockNumberRule []interface{}
	for _, _l2BlockNumberItem := range _l2BlockNumber {
		_l2BlockNumberRule = append(_l2BlockNumberRule, _l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "l2OutputDeleted", _l2OutputRule, _l1TimestampRule, _l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleL2OutputDeleted)
				if err := _L2OutputOracle.contract.UnpackLog(event, "l2OutputDeleted", log); err != nil {
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

// ParseL2OutputDeleted is a log parse operation binding the contract event 0x6897e92e2fea3b89bf0d45ed867487716da71bfa624c6878569d227d736c7009.
//
// Solidity: event l2OutputDeleted(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseL2OutputDeleted(log types.Log) (*L2OutputOracleL2OutputDeleted, error) {
	event := new(L2OutputOracleL2OutputDeleted)
	if err := _L2OutputOracle.contract.UnpackLog(event, "l2OutputDeleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
