// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"math/big"
	"strings"

	ethereum "github.com/ledgerwatch/erigon"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/accounts/abi"
	"github.com/ledgerwatch/erigon/accounts/abi/bind"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = libcommon.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// BobaTuringCreditABI is the input ABI used to generate the binding from.
const BobaTuringCreditABI = "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_turingPrice\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"balanceAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"helperContractAddress\",\"type\":\"address\"}],\"name\":\"AddBalanceTo\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"TransferOwnership\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"withdrawAmount\",\"type\":\"uint256\"}],\"name\":\"WithdrawRevenue\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"HCHelperAddr\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_addBalanceAmount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_helperContractAddress\",\"type\":\"address\"}],\"name\":\"addBalanceTo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_helperContractAddress\",\"type\":\"address\"}],\"name\":\"getCreditAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ownerRevenue\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"prepaidBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_helperContractAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"spendCredit\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"turingPrice\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"turingToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_turingPrice\",\"type\":\"uint256\"}],\"name\":\"updateTuringPrice\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_turingToken\",\"type\":\"address\"}],\"name\":\"updateTuringToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_withdrawAmount\",\"type\":\"uint256\"}],\"name\":\"withdrawRevenue\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// BobaTuringCreditBin is the compiled bytecode used for deploying new contracts.
var BobaTuringCreditBin = "0x60a060405273420000000000000000000000000000000000002260805234801561002857600080fd5b506040516114ac3803806114ac8339810160408190526100479161004f565b600355610068565b60006020828403121561006157600080fd5b5051919050565b60805161142261008a60003960008181610216015261026601526114226000f3fe608060405234801561001057600080fd5b50600436106100de5760003560e01c80638da5cb5b1161008c578063f2fde38b11610066578063f2fde38b146101eb578063f7cd3be8146101fe578063f919569d14610211578063fd8922781461023857600080fd5b80638da5cb5b146101af578063a52b962d146101cf578063e24dfcde146101e257600080fd5b80630ceff204116100bd5780630ceff2041461014257806335d6eac414610157578063853383921461016a57600080fd5b8062292526146100e357806309da3981146100ff5780630a8c07b01461011f575b600080fd5b6100ec60045481565b6040519081526020015b60405180910390f35b6100ec61010d366004611237565b60016020526000908152604090205481565b61013261012d366004611252565b61024b565b60405190151581526020016100f6565b61015561015036600461127c565b61037d565b005b610155610165366004611237565b6105ad565b60025461018a9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100f6565b60005461018a9073ffffffffffffffffffffffffffffffffffffffff1681565b6100ec6101dd366004611237565b610716565b6100ec60035481565b6101556101f9366004611237565b6107bc565b61015561020c36600461127c565b6108fe565b61018a7f000000000000000000000000000000000000000000000000000000000000000081565b610155610246366004611295565b6109a5565b6000803373ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016146102f2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f43616e206f6e6c792062652063616c6c65642066726f6d20484348656c70657260448201526064015b60405180910390fd5b73ffffffffffffffffffffffffffffffffffffffff841660009081526001602052604090205483116103765773ffffffffffffffffffffffffffffffffffffffff8416600090815260016020526040812080548592906103539084906112f0565b92505081905550826004600082825461036c9190611307565b9091555060019150505b9392505050565b60005473ffffffffffffffffffffffffffffffffffffffff163314806103b9575060005473ffffffffffffffffffffffffffffffffffffffff16155b61041f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016102e9565b60025473ffffffffffffffffffffffffffffffffffffffff166104c4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f436f6e747261637420686173206e6f7420796574206265656e20696e6974696160448201527f6c697a656400000000000000000000000000000000000000000000000000000060648201526084016102e9565b600454811115610530576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f496e76616c696420416d6f756e7400000000000000000000000000000000000060448201526064016102e9565b806004600082825461054291906112f0565b909155505060408051338152602081018390527f447d53be88e315476bdbe2e63cef309461f6305d09aada67641c29e6b897e301910160405180910390a16000546002546105aa9173ffffffffffffffffffffffffffffffffffffffff918216911683610c7a565b50565b60005473ffffffffffffffffffffffffffffffffffffffff163314806105e9575060005473ffffffffffffffffffffffffffffffffffffffff16155b61064f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016102e9565b60025473ffffffffffffffffffffffffffffffffffffffff16156106cf576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f436f6e747261637420686173206265656e20696e697469616c697a656400000060448201526064016102e9565b600280547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b6000600354600003610784576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601060248201527f556e6c696d69746564206372656469740000000000000000000000000000000060448201526064016102e9565b60035473ffffffffffffffffffffffffffffffffffffffff83166000908152600160205260409020546107b691610d53565b92915050565b60005473ffffffffffffffffffffffffffffffffffffffff163314806107f8575060005473ffffffffffffffffffffffffffffffffffffffff16155b61085e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016102e9565b73ffffffffffffffffffffffffffffffffffffffff811661087e57600080fd5b600080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff83169081179091556040805133815260208101929092527f5c486528ec3e3f0ea91181cff8116f02bfa350e03b8b6f12e00765adbb5af85c910160405180910390a150565b60005473ffffffffffffffffffffffffffffffffffffffff1633148061093a575060005473ffffffffffffffffffffffffffffffffffffffff16155b6109a0576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016102e9565b600355565b60025473ffffffffffffffffffffffffffffffffffffffff16610a4a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f436f6e747261637420686173206e6f7420796574206265656e20696e6974696160448201527f6c697a656400000000000000000000000000000000000000000000000000000060648201526084016102e9565b81600003610ab4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f496e76616c696420616d6f756e7400000000000000000000000000000000000060448201526064016102e9565b73ffffffffffffffffffffffffffffffffffffffff81163b610b32576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f4164647265737320697320454f4100000000000000000000000000000000000060448201526064016102e9565b610b5c817f2f7adf4300000000000000000000000000000000000000000000000000000000610d5f565b610bc2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f496e76616c69642048656c70657220436f6e747261637400000000000000000060448201526064016102e9565b73ffffffffffffffffffffffffffffffffffffffff811660009081526001602052604081208054849290610bf7908490611307565b9091555050604080513381526020810184905273ffffffffffffffffffffffffffffffffffffffff83168183015290517f63611f4b2e0fff4acd8e17bd95ebb62a3bc834c76cf85e7a972a502990b6257a9181900360600190a1600254610c769073ffffffffffffffffffffffffffffffffffffffff16333085610d7b565b5050565b60405173ffffffffffffffffffffffffffffffffffffffff8316602482015260448101829052610d4e9084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff0000000000000000000000000000000000000000000000000000000090931692909217909152610ddf565b505050565b6000610376828461131f565b6000610d6a83610eeb565b801561037657506103768383610f4f565b60405173ffffffffffffffffffffffffffffffffffffffff80851660248301528316604482015260648101829052610dd99085907f23b872dd0000000000000000000000000000000000000000000000000000000090608401610ccc565b50505050565b6000610e41826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c65648152508573ffffffffffffffffffffffffffffffffffffffff1661101e9092919063ffffffff16565b805190915015610d4e5780806020019051810190610e5f919061135a565b610d4e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f7420737563636565640000000000000000000000000000000000000000000060648201526084016102e9565b6000610f17827f01ffc9a700000000000000000000000000000000000000000000000000000000610f4f565b80156107b65750610f48827fffffffff00000000000000000000000000000000000000000000000000000000610f4f565b1592915050565b604080517fffffffff000000000000000000000000000000000000000000000000000000008316602480830191909152825180830390910181526044909101909152602080820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01ffc9a700000000000000000000000000000000000000000000000000000000178152825160009392849283928392918391908a617530fa92503d91506000519050828015611007575060208210155b80156110135750600081115b979650505050505050565b606061102d8484600085611035565b949350505050565b6060824710156110c7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c000000000000000000000000000000000000000000000000000060648201526084016102e9565b73ffffffffffffffffffffffffffffffffffffffff85163b611145576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e747261637400000060448201526064016102e9565b6000808673ffffffffffffffffffffffffffffffffffffffff16858760405161116e91906113a8565b60006040518083038185875af1925050503d80600081146111ab576040519150601f19603f3d011682016040523d82523d6000602084013e6111b0565b606091505b5091509150611013828286606083156111ca575081610376565b8251156111da5782518084602001fd5b816040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102e991906113c4565b803573ffffffffffffffffffffffffffffffffffffffff8116811461123257600080fd5b919050565b60006020828403121561124957600080fd5b6103768261120e565b6000806040838503121561126557600080fd5b61126e8361120e565b946020939093013593505050565b60006020828403121561128e57600080fd5b5035919050565b600080604083850312156112a857600080fd5b823591506112b86020840161120e565b90509250929050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015611302576113026112c1565b500390565b6000821982111561131a5761131a6112c1565b500190565b600082611355577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b500490565b60006020828403121561136c57600080fd5b8151801515811461037657600080fd5b60005b8381101561139757818101518382015260200161137f565b83811115610dd95750506000910152565b600082516113ba81846020870161137c565b9190910192915050565b60208152600082518060208401526113e381604085016020870161137c565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016919091016040019291505056fea164736f6c634300080f000a"

// DeployBobaTuringCredit deploys a new Ethereum contract, binding an instance of BobaTuringCredit to it.
func DeployBobaTuringCredit(auth *bind.TransactOpts, backend bind.ContractBackend, _turingPrice *big.Int) (libcommon.Address, types.Transaction, *BobaTuringCredit, error) {
	parsed, err := abi.JSON(strings.NewReader(BobaTuringCreditABI))
	if err != nil {
		return libcommon.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, libcommon.FromHex(BobaTuringCreditBin), backend, _turingPrice)
	if err != nil {
		return libcommon.Address{}, nil, nil, err
	}
	return address, tx, &BobaTuringCredit{BobaTuringCreditCaller: BobaTuringCreditCaller{contract: contract}, BobaTuringCreditTransactor: BobaTuringCreditTransactor{contract: contract}, BobaTuringCreditFilterer: BobaTuringCreditFilterer{contract: contract}}, nil
}

// BobaTuringCredit is an auto generated Go binding around an Ethereum contract.
type BobaTuringCredit struct {
	BobaTuringCreditCaller     // Read-only binding to the contract
	BobaTuringCreditTransactor // Write-only binding to the contract
	BobaTuringCreditFilterer   // Log filterer for contract events
}

// BobaTuringCreditCaller is an auto generated read-only Go binding around an Ethereum contract.
type BobaTuringCreditCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BobaTuringCreditTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BobaTuringCreditTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BobaTuringCreditFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BobaTuringCreditFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BobaTuringCreditSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BobaTuringCreditSession struct {
	Contract     *BobaTuringCredit // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BobaTuringCreditCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BobaTuringCreditCallerSession struct {
	Contract *BobaTuringCreditCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// BobaTuringCreditTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BobaTuringCreditTransactorSession struct {
	Contract     *BobaTuringCreditTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// BobaTuringCreditRaw is an auto generated low-level Go binding around an Ethereum contract.
type BobaTuringCreditRaw struct {
	Contract *BobaTuringCredit // Generic contract binding to access the raw methods on
}

// BobaTuringCreditCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BobaTuringCreditCallerRaw struct {
	Contract *BobaTuringCreditCaller // Generic read-only contract binding to access the raw methods on
}

// BobaTuringCreditTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BobaTuringCreditTransactorRaw struct {
	Contract *BobaTuringCreditTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBobaTuringCredit creates a new instance of BobaTuringCredit, bound to a specific deployed contract.
func NewBobaTuringCredit(address libcommon.Address, backend bind.ContractBackend) (*BobaTuringCredit, error) {
	contract, err := bindBobaTuringCredit(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BobaTuringCredit{BobaTuringCreditCaller: BobaTuringCreditCaller{contract: contract}, BobaTuringCreditTransactor: BobaTuringCreditTransactor{contract: contract}, BobaTuringCreditFilterer: BobaTuringCreditFilterer{contract: contract}}, nil
}

// NewBobaTuringCreditCaller creates a new read-only instance of BobaTuringCredit, bound to a specific deployed contract.
func NewBobaTuringCreditCaller(address libcommon.Address, caller bind.ContractCaller) (*BobaTuringCreditCaller, error) {
	contract, err := bindBobaTuringCredit(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BobaTuringCreditCaller{contract: contract}, nil
}

// NewBobaTuringCreditTransactor creates a new write-only instance of BobaTuringCredit, bound to a specific deployed contract.
func NewBobaTuringCreditTransactor(address libcommon.Address, transactor bind.ContractTransactor) (*BobaTuringCreditTransactor, error) {
	contract, err := bindBobaTuringCredit(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BobaTuringCreditTransactor{contract: contract}, nil
}

// NewBobaTuringCreditFilterer creates a new log filterer instance of BobaTuringCredit, bound to a specific deployed contract.
func NewBobaTuringCreditFilterer(address libcommon.Address, filterer bind.ContractFilterer) (*BobaTuringCreditFilterer, error) {
	contract, err := bindBobaTuringCredit(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BobaTuringCreditFilterer{contract: contract}, nil
}

// bindBobaTuringCredit binds a generic wrapper to an already deployed contract.
func bindBobaTuringCredit(address libcommon.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BobaTuringCreditABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BobaTuringCredit *BobaTuringCreditRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BobaTuringCredit.Contract.BobaTuringCreditCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BobaTuringCredit *BobaTuringCreditRaw) Transfer(opts *bind.TransactOpts) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.BobaTuringCreditTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BobaTuringCredit *BobaTuringCreditRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.BobaTuringCreditTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BobaTuringCredit *BobaTuringCreditCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BobaTuringCredit.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BobaTuringCredit *BobaTuringCreditTransactorRaw) Transfer(opts *bind.TransactOpts) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BobaTuringCredit *BobaTuringCreditTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.contract.Transact(opts, method, params...)
}

// HCHelperAddr is a free data retrieval call binding the contract method 0xf919569d.
//
// Solidity: function HCHelperAddr() view returns(address)
func (_BobaTuringCredit *BobaTuringCreditCaller) HCHelperAddr(opts *bind.CallOpts) (libcommon.Address, error) {
	var out []interface{}
	err := _BobaTuringCredit.contract.Call(opts, &out, "HCHelperAddr")

	if err != nil {
		return *new(libcommon.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(libcommon.Address)).(*libcommon.Address)

	return out0, err

}

// HCHelperAddr is a free data retrieval call binding the contract method 0xf919569d.
//
// Solidity: function HCHelperAddr() view returns(address)
func (_BobaTuringCredit *BobaTuringCreditSession) HCHelperAddr() (libcommon.Address, error) {
	return _BobaTuringCredit.Contract.HCHelperAddr(&_BobaTuringCredit.CallOpts)
}

// HCHelperAddr is a free data retrieval call binding the contract method 0xf919569d.
//
// Solidity: function HCHelperAddr() view returns(address)
func (_BobaTuringCredit *BobaTuringCreditCallerSession) HCHelperAddr() (libcommon.Address, error) {
	return _BobaTuringCredit.Contract.HCHelperAddr(&_BobaTuringCredit.CallOpts)
}

// GetCreditAmount is a free data retrieval call binding the contract method 0xa52b962d.
//
// Solidity: function getCreditAmount(address _helperContractAddress) view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditCaller) GetCreditAmount(opts *bind.CallOpts, _helperContractAddress libcommon.Address) (*big.Int, error) {
	var out []interface{}
	err := _BobaTuringCredit.contract.Call(opts, &out, "getCreditAmount", _helperContractAddress)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCreditAmount is a free data retrieval call binding the contract method 0xa52b962d.
//
// Solidity: function getCreditAmount(address _helperContractAddress) view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditSession) GetCreditAmount(_helperContractAddress libcommon.Address) (*big.Int, error) {
	return _BobaTuringCredit.Contract.GetCreditAmount(&_BobaTuringCredit.CallOpts, _helperContractAddress)
}

// GetCreditAmount is a free data retrieval call binding the contract method 0xa52b962d.
//
// Solidity: function getCreditAmount(address _helperContractAddress) view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditCallerSession) GetCreditAmount(_helperContractAddress libcommon.Address) (*big.Int, error) {
	return _BobaTuringCredit.Contract.GetCreditAmount(&_BobaTuringCredit.CallOpts, _helperContractAddress)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BobaTuringCredit *BobaTuringCreditCaller) Owner(opts *bind.CallOpts) (libcommon.Address, error) {
	var out []interface{}
	err := _BobaTuringCredit.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(libcommon.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(libcommon.Address)).(*libcommon.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BobaTuringCredit *BobaTuringCreditSession) Owner() (libcommon.Address, error) {
	return _BobaTuringCredit.Contract.Owner(&_BobaTuringCredit.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BobaTuringCredit *BobaTuringCreditCallerSession) Owner() (libcommon.Address, error) {
	return _BobaTuringCredit.Contract.Owner(&_BobaTuringCredit.CallOpts)
}

// OwnerRevenue is a free data retrieval call binding the contract method 0x00292526.
//
// Solidity: function ownerRevenue() view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditCaller) OwnerRevenue(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BobaTuringCredit.contract.Call(opts, &out, "ownerRevenue")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OwnerRevenue is a free data retrieval call binding the contract method 0x00292526.
//
// Solidity: function ownerRevenue() view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditSession) OwnerRevenue() (*big.Int, error) {
	return _BobaTuringCredit.Contract.OwnerRevenue(&_BobaTuringCredit.CallOpts)
}

// OwnerRevenue is a free data retrieval call binding the contract method 0x00292526.
//
// Solidity: function ownerRevenue() view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditCallerSession) OwnerRevenue() (*big.Int, error) {
	return _BobaTuringCredit.Contract.OwnerRevenue(&_BobaTuringCredit.CallOpts)
}

// PrepaidBalance is a free data retrieval call binding the contract method 0x09da3981.
//
// Solidity: function prepaidBalance(address ) view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditCaller) PrepaidBalance(opts *bind.CallOpts, arg0 libcommon.Address) (*big.Int, error) {
	var out []interface{}
	err := _BobaTuringCredit.contract.Call(opts, &out, "prepaidBalance", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PrepaidBalance is a free data retrieval call binding the contract method 0x09da3981.
//
// Solidity: function prepaidBalance(address ) view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditSession) PrepaidBalance(arg0 libcommon.Address) (*big.Int, error) {
	return _BobaTuringCredit.Contract.PrepaidBalance(&_BobaTuringCredit.CallOpts, arg0)
}

// PrepaidBalance is a free data retrieval call binding the contract method 0x09da3981.
//
// Solidity: function prepaidBalance(address ) view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditCallerSession) PrepaidBalance(arg0 libcommon.Address) (*big.Int, error) {
	return _BobaTuringCredit.Contract.PrepaidBalance(&_BobaTuringCredit.CallOpts, arg0)
}

// TuringPrice is a free data retrieval call binding the contract method 0xe24dfcde.
//
// Solidity: function turingPrice() view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditCaller) TuringPrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BobaTuringCredit.contract.Call(opts, &out, "turingPrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TuringPrice is a free data retrieval call binding the contract method 0xe24dfcde.
//
// Solidity: function turingPrice() view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditSession) TuringPrice() (*big.Int, error) {
	return _BobaTuringCredit.Contract.TuringPrice(&_BobaTuringCredit.CallOpts)
}

// TuringPrice is a free data retrieval call binding the contract method 0xe24dfcde.
//
// Solidity: function turingPrice() view returns(uint256)
func (_BobaTuringCredit *BobaTuringCreditCallerSession) TuringPrice() (*big.Int, error) {
	return _BobaTuringCredit.Contract.TuringPrice(&_BobaTuringCredit.CallOpts)
}

// TuringToken is a free data retrieval call binding the contract method 0x85338392.
//
// Solidity: function turingToken() view returns(address)
func (_BobaTuringCredit *BobaTuringCreditCaller) TuringToken(opts *bind.CallOpts) (libcommon.Address, error) {
	var out []interface{}
	err := _BobaTuringCredit.contract.Call(opts, &out, "turingToken")

	if err != nil {
		return *new(libcommon.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(libcommon.Address)).(*libcommon.Address)

	return out0, err

}

// TuringToken is a free data retrieval call binding the contract method 0x85338392.
//
// Solidity: function turingToken() view returns(address)
func (_BobaTuringCredit *BobaTuringCreditSession) TuringToken() (libcommon.Address, error) {
	return _BobaTuringCredit.Contract.TuringToken(&_BobaTuringCredit.CallOpts)
}

// TuringToken is a free data retrieval call binding the contract method 0x85338392.
//
// Solidity: function turingToken() view returns(address)
func (_BobaTuringCredit *BobaTuringCreditCallerSession) TuringToken() (libcommon.Address, error) {
	return _BobaTuringCredit.Contract.TuringToken(&_BobaTuringCredit.CallOpts)
}

// AddBalanceTo is a paid mutator transaction binding the contract method 0xfd892278.
//
// Solidity: function addBalanceTo(uint256 _addBalanceAmount, address _helperContractAddress) returns()
func (_BobaTuringCredit *BobaTuringCreditTransactor) AddBalanceTo(opts *bind.TransactOpts, _addBalanceAmount *big.Int, _helperContractAddress libcommon.Address) (types.Transaction, error) {
	return _BobaTuringCredit.contract.Transact(opts, "addBalanceTo", _addBalanceAmount, _helperContractAddress)
}

// AddBalanceTo is a paid mutator transaction binding the contract method 0xfd892278.
//
// Solidity: function addBalanceTo(uint256 _addBalanceAmount, address _helperContractAddress) returns()
func (_BobaTuringCredit *BobaTuringCreditSession) AddBalanceTo(_addBalanceAmount *big.Int, _helperContractAddress libcommon.Address) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.AddBalanceTo(&_BobaTuringCredit.TransactOpts, _addBalanceAmount, _helperContractAddress)
}

// AddBalanceTo is a paid mutator transaction binding the contract method 0xfd892278.
//
// Solidity: function addBalanceTo(uint256 _addBalanceAmount, address _helperContractAddress) returns()
func (_BobaTuringCredit *BobaTuringCreditTransactorSession) AddBalanceTo(_addBalanceAmount *big.Int, _helperContractAddress libcommon.Address) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.AddBalanceTo(&_BobaTuringCredit.TransactOpts, _addBalanceAmount, _helperContractAddress)
}

// SpendCredit is a paid mutator transaction binding the contract method 0x0a8c07b0.
//
// Solidity: function spendCredit(address _helperContractAddress, uint256 _amount) returns(bool)
func (_BobaTuringCredit *BobaTuringCreditTransactor) SpendCredit(opts *bind.TransactOpts, _helperContractAddress libcommon.Address, _amount *big.Int) (types.Transaction, error) {
	return _BobaTuringCredit.contract.Transact(opts, "spendCredit", _helperContractAddress, _amount)
}

// SpendCredit is a paid mutator transaction binding the contract method 0x0a8c07b0.
//
// Solidity: function spendCredit(address _helperContractAddress, uint256 _amount) returns(bool)
func (_BobaTuringCredit *BobaTuringCreditSession) SpendCredit(_helperContractAddress libcommon.Address, _amount *big.Int) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.SpendCredit(&_BobaTuringCredit.TransactOpts, _helperContractAddress, _amount)
}

// SpendCredit is a paid mutator transaction binding the contract method 0x0a8c07b0.
//
// Solidity: function spendCredit(address _helperContractAddress, uint256 _amount) returns(bool)
func (_BobaTuringCredit *BobaTuringCreditTransactorSession) SpendCredit(_helperContractAddress libcommon.Address, _amount *big.Int) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.SpendCredit(&_BobaTuringCredit.TransactOpts, _helperContractAddress, _amount)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_BobaTuringCredit *BobaTuringCreditTransactor) TransferOwnership(opts *bind.TransactOpts, _newOwner libcommon.Address) (types.Transaction, error) {
	return _BobaTuringCredit.contract.Transact(opts, "transferOwnership", _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_BobaTuringCredit *BobaTuringCreditSession) TransferOwnership(_newOwner libcommon.Address) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.TransferOwnership(&_BobaTuringCredit.TransactOpts, _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_BobaTuringCredit *BobaTuringCreditTransactorSession) TransferOwnership(_newOwner libcommon.Address) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.TransferOwnership(&_BobaTuringCredit.TransactOpts, _newOwner)
}

// UpdateTuringPrice is a paid mutator transaction binding the contract method 0xf7cd3be8.
//
// Solidity: function updateTuringPrice(uint256 _turingPrice) returns()
func (_BobaTuringCredit *BobaTuringCreditTransactor) UpdateTuringPrice(opts *bind.TransactOpts, _turingPrice *big.Int) (types.Transaction, error) {
	return _BobaTuringCredit.contract.Transact(opts, "updateTuringPrice", _turingPrice)
}

// UpdateTuringPrice is a paid mutator transaction binding the contract method 0xf7cd3be8.
//
// Solidity: function updateTuringPrice(uint256 _turingPrice) returns()
func (_BobaTuringCredit *BobaTuringCreditSession) UpdateTuringPrice(_turingPrice *big.Int) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.UpdateTuringPrice(&_BobaTuringCredit.TransactOpts, _turingPrice)
}

// UpdateTuringPrice is a paid mutator transaction binding the contract method 0xf7cd3be8.
//
// Solidity: function updateTuringPrice(uint256 _turingPrice) returns()
func (_BobaTuringCredit *BobaTuringCreditTransactorSession) UpdateTuringPrice(_turingPrice *big.Int) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.UpdateTuringPrice(&_BobaTuringCredit.TransactOpts, _turingPrice)
}

// UpdateTuringToken is a paid mutator transaction binding the contract method 0x35d6eac4.
//
// Solidity: function updateTuringToken(address _turingToken) returns()
func (_BobaTuringCredit *BobaTuringCreditTransactor) UpdateTuringToken(opts *bind.TransactOpts, _turingToken libcommon.Address) (types.Transaction, error) {
	return _BobaTuringCredit.contract.Transact(opts, "updateTuringToken", _turingToken)
}

// UpdateTuringToken is a paid mutator transaction binding the contract method 0x35d6eac4.
//
// Solidity: function updateTuringToken(address _turingToken) returns()
func (_BobaTuringCredit *BobaTuringCreditSession) UpdateTuringToken(_turingToken libcommon.Address) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.UpdateTuringToken(&_BobaTuringCredit.TransactOpts, _turingToken)
}

// UpdateTuringToken is a paid mutator transaction binding the contract method 0x35d6eac4.
//
// Solidity: function updateTuringToken(address _turingToken) returns()
func (_BobaTuringCredit *BobaTuringCreditTransactorSession) UpdateTuringToken(_turingToken libcommon.Address) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.UpdateTuringToken(&_BobaTuringCredit.TransactOpts, _turingToken)
}

// WithdrawRevenue is a paid mutator transaction binding the contract method 0x0ceff204.
//
// Solidity: function withdrawRevenue(uint256 _withdrawAmount) returns()
func (_BobaTuringCredit *BobaTuringCreditTransactor) WithdrawRevenue(opts *bind.TransactOpts, _withdrawAmount *big.Int) (types.Transaction, error) {
	return _BobaTuringCredit.contract.Transact(opts, "withdrawRevenue", _withdrawAmount)
}

// WithdrawRevenue is a paid mutator transaction binding the contract method 0x0ceff204.
//
// Solidity: function withdrawRevenue(uint256 _withdrawAmount) returns()
func (_BobaTuringCredit *BobaTuringCreditSession) WithdrawRevenue(_withdrawAmount *big.Int) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.WithdrawRevenue(&_BobaTuringCredit.TransactOpts, _withdrawAmount)
}

// WithdrawRevenue is a paid mutator transaction binding the contract method 0x0ceff204.
//
// Solidity: function withdrawRevenue(uint256 _withdrawAmount) returns()
func (_BobaTuringCredit *BobaTuringCreditTransactorSession) WithdrawRevenue(_withdrawAmount *big.Int) (types.Transaction, error) {
	return _BobaTuringCredit.Contract.WithdrawRevenue(&_BobaTuringCredit.TransactOpts, _withdrawAmount)
}

// BobaTuringCreditAddBalanceToIterator is returned from FilterAddBalanceTo and is used to iterate over the raw logs and unpacked data for AddBalanceTo events raised by the BobaTuringCredit contract.
type BobaTuringCreditAddBalanceToIterator struct {
	Event *BobaTuringCreditAddBalanceTo // Event containing the contract specifics and raw log

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
func (it *BobaTuringCreditAddBalanceToIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaTuringCreditAddBalanceTo)
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
		it.Event = new(BobaTuringCreditAddBalanceTo)
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
func (it *BobaTuringCreditAddBalanceToIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaTuringCreditAddBalanceToIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaTuringCreditAddBalanceTo represents a AddBalanceTo event raised by the BobaTuringCredit contract.
type BobaTuringCreditAddBalanceTo struct {
	Sender                libcommon.Address
	BalanceAmount         *big.Int
	HelperContractAddress libcommon.Address
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterAddBalanceTo is a free log retrieval operation binding the contract event 0x63611f4b2e0fff4acd8e17bd95ebb62a3bc834c76cf85e7a972a502990b6257a.
//
// Solidity: event AddBalanceTo(address sender, uint256 balanceAmount, address helperContractAddress)
func (_BobaTuringCredit *BobaTuringCreditFilterer) FilterAddBalanceTo(opts *bind.FilterOpts) (*BobaTuringCreditAddBalanceToIterator, error) {

	logs, sub, err := _BobaTuringCredit.contract.FilterLogs(opts, "AddBalanceTo")
	if err != nil {
		return nil, err
	}
	return &BobaTuringCreditAddBalanceToIterator{contract: _BobaTuringCredit.contract, event: "AddBalanceTo", logs: logs, sub: sub}, nil
}

// WatchAddBalanceTo is a free log subscription operation binding the contract event 0x63611f4b2e0fff4acd8e17bd95ebb62a3bc834c76cf85e7a972a502990b6257a.
//
// Solidity: event AddBalanceTo(address sender, uint256 balanceAmount, address helperContractAddress)
func (_BobaTuringCredit *BobaTuringCreditFilterer) WatchAddBalanceTo(opts *bind.WatchOpts, sink chan<- *BobaTuringCreditAddBalanceTo) (event.Subscription, error) {

	logs, sub, err := _BobaTuringCredit.contract.WatchLogs(opts, "AddBalanceTo")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaTuringCreditAddBalanceTo)
				if err := _BobaTuringCredit.contract.UnpackLog(event, "AddBalanceTo", log); err != nil {
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

// ParseAddBalanceTo is a log parse operation binding the contract event 0x63611f4b2e0fff4acd8e17bd95ebb62a3bc834c76cf85e7a972a502990b6257a.
//
// Solidity: event AddBalanceTo(address sender, uint256 balanceAmount, address helperContractAddress)
func (_BobaTuringCredit *BobaTuringCreditFilterer) ParseAddBalanceTo(log types.Log) (*BobaTuringCreditAddBalanceTo, error) {
	event := new(BobaTuringCreditAddBalanceTo)
	if err := _BobaTuringCredit.contract.UnpackLog(event, "AddBalanceTo", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaTuringCreditTransferOwnershipIterator is returned from FilterTransferOwnership and is used to iterate over the raw logs and unpacked data for TransferOwnership events raised by the BobaTuringCredit contract.
type BobaTuringCreditTransferOwnershipIterator struct {
	Event *BobaTuringCreditTransferOwnership // Event containing the contract specifics and raw log

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
func (it *BobaTuringCreditTransferOwnershipIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaTuringCreditTransferOwnership)
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
		it.Event = new(BobaTuringCreditTransferOwnership)
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
func (it *BobaTuringCreditTransferOwnershipIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaTuringCreditTransferOwnershipIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaTuringCreditTransferOwnership represents a TransferOwnership event raised by the BobaTuringCredit contract.
type BobaTuringCreditTransferOwnership struct {
	OldOwner libcommon.Address
	NewOwner libcommon.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterTransferOwnership is a free log retrieval operation binding the contract event 0x5c486528ec3e3f0ea91181cff8116f02bfa350e03b8b6f12e00765adbb5af85c.
//
// Solidity: event TransferOwnership(address oldOwner, address newOwner)
func (_BobaTuringCredit *BobaTuringCreditFilterer) FilterTransferOwnership(opts *bind.FilterOpts) (*BobaTuringCreditTransferOwnershipIterator, error) {

	logs, sub, err := _BobaTuringCredit.contract.FilterLogs(opts, "TransferOwnership")
	if err != nil {
		return nil, err
	}
	return &BobaTuringCreditTransferOwnershipIterator{contract: _BobaTuringCredit.contract, event: "TransferOwnership", logs: logs, sub: sub}, nil
}

// WatchTransferOwnership is a free log subscription operation binding the contract event 0x5c486528ec3e3f0ea91181cff8116f02bfa350e03b8b6f12e00765adbb5af85c.
//
// Solidity: event TransferOwnership(address oldOwner, address newOwner)
func (_BobaTuringCredit *BobaTuringCreditFilterer) WatchTransferOwnership(opts *bind.WatchOpts, sink chan<- *BobaTuringCreditTransferOwnership) (event.Subscription, error) {

	logs, sub, err := _BobaTuringCredit.contract.WatchLogs(opts, "TransferOwnership")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaTuringCreditTransferOwnership)
				if err := _BobaTuringCredit.contract.UnpackLog(event, "TransferOwnership", log); err != nil {
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

// ParseTransferOwnership is a log parse operation binding the contract event 0x5c486528ec3e3f0ea91181cff8116f02bfa350e03b8b6f12e00765adbb5af85c.
//
// Solidity: event TransferOwnership(address oldOwner, address newOwner)
func (_BobaTuringCredit *BobaTuringCreditFilterer) ParseTransferOwnership(log types.Log) (*BobaTuringCreditTransferOwnership, error) {
	event := new(BobaTuringCreditTransferOwnership)
	if err := _BobaTuringCredit.contract.UnpackLog(event, "TransferOwnership", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaTuringCreditWithdrawRevenueIterator is returned from FilterWithdrawRevenue and is used to iterate over the raw logs and unpacked data for WithdrawRevenue events raised by the BobaTuringCredit contract.
type BobaTuringCreditWithdrawRevenueIterator struct {
	Event *BobaTuringCreditWithdrawRevenue // Event containing the contract specifics and raw log

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
func (it *BobaTuringCreditWithdrawRevenueIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaTuringCreditWithdrawRevenue)
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
		it.Event = new(BobaTuringCreditWithdrawRevenue)
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
func (it *BobaTuringCreditWithdrawRevenueIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaTuringCreditWithdrawRevenueIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaTuringCreditWithdrawRevenue represents a WithdrawRevenue event raised by the BobaTuringCredit contract.
type BobaTuringCreditWithdrawRevenue struct {
	Sender         libcommon.Address
	WithdrawAmount *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterWithdrawRevenue is a free log retrieval operation binding the contract event 0x447d53be88e315476bdbe2e63cef309461f6305d09aada67641c29e6b897e301.
//
// Solidity: event WithdrawRevenue(address sender, uint256 withdrawAmount)
func (_BobaTuringCredit *BobaTuringCreditFilterer) FilterWithdrawRevenue(opts *bind.FilterOpts) (*BobaTuringCreditWithdrawRevenueIterator, error) {

	logs, sub, err := _BobaTuringCredit.contract.FilterLogs(opts, "WithdrawRevenue")
	if err != nil {
		return nil, err
	}
	return &BobaTuringCreditWithdrawRevenueIterator{contract: _BobaTuringCredit.contract, event: "WithdrawRevenue", logs: logs, sub: sub}, nil
}

// WatchWithdrawRevenue is a free log subscription operation binding the contract event 0x447d53be88e315476bdbe2e63cef309461f6305d09aada67641c29e6b897e301.
//
// Solidity: event WithdrawRevenue(address sender, uint256 withdrawAmount)
func (_BobaTuringCredit *BobaTuringCreditFilterer) WatchWithdrawRevenue(opts *bind.WatchOpts, sink chan<- *BobaTuringCreditWithdrawRevenue) (event.Subscription, error) {

	logs, sub, err := _BobaTuringCredit.contract.WatchLogs(opts, "WithdrawRevenue")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaTuringCreditWithdrawRevenue)
				if err := _BobaTuringCredit.contract.UnpackLog(event, "WithdrawRevenue", log); err != nil {
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

// ParseWithdrawRevenue is a log parse operation binding the contract event 0x447d53be88e315476bdbe2e63cef309461f6305d09aada67641c29e6b897e301.
//
// Solidity: event WithdrawRevenue(address sender, uint256 withdrawAmount)
func (_BobaTuringCredit *BobaTuringCreditFilterer) ParseWithdrawRevenue(log types.Log) (*BobaTuringCreditWithdrawRevenue, error) {
	event := new(BobaTuringCreditWithdrawRevenue)
	if err := _BobaTuringCredit.contract.UnpackLog(event, "WithdrawRevenue", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
