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

// FaultDisputeGameMetaData contains all meta data concerning the FaultDisputeGame contract.
var FaultDisputeGameMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"GameType\",\"name\":\"_gameType\",\"type\":\"uint32\"},{\"internalType\":\"Claim\",\"name\":\"_absolutePrestate\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_maxGameDepth\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_splitDepth\",\"type\":\"uint256\"},{\"internalType\":\"Duration\",\"name\":\"_clockExtension\",\"type\":\"uint64\"},{\"internalType\":\"Duration\",\"name\":\"_maxClockDuration\",\"type\":\"uint64\"},{\"internalType\":\"contractIBigStepper\",\"name\":\"_vm\",\"type\":\"address\"},{\"internalType\":\"contractIDelayedWETH\",\"name\":\"_weth\",\"type\":\"address\"},{\"internalType\":\"contractIAnchorStateRegistry\",\"name\":\"_anchorStateRegistry\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_l2ChainId\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"absolutePrestate\",\"outputs\":[{\"internalType\":\"Claim\",\"name\":\"absolutePrestate_\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_ident\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_execLeafIdx\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_partOffset\",\"type\":\"uint256\"}],\"name\":\"addLocalData\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"anchorStateRegistry\",\"outputs\":[{\"internalType\":\"contractIAnchorStateRegistry\",\"name\":\"registry_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Claim\",\"name\":\"_disputed\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_parentIndex\",\"type\":\"uint256\"},{\"internalType\":\"Claim\",\"name\":\"_claim\",\"type\":\"bytes32\"}],\"name\":\"attack\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"version\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"messagePasserStorageRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"latestBlockhash\",\"type\":\"bytes32\"}],\"internalType\":\"structTypes.OutputRootProof\",\"name\":\"_outputRootProof\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"_headerRLP\",\"type\":\"bytes\"}],\"name\":\"challengeRootL2Block\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_recipient\",\"type\":\"address\"}],\"name\":\"claimCredit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"claimData\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"parentIndex\",\"type\":\"uint32\"},{\"internalType\":\"address\",\"name\":\"counteredBy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"claimant\",\"type\":\"address\"},{\"internalType\":\"uint128\",\"name\":\"bond\",\"type\":\"uint128\"},{\"internalType\":\"Claim\",\"name\":\"claim\",\"type\":\"bytes32\"},{\"internalType\":\"Position\",\"name\":\"position\",\"type\":\"uint128\"},{\"internalType\":\"Clock\",\"name\":\"clock\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claimDataLen\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"len_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Hash\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"claims\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"clockExtension\",\"outputs\":[{\"internalType\":\"Duration\",\"name\":\"clockExtension_\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"createdAt\",\"outputs\":[{\"internalType\":\"Timestamp\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"credit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Claim\",\"name\":\"_disputed\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_parentIndex\",\"type\":\"uint256\"},{\"internalType\":\"Claim\",\"name\":\"_claim\",\"type\":\"bytes32\"}],\"name\":\"defend\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"extraData\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"extraData_\",\"type\":\"bytes\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gameCreator\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"creator_\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gameData\",\"outputs\":[{\"internalType\":\"GameType\",\"name\":\"gameType_\",\"type\":\"uint32\"},{\"internalType\":\"Claim\",\"name\":\"rootClaim_\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"extraData_\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gameType\",\"outputs\":[{\"internalType\":\"GameType\",\"name\":\"gameType_\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_claimIndex\",\"type\":\"uint256\"}],\"name\":\"getChallengerDuration\",\"outputs\":[{\"internalType\":\"Duration\",\"name\":\"duration_\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_claimIndex\",\"type\":\"uint256\"}],\"name\":\"getNumToResolve\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"numRemainingChildren_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Position\",\"name\":\"_position\",\"type\":\"uint128\"}],\"name\":\"getRequiredBond\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"requiredBond_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1Head\",\"outputs\":[{\"internalType\":\"Hash\",\"name\":\"l1Head_\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2BlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"l2BlockNumber_\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2BlockNumberChallenged\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2BlockNumberChallenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"l2ChainId_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxClockDuration\",\"outputs\":[{\"internalType\":\"Duration\",\"name\":\"maxClockDuration_\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxGameDepth\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxGameDepth_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Claim\",\"name\":\"_disputed\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_challengeIndex\",\"type\":\"uint256\"},{\"internalType\":\"Claim\",\"name\":\"_claim\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"_isAttack\",\"type\":\"bool\"}],\"name\":\"move\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"resolutionCheckpoints\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"initialCheckpointComplete\",\"type\":\"bool\"},{\"internalType\":\"uint32\",\"name\":\"subgameIndex\",\"type\":\"uint32\"},{\"internalType\":\"Position\",\"name\":\"leftmostPosition\",\"type\":\"uint128\"},{\"internalType\":\"address\",\"name\":\"counteredBy\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"resolve\",\"outputs\":[{\"internalType\":\"enumGameStatus\",\"name\":\"status_\",\"type\":\"uint8\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_claimIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_numToResolve\",\"type\":\"uint256\"}],\"name\":\"resolveClaim\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"resolvedAt\",\"outputs\":[{\"internalType\":\"Timestamp\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"resolvedSubgames\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rootClaim\",\"outputs\":[{\"internalType\":\"Claim\",\"name\":\"rootClaim_\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"splitDepth\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"splitDepth_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startingBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"startingBlockNumber_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startingOutputRoot\",\"outputs\":[{\"internalType\":\"Hash\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"l2BlockNumber\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startingRootHash\",\"outputs\":[{\"internalType\":\"Hash\",\"name\":\"startingRootHash_\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"status\",\"outputs\":[{\"internalType\":\"enumGameStatus\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_claimIndex\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"_isAttack\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"_stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_proof\",\"type\":\"bytes\"}],\"name\":\"step\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"subgames\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"vm\",\"outputs\":[{\"internalType\":\"contractIBigStepper\",\"name\":\"vm_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"weth\",\"outputs\":[{\"internalType\":\"contractIDelayedWETH\",\"name\":\"weth_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"parentIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"Claim\",\"name\":\"claim\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"claimant\",\"type\":\"address\"}],\"name\":\"Move\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"enumGameStatus\",\"name\":\"status\",\"type\":\"uint8\"}],\"name\":\"Resolved\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"AlreadyInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AnchorRootNotFound\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BlockNumberMatches\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BondTransferFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"CannotDefendRootClaim\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ClaimAboveSplit\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ClaimAlreadyExists\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ClaimAlreadyResolved\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ClockNotExpired\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ClockTimeExceeded\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ContentLengthMismatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DuplicateStep\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyItem\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"GameDepthExceeded\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"GameNotInProgress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"IncorrectBondAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidClockExtension\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidDataRemainder\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidDisputedClaimIndex\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidHeader\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidHeaderRLP\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidLocalIdent\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidOutputRootProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidParent\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidPrestate\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSplitDepth\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"L2BlockNumberChallenged\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MaxDepthTooLarge\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NoCreditToClaim\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OutOfOrderResolution\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnexpectedList\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"Claim\",\"name\":\"rootClaim\",\"type\":\"bytes32\"}],\"name\":\"UnexpectedRootClaim\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnexpectedString\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ValidStep\",\"type\":\"error\"}]",
}

// FaultDisputeGameABI is the input ABI used to generate the binding from.
// Deprecated: Use FaultDisputeGameMetaData.ABI instead.
var FaultDisputeGameABI = FaultDisputeGameMetaData.ABI

// FaultDisputeGame is an auto generated Go binding around an Ethereum contract.
type FaultDisputeGame struct {
	FaultDisputeGameCaller     // Read-only binding to the contract
	FaultDisputeGameTransactor // Write-only binding to the contract
	FaultDisputeGameFilterer   // Log filterer for contract events
}

// FaultDisputeGameCaller is an auto generated read-only Go binding around an Ethereum contract.
type FaultDisputeGameCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FaultDisputeGameTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FaultDisputeGameTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FaultDisputeGameFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FaultDisputeGameFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FaultDisputeGameSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FaultDisputeGameSession struct {
	Contract     *FaultDisputeGame // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// FaultDisputeGameCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FaultDisputeGameCallerSession struct {
	Contract *FaultDisputeGameCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// FaultDisputeGameTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FaultDisputeGameTransactorSession struct {
	Contract     *FaultDisputeGameTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// FaultDisputeGameRaw is an auto generated low-level Go binding around an Ethereum contract.
type FaultDisputeGameRaw struct {
	Contract *FaultDisputeGame // Generic contract binding to access the raw methods on
}

// FaultDisputeGameCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FaultDisputeGameCallerRaw struct {
	Contract *FaultDisputeGameCaller // Generic read-only contract binding to access the raw methods on
}

// FaultDisputeGameTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FaultDisputeGameTransactorRaw struct {
	Contract *FaultDisputeGameTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFaultDisputeGame creates a new instance of FaultDisputeGame, bound to a specific deployed contract.
func NewFaultDisputeGame(address common.Address, backend bind.ContractBackend) (*FaultDisputeGame, error) {
	contract, err := bindFaultDisputeGame(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGame{FaultDisputeGameCaller: FaultDisputeGameCaller{contract: contract}, FaultDisputeGameTransactor: FaultDisputeGameTransactor{contract: contract}, FaultDisputeGameFilterer: FaultDisputeGameFilterer{contract: contract}}, nil
}

// NewFaultDisputeGameCaller creates a new read-only instance of FaultDisputeGame, bound to a specific deployed contract.
func NewFaultDisputeGameCaller(address common.Address, caller bind.ContractCaller) (*FaultDisputeGameCaller, error) {
	contract, err := bindFaultDisputeGame(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameCaller{contract: contract}, nil
}

// NewFaultDisputeGameTransactor creates a new write-only instance of FaultDisputeGame, bound to a specific deployed contract.
func NewFaultDisputeGameTransactor(address common.Address, transactor bind.ContractTransactor) (*FaultDisputeGameTransactor, error) {
	contract, err := bindFaultDisputeGame(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameTransactor{contract: contract}, nil
}

// NewFaultDisputeGameFilterer creates a new log filterer instance of FaultDisputeGame, bound to a specific deployed contract.
func NewFaultDisputeGameFilterer(address common.Address, filterer bind.ContractFilterer) (*FaultDisputeGameFilterer, error) {
	contract, err := bindFaultDisputeGame(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameFilterer{contract: contract}, nil
}

// bindFaultDisputeGame binds a generic wrapper to an already deployed contract.
func bindFaultDisputeGame(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(FaultDisputeGameABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FaultDisputeGame *FaultDisputeGameRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FaultDisputeGame.Contract.FaultDisputeGameCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FaultDisputeGame *FaultDisputeGameRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.FaultDisputeGameTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FaultDisputeGame *FaultDisputeGameRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.FaultDisputeGameTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FaultDisputeGame *FaultDisputeGameCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FaultDisputeGame.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FaultDisputeGame *FaultDisputeGameTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FaultDisputeGame *FaultDisputeGameTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.contract.Transact(opts, method, params...)
}

// AbsolutePrestate is a free data retrieval call binding the contract method 0x8d450a95.
//
// Solidity: function absolutePrestate() view returns(bytes32 absolutePrestate_)
func (_FaultDisputeGame *FaultDisputeGameCaller) AbsolutePrestate(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "absolutePrestate")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// AbsolutePrestate is a free data retrieval call binding the contract method 0x8d450a95.
//
// Solidity: function absolutePrestate() view returns(bytes32 absolutePrestate_)
func (_FaultDisputeGame *FaultDisputeGameSession) AbsolutePrestate() ([32]byte, error) {
	return _FaultDisputeGame.Contract.AbsolutePrestate(&_FaultDisputeGame.CallOpts)
}

// AbsolutePrestate is a free data retrieval call binding the contract method 0x8d450a95.
//
// Solidity: function absolutePrestate() view returns(bytes32 absolutePrestate_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) AbsolutePrestate() ([32]byte, error) {
	return _FaultDisputeGame.Contract.AbsolutePrestate(&_FaultDisputeGame.CallOpts)
}

// AnchorStateRegistry is a free data retrieval call binding the contract method 0x5c0cba33.
//
// Solidity: function anchorStateRegistry() view returns(address registry_)
func (_FaultDisputeGame *FaultDisputeGameCaller) AnchorStateRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "anchorStateRegistry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AnchorStateRegistry is a free data retrieval call binding the contract method 0x5c0cba33.
//
// Solidity: function anchorStateRegistry() view returns(address registry_)
func (_FaultDisputeGame *FaultDisputeGameSession) AnchorStateRegistry() (common.Address, error) {
	return _FaultDisputeGame.Contract.AnchorStateRegistry(&_FaultDisputeGame.CallOpts)
}

// AnchorStateRegistry is a free data retrieval call binding the contract method 0x5c0cba33.
//
// Solidity: function anchorStateRegistry() view returns(address registry_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) AnchorStateRegistry() (common.Address, error) {
	return _FaultDisputeGame.Contract.AnchorStateRegistry(&_FaultDisputeGame.CallOpts)
}

// ClaimData is a free data retrieval call binding the contract method 0xc6f0308c.
//
// Solidity: function claimData(uint256 ) view returns(uint32 parentIndex, address counteredBy, address claimant, uint128 bond, bytes32 claim, uint128 position, uint128 clock)
func (_FaultDisputeGame *FaultDisputeGameCaller) ClaimData(opts *bind.CallOpts, arg0 *big.Int) (struct {
	ParentIndex uint32
	CounteredBy common.Address
	Claimant    common.Address
	Bond        *big.Int
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "claimData", arg0)

	outstruct := new(struct {
		ParentIndex uint32
		CounteredBy common.Address
		Claimant    common.Address
		Bond        *big.Int
		Claim       [32]byte
		Position    *big.Int
		Clock       *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ParentIndex = *abi.ConvertType(out[0], new(uint32)).(*uint32)
	outstruct.CounteredBy = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Claimant = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.Bond = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Claim = *abi.ConvertType(out[4], new([32]byte)).(*[32]byte)
	outstruct.Position = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.Clock = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// ClaimData is a free data retrieval call binding the contract method 0xc6f0308c.
//
// Solidity: function claimData(uint256 ) view returns(uint32 parentIndex, address counteredBy, address claimant, uint128 bond, bytes32 claim, uint128 position, uint128 clock)
func (_FaultDisputeGame *FaultDisputeGameSession) ClaimData(arg0 *big.Int) (struct {
	ParentIndex uint32
	CounteredBy common.Address
	Claimant    common.Address
	Bond        *big.Int
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}, error) {
	return _FaultDisputeGame.Contract.ClaimData(&_FaultDisputeGame.CallOpts, arg0)
}

// ClaimData is a free data retrieval call binding the contract method 0xc6f0308c.
//
// Solidity: function claimData(uint256 ) view returns(uint32 parentIndex, address counteredBy, address claimant, uint128 bond, bytes32 claim, uint128 position, uint128 clock)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ClaimData(arg0 *big.Int) (struct {
	ParentIndex uint32
	CounteredBy common.Address
	Claimant    common.Address
	Bond        *big.Int
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}, error) {
	return _FaultDisputeGame.Contract.ClaimData(&_FaultDisputeGame.CallOpts, arg0)
}

// ClaimDataLen is a free data retrieval call binding the contract method 0x8980e0cc.
//
// Solidity: function claimDataLen() view returns(uint256 len_)
func (_FaultDisputeGame *FaultDisputeGameCaller) ClaimDataLen(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "claimDataLen")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ClaimDataLen is a free data retrieval call binding the contract method 0x8980e0cc.
//
// Solidity: function claimDataLen() view returns(uint256 len_)
func (_FaultDisputeGame *FaultDisputeGameSession) ClaimDataLen() (*big.Int, error) {
	return _FaultDisputeGame.Contract.ClaimDataLen(&_FaultDisputeGame.CallOpts)
}

// ClaimDataLen is a free data retrieval call binding the contract method 0x8980e0cc.
//
// Solidity: function claimDataLen() view returns(uint256 len_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ClaimDataLen() (*big.Int, error) {
	return _FaultDisputeGame.Contract.ClaimDataLen(&_FaultDisputeGame.CallOpts)
}

// Claims is a free data retrieval call binding the contract method 0xeff0f592.
//
// Solidity: function claims(bytes32 ) view returns(bool)
func (_FaultDisputeGame *FaultDisputeGameCaller) Claims(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "claims", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Claims is a free data retrieval call binding the contract method 0xeff0f592.
//
// Solidity: function claims(bytes32 ) view returns(bool)
func (_FaultDisputeGame *FaultDisputeGameSession) Claims(arg0 [32]byte) (bool, error) {
	return _FaultDisputeGame.Contract.Claims(&_FaultDisputeGame.CallOpts, arg0)
}

// Claims is a free data retrieval call binding the contract method 0xeff0f592.
//
// Solidity: function claims(bytes32 ) view returns(bool)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) Claims(arg0 [32]byte) (bool, error) {
	return _FaultDisputeGame.Contract.Claims(&_FaultDisputeGame.CallOpts, arg0)
}

// ClockExtension is a free data retrieval call binding the contract method 0x6b6716c0.
//
// Solidity: function clockExtension() view returns(uint64 clockExtension_)
func (_FaultDisputeGame *FaultDisputeGameCaller) ClockExtension(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "clockExtension")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// ClockExtension is a free data retrieval call binding the contract method 0x6b6716c0.
//
// Solidity: function clockExtension() view returns(uint64 clockExtension_)
func (_FaultDisputeGame *FaultDisputeGameSession) ClockExtension() (uint64, error) {
	return _FaultDisputeGame.Contract.ClockExtension(&_FaultDisputeGame.CallOpts)
}

// ClockExtension is a free data retrieval call binding the contract method 0x6b6716c0.
//
// Solidity: function clockExtension() view returns(uint64 clockExtension_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ClockExtension() (uint64, error) {
	return _FaultDisputeGame.Contract.ClockExtension(&_FaultDisputeGame.CallOpts)
}

// CreatedAt is a free data retrieval call binding the contract method 0xcf09e0d0.
//
// Solidity: function createdAt() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameCaller) CreatedAt(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "createdAt")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// CreatedAt is a free data retrieval call binding the contract method 0xcf09e0d0.
//
// Solidity: function createdAt() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameSession) CreatedAt() (uint64, error) {
	return _FaultDisputeGame.Contract.CreatedAt(&_FaultDisputeGame.CallOpts)
}

// CreatedAt is a free data retrieval call binding the contract method 0xcf09e0d0.
//
// Solidity: function createdAt() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) CreatedAt() (uint64, error) {
	return _FaultDisputeGame.Contract.CreatedAt(&_FaultDisputeGame.CallOpts)
}

// Credit is a free data retrieval call binding the contract method 0xd5d44d80.
//
// Solidity: function credit(address ) view returns(uint256)
func (_FaultDisputeGame *FaultDisputeGameCaller) Credit(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "credit", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Credit is a free data retrieval call binding the contract method 0xd5d44d80.
//
// Solidity: function credit(address ) view returns(uint256)
func (_FaultDisputeGame *FaultDisputeGameSession) Credit(arg0 common.Address) (*big.Int, error) {
	return _FaultDisputeGame.Contract.Credit(&_FaultDisputeGame.CallOpts, arg0)
}

// Credit is a free data retrieval call binding the contract method 0xd5d44d80.
//
// Solidity: function credit(address ) view returns(uint256)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) Credit(arg0 common.Address) (*big.Int, error) {
	return _FaultDisputeGame.Contract.Credit(&_FaultDisputeGame.CallOpts, arg0)
}

// ExtraData is a free data retrieval call binding the contract method 0x609d3334.
//
// Solidity: function extraData() pure returns(bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameCaller) ExtraData(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "extraData")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// ExtraData is a free data retrieval call binding the contract method 0x609d3334.
//
// Solidity: function extraData() pure returns(bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameSession) ExtraData() ([]byte, error) {
	return _FaultDisputeGame.Contract.ExtraData(&_FaultDisputeGame.CallOpts)
}

// ExtraData is a free data retrieval call binding the contract method 0x609d3334.
//
// Solidity: function extraData() pure returns(bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ExtraData() ([]byte, error) {
	return _FaultDisputeGame.Contract.ExtraData(&_FaultDisputeGame.CallOpts)
}

// GameCreator is a free data retrieval call binding the contract method 0x37b1b229.
//
// Solidity: function gameCreator() pure returns(address creator_)
func (_FaultDisputeGame *FaultDisputeGameCaller) GameCreator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "gameCreator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GameCreator is a free data retrieval call binding the contract method 0x37b1b229.
//
// Solidity: function gameCreator() pure returns(address creator_)
func (_FaultDisputeGame *FaultDisputeGameSession) GameCreator() (common.Address, error) {
	return _FaultDisputeGame.Contract.GameCreator(&_FaultDisputeGame.CallOpts)
}

// GameCreator is a free data retrieval call binding the contract method 0x37b1b229.
//
// Solidity: function gameCreator() pure returns(address creator_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) GameCreator() (common.Address, error) {
	return _FaultDisputeGame.Contract.GameCreator(&_FaultDisputeGame.CallOpts)
}

// GameData is a free data retrieval call binding the contract method 0xfa24f743.
//
// Solidity: function gameData() view returns(uint32 gameType_, bytes32 rootClaim_, bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameCaller) GameData(opts *bind.CallOpts) (struct {
	GameType  uint32
	RootClaim [32]byte
	ExtraData []byte
}, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "gameData")

	outstruct := new(struct {
		GameType  uint32
		RootClaim [32]byte
		ExtraData []byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.GameType = *abi.ConvertType(out[0], new(uint32)).(*uint32)
	outstruct.RootClaim = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.ExtraData = *abi.ConvertType(out[2], new([]byte)).(*[]byte)

	return *outstruct, err

}

// GameData is a free data retrieval call binding the contract method 0xfa24f743.
//
// Solidity: function gameData() view returns(uint32 gameType_, bytes32 rootClaim_, bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameSession) GameData() (struct {
	GameType  uint32
	RootClaim [32]byte
	ExtraData []byte
}, error) {
	return _FaultDisputeGame.Contract.GameData(&_FaultDisputeGame.CallOpts)
}

// GameData is a free data retrieval call binding the contract method 0xfa24f743.
//
// Solidity: function gameData() view returns(uint32 gameType_, bytes32 rootClaim_, bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) GameData() (struct {
	GameType  uint32
	RootClaim [32]byte
	ExtraData []byte
}, error) {
	return _FaultDisputeGame.Contract.GameData(&_FaultDisputeGame.CallOpts)
}

// GameType is a free data retrieval call binding the contract method 0xbbdc02db.
//
// Solidity: function gameType() view returns(uint32 gameType_)
func (_FaultDisputeGame *FaultDisputeGameCaller) GameType(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "gameType")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// GameType is a free data retrieval call binding the contract method 0xbbdc02db.
//
// Solidity: function gameType() view returns(uint32 gameType_)
func (_FaultDisputeGame *FaultDisputeGameSession) GameType() (uint32, error) {
	return _FaultDisputeGame.Contract.GameType(&_FaultDisputeGame.CallOpts)
}

// GameType is a free data retrieval call binding the contract method 0xbbdc02db.
//
// Solidity: function gameType() view returns(uint32 gameType_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) GameType() (uint32, error) {
	return _FaultDisputeGame.Contract.GameType(&_FaultDisputeGame.CallOpts)
}

// GetChallengerDuration is a free data retrieval call binding the contract method 0xbd8da956.
//
// Solidity: function getChallengerDuration(uint256 _claimIndex) view returns(uint64 duration_)
func (_FaultDisputeGame *FaultDisputeGameCaller) GetChallengerDuration(opts *bind.CallOpts, _claimIndex *big.Int) (uint64, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "getChallengerDuration", _claimIndex)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetChallengerDuration is a free data retrieval call binding the contract method 0xbd8da956.
//
// Solidity: function getChallengerDuration(uint256 _claimIndex) view returns(uint64 duration_)
func (_FaultDisputeGame *FaultDisputeGameSession) GetChallengerDuration(_claimIndex *big.Int) (uint64, error) {
	return _FaultDisputeGame.Contract.GetChallengerDuration(&_FaultDisputeGame.CallOpts, _claimIndex)
}

// GetChallengerDuration is a free data retrieval call binding the contract method 0xbd8da956.
//
// Solidity: function getChallengerDuration(uint256 _claimIndex) view returns(uint64 duration_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) GetChallengerDuration(_claimIndex *big.Int) (uint64, error) {
	return _FaultDisputeGame.Contract.GetChallengerDuration(&_FaultDisputeGame.CallOpts, _claimIndex)
}

// GetNumToResolve is a free data retrieval call binding the contract method 0x5a5fa2d9.
//
// Solidity: function getNumToResolve(uint256 _claimIndex) view returns(uint256 numRemainingChildren_)
func (_FaultDisputeGame *FaultDisputeGameCaller) GetNumToResolve(opts *bind.CallOpts, _claimIndex *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "getNumToResolve", _claimIndex)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNumToResolve is a free data retrieval call binding the contract method 0x5a5fa2d9.
//
// Solidity: function getNumToResolve(uint256 _claimIndex) view returns(uint256 numRemainingChildren_)
func (_FaultDisputeGame *FaultDisputeGameSession) GetNumToResolve(_claimIndex *big.Int) (*big.Int, error) {
	return _FaultDisputeGame.Contract.GetNumToResolve(&_FaultDisputeGame.CallOpts, _claimIndex)
}

// GetNumToResolve is a free data retrieval call binding the contract method 0x5a5fa2d9.
//
// Solidity: function getNumToResolve(uint256 _claimIndex) view returns(uint256 numRemainingChildren_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) GetNumToResolve(_claimIndex *big.Int) (*big.Int, error) {
	return _FaultDisputeGame.Contract.GetNumToResolve(&_FaultDisputeGame.CallOpts, _claimIndex)
}

// GetRequiredBond is a free data retrieval call binding the contract method 0xc395e1ca.
//
// Solidity: function getRequiredBond(uint128 _position) view returns(uint256 requiredBond_)
func (_FaultDisputeGame *FaultDisputeGameCaller) GetRequiredBond(opts *bind.CallOpts, _position *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "getRequiredBond", _position)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRequiredBond is a free data retrieval call binding the contract method 0xc395e1ca.
//
// Solidity: function getRequiredBond(uint128 _position) view returns(uint256 requiredBond_)
func (_FaultDisputeGame *FaultDisputeGameSession) GetRequiredBond(_position *big.Int) (*big.Int, error) {
	return _FaultDisputeGame.Contract.GetRequiredBond(&_FaultDisputeGame.CallOpts, _position)
}

// GetRequiredBond is a free data retrieval call binding the contract method 0xc395e1ca.
//
// Solidity: function getRequiredBond(uint128 _position) view returns(uint256 requiredBond_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) GetRequiredBond(_position *big.Int) (*big.Int, error) {
	return _FaultDisputeGame.Contract.GetRequiredBond(&_FaultDisputeGame.CallOpts, _position)
}

// L1Head is a free data retrieval call binding the contract method 0x6361506d.
//
// Solidity: function l1Head() pure returns(bytes32 l1Head_)
func (_FaultDisputeGame *FaultDisputeGameCaller) L1Head(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "l1Head")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L1Head is a free data retrieval call binding the contract method 0x6361506d.
//
// Solidity: function l1Head() pure returns(bytes32 l1Head_)
func (_FaultDisputeGame *FaultDisputeGameSession) L1Head() ([32]byte, error) {
	return _FaultDisputeGame.Contract.L1Head(&_FaultDisputeGame.CallOpts)
}

// L1Head is a free data retrieval call binding the contract method 0x6361506d.
//
// Solidity: function l1Head() pure returns(bytes32 l1Head_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) L1Head() ([32]byte, error) {
	return _FaultDisputeGame.Contract.L1Head(&_FaultDisputeGame.CallOpts)
}

// L2BlockNumber is a free data retrieval call binding the contract method 0x8b85902b.
//
// Solidity: function l2BlockNumber() pure returns(uint256 l2BlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameCaller) L2BlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "l2BlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2BlockNumber is a free data retrieval call binding the contract method 0x8b85902b.
//
// Solidity: function l2BlockNumber() pure returns(uint256 l2BlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameSession) L2BlockNumber() (*big.Int, error) {
	return _FaultDisputeGame.Contract.L2BlockNumber(&_FaultDisputeGame.CallOpts)
}

// L2BlockNumber is a free data retrieval call binding the contract method 0x8b85902b.
//
// Solidity: function l2BlockNumber() pure returns(uint256 l2BlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) L2BlockNumber() (*big.Int, error) {
	return _FaultDisputeGame.Contract.L2BlockNumber(&_FaultDisputeGame.CallOpts)
}

// L2BlockNumberChallenged is a free data retrieval call binding the contract method 0x3e3ac912.
//
// Solidity: function l2BlockNumberChallenged() view returns(bool)
func (_FaultDisputeGame *FaultDisputeGameCaller) L2BlockNumberChallenged(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "l2BlockNumberChallenged")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// L2BlockNumberChallenged is a free data retrieval call binding the contract method 0x3e3ac912.
//
// Solidity: function l2BlockNumberChallenged() view returns(bool)
func (_FaultDisputeGame *FaultDisputeGameSession) L2BlockNumberChallenged() (bool, error) {
	return _FaultDisputeGame.Contract.L2BlockNumberChallenged(&_FaultDisputeGame.CallOpts)
}

// L2BlockNumberChallenged is a free data retrieval call binding the contract method 0x3e3ac912.
//
// Solidity: function l2BlockNumberChallenged() view returns(bool)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) L2BlockNumberChallenged() (bool, error) {
	return _FaultDisputeGame.Contract.L2BlockNumberChallenged(&_FaultDisputeGame.CallOpts)
}

// L2BlockNumberChallenger is a free data retrieval call binding the contract method 0x30dbe570.
//
// Solidity: function l2BlockNumberChallenger() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameCaller) L2BlockNumberChallenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "l2BlockNumberChallenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2BlockNumberChallenger is a free data retrieval call binding the contract method 0x30dbe570.
//
// Solidity: function l2BlockNumberChallenger() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameSession) L2BlockNumberChallenger() (common.Address, error) {
	return _FaultDisputeGame.Contract.L2BlockNumberChallenger(&_FaultDisputeGame.CallOpts)
}

// L2BlockNumberChallenger is a free data retrieval call binding the contract method 0x30dbe570.
//
// Solidity: function l2BlockNumberChallenger() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) L2BlockNumberChallenger() (common.Address, error) {
	return _FaultDisputeGame.Contract.L2BlockNumberChallenger(&_FaultDisputeGame.CallOpts)
}

// L2ChainId is a free data retrieval call binding the contract method 0xd6ae3cd5.
//
// Solidity: function l2ChainId() view returns(uint256 l2ChainId_)
func (_FaultDisputeGame *FaultDisputeGameCaller) L2ChainId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "l2ChainId")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ChainId is a free data retrieval call binding the contract method 0xd6ae3cd5.
//
// Solidity: function l2ChainId() view returns(uint256 l2ChainId_)
func (_FaultDisputeGame *FaultDisputeGameSession) L2ChainId() (*big.Int, error) {
	return _FaultDisputeGame.Contract.L2ChainId(&_FaultDisputeGame.CallOpts)
}

// L2ChainId is a free data retrieval call binding the contract method 0xd6ae3cd5.
//
// Solidity: function l2ChainId() view returns(uint256 l2ChainId_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) L2ChainId() (*big.Int, error) {
	return _FaultDisputeGame.Contract.L2ChainId(&_FaultDisputeGame.CallOpts)
}

// MaxClockDuration is a free data retrieval call binding the contract method 0xdabd396d.
//
// Solidity: function maxClockDuration() view returns(uint64 maxClockDuration_)
func (_FaultDisputeGame *FaultDisputeGameCaller) MaxClockDuration(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "maxClockDuration")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MaxClockDuration is a free data retrieval call binding the contract method 0xdabd396d.
//
// Solidity: function maxClockDuration() view returns(uint64 maxClockDuration_)
func (_FaultDisputeGame *FaultDisputeGameSession) MaxClockDuration() (uint64, error) {
	return _FaultDisputeGame.Contract.MaxClockDuration(&_FaultDisputeGame.CallOpts)
}

// MaxClockDuration is a free data retrieval call binding the contract method 0xdabd396d.
//
// Solidity: function maxClockDuration() view returns(uint64 maxClockDuration_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) MaxClockDuration() (uint64, error) {
	return _FaultDisputeGame.Contract.MaxClockDuration(&_FaultDisputeGame.CallOpts)
}

// MaxGameDepth is a free data retrieval call binding the contract method 0xfa315aa9.
//
// Solidity: function maxGameDepth() view returns(uint256 maxGameDepth_)
func (_FaultDisputeGame *FaultDisputeGameCaller) MaxGameDepth(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "maxGameDepth")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxGameDepth is a free data retrieval call binding the contract method 0xfa315aa9.
//
// Solidity: function maxGameDepth() view returns(uint256 maxGameDepth_)
func (_FaultDisputeGame *FaultDisputeGameSession) MaxGameDepth() (*big.Int, error) {
	return _FaultDisputeGame.Contract.MaxGameDepth(&_FaultDisputeGame.CallOpts)
}

// MaxGameDepth is a free data retrieval call binding the contract method 0xfa315aa9.
//
// Solidity: function maxGameDepth() view returns(uint256 maxGameDepth_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) MaxGameDepth() (*big.Int, error) {
	return _FaultDisputeGame.Contract.MaxGameDepth(&_FaultDisputeGame.CallOpts)
}

// ResolutionCheckpoints is a free data retrieval call binding the contract method 0xa445ece6.
//
// Solidity: function resolutionCheckpoints(uint256 ) view returns(bool initialCheckpointComplete, uint32 subgameIndex, uint128 leftmostPosition, address counteredBy)
func (_FaultDisputeGame *FaultDisputeGameCaller) ResolutionCheckpoints(opts *bind.CallOpts, arg0 *big.Int) (struct {
	InitialCheckpointComplete bool
	SubgameIndex              uint32
	LeftmostPosition          *big.Int
	CounteredBy               common.Address
}, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "resolutionCheckpoints", arg0)

	outstruct := new(struct {
		InitialCheckpointComplete bool
		SubgameIndex              uint32
		LeftmostPosition          *big.Int
		CounteredBy               common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.InitialCheckpointComplete = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.SubgameIndex = *abi.ConvertType(out[1], new(uint32)).(*uint32)
	outstruct.LeftmostPosition = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.CounteredBy = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// ResolutionCheckpoints is a free data retrieval call binding the contract method 0xa445ece6.
//
// Solidity: function resolutionCheckpoints(uint256 ) view returns(bool initialCheckpointComplete, uint32 subgameIndex, uint128 leftmostPosition, address counteredBy)
func (_FaultDisputeGame *FaultDisputeGameSession) ResolutionCheckpoints(arg0 *big.Int) (struct {
	InitialCheckpointComplete bool
	SubgameIndex              uint32
	LeftmostPosition          *big.Int
	CounteredBy               common.Address
}, error) {
	return _FaultDisputeGame.Contract.ResolutionCheckpoints(&_FaultDisputeGame.CallOpts, arg0)
}

// ResolutionCheckpoints is a free data retrieval call binding the contract method 0xa445ece6.
//
// Solidity: function resolutionCheckpoints(uint256 ) view returns(bool initialCheckpointComplete, uint32 subgameIndex, uint128 leftmostPosition, address counteredBy)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ResolutionCheckpoints(arg0 *big.Int) (struct {
	InitialCheckpointComplete bool
	SubgameIndex              uint32
	LeftmostPosition          *big.Int
	CounteredBy               common.Address
}, error) {
	return _FaultDisputeGame.Contract.ResolutionCheckpoints(&_FaultDisputeGame.CallOpts, arg0)
}

// ResolvedAt is a free data retrieval call binding the contract method 0x19effeb4.
//
// Solidity: function resolvedAt() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameCaller) ResolvedAt(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "resolvedAt")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// ResolvedAt is a free data retrieval call binding the contract method 0x19effeb4.
//
// Solidity: function resolvedAt() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameSession) ResolvedAt() (uint64, error) {
	return _FaultDisputeGame.Contract.ResolvedAt(&_FaultDisputeGame.CallOpts)
}

// ResolvedAt is a free data retrieval call binding the contract method 0x19effeb4.
//
// Solidity: function resolvedAt() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ResolvedAt() (uint64, error) {
	return _FaultDisputeGame.Contract.ResolvedAt(&_FaultDisputeGame.CallOpts)
}

// ResolvedSubgames is a free data retrieval call binding the contract method 0xfe2bbeb2.
//
// Solidity: function resolvedSubgames(uint256 ) view returns(bool)
func (_FaultDisputeGame *FaultDisputeGameCaller) ResolvedSubgames(opts *bind.CallOpts, arg0 *big.Int) (bool, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "resolvedSubgames", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ResolvedSubgames is a free data retrieval call binding the contract method 0xfe2bbeb2.
//
// Solidity: function resolvedSubgames(uint256 ) view returns(bool)
func (_FaultDisputeGame *FaultDisputeGameSession) ResolvedSubgames(arg0 *big.Int) (bool, error) {
	return _FaultDisputeGame.Contract.ResolvedSubgames(&_FaultDisputeGame.CallOpts, arg0)
}

// ResolvedSubgames is a free data retrieval call binding the contract method 0xfe2bbeb2.
//
// Solidity: function resolvedSubgames(uint256 ) view returns(bool)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ResolvedSubgames(arg0 *big.Int) (bool, error) {
	return _FaultDisputeGame.Contract.ResolvedSubgames(&_FaultDisputeGame.CallOpts, arg0)
}

// RootClaim is a free data retrieval call binding the contract method 0xbcef3b55.
//
// Solidity: function rootClaim() pure returns(bytes32 rootClaim_)
func (_FaultDisputeGame *FaultDisputeGameCaller) RootClaim(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "rootClaim")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RootClaim is a free data retrieval call binding the contract method 0xbcef3b55.
//
// Solidity: function rootClaim() pure returns(bytes32 rootClaim_)
func (_FaultDisputeGame *FaultDisputeGameSession) RootClaim() ([32]byte, error) {
	return _FaultDisputeGame.Contract.RootClaim(&_FaultDisputeGame.CallOpts)
}

// RootClaim is a free data retrieval call binding the contract method 0xbcef3b55.
//
// Solidity: function rootClaim() pure returns(bytes32 rootClaim_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) RootClaim() ([32]byte, error) {
	return _FaultDisputeGame.Contract.RootClaim(&_FaultDisputeGame.CallOpts)
}

// SplitDepth is a free data retrieval call binding the contract method 0xec5e6308.
//
// Solidity: function splitDepth() view returns(uint256 splitDepth_)
func (_FaultDisputeGame *FaultDisputeGameCaller) SplitDepth(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "splitDepth")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SplitDepth is a free data retrieval call binding the contract method 0xec5e6308.
//
// Solidity: function splitDepth() view returns(uint256 splitDepth_)
func (_FaultDisputeGame *FaultDisputeGameSession) SplitDepth() (*big.Int, error) {
	return _FaultDisputeGame.Contract.SplitDepth(&_FaultDisputeGame.CallOpts)
}

// SplitDepth is a free data retrieval call binding the contract method 0xec5e6308.
//
// Solidity: function splitDepth() view returns(uint256 splitDepth_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) SplitDepth() (*big.Int, error) {
	return _FaultDisputeGame.Contract.SplitDepth(&_FaultDisputeGame.CallOpts)
}

// StartingBlockNumber is a free data retrieval call binding the contract method 0x70872aa5.
//
// Solidity: function startingBlockNumber() view returns(uint256 startingBlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameCaller) StartingBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "startingBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StartingBlockNumber is a free data retrieval call binding the contract method 0x70872aa5.
//
// Solidity: function startingBlockNumber() view returns(uint256 startingBlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameSession) StartingBlockNumber() (*big.Int, error) {
	return _FaultDisputeGame.Contract.StartingBlockNumber(&_FaultDisputeGame.CallOpts)
}

// StartingBlockNumber is a free data retrieval call binding the contract method 0x70872aa5.
//
// Solidity: function startingBlockNumber() view returns(uint256 startingBlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) StartingBlockNumber() (*big.Int, error) {
	return _FaultDisputeGame.Contract.StartingBlockNumber(&_FaultDisputeGame.CallOpts)
}

// StartingOutputRoot is a free data retrieval call binding the contract method 0x57da950e.
//
// Solidity: function startingOutputRoot() view returns(bytes32 root, uint256 l2BlockNumber)
func (_FaultDisputeGame *FaultDisputeGameCaller) StartingOutputRoot(opts *bind.CallOpts) (struct {
	Root          [32]byte
	L2BlockNumber *big.Int
}, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "startingOutputRoot")

	outstruct := new(struct {
		Root          [32]byte
		L2BlockNumber *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Root = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.L2BlockNumber = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// StartingOutputRoot is a free data retrieval call binding the contract method 0x57da950e.
//
// Solidity: function startingOutputRoot() view returns(bytes32 root, uint256 l2BlockNumber)
func (_FaultDisputeGame *FaultDisputeGameSession) StartingOutputRoot() (struct {
	Root          [32]byte
	L2BlockNumber *big.Int
}, error) {
	return _FaultDisputeGame.Contract.StartingOutputRoot(&_FaultDisputeGame.CallOpts)
}

// StartingOutputRoot is a free data retrieval call binding the contract method 0x57da950e.
//
// Solidity: function startingOutputRoot() view returns(bytes32 root, uint256 l2BlockNumber)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) StartingOutputRoot() (struct {
	Root          [32]byte
	L2BlockNumber *big.Int
}, error) {
	return _FaultDisputeGame.Contract.StartingOutputRoot(&_FaultDisputeGame.CallOpts)
}

// StartingRootHash is a free data retrieval call binding the contract method 0x25fc2ace.
//
// Solidity: function startingRootHash() view returns(bytes32 startingRootHash_)
func (_FaultDisputeGame *FaultDisputeGameCaller) StartingRootHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "startingRootHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// StartingRootHash is a free data retrieval call binding the contract method 0x25fc2ace.
//
// Solidity: function startingRootHash() view returns(bytes32 startingRootHash_)
func (_FaultDisputeGame *FaultDisputeGameSession) StartingRootHash() ([32]byte, error) {
	return _FaultDisputeGame.Contract.StartingRootHash(&_FaultDisputeGame.CallOpts)
}

// StartingRootHash is a free data retrieval call binding the contract method 0x25fc2ace.
//
// Solidity: function startingRootHash() view returns(bytes32 startingRootHash_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) StartingRootHash() ([32]byte, error) {
	return _FaultDisputeGame.Contract.StartingRootHash(&_FaultDisputeGame.CallOpts)
}

// Status is a free data retrieval call binding the contract method 0x200d2ed2.
//
// Solidity: function status() view returns(uint8)
func (_FaultDisputeGame *FaultDisputeGameCaller) Status(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "status")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Status is a free data retrieval call binding the contract method 0x200d2ed2.
//
// Solidity: function status() view returns(uint8)
func (_FaultDisputeGame *FaultDisputeGameSession) Status() (uint8, error) {
	return _FaultDisputeGame.Contract.Status(&_FaultDisputeGame.CallOpts)
}

// Status is a free data retrieval call binding the contract method 0x200d2ed2.
//
// Solidity: function status() view returns(uint8)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) Status() (uint8, error) {
	return _FaultDisputeGame.Contract.Status(&_FaultDisputeGame.CallOpts)
}

// Subgames is a free data retrieval call binding the contract method 0x2ad69aeb.
//
// Solidity: function subgames(uint256 , uint256 ) view returns(uint256)
func (_FaultDisputeGame *FaultDisputeGameCaller) Subgames(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "subgames", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Subgames is a free data retrieval call binding the contract method 0x2ad69aeb.
//
// Solidity: function subgames(uint256 , uint256 ) view returns(uint256)
func (_FaultDisputeGame *FaultDisputeGameSession) Subgames(arg0 *big.Int, arg1 *big.Int) (*big.Int, error) {
	return _FaultDisputeGame.Contract.Subgames(&_FaultDisputeGame.CallOpts, arg0, arg1)
}

// Subgames is a free data retrieval call binding the contract method 0x2ad69aeb.
//
// Solidity: function subgames(uint256 , uint256 ) view returns(uint256)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) Subgames(arg0 *big.Int, arg1 *big.Int) (*big.Int, error) {
	return _FaultDisputeGame.Contract.Subgames(&_FaultDisputeGame.CallOpts, arg0, arg1)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_FaultDisputeGame *FaultDisputeGameCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_FaultDisputeGame *FaultDisputeGameSession) Version() (string, error) {
	return _FaultDisputeGame.Contract.Version(&_FaultDisputeGame.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) Version() (string, error) {
	return _FaultDisputeGame.Contract.Version(&_FaultDisputeGame.CallOpts)
}

// Vm is a free data retrieval call binding the contract method 0x3a768463.
//
// Solidity: function vm() view returns(address vm_)
func (_FaultDisputeGame *FaultDisputeGameCaller) Vm(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "vm")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Vm is a free data retrieval call binding the contract method 0x3a768463.
//
// Solidity: function vm() view returns(address vm_)
func (_FaultDisputeGame *FaultDisputeGameSession) Vm() (common.Address, error) {
	return _FaultDisputeGame.Contract.Vm(&_FaultDisputeGame.CallOpts)
}

// Vm is a free data retrieval call binding the contract method 0x3a768463.
//
// Solidity: function vm() view returns(address vm_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) Vm() (common.Address, error) {
	return _FaultDisputeGame.Contract.Vm(&_FaultDisputeGame.CallOpts)
}

// Weth is a free data retrieval call binding the contract method 0x3fc8cef3.
//
// Solidity: function weth() view returns(address weth_)
func (_FaultDisputeGame *FaultDisputeGameCaller) Weth(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "weth")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Weth is a free data retrieval call binding the contract method 0x3fc8cef3.
//
// Solidity: function weth() view returns(address weth_)
func (_FaultDisputeGame *FaultDisputeGameSession) Weth() (common.Address, error) {
	return _FaultDisputeGame.Contract.Weth(&_FaultDisputeGame.CallOpts)
}

// Weth is a free data retrieval call binding the contract method 0x3fc8cef3.
//
// Solidity: function weth() view returns(address weth_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) Weth() (common.Address, error) {
	return _FaultDisputeGame.Contract.Weth(&_FaultDisputeGame.CallOpts)
}

// AddLocalData is a paid mutator transaction binding the contract method 0xf8f43ff6.
//
// Solidity: function addLocalData(uint256 _ident, uint256 _execLeafIdx, uint256 _partOffset) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) AddLocalData(opts *bind.TransactOpts, _ident *big.Int, _execLeafIdx *big.Int, _partOffset *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "addLocalData", _ident, _execLeafIdx, _partOffset)
}

// AddLocalData is a paid mutator transaction binding the contract method 0xf8f43ff6.
//
// Solidity: function addLocalData(uint256 _ident, uint256 _execLeafIdx, uint256 _partOffset) returns()
func (_FaultDisputeGame *FaultDisputeGameSession) AddLocalData(_ident *big.Int, _execLeafIdx *big.Int, _partOffset *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.AddLocalData(&_FaultDisputeGame.TransactOpts, _ident, _execLeafIdx, _partOffset)
}

// AddLocalData is a paid mutator transaction binding the contract method 0xf8f43ff6.
//
// Solidity: function addLocalData(uint256 _ident, uint256 _execLeafIdx, uint256 _partOffset) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) AddLocalData(_ident *big.Int, _execLeafIdx *big.Int, _partOffset *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.AddLocalData(&_FaultDisputeGame.TransactOpts, _ident, _execLeafIdx, _partOffset)
}

// Attack is a paid mutator transaction binding the contract method 0x472777c6.
//
// Solidity: function attack(bytes32 _disputed, uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) Attack(opts *bind.TransactOpts, _disputed [32]byte, _parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "attack", _disputed, _parentIndex, _claim)
}

// Attack is a paid mutator transaction binding the contract method 0x472777c6.
//
// Solidity: function attack(bytes32 _disputed, uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameSession) Attack(_disputed [32]byte, _parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Attack(&_FaultDisputeGame.TransactOpts, _disputed, _parentIndex, _claim)
}

// Attack is a paid mutator transaction binding the contract method 0x472777c6.
//
// Solidity: function attack(bytes32 _disputed, uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Attack(_disputed [32]byte, _parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Attack(&_FaultDisputeGame.TransactOpts, _disputed, _parentIndex, _claim)
}

// ChallengeRootL2Block is a paid mutator transaction binding the contract method 0x01935130.
//
// Solidity: function challengeRootL2Block((bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _headerRLP) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) ChallengeRootL2Block(opts *bind.TransactOpts, _outputRootProof TypesOutputRootProof, _headerRLP []byte) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "challengeRootL2Block", _outputRootProof, _headerRLP)
}

// ChallengeRootL2Block is a paid mutator transaction binding the contract method 0x01935130.
//
// Solidity: function challengeRootL2Block((bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _headerRLP) returns()
func (_FaultDisputeGame *FaultDisputeGameSession) ChallengeRootL2Block(_outputRootProof TypesOutputRootProof, _headerRLP []byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.ChallengeRootL2Block(&_FaultDisputeGame.TransactOpts, _outputRootProof, _headerRLP)
}

// ChallengeRootL2Block is a paid mutator transaction binding the contract method 0x01935130.
//
// Solidity: function challengeRootL2Block((bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _headerRLP) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) ChallengeRootL2Block(_outputRootProof TypesOutputRootProof, _headerRLP []byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.ChallengeRootL2Block(&_FaultDisputeGame.TransactOpts, _outputRootProof, _headerRLP)
}

// ClaimCredit is a paid mutator transaction binding the contract method 0x60e27464.
//
// Solidity: function claimCredit(address _recipient) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) ClaimCredit(opts *bind.TransactOpts, _recipient common.Address) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "claimCredit", _recipient)
}

// ClaimCredit is a paid mutator transaction binding the contract method 0x60e27464.
//
// Solidity: function claimCredit(address _recipient) returns()
func (_FaultDisputeGame *FaultDisputeGameSession) ClaimCredit(_recipient common.Address) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.ClaimCredit(&_FaultDisputeGame.TransactOpts, _recipient)
}

// ClaimCredit is a paid mutator transaction binding the contract method 0x60e27464.
//
// Solidity: function claimCredit(address _recipient) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) ClaimCredit(_recipient common.Address) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.ClaimCredit(&_FaultDisputeGame.TransactOpts, _recipient)
}

// Defend is a paid mutator transaction binding the contract method 0x7b0f0adc.
//
// Solidity: function defend(bytes32 _disputed, uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) Defend(opts *bind.TransactOpts, _disputed [32]byte, _parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "defend", _disputed, _parentIndex, _claim)
}

// Defend is a paid mutator transaction binding the contract method 0x7b0f0adc.
//
// Solidity: function defend(bytes32 _disputed, uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameSession) Defend(_disputed [32]byte, _parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Defend(&_FaultDisputeGame.TransactOpts, _disputed, _parentIndex, _claim)
}

// Defend is a paid mutator transaction binding the contract method 0x7b0f0adc.
//
// Solidity: function defend(bytes32 _disputed, uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Defend(_disputed [32]byte, _parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Defend(&_FaultDisputeGame.TransactOpts, _disputed, _parentIndex, _claim)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) Initialize(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "initialize")
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() payable returns()
func (_FaultDisputeGame *FaultDisputeGameSession) Initialize() (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Initialize(&_FaultDisputeGame.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Initialize() (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Initialize(&_FaultDisputeGame.TransactOpts)
}

// Move is a paid mutator transaction binding the contract method 0x6f034409.
//
// Solidity: function move(bytes32 _disputed, uint256 _challengeIndex, bytes32 _claim, bool _isAttack) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) Move(opts *bind.TransactOpts, _disputed [32]byte, _challengeIndex *big.Int, _claim [32]byte, _isAttack bool) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "move", _disputed, _challengeIndex, _claim, _isAttack)
}

// Move is a paid mutator transaction binding the contract method 0x6f034409.
//
// Solidity: function move(bytes32 _disputed, uint256 _challengeIndex, bytes32 _claim, bool _isAttack) payable returns()
func (_FaultDisputeGame *FaultDisputeGameSession) Move(_disputed [32]byte, _challengeIndex *big.Int, _claim [32]byte, _isAttack bool) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Move(&_FaultDisputeGame.TransactOpts, _disputed, _challengeIndex, _claim, _isAttack)
}

// Move is a paid mutator transaction binding the contract method 0x6f034409.
//
// Solidity: function move(bytes32 _disputed, uint256 _challengeIndex, bytes32 _claim, bool _isAttack) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Move(_disputed [32]byte, _challengeIndex *big.Int, _claim [32]byte, _isAttack bool) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Move(&_FaultDisputeGame.TransactOpts, _disputed, _challengeIndex, _claim, _isAttack)
}

// Resolve is a paid mutator transaction binding the contract method 0x2810e1d6.
//
// Solidity: function resolve() returns(uint8 status_)
func (_FaultDisputeGame *FaultDisputeGameTransactor) Resolve(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "resolve")
}

// Resolve is a paid mutator transaction binding the contract method 0x2810e1d6.
//
// Solidity: function resolve() returns(uint8 status_)
func (_FaultDisputeGame *FaultDisputeGameSession) Resolve() (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Resolve(&_FaultDisputeGame.TransactOpts)
}

// Resolve is a paid mutator transaction binding the contract method 0x2810e1d6.
//
// Solidity: function resolve() returns(uint8 status_)
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Resolve() (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Resolve(&_FaultDisputeGame.TransactOpts)
}

// ResolveClaim is a paid mutator transaction binding the contract method 0x03c2924d.
//
// Solidity: function resolveClaim(uint256 _claimIndex, uint256 _numToResolve) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) ResolveClaim(opts *bind.TransactOpts, _claimIndex *big.Int, _numToResolve *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "resolveClaim", _claimIndex, _numToResolve)
}

// ResolveClaim is a paid mutator transaction binding the contract method 0x03c2924d.
//
// Solidity: function resolveClaim(uint256 _claimIndex, uint256 _numToResolve) returns()
func (_FaultDisputeGame *FaultDisputeGameSession) ResolveClaim(_claimIndex *big.Int, _numToResolve *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.ResolveClaim(&_FaultDisputeGame.TransactOpts, _claimIndex, _numToResolve)
}

// ResolveClaim is a paid mutator transaction binding the contract method 0x03c2924d.
//
// Solidity: function resolveClaim(uint256 _claimIndex, uint256 _numToResolve) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) ResolveClaim(_claimIndex *big.Int, _numToResolve *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.ResolveClaim(&_FaultDisputeGame.TransactOpts, _claimIndex, _numToResolve)
}

// Step is a paid mutator transaction binding the contract method 0xd8cc1a3c.
//
// Solidity: function step(uint256 _claimIndex, bool _isAttack, bytes _stateData, bytes _proof) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) Step(opts *bind.TransactOpts, _claimIndex *big.Int, _isAttack bool, _stateData []byte, _proof []byte) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "step", _claimIndex, _isAttack, _stateData, _proof)
}

// Step is a paid mutator transaction binding the contract method 0xd8cc1a3c.
//
// Solidity: function step(uint256 _claimIndex, bool _isAttack, bytes _stateData, bytes _proof) returns()
func (_FaultDisputeGame *FaultDisputeGameSession) Step(_claimIndex *big.Int, _isAttack bool, _stateData []byte, _proof []byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Step(&_FaultDisputeGame.TransactOpts, _claimIndex, _isAttack, _stateData, _proof)
}

// Step is a paid mutator transaction binding the contract method 0xd8cc1a3c.
//
// Solidity: function step(uint256 _claimIndex, bool _isAttack, bytes _stateData, bytes _proof) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Step(_claimIndex *big.Int, _isAttack bool, _stateData []byte, _proof []byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Step(&_FaultDisputeGame.TransactOpts, _claimIndex, _isAttack, _stateData, _proof)
}

// FaultDisputeGameMoveIterator is returned from FilterMove and is used to iterate over the raw logs and unpacked data for Move events raised by the FaultDisputeGame contract.
type FaultDisputeGameMoveIterator struct {
	Event *FaultDisputeGameMove // Event containing the contract specifics and raw log

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
func (it *FaultDisputeGameMoveIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FaultDisputeGameMove)
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
		it.Event = new(FaultDisputeGameMove)
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
func (it *FaultDisputeGameMoveIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FaultDisputeGameMoveIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FaultDisputeGameMove represents a Move event raised by the FaultDisputeGame contract.
type FaultDisputeGameMove struct {
	ParentIndex *big.Int
	Claim       [32]byte
	Claimant    common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterMove is a free log retrieval operation binding the contract event 0x9b3245740ec3b155098a55be84957a4da13eaf7f14a8bc6f53126c0b9350f2be.
//
// Solidity: event Move(uint256 indexed parentIndex, bytes32 indexed claim, address indexed claimant)
func (_FaultDisputeGame *FaultDisputeGameFilterer) FilterMove(opts *bind.FilterOpts, parentIndex []*big.Int, claim [][32]byte, claimant []common.Address) (*FaultDisputeGameMoveIterator, error) {

	var parentIndexRule []interface{}
	for _, parentIndexItem := range parentIndex {
		parentIndexRule = append(parentIndexRule, parentIndexItem)
	}
	var claimRule []interface{}
	for _, claimItem := range claim {
		claimRule = append(claimRule, claimItem)
	}
	var claimantRule []interface{}
	for _, claimantItem := range claimant {
		claimantRule = append(claimantRule, claimantItem)
	}

	logs, sub, err := _FaultDisputeGame.contract.FilterLogs(opts, "Move", parentIndexRule, claimRule, claimantRule)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameMoveIterator{contract: _FaultDisputeGame.contract, event: "Move", logs: logs, sub: sub}, nil
}

// WatchMove is a free log subscription operation binding the contract event 0x9b3245740ec3b155098a55be84957a4da13eaf7f14a8bc6f53126c0b9350f2be.
//
// Solidity: event Move(uint256 indexed parentIndex, bytes32 indexed claim, address indexed claimant)
func (_FaultDisputeGame *FaultDisputeGameFilterer) WatchMove(opts *bind.WatchOpts, sink chan<- *FaultDisputeGameMove, parentIndex []*big.Int, claim [][32]byte, claimant []common.Address) (event.Subscription, error) {

	var parentIndexRule []interface{}
	for _, parentIndexItem := range parentIndex {
		parentIndexRule = append(parentIndexRule, parentIndexItem)
	}
	var claimRule []interface{}
	for _, claimItem := range claim {
		claimRule = append(claimRule, claimItem)
	}
	var claimantRule []interface{}
	for _, claimantItem := range claimant {
		claimantRule = append(claimantRule, claimantItem)
	}

	logs, sub, err := _FaultDisputeGame.contract.WatchLogs(opts, "Move", parentIndexRule, claimRule, claimantRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FaultDisputeGameMove)
				if err := _FaultDisputeGame.contract.UnpackLog(event, "Move", log); err != nil {
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

// ParseMove is a log parse operation binding the contract event 0x9b3245740ec3b155098a55be84957a4da13eaf7f14a8bc6f53126c0b9350f2be.
//
// Solidity: event Move(uint256 indexed parentIndex, bytes32 indexed claim, address indexed claimant)
func (_FaultDisputeGame *FaultDisputeGameFilterer) ParseMove(log types.Log) (*FaultDisputeGameMove, error) {
	event := new(FaultDisputeGameMove)
	if err := _FaultDisputeGame.contract.UnpackLog(event, "Move", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FaultDisputeGameResolvedIterator is returned from FilterResolved and is used to iterate over the raw logs and unpacked data for Resolved events raised by the FaultDisputeGame contract.
type FaultDisputeGameResolvedIterator struct {
	Event *FaultDisputeGameResolved // Event containing the contract specifics and raw log

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
func (it *FaultDisputeGameResolvedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FaultDisputeGameResolved)
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
		it.Event = new(FaultDisputeGameResolved)
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
func (it *FaultDisputeGameResolvedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FaultDisputeGameResolvedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FaultDisputeGameResolved represents a Resolved event raised by the FaultDisputeGame contract.
type FaultDisputeGameResolved struct {
	Status uint8
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterResolved is a free log retrieval operation binding the contract event 0x5e186f09b9c93491f14e277eea7faa5de6a2d4bda75a79af7a3684fbfb42da60.
//
// Solidity: event Resolved(uint8 indexed status)
func (_FaultDisputeGame *FaultDisputeGameFilterer) FilterResolved(opts *bind.FilterOpts, status []uint8) (*FaultDisputeGameResolvedIterator, error) {

	var statusRule []interface{}
	for _, statusItem := range status {
		statusRule = append(statusRule, statusItem)
	}

	logs, sub, err := _FaultDisputeGame.contract.FilterLogs(opts, "Resolved", statusRule)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameResolvedIterator{contract: _FaultDisputeGame.contract, event: "Resolved", logs: logs, sub: sub}, nil
}

// WatchResolved is a free log subscription operation binding the contract event 0x5e186f09b9c93491f14e277eea7faa5de6a2d4bda75a79af7a3684fbfb42da60.
//
// Solidity: event Resolved(uint8 indexed status)
func (_FaultDisputeGame *FaultDisputeGameFilterer) WatchResolved(opts *bind.WatchOpts, sink chan<- *FaultDisputeGameResolved, status []uint8) (event.Subscription, error) {

	var statusRule []interface{}
	for _, statusItem := range status {
		statusRule = append(statusRule, statusItem)
	}

	logs, sub, err := _FaultDisputeGame.contract.WatchLogs(opts, "Resolved", statusRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FaultDisputeGameResolved)
				if err := _FaultDisputeGame.contract.UnpackLog(event, "Resolved", log); err != nil {
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

// ParseResolved is a log parse operation binding the contract event 0x5e186f09b9c93491f14e277eea7faa5de6a2d4bda75a79af7a3684fbfb42da60.
//
// Solidity: event Resolved(uint8 indexed status)
func (_FaultDisputeGame *FaultDisputeGameFilterer) ParseResolved(log types.Log) (*FaultDisputeGameResolved, error) {
	event := new(FaultDisputeGameResolved)
	if err := _FaultDisputeGame.contract.UnpackLog(event, "Resolved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
