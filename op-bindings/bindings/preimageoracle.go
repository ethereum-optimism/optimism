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

// PreimageOracleMetaData contains all meta data concerning the PreimageOracle contract.
var PreimageOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_key\",\"type\":\"bytes32\"}],\"name\":\"_getKeyKindMappingSlots\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"partOk_\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"lengths_\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"parts_\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"partOffset\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"part\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"size\",\"type\":\"uint256\"}],\"name\":\"cheat\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"partOffset\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"preimage\",\"type\":\"bytes\"}],\"name\":\"loadKeccak256PreimagePart\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_game\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_partOffset\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_bootData\",\"type\":\"bytes\"}],\"name\":\"loadLocalBootData\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"offset\",\"type\":\"uint256\"}],\"name\":\"readPreimage\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"dat\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"datLen\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506105e5806100206000396000f3fe608060405234801561001057600080fd5b50600436106100675760003560e01c8063e03110e111610050578063e03110e1146100d4578063e1592611146100fc578063fe4ac08e1461010f57600080fd5b8063a1a7c5b41461006c578063d4418d68146100bf575b600080fd5b61009f61007a366004610461565b60f81c6000818152602080822061010084178352818320610200909417835291209092565b604080519384526020840192909252908201526060015b60405180910390f35b6100d26100cd3660046104c3565b610122565b005b6100e76100e2366004610538565b610233565b604080519283526020830191909152016100b6565b6100d261010a36600461055a565b6102d7565b6100d261011d3660046105a6565b6103dd565b61020160009081527fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6907f109ea3cebb188b9c1b9fc5bb3920be60dfdc8699098dff92f3d80daaca747689907f1d5547087d18552f21de5bab5e2346289fbe5450d69c19cc3d98dea7d13e3ba29060643590806008830189106101a457600080fd5b60808360c01b8152838960088301378901519150507f010000000000000000000000000000000000000000000000000000000000000160588a901b17610203818a60018960009384526020918252604080852093855292909152912055565b6000818152602092835260408082209a825299835289812094909455835292909252949094209390935550505050565b60008060008060006102648760f81c6000818152602080822061010084178352818320610200909417835291209092565b92509250925060008781526020848152604080832089845290915290205461028b57600080fd5b600087815260208381526040909120549094506020870160088201116102b45786600882010394505b506000968752602090815260408088209688529590525050919092205492909150565b6044356000806008830186106102ec57600080fd5b60c083901b6080526088838682378087017ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80151908490207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f02000000000000000000000000000000000000000000000000000000000000001792509050600080806103988560f81c6000818152602080822061010084178352818320610200909417835291209092565b60008881526001602090815260408083208f84528252808320959095558982529788528381209c81529b8752828c2055958a5294909352505050909320929092555050565b600080600061040b8660f81c6000818152602080822061010084178352818320610200909417835291209092565b92509250925060018660001a03610425573360581b861795505b60008681526001602090815260408083208a84528252808320959095558782529586528381209781529685528287205593855292909152912055565b60006020828403121561047357600080fd5b5035919050565b60008083601f84011261048c57600080fd5b50813567ffffffffffffffff8111156104a457600080fd5b6020830191508360208285010111156104bc57600080fd5b9250929050565b600080600080606085870312156104d957600080fd5b843573ffffffffffffffffffffffffffffffffffffffff811681146104fd57600080fd5b935060208501359250604085013567ffffffffffffffff81111561052057600080fd5b61052c8782880161047a565b95989497509550505050565b6000806040838503121561054b57600080fd5b50508035926020909101359150565b60008060006040848603121561056f57600080fd5b83359250602084013567ffffffffffffffff81111561058d57600080fd5b6105998682870161047a565b9497909650939450505050565b600080600080608085870312156105bc57600080fd5b505082359460208401359450604084013593606001359250905056fea164736f6c634300080f000a",
}

// PreimageOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use PreimageOracleMetaData.ABI instead.
var PreimageOracleABI = PreimageOracleMetaData.ABI

// PreimageOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use PreimageOracleMetaData.Bin instead.
var PreimageOracleBin = PreimageOracleMetaData.Bin

// DeployPreimageOracle deploys a new Ethereum contract, binding an instance of PreimageOracle to it.
func DeployPreimageOracle(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *PreimageOracle, error) {
	parsed, err := PreimageOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PreimageOracleBin), backend)
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
	parsed, err := abi.JSON(strings.NewReader(PreimageOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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

// GetKeyKindMappingSlots is a free data retrieval call binding the contract method 0xa1a7c5b4.
//
// Solidity: function _getKeyKindMappingSlots(bytes32 _key) pure returns(bytes32 partOk_, bytes32 lengths_, bytes32 parts_)
func (_PreimageOracle *PreimageOracleCaller) GetKeyKindMappingSlots(opts *bind.CallOpts, _key [32]byte) (struct {
	PartOk  [32]byte
	Lengths [32]byte
	Parts   [32]byte
}, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "_getKeyKindMappingSlots", _key)

	outstruct := new(struct {
		PartOk  [32]byte
		Lengths [32]byte
		Parts   [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.PartOk = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Lengths = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.Parts = *abi.ConvertType(out[2], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// GetKeyKindMappingSlots is a free data retrieval call binding the contract method 0xa1a7c5b4.
//
// Solidity: function _getKeyKindMappingSlots(bytes32 _key) pure returns(bytes32 partOk_, bytes32 lengths_, bytes32 parts_)
func (_PreimageOracle *PreimageOracleSession) GetKeyKindMappingSlots(_key [32]byte) (struct {
	PartOk  [32]byte
	Lengths [32]byte
	Parts   [32]byte
}, error) {
	return _PreimageOracle.Contract.GetKeyKindMappingSlots(&_PreimageOracle.CallOpts, _key)
}

// GetKeyKindMappingSlots is a free data retrieval call binding the contract method 0xa1a7c5b4.
//
// Solidity: function _getKeyKindMappingSlots(bytes32 _key) pure returns(bytes32 partOk_, bytes32 lengths_, bytes32 parts_)
func (_PreimageOracle *PreimageOracleCallerSession) GetKeyKindMappingSlots(_key [32]byte) (struct {
	PartOk  [32]byte
	Lengths [32]byte
	Parts   [32]byte
}, error) {
	return _PreimageOracle.Contract.GetKeyKindMappingSlots(&_PreimageOracle.CallOpts, _key)
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

// LoadLocalBootData is a paid mutator transaction binding the contract method 0xd4418d68.
//
// Solidity: function loadLocalBootData(address _game, uint256 _partOffset, bytes _bootData) returns()
func (_PreimageOracle *PreimageOracleTransactor) LoadLocalBootData(opts *bind.TransactOpts, _game common.Address, _partOffset *big.Int, _bootData []byte) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "loadLocalBootData", _game, _partOffset, _bootData)
}

// LoadLocalBootData is a paid mutator transaction binding the contract method 0xd4418d68.
//
// Solidity: function loadLocalBootData(address _game, uint256 _partOffset, bytes _bootData) returns()
func (_PreimageOracle *PreimageOracleSession) LoadLocalBootData(_game common.Address, _partOffset *big.Int, _bootData []byte) (*types.Transaction, error) {
	return _PreimageOracle.Contract.LoadLocalBootData(&_PreimageOracle.TransactOpts, _game, _partOffset, _bootData)
}

// LoadLocalBootData is a paid mutator transaction binding the contract method 0xd4418d68.
//
// Solidity: function loadLocalBootData(address _game, uint256 _partOffset, bytes _bootData) returns()
func (_PreimageOracle *PreimageOracleTransactorSession) LoadLocalBootData(_game common.Address, _partOffset *big.Int, _bootData []byte) (*types.Transaction, error) {
	return _PreimageOracle.Contract.LoadLocalBootData(&_PreimageOracle.TransactOpts, _game, _partOffset, _bootData)
}
