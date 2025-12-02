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

// ISharesCalculatorShareInfo is an auto generated low-level Go binding around an user-defined struct.
type ISharesCalculatorShareInfo struct {
	Recipient common.Address
	Amount    *big.Int
}

// FeeSplitterMetaData contains all meta data concerning the FeeSplitter contract.
var FeeSplitterMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"MAX_DISBURSEMENT_INTERVAL\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint128\",\"internalType\":\"uint128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"disburseFees\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"feeDisbursementInterval\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint128\",\"internalType\":\"uint128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_sharesCalculator\",\"type\":\"address\",\"internalType\":\"contractISharesCalculator\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"lastDisbursementTime\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint128\",\"internalType\":\"uint128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setFeeDisbursementInterval\",\"inputs\":[{\"name\":\"_newFeeDisbursementInterval\",\"type\":\"uint128\",\"internalType\":\"uint128\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setSharesCalculator\",\"inputs\":[{\"name\":\"_newSharesCalculator\",\"type\":\"address\",\"internalType\":\"contractISharesCalculator\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"sharesCalculator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractISharesCalculator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"FeeDisbursementIntervalUpdated\",\"inputs\":[{\"name\":\"oldFeeDisbursementInterval\",\"type\":\"uint128\",\"indexed\":false,\"internalType\":\"uint128\"},{\"name\":\"newFeeDisbursementInterval\",\"type\":\"uint128\",\"indexed\":false,\"internalType\":\"uint128\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"FeesDisbursed\",\"inputs\":[{\"name\":\"shareInfo\",\"type\":\"tuple[]\",\"indexed\":false,\"internalType\":\"structISharesCalculator.ShareInfo[]\",\"components\":[{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"addresspayable\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"grossRevenue\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"FeesReceived\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"newBalance\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SharesCalculatorUpdated\",\"inputs\":[{\"name\":\"oldSharesCalculator\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"newSharesCalculator\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"FeeSplitter_DisbursementIntervalNotReached\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_ExceedsMaxFeeDisbursementTime\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_FailedToSendToRevenueShareRecipient\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_FeeDisbursementIntervalCannotBeZero\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_FeeShareInfoEmpty\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_FeeVaultMustWithdrawToFeeSplitter\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_FeeVaultMustWithdrawToL2\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_NoFeesCollected\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_OnlyProxyAdminOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_ReceiveWindowClosed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_SenderNotApprovedVault\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_SharesCalculatorCannotBeZero\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeSplitter_SharesCalculatorMalformedOutput\",\"inputs\":[]}]",
	Bin: "0x6080604052348015600e575f80fd5b5060156019565b60d4565b5f54610100900460ff161560835760405162461bcd60e51b815260206004820152602760248201527f496e697469616c697a61626c653a20636f6e747261637420697320696e697469604482015266616c697a696e6760c81b606482015260840160405180910390fd5b5f5460ff908116101560d2575f805460ff191660ff9081179091556040519081527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b565b611344806100e15f395ff3fe608060405260043610610096575f3560e01c80637dfbd04911610066578063b87ea8d41161004c578063b87ea8d41461031b578063c4d66de81461032f578063d61a398b1461034e575f80fd5b80637dfbd049146102e55780637fc81bb7146102fc575f80fd5b80630a7617b3146101e55780630c0544a314610206578063394d27311461026857806354fd4d5014610290575f80fd5b366101e1577fe3007e9730850b5618eacb0537bef0cf0f1600267ae8549e472449d77b731e455c6100f3576040517f17617f6100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b337342000000000000000000000000000000000000111480159061012b57503373420000000000000000000000000000000000001914155b801561014b57503373420000000000000000000000000000000000001a14155b801561016b57503373420000000000000000000000000000000000001b14155b156101a2576040517f9dcde10900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040805134815247602082018190529133917f213e72af0d3613bd643cff3059f872c1015e6541624e37872bf95eefbaf220a8910160405180910390a2005b5f80fd5b3480156101f0575f80fd5b506102046101ff366004610f9e565b6103a4565b005b348015610211575f80fd5b506001546102429070010000000000000000000000000000000090046fffffffffffffffffffffffffffffffff1681565b6040516fffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b348015610273575f80fd5b50600154610242906fffffffffffffffffffffffffffffffff1681565b34801561029b575f80fd5b506102d86040518060400160405280600581526020017f312e302e3000000000000000000000000000000000000000000000000000000081525081565b60405161025f9190610fb9565b3480156102f0575f80fd5b506102426301e1338081565b348015610307575f80fd5b5061020461031636600461100c565b610566565b348015610326575f80fd5b50610204610759565b34801561033a575f80fd5b50610204610349366004610f9e565b610b3d565b348015610359575f80fd5b505f5461037f9062010000900473ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200161025f565b73420000000000000000000000000000000000001873ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610401573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610425919061103b565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610489576040517f38bac74200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b73ffffffffffffffffffffffffffffffffffffffff81166104d6576040517f99c6ec0800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b5f805473ffffffffffffffffffffffffffffffffffffffff838116620100008181027fffffffffffffffffffff0000000000000000000000000000000000000000ffff85161790945560408051949093049091168084526020840191909152917f16417cc372deec0caee5f52e2ad77a5f07b4591fd56b4ff31b6e20f817d4daeb91015b60405180910390a15050565b73420000000000000000000000000000000000001873ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105c3573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906105e7919061103b565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161461064b576040517f38bac74200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b806fffffffffffffffffffffffffffffffff165f03610696576040517fcf85916100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6301e133806fffffffffffffffffffffffffffffffff821611156106e6576040517f30b9f35e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600180546fffffffffffffffffffffffffffffffff8381167001000000000000000000000000000000008181028385161790945560408051949093049091168084526020840191909152917f4492086b630ed3846eec0979dd87a71c814ceb1c6dab80ab81e3450b21e4de28910161055a565b60015461078e906fffffffffffffffffffffffffffffffff700100000000000000000000000000000000820481169116611083565b6fffffffffffffffffffffffffffffffff164210156107d9576040517f1e4a9f3a00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600180547fffffffffffffffffffffffffffffffff0000000000000000000000000000000016426fffffffffffffffffffffffffffffffff1617815561081e90610d33565b5f61083c734200000000000000000000000000000000000011610d59565b90505f61085c734200000000000000000000000000000000000019610d59565b90505f61087c73420000000000000000000000000000000000001a610d59565b90505f61089c73420000000000000000000000000000000000001b610d59565b90506108a75f610d33565b5f82826108b486886110b3565b6108be91906110b3565b6108c891906110b3565b9050805f03610903576040517fc8972e5200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b5f80546040517f54e7f42d000000000000000000000000000000000000000000000000000000008152600481018890526024810187905260448101859052606481018690526201000090910473ffffffffffffffffffffffffffffffffffffffff16906354e7f42d906084015f60405180830381865afa158015610989573d5f803e3d5ffd5b505050506040513d5f823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01682016040526109ce919081019061116b565b905080515f03610a0a576040517f763970d600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b5f805b8251811015610ac1575f838281518110610a2957610a2961123a565b60200260200101515f015190505f848381518110610a4957610a4961123a565b6020026020010151602001519050805f03610a65575050610ab9565b5f610a708383610f56565b905080610aa9576040517fd68d1b1800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b610ab382866110b3565b94505050505b600101610a0d565b50828114610afb576040517f9c01eac000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b7f73f9a13241a1848ec157967f3a85601709353e616f1f2605d818c0f2d21774df8284604051610b2c929190611267565b60405180910390a150505050505050565b5f54610100900460ff1615808015610b5b57505f54600160ff909116105b80610b745750303b158015610b7457505f5460ff166001145b610c04576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a6564000000000000000000000000000000000000606482015260840160405180910390fd5b5f80547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790558015610c60575f80547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b5f80547fffffffffffffffffffff0000000000000000000000000000000000000000ffff166201000073ffffffffffffffffffffffffffffffffffffffff851602179055600180546fffffffffffffffffffffffffffffffff1672015180000000000000000000000000000000001790558015610d2f575f80547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200161055a565b5050565b807fe3007e9730850b5618eacb0537bef0cf0f1600267ae8549e472449d77b731e455d50565b5f60018273ffffffffffffffffffffffffffffffffffffffff166382356d8a6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610da5573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610dc99190611302565b6001811115610dda57610dda6112d5565b14610e11576040517fb4726cbe00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff166366d003ac6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610e71573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610e95919061103b565b73ffffffffffffffffffffffffffffffffffffffff1614610ee2576040517fc3380cef00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8173ffffffffffffffffffffffffffffffffffffffff16633ccfd60b6040518163ffffffff1660e01b81526004016020604051808303815f875af1158015610f2c573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610f509190611320565b92915050565b5f610f62835a84610f69565b9392505050565b5f805f805f858888f1949350505050565b73ffffffffffffffffffffffffffffffffffffffff81168114610f9b575f80fd5b50565b5f60208284031215610fae575f80fd5b8135610f6281610f7a565b602081525f82518060208401528060208501604085015e5f6040828501015260407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f83011684010191505092915050565b5f6020828403121561101c575f80fd5b81356fffffffffffffffffffffffffffffffff81168114610f62575f80fd5b5f6020828403121561104b575f80fd5b8151610f6281610f7a565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b6fffffffffffffffffffffffffffffffff8181168382160190808211156110ac576110ac611056565b5092915050565b80820180821115610f5057610f50611056565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b6040805190810167ffffffffffffffff81118282101715611116576111166110c6565b60405290565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff81118282101715611163576111636110c6565b604052919050565b5f602080838503121561117c575f80fd5b825167ffffffffffffffff80821115611193575f80fd5b818501915085601f8301126111a6575f80fd5b8151818111156111b8576111b86110c6565b6111c6848260051b0161111c565b818152848101925060069190911b8301840190878211156111e5575f80fd5b928401925b8184101561122f5760408489031215611201575f80fd5b6112096110f3565b845161121481610f7a565b815284860151868201528352604090930192918401916111ea565b979650505050505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b604080825283518282018190525f91906020906060850190828801855b828110156112bf578151805173ffffffffffffffffffffffffffffffffffffffff168552850151858501529285019290840190600101611284565b5050508093505050508260208301529392505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602160045260245ffd5b5f60208284031215611312575f80fd5b815160028110610f62575f80fd5b5f60208284031215611330575f80fd5b505191905056fea164736f6c6343000819000a",
}

// FeeSplitterABI is the input ABI used to generate the binding from.
// Deprecated: Use FeeSplitterMetaData.ABI instead.
var FeeSplitterABI = FeeSplitterMetaData.ABI

// FeeSplitterBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use FeeSplitterMetaData.Bin instead.
var FeeSplitterBin = FeeSplitterMetaData.Bin

// DeployFeeSplitter deploys a new Ethereum contract, binding an instance of FeeSplitter to it.
func DeployFeeSplitter(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *FeeSplitter, error) {
	parsed, err := FeeSplitterMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(FeeSplitterBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &FeeSplitter{FeeSplitterCaller: FeeSplitterCaller{contract: contract}, FeeSplitterTransactor: FeeSplitterTransactor{contract: contract}, FeeSplitterFilterer: FeeSplitterFilterer{contract: contract}}, nil
}

// FeeSplitter is an auto generated Go binding around an Ethereum contract.
type FeeSplitter struct {
	FeeSplitterCaller     // Read-only binding to the contract
	FeeSplitterTransactor // Write-only binding to the contract
	FeeSplitterFilterer   // Log filterer for contract events
}

// FeeSplitterCaller is an auto generated read-only Go binding around an Ethereum contract.
type FeeSplitterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FeeSplitterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FeeSplitterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FeeSplitterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FeeSplitterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FeeSplitterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FeeSplitterSession struct {
	Contract     *FeeSplitter      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// FeeSplitterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FeeSplitterCallerSession struct {
	Contract *FeeSplitterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// FeeSplitterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FeeSplitterTransactorSession struct {
	Contract     *FeeSplitterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// FeeSplitterRaw is an auto generated low-level Go binding around an Ethereum contract.
type FeeSplitterRaw struct {
	Contract *FeeSplitter // Generic contract binding to access the raw methods on
}

// FeeSplitterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FeeSplitterCallerRaw struct {
	Contract *FeeSplitterCaller // Generic read-only contract binding to access the raw methods on
}

// FeeSplitterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FeeSplitterTransactorRaw struct {
	Contract *FeeSplitterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFeeSplitter creates a new instance of FeeSplitter, bound to a specific deployed contract.
func NewFeeSplitter(address common.Address, backend bind.ContractBackend) (*FeeSplitter, error) {
	contract, err := bindFeeSplitter(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &FeeSplitter{FeeSplitterCaller: FeeSplitterCaller{contract: contract}, FeeSplitterTransactor: FeeSplitterTransactor{contract: contract}, FeeSplitterFilterer: FeeSplitterFilterer{contract: contract}}, nil
}

// NewFeeSplitterCaller creates a new read-only instance of FeeSplitter, bound to a specific deployed contract.
func NewFeeSplitterCaller(address common.Address, caller bind.ContractCaller) (*FeeSplitterCaller, error) {
	contract, err := bindFeeSplitter(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FeeSplitterCaller{contract: contract}, nil
}

// NewFeeSplitterTransactor creates a new write-only instance of FeeSplitter, bound to a specific deployed contract.
func NewFeeSplitterTransactor(address common.Address, transactor bind.ContractTransactor) (*FeeSplitterTransactor, error) {
	contract, err := bindFeeSplitter(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FeeSplitterTransactor{contract: contract}, nil
}

// NewFeeSplitterFilterer creates a new log filterer instance of FeeSplitter, bound to a specific deployed contract.
func NewFeeSplitterFilterer(address common.Address, filterer bind.ContractFilterer) (*FeeSplitterFilterer, error) {
	contract, err := bindFeeSplitter(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FeeSplitterFilterer{contract: contract}, nil
}

// bindFeeSplitter binds a generic wrapper to an already deployed contract.
func bindFeeSplitter(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := FeeSplitterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FeeSplitter *FeeSplitterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FeeSplitter.Contract.FeeSplitterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FeeSplitter *FeeSplitterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FeeSplitter.Contract.FeeSplitterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FeeSplitter *FeeSplitterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FeeSplitter.Contract.FeeSplitterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FeeSplitter *FeeSplitterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FeeSplitter.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FeeSplitter *FeeSplitterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FeeSplitter.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FeeSplitter *FeeSplitterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FeeSplitter.Contract.contract.Transact(opts, method, params...)
}

// MAXDISBURSEMENTINTERVAL is a free data retrieval call binding the contract method 0x7dfbd049.
//
// Solidity: function MAX_DISBURSEMENT_INTERVAL() view returns(uint128)
func (_FeeSplitter *FeeSplitterCaller) MAXDISBURSEMENTINTERVAL(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FeeSplitter.contract.Call(opts, &out, "MAX_DISBURSEMENT_INTERVAL")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXDISBURSEMENTINTERVAL is a free data retrieval call binding the contract method 0x7dfbd049.
//
// Solidity: function MAX_DISBURSEMENT_INTERVAL() view returns(uint128)
func (_FeeSplitter *FeeSplitterSession) MAXDISBURSEMENTINTERVAL() (*big.Int, error) {
	return _FeeSplitter.Contract.MAXDISBURSEMENTINTERVAL(&_FeeSplitter.CallOpts)
}

// MAXDISBURSEMENTINTERVAL is a free data retrieval call binding the contract method 0x7dfbd049.
//
// Solidity: function MAX_DISBURSEMENT_INTERVAL() view returns(uint128)
func (_FeeSplitter *FeeSplitterCallerSession) MAXDISBURSEMENTINTERVAL() (*big.Int, error) {
	return _FeeSplitter.Contract.MAXDISBURSEMENTINTERVAL(&_FeeSplitter.CallOpts)
}

// FeeDisbursementInterval is a free data retrieval call binding the contract method 0x0c0544a3.
//
// Solidity: function feeDisbursementInterval() view returns(uint128)
func (_FeeSplitter *FeeSplitterCaller) FeeDisbursementInterval(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FeeSplitter.contract.Call(opts, &out, "feeDisbursementInterval")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FeeDisbursementInterval is a free data retrieval call binding the contract method 0x0c0544a3.
//
// Solidity: function feeDisbursementInterval() view returns(uint128)
func (_FeeSplitter *FeeSplitterSession) FeeDisbursementInterval() (*big.Int, error) {
	return _FeeSplitter.Contract.FeeDisbursementInterval(&_FeeSplitter.CallOpts)
}

// FeeDisbursementInterval is a free data retrieval call binding the contract method 0x0c0544a3.
//
// Solidity: function feeDisbursementInterval() view returns(uint128)
func (_FeeSplitter *FeeSplitterCallerSession) FeeDisbursementInterval() (*big.Int, error) {
	return _FeeSplitter.Contract.FeeDisbursementInterval(&_FeeSplitter.CallOpts)
}

// LastDisbursementTime is a free data retrieval call binding the contract method 0x394d2731.
//
// Solidity: function lastDisbursementTime() view returns(uint128)
func (_FeeSplitter *FeeSplitterCaller) LastDisbursementTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FeeSplitter.contract.Call(opts, &out, "lastDisbursementTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastDisbursementTime is a free data retrieval call binding the contract method 0x394d2731.
//
// Solidity: function lastDisbursementTime() view returns(uint128)
func (_FeeSplitter *FeeSplitterSession) LastDisbursementTime() (*big.Int, error) {
	return _FeeSplitter.Contract.LastDisbursementTime(&_FeeSplitter.CallOpts)
}

// LastDisbursementTime is a free data retrieval call binding the contract method 0x394d2731.
//
// Solidity: function lastDisbursementTime() view returns(uint128)
func (_FeeSplitter *FeeSplitterCallerSession) LastDisbursementTime() (*big.Int, error) {
	return _FeeSplitter.Contract.LastDisbursementTime(&_FeeSplitter.CallOpts)
}

// SharesCalculator is a free data retrieval call binding the contract method 0xd61a398b.
//
// Solidity: function sharesCalculator() view returns(address)
func (_FeeSplitter *FeeSplitterCaller) SharesCalculator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FeeSplitter.contract.Call(opts, &out, "sharesCalculator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SharesCalculator is a free data retrieval call binding the contract method 0xd61a398b.
//
// Solidity: function sharesCalculator() view returns(address)
func (_FeeSplitter *FeeSplitterSession) SharesCalculator() (common.Address, error) {
	return _FeeSplitter.Contract.SharesCalculator(&_FeeSplitter.CallOpts)
}

// SharesCalculator is a free data retrieval call binding the contract method 0xd61a398b.
//
// Solidity: function sharesCalculator() view returns(address)
func (_FeeSplitter *FeeSplitterCallerSession) SharesCalculator() (common.Address, error) {
	return _FeeSplitter.Contract.SharesCalculator(&_FeeSplitter.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_FeeSplitter *FeeSplitterCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _FeeSplitter.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_FeeSplitter *FeeSplitterSession) Version() (string, error) {
	return _FeeSplitter.Contract.Version(&_FeeSplitter.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_FeeSplitter *FeeSplitterCallerSession) Version() (string, error) {
	return _FeeSplitter.Contract.Version(&_FeeSplitter.CallOpts)
}

// DisburseFees is a paid mutator transaction binding the contract method 0xb87ea8d4.
//
// Solidity: function disburseFees() returns()
func (_FeeSplitter *FeeSplitterTransactor) DisburseFees(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FeeSplitter.contract.Transact(opts, "disburseFees")
}

// DisburseFees is a paid mutator transaction binding the contract method 0xb87ea8d4.
//
// Solidity: function disburseFees() returns()
func (_FeeSplitter *FeeSplitterSession) DisburseFees() (*types.Transaction, error) {
	return _FeeSplitter.Contract.DisburseFees(&_FeeSplitter.TransactOpts)
}

// DisburseFees is a paid mutator transaction binding the contract method 0xb87ea8d4.
//
// Solidity: function disburseFees() returns()
func (_FeeSplitter *FeeSplitterTransactorSession) DisburseFees() (*types.Transaction, error) {
	return _FeeSplitter.Contract.DisburseFees(&_FeeSplitter.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _sharesCalculator) returns()
func (_FeeSplitter *FeeSplitterTransactor) Initialize(opts *bind.TransactOpts, _sharesCalculator common.Address) (*types.Transaction, error) {
	return _FeeSplitter.contract.Transact(opts, "initialize", _sharesCalculator)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _sharesCalculator) returns()
func (_FeeSplitter *FeeSplitterSession) Initialize(_sharesCalculator common.Address) (*types.Transaction, error) {
	return _FeeSplitter.Contract.Initialize(&_FeeSplitter.TransactOpts, _sharesCalculator)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _sharesCalculator) returns()
func (_FeeSplitter *FeeSplitterTransactorSession) Initialize(_sharesCalculator common.Address) (*types.Transaction, error) {
	return _FeeSplitter.Contract.Initialize(&_FeeSplitter.TransactOpts, _sharesCalculator)
}

// SetFeeDisbursementInterval is a paid mutator transaction binding the contract method 0x7fc81bb7.
//
// Solidity: function setFeeDisbursementInterval(uint128 _newFeeDisbursementInterval) returns()
func (_FeeSplitter *FeeSplitterTransactor) SetFeeDisbursementInterval(opts *bind.TransactOpts, _newFeeDisbursementInterval *big.Int) (*types.Transaction, error) {
	return _FeeSplitter.contract.Transact(opts, "setFeeDisbursementInterval", _newFeeDisbursementInterval)
}

// SetFeeDisbursementInterval is a paid mutator transaction binding the contract method 0x7fc81bb7.
//
// Solidity: function setFeeDisbursementInterval(uint128 _newFeeDisbursementInterval) returns()
func (_FeeSplitter *FeeSplitterSession) SetFeeDisbursementInterval(_newFeeDisbursementInterval *big.Int) (*types.Transaction, error) {
	return _FeeSplitter.Contract.SetFeeDisbursementInterval(&_FeeSplitter.TransactOpts, _newFeeDisbursementInterval)
}

// SetFeeDisbursementInterval is a paid mutator transaction binding the contract method 0x7fc81bb7.
//
// Solidity: function setFeeDisbursementInterval(uint128 _newFeeDisbursementInterval) returns()
func (_FeeSplitter *FeeSplitterTransactorSession) SetFeeDisbursementInterval(_newFeeDisbursementInterval *big.Int) (*types.Transaction, error) {
	return _FeeSplitter.Contract.SetFeeDisbursementInterval(&_FeeSplitter.TransactOpts, _newFeeDisbursementInterval)
}

// SetSharesCalculator is a paid mutator transaction binding the contract method 0x0a7617b3.
//
// Solidity: function setSharesCalculator(address _newSharesCalculator) returns()
func (_FeeSplitter *FeeSplitterTransactor) SetSharesCalculator(opts *bind.TransactOpts, _newSharesCalculator common.Address) (*types.Transaction, error) {
	return _FeeSplitter.contract.Transact(opts, "setSharesCalculator", _newSharesCalculator)
}

// SetSharesCalculator is a paid mutator transaction binding the contract method 0x0a7617b3.
//
// Solidity: function setSharesCalculator(address _newSharesCalculator) returns()
func (_FeeSplitter *FeeSplitterSession) SetSharesCalculator(_newSharesCalculator common.Address) (*types.Transaction, error) {
	return _FeeSplitter.Contract.SetSharesCalculator(&_FeeSplitter.TransactOpts, _newSharesCalculator)
}

// SetSharesCalculator is a paid mutator transaction binding the contract method 0x0a7617b3.
//
// Solidity: function setSharesCalculator(address _newSharesCalculator) returns()
func (_FeeSplitter *FeeSplitterTransactorSession) SetSharesCalculator(_newSharesCalculator common.Address) (*types.Transaction, error) {
	return _FeeSplitter.Contract.SetSharesCalculator(&_FeeSplitter.TransactOpts, _newSharesCalculator)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_FeeSplitter *FeeSplitterTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FeeSplitter.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_FeeSplitter *FeeSplitterSession) Receive() (*types.Transaction, error) {
	return _FeeSplitter.Contract.Receive(&_FeeSplitter.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_FeeSplitter *FeeSplitterTransactorSession) Receive() (*types.Transaction, error) {
	return _FeeSplitter.Contract.Receive(&_FeeSplitter.TransactOpts)
}

// FeeSplitterFeeDisbursementIntervalUpdatedIterator is returned from FilterFeeDisbursementIntervalUpdated and is used to iterate over the raw logs and unpacked data for FeeDisbursementIntervalUpdated events raised by the FeeSplitter contract.
type FeeSplitterFeeDisbursementIntervalUpdatedIterator struct {
	Event *FeeSplitterFeeDisbursementIntervalUpdated // Event containing the contract specifics and raw log

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
func (it *FeeSplitterFeeDisbursementIntervalUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FeeSplitterFeeDisbursementIntervalUpdated)
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
		it.Event = new(FeeSplitterFeeDisbursementIntervalUpdated)
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
func (it *FeeSplitterFeeDisbursementIntervalUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FeeSplitterFeeDisbursementIntervalUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FeeSplitterFeeDisbursementIntervalUpdated represents a FeeDisbursementIntervalUpdated event raised by the FeeSplitter contract.
type FeeSplitterFeeDisbursementIntervalUpdated struct {
	OldFeeDisbursementInterval *big.Int
	NewFeeDisbursementInterval *big.Int
	Raw                        types.Log // Blockchain specific contextual infos
}

// FilterFeeDisbursementIntervalUpdated is a free log retrieval operation binding the contract event 0x4492086b630ed3846eec0979dd87a71c814ceb1c6dab80ab81e3450b21e4de28.
//
// Solidity: event FeeDisbursementIntervalUpdated(uint128 oldFeeDisbursementInterval, uint128 newFeeDisbursementInterval)
func (_FeeSplitter *FeeSplitterFilterer) FilterFeeDisbursementIntervalUpdated(opts *bind.FilterOpts) (*FeeSplitterFeeDisbursementIntervalUpdatedIterator, error) {

	logs, sub, err := _FeeSplitter.contract.FilterLogs(opts, "FeeDisbursementIntervalUpdated")
	if err != nil {
		return nil, err
	}
	return &FeeSplitterFeeDisbursementIntervalUpdatedIterator{contract: _FeeSplitter.contract, event: "FeeDisbursementIntervalUpdated", logs: logs, sub: sub}, nil
}

// WatchFeeDisbursementIntervalUpdated is a free log subscription operation binding the contract event 0x4492086b630ed3846eec0979dd87a71c814ceb1c6dab80ab81e3450b21e4de28.
//
// Solidity: event FeeDisbursementIntervalUpdated(uint128 oldFeeDisbursementInterval, uint128 newFeeDisbursementInterval)
func (_FeeSplitter *FeeSplitterFilterer) WatchFeeDisbursementIntervalUpdated(opts *bind.WatchOpts, sink chan<- *FeeSplitterFeeDisbursementIntervalUpdated) (event.Subscription, error) {

	logs, sub, err := _FeeSplitter.contract.WatchLogs(opts, "FeeDisbursementIntervalUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FeeSplitterFeeDisbursementIntervalUpdated)
				if err := _FeeSplitter.contract.UnpackLog(event, "FeeDisbursementIntervalUpdated", log); err != nil {
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

// ParseFeeDisbursementIntervalUpdated is a log parse operation binding the contract event 0x4492086b630ed3846eec0979dd87a71c814ceb1c6dab80ab81e3450b21e4de28.
//
// Solidity: event FeeDisbursementIntervalUpdated(uint128 oldFeeDisbursementInterval, uint128 newFeeDisbursementInterval)
func (_FeeSplitter *FeeSplitterFilterer) ParseFeeDisbursementIntervalUpdated(log types.Log) (*FeeSplitterFeeDisbursementIntervalUpdated, error) {
	event := new(FeeSplitterFeeDisbursementIntervalUpdated)
	if err := _FeeSplitter.contract.UnpackLog(event, "FeeDisbursementIntervalUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FeeSplitterFeesDisbursedIterator is returned from FilterFeesDisbursed and is used to iterate over the raw logs and unpacked data for FeesDisbursed events raised by the FeeSplitter contract.
type FeeSplitterFeesDisbursedIterator struct {
	Event *FeeSplitterFeesDisbursed // Event containing the contract specifics and raw log

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
func (it *FeeSplitterFeesDisbursedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FeeSplitterFeesDisbursed)
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
		it.Event = new(FeeSplitterFeesDisbursed)
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
func (it *FeeSplitterFeesDisbursedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FeeSplitterFeesDisbursedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FeeSplitterFeesDisbursed represents a FeesDisbursed event raised by the FeeSplitter contract.
type FeeSplitterFeesDisbursed struct {
	ShareInfo    []ISharesCalculatorShareInfo
	GrossRevenue *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterFeesDisbursed is a free log retrieval operation binding the contract event 0x73f9a13241a1848ec157967f3a85601709353e616f1f2605d818c0f2d21774df.
//
// Solidity: event FeesDisbursed((address,uint256)[] shareInfo, uint256 grossRevenue)
func (_FeeSplitter *FeeSplitterFilterer) FilterFeesDisbursed(opts *bind.FilterOpts) (*FeeSplitterFeesDisbursedIterator, error) {

	logs, sub, err := _FeeSplitter.contract.FilterLogs(opts, "FeesDisbursed")
	if err != nil {
		return nil, err
	}
	return &FeeSplitterFeesDisbursedIterator{contract: _FeeSplitter.contract, event: "FeesDisbursed", logs: logs, sub: sub}, nil
}

// WatchFeesDisbursed is a free log subscription operation binding the contract event 0x73f9a13241a1848ec157967f3a85601709353e616f1f2605d818c0f2d21774df.
//
// Solidity: event FeesDisbursed((address,uint256)[] shareInfo, uint256 grossRevenue)
func (_FeeSplitter *FeeSplitterFilterer) WatchFeesDisbursed(opts *bind.WatchOpts, sink chan<- *FeeSplitterFeesDisbursed) (event.Subscription, error) {

	logs, sub, err := _FeeSplitter.contract.WatchLogs(opts, "FeesDisbursed")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FeeSplitterFeesDisbursed)
				if err := _FeeSplitter.contract.UnpackLog(event, "FeesDisbursed", log); err != nil {
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

// ParseFeesDisbursed is a log parse operation binding the contract event 0x73f9a13241a1848ec157967f3a85601709353e616f1f2605d818c0f2d21774df.
//
// Solidity: event FeesDisbursed((address,uint256)[] shareInfo, uint256 grossRevenue)
func (_FeeSplitter *FeeSplitterFilterer) ParseFeesDisbursed(log types.Log) (*FeeSplitterFeesDisbursed, error) {
	event := new(FeeSplitterFeesDisbursed)
	if err := _FeeSplitter.contract.UnpackLog(event, "FeesDisbursed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FeeSplitterFeesReceivedIterator is returned from FilterFeesReceived and is used to iterate over the raw logs and unpacked data for FeesReceived events raised by the FeeSplitter contract.
type FeeSplitterFeesReceivedIterator struct {
	Event *FeeSplitterFeesReceived // Event containing the contract specifics and raw log

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
func (it *FeeSplitterFeesReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FeeSplitterFeesReceived)
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
		it.Event = new(FeeSplitterFeesReceived)
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
func (it *FeeSplitterFeesReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FeeSplitterFeesReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FeeSplitterFeesReceived represents a FeesReceived event raised by the FeeSplitter contract.
type FeeSplitterFeesReceived struct {
	Sender     common.Address
	Amount     *big.Int
	NewBalance *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterFeesReceived is a free log retrieval operation binding the contract event 0x213e72af0d3613bd643cff3059f872c1015e6541624e37872bf95eefbaf220a8.
//
// Solidity: event FeesReceived(address indexed sender, uint256 amount, uint256 newBalance)
func (_FeeSplitter *FeeSplitterFilterer) FilterFeesReceived(opts *bind.FilterOpts, sender []common.Address) (*FeeSplitterFeesReceivedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _FeeSplitter.contract.FilterLogs(opts, "FeesReceived", senderRule)
	if err != nil {
		return nil, err
	}
	return &FeeSplitterFeesReceivedIterator{contract: _FeeSplitter.contract, event: "FeesReceived", logs: logs, sub: sub}, nil
}

// WatchFeesReceived is a free log subscription operation binding the contract event 0x213e72af0d3613bd643cff3059f872c1015e6541624e37872bf95eefbaf220a8.
//
// Solidity: event FeesReceived(address indexed sender, uint256 amount, uint256 newBalance)
func (_FeeSplitter *FeeSplitterFilterer) WatchFeesReceived(opts *bind.WatchOpts, sink chan<- *FeeSplitterFeesReceived, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _FeeSplitter.contract.WatchLogs(opts, "FeesReceived", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FeeSplitterFeesReceived)
				if err := _FeeSplitter.contract.UnpackLog(event, "FeesReceived", log); err != nil {
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

// ParseFeesReceived is a log parse operation binding the contract event 0x213e72af0d3613bd643cff3059f872c1015e6541624e37872bf95eefbaf220a8.
//
// Solidity: event FeesReceived(address indexed sender, uint256 amount, uint256 newBalance)
func (_FeeSplitter *FeeSplitterFilterer) ParseFeesReceived(log types.Log) (*FeeSplitterFeesReceived, error) {
	event := new(FeeSplitterFeesReceived)
	if err := _FeeSplitter.contract.UnpackLog(event, "FeesReceived", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FeeSplitterInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the FeeSplitter contract.
type FeeSplitterInitializedIterator struct {
	Event *FeeSplitterInitialized // Event containing the contract specifics and raw log

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
func (it *FeeSplitterInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FeeSplitterInitialized)
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
		it.Event = new(FeeSplitterInitialized)
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
func (it *FeeSplitterInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FeeSplitterInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FeeSplitterInitialized represents a Initialized event raised by the FeeSplitter contract.
type FeeSplitterInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_FeeSplitter *FeeSplitterFilterer) FilterInitialized(opts *bind.FilterOpts) (*FeeSplitterInitializedIterator, error) {

	logs, sub, err := _FeeSplitter.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &FeeSplitterInitializedIterator{contract: _FeeSplitter.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_FeeSplitter *FeeSplitterFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *FeeSplitterInitialized) (event.Subscription, error) {

	logs, sub, err := _FeeSplitter.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FeeSplitterInitialized)
				if err := _FeeSplitter.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_FeeSplitter *FeeSplitterFilterer) ParseInitialized(log types.Log) (*FeeSplitterInitialized, error) {
	event := new(FeeSplitterInitialized)
	if err := _FeeSplitter.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FeeSplitterSharesCalculatorUpdatedIterator is returned from FilterSharesCalculatorUpdated and is used to iterate over the raw logs and unpacked data for SharesCalculatorUpdated events raised by the FeeSplitter contract.
type FeeSplitterSharesCalculatorUpdatedIterator struct {
	Event *FeeSplitterSharesCalculatorUpdated // Event containing the contract specifics and raw log

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
func (it *FeeSplitterSharesCalculatorUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FeeSplitterSharesCalculatorUpdated)
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
		it.Event = new(FeeSplitterSharesCalculatorUpdated)
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
func (it *FeeSplitterSharesCalculatorUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FeeSplitterSharesCalculatorUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FeeSplitterSharesCalculatorUpdated represents a SharesCalculatorUpdated event raised by the FeeSplitter contract.
type FeeSplitterSharesCalculatorUpdated struct {
	OldSharesCalculator common.Address
	NewSharesCalculator common.Address
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterSharesCalculatorUpdated is a free log retrieval operation binding the contract event 0x16417cc372deec0caee5f52e2ad77a5f07b4591fd56b4ff31b6e20f817d4daeb.
//
// Solidity: event SharesCalculatorUpdated(address oldSharesCalculator, address newSharesCalculator)
func (_FeeSplitter *FeeSplitterFilterer) FilterSharesCalculatorUpdated(opts *bind.FilterOpts) (*FeeSplitterSharesCalculatorUpdatedIterator, error) {

	logs, sub, err := _FeeSplitter.contract.FilterLogs(opts, "SharesCalculatorUpdated")
	if err != nil {
		return nil, err
	}
	return &FeeSplitterSharesCalculatorUpdatedIterator{contract: _FeeSplitter.contract, event: "SharesCalculatorUpdated", logs: logs, sub: sub}, nil
}

// WatchSharesCalculatorUpdated is a free log subscription operation binding the contract event 0x16417cc372deec0caee5f52e2ad77a5f07b4591fd56b4ff31b6e20f817d4daeb.
//
// Solidity: event SharesCalculatorUpdated(address oldSharesCalculator, address newSharesCalculator)
func (_FeeSplitter *FeeSplitterFilterer) WatchSharesCalculatorUpdated(opts *bind.WatchOpts, sink chan<- *FeeSplitterSharesCalculatorUpdated) (event.Subscription, error) {

	logs, sub, err := _FeeSplitter.contract.WatchLogs(opts, "SharesCalculatorUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FeeSplitterSharesCalculatorUpdated)
				if err := _FeeSplitter.contract.UnpackLog(event, "SharesCalculatorUpdated", log); err != nil {
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

// ParseSharesCalculatorUpdated is a log parse operation binding the contract event 0x16417cc372deec0caee5f52e2ad77a5f07b4591fd56b4ff31b6e20f817d4daeb.
//
// Solidity: event SharesCalculatorUpdated(address oldSharesCalculator, address newSharesCalculator)
func (_FeeSplitter *FeeSplitterFilterer) ParseSharesCalculatorUpdated(log types.Log) (*FeeSplitterSharesCalculatorUpdated, error) {
	event := new(FeeSplitterSharesCalculatorUpdated)
	if err := _FeeSplitter.contract.UnpackLog(event, "SharesCalculatorUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
