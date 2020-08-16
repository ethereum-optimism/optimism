// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package plasma

import (
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
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// PlasmaABI is the input ABI used to generate the binding from.
const PlasmaABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"isChildChainActivated\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"childBlockInterval\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getMaintainer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"nextChildBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_vaultId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_vaultAddress\",\"type\":\"address\"}],\"name\":\"registerVault\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_vaultId\",\"type\":\"uint256\"}],\"name\":\"vaults\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"nextDeposit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"authority\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"address\",\"name\":\"_vaultAddress\",\"type\":\"address\"}],\"name\":\"vaultToId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"blocks\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_interval\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_minExitPeriod\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_initialImmuneVaults\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_authority\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"BlockSubmitted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"authority\",\"type\":\"address\"}],\"name\":\"ChildChainActivated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"vaultId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"vaultAddress\",\"type\":\"address\"}],\"name\":\"VaultRegistered\",\"type\":\"event\"},{\"constant\":false,\"inputs\":[],\"name\":\"activateChildChain\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_blockRoot\",\"type\":\"bytes32\"}],\"name\":\"submitBlock\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_blockRoot\",\"type\":\"bytes32\"}],\"name\":\"submitDepositBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"nextDepositBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNum\",\"type\":\"uint256\"}],\"name\":\"isDeposit\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"

// Plasma is an auto generated Go binding around an Ethereum contract.
type Plasma struct {
	PlasmaCaller     // Read-only binding to the contract
	PlasmaTransactor // Write-only binding to the contract
	PlasmaFilterer   // Log filterer for contract events
}

// PlasmaCaller is an auto generated read-only Go binding around an Ethereum contract.
type PlasmaCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PlasmaTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PlasmaTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PlasmaFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PlasmaFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PlasmaSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PlasmaSession struct {
	Contract     *Plasma           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PlasmaCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PlasmaCallerSession struct {
	Contract *PlasmaCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// PlasmaTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PlasmaTransactorSession struct {
	Contract     *PlasmaTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PlasmaRaw is an auto generated low-level Go binding around an Ethereum contract.
type PlasmaRaw struct {
	Contract *Plasma // Generic contract binding to access the raw methods on
}

// PlasmaCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PlasmaCallerRaw struct {
	Contract *PlasmaCaller // Generic read-only contract binding to access the raw methods on
}

// PlasmaTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PlasmaTransactorRaw struct {
	Contract *PlasmaTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPlasma creates a new instance of Plasma, bound to a specific deployed contract.
func NewPlasma(address common.Address, backend bind.ContractBackend) (*Plasma, error) {
	contract, err := bindPlasma(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Plasma{PlasmaCaller: PlasmaCaller{contract: contract}, PlasmaTransactor: PlasmaTransactor{contract: contract}, PlasmaFilterer: PlasmaFilterer{contract: contract}}, nil
}

// NewPlasmaCaller creates a new read-only instance of Plasma, bound to a specific deployed contract.
func NewPlasmaCaller(address common.Address, caller bind.ContractCaller) (*PlasmaCaller, error) {
	contract, err := bindPlasma(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PlasmaCaller{contract: contract}, nil
}

// NewPlasmaTransactor creates a new write-only instance of Plasma, bound to a specific deployed contract.
func NewPlasmaTransactor(address common.Address, transactor bind.ContractTransactor) (*PlasmaTransactor, error) {
	contract, err := bindPlasma(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PlasmaTransactor{contract: contract}, nil
}

// NewPlasmaFilterer creates a new log filterer instance of Plasma, bound to a specific deployed contract.
func NewPlasmaFilterer(address common.Address, filterer bind.ContractFilterer) (*PlasmaFilterer, error) {
	contract, err := bindPlasma(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PlasmaFilterer{contract: contract}, nil
}

// bindPlasma binds a generic wrapper to an already deployed contract.
func bindPlasma(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PlasmaABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Plasma *PlasmaRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Plasma.Contract.PlasmaCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Plasma *PlasmaRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Plasma.Contract.PlasmaTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Plasma *PlasmaRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Plasma.Contract.PlasmaTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Plasma *PlasmaCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Plasma.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Plasma *PlasmaTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Plasma.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Plasma *PlasmaTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Plasma.Contract.contract.Transact(opts, method, params...)
}

// Authority is a free data retrieval call binding the contract method 0xbf7e214f.
//
// Solidity: function authority() view returns(address)
func (_Plasma *PlasmaCaller) Authority(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Plasma.contract.Call(opts, out, "authority")
	return *ret0, err
}

// Authority is a free data retrieval call binding the contract method 0xbf7e214f.
//
// Solidity: function authority() view returns(address)
func (_Plasma *PlasmaSession) Authority() (common.Address, error) {
	return _Plasma.Contract.Authority(&_Plasma.CallOpts)
}

// Authority is a free data retrieval call binding the contract method 0xbf7e214f.
//
// Solidity: function authority() view returns(address)
func (_Plasma *PlasmaCallerSession) Authority() (common.Address, error) {
	return _Plasma.Contract.Authority(&_Plasma.CallOpts)
}

// Blocks is a free data retrieval call binding the contract method 0xf25b3f99.
//
// Solidity: function blocks(uint256 ) view returns(bytes32 root, uint256 timestamp)
func (_Plasma *PlasmaCaller) Blocks(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Root      [32]byte
	Timestamp *big.Int
}, error) {
	ret := new(struct {
		Root      [32]byte
		Timestamp *big.Int
	})
	out := ret
	err := _Plasma.contract.Call(opts, out, "blocks", arg0)
	return *ret, err
}

// Blocks is a free data retrieval call binding the contract method 0xf25b3f99.
//
// Solidity: function blocks(uint256 ) view returns(bytes32 root, uint256 timestamp)
func (_Plasma *PlasmaSession) Blocks(arg0 *big.Int) (struct {
	Root      [32]byte
	Timestamp *big.Int
}, error) {
	return _Plasma.Contract.Blocks(&_Plasma.CallOpts, arg0)
}

// Blocks is a free data retrieval call binding the contract method 0xf25b3f99.
//
// Solidity: function blocks(uint256 ) view returns(bytes32 root, uint256 timestamp)
func (_Plasma *PlasmaCallerSession) Blocks(arg0 *big.Int) (struct {
	Root      [32]byte
	Timestamp *big.Int
}, error) {
	return _Plasma.Contract.Blocks(&_Plasma.CallOpts, arg0)
}

// ChildBlockInterval is a free data retrieval call binding the contract method 0x38a9e0bc.
//
// Solidity: function childBlockInterval() view returns(uint256)
func (_Plasma *PlasmaCaller) ChildBlockInterval(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Plasma.contract.Call(opts, out, "childBlockInterval")
	return *ret0, err
}

// ChildBlockInterval is a free data retrieval call binding the contract method 0x38a9e0bc.
//
// Solidity: function childBlockInterval() view returns(uint256)
func (_Plasma *PlasmaSession) ChildBlockInterval() (*big.Int, error) {
	return _Plasma.Contract.ChildBlockInterval(&_Plasma.CallOpts)
}

// ChildBlockInterval is a free data retrieval call binding the contract method 0x38a9e0bc.
//
// Solidity: function childBlockInterval() view returns(uint256)
func (_Plasma *PlasmaCallerSession) ChildBlockInterval() (*big.Int, error) {
	return _Plasma.Contract.ChildBlockInterval(&_Plasma.CallOpts)
}

// GetMaintainer is a free data retrieval call binding the contract method 0x4b0a72bc.
//
// Solidity: function getMaintainer() view returns(address)
func (_Plasma *PlasmaCaller) GetMaintainer(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Plasma.contract.Call(opts, out, "getMaintainer")
	return *ret0, err
}

// GetMaintainer is a free data retrieval call binding the contract method 0x4b0a72bc.
//
// Solidity: function getMaintainer() view returns(address)
func (_Plasma *PlasmaSession) GetMaintainer() (common.Address, error) {
	return _Plasma.Contract.GetMaintainer(&_Plasma.CallOpts)
}

// GetMaintainer is a free data retrieval call binding the contract method 0x4b0a72bc.
//
// Solidity: function getMaintainer() view returns(address)
func (_Plasma *PlasmaCallerSession) GetMaintainer() (common.Address, error) {
	return _Plasma.Contract.GetMaintainer(&_Plasma.CallOpts)
}

// IsChildChainActivated is a free data retrieval call binding the contract method 0x0e71ee02.
//
// Solidity: function isChildChainActivated() view returns(bool)
func (_Plasma *PlasmaCaller) IsChildChainActivated(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Plasma.contract.Call(opts, out, "isChildChainActivated")
	return *ret0, err
}

// IsChildChainActivated is a free data retrieval call binding the contract method 0x0e71ee02.
//
// Solidity: function isChildChainActivated() view returns(bool)
func (_Plasma *PlasmaSession) IsChildChainActivated() (bool, error) {
	return _Plasma.Contract.IsChildChainActivated(&_Plasma.CallOpts)
}

// IsChildChainActivated is a free data retrieval call binding the contract method 0x0e71ee02.
//
// Solidity: function isChildChainActivated() view returns(bool)
func (_Plasma *PlasmaCallerSession) IsChildChainActivated() (bool, error) {
	return _Plasma.Contract.IsChildChainActivated(&_Plasma.CallOpts)
}

// IsDeposit is a free data retrieval call binding the contract method 0xebde2ec9.
//
// Solidity: function isDeposit(uint256 blockNum) view returns(bool)
func (_Plasma *PlasmaCaller) IsDeposit(opts *bind.CallOpts, blockNum *big.Int) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Plasma.contract.Call(opts, out, "isDeposit", blockNum)
	return *ret0, err
}

// IsDeposit is a free data retrieval call binding the contract method 0xebde2ec9.
//
// Solidity: function isDeposit(uint256 blockNum) view returns(bool)
func (_Plasma *PlasmaSession) IsDeposit(blockNum *big.Int) (bool, error) {
	return _Plasma.Contract.IsDeposit(&_Plasma.CallOpts, blockNum)
}

// IsDeposit is a free data retrieval call binding the contract method 0xebde2ec9.
//
// Solidity: function isDeposit(uint256 blockNum) view returns(bool)
func (_Plasma *PlasmaCallerSession) IsDeposit(blockNum *big.Int) (bool, error) {
	return _Plasma.Contract.IsDeposit(&_Plasma.CallOpts, blockNum)
}

// NextChildBlock is a free data retrieval call binding the contract method 0x4ca8714f.
//
// Solidity: function nextChildBlock() view returns(uint256)
func (_Plasma *PlasmaCaller) NextChildBlock(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Plasma.contract.Call(opts, out, "nextChildBlock")
	return *ret0, err
}

// NextChildBlock is a free data retrieval call binding the contract method 0x4ca8714f.
//
// Solidity: function nextChildBlock() view returns(uint256)
func (_Plasma *PlasmaSession) NextChildBlock() (*big.Int, error) {
	return _Plasma.Contract.NextChildBlock(&_Plasma.CallOpts)
}

// NextChildBlock is a free data retrieval call binding the contract method 0x4ca8714f.
//
// Solidity: function nextChildBlock() view returns(uint256)
func (_Plasma *PlasmaCallerSession) NextChildBlock() (*big.Int, error) {
	return _Plasma.Contract.NextChildBlock(&_Plasma.CallOpts)
}

// NextDeposit is a free data retrieval call binding the contract method 0xa8cabcd5.
//
// Solidity: function nextDeposit() view returns(uint256)
func (_Plasma *PlasmaCaller) NextDeposit(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Plasma.contract.Call(opts, out, "nextDeposit")
	return *ret0, err
}

// NextDeposit is a free data retrieval call binding the contract method 0xa8cabcd5.
//
// Solidity: function nextDeposit() view returns(uint256)
func (_Plasma *PlasmaSession) NextDeposit() (*big.Int, error) {
	return _Plasma.Contract.NextDeposit(&_Plasma.CallOpts)
}

// NextDeposit is a free data retrieval call binding the contract method 0xa8cabcd5.
//
// Solidity: function nextDeposit() view returns(uint256)
func (_Plasma *PlasmaCallerSession) NextDeposit() (*big.Int, error) {
	return _Plasma.Contract.NextDeposit(&_Plasma.CallOpts)
}

// NextDepositBlock is a free data retrieval call binding the contract method 0x8701fc5d.
//
// Solidity: function nextDepositBlock() view returns(uint256)
func (_Plasma *PlasmaCaller) NextDepositBlock(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Plasma.contract.Call(opts, out, "nextDepositBlock")
	return *ret0, err
}

// NextDepositBlock is a free data retrieval call binding the contract method 0x8701fc5d.
//
// Solidity: function nextDepositBlock() view returns(uint256)
func (_Plasma *PlasmaSession) NextDepositBlock() (*big.Int, error) {
	return _Plasma.Contract.NextDepositBlock(&_Plasma.CallOpts)
}

// NextDepositBlock is a free data retrieval call binding the contract method 0x8701fc5d.
//
// Solidity: function nextDepositBlock() view returns(uint256)
func (_Plasma *PlasmaCallerSession) NextDepositBlock() (*big.Int, error) {
	return _Plasma.Contract.NextDepositBlock(&_Plasma.CallOpts)
}

// VaultToId is a free data retrieval call binding the contract method 0xdfb494f0.
//
// Solidity: function vaultToId(address _vaultAddress) view returns(uint256)
func (_Plasma *PlasmaCaller) VaultToId(opts *bind.CallOpts, _vaultAddress common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Plasma.contract.Call(opts, out, "vaultToId", _vaultAddress)
	return *ret0, err
}

// VaultToId is a free data retrieval call binding the contract method 0xdfb494f0.
//
// Solidity: function vaultToId(address _vaultAddress) view returns(uint256)
func (_Plasma *PlasmaSession) VaultToId(_vaultAddress common.Address) (*big.Int, error) {
	return _Plasma.Contract.VaultToId(&_Plasma.CallOpts, _vaultAddress)
}

// VaultToId is a free data retrieval call binding the contract method 0xdfb494f0.
//
// Solidity: function vaultToId(address _vaultAddress) view returns(uint256)
func (_Plasma *PlasmaCallerSession) VaultToId(_vaultAddress common.Address) (*big.Int, error) {
	return _Plasma.Contract.VaultToId(&_Plasma.CallOpts, _vaultAddress)
}

// Vaults is a free data retrieval call binding the contract method 0x8c64ea4a.
//
// Solidity: function vaults(uint256 _vaultId) view returns(address)
func (_Plasma *PlasmaCaller) Vaults(opts *bind.CallOpts, _vaultId *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Plasma.contract.Call(opts, out, "vaults", _vaultId)
	return *ret0, err
}

// Vaults is a free data retrieval call binding the contract method 0x8c64ea4a.
//
// Solidity: function vaults(uint256 _vaultId) view returns(address)
func (_Plasma *PlasmaSession) Vaults(_vaultId *big.Int) (common.Address, error) {
	return _Plasma.Contract.Vaults(&_Plasma.CallOpts, _vaultId)
}

// Vaults is a free data retrieval call binding the contract method 0x8c64ea4a.
//
// Solidity: function vaults(uint256 _vaultId) view returns(address)
func (_Plasma *PlasmaCallerSession) Vaults(_vaultId *big.Int) (common.Address, error) {
	return _Plasma.Contract.Vaults(&_Plasma.CallOpts, _vaultId)
}

// ActivateChildChain is a paid mutator transaction binding the contract method 0xa11dcc34.
//
// Solidity: function activateChildChain() returns()
func (_Plasma *PlasmaTransactor) ActivateChildChain(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Plasma.contract.Transact(opts, "activateChildChain")
}

// ActivateChildChain is a paid mutator transaction binding the contract method 0xa11dcc34.
//
// Solidity: function activateChildChain() returns()
func (_Plasma *PlasmaSession) ActivateChildChain() (*types.Transaction, error) {
	return _Plasma.Contract.ActivateChildChain(&_Plasma.TransactOpts)
}

// ActivateChildChain is a paid mutator transaction binding the contract method 0xa11dcc34.
//
// Solidity: function activateChildChain() returns()
func (_Plasma *PlasmaTransactorSession) ActivateChildChain() (*types.Transaction, error) {
	return _Plasma.Contract.ActivateChildChain(&_Plasma.TransactOpts)
}

// RegisterVault is a paid mutator transaction binding the contract method 0x6a51fd63.
//
// Solidity: function registerVault(uint256 _vaultId, address _vaultAddress) returns()
func (_Plasma *PlasmaTransactor) RegisterVault(opts *bind.TransactOpts, _vaultId *big.Int, _vaultAddress common.Address) (*types.Transaction, error) {
	return _Plasma.contract.Transact(opts, "registerVault", _vaultId, _vaultAddress)
}

// RegisterVault is a paid mutator transaction binding the contract method 0x6a51fd63.
//
// Solidity: function registerVault(uint256 _vaultId, address _vaultAddress) returns()
func (_Plasma *PlasmaSession) RegisterVault(_vaultId *big.Int, _vaultAddress common.Address) (*types.Transaction, error) {
	return _Plasma.Contract.RegisterVault(&_Plasma.TransactOpts, _vaultId, _vaultAddress)
}

// RegisterVault is a paid mutator transaction binding the contract method 0x6a51fd63.
//
// Solidity: function registerVault(uint256 _vaultId, address _vaultAddress) returns()
func (_Plasma *PlasmaTransactorSession) RegisterVault(_vaultId *big.Int, _vaultAddress common.Address) (*types.Transaction, error) {
	return _Plasma.Contract.RegisterVault(&_Plasma.TransactOpts, _vaultId, _vaultAddress)
}

// SubmitBlock is a paid mutator transaction binding the contract method 0xbaa47694.
//
// Solidity: function submitBlock(bytes32 _blockRoot) returns()
func (_Plasma *PlasmaTransactor) SubmitBlock(opts *bind.TransactOpts, _blockRoot [32]byte) (*types.Transaction, error) {
	return _Plasma.contract.Transact(opts, "submitBlock", _blockRoot)
}

// SubmitBlock is a paid mutator transaction binding the contract method 0xbaa47694.
//
// Solidity: function submitBlock(bytes32 _blockRoot) returns()
func (_Plasma *PlasmaSession) SubmitBlock(_blockRoot [32]byte) (*types.Transaction, error) {
	return _Plasma.Contract.SubmitBlock(&_Plasma.TransactOpts, _blockRoot)
}

// SubmitBlock is a paid mutator transaction binding the contract method 0xbaa47694.
//
// Solidity: function submitBlock(bytes32 _blockRoot) returns()
func (_Plasma *PlasmaTransactorSession) SubmitBlock(_blockRoot [32]byte) (*types.Transaction, error) {
	return _Plasma.Contract.SubmitBlock(&_Plasma.TransactOpts, _blockRoot)
}

// SubmitDepositBlock is a paid mutator transaction binding the contract method 0xbe5ac698.
//
// Solidity: function submitDepositBlock(bytes32 _blockRoot) returns(uint256)
func (_Plasma *PlasmaTransactor) SubmitDepositBlock(opts *bind.TransactOpts, _blockRoot [32]byte) (*types.Transaction, error) {
	return _Plasma.contract.Transact(opts, "submitDepositBlock", _blockRoot)
}

// SubmitDepositBlock is a paid mutator transaction binding the contract method 0xbe5ac698.
//
// Solidity: function submitDepositBlock(bytes32 _blockRoot) returns(uint256)
func (_Plasma *PlasmaSession) SubmitDepositBlock(_blockRoot [32]byte) (*types.Transaction, error) {
	return _Plasma.Contract.SubmitDepositBlock(&_Plasma.TransactOpts, _blockRoot)
}

// SubmitDepositBlock is a paid mutator transaction binding the contract method 0xbe5ac698.
//
// Solidity: function submitDepositBlock(bytes32 _blockRoot) returns(uint256)
func (_Plasma *PlasmaTransactorSession) SubmitDepositBlock(_blockRoot [32]byte) (*types.Transaction, error) {
	return _Plasma.Contract.SubmitDepositBlock(&_Plasma.TransactOpts, _blockRoot)
}

// PlasmaBlockSubmittedIterator is returned from FilterBlockSubmitted and is used to iterate over the raw logs and unpacked data for BlockSubmitted events raised by the Plasma contract.
type PlasmaBlockSubmittedIterator struct {
	Event *PlasmaBlockSubmitted // Event containing the contract specifics and raw log

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
func (it *PlasmaBlockSubmittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PlasmaBlockSubmitted)
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
		it.Event = new(PlasmaBlockSubmitted)
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
func (it *PlasmaBlockSubmittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PlasmaBlockSubmittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PlasmaBlockSubmitted represents a BlockSubmitted event raised by the Plasma contract.
type PlasmaBlockSubmitted struct {
	BlockNumber *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterBlockSubmitted is a free log retrieval operation binding the contract event 0x5a978f4723b249ccf79cd7a658a8601ce1ff8b89fc770251a6be35216351ce32.
//
// Solidity: event BlockSubmitted(uint256 blockNumber)
func (_Plasma *PlasmaFilterer) FilterBlockSubmitted(opts *bind.FilterOpts) (*PlasmaBlockSubmittedIterator, error) {

	logs, sub, err := _Plasma.contract.FilterLogs(opts, "BlockSubmitted")
	if err != nil {
		return nil, err
	}
	return &PlasmaBlockSubmittedIterator{contract: _Plasma.contract, event: "BlockSubmitted", logs: logs, sub: sub}, nil
}

// WatchBlockSubmitted is a free log subscription operation binding the contract event 0x5a978f4723b249ccf79cd7a658a8601ce1ff8b89fc770251a6be35216351ce32.
//
// Solidity: event BlockSubmitted(uint256 blockNumber)
func (_Plasma *PlasmaFilterer) WatchBlockSubmitted(opts *bind.WatchOpts, sink chan<- *PlasmaBlockSubmitted) (event.Subscription, error) {

	logs, sub, err := _Plasma.contract.WatchLogs(opts, "BlockSubmitted")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PlasmaBlockSubmitted)
				if err := _Plasma.contract.UnpackLog(event, "BlockSubmitted", log); err != nil {
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

// ParseBlockSubmitted is a log parse operation binding the contract event 0x5a978f4723b249ccf79cd7a658a8601ce1ff8b89fc770251a6be35216351ce32.
//
// Solidity: event BlockSubmitted(uint256 blockNumber)
func (_Plasma *PlasmaFilterer) ParseBlockSubmitted(log types.Log) (*PlasmaBlockSubmitted, error) {
	event := new(PlasmaBlockSubmitted)
	if err := _Plasma.contract.UnpackLog(event, "BlockSubmitted", log); err != nil {
		return nil, err
	}
	return event, nil
}

// PlasmaChildChainActivatedIterator is returned from FilterChildChainActivated and is used to iterate over the raw logs and unpacked data for ChildChainActivated events raised by the Plasma contract.
type PlasmaChildChainActivatedIterator struct {
	Event *PlasmaChildChainActivated // Event containing the contract specifics and raw log

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
func (it *PlasmaChildChainActivatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PlasmaChildChainActivated)
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
		it.Event = new(PlasmaChildChainActivated)
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
func (it *PlasmaChildChainActivatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PlasmaChildChainActivatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PlasmaChildChainActivated represents a ChildChainActivated event raised by the Plasma contract.
type PlasmaChildChainActivated struct {
	Authority common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterChildChainActivated is a free log retrieval operation binding the contract event 0xb8421a1acb5f1e701a4f11ecaad76fa438b3947e8dfd6960b6086130e68e0aed.
//
// Solidity: event ChildChainActivated(address authority)
func (_Plasma *PlasmaFilterer) FilterChildChainActivated(opts *bind.FilterOpts) (*PlasmaChildChainActivatedIterator, error) {

	logs, sub, err := _Plasma.contract.FilterLogs(opts, "ChildChainActivated")
	if err != nil {
		return nil, err
	}
	return &PlasmaChildChainActivatedIterator{contract: _Plasma.contract, event: "ChildChainActivated", logs: logs, sub: sub}, nil
}

// WatchChildChainActivated is a free log subscription operation binding the contract event 0xb8421a1acb5f1e701a4f11ecaad76fa438b3947e8dfd6960b6086130e68e0aed.
//
// Solidity: event ChildChainActivated(address authority)
func (_Plasma *PlasmaFilterer) WatchChildChainActivated(opts *bind.WatchOpts, sink chan<- *PlasmaChildChainActivated) (event.Subscription, error) {

	logs, sub, err := _Plasma.contract.WatchLogs(opts, "ChildChainActivated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PlasmaChildChainActivated)
				if err := _Plasma.contract.UnpackLog(event, "ChildChainActivated", log); err != nil {
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

// ParseChildChainActivated is a log parse operation binding the contract event 0xb8421a1acb5f1e701a4f11ecaad76fa438b3947e8dfd6960b6086130e68e0aed.
//
// Solidity: event ChildChainActivated(address authority)
func (_Plasma *PlasmaFilterer) ParseChildChainActivated(log types.Log) (*PlasmaChildChainActivated, error) {
	event := new(PlasmaChildChainActivated)
	if err := _Plasma.contract.UnpackLog(event, "ChildChainActivated", log); err != nil {
		return nil, err
	}
	return event, nil
}

// PlasmaVaultRegisteredIterator is returned from FilterVaultRegistered and is used to iterate over the raw logs and unpacked data for VaultRegistered events raised by the Plasma contract.
type PlasmaVaultRegisteredIterator struct {
	Event *PlasmaVaultRegistered // Event containing the contract specifics and raw log

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
func (it *PlasmaVaultRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PlasmaVaultRegistered)
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
		it.Event = new(PlasmaVaultRegistered)
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
func (it *PlasmaVaultRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PlasmaVaultRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PlasmaVaultRegistered represents a VaultRegistered event raised by the Plasma contract.
type PlasmaVaultRegistered struct {
	VaultId      *big.Int
	VaultAddress common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterVaultRegistered is a free log retrieval operation binding the contract event 0x7051aac27f9b76ec8f37bd5796f2dcf402bd840a1c45c952a6eeb0e11bde0996.
//
// Solidity: event VaultRegistered(uint256 vaultId, address vaultAddress)
func (_Plasma *PlasmaFilterer) FilterVaultRegistered(opts *bind.FilterOpts) (*PlasmaVaultRegisteredIterator, error) {

	logs, sub, err := _Plasma.contract.FilterLogs(opts, "VaultRegistered")
	if err != nil {
		return nil, err
	}
	return &PlasmaVaultRegisteredIterator{contract: _Plasma.contract, event: "VaultRegistered", logs: logs, sub: sub}, nil
}

// WatchVaultRegistered is a free log subscription operation binding the contract event 0x7051aac27f9b76ec8f37bd5796f2dcf402bd840a1c45c952a6eeb0e11bde0996.
//
// Solidity: event VaultRegistered(uint256 vaultId, address vaultAddress)
func (_Plasma *PlasmaFilterer) WatchVaultRegistered(opts *bind.WatchOpts, sink chan<- *PlasmaVaultRegistered) (event.Subscription, error) {

	logs, sub, err := _Plasma.contract.WatchLogs(opts, "VaultRegistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PlasmaVaultRegistered)
				if err := _Plasma.contract.UnpackLog(event, "VaultRegistered", log); err != nil {
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

// ParseVaultRegistered is a log parse operation binding the contract event 0x7051aac27f9b76ec8f37bd5796f2dcf402bd840a1c45c952a6eeb0e11bde0996.
//
// Solidity: event VaultRegistered(uint256 vaultId, address vaultAddress)
func (_Plasma *PlasmaFilterer) ParseVaultRegistered(log types.Log) (*PlasmaVaultRegistered, error) {
	event := new(PlasmaVaultRegistered)
	if err := _Plasma.contract.UnpackLog(event, "VaultRegistered", log); err != nil {
		return nil, err
	}
	return event, nil
}
