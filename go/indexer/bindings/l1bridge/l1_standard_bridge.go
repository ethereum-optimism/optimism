// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package l1bridge

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

// L1StandardBridgeMetaData contains all meta data concerning the L1StandardBridge contract.
var L1StandardBridgeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_l1Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_l2Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"ERC20DepositInitiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_l1Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_l2Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"ERC20WithdrawalFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"ETHDepositInitiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"ETHWithdrawalFinalized\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1Token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_l2Token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_l2Gas\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositERC20\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1Token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_l2Token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_l2Gas\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositERC20To\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"_l2Gas\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"_l2Gas\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositETHTo\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"deposits\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"donateETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1Token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_l2Token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"finalizeERC20Withdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"finalizeETHWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1messenger\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_l2TokenBridge\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2TokenBridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x608060405234801561001057600080fd5b50600080546001600160a01b031916905561199e806100306000396000f3fe6080604052600436106100c05760003560e01c80638b4c40b0116100745780639a2ac6d51161004e5780639a2ac6d514610288578063a9f9e6751461029b578063b1a1a882146102bb57600080fd5b80638b4c40b0146101375780638f601f661461021557806391c49bf81461025b57600080fd5b8063485cc955116100a5578063485cc955146101b557806358a997f6146101d5578063838b2520146101f557600080fd5b80631532ec341461013e5780633cb747bf1461015e57600080fd5b3661013957333b156101195760405162461bcd60e51b815260206004820152600f60248201527f4163636f756e74206e6f7420454f41000000000000000000000000000000000060448201526064015b60405180910390fd5b610137333362030d40604051806020016040528060008152506102ce565b005b600080fd5b34801561014a57600080fd5b50610137610159366004611357565b61041a565b34801561016a57600080fd5b5060005461018b9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b3480156101c157600080fd5b506101376101d03660046113ca565b610790565b3480156101e157600080fd5b506101376101f036600461141c565b61086f565b34801561020157600080fd5b5061013761021036600461149f565b6108d6565b34801561022157600080fd5b5061024d6102303660046113ca565b600260209081526000928352604080842090915290825290205481565b6040519081526020016101ac565b34801561026757600080fd5b5060015461018b9073ffffffffffffffffffffffffffffffffffffffff1681565b610137610296366004611535565b6108ef565b3480156102a757600080fd5b506101376102b6366004611598565b610937565b6101376102c9366004611611565b610c59565b600063662a633a60e01b600073deaddeaddeaddeaddeaddeaddeaddeaddead000087873487604051602401610308969594939291906116da565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff00000000000000000000000000000000000000000000000000000000909316929092179091526001549091506103ac9073ffffffffffffffffffffffffffffffffffffffff168483610cef565b8373ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff167f35d79ab81f2b2017e19afb5c5571778877782d7a8786f5907f93b0f4702f4f23348560405161040b929190611735565b60405180910390a35050505050565b60015473ffffffffffffffffffffffffffffffffffffffff1661045260005473ffffffffffffffffffffffffffffffffffffffff1690565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146104f25760405162461bcd60e51b815260206004820152602e60248201527f4f564d5f58434841494e3a206d657373656e67657220636f6e7472616374207560448201527f6e61757468656e746963617465640000000000000000000000000000000000006064820152608401610110565b8073ffffffffffffffffffffffffffffffffffffffff1661052860005473ffffffffffffffffffffffffffffffffffffffff1690565b73ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b815260040160206040518083038186803b15801561056d57600080fd5b505afa158015610581573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105a5919061174e565b73ffffffffffffffffffffffffffffffffffffffff161461062e5760405162461bcd60e51b815260206004820152603060248201527f4f564d5f58434841494e3a2077726f6e672073656e646572206f662063726f7360448201527f732d646f6d61696e206d657373616765000000000000000000000000000000006064820152608401610110565b6040805160008082526020820190925273ffffffffffffffffffffffffffffffffffffffff8716908690604051610665919061176b565b60006040518083038185875af1925050503d80600081146106a2576040519150601f19603f3d011682016040523d82523d6000602084013e6106a7565b606091505b505090508061071e5760405162461bcd60e51b815260206004820152603460248201527f5472616e7366657248656c7065723a3a736166655472616e736665724554483a60448201527f20455448207472616e73666572206661696c65640000000000000000000000006064820152608401610110565b8573ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff167f2ac69ee804d9a7a0984249f508dfab7cb2534b465b6ce1580f99a38ba9c5e63187878760405161077f939291906117d0565b60405180910390a350505050505050565b60005473ffffffffffffffffffffffffffffffffffffffff161561081c5760405162461bcd60e51b815260206004820152602660248201527f436f6e74726163742068617320616c7265616479206265656e20696e6974696160448201527f6c697a65642e00000000000000000000000000000000000000000000000000006064820152608401610110565b6000805473ffffffffffffffffffffffffffffffffffffffff9384167fffffffffffffffffffffffff00000000000000000000000000000000000000009182161790915560018054929093169116179055565b333b156108be5760405162461bcd60e51b815260206004820152600f60248201527f4163636f756e74206e6f7420454f4100000000000000000000000000000000006044820152606401610110565b6108ce8686333388888888610d77565b505050505050565b6108e68787338888888888610d77565b50505050505050565b61093133858585858080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152506102ce92505050565b50505050565b60015473ffffffffffffffffffffffffffffffffffffffff1661096f60005473ffffffffffffffffffffffffffffffffffffffff1690565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610a0f5760405162461bcd60e51b815260206004820152602e60248201527f4f564d5f58434841494e3a206d657373656e67657220636f6e7472616374207560448201527f6e61757468656e746963617465640000000000000000000000000000000000006064820152608401610110565b8073ffffffffffffffffffffffffffffffffffffffff16610a4560005473ffffffffffffffffffffffffffffffffffffffff1690565b73ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b815260040160206040518083038186803b158015610a8a57600080fd5b505afa158015610a9e573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610ac2919061174e565b73ffffffffffffffffffffffffffffffffffffffff1614610b4b5760405162461bcd60e51b815260206004820152603060248201527f4f564d5f58434841494e3a2077726f6e672073656e646572206f662063726f7360448201527f732d646f6d61696e206d657373616765000000000000000000000000000000006064820152608401610110565b73ffffffffffffffffffffffffffffffffffffffff8089166000908152600260209081526040808320938b1683529290522054610b89908590611822565b73ffffffffffffffffffffffffffffffffffffffff808a166000818152600260209081526040808320948d1683529390529190912091909155610bcd908686610f27565b8573ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff168973ffffffffffffffffffffffffffffffffffffffff167f3ceee06c1e37648fcbb6ed52e17b3e1f275a1f8c7b22a84b2b84732431e046b388888888604051610c479493929190611839565b60405180910390a45050505050505050565b333b15610ca85760405162461bcd60e51b815260206004820152600f60248201527f4163636f756e74206e6f7420454f4100000000000000000000000000000000006044820152606401610110565b610cea33338585858080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152506102ce92505050565b505050565b6000546040517f3dbb202b00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff90911690633dbb202b90610d4990869085908790600401611879565b600060405180830381600087803b158015610d6357600080fd5b505af11580156108e6573d6000803e3d6000fd5b610d9973ffffffffffffffffffffffffffffffffffffffff8916873087610ffb565b600063662a633a60e01b89898989898888604051602401610dc097969594939291906118be565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff0000000000000000000000000000000000000000000000000000000090931692909217909152600154909150610e649073ffffffffffffffffffffffffffffffffffffffff168583610cef565b73ffffffffffffffffffffffffffffffffffffffff808a166000908152600260209081526040808320938c1683529290522054610ea290869061191b565b73ffffffffffffffffffffffffffffffffffffffff808b1660008181526002602090815260408083208e86168085529252918290209490945551918a1692917f718594027abd4eaed59f95162563e0cc6d0e8d5b86b1c7be8b1b0ac3343d039690610f14908b908b908a908a90611839565b60405180910390a4505050505050505050565b60405173ffffffffffffffffffffffffffffffffffffffff8316602482015260448101829052610cea9084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff0000000000000000000000000000000000000000000000000000000090931692909217909152611059565b60405173ffffffffffffffffffffffffffffffffffffffff808516602483015283166044820152606481018290526109319085907f23b872dd0000000000000000000000000000000000000000000000000000000090608401610f79565b60006110bb826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c65648152508573ffffffffffffffffffffffffffffffffffffffff1661114b9092919063ffffffff16565b805190915015610cea57808060200190518101906110d99190611933565b610cea5760405162461bcd60e51b815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f742073756363656564000000000000000000000000000000000000000000006064820152608401610110565b606061115a8484600085611164565b90505b9392505050565b6060824710156111dc5760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c00000000000000000000000000000000000000000000000000006064820152608401610110565b843b61122a5760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e74726163740000006044820152606401610110565b6000808673ffffffffffffffffffffffffffffffffffffffff168587604051611253919061176b565b60006040518083038185875af1925050503d8060008114611290576040519150601f19603f3d011682016040523d82523d6000602084013e611295565b606091505b50915091506112a58282866112b0565b979650505050505050565b606083156112bf57508161115d565b8251156112cf5782518084602001fd5b8160405162461bcd60e51b81526004016101109190611955565b73ffffffffffffffffffffffffffffffffffffffff8116811461130b57600080fd5b50565b60008083601f84011261132057600080fd5b50813567ffffffffffffffff81111561133857600080fd5b60208301915083602082850101111561135057600080fd5b9250929050565b60008060008060006080868803121561136f57600080fd5b853561137a816112e9565b9450602086013561138a816112e9565b935060408601359250606086013567ffffffffffffffff8111156113ad57600080fd5b6113b98882890161130e565b969995985093965092949392505050565b600080604083850312156113dd57600080fd5b82356113e8816112e9565b915060208301356113f8816112e9565b809150509250929050565b803563ffffffff8116811461141757600080fd5b919050565b60008060008060008060a0878903121561143557600080fd5b8635611440816112e9565b95506020870135611450816112e9565b94506040870135935061146560608801611403565b9250608087013567ffffffffffffffff81111561148157600080fd5b61148d89828a0161130e565b979a9699509497509295939492505050565b600080600080600080600060c0888a0312156114ba57600080fd5b87356114c5816112e9565b965060208801356114d5816112e9565b955060408801356114e5816112e9565b9450606088013593506114fa60808901611403565b925060a088013567ffffffffffffffff81111561151657600080fd5b6115228a828b0161130e565b989b979a50959850939692959293505050565b6000806000806060858703121561154b57600080fd5b8435611556816112e9565b935061156460208601611403565b9250604085013567ffffffffffffffff81111561158057600080fd5b61158c8782880161130e565b95989497509550505050565b600080600080600080600060c0888a0312156115b357600080fd5b87356115be816112e9565b965060208801356115ce816112e9565b955060408801356115de816112e9565b945060608801356115ee816112e9565b93506080880135925060a088013567ffffffffffffffff81111561151657600080fd5b60008060006040848603121561162657600080fd5b61162f84611403565b9250602084013567ffffffffffffffff81111561164b57600080fd5b6116578682870161130e565b9497909650939450505050565b60005b8381101561167f578181015183820152602001611667565b838111156109315750506000910152565b600081518084526116a8816020860160208601611664565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b600073ffffffffffffffffffffffffffffffffffffffff80891683528088166020840152808716604084015280861660608401525083608083015260c060a083015261172960c0830184611690565b98975050505050505050565b82815260406020820152600061115a6040830184611690565b60006020828403121561176057600080fd5b815161115d816112e9565b6000825161177d818460208701611664565b9190910192915050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b8381526040602082015260006117ea604083018486611787565b95945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015611834576118346117f3565b500390565b73ffffffffffffffffffffffffffffffffffffffff8516815283602082015260606040820152600061186f606083018486611787565b9695505050505050565b73ffffffffffffffffffffffffffffffffffffffff841681526060602082015260006118a86060830185611690565b905063ffffffff83166040830152949350505050565b600073ffffffffffffffffffffffffffffffffffffffff808a1683528089166020840152808816604084015280871660608401525084608083015260c060a083015261190e60c083018486611787565b9998505050505050505050565b6000821982111561192e5761192e6117f3565b500190565b60006020828403121561194557600080fd5b8151801515811461115d57600080fd5b60208152600061115d602083018461169056fea2646970667358221220ab0015e7cee64db6dc95e6eb0899ae4017c78ab54ebce874e3b018a7bc88850364736f6c63430008090033",
}

// L1StandardBridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use L1StandardBridgeMetaData.ABI instead.
var L1StandardBridgeABI = L1StandardBridgeMetaData.ABI

// L1StandardBridgeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1StandardBridgeMetaData.Bin instead.
var L1StandardBridgeBin = L1StandardBridgeMetaData.Bin

// DeployL1StandardBridge deploys a new Ethereum contract, binding an instance of L1StandardBridge to it.
func DeployL1StandardBridge(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L1StandardBridge, error) {
	parsed, err := L1StandardBridgeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1StandardBridgeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1StandardBridge{L1StandardBridgeCaller: L1StandardBridgeCaller{contract: contract}, L1StandardBridgeTransactor: L1StandardBridgeTransactor{contract: contract}, L1StandardBridgeFilterer: L1StandardBridgeFilterer{contract: contract}}, nil
}

// L1StandardBridge is an auto generated Go binding around an Ethereum contract.
type L1StandardBridge struct {
	L1StandardBridgeCaller     // Read-only binding to the contract
	L1StandardBridgeTransactor // Write-only binding to the contract
	L1StandardBridgeFilterer   // Log filterer for contract events
}

// L1StandardBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1StandardBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1StandardBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1StandardBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1StandardBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1StandardBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1StandardBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1StandardBridgeSession struct {
	Contract     *L1StandardBridge // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1StandardBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1StandardBridgeCallerSession struct {
	Contract *L1StandardBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// L1StandardBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1StandardBridgeTransactorSession struct {
	Contract     *L1StandardBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// L1StandardBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1StandardBridgeRaw struct {
	Contract *L1StandardBridge // Generic contract binding to access the raw methods on
}

// L1StandardBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1StandardBridgeCallerRaw struct {
	Contract *L1StandardBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// L1StandardBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1StandardBridgeTransactorRaw struct {
	Contract *L1StandardBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1StandardBridge creates a new instance of L1StandardBridge, bound to a specific deployed contract.
func NewL1StandardBridge(address common.Address, backend bind.ContractBackend) (*L1StandardBridge, error) {
	contract, err := bindL1StandardBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1StandardBridge{L1StandardBridgeCaller: L1StandardBridgeCaller{contract: contract}, L1StandardBridgeTransactor: L1StandardBridgeTransactor{contract: contract}, L1StandardBridgeFilterer: L1StandardBridgeFilterer{contract: contract}}, nil
}

// NewL1StandardBridgeCaller creates a new read-only instance of L1StandardBridge, bound to a specific deployed contract.
func NewL1StandardBridgeCaller(address common.Address, caller bind.ContractCaller) (*L1StandardBridgeCaller, error) {
	contract, err := bindL1StandardBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1StandardBridgeCaller{contract: contract}, nil
}

// NewL1StandardBridgeTransactor creates a new write-only instance of L1StandardBridge, bound to a specific deployed contract.
func NewL1StandardBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*L1StandardBridgeTransactor, error) {
	contract, err := bindL1StandardBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1StandardBridgeTransactor{contract: contract}, nil
}

// NewL1StandardBridgeFilterer creates a new log filterer instance of L1StandardBridge, bound to a specific deployed contract.
func NewL1StandardBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*L1StandardBridgeFilterer, error) {
	contract, err := bindL1StandardBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1StandardBridgeFilterer{contract: contract}, nil
}

// bindL1StandardBridge binds a generic wrapper to an already deployed contract.
func bindL1StandardBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1StandardBridgeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1StandardBridge *L1StandardBridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1StandardBridge.Contract.L1StandardBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1StandardBridge *L1StandardBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.L1StandardBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1StandardBridge *L1StandardBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.L1StandardBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1StandardBridge *L1StandardBridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1StandardBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1StandardBridge *L1StandardBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1StandardBridge *L1StandardBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.contract.Transact(opts, method, params...)
}

// Deposits is a free data retrieval call binding the contract method 0x8f601f66.
//
// Solidity: function deposits(address , address ) view returns(uint256)
func (_L1StandardBridge *L1StandardBridgeCaller) Deposits(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _L1StandardBridge.contract.Call(opts, &out, "deposits", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Deposits is a free data retrieval call binding the contract method 0x8f601f66.
//
// Solidity: function deposits(address , address ) view returns(uint256)
func (_L1StandardBridge *L1StandardBridgeSession) Deposits(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _L1StandardBridge.Contract.Deposits(&_L1StandardBridge.CallOpts, arg0, arg1)
}

// Deposits is a free data retrieval call binding the contract method 0x8f601f66.
//
// Solidity: function deposits(address , address ) view returns(uint256)
func (_L1StandardBridge *L1StandardBridgeCallerSession) Deposits(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _L1StandardBridge.Contract.Deposits(&_L1StandardBridge.CallOpts, arg0, arg1)
}

// L2TokenBridge is a free data retrieval call binding the contract method 0x91c49bf8.
//
// Solidity: function l2TokenBridge() view returns(address)
func (_L1StandardBridge *L1StandardBridgeCaller) L2TokenBridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1StandardBridge.contract.Call(opts, &out, "l2TokenBridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2TokenBridge is a free data retrieval call binding the contract method 0x91c49bf8.
//
// Solidity: function l2TokenBridge() view returns(address)
func (_L1StandardBridge *L1StandardBridgeSession) L2TokenBridge() (common.Address, error) {
	return _L1StandardBridge.Contract.L2TokenBridge(&_L1StandardBridge.CallOpts)
}

// L2TokenBridge is a free data retrieval call binding the contract method 0x91c49bf8.
//
// Solidity: function l2TokenBridge() view returns(address)
func (_L1StandardBridge *L1StandardBridgeCallerSession) L2TokenBridge() (common.Address, error) {
	return _L1StandardBridge.Contract.L2TokenBridge(&_L1StandardBridge.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1StandardBridge *L1StandardBridgeCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1StandardBridge.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1StandardBridge *L1StandardBridgeSession) Messenger() (common.Address, error) {
	return _L1StandardBridge.Contract.Messenger(&_L1StandardBridge.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1StandardBridge *L1StandardBridgeCallerSession) Messenger() (common.Address, error) {
	return _L1StandardBridge.Contract.Messenger(&_L1StandardBridge.CallOpts)
}

// DepositERC20 is a paid mutator transaction binding the contract method 0x58a997f6.
//
// Solidity: function depositERC20(address _l1Token, address _l2Token, uint256 _amount, uint32 _l2Gas, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeTransactor) DepositERC20(opts *bind.TransactOpts, _l1Token common.Address, _l2Token common.Address, _amount *big.Int, _l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.contract.Transact(opts, "depositERC20", _l1Token, _l2Token, _amount, _l2Gas, _data)
}

// DepositERC20 is a paid mutator transaction binding the contract method 0x58a997f6.
//
// Solidity: function depositERC20(address _l1Token, address _l2Token, uint256 _amount, uint32 _l2Gas, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeSession) DepositERC20(_l1Token common.Address, _l2Token common.Address, _amount *big.Int, _l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.DepositERC20(&_L1StandardBridge.TransactOpts, _l1Token, _l2Token, _amount, _l2Gas, _data)
}

// DepositERC20 is a paid mutator transaction binding the contract method 0x58a997f6.
//
// Solidity: function depositERC20(address _l1Token, address _l2Token, uint256 _amount, uint32 _l2Gas, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeTransactorSession) DepositERC20(_l1Token common.Address, _l2Token common.Address, _amount *big.Int, _l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.DepositERC20(&_L1StandardBridge.TransactOpts, _l1Token, _l2Token, _amount, _l2Gas, _data)
}

// DepositERC20To is a paid mutator transaction binding the contract method 0x838b2520.
//
// Solidity: function depositERC20To(address _l1Token, address _l2Token, address _to, uint256 _amount, uint32 _l2Gas, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeTransactor) DepositERC20To(opts *bind.TransactOpts, _l1Token common.Address, _l2Token common.Address, _to common.Address, _amount *big.Int, _l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.contract.Transact(opts, "depositERC20To", _l1Token, _l2Token, _to, _amount, _l2Gas, _data)
}

// DepositERC20To is a paid mutator transaction binding the contract method 0x838b2520.
//
// Solidity: function depositERC20To(address _l1Token, address _l2Token, address _to, uint256 _amount, uint32 _l2Gas, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeSession) DepositERC20To(_l1Token common.Address, _l2Token common.Address, _to common.Address, _amount *big.Int, _l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.DepositERC20To(&_L1StandardBridge.TransactOpts, _l1Token, _l2Token, _to, _amount, _l2Gas, _data)
}

// DepositERC20To is a paid mutator transaction binding the contract method 0x838b2520.
//
// Solidity: function depositERC20To(address _l1Token, address _l2Token, address _to, uint256 _amount, uint32 _l2Gas, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeTransactorSession) DepositERC20To(_l1Token common.Address, _l2Token common.Address, _to common.Address, _amount *big.Int, _l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.DepositERC20To(&_L1StandardBridge.TransactOpts, _l1Token, _l2Token, _to, _amount, _l2Gas, _data)
}

// DepositETH is a paid mutator transaction binding the contract method 0xb1a1a882.
//
// Solidity: function depositETH(uint32 _l2Gas, bytes _data) payable returns()
func (_L1StandardBridge *L1StandardBridgeTransactor) DepositETH(opts *bind.TransactOpts, _l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.contract.Transact(opts, "depositETH", _l2Gas, _data)
}

// DepositETH is a paid mutator transaction binding the contract method 0xb1a1a882.
//
// Solidity: function depositETH(uint32 _l2Gas, bytes _data) payable returns()
func (_L1StandardBridge *L1StandardBridgeSession) DepositETH(_l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.DepositETH(&_L1StandardBridge.TransactOpts, _l2Gas, _data)
}

// DepositETH is a paid mutator transaction binding the contract method 0xb1a1a882.
//
// Solidity: function depositETH(uint32 _l2Gas, bytes _data) payable returns()
func (_L1StandardBridge *L1StandardBridgeTransactorSession) DepositETH(_l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.DepositETH(&_L1StandardBridge.TransactOpts, _l2Gas, _data)
}

// DepositETHTo is a paid mutator transaction binding the contract method 0x9a2ac6d5.
//
// Solidity: function depositETHTo(address _to, uint32 _l2Gas, bytes _data) payable returns()
func (_L1StandardBridge *L1StandardBridgeTransactor) DepositETHTo(opts *bind.TransactOpts, _to common.Address, _l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.contract.Transact(opts, "depositETHTo", _to, _l2Gas, _data)
}

// DepositETHTo is a paid mutator transaction binding the contract method 0x9a2ac6d5.
//
// Solidity: function depositETHTo(address _to, uint32 _l2Gas, bytes _data) payable returns()
func (_L1StandardBridge *L1StandardBridgeSession) DepositETHTo(_to common.Address, _l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.DepositETHTo(&_L1StandardBridge.TransactOpts, _to, _l2Gas, _data)
}

// DepositETHTo is a paid mutator transaction binding the contract method 0x9a2ac6d5.
//
// Solidity: function depositETHTo(address _to, uint32 _l2Gas, bytes _data) payable returns()
func (_L1StandardBridge *L1StandardBridgeTransactorSession) DepositETHTo(_to common.Address, _l2Gas uint32, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.DepositETHTo(&_L1StandardBridge.TransactOpts, _to, _l2Gas, _data)
}

// DonateETH is a paid mutator transaction binding the contract method 0x8b4c40b0.
//
// Solidity: function donateETH() payable returns()
func (_L1StandardBridge *L1StandardBridgeTransactor) DonateETH(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1StandardBridge.contract.Transact(opts, "donateETH")
}

// DonateETH is a paid mutator transaction binding the contract method 0x8b4c40b0.
//
// Solidity: function donateETH() payable returns()
func (_L1StandardBridge *L1StandardBridgeSession) DonateETH() (*types.Transaction, error) {
	return _L1StandardBridge.Contract.DonateETH(&_L1StandardBridge.TransactOpts)
}

// DonateETH is a paid mutator transaction binding the contract method 0x8b4c40b0.
//
// Solidity: function donateETH() payable returns()
func (_L1StandardBridge *L1StandardBridgeTransactorSession) DonateETH() (*types.Transaction, error) {
	return _L1StandardBridge.Contract.DonateETH(&_L1StandardBridge.TransactOpts)
}

// FinalizeERC20Withdrawal is a paid mutator transaction binding the contract method 0xa9f9e675.
//
// Solidity: function finalizeERC20Withdrawal(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeTransactor) FinalizeERC20Withdrawal(opts *bind.TransactOpts, _l1Token common.Address, _l2Token common.Address, _from common.Address, _to common.Address, _amount *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.contract.Transact(opts, "finalizeERC20Withdrawal", _l1Token, _l2Token, _from, _to, _amount, _data)
}

// FinalizeERC20Withdrawal is a paid mutator transaction binding the contract method 0xa9f9e675.
//
// Solidity: function finalizeERC20Withdrawal(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeSession) FinalizeERC20Withdrawal(_l1Token common.Address, _l2Token common.Address, _from common.Address, _to common.Address, _amount *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.FinalizeERC20Withdrawal(&_L1StandardBridge.TransactOpts, _l1Token, _l2Token, _from, _to, _amount, _data)
}

// FinalizeERC20Withdrawal is a paid mutator transaction binding the contract method 0xa9f9e675.
//
// Solidity: function finalizeERC20Withdrawal(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeTransactorSession) FinalizeERC20Withdrawal(_l1Token common.Address, _l2Token common.Address, _from common.Address, _to common.Address, _amount *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.FinalizeERC20Withdrawal(&_L1StandardBridge.TransactOpts, _l1Token, _l2Token, _from, _to, _amount, _data)
}

// FinalizeETHWithdrawal is a paid mutator transaction binding the contract method 0x1532ec34.
//
// Solidity: function finalizeETHWithdrawal(address _from, address _to, uint256 _amount, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeTransactor) FinalizeETHWithdrawal(opts *bind.TransactOpts, _from common.Address, _to common.Address, _amount *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.contract.Transact(opts, "finalizeETHWithdrawal", _from, _to, _amount, _data)
}

// FinalizeETHWithdrawal is a paid mutator transaction binding the contract method 0x1532ec34.
//
// Solidity: function finalizeETHWithdrawal(address _from, address _to, uint256 _amount, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeSession) FinalizeETHWithdrawal(_from common.Address, _to common.Address, _amount *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.FinalizeETHWithdrawal(&_L1StandardBridge.TransactOpts, _from, _to, _amount, _data)
}

// FinalizeETHWithdrawal is a paid mutator transaction binding the contract method 0x1532ec34.
//
// Solidity: function finalizeETHWithdrawal(address _from, address _to, uint256 _amount, bytes _data) returns()
func (_L1StandardBridge *L1StandardBridgeTransactorSession) FinalizeETHWithdrawal(_from common.Address, _to common.Address, _amount *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.FinalizeETHWithdrawal(&_L1StandardBridge.TransactOpts, _from, _to, _amount, _data)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _l1messenger, address _l2TokenBridge) returns()
func (_L1StandardBridge *L1StandardBridgeTransactor) Initialize(opts *bind.TransactOpts, _l1messenger common.Address, _l2TokenBridge common.Address) (*types.Transaction, error) {
	return _L1StandardBridge.contract.Transact(opts, "initialize", _l1messenger, _l2TokenBridge)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _l1messenger, address _l2TokenBridge) returns()
func (_L1StandardBridge *L1StandardBridgeSession) Initialize(_l1messenger common.Address, _l2TokenBridge common.Address) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.Initialize(&_L1StandardBridge.TransactOpts, _l1messenger, _l2TokenBridge)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _l1messenger, address _l2TokenBridge) returns()
func (_L1StandardBridge *L1StandardBridgeTransactorSession) Initialize(_l1messenger common.Address, _l2TokenBridge common.Address) (*types.Transaction, error) {
	return _L1StandardBridge.Contract.Initialize(&_L1StandardBridge.TransactOpts, _l1messenger, _l2TokenBridge)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1StandardBridge *L1StandardBridgeTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1StandardBridge.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1StandardBridge *L1StandardBridgeSession) Receive() (*types.Transaction, error) {
	return _L1StandardBridge.Contract.Receive(&_L1StandardBridge.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1StandardBridge *L1StandardBridgeTransactorSession) Receive() (*types.Transaction, error) {
	return _L1StandardBridge.Contract.Receive(&_L1StandardBridge.TransactOpts)
}

// L1StandardBridgeERC20DepositInitiatedIterator is returned from FilterERC20DepositInitiated and is used to iterate over the raw logs and unpacked data for ERC20DepositInitiated events raised by the L1StandardBridge contract.
type L1StandardBridgeERC20DepositInitiatedIterator struct {
	Event *L1StandardBridgeERC20DepositInitiated // Event containing the contract specifics and raw log

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
func (it *L1StandardBridgeERC20DepositInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1StandardBridgeERC20DepositInitiated)
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
		it.Event = new(L1StandardBridgeERC20DepositInitiated)
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
func (it *L1StandardBridgeERC20DepositInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1StandardBridgeERC20DepositInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1StandardBridgeERC20DepositInitiated represents a ERC20DepositInitiated event raised by the L1StandardBridge contract.
type L1StandardBridgeERC20DepositInitiated struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterERC20DepositInitiated is a free log retrieval operation binding the contract event 0x718594027abd4eaed59f95162563e0cc6d0e8d5b86b1c7be8b1b0ac3343d0396.
//
// Solidity: event ERC20DepositInitiated(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) FilterERC20DepositInitiated(opts *bind.FilterOpts, _l1Token []common.Address, _l2Token []common.Address, _from []common.Address) (*L1StandardBridgeERC20DepositInitiatedIterator, error) {

	var _l1TokenRule []interface{}
	for _, _l1TokenItem := range _l1Token {
		_l1TokenRule = append(_l1TokenRule, _l1TokenItem)
	}
	var _l2TokenRule []interface{}
	for _, _l2TokenItem := range _l2Token {
		_l2TokenRule = append(_l2TokenRule, _l2TokenItem)
	}
	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}

	logs, sub, err := _L1StandardBridge.contract.FilterLogs(opts, "ERC20DepositInitiated", _l1TokenRule, _l2TokenRule, _fromRule)
	if err != nil {
		return nil, err
	}
	return &L1StandardBridgeERC20DepositInitiatedIterator{contract: _L1StandardBridge.contract, event: "ERC20DepositInitiated", logs: logs, sub: sub}, nil
}

// WatchERC20DepositInitiated is a free log subscription operation binding the contract event 0x718594027abd4eaed59f95162563e0cc6d0e8d5b86b1c7be8b1b0ac3343d0396.
//
// Solidity: event ERC20DepositInitiated(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) WatchERC20DepositInitiated(opts *bind.WatchOpts, sink chan<- *L1StandardBridgeERC20DepositInitiated, _l1Token []common.Address, _l2Token []common.Address, _from []common.Address) (event.Subscription, error) {

	var _l1TokenRule []interface{}
	for _, _l1TokenItem := range _l1Token {
		_l1TokenRule = append(_l1TokenRule, _l1TokenItem)
	}
	var _l2TokenRule []interface{}
	for _, _l2TokenItem := range _l2Token {
		_l2TokenRule = append(_l2TokenRule, _l2TokenItem)
	}
	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}

	logs, sub, err := _L1StandardBridge.contract.WatchLogs(opts, "ERC20DepositInitiated", _l1TokenRule, _l2TokenRule, _fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1StandardBridgeERC20DepositInitiated)
				if err := _L1StandardBridge.contract.UnpackLog(event, "ERC20DepositInitiated", log); err != nil {
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

// ParseERC20DepositInitiated is a log parse operation binding the contract event 0x718594027abd4eaed59f95162563e0cc6d0e8d5b86b1c7be8b1b0ac3343d0396.
//
// Solidity: event ERC20DepositInitiated(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) ParseERC20DepositInitiated(log types.Log) (*L1StandardBridgeERC20DepositInitiated, error) {
	event := new(L1StandardBridgeERC20DepositInitiated)
	if err := _L1StandardBridge.contract.UnpackLog(event, "ERC20DepositInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1StandardBridgeERC20WithdrawalFinalizedIterator is returned from FilterERC20WithdrawalFinalized and is used to iterate over the raw logs and unpacked data for ERC20WithdrawalFinalized events raised by the L1StandardBridge contract.
type L1StandardBridgeERC20WithdrawalFinalizedIterator struct {
	Event *L1StandardBridgeERC20WithdrawalFinalized // Event containing the contract specifics and raw log

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
func (it *L1StandardBridgeERC20WithdrawalFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1StandardBridgeERC20WithdrawalFinalized)
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
		it.Event = new(L1StandardBridgeERC20WithdrawalFinalized)
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
func (it *L1StandardBridgeERC20WithdrawalFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1StandardBridgeERC20WithdrawalFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1StandardBridgeERC20WithdrawalFinalized represents a ERC20WithdrawalFinalized event raised by the L1StandardBridge contract.
type L1StandardBridgeERC20WithdrawalFinalized struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterERC20WithdrawalFinalized is a free log retrieval operation binding the contract event 0x3ceee06c1e37648fcbb6ed52e17b3e1f275a1f8c7b22a84b2b84732431e046b3.
//
// Solidity: event ERC20WithdrawalFinalized(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) FilterERC20WithdrawalFinalized(opts *bind.FilterOpts, _l1Token []common.Address, _l2Token []common.Address, _from []common.Address) (*L1StandardBridgeERC20WithdrawalFinalizedIterator, error) {

	var _l1TokenRule []interface{}
	for _, _l1TokenItem := range _l1Token {
		_l1TokenRule = append(_l1TokenRule, _l1TokenItem)
	}
	var _l2TokenRule []interface{}
	for _, _l2TokenItem := range _l2Token {
		_l2TokenRule = append(_l2TokenRule, _l2TokenItem)
	}
	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}

	logs, sub, err := _L1StandardBridge.contract.FilterLogs(opts, "ERC20WithdrawalFinalized", _l1TokenRule, _l2TokenRule, _fromRule)
	if err != nil {
		return nil, err
	}
	return &L1StandardBridgeERC20WithdrawalFinalizedIterator{contract: _L1StandardBridge.contract, event: "ERC20WithdrawalFinalized", logs: logs, sub: sub}, nil
}

// WatchERC20WithdrawalFinalized is a free log subscription operation binding the contract event 0x3ceee06c1e37648fcbb6ed52e17b3e1f275a1f8c7b22a84b2b84732431e046b3.
//
// Solidity: event ERC20WithdrawalFinalized(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) WatchERC20WithdrawalFinalized(opts *bind.WatchOpts, sink chan<- *L1StandardBridgeERC20WithdrawalFinalized, _l1Token []common.Address, _l2Token []common.Address, _from []common.Address) (event.Subscription, error) {

	var _l1TokenRule []interface{}
	for _, _l1TokenItem := range _l1Token {
		_l1TokenRule = append(_l1TokenRule, _l1TokenItem)
	}
	var _l2TokenRule []interface{}
	for _, _l2TokenItem := range _l2Token {
		_l2TokenRule = append(_l2TokenRule, _l2TokenItem)
	}
	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}

	logs, sub, err := _L1StandardBridge.contract.WatchLogs(opts, "ERC20WithdrawalFinalized", _l1TokenRule, _l2TokenRule, _fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1StandardBridgeERC20WithdrawalFinalized)
				if err := _L1StandardBridge.contract.UnpackLog(event, "ERC20WithdrawalFinalized", log); err != nil {
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

// ParseERC20WithdrawalFinalized is a log parse operation binding the contract event 0x3ceee06c1e37648fcbb6ed52e17b3e1f275a1f8c7b22a84b2b84732431e046b3.
//
// Solidity: event ERC20WithdrawalFinalized(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) ParseERC20WithdrawalFinalized(log types.Log) (*L1StandardBridgeERC20WithdrawalFinalized, error) {
	event := new(L1StandardBridgeERC20WithdrawalFinalized)
	if err := _L1StandardBridge.contract.UnpackLog(event, "ERC20WithdrawalFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1StandardBridgeETHDepositInitiatedIterator is returned from FilterETHDepositInitiated and is used to iterate over the raw logs and unpacked data for ETHDepositInitiated events raised by the L1StandardBridge contract.
type L1StandardBridgeETHDepositInitiatedIterator struct {
	Event *L1StandardBridgeETHDepositInitiated // Event containing the contract specifics and raw log

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
func (it *L1StandardBridgeETHDepositInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1StandardBridgeETHDepositInitiated)
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
		it.Event = new(L1StandardBridgeETHDepositInitiated)
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
func (it *L1StandardBridgeETHDepositInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1StandardBridgeETHDepositInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1StandardBridgeETHDepositInitiated represents a ETHDepositInitiated event raised by the L1StandardBridge contract.
type L1StandardBridgeETHDepositInitiated struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Data   []byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterETHDepositInitiated is a free log retrieval operation binding the contract event 0x35d79ab81f2b2017e19afb5c5571778877782d7a8786f5907f93b0f4702f4f23.
//
// Solidity: event ETHDepositInitiated(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) FilterETHDepositInitiated(opts *bind.FilterOpts, _from []common.Address, _to []common.Address) (*L1StandardBridgeETHDepositInitiatedIterator, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _L1StandardBridge.contract.FilterLogs(opts, "ETHDepositInitiated", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return &L1StandardBridgeETHDepositInitiatedIterator{contract: _L1StandardBridge.contract, event: "ETHDepositInitiated", logs: logs, sub: sub}, nil
}

// WatchETHDepositInitiated is a free log subscription operation binding the contract event 0x35d79ab81f2b2017e19afb5c5571778877782d7a8786f5907f93b0f4702f4f23.
//
// Solidity: event ETHDepositInitiated(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) WatchETHDepositInitiated(opts *bind.WatchOpts, sink chan<- *L1StandardBridgeETHDepositInitiated, _from []common.Address, _to []common.Address) (event.Subscription, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _L1StandardBridge.contract.WatchLogs(opts, "ETHDepositInitiated", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1StandardBridgeETHDepositInitiated)
				if err := _L1StandardBridge.contract.UnpackLog(event, "ETHDepositInitiated", log); err != nil {
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

// ParseETHDepositInitiated is a log parse operation binding the contract event 0x35d79ab81f2b2017e19afb5c5571778877782d7a8786f5907f93b0f4702f4f23.
//
// Solidity: event ETHDepositInitiated(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) ParseETHDepositInitiated(log types.Log) (*L1StandardBridgeETHDepositInitiated, error) {
	event := new(L1StandardBridgeETHDepositInitiated)
	if err := _L1StandardBridge.contract.UnpackLog(event, "ETHDepositInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1StandardBridgeETHWithdrawalFinalizedIterator is returned from FilterETHWithdrawalFinalized and is used to iterate over the raw logs and unpacked data for ETHWithdrawalFinalized events raised by the L1StandardBridge contract.
type L1StandardBridgeETHWithdrawalFinalizedIterator struct {
	Event *L1StandardBridgeETHWithdrawalFinalized // Event containing the contract specifics and raw log

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
func (it *L1StandardBridgeETHWithdrawalFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1StandardBridgeETHWithdrawalFinalized)
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
		it.Event = new(L1StandardBridgeETHWithdrawalFinalized)
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
func (it *L1StandardBridgeETHWithdrawalFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1StandardBridgeETHWithdrawalFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1StandardBridgeETHWithdrawalFinalized represents a ETHWithdrawalFinalized event raised by the L1StandardBridge contract.
type L1StandardBridgeETHWithdrawalFinalized struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Data   []byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterETHWithdrawalFinalized is a free log retrieval operation binding the contract event 0x2ac69ee804d9a7a0984249f508dfab7cb2534b465b6ce1580f99a38ba9c5e631.
//
// Solidity: event ETHWithdrawalFinalized(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) FilterETHWithdrawalFinalized(opts *bind.FilterOpts, _from []common.Address, _to []common.Address) (*L1StandardBridgeETHWithdrawalFinalizedIterator, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _L1StandardBridge.contract.FilterLogs(opts, "ETHWithdrawalFinalized", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return &L1StandardBridgeETHWithdrawalFinalizedIterator{contract: _L1StandardBridge.contract, event: "ETHWithdrawalFinalized", logs: logs, sub: sub}, nil
}

// WatchETHWithdrawalFinalized is a free log subscription operation binding the contract event 0x2ac69ee804d9a7a0984249f508dfab7cb2534b465b6ce1580f99a38ba9c5e631.
//
// Solidity: event ETHWithdrawalFinalized(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) WatchETHWithdrawalFinalized(opts *bind.WatchOpts, sink chan<- *L1StandardBridgeETHWithdrawalFinalized, _from []common.Address, _to []common.Address) (event.Subscription, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _L1StandardBridge.contract.WatchLogs(opts, "ETHWithdrawalFinalized", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1StandardBridgeETHWithdrawalFinalized)
				if err := _L1StandardBridge.contract.UnpackLog(event, "ETHWithdrawalFinalized", log); err != nil {
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

// ParseETHWithdrawalFinalized is a log parse operation binding the contract event 0x2ac69ee804d9a7a0984249f508dfab7cb2534b465b6ce1580f99a38ba9c5e631.
//
// Solidity: event ETHWithdrawalFinalized(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
func (_L1StandardBridge *L1StandardBridgeFilterer) ParseETHWithdrawalFinalized(log types.Log) (*L1StandardBridgeETHWithdrawalFinalized, error) {
	event := new(L1StandardBridgeETHWithdrawalFinalized)
	if err := _L1StandardBridge.contract.UnpackLog(event, "ETHWithdrawalFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
