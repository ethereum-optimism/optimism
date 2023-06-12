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

// BobaGasPriceOracleABI is the input ABI used to generate the binding from.
const BobaGasPriceOracleABI = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"SwapBOBAForETHMetaTransaction\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"TransferOwnership\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"UpdateGasPriceOracleAddress\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"UpdateMaxPriceRatio\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"UpdateMetaTransactionFee\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"UpdateMinPriceRatio\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"UpdatePriceRatio\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"UpdateReceivedETHAmount\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"UseBobaAsFeeToken\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"UseETHAsFeeToken\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"WithdrawBOBA\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"WithdrawETH\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MIN_WITHDRAWAL_AMOUNT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"bobaFeeTokenUsers\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeWallet\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gasPriceOracleAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBOBAForSwap\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_txData\",\"type\":\"bytes\"}],\"name\":\"getL1BobaFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_feeWallet\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_l2BobaAddress\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2BobaAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"marketPriceRatio\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxPriceRatio\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"metaTransactionFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minPriceRatio\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"priceRatio\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"receivedETHAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"swapBOBAForETHMetaTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_gasPriceOracleAddress\",\"type\":\"address\"}],\"name\":\"updateGasPriceOracleAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxPriceRatio\",\"type\":\"uint256\"}],\"name\":\"updateMaxPriceRatio\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_metaTransactionFee\",\"type\":\"uint256\"}],\"name\":\"updateMetaTransactionFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minPriceRatio\",\"type\":\"uint256\"}],\"name\":\"updateMinPriceRatio\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_priceRatio\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_marketPriceRatio\",\"type\":\"uint256\"}],\"name\":\"updatePriceRatio\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_receivedETHAmount\",\"type\":\"uint256\"}],\"name\":\"updateReceivedETHAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"useBobaAsFeeToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"useETHAsFeeToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawBOBA\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawETH\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]"

// BobaGasPriceOracleBin is the compiled bytecode used for deploying new contracts.
var BobaGasPriceOracleBin = "0x60806040526113886003556101f4600455600680546001600160a01b03191673420000000000000000000000000000000000000f1790556729a2241af62c00006008556611c37937e0800060095534801561005957600080fd5b506122b2806100696000396000f3fe6080604052600436106101af5760003560e01c806389df963d116100ec578063d2e1fb221161008a578063e086e5ec11610064578063e086e5ec146104af578063e3aea9ba146104c4578063f25f4b56146104e4578063f2fde38b1461051157600080fd5b8063d2e1fb2214610466578063d3e5792b1461047c578063d86732ef1461049957600080fd5b8063b54016dc116100c6578063b54016dc146103f0578063bc9bd6ee14610410578063c8a0541314610430578063cd0514ad1461045057600080fd5b806389df963d146103905780638da5cb5b146103a55780638fcfc813146103d057600080fd5b806334fe1b16116101595780635b9da5c6116101335780635b9da5c6146102ed5780636805491b1461030d5780637728195c1461034d578063872ea4991461037a57600080fd5b806334fe1b16146102a3578063438ac96c146102b8578063485cc955146102cd57600080fd5b80631b6771991161018a5780631b6771991461021c57806323ec63201461023157806324b20eda1461025157600080fd5b80625c5fb2146101bb5780630aa2f420146101dd57806315a0c1ac1461020657600080fd5b366101b657005b600080fd5b3480156101c757600080fd5b506101db6101d6366004611ecb565b610531565b005b3480156101e957600080fd5b506101f360055481565b6040519081526020015b60405180910390f35b34801561021257600080fd5b506101f3600a5481565b34801561022857600080fd5b506101db610649565b34801561023d57600080fd5b506101f361024c366004611f13565b610790565b34801561025d57600080fd5b5060025461027e9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101fd565b3480156102af57600080fd5b506101db61083f565b3480156102c457600080fd5b506101f3610a13565b3480156102d957600080fd5b506101db6102e8366004612007565b610a3d565b3480156102f957600080fd5b506101db610308366004611ecb565b610ba4565b34801561031957600080fd5b5061033d610328366004612040565b60076020526000908152604090205460ff1681565b60405190151581526020016101fd565b34801561035957600080fd5b5060065461027e9073ffffffffffffffffffffffffffffffffffffffff1681565b34801561038657600080fd5b506101f360085481565b34801561039c57600080fd5b506101db610c8f565b3480156103b157600080fd5b5060005473ffffffffffffffffffffffffffffffffffffffff1661027e565b3480156103dc57600080fd5b506101db6103eb366004612040565b610f68565b3480156103fc57600080fd5b506101db61040b36600461205d565b611135565b34801561041c57600080fd5b506101db61042b3660046120d4565b6114b6565b34801561043c57600080fd5b506101db61044b366004611ecb565b6115f5565b34801561045c57600080fd5b506101f360095481565b34801561047257600080fd5b506101f360045481565b34801561048857600080fd5b506101f3680821ab0d441498000081565b3480156104a557600080fd5b506101f360035481565b3480156104bb57600080fd5b506101db6116d6565b3480156104d057600080fd5b506101db6104df366004611ecb565b611891565b3480156104f057600080fd5b5060015461027e9073ffffffffffffffffffffffffffffffffffffffff1681565b34801561051d57600080fd5b506101db61052c366004612040565b611964565b60005473ffffffffffffffffffffffffffffffffffffffff1633146105b7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064015b60405180910390fd5b60035481111580156105c95750600081115b6105d257600080fd5b60048190557f680f379280fc8680df45c979a924c0084a250758604482cb01dadedbaa1c09c961061760005473ffffffffffffffffffffffffffffffffffffffff1690565b6040805173ffffffffffffffffffffffffffffffffffffffff909216825260208201849052015b60405180910390a150565b333b156106b2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600f60248201527f4163636f756e74206e6f7420454f41000000000000000000000000000000000060448201526064016105ae565b66071afd498d000033311015610724576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601860248201527f496e73756666696369656e74204554482062616c616e6365000000000000000060448201526064016105ae565b3360008181526007602090815260409182902080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016905590519182527f764389830e6a6b84f4ea3f2551a4c5afbb6dff806f2d8f571f6913c6c4b62a4091015b60405180910390a1565b6006546005546040517f49948e0e00000000000000000000000000000000000000000000000000000000815260009273ffffffffffffffffffffffffffffffffffffffff16919082906349948e0e906107ed90879060040161216c565b602060405180830381865afa15801561080a573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061082e919061217f565b61083891906121c7565b9392505050565b333b156108a8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600f60248201527f4163636f756e74206e6f7420454f41000000000000000000000000000000000060448201526064016105ae565b6002546040517f70a082310000000000000000000000000000000000000000000000000000000081523360048201526729a2241af62c00009173ffffffffffffffffffffffffffffffffffffffff16906370a0823190602401602060405180830381865afa15801561091e573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610942919061217f565b10156109aa576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601960248201527f496e73756666696369656e7420426f62612062616c616e63650000000000000060448201526064016105ae565b3360008181526007602090815260409182902080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600117905590519182527fd1787ba09c5383b33cf88983fbbf2e6ae348746a3a906e1a1bb67c729661a4ac9101610786565b6000610a38600854610a32600a54600954611b0790919063ffffffff16565b90611b13565b905090565b60015473ffffffffffffffffffffffffffffffffffffffff1615610abd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f436f6e747261637420686173206265656e20696e697469616c697a656400000060448201526064016105ae565b73ffffffffffffffffffffffffffffffffffffffff821615801590610af7575073ffffffffffffffffffffffffffffffffffffffff811615155b610b0057600080fd5b6001805473ffffffffffffffffffffffffffffffffffffffff9384167fffffffffffffffffffffffff000000000000000000000000000000000000000091821617909155600280549290931691811691909117909155600080548216331790556006805490911673420000000000000000000000000000000000000f1790556729a2241af62c00006008556113886003556107d060058190556101f4600455600a55565b60005473ffffffffffffffffffffffffffffffffffffffff163314610c25576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016105ae565b66038d7ea4c6800081118015610c415750662386f26fc1000081105b610c4a57600080fd5b60098190557fdcb9e069a0d16a974c9c0f4a88e2c9b79df5c45d9721c26461043d51c446820761061760005473ffffffffffffffffffffffffffffffffffffffff1690565b6002546040517f70a08231000000000000000000000000000000000000000000000000000000008152306004820152680821ab0d44149800009173ffffffffffffffffffffffffffffffffffffffff16906370a0823190602401602060405180830381865afa158015610d06573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610d2a919061217f565b1015610dde576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605560248201527f426f62615f47617350726963654f7261636c653a207769746864726177616c2060448201527f616d6f756e74206d7573742062652067726561746572207468616e206d696e6960648201527f6d756d207769746864726177616c20616d6f756e740000000000000000000000608482015260a4016105ae565b6002546001546040517f70a082310000000000000000000000000000000000000000000000000000000081523060048201527342000000000000000000000000000000000000109263a3a795489273ffffffffffffffffffffffffffffffffffffffff9182169291169082906370a0823190602401602060405180830381865afa158015610e70573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610e94919061217f565b6000604051806020016040528060008152506040518663ffffffff1660e01b8152600401610ec6959493929190612204565b600060405180830381600087803b158015610ee057600080fd5b505af1158015610ef4573d6000803e3d6000fd5b505050507f2c69c3957d9ca9782726f647b7a3592dd381f4370288551f5ed43fd3cc5b7753610f3860005473ffffffffffffffffffffffffffffffffffffffff1690565b6001546040805173ffffffffffffffffffffffffffffffffffffffff938416815292909116602083015201610786565b60005473ffffffffffffffffffffffffffffffffffffffff163314610fe9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016105ae565b73ffffffffffffffffffffffffffffffffffffffff81163b611067576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f4163636f756e7420697320454f4100000000000000000000000000000000000060448201526064016105ae565b73ffffffffffffffffffffffffffffffffffffffff811661108757600080fd5b6006805473ffffffffffffffffffffffffffffffffffffffff83167fffffffffffffffffffffffff00000000000000000000000000000000000000009091161790557f226bf99888a1e70d41ce744b11ce2acd4d1d1b8cf4ad17a0e72e67acff4bf5a761110960005473ffffffffffffffffffffffffffffffffffffffff1690565b6040805173ffffffffffffffffffffffffffffffffffffffff928316815291841660208301520161063e565b73ffffffffffffffffffffffffffffffffffffffff87163b156111b4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600f60248201527f4163636f756e74206e6f7420454f41000000000000000000000000000000000060448201526064016105ae565b73ffffffffffffffffffffffffffffffffffffffff86163014611233576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601c60248201527f5370656e646572206973206e6f74207468697320636f6e74726163740000000060448201526064016105ae565b6000611252600854610a32600a54600954611b0790919063ffffffff16565b9050808610156112be576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601360248201527f56616c7565206973206e6f7420656e6f7567680000000000000000000000000060448201526064016105ae565b6002546040517fd505accf00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8a811660048301528981166024830152604482018990526064820188905260ff8716608483015260a4820186905260c4820185905290911690819063d505accf9060e401600060405180830381600087803b15801561135a57600080fd5b505af115801561136e573d6000803e3d6000fd5b5050600254611398925073ffffffffffffffffffffffffffffffffffffffff1690508a3085611b1f565b60095460405160009173ffffffffffffffffffffffffffffffffffffffff8c16918381818185875af1925050503d80600081146113f1576040519150601f19603f3d011682016040523d82523d6000602084013e6113f6565b606091505b5050905080611461576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601260248201527f4661696c656420746f2073656e6420455448000000000000000000000000000060448201526064016105ae565b60405173ffffffffffffffffffffffffffffffffffffffff8b1681527fb92b4b358dfa6e521f7f80a5d0522cf04a2082482701a0d78ff2bb615df646be9060200160405180910390a150505050505050505050565b60005473ffffffffffffffffffffffffffffffffffffffff163314611537576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016105ae565b600354821115801561154b57506004548210155b61155457600080fd5b600354811115801561156857506004548110155b61157157600080fd5b6005829055600a8190557f23632bbb735dece542dac9735a2ba4253234eb119ce45cdf9968cbbe12aa67906115bb60005473ffffffffffffffffffffffffffffffffffffffff1690565b6040805173ffffffffffffffffffffffffffffffffffffffff90921682526020820185905281018390526060015b60405180910390a15050565b60005473ffffffffffffffffffffffffffffffffffffffff163314611676576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016105ae565b60045481101580156116885750600081115b61169157600080fd5b60038190557f7a28f69b71e51c4a30f620a2cfe4ce5aad2cd3fe5cc9647e400e252b65033d4161061760005473ffffffffffffffffffffffffffffffffffffffff1690565b60005473ffffffffffffffffffffffffffffffffffffffff163314611757576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016105ae565b60015460405160009173ffffffffffffffffffffffffffffffffffffffff169047908381818185875af1925050503d80600081146117b1576040519150601f19603f3d011682016040523d82523d6000602084013e6117b6565b606091505b5050905080611821576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4661696c656420746f2073656e642045544820746f206665652077616c6c657460448201526064016105ae565b7f6de63bb986f2779478e384365c03cc2e62f06b453856acca87d5a519ce02664961186160005473ffffffffffffffffffffffffffffffffffffffff1690565b6001546040805173ffffffffffffffffffffffffffffffffffffffff93841681529290911660208301520161063e565b60005473ffffffffffffffffffffffffffffffffffffffff163314611912576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016105ae565b6000811161191f57600080fd5b60088190557f1071f61d642716391065a6f38aac12cdc6a436ca6a6622a18ae0530495738afc61061760005473ffffffffffffffffffffffffffffffffffffffff1690565b60005473ffffffffffffffffffffffffffffffffffffffff1633146119e5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f63616c6c6572206973206e6f7420746865206f776e657200000000000000000060448201526064016105ae565b73ffffffffffffffffffffffffffffffffffffffff8116611a88576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084016105ae565b6000805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff000000000000000000000000000000000000000083168117909355604080519190921680825260208201939093527f5c486528ec3e3f0ea91181cff8116f02bfa350e03b8b6f12e00765adbb5af85c91016115e9565b600061083882846121c7565b6000610838828461224f565b6040805173ffffffffffffffffffffffffffffffffffffffff85811660248301528416604482015260648082018490528251808303909101815260849091019091526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f23b872dd00000000000000000000000000000000000000000000000000000000179052611bb4908590611bba565b50505050565b6000611c1c826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c65648152508573ffffffffffffffffffffffffffffffffffffffff16611ccb9092919063ffffffff16565b805190915015611cc65780806020019051810190611c3a9190612267565b611cc6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f7420737563636565640000000000000000000000000000000000000000000060648201526084016105ae565b505050565b6060611cda8484600085611ce2565b949350505050565b606082471015611d74576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c000000000000000000000000000000000000000000000000000060648201526084016105ae565b73ffffffffffffffffffffffffffffffffffffffff85163b611df2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e747261637400000060448201526064016105ae565b6000808673ffffffffffffffffffffffffffffffffffffffff168587604051611e1b9190612289565b60006040518083038185875af1925050503d8060008114611e58576040519150601f19603f3d011682016040523d82523d6000602084013e611e5d565b606091505b5091509150611e6d828286611e78565b979650505050505050565b60608315611e87575081610838565b825115611e975782518084602001fd5b816040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016105ae919061216c565b600060208284031215611edd57600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600060208284031215611f2557600080fd5b813567ffffffffffffffff80821115611f3d57600080fd5b818401915084601f830112611f5157600080fd5b813581811115611f6357611f63611ee4565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908382118183101715611fa957611fa9611ee4565b81604052828152876020848701011115611fc257600080fd5b826020860160208301376000928101602001929092525095945050505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461200457600080fd5b50565b6000806040838503121561201a57600080fd5b823561202581611fe2565b9150602083013561203581611fe2565b809150509250929050565b60006020828403121561205257600080fd5b813561083881611fe2565b600080600080600080600060e0888a03121561207857600080fd5b873561208381611fe2565b9650602088013561209381611fe2565b95506040880135945060608801359350608088013560ff811681146120b757600080fd5b9699959850939692959460a0840135945060c09093013592915050565b600080604083850312156120e757600080fd5b50508035926020909101359150565b60005b838110156121115781810151838201526020016120f9565b83811115611bb45750506000910152565b6000815180845261213a8160208601602086016120f6565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006108386020830184612122565b60006020828403121561219157600080fd5b5051919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff04831182151516156121ff576121ff612198565b500290565b600073ffffffffffffffffffffffffffffffffffffffff808816835280871660208401525084604083015263ffffffff8416606083015260a06080830152611e6d60a0830184612122565b6000821982111561226257612262612198565b500190565b60006020828403121561227957600080fd5b8151801515811461083857600080fd5b6000825161229b8184602087016120f6565b919091019291505056fea164736f6c634300080f000a"

// DeployBobaGasPriceOracle deploys a new Ethereum contract, binding an instance of BobaGasPriceOracle to it.
func DeployBobaGasPriceOracle(auth *bind.TransactOpts, backend bind.ContractBackend) (libcommon.Address, types.Transaction, *BobaGasPriceOracle, error) {
	parsed, err := abi.JSON(strings.NewReader(BobaGasPriceOracleABI))
	if err != nil {
		return libcommon.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, libcommon.FromHex(BobaGasPriceOracleBin), backend)
	if err != nil {
		return libcommon.Address{}, nil, nil, err
	}
	return address, tx, &BobaGasPriceOracle{BobaGasPriceOracleCaller: BobaGasPriceOracleCaller{contract: contract}, BobaGasPriceOracleTransactor: BobaGasPriceOracleTransactor{contract: contract}, BobaGasPriceOracleFilterer: BobaGasPriceOracleFilterer{contract: contract}}, nil
}

// BobaGasPriceOracle is an auto generated Go binding around an Ethereum contract.
type BobaGasPriceOracle struct {
	BobaGasPriceOracleCaller     // Read-only binding to the contract
	BobaGasPriceOracleTransactor // Write-only binding to the contract
	BobaGasPriceOracleFilterer   // Log filterer for contract events
}

// BobaGasPriceOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type BobaGasPriceOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BobaGasPriceOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BobaGasPriceOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BobaGasPriceOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BobaGasPriceOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BobaGasPriceOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BobaGasPriceOracleSession struct {
	Contract     *BobaGasPriceOracle // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// BobaGasPriceOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BobaGasPriceOracleCallerSession struct {
	Contract *BobaGasPriceOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// BobaGasPriceOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BobaGasPriceOracleTransactorSession struct {
	Contract     *BobaGasPriceOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// BobaGasPriceOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type BobaGasPriceOracleRaw struct {
	Contract *BobaGasPriceOracle // Generic contract binding to access the raw methods on
}

// BobaGasPriceOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BobaGasPriceOracleCallerRaw struct {
	Contract *BobaGasPriceOracleCaller // Generic read-only contract binding to access the raw methods on
}

// BobaGasPriceOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BobaGasPriceOracleTransactorRaw struct {
	Contract *BobaGasPriceOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBobaGasPriceOracle creates a new instance of BobaGasPriceOracle, bound to a specific deployed contract.
func NewBobaGasPriceOracle(address libcommon.Address, backend bind.ContractBackend) (*BobaGasPriceOracle, error) {
	contract, err := bindBobaGasPriceOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracle{BobaGasPriceOracleCaller: BobaGasPriceOracleCaller{contract: contract}, BobaGasPriceOracleTransactor: BobaGasPriceOracleTransactor{contract: contract}, BobaGasPriceOracleFilterer: BobaGasPriceOracleFilterer{contract: contract}}, nil
}

// NewBobaGasPriceOracleCaller creates a new read-only instance of BobaGasPriceOracle, bound to a specific deployed contract.
func NewBobaGasPriceOracleCaller(address libcommon.Address, caller bind.ContractCaller) (*BobaGasPriceOracleCaller, error) {
	contract, err := bindBobaGasPriceOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleCaller{contract: contract}, nil
}

// NewBobaGasPriceOracleTransactor creates a new write-only instance of BobaGasPriceOracle, bound to a specific deployed contract.
func NewBobaGasPriceOracleTransactor(address libcommon.Address, transactor bind.ContractTransactor) (*BobaGasPriceOracleTransactor, error) {
	contract, err := bindBobaGasPriceOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleTransactor{contract: contract}, nil
}

// NewBobaGasPriceOracleFilterer creates a new log filterer instance of BobaGasPriceOracle, bound to a specific deployed contract.
func NewBobaGasPriceOracleFilterer(address libcommon.Address, filterer bind.ContractFilterer) (*BobaGasPriceOracleFilterer, error) {
	contract, err := bindBobaGasPriceOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleFilterer{contract: contract}, nil
}

// bindBobaGasPriceOracle binds a generic wrapper to an already deployed contract.
func bindBobaGasPriceOracle(address libcommon.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BobaGasPriceOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BobaGasPriceOracle *BobaGasPriceOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BobaGasPriceOracle.Contract.BobaGasPriceOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BobaGasPriceOracle *BobaGasPriceOracleRaw) Transfer(opts *bind.TransactOpts) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.BobaGasPriceOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BobaGasPriceOracle *BobaGasPriceOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.BobaGasPriceOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BobaGasPriceOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.contract.Transact(opts, method, params...)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) MINWITHDRAWALAMOUNT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "MIN_WITHDRAWAL_AMOUNT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.MINWITHDRAWALAMOUNT(&_BobaGasPriceOracle.CallOpts)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.MINWITHDRAWALAMOUNT(&_BobaGasPriceOracle.CallOpts)
}

// BobaFeeTokenUsers is a free data retrieval call binding the contract method 0x6805491b.
//
// Solidity: function bobaFeeTokenUsers(address ) view returns(bool)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) BobaFeeTokenUsers(opts *bind.CallOpts, arg0 libcommon.Address) (bool, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "bobaFeeTokenUsers", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// BobaFeeTokenUsers is a free data retrieval call binding the contract method 0x6805491b.
//
// Solidity: function bobaFeeTokenUsers(address ) view returns(bool)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) BobaFeeTokenUsers(arg0 libcommon.Address) (bool, error) {
	return _BobaGasPriceOracle.Contract.BobaFeeTokenUsers(&_BobaGasPriceOracle.CallOpts, arg0)
}

// BobaFeeTokenUsers is a free data retrieval call binding the contract method 0x6805491b.
//
// Solidity: function bobaFeeTokenUsers(address ) view returns(bool)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) BobaFeeTokenUsers(arg0 libcommon.Address) (bool, error) {
	return _BobaGasPriceOracle.Contract.BobaFeeTokenUsers(&_BobaGasPriceOracle.CallOpts, arg0)
}

// FeeWallet is a free data retrieval call binding the contract method 0xf25f4b56.
//
// Solidity: function feeWallet() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) FeeWallet(opts *bind.CallOpts) (libcommon.Address, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "feeWallet")

	if err != nil {
		return *new(libcommon.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(libcommon.Address)).(*libcommon.Address)

	return out0, err

}

// FeeWallet is a free data retrieval call binding the contract method 0xf25f4b56.
//
// Solidity: function feeWallet() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) FeeWallet() (libcommon.Address, error) {
	return _BobaGasPriceOracle.Contract.FeeWallet(&_BobaGasPriceOracle.CallOpts)
}

// FeeWallet is a free data retrieval call binding the contract method 0xf25f4b56.
//
// Solidity: function feeWallet() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) FeeWallet() (libcommon.Address, error) {
	return _BobaGasPriceOracle.Contract.FeeWallet(&_BobaGasPriceOracle.CallOpts)
}

// GasPriceOracleAddress is a free data retrieval call binding the contract method 0x7728195c.
//
// Solidity: function gasPriceOracleAddress() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) GasPriceOracleAddress(opts *bind.CallOpts) (libcommon.Address, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "gasPriceOracleAddress")

	if err != nil {
		return *new(libcommon.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(libcommon.Address)).(*libcommon.Address)

	return out0, err

}

// GasPriceOracleAddress is a free data retrieval call binding the contract method 0x7728195c.
//
// Solidity: function gasPriceOracleAddress() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) GasPriceOracleAddress() (libcommon.Address, error) {
	return _BobaGasPriceOracle.Contract.GasPriceOracleAddress(&_BobaGasPriceOracle.CallOpts)
}

// GasPriceOracleAddress is a free data retrieval call binding the contract method 0x7728195c.
//
// Solidity: function gasPriceOracleAddress() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) GasPriceOracleAddress() (libcommon.Address, error) {
	return _BobaGasPriceOracle.Contract.GasPriceOracleAddress(&_BobaGasPriceOracle.CallOpts)
}

// GetBOBAForSwap is a free data retrieval call binding the contract method 0x438ac96c.
//
// Solidity: function getBOBAForSwap() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) GetBOBAForSwap(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "getBOBAForSwap")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBOBAForSwap is a free data retrieval call binding the contract method 0x438ac96c.
//
// Solidity: function getBOBAForSwap() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) GetBOBAForSwap() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.GetBOBAForSwap(&_BobaGasPriceOracle.CallOpts)
}

// GetBOBAForSwap is a free data retrieval call binding the contract method 0x438ac96c.
//
// Solidity: function getBOBAForSwap() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) GetBOBAForSwap() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.GetBOBAForSwap(&_BobaGasPriceOracle.CallOpts)
}

// GetL1BobaFee is a free data retrieval call binding the contract method 0x23ec6320.
//
// Solidity: function getL1BobaFee(bytes _txData) view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) GetL1BobaFee(opts *bind.CallOpts, _txData []byte) (*big.Int, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "getL1BobaFee", _txData)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1BobaFee is a free data retrieval call binding the contract method 0x23ec6320.
//
// Solidity: function getL1BobaFee(bytes _txData) view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) GetL1BobaFee(_txData []byte) (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.GetL1BobaFee(&_BobaGasPriceOracle.CallOpts, _txData)
}

// GetL1BobaFee is a free data retrieval call binding the contract method 0x23ec6320.
//
// Solidity: function getL1BobaFee(bytes _txData) view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) GetL1BobaFee(_txData []byte) (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.GetL1BobaFee(&_BobaGasPriceOracle.CallOpts, _txData)
}

// L2BobaAddress is a free data retrieval call binding the contract method 0x24b20eda.
//
// Solidity: function l2BobaAddress() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) L2BobaAddress(opts *bind.CallOpts) (libcommon.Address, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "l2BobaAddress")

	if err != nil {
		return *new(libcommon.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(libcommon.Address)).(*libcommon.Address)

	return out0, err

}

// L2BobaAddress is a free data retrieval call binding the contract method 0x24b20eda.
//
// Solidity: function l2BobaAddress() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) L2BobaAddress() (libcommon.Address, error) {
	return _BobaGasPriceOracle.Contract.L2BobaAddress(&_BobaGasPriceOracle.CallOpts)
}

// L2BobaAddress is a free data retrieval call binding the contract method 0x24b20eda.
//
// Solidity: function l2BobaAddress() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) L2BobaAddress() (libcommon.Address, error) {
	return _BobaGasPriceOracle.Contract.L2BobaAddress(&_BobaGasPriceOracle.CallOpts)
}

// MarketPriceRatio is a free data retrieval call binding the contract method 0x15a0c1ac.
//
// Solidity: function marketPriceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) MarketPriceRatio(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "marketPriceRatio")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MarketPriceRatio is a free data retrieval call binding the contract method 0x15a0c1ac.
//
// Solidity: function marketPriceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) MarketPriceRatio() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.MarketPriceRatio(&_BobaGasPriceOracle.CallOpts)
}

// MarketPriceRatio is a free data retrieval call binding the contract method 0x15a0c1ac.
//
// Solidity: function marketPriceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) MarketPriceRatio() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.MarketPriceRatio(&_BobaGasPriceOracle.CallOpts)
}

// MaxPriceRatio is a free data retrieval call binding the contract method 0xd86732ef.
//
// Solidity: function maxPriceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) MaxPriceRatio(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "maxPriceRatio")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxPriceRatio is a free data retrieval call binding the contract method 0xd86732ef.
//
// Solidity: function maxPriceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) MaxPriceRatio() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.MaxPriceRatio(&_BobaGasPriceOracle.CallOpts)
}

// MaxPriceRatio is a free data retrieval call binding the contract method 0xd86732ef.
//
// Solidity: function maxPriceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) MaxPriceRatio() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.MaxPriceRatio(&_BobaGasPriceOracle.CallOpts)
}

// MetaTransactionFee is a free data retrieval call binding the contract method 0x872ea499.
//
// Solidity: function metaTransactionFee() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) MetaTransactionFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "metaTransactionFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MetaTransactionFee is a free data retrieval call binding the contract method 0x872ea499.
//
// Solidity: function metaTransactionFee() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) MetaTransactionFee() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.MetaTransactionFee(&_BobaGasPriceOracle.CallOpts)
}

// MetaTransactionFee is a free data retrieval call binding the contract method 0x872ea499.
//
// Solidity: function metaTransactionFee() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) MetaTransactionFee() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.MetaTransactionFee(&_BobaGasPriceOracle.CallOpts)
}

// MinPriceRatio is a free data retrieval call binding the contract method 0xd2e1fb22.
//
// Solidity: function minPriceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) MinPriceRatio(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "minPriceRatio")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinPriceRatio is a free data retrieval call binding the contract method 0xd2e1fb22.
//
// Solidity: function minPriceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) MinPriceRatio() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.MinPriceRatio(&_BobaGasPriceOracle.CallOpts)
}

// MinPriceRatio is a free data retrieval call binding the contract method 0xd2e1fb22.
//
// Solidity: function minPriceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) MinPriceRatio() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.MinPriceRatio(&_BobaGasPriceOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) Owner(opts *bind.CallOpts) (libcommon.Address, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(libcommon.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(libcommon.Address)).(*libcommon.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) Owner() (libcommon.Address, error) {
	return _BobaGasPriceOracle.Contract.Owner(&_BobaGasPriceOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) Owner() (libcommon.Address, error) {
	return _BobaGasPriceOracle.Contract.Owner(&_BobaGasPriceOracle.CallOpts)
}

// PriceRatio is a free data retrieval call binding the contract method 0x0aa2f420.
//
// Solidity: function priceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) PriceRatio(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "priceRatio")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PriceRatio is a free data retrieval call binding the contract method 0x0aa2f420.
//
// Solidity: function priceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) PriceRatio() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.PriceRatio(&_BobaGasPriceOracle.CallOpts)
}

// PriceRatio is a free data retrieval call binding the contract method 0x0aa2f420.
//
// Solidity: function priceRatio() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) PriceRatio() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.PriceRatio(&_BobaGasPriceOracle.CallOpts)
}

// ReceivedETHAmount is a free data retrieval call binding the contract method 0xcd0514ad.
//
// Solidity: function receivedETHAmount() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCaller) ReceivedETHAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BobaGasPriceOracle.contract.Call(opts, &out, "receivedETHAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ReceivedETHAmount is a free data retrieval call binding the contract method 0xcd0514ad.
//
// Solidity: function receivedETHAmount() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) ReceivedETHAmount() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.ReceivedETHAmount(&_BobaGasPriceOracle.CallOpts)
}

// ReceivedETHAmount is a free data retrieval call binding the contract method 0xcd0514ad.
//
// Solidity: function receivedETHAmount() view returns(uint256)
func (_BobaGasPriceOracle *BobaGasPriceOracleCallerSession) ReceivedETHAmount() (*big.Int, error) {
	return _BobaGasPriceOracle.Contract.ReceivedETHAmount(&_BobaGasPriceOracle.CallOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _feeWallet, address _l2BobaAddress) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) Initialize(opts *bind.TransactOpts, _feeWallet libcommon.Address, _l2BobaAddress libcommon.Address) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "initialize", _feeWallet, _l2BobaAddress)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _feeWallet, address _l2BobaAddress) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) Initialize(_feeWallet libcommon.Address, _l2BobaAddress libcommon.Address) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.Initialize(&_BobaGasPriceOracle.TransactOpts, _feeWallet, _l2BobaAddress)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _feeWallet, address _l2BobaAddress) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) Initialize(_feeWallet libcommon.Address, _l2BobaAddress libcommon.Address) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.Initialize(&_BobaGasPriceOracle.TransactOpts, _feeWallet, _l2BobaAddress)
}

// SwapBOBAForETHMetaTransaction is a paid mutator transaction binding the contract method 0xb54016dc.
//
// Solidity: function swapBOBAForETHMetaTransaction(address tokenOwner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) SwapBOBAForETHMetaTransaction(opts *bind.TransactOpts, tokenOwner libcommon.Address, spender libcommon.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "swapBOBAForETHMetaTransaction", tokenOwner, spender, value, deadline, v, r, s)
}

// SwapBOBAForETHMetaTransaction is a paid mutator transaction binding the contract method 0xb54016dc.
//
// Solidity: function swapBOBAForETHMetaTransaction(address tokenOwner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) SwapBOBAForETHMetaTransaction(tokenOwner libcommon.Address, spender libcommon.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.SwapBOBAForETHMetaTransaction(&_BobaGasPriceOracle.TransactOpts, tokenOwner, spender, value, deadline, v, r, s)
}

// SwapBOBAForETHMetaTransaction is a paid mutator transaction binding the contract method 0xb54016dc.
//
// Solidity: function swapBOBAForETHMetaTransaction(address tokenOwner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) SwapBOBAForETHMetaTransaction(tokenOwner libcommon.Address, spender libcommon.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.SwapBOBAForETHMetaTransaction(&_BobaGasPriceOracle.TransactOpts, tokenOwner, spender, value, deadline, v, r, s)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) TransferOwnership(opts *bind.TransactOpts, _newOwner libcommon.Address) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "transferOwnership", _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) TransferOwnership(_newOwner libcommon.Address) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.TransferOwnership(&_BobaGasPriceOracle.TransactOpts, _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) TransferOwnership(_newOwner libcommon.Address) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.TransferOwnership(&_BobaGasPriceOracle.TransactOpts, _newOwner)
}

// UpdateGasPriceOracleAddress is a paid mutator transaction binding the contract method 0x8fcfc813.
//
// Solidity: function updateGasPriceOracleAddress(address _gasPriceOracleAddress) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) UpdateGasPriceOracleAddress(opts *bind.TransactOpts, _gasPriceOracleAddress libcommon.Address) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "updateGasPriceOracleAddress", _gasPriceOracleAddress)
}

// UpdateGasPriceOracleAddress is a paid mutator transaction binding the contract method 0x8fcfc813.
//
// Solidity: function updateGasPriceOracleAddress(address _gasPriceOracleAddress) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) UpdateGasPriceOracleAddress(_gasPriceOracleAddress libcommon.Address) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdateGasPriceOracleAddress(&_BobaGasPriceOracle.TransactOpts, _gasPriceOracleAddress)
}

// UpdateGasPriceOracleAddress is a paid mutator transaction binding the contract method 0x8fcfc813.
//
// Solidity: function updateGasPriceOracleAddress(address _gasPriceOracleAddress) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) UpdateGasPriceOracleAddress(_gasPriceOracleAddress libcommon.Address) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdateGasPriceOracleAddress(&_BobaGasPriceOracle.TransactOpts, _gasPriceOracleAddress)
}

// UpdateMaxPriceRatio is a paid mutator transaction binding the contract method 0xc8a05413.
//
// Solidity: function updateMaxPriceRatio(uint256 _maxPriceRatio) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) UpdateMaxPriceRatio(opts *bind.TransactOpts, _maxPriceRatio *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "updateMaxPriceRatio", _maxPriceRatio)
}

// UpdateMaxPriceRatio is a paid mutator transaction binding the contract method 0xc8a05413.
//
// Solidity: function updateMaxPriceRatio(uint256 _maxPriceRatio) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) UpdateMaxPriceRatio(_maxPriceRatio *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdateMaxPriceRatio(&_BobaGasPriceOracle.TransactOpts, _maxPriceRatio)
}

// UpdateMaxPriceRatio is a paid mutator transaction binding the contract method 0xc8a05413.
//
// Solidity: function updateMaxPriceRatio(uint256 _maxPriceRatio) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) UpdateMaxPriceRatio(_maxPriceRatio *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdateMaxPriceRatio(&_BobaGasPriceOracle.TransactOpts, _maxPriceRatio)
}

// UpdateMetaTransactionFee is a paid mutator transaction binding the contract method 0xe3aea9ba.
//
// Solidity: function updateMetaTransactionFee(uint256 _metaTransactionFee) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) UpdateMetaTransactionFee(opts *bind.TransactOpts, _metaTransactionFee *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "updateMetaTransactionFee", _metaTransactionFee)
}

// UpdateMetaTransactionFee is a paid mutator transaction binding the contract method 0xe3aea9ba.
//
// Solidity: function updateMetaTransactionFee(uint256 _metaTransactionFee) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) UpdateMetaTransactionFee(_metaTransactionFee *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdateMetaTransactionFee(&_BobaGasPriceOracle.TransactOpts, _metaTransactionFee)
}

// UpdateMetaTransactionFee is a paid mutator transaction binding the contract method 0xe3aea9ba.
//
// Solidity: function updateMetaTransactionFee(uint256 _metaTransactionFee) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) UpdateMetaTransactionFee(_metaTransactionFee *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdateMetaTransactionFee(&_BobaGasPriceOracle.TransactOpts, _metaTransactionFee)
}

// UpdateMinPriceRatio is a paid mutator transaction binding the contract method 0x005c5fb2.
//
// Solidity: function updateMinPriceRatio(uint256 _minPriceRatio) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) UpdateMinPriceRatio(opts *bind.TransactOpts, _minPriceRatio *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "updateMinPriceRatio", _minPriceRatio)
}

// UpdateMinPriceRatio is a paid mutator transaction binding the contract method 0x005c5fb2.
//
// Solidity: function updateMinPriceRatio(uint256 _minPriceRatio) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) UpdateMinPriceRatio(_minPriceRatio *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdateMinPriceRatio(&_BobaGasPriceOracle.TransactOpts, _minPriceRatio)
}

// UpdateMinPriceRatio is a paid mutator transaction binding the contract method 0x005c5fb2.
//
// Solidity: function updateMinPriceRatio(uint256 _minPriceRatio) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) UpdateMinPriceRatio(_minPriceRatio *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdateMinPriceRatio(&_BobaGasPriceOracle.TransactOpts, _minPriceRatio)
}

// UpdatePriceRatio is a paid mutator transaction binding the contract method 0xbc9bd6ee.
//
// Solidity: function updatePriceRatio(uint256 _priceRatio, uint256 _marketPriceRatio) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) UpdatePriceRatio(opts *bind.TransactOpts, _priceRatio *big.Int, _marketPriceRatio *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "updatePriceRatio", _priceRatio, _marketPriceRatio)
}

// UpdatePriceRatio is a paid mutator transaction binding the contract method 0xbc9bd6ee.
//
// Solidity: function updatePriceRatio(uint256 _priceRatio, uint256 _marketPriceRatio) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) UpdatePriceRatio(_priceRatio *big.Int, _marketPriceRatio *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdatePriceRatio(&_BobaGasPriceOracle.TransactOpts, _priceRatio, _marketPriceRatio)
}

// UpdatePriceRatio is a paid mutator transaction binding the contract method 0xbc9bd6ee.
//
// Solidity: function updatePriceRatio(uint256 _priceRatio, uint256 _marketPriceRatio) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) UpdatePriceRatio(_priceRatio *big.Int, _marketPriceRatio *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdatePriceRatio(&_BobaGasPriceOracle.TransactOpts, _priceRatio, _marketPriceRatio)
}

// UpdateReceivedETHAmount is a paid mutator transaction binding the contract method 0x5b9da5c6.
//
// Solidity: function updateReceivedETHAmount(uint256 _receivedETHAmount) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) UpdateReceivedETHAmount(opts *bind.TransactOpts, _receivedETHAmount *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "updateReceivedETHAmount", _receivedETHAmount)
}

// UpdateReceivedETHAmount is a paid mutator transaction binding the contract method 0x5b9da5c6.
//
// Solidity: function updateReceivedETHAmount(uint256 _receivedETHAmount) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) UpdateReceivedETHAmount(_receivedETHAmount *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdateReceivedETHAmount(&_BobaGasPriceOracle.TransactOpts, _receivedETHAmount)
}

// UpdateReceivedETHAmount is a paid mutator transaction binding the contract method 0x5b9da5c6.
//
// Solidity: function updateReceivedETHAmount(uint256 _receivedETHAmount) returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) UpdateReceivedETHAmount(_receivedETHAmount *big.Int) (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UpdateReceivedETHAmount(&_BobaGasPriceOracle.TransactOpts, _receivedETHAmount)
}

// UseBobaAsFeeToken is a paid mutator transaction binding the contract method 0x34fe1b16.
//
// Solidity: function useBobaAsFeeToken() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) UseBobaAsFeeToken(opts *bind.TransactOpts) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "useBobaAsFeeToken")
}

// UseBobaAsFeeToken is a paid mutator transaction binding the contract method 0x34fe1b16.
//
// Solidity: function useBobaAsFeeToken() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) UseBobaAsFeeToken() (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UseBobaAsFeeToken(&_BobaGasPriceOracle.TransactOpts)
}

// UseBobaAsFeeToken is a paid mutator transaction binding the contract method 0x34fe1b16.
//
// Solidity: function useBobaAsFeeToken() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) UseBobaAsFeeToken() (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UseBobaAsFeeToken(&_BobaGasPriceOracle.TransactOpts)
}

// UseETHAsFeeToken is a paid mutator transaction binding the contract method 0x1b677199.
//
// Solidity: function useETHAsFeeToken() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) UseETHAsFeeToken(opts *bind.TransactOpts) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "useETHAsFeeToken")
}

// UseETHAsFeeToken is a paid mutator transaction binding the contract method 0x1b677199.
//
// Solidity: function useETHAsFeeToken() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) UseETHAsFeeToken() (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UseETHAsFeeToken(&_BobaGasPriceOracle.TransactOpts)
}

// UseETHAsFeeToken is a paid mutator transaction binding the contract method 0x1b677199.
//
// Solidity: function useETHAsFeeToken() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) UseETHAsFeeToken() (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.UseETHAsFeeToken(&_BobaGasPriceOracle.TransactOpts)
}

// WithdrawBOBA is a paid mutator transaction binding the contract method 0x89df963d.
//
// Solidity: function withdrawBOBA() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) WithdrawBOBA(opts *bind.TransactOpts) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "withdrawBOBA")
}

// WithdrawBOBA is a paid mutator transaction binding the contract method 0x89df963d.
//
// Solidity: function withdrawBOBA() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) WithdrawBOBA() (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.WithdrawBOBA(&_BobaGasPriceOracle.TransactOpts)
}

// WithdrawBOBA is a paid mutator transaction binding the contract method 0x89df963d.
//
// Solidity: function withdrawBOBA() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) WithdrawBOBA() (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.WithdrawBOBA(&_BobaGasPriceOracle.TransactOpts)
}

// WithdrawETH is a paid mutator transaction binding the contract method 0xe086e5ec.
//
// Solidity: function withdrawETH() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) WithdrawETH(opts *bind.TransactOpts) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.Transact(opts, "withdrawETH")
}

// WithdrawETH is a paid mutator transaction binding the contract method 0xe086e5ec.
//
// Solidity: function withdrawETH() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) WithdrawETH() (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.WithdrawETH(&_BobaGasPriceOracle.TransactOpts)
}

// WithdrawETH is a paid mutator transaction binding the contract method 0xe086e5ec.
//
// Solidity: function withdrawETH() returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) WithdrawETH() (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.WithdrawETH(&_BobaGasPriceOracle.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactor) Receive(opts *bind.TransactOpts) (types.Transaction, error) {
	return _BobaGasPriceOracle.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleSession) Receive() (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.Receive(&_BobaGasPriceOracle.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BobaGasPriceOracle *BobaGasPriceOracleTransactorSession) Receive() (types.Transaction, error) {
	return _BobaGasPriceOracle.Contract.Receive(&_BobaGasPriceOracle.TransactOpts)
}

// BobaGasPriceOracleSwapBOBAForETHMetaTransactionIterator is returned from FilterSwapBOBAForETHMetaTransaction and is used to iterate over the raw logs and unpacked data for SwapBOBAForETHMetaTransaction events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleSwapBOBAForETHMetaTransactionIterator struct {
	Event *BobaGasPriceOracleSwapBOBAForETHMetaTransaction // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleSwapBOBAForETHMetaTransactionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleSwapBOBAForETHMetaTransaction)
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
		it.Event = new(BobaGasPriceOracleSwapBOBAForETHMetaTransaction)
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
func (it *BobaGasPriceOracleSwapBOBAForETHMetaTransactionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleSwapBOBAForETHMetaTransactionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleSwapBOBAForETHMetaTransaction represents a SwapBOBAForETHMetaTransaction event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleSwapBOBAForETHMetaTransaction struct {
	Arg0 libcommon.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterSwapBOBAForETHMetaTransaction is a free log retrieval operation binding the contract event 0xb92b4b358dfa6e521f7f80a5d0522cf04a2082482701a0d78ff2bb615df646be.
//
// Solidity: event SwapBOBAForETHMetaTransaction(address arg0)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterSwapBOBAForETHMetaTransaction(opts *bind.FilterOpts) (*BobaGasPriceOracleSwapBOBAForETHMetaTransactionIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "SwapBOBAForETHMetaTransaction")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleSwapBOBAForETHMetaTransactionIterator{contract: _BobaGasPriceOracle.contract, event: "SwapBOBAForETHMetaTransaction", logs: logs, sub: sub}, nil
}

// WatchSwapBOBAForETHMetaTransaction is a free log subscription operation binding the contract event 0xb92b4b358dfa6e521f7f80a5d0522cf04a2082482701a0d78ff2bb615df646be.
//
// Solidity: event SwapBOBAForETHMetaTransaction(address arg0)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchSwapBOBAForETHMetaTransaction(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleSwapBOBAForETHMetaTransaction) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "SwapBOBAForETHMetaTransaction")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleSwapBOBAForETHMetaTransaction)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "SwapBOBAForETHMetaTransaction", log); err != nil {
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

// ParseSwapBOBAForETHMetaTransaction is a log parse operation binding the contract event 0xb92b4b358dfa6e521f7f80a5d0522cf04a2082482701a0d78ff2bb615df646be.
//
// Solidity: event SwapBOBAForETHMetaTransaction(address arg0)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseSwapBOBAForETHMetaTransaction(log types.Log) (*BobaGasPriceOracleSwapBOBAForETHMetaTransaction, error) {
	event := new(BobaGasPriceOracleSwapBOBAForETHMetaTransaction)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "SwapBOBAForETHMetaTransaction", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleTransferOwnershipIterator is returned from FilterTransferOwnership and is used to iterate over the raw logs and unpacked data for TransferOwnership events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleTransferOwnershipIterator struct {
	Event *BobaGasPriceOracleTransferOwnership // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleTransferOwnershipIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleTransferOwnership)
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
		it.Event = new(BobaGasPriceOracleTransferOwnership)
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
func (it *BobaGasPriceOracleTransferOwnershipIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleTransferOwnershipIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleTransferOwnership represents a TransferOwnership event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleTransferOwnership struct {
	Arg0 libcommon.Address
	Arg1 libcommon.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterTransferOwnership is a free log retrieval operation binding the contract event 0x5c486528ec3e3f0ea91181cff8116f02bfa350e03b8b6f12e00765adbb5af85c.
//
// Solidity: event TransferOwnership(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterTransferOwnership(opts *bind.FilterOpts) (*BobaGasPriceOracleTransferOwnershipIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "TransferOwnership")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleTransferOwnershipIterator{contract: _BobaGasPriceOracle.contract, event: "TransferOwnership", logs: logs, sub: sub}, nil
}

// WatchTransferOwnership is a free log subscription operation binding the contract event 0x5c486528ec3e3f0ea91181cff8116f02bfa350e03b8b6f12e00765adbb5af85c.
//
// Solidity: event TransferOwnership(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchTransferOwnership(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleTransferOwnership) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "TransferOwnership")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleTransferOwnership)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "TransferOwnership", log); err != nil {
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
// Solidity: event TransferOwnership(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseTransferOwnership(log types.Log) (*BobaGasPriceOracleTransferOwnership, error) {
	event := new(BobaGasPriceOracleTransferOwnership)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "TransferOwnership", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleUpdateGasPriceOracleAddressIterator is returned from FilterUpdateGasPriceOracleAddress and is used to iterate over the raw logs and unpacked data for UpdateGasPriceOracleAddress events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdateGasPriceOracleAddressIterator struct {
	Event *BobaGasPriceOracleUpdateGasPriceOracleAddress // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleUpdateGasPriceOracleAddressIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleUpdateGasPriceOracleAddress)
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
		it.Event = new(BobaGasPriceOracleUpdateGasPriceOracleAddress)
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
func (it *BobaGasPriceOracleUpdateGasPriceOracleAddressIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleUpdateGasPriceOracleAddressIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleUpdateGasPriceOracleAddress represents a UpdateGasPriceOracleAddress event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdateGasPriceOracleAddress struct {
	Arg0 libcommon.Address
	Arg1 libcommon.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterUpdateGasPriceOracleAddress is a free log retrieval operation binding the contract event 0x226bf99888a1e70d41ce744b11ce2acd4d1d1b8cf4ad17a0e72e67acff4bf5a7.
//
// Solidity: event UpdateGasPriceOracleAddress(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterUpdateGasPriceOracleAddress(opts *bind.FilterOpts) (*BobaGasPriceOracleUpdateGasPriceOracleAddressIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "UpdateGasPriceOracleAddress")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleUpdateGasPriceOracleAddressIterator{contract: _BobaGasPriceOracle.contract, event: "UpdateGasPriceOracleAddress", logs: logs, sub: sub}, nil
}

// WatchUpdateGasPriceOracleAddress is a free log subscription operation binding the contract event 0x226bf99888a1e70d41ce744b11ce2acd4d1d1b8cf4ad17a0e72e67acff4bf5a7.
//
// Solidity: event UpdateGasPriceOracleAddress(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchUpdateGasPriceOracleAddress(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleUpdateGasPriceOracleAddress) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "UpdateGasPriceOracleAddress")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleUpdateGasPriceOracleAddress)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdateGasPriceOracleAddress", log); err != nil {
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

// ParseUpdateGasPriceOracleAddress is a log parse operation binding the contract event 0x226bf99888a1e70d41ce744b11ce2acd4d1d1b8cf4ad17a0e72e67acff4bf5a7.
//
// Solidity: event UpdateGasPriceOracleAddress(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseUpdateGasPriceOracleAddress(log types.Log) (*BobaGasPriceOracleUpdateGasPriceOracleAddress, error) {
	event := new(BobaGasPriceOracleUpdateGasPriceOracleAddress)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdateGasPriceOracleAddress", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleUpdateMaxPriceRatioIterator is returned from FilterUpdateMaxPriceRatio and is used to iterate over the raw logs and unpacked data for UpdateMaxPriceRatio events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdateMaxPriceRatioIterator struct {
	Event *BobaGasPriceOracleUpdateMaxPriceRatio // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleUpdateMaxPriceRatioIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleUpdateMaxPriceRatio)
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
		it.Event = new(BobaGasPriceOracleUpdateMaxPriceRatio)
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
func (it *BobaGasPriceOracleUpdateMaxPriceRatioIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleUpdateMaxPriceRatioIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleUpdateMaxPriceRatio represents a UpdateMaxPriceRatio event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdateMaxPriceRatio struct {
	Arg0 libcommon.Address
	Arg1 *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterUpdateMaxPriceRatio is a free log retrieval operation binding the contract event 0x7a28f69b71e51c4a30f620a2cfe4ce5aad2cd3fe5cc9647e400e252b65033d41.
//
// Solidity: event UpdateMaxPriceRatio(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterUpdateMaxPriceRatio(opts *bind.FilterOpts) (*BobaGasPriceOracleUpdateMaxPriceRatioIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "UpdateMaxPriceRatio")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleUpdateMaxPriceRatioIterator{contract: _BobaGasPriceOracle.contract, event: "UpdateMaxPriceRatio", logs: logs, sub: sub}, nil
}

// WatchUpdateMaxPriceRatio is a free log subscription operation binding the contract event 0x7a28f69b71e51c4a30f620a2cfe4ce5aad2cd3fe5cc9647e400e252b65033d41.
//
// Solidity: event UpdateMaxPriceRatio(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchUpdateMaxPriceRatio(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleUpdateMaxPriceRatio) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "UpdateMaxPriceRatio")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleUpdateMaxPriceRatio)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdateMaxPriceRatio", log); err != nil {
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

// ParseUpdateMaxPriceRatio is a log parse operation binding the contract event 0x7a28f69b71e51c4a30f620a2cfe4ce5aad2cd3fe5cc9647e400e252b65033d41.
//
// Solidity: event UpdateMaxPriceRatio(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseUpdateMaxPriceRatio(log types.Log) (*BobaGasPriceOracleUpdateMaxPriceRatio, error) {
	event := new(BobaGasPriceOracleUpdateMaxPriceRatio)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdateMaxPriceRatio", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleUpdateMetaTransactionFeeIterator is returned from FilterUpdateMetaTransactionFee and is used to iterate over the raw logs and unpacked data for UpdateMetaTransactionFee events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdateMetaTransactionFeeIterator struct {
	Event *BobaGasPriceOracleUpdateMetaTransactionFee // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleUpdateMetaTransactionFeeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleUpdateMetaTransactionFee)
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
		it.Event = new(BobaGasPriceOracleUpdateMetaTransactionFee)
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
func (it *BobaGasPriceOracleUpdateMetaTransactionFeeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleUpdateMetaTransactionFeeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleUpdateMetaTransactionFee represents a UpdateMetaTransactionFee event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdateMetaTransactionFee struct {
	Arg0 libcommon.Address
	Arg1 *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterUpdateMetaTransactionFee is a free log retrieval operation binding the contract event 0x1071f61d642716391065a6f38aac12cdc6a436ca6a6622a18ae0530495738afc.
//
// Solidity: event UpdateMetaTransactionFee(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterUpdateMetaTransactionFee(opts *bind.FilterOpts) (*BobaGasPriceOracleUpdateMetaTransactionFeeIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "UpdateMetaTransactionFee")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleUpdateMetaTransactionFeeIterator{contract: _BobaGasPriceOracle.contract, event: "UpdateMetaTransactionFee", logs: logs, sub: sub}, nil
}

// WatchUpdateMetaTransactionFee is a free log subscription operation binding the contract event 0x1071f61d642716391065a6f38aac12cdc6a436ca6a6622a18ae0530495738afc.
//
// Solidity: event UpdateMetaTransactionFee(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchUpdateMetaTransactionFee(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleUpdateMetaTransactionFee) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "UpdateMetaTransactionFee")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleUpdateMetaTransactionFee)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdateMetaTransactionFee", log); err != nil {
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

// ParseUpdateMetaTransactionFee is a log parse operation binding the contract event 0x1071f61d642716391065a6f38aac12cdc6a436ca6a6622a18ae0530495738afc.
//
// Solidity: event UpdateMetaTransactionFee(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseUpdateMetaTransactionFee(log types.Log) (*BobaGasPriceOracleUpdateMetaTransactionFee, error) {
	event := new(BobaGasPriceOracleUpdateMetaTransactionFee)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdateMetaTransactionFee", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleUpdateMinPriceRatioIterator is returned from FilterUpdateMinPriceRatio and is used to iterate over the raw logs and unpacked data for UpdateMinPriceRatio events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdateMinPriceRatioIterator struct {
	Event *BobaGasPriceOracleUpdateMinPriceRatio // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleUpdateMinPriceRatioIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleUpdateMinPriceRatio)
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
		it.Event = new(BobaGasPriceOracleUpdateMinPriceRatio)
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
func (it *BobaGasPriceOracleUpdateMinPriceRatioIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleUpdateMinPriceRatioIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleUpdateMinPriceRatio represents a UpdateMinPriceRatio event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdateMinPriceRatio struct {
	Arg0 libcommon.Address
	Arg1 *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterUpdateMinPriceRatio is a free log retrieval operation binding the contract event 0x680f379280fc8680df45c979a924c0084a250758604482cb01dadedbaa1c09c9.
//
// Solidity: event UpdateMinPriceRatio(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterUpdateMinPriceRatio(opts *bind.FilterOpts) (*BobaGasPriceOracleUpdateMinPriceRatioIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "UpdateMinPriceRatio")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleUpdateMinPriceRatioIterator{contract: _BobaGasPriceOracle.contract, event: "UpdateMinPriceRatio", logs: logs, sub: sub}, nil
}

// WatchUpdateMinPriceRatio is a free log subscription operation binding the contract event 0x680f379280fc8680df45c979a924c0084a250758604482cb01dadedbaa1c09c9.
//
// Solidity: event UpdateMinPriceRatio(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchUpdateMinPriceRatio(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleUpdateMinPriceRatio) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "UpdateMinPriceRatio")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleUpdateMinPriceRatio)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdateMinPriceRatio", log); err != nil {
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

// ParseUpdateMinPriceRatio is a log parse operation binding the contract event 0x680f379280fc8680df45c979a924c0084a250758604482cb01dadedbaa1c09c9.
//
// Solidity: event UpdateMinPriceRatio(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseUpdateMinPriceRatio(log types.Log) (*BobaGasPriceOracleUpdateMinPriceRatio, error) {
	event := new(BobaGasPriceOracleUpdateMinPriceRatio)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdateMinPriceRatio", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleUpdatePriceRatioIterator is returned from FilterUpdatePriceRatio and is used to iterate over the raw logs and unpacked data for UpdatePriceRatio events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdatePriceRatioIterator struct {
	Event *BobaGasPriceOracleUpdatePriceRatio // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleUpdatePriceRatioIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleUpdatePriceRatio)
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
		it.Event = new(BobaGasPriceOracleUpdatePriceRatio)
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
func (it *BobaGasPriceOracleUpdatePriceRatioIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleUpdatePriceRatioIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleUpdatePriceRatio represents a UpdatePriceRatio event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdatePriceRatio struct {
	Arg0 libcommon.Address
	Arg1 *big.Int
	Arg2 *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterUpdatePriceRatio is a free log retrieval operation binding the contract event 0x23632bbb735dece542dac9735a2ba4253234eb119ce45cdf9968cbbe12aa6790.
//
// Solidity: event UpdatePriceRatio(address arg0, uint256 arg1, uint256 arg2)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterUpdatePriceRatio(opts *bind.FilterOpts) (*BobaGasPriceOracleUpdatePriceRatioIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "UpdatePriceRatio")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleUpdatePriceRatioIterator{contract: _BobaGasPriceOracle.contract, event: "UpdatePriceRatio", logs: logs, sub: sub}, nil
}

// WatchUpdatePriceRatio is a free log subscription operation binding the contract event 0x23632bbb735dece542dac9735a2ba4253234eb119ce45cdf9968cbbe12aa6790.
//
// Solidity: event UpdatePriceRatio(address arg0, uint256 arg1, uint256 arg2)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchUpdatePriceRatio(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleUpdatePriceRatio) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "UpdatePriceRatio")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleUpdatePriceRatio)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdatePriceRatio", log); err != nil {
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

// ParseUpdatePriceRatio is a log parse operation binding the contract event 0x23632bbb735dece542dac9735a2ba4253234eb119ce45cdf9968cbbe12aa6790.
//
// Solidity: event UpdatePriceRatio(address arg0, uint256 arg1, uint256 arg2)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseUpdatePriceRatio(log types.Log) (*BobaGasPriceOracleUpdatePriceRatio, error) {
	event := new(BobaGasPriceOracleUpdatePriceRatio)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdatePriceRatio", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleUpdateReceivedETHAmountIterator is returned from FilterUpdateReceivedETHAmount and is used to iterate over the raw logs and unpacked data for UpdateReceivedETHAmount events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdateReceivedETHAmountIterator struct {
	Event *BobaGasPriceOracleUpdateReceivedETHAmount // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleUpdateReceivedETHAmountIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleUpdateReceivedETHAmount)
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
		it.Event = new(BobaGasPriceOracleUpdateReceivedETHAmount)
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
func (it *BobaGasPriceOracleUpdateReceivedETHAmountIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleUpdateReceivedETHAmountIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleUpdateReceivedETHAmount represents a UpdateReceivedETHAmount event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUpdateReceivedETHAmount struct {
	Arg0 libcommon.Address
	Arg1 *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterUpdateReceivedETHAmount is a free log retrieval operation binding the contract event 0xdcb9e069a0d16a974c9c0f4a88e2c9b79df5c45d9721c26461043d51c4468207.
//
// Solidity: event UpdateReceivedETHAmount(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterUpdateReceivedETHAmount(opts *bind.FilterOpts) (*BobaGasPriceOracleUpdateReceivedETHAmountIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "UpdateReceivedETHAmount")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleUpdateReceivedETHAmountIterator{contract: _BobaGasPriceOracle.contract, event: "UpdateReceivedETHAmount", logs: logs, sub: sub}, nil
}

// WatchUpdateReceivedETHAmount is a free log subscription operation binding the contract event 0xdcb9e069a0d16a974c9c0f4a88e2c9b79df5c45d9721c26461043d51c4468207.
//
// Solidity: event UpdateReceivedETHAmount(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchUpdateReceivedETHAmount(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleUpdateReceivedETHAmount) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "UpdateReceivedETHAmount")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleUpdateReceivedETHAmount)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdateReceivedETHAmount", log); err != nil {
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

// ParseUpdateReceivedETHAmount is a log parse operation binding the contract event 0xdcb9e069a0d16a974c9c0f4a88e2c9b79df5c45d9721c26461043d51c4468207.
//
// Solidity: event UpdateReceivedETHAmount(address arg0, uint256 arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseUpdateReceivedETHAmount(log types.Log) (*BobaGasPriceOracleUpdateReceivedETHAmount, error) {
	event := new(BobaGasPriceOracleUpdateReceivedETHAmount)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UpdateReceivedETHAmount", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleUseBobaAsFeeTokenIterator is returned from FilterUseBobaAsFeeToken and is used to iterate over the raw logs and unpacked data for UseBobaAsFeeToken events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUseBobaAsFeeTokenIterator struct {
	Event *BobaGasPriceOracleUseBobaAsFeeToken // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleUseBobaAsFeeTokenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleUseBobaAsFeeToken)
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
		it.Event = new(BobaGasPriceOracleUseBobaAsFeeToken)
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
func (it *BobaGasPriceOracleUseBobaAsFeeTokenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleUseBobaAsFeeTokenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleUseBobaAsFeeToken represents a UseBobaAsFeeToken event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUseBobaAsFeeToken struct {
	Arg0 libcommon.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterUseBobaAsFeeToken is a free log retrieval operation binding the contract event 0xd1787ba09c5383b33cf88983fbbf2e6ae348746a3a906e1a1bb67c729661a4ac.
//
// Solidity: event UseBobaAsFeeToken(address arg0)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterUseBobaAsFeeToken(opts *bind.FilterOpts) (*BobaGasPriceOracleUseBobaAsFeeTokenIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "UseBobaAsFeeToken")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleUseBobaAsFeeTokenIterator{contract: _BobaGasPriceOracle.contract, event: "UseBobaAsFeeToken", logs: logs, sub: sub}, nil
}

// WatchUseBobaAsFeeToken is a free log subscription operation binding the contract event 0xd1787ba09c5383b33cf88983fbbf2e6ae348746a3a906e1a1bb67c729661a4ac.
//
// Solidity: event UseBobaAsFeeToken(address arg0)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchUseBobaAsFeeToken(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleUseBobaAsFeeToken) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "UseBobaAsFeeToken")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleUseBobaAsFeeToken)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UseBobaAsFeeToken", log); err != nil {
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

// ParseUseBobaAsFeeToken is a log parse operation binding the contract event 0xd1787ba09c5383b33cf88983fbbf2e6ae348746a3a906e1a1bb67c729661a4ac.
//
// Solidity: event UseBobaAsFeeToken(address arg0)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseUseBobaAsFeeToken(log types.Log) (*BobaGasPriceOracleUseBobaAsFeeToken, error) {
	event := new(BobaGasPriceOracleUseBobaAsFeeToken)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UseBobaAsFeeToken", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleUseETHAsFeeTokenIterator is returned from FilterUseETHAsFeeToken and is used to iterate over the raw logs and unpacked data for UseETHAsFeeToken events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUseETHAsFeeTokenIterator struct {
	Event *BobaGasPriceOracleUseETHAsFeeToken // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleUseETHAsFeeTokenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleUseETHAsFeeToken)
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
		it.Event = new(BobaGasPriceOracleUseETHAsFeeToken)
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
func (it *BobaGasPriceOracleUseETHAsFeeTokenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleUseETHAsFeeTokenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleUseETHAsFeeToken represents a UseETHAsFeeToken event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleUseETHAsFeeToken struct {
	Arg0 libcommon.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterUseETHAsFeeToken is a free log retrieval operation binding the contract event 0x764389830e6a6b84f4ea3f2551a4c5afbb6dff806f2d8f571f6913c6c4b62a40.
//
// Solidity: event UseETHAsFeeToken(address arg0)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterUseETHAsFeeToken(opts *bind.FilterOpts) (*BobaGasPriceOracleUseETHAsFeeTokenIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "UseETHAsFeeToken")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleUseETHAsFeeTokenIterator{contract: _BobaGasPriceOracle.contract, event: "UseETHAsFeeToken", logs: logs, sub: sub}, nil
}

// WatchUseETHAsFeeToken is a free log subscription operation binding the contract event 0x764389830e6a6b84f4ea3f2551a4c5afbb6dff806f2d8f571f6913c6c4b62a40.
//
// Solidity: event UseETHAsFeeToken(address arg0)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchUseETHAsFeeToken(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleUseETHAsFeeToken) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "UseETHAsFeeToken")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleUseETHAsFeeToken)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UseETHAsFeeToken", log); err != nil {
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

// ParseUseETHAsFeeToken is a log parse operation binding the contract event 0x764389830e6a6b84f4ea3f2551a4c5afbb6dff806f2d8f571f6913c6c4b62a40.
//
// Solidity: event UseETHAsFeeToken(address arg0)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseUseETHAsFeeToken(log types.Log) (*BobaGasPriceOracleUseETHAsFeeToken, error) {
	event := new(BobaGasPriceOracleUseETHAsFeeToken)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "UseETHAsFeeToken", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleWithdrawBOBAIterator is returned from FilterWithdrawBOBA and is used to iterate over the raw logs and unpacked data for WithdrawBOBA events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleWithdrawBOBAIterator struct {
	Event *BobaGasPriceOracleWithdrawBOBA // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleWithdrawBOBAIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleWithdrawBOBA)
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
		it.Event = new(BobaGasPriceOracleWithdrawBOBA)
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
func (it *BobaGasPriceOracleWithdrawBOBAIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleWithdrawBOBAIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleWithdrawBOBA represents a WithdrawBOBA event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleWithdrawBOBA struct {
	Arg0 libcommon.Address
	Arg1 libcommon.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterWithdrawBOBA is a free log retrieval operation binding the contract event 0x2c69c3957d9ca9782726f647b7a3592dd381f4370288551f5ed43fd3cc5b7753.
//
// Solidity: event WithdrawBOBA(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterWithdrawBOBA(opts *bind.FilterOpts) (*BobaGasPriceOracleWithdrawBOBAIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "WithdrawBOBA")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleWithdrawBOBAIterator{contract: _BobaGasPriceOracle.contract, event: "WithdrawBOBA", logs: logs, sub: sub}, nil
}

// WatchWithdrawBOBA is a free log subscription operation binding the contract event 0x2c69c3957d9ca9782726f647b7a3592dd381f4370288551f5ed43fd3cc5b7753.
//
// Solidity: event WithdrawBOBA(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchWithdrawBOBA(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleWithdrawBOBA) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "WithdrawBOBA")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleWithdrawBOBA)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "WithdrawBOBA", log); err != nil {
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

// ParseWithdrawBOBA is a log parse operation binding the contract event 0x2c69c3957d9ca9782726f647b7a3592dd381f4370288551f5ed43fd3cc5b7753.
//
// Solidity: event WithdrawBOBA(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseWithdrawBOBA(log types.Log) (*BobaGasPriceOracleWithdrawBOBA, error) {
	event := new(BobaGasPriceOracleWithdrawBOBA)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "WithdrawBOBA", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BobaGasPriceOracleWithdrawETHIterator is returned from FilterWithdrawETH and is used to iterate over the raw logs and unpacked data for WithdrawETH events raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleWithdrawETHIterator struct {
	Event *BobaGasPriceOracleWithdrawETH // Event containing the contract specifics and raw log

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
func (it *BobaGasPriceOracleWithdrawETHIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BobaGasPriceOracleWithdrawETH)
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
		it.Event = new(BobaGasPriceOracleWithdrawETH)
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
func (it *BobaGasPriceOracleWithdrawETHIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BobaGasPriceOracleWithdrawETHIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BobaGasPriceOracleWithdrawETH represents a WithdrawETH event raised by the BobaGasPriceOracle contract.
type BobaGasPriceOracleWithdrawETH struct {
	Arg0 libcommon.Address
	Arg1 libcommon.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterWithdrawETH is a free log retrieval operation binding the contract event 0x6de63bb986f2779478e384365c03cc2e62f06b453856acca87d5a519ce026649.
//
// Solidity: event WithdrawETH(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) FilterWithdrawETH(opts *bind.FilterOpts) (*BobaGasPriceOracleWithdrawETHIterator, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.FilterLogs(opts, "WithdrawETH")
	if err != nil {
		return nil, err
	}
	return &BobaGasPriceOracleWithdrawETHIterator{contract: _BobaGasPriceOracle.contract, event: "WithdrawETH", logs: logs, sub: sub}, nil
}

// WatchWithdrawETH is a free log subscription operation binding the contract event 0x6de63bb986f2779478e384365c03cc2e62f06b453856acca87d5a519ce026649.
//
// Solidity: event WithdrawETH(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) WatchWithdrawETH(opts *bind.WatchOpts, sink chan<- *BobaGasPriceOracleWithdrawETH) (event.Subscription, error) {

	logs, sub, err := _BobaGasPriceOracle.contract.WatchLogs(opts, "WithdrawETH")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BobaGasPriceOracleWithdrawETH)
				if err := _BobaGasPriceOracle.contract.UnpackLog(event, "WithdrawETH", log); err != nil {
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

// ParseWithdrawETH is a log parse operation binding the contract event 0x6de63bb986f2779478e384365c03cc2e62f06b453856acca87d5a519ce026649.
//
// Solidity: event WithdrawETH(address arg0, address arg1)
func (_BobaGasPriceOracle *BobaGasPriceOracleFilterer) ParseWithdrawETH(log types.Log) (*BobaGasPriceOracleWithdrawETH, error) {
	event := new(BobaGasPriceOracleWithdrawETH)
	if err := _BobaGasPriceOracle.contract.UnpackLog(event, "WithdrawETH", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
