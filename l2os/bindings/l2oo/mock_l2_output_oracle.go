// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package l2oo

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

// MockL2OutputOracleMetaData contains all meta data concerning the MockL2OutputOracle contract.
var MockL2OutputOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_submissionFrequency\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockTime\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_genesisL2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_historicalTotalBlocks\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_timestamp\",\"type\":\"uint256\"}],\"name\":\"appendL2Output\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_timestamp\",\"type\":\"uint256\"}],\"name\":\"computeL2BlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"historicalTotalBlocks\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2BlockTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"l2Outputs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startingBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"submissionFrequency\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5060405161098f38038061098f833981810160405281019061003291906100e7565b83600081905550826001819055508160026000428152602001908152602001600020819055508060038190555042600481905550426005819055505050505061014e565b600080fd5b6000819050919050565b61008e8161007b565b811461009957600080fd5b50565b6000815190506100ab81610085565b92915050565b6000819050919050565b6100c4816100b1565b81146100cf57600080fd5b50565b6000815190506100e1816100bb565b92915050565b6000806000806080858703121561010157610100610076565b5b600061010f8782880161009c565b94505060206101208782880161009c565b9350506040610131878288016100d2565b92505060606101428782880161009c565b91505092959194509250565b6108328061015d6000396000f3fe608060405234801561001057600080fd5b50600436106100935760003560e01c806393991af31161006657806393991af314610134578063b210dc2114610152578063b71d13e214610170578063c5095d681461018c578063c90ec2da146101aa57610093565b806302be8bfe1461009857806302e51345146100c85780630c1952d3146100f8578063357e951f14610116575b600080fd5b6100b260048036038101906100ad91906103b9565b6101c8565b6040516100bf91906103ff565b60405180910390f35b6100e260048036038101906100dd91906103b9565b6101e0565b6040516100ef9190610429565b60405180910390f35b610100610256565b60405161010d9190610429565b60405180910390f35b61011e61025c565b60405161012b9190610429565b60405180910390f35b61013c610272565b6040516101499190610429565b60405180910390f35b61015a610278565b6040516101679190610429565b60405180910390f35b61018a60048036038101906101859190610470565b61027e565b005b610194610372565b6040516101a19190610429565b60405180910390f35b6101b2610378565b6040516101bf9190610429565b60405180910390f35b60026020528060005260406000206000915090505481565b6000600554821015610227576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161021e90610533565b60405180910390fd5b600154600554836102389190610582565b61024291906105e5565b60035461024f9190610616565b9050919050565b60045481565b6000805460045461026d9190610616565b905090565b60015481565b60005481565b8042116102c0576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102b7906106de565b60405180910390fd5b6000801b821415610306576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102fd9061074a565b60405180910390fd5b61030e61025c565b811461034f576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610346906107dc565b60405180910390fd5b816002600083815260200190815260200160002081905550806004819055505050565b60055481565b60035481565b600080fd5b6000819050919050565b61039681610383565b81146103a157600080fd5b50565b6000813590506103b38161038d565b92915050565b6000602082840312156103cf576103ce61037e565b5b60006103dd848285016103a4565b91505092915050565b6000819050919050565b6103f9816103e6565b82525050565b600060208201905061041460008301846103f0565b92915050565b61042381610383565b82525050565b600060208201905061043e600083018461041a565b92915050565b61044d816103e6565b811461045857600080fd5b50565b60008135905061046a81610444565b92915050565b600080604083850312156104875761048661037e565b5b60006104958582860161045b565b92505060206104a6858286016103a4565b9150509250929050565b600082825260208201905092915050565b7f74696d657374616d70207072696f7220746f207374617274696e67426c6f636b60008201527f54696d657374616d700000000000000000000000000000000000000000000000602082015250565b600061051d6029836104b0565b9150610528826104c1565b604082019050919050565b6000602082019050818103600083015261054c81610510565b9050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600061058d82610383565b915061059883610383565b9250828210156105ab576105aa610553565b5b828203905092915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b60006105f082610383565b91506105fb83610383565b92508261060b5761060a6105b6565b5b828204905092915050565b600061062182610383565b915061062c83610383565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0382111561066157610660610553565b5b828201905092915050565b7f43616e6e6f7420617070656e64204c32206f757470757420696e20667574757260008201527f6500000000000000000000000000000000000000000000000000000000000000602082015250565b60006106c86021836104b0565b91506106d38261066c565b604082019050919050565b600060208201905081810360008301526106f7816106bb565b9050919050565b7f43616e6e6f74207375626d697420656d707479204c32206f7574707574000000600082015250565b6000610734601d836104b0565b915061073f826106fe565b602082019050919050565b6000602082019050818103600083015261076381610727565b9050919050565b7f54696d657374616d70206e6f7420657175616c20746f206e657874206578706560008201527f637465642074696d657374616d70000000000000000000000000000000000000602082015250565b60006107c6602e836104b0565b91506107d18261076a565b604082019050919050565b600060208201905081810360008301526107f5816107b9565b905091905056fea2646970667358221220af714f0befbe4567f9ca655b787c21715645bed1c576fe2c2b7649d5dfe0af1564736f6c634300080b0033",
}

// MockL2OutputOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use MockL2OutputOracleMetaData.ABI instead.
var MockL2OutputOracleABI = MockL2OutputOracleMetaData.ABI

// MockL2OutputOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MockL2OutputOracleMetaData.Bin instead.
var MockL2OutputOracleBin = MockL2OutputOracleMetaData.Bin

// DeployMockL2OutputOracle deploys a new Ethereum contract, binding an instance of MockL2OutputOracle to it.
func DeployMockL2OutputOracle(auth *bind.TransactOpts, backend bind.ContractBackend, _submissionFrequency *big.Int, _l2BlockTime *big.Int, _genesisL2Output [32]byte, _historicalTotalBlocks *big.Int) (common.Address, *types.Transaction, *MockL2OutputOracle, error) {
	parsed, err := MockL2OutputOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MockL2OutputOracleBin), backend, _submissionFrequency, _l2BlockTime, _genesisL2Output, _historicalTotalBlocks)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MockL2OutputOracle{MockL2OutputOracleCaller: MockL2OutputOracleCaller{contract: contract}, MockL2OutputOracleTransactor: MockL2OutputOracleTransactor{contract: contract}, MockL2OutputOracleFilterer: MockL2OutputOracleFilterer{contract: contract}}, nil
}

// MockL2OutputOracle is an auto generated Go binding around an Ethereum contract.
type MockL2OutputOracle struct {
	MockL2OutputOracleCaller     // Read-only binding to the contract
	MockL2OutputOracleTransactor // Write-only binding to the contract
	MockL2OutputOracleFilterer   // Log filterer for contract events
}

// MockL2OutputOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type MockL2OutputOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockL2OutputOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MockL2OutputOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockL2OutputOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MockL2OutputOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockL2OutputOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MockL2OutputOracleSession struct {
	Contract     *MockL2OutputOracle // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// MockL2OutputOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MockL2OutputOracleCallerSession struct {
	Contract *MockL2OutputOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// MockL2OutputOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MockL2OutputOracleTransactorSession struct {
	Contract     *MockL2OutputOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// MockL2OutputOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type MockL2OutputOracleRaw struct {
	Contract *MockL2OutputOracle // Generic contract binding to access the raw methods on
}

// MockL2OutputOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MockL2OutputOracleCallerRaw struct {
	Contract *MockL2OutputOracleCaller // Generic read-only contract binding to access the raw methods on
}

// MockL2OutputOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MockL2OutputOracleTransactorRaw struct {
	Contract *MockL2OutputOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMockL2OutputOracle creates a new instance of MockL2OutputOracle, bound to a specific deployed contract.
func NewMockL2OutputOracle(address common.Address, backend bind.ContractBackend) (*MockL2OutputOracle, error) {
	contract, err := bindMockL2OutputOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MockL2OutputOracle{MockL2OutputOracleCaller: MockL2OutputOracleCaller{contract: contract}, MockL2OutputOracleTransactor: MockL2OutputOracleTransactor{contract: contract}, MockL2OutputOracleFilterer: MockL2OutputOracleFilterer{contract: contract}}, nil
}

// NewMockL2OutputOracleCaller creates a new read-only instance of MockL2OutputOracle, bound to a specific deployed contract.
func NewMockL2OutputOracleCaller(address common.Address, caller bind.ContractCaller) (*MockL2OutputOracleCaller, error) {
	contract, err := bindMockL2OutputOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MockL2OutputOracleCaller{contract: contract}, nil
}

// NewMockL2OutputOracleTransactor creates a new write-only instance of MockL2OutputOracle, bound to a specific deployed contract.
func NewMockL2OutputOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*MockL2OutputOracleTransactor, error) {
	contract, err := bindMockL2OutputOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MockL2OutputOracleTransactor{contract: contract}, nil
}

// NewMockL2OutputOracleFilterer creates a new log filterer instance of MockL2OutputOracle, bound to a specific deployed contract.
func NewMockL2OutputOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*MockL2OutputOracleFilterer, error) {
	contract, err := bindMockL2OutputOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MockL2OutputOracleFilterer{contract: contract}, nil
}

// bindMockL2OutputOracle binds a generic wrapper to an already deployed contract.
func bindMockL2OutputOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MockL2OutputOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockL2OutputOracle *MockL2OutputOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockL2OutputOracle.Contract.MockL2OutputOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockL2OutputOracle *MockL2OutputOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockL2OutputOracle.Contract.MockL2OutputOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockL2OutputOracle *MockL2OutputOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockL2OutputOracle.Contract.MockL2OutputOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockL2OutputOracle *MockL2OutputOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockL2OutputOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockL2OutputOracle *MockL2OutputOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockL2OutputOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockL2OutputOracle *MockL2OutputOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockL2OutputOracle.Contract.contract.Transact(opts, method, params...)
}

// ComputeL2BlockNumber is a free data retrieval call binding the contract method 0x02e51345.
//
// Solidity: function computeL2BlockNumber(uint256 _timestamp) view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCaller) ComputeL2BlockNumber(opts *bind.CallOpts, _timestamp *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _MockL2OutputOracle.contract.Call(opts, &out, "computeL2BlockNumber", _timestamp)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ComputeL2BlockNumber is a free data retrieval call binding the contract method 0x02e51345.
//
// Solidity: function computeL2BlockNumber(uint256 _timestamp) view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleSession) ComputeL2BlockNumber(_timestamp *big.Int) (*big.Int, error) {
	return _MockL2OutputOracle.Contract.ComputeL2BlockNumber(&_MockL2OutputOracle.CallOpts, _timestamp)
}

// ComputeL2BlockNumber is a free data retrieval call binding the contract method 0x02e51345.
//
// Solidity: function computeL2BlockNumber(uint256 _timestamp) view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCallerSession) ComputeL2BlockNumber(_timestamp *big.Int) (*big.Int, error) {
	return _MockL2OutputOracle.Contract.ComputeL2BlockNumber(&_MockL2OutputOracle.CallOpts, _timestamp)
}

// HistoricalTotalBlocks is a free data retrieval call binding the contract method 0xc90ec2da.
//
// Solidity: function historicalTotalBlocks() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCaller) HistoricalTotalBlocks(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MockL2OutputOracle.contract.Call(opts, &out, "historicalTotalBlocks")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HistoricalTotalBlocks is a free data retrieval call binding the contract method 0xc90ec2da.
//
// Solidity: function historicalTotalBlocks() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleSession) HistoricalTotalBlocks() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.HistoricalTotalBlocks(&_MockL2OutputOracle.CallOpts)
}

// HistoricalTotalBlocks is a free data retrieval call binding the contract method 0xc90ec2da.
//
// Solidity: function historicalTotalBlocks() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCallerSession) HistoricalTotalBlocks() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.HistoricalTotalBlocks(&_MockL2OutputOracle.CallOpts)
}

// L2BlockTime is a free data retrieval call binding the contract method 0x93991af3.
//
// Solidity: function l2BlockTime() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCaller) L2BlockTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MockL2OutputOracle.contract.Call(opts, &out, "l2BlockTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2BlockTime is a free data retrieval call binding the contract method 0x93991af3.
//
// Solidity: function l2BlockTime() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleSession) L2BlockTime() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.L2BlockTime(&_MockL2OutputOracle.CallOpts)
}

// L2BlockTime is a free data retrieval call binding the contract method 0x93991af3.
//
// Solidity: function l2BlockTime() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCallerSession) L2BlockTime() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.L2BlockTime(&_MockL2OutputOracle.CallOpts)
}

// L2Outputs is a free data retrieval call binding the contract method 0x02be8bfe.
//
// Solidity: function l2Outputs(uint256 ) view returns(bytes32)
func (_MockL2OutputOracle *MockL2OutputOracleCaller) L2Outputs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _MockL2OutputOracle.contract.Call(opts, &out, "l2Outputs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L2Outputs is a free data retrieval call binding the contract method 0x02be8bfe.
//
// Solidity: function l2Outputs(uint256 ) view returns(bytes32)
func (_MockL2OutputOracle *MockL2OutputOracleSession) L2Outputs(arg0 *big.Int) ([32]byte, error) {
	return _MockL2OutputOracle.Contract.L2Outputs(&_MockL2OutputOracle.CallOpts, arg0)
}

// L2Outputs is a free data retrieval call binding the contract method 0x02be8bfe.
//
// Solidity: function l2Outputs(uint256 ) view returns(bytes32)
func (_MockL2OutputOracle *MockL2OutputOracleCallerSession) L2Outputs(arg0 *big.Int) ([32]byte, error) {
	return _MockL2OutputOracle.Contract.L2Outputs(&_MockL2OutputOracle.CallOpts, arg0)
}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCaller) LatestBlockTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MockL2OutputOracle.contract.Call(opts, &out, "latestBlockTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleSession) LatestBlockTimestamp() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.LatestBlockTimestamp(&_MockL2OutputOracle.CallOpts)
}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCallerSession) LatestBlockTimestamp() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.LatestBlockTimestamp(&_MockL2OutputOracle.CallOpts)
}

// NextTimestamp is a free data retrieval call binding the contract method 0x357e951f.
//
// Solidity: function nextTimestamp() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCaller) NextTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MockL2OutputOracle.contract.Call(opts, &out, "nextTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextTimestamp is a free data retrieval call binding the contract method 0x357e951f.
//
// Solidity: function nextTimestamp() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleSession) NextTimestamp() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.NextTimestamp(&_MockL2OutputOracle.CallOpts)
}

// NextTimestamp is a free data retrieval call binding the contract method 0x357e951f.
//
// Solidity: function nextTimestamp() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCallerSession) NextTimestamp() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.NextTimestamp(&_MockL2OutputOracle.CallOpts)
}

// StartingBlockTimestamp is a free data retrieval call binding the contract method 0xc5095d68.
//
// Solidity: function startingBlockTimestamp() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCaller) StartingBlockTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MockL2OutputOracle.contract.Call(opts, &out, "startingBlockTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StartingBlockTimestamp is a free data retrieval call binding the contract method 0xc5095d68.
//
// Solidity: function startingBlockTimestamp() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleSession) StartingBlockTimestamp() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.StartingBlockTimestamp(&_MockL2OutputOracle.CallOpts)
}

// StartingBlockTimestamp is a free data retrieval call binding the contract method 0xc5095d68.
//
// Solidity: function startingBlockTimestamp() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCallerSession) StartingBlockTimestamp() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.StartingBlockTimestamp(&_MockL2OutputOracle.CallOpts)
}

// SubmissionFrequency is a free data retrieval call binding the contract method 0xb210dc21.
//
// Solidity: function submissionFrequency() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCaller) SubmissionFrequency(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MockL2OutputOracle.contract.Call(opts, &out, "submissionFrequency")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SubmissionFrequency is a free data retrieval call binding the contract method 0xb210dc21.
//
// Solidity: function submissionFrequency() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleSession) SubmissionFrequency() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.SubmissionFrequency(&_MockL2OutputOracle.CallOpts)
}

// SubmissionFrequency is a free data retrieval call binding the contract method 0xb210dc21.
//
// Solidity: function submissionFrequency() view returns(uint256)
func (_MockL2OutputOracle *MockL2OutputOracleCallerSession) SubmissionFrequency() (*big.Int, error) {
	return _MockL2OutputOracle.Contract.SubmissionFrequency(&_MockL2OutputOracle.CallOpts)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0xb71d13e2.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _timestamp) returns()
func (_MockL2OutputOracle *MockL2OutputOracleTransactor) AppendL2Output(opts *bind.TransactOpts, _l2Output [32]byte, _timestamp *big.Int) (*types.Transaction, error) {
	return _MockL2OutputOracle.contract.Transact(opts, "appendL2Output", _l2Output, _timestamp)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0xb71d13e2.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _timestamp) returns()
func (_MockL2OutputOracle *MockL2OutputOracleSession) AppendL2Output(_l2Output [32]byte, _timestamp *big.Int) (*types.Transaction, error) {
	return _MockL2OutputOracle.Contract.AppendL2Output(&_MockL2OutputOracle.TransactOpts, _l2Output, _timestamp)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0xb71d13e2.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _timestamp) returns()
func (_MockL2OutputOracle *MockL2OutputOracleTransactorSession) AppendL2Output(_l2Output [32]byte, _timestamp *big.Int) (*types.Transaction, error) {
	return _MockL2OutputOracle.Contract.AppendL2Output(&_MockL2OutputOracle.TransactOpts, _l2Output, _timestamp)
}
