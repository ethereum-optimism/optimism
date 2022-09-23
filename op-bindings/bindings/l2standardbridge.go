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

// L2StandardBridgeMetaData contains all meta data concerning the L2StandardBridge contract.
var L2StandardBridgeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_otherBridge\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"DepositFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ERC20BridgeFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ERC20BridgeInitiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ETHBridgeFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ETHBridgeInitiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"WithdrawalInitiated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeERC20\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeERC20To\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeETHTo\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"deposits\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"finalizeBridgeERC20\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"finalizeBridgeETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1Token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_l2Token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"finalizeDeposit\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"otherBridge\",\"outputs\":[{\"internalType\":\"contractStandardBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l2Token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l2Token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"withdrawTo\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x6101206040523480156200001257600080fd5b50604051620029043803806200290483398101604081905262000035916200006f565b7342000000000000000000000000000000000000076080526001600160a01b031660a052600060c081905260e052600261010052620000a1565b6000602082840312156200008257600080fd5b81516001600160a01b03811681146200009a57600080fd5b9392505050565b60805160a05160c05160e051610100516127d162000133600039600061102f0152600061100601526000610fdd01526000818161032c015281816104070152818161055301528181610a5e0152611948015260008181610200015281816103ca015281816105290152818161058a01528181610a3401528181610a9501528181610d22015261190c01526127d16000f3fe6080604052600436106100d65760003560e01c806354fd4d501161007f5780638f601f66116100595780638f601f66146102c1578063a3a7954814610307578063c89701a21461031a578063e11013dd1461034e57600080fd5b806354fd4d501461026c578063662a633a1461028e57806387087623146102a157600080fd5b806332b7006d116100b057806332b7006d146101db5780633cb747bf146101ee578063540abf731461024c57600080fd5b80630166a07a1461019557806309fc8843146101b55780631635f5fd146101c857600080fd5b3661019057333b1561016f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f4100000000000000000060648201526084015b60405180910390fd5b61018e33333462030d4060405180602001604052806000815250610361565b005b600080fd5b3480156101a157600080fd5b5061018e6101b0366004612018565b610511565b61018e6101c33660046120c9565b610945565b61018e6101d636600461211c565b610a1c565b61018e6101e936600461218f565b610f18565b3480156101fa57600080fd5b506102227f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b34801561025857600080fd5b5061018e6102673660046121e3565b610fbd565b34801561027857600080fd5b50610281610fd6565b60405161024391906122d0565b61018e61029c366004612018565b611079565b3480156102ad57600080fd5b5061018e6102bc3660046122e3565b611166565b3480156102cd57600080fd5b506102f96102dc366004612366565b600260209081526000928352604080842090915290825290205481565b604051908152602001610243565b61018e6103153660046122e3565b611205565b34801561032657600080fd5b506102227f000000000000000000000000000000000000000000000000000000000000000081565b61018e61035c36600461239f565b611214565b8373ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff167f2849b43074093a05396b6f2a937dee8565b15a48a7b3d4bffb732a5017380af585846040516103c0929190612402565b60405180910390a37f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16633dbb202b847f0000000000000000000000000000000000000000000000000000000000000000631635f5fd60e01b89898988604051602401610445949392919061241b565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e086901b90921682526104d892918890600401612464565b6000604051808303818588803b1580156104f157600080fd5b505af1158015610505573d6000803e3d6000fd5b50505050505050505050565b3373ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614801561062f57507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105f3573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061061791906124a9565b73ffffffffffffffffffffffffffffffffffffffff16145b6106e1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20746865206f7468657220627269646760648201527f6500000000000000000000000000000000000000000000000000000000000000608482015260a401610166565b6106ea8761125d565b15610838576106f987876112bf565b6107ab576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f5374616e646172644272696467653a2077726f6e672072656d6f746520746f6b60448201527f656e20666f72204f7074696d69736d204d696e7461626c65204552433230206c60648201527f6f63616c20746f6b656e00000000000000000000000000000000000000000000608482015260a401610166565b6040517f40c10f1900000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8581166004830152602482018590528816906340c10f1990604401600060405180830381600087803b15801561081b57600080fd5b505af115801561082f573d6000803e3d6000fd5b505050506108ba565b73ffffffffffffffffffffffffffffffffffffffff8088166000908152600260209081526040808320938a16835292905220546108769084906124f5565b73ffffffffffffffffffffffffffffffffffffffff8089166000818152600260209081526040808320948c16835293905291909120919091556108ba908585611366565b8473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff167fd59c65b35445225835c83f50b6ede06a7be047d22e357073e250d9af537518cd878787876040516109349493929190612555565b60405180910390a450505050505050565b333b156109d4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f410000000000000000006064820152608401610166565b610a173333348686868080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061036192505050565b505050565b3373ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016148015610b3a57507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b8152600401602060405180830381865afa158015610afe573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b2291906124a9565b73ffffffffffffffffffffffffffffffffffffffff16145b610bec576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20746865206f7468657220627269646760648201527f6500000000000000000000000000000000000000000000000000000000000000608482015260a401610166565b823414610c7b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f5374616e646172644272696467653a20616d6f756e742073656e7420646f657360448201527f206e6f74206d6174636820616d6f756e742072657175697265640000000000006064820152608401610166565b3073ffffffffffffffffffffffffffffffffffffffff851603610d20576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f5374616e646172644272696467653a2063616e6e6f742073656e6420746f207360448201527f656c6600000000000000000000000000000000000000000000000000000000006064820152608401610166565b7f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff1603610dfb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602860248201527f5374616e646172644272696467653a2063616e6e6f742073656e6420746f206d60448201527f657373656e6765720000000000000000000000000000000000000000000000006064820152608401610166565b8373ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff167f31b2166ff604fc5672ea5df08a78081d2bc6d746cadce880747f3643d819e83d858585604051610e5c9392919061258b565b60405180910390a36000610e81855a866040518060200160405280600081525061143a565b905080610f10576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f5374616e646172644272696467653a20455448207472616e736665722066616960448201527f6c656400000000000000000000000000000000000000000000000000000000006064820152608401610166565b505050505050565b333b15610fa7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f410000000000000000006064820152608401610166565b610fb685333387878787611454565b5050505050565b610fcd878733888888888861169d565b50505050505050565b60606110017f0000000000000000000000000000000000000000000000000000000000000000611a5b565b61102a7f0000000000000000000000000000000000000000000000000000000000000000611a5b565b6110537f0000000000000000000000000000000000000000000000000000000000000000611a5b565b604051602001611065939291906125ae565b604051602081830303815290604052905090565b73ffffffffffffffffffffffffffffffffffffffff87161580156110c6575073ffffffffffffffffffffffffffffffffffffffff861673deaddeaddeaddeaddeaddeaddeaddeaddead0000145b156110dd576110d88585858585610a1c565b6110ec565b6110ec86888787878787610511565b8473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff167fb0444523268717a02698be47d0803aa7468c00acbed2f8bd93a0459cde61dd89878787876040516109349493929190612555565b333b156111f5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f410000000000000000006064820152608401610166565b610f10868633338888888861169d565b610f1086338787878787611454565b6112573385348686868080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061036192505050565b50505050565b6000611289827f1d1d8b6300000000000000000000000000000000000000000000000000000000611b98565b806112b957506112b9827fec4fc8e300000000000000000000000000000000000000000000000000000000611b98565b92915050565b60008273ffffffffffffffffffffffffffffffffffffffff1663c01e1bd66040518163ffffffff1660e01b8152600401602060405180830381865afa15801561130c573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061133091906124a9565b73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1614905092915050565b60405173ffffffffffffffffffffffffffffffffffffffff8316602482015260448101829052610a179084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff0000000000000000000000000000000000000000000000000000000090931692909217909152611bbb565b600080600080845160208601878a8af19695505050505050565b60008773ffffffffffffffffffffffffffffffffffffffff1663c01e1bd66040518163ffffffff1660e01b8152600401602060405180830381865afa1580156114a1573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114c591906124a9565b90507fffffffffffffffffffffffff215221522152215221522152215221522153000073ffffffffffffffffffffffffffffffffffffffff891601611601578434146115b9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604360248201527f4c325374616e646172644272696467653a20455448207769746864726177616c60448201527f73206d75737420696e636c7564652073756666696369656e742045544820766160648201527f6c75650000000000000000000000000000000000000000000000000000000000608482015260a401610166565b6115fc8787878787878080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061036192505050565b611611565b611611888289898989898961169d565b8673ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff167f73d170910aba9e6d50b102db522b1dbcd796216f5128b445aa2135272886497e8989888860405161168b9493929190612555565b60405180910390a45050505050505050565b6116a68861125d565b156117f4576116b588886112bf565b611767576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f5374616e646172644272696467653a2077726f6e672072656d6f746520746f6b60448201527f656e20666f72204f7074696d69736d204d696e7461626c65204552433230206c60648201527f6f63616c20746f6b656e00000000000000000000000000000000000000000000608482015260a401610166565b6040517f9dc29fac00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff878116600483015260248201869052891690639dc29fac90604401600060405180830381600087803b1580156117d757600080fd5b505af11580156117eb573d6000803e3d6000fd5b50505050611888565b61181673ffffffffffffffffffffffffffffffffffffffff8916873087611cc7565b73ffffffffffffffffffffffffffffffffffffffff8089166000908152600260209081526040808320938b1683529290522054611854908590612624565b73ffffffffffffffffffffffffffffffffffffffff808a166000908152600260209081526040808320938c16835292905220555b8573ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff168973ffffffffffffffffffffffffffffffffffffffff167f7ff126db8024424bbfd9826e8ab82ff59136289ea440b04b39a0df1b03b9cabf888887876040516119029493929190612555565b60405180910390a47f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16633dbb202b7f0000000000000000000000000000000000000000000000000000000000000000630166a07a60e01b8a8c8b8b8b8a8a60405160240161198c979695949392919061263c565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e085901b9092168252611a1f92918890600401612464565b600060405180830381600087803b158015611a3957600080fd5b505af1158015611a4d573d6000803e3d6000fd5b505050505050505050505050565b606081600003611a9e57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115611ac85780611ab281612699565b9150611ac19050600a83612700565b9150611aa2565b60008167ffffffffffffffff811115611ae357611ae3612714565b6040519080825280601f01601f191660200182016040528015611b0d576020820181803683370190505b5090505b8415611b9057611b226001836124f5565b9150611b2f600a86612743565b611b3a906030612624565b60f81b818381518110611b4f57611b4f612757565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350611b89600a86612700565b9450611b11565b949350505050565b6000611ba383611d25565b8015611bb45750611bb48383611d89565b9392505050565b6000611c1d826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c65648152508573ffffffffffffffffffffffffffffffffffffffff16611e589092919063ffffffff16565b805190915015610a175780806020019051810190611c3b9190612786565b610a17576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f742073756363656564000000000000000000000000000000000000000000006064820152608401610166565b60405173ffffffffffffffffffffffffffffffffffffffff808516602483015283166044820152606481018290526112579085907f23b872dd00000000000000000000000000000000000000000000000000000000906084016113b8565b6000611d51827f01ffc9a700000000000000000000000000000000000000000000000000000000611d89565b80156112b95750611d82827fffffffff00000000000000000000000000000000000000000000000000000000611d89565b1592915050565b604080517fffffffff000000000000000000000000000000000000000000000000000000008316602480830191909152825180830390910181526044909101909152602080820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01ffc9a700000000000000000000000000000000000000000000000000000000178152825160009392849283928392918391908a617530fa92503d91506000519050828015611e41575060208210155b8015611e4d5750600081115b979650505050505050565b6060611b9084846000858573ffffffffffffffffffffffffffffffffffffffff85163b611ee1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e74726163740000006044820152606401610166565b6000808673ffffffffffffffffffffffffffffffffffffffff168587604051611f0a91906127a8565b60006040518083038185875af1925050503d8060008114611f47576040519150601f19603f3d011682016040523d82523d6000602084013e611f4c565b606091505b5091509150611e4d82828660608315611f66575081611bb4565b825115611f765782518084602001fd5b816040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161016691906122d0565b73ffffffffffffffffffffffffffffffffffffffff81168114611fcc57600080fd5b50565b60008083601f840112611fe157600080fd5b50813567ffffffffffffffff811115611ff957600080fd5b60208301915083602082850101111561201157600080fd5b9250929050565b600080600080600080600060c0888a03121561203357600080fd5b873561203e81611faa565b9650602088013561204e81611faa565b9550604088013561205e81611faa565b9450606088013561206e81611faa565b93506080880135925060a088013567ffffffffffffffff81111561209157600080fd5b61209d8a828b01611fcf565b989b979a50959850939692959293505050565b803563ffffffff811681146120c457600080fd5b919050565b6000806000604084860312156120de57600080fd5b6120e7846120b0565b9250602084013567ffffffffffffffff81111561210357600080fd5b61210f86828701611fcf565b9497909650939450505050565b60008060008060006080868803121561213457600080fd5b853561213f81611faa565b9450602086013561214f81611faa565b935060408601359250606086013567ffffffffffffffff81111561217257600080fd5b61217e88828901611fcf565b969995985093965092949392505050565b6000806000806000608086880312156121a757600080fd5b85356121b281611faa565b9450602086013593506121c7604087016120b0565b9250606086013567ffffffffffffffff81111561217257600080fd5b600080600080600080600060c0888a0312156121fe57600080fd5b873561220981611faa565b9650602088013561221981611faa565b9550604088013561222981611faa565b94506060880135935061223e608089016120b0565b925060a088013567ffffffffffffffff81111561209157600080fd5b60005b8381101561227557818101518382015260200161225d565b838111156112575750506000910152565b6000815180845261229e81602086016020860161225a565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000611bb46020830184612286565b60008060008060008060a087890312156122fc57600080fd5b863561230781611faa565b9550602087013561231781611faa565b94506040870135935061232c606088016120b0565b9250608087013567ffffffffffffffff81111561234857600080fd5b61235489828a01611fcf565b979a9699509497509295939492505050565b6000806040838503121561237957600080fd5b823561238481611faa565b9150602083013561239481611faa565b809150509250929050565b600080600080606085870312156123b557600080fd5b84356123c081611faa565b93506123ce602086016120b0565b9250604085013567ffffffffffffffff8111156123ea57600080fd5b6123f687828801611fcf565b95989497509550505050565b828152604060208201526000611b906040830184612286565b600073ffffffffffffffffffffffffffffffffffffffff80871683528086166020840152508360408301526080606083015261245a6080830184612286565b9695505050505050565b73ffffffffffffffffffffffffffffffffffffffff841681526060602082015260006124936060830185612286565b905063ffffffff83166040830152949350505050565b6000602082840312156124bb57600080fd5b8151611bb481611faa565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015612507576125076124c6565b500390565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b73ffffffffffffffffffffffffffffffffffffffff8516815283602082015260606040820152600061245a60608301848661250c565b8381526040602082015260006125a560408301848661250c565b95945050505050565b600084516125c081846020890161225a565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516125fc816001850160208a0161225a565b6001920191820152835161261781600284016020880161225a565b0160020195945050505050565b60008219821115612637576126376124c6565b500190565b600073ffffffffffffffffffffffffffffffffffffffff808a1683528089166020840152808816604084015280871660608401525084608083015260c060a083015261268c60c08301848661250c565b9998505050505050505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036126ca576126ca6124c6565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b60008261270f5761270f6126d1565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600082612752576127526126d1565b500690565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60006020828403121561279857600080fd5b81518015158114611bb457600080fd5b600082516127ba81846020870161225a565b919091019291505056fea164736f6c634300080f000a",
}

// L2StandardBridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use L2StandardBridgeMetaData.ABI instead.
var L2StandardBridgeABI = L2StandardBridgeMetaData.ABI

// L2StandardBridgeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2StandardBridgeMetaData.Bin instead.
var L2StandardBridgeBin = L2StandardBridgeMetaData.Bin

// DeployL2StandardBridge deploys a new Ethereum contract, binding an instance of L2StandardBridge to it.
func DeployL2StandardBridge(auth *bind.TransactOpts, backend bind.ContractBackend, _otherBridge common.Address) (common.Address, *types.Transaction, *L2StandardBridge, error) {
	parsed, err := L2StandardBridgeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2StandardBridgeBin), backend, _otherBridge)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2StandardBridge{L2StandardBridgeCaller: L2StandardBridgeCaller{contract: contract}, L2StandardBridgeTransactor: L2StandardBridgeTransactor{contract: contract}, L2StandardBridgeFilterer: L2StandardBridgeFilterer{contract: contract}}, nil
}

// L2StandardBridge is an auto generated Go binding around an Ethereum contract.
type L2StandardBridge struct {
	L2StandardBridgeCaller     // Read-only binding to the contract
	L2StandardBridgeTransactor // Write-only binding to the contract
	L2StandardBridgeFilterer   // Log filterer for contract events
}

// L2StandardBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2StandardBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2StandardBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2StandardBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2StandardBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2StandardBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2StandardBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2StandardBridgeSession struct {
	Contract     *L2StandardBridge // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L2StandardBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2StandardBridgeCallerSession struct {
	Contract *L2StandardBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// L2StandardBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2StandardBridgeTransactorSession struct {
	Contract     *L2StandardBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// L2StandardBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2StandardBridgeRaw struct {
	Contract *L2StandardBridge // Generic contract binding to access the raw methods on
}

// L2StandardBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2StandardBridgeCallerRaw struct {
	Contract *L2StandardBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// L2StandardBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2StandardBridgeTransactorRaw struct {
	Contract *L2StandardBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2StandardBridge creates a new instance of L2StandardBridge, bound to a specific deployed contract.
func NewL2StandardBridge(address common.Address, backend bind.ContractBackend) (*L2StandardBridge, error) {
	contract, err := bindL2StandardBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2StandardBridge{L2StandardBridgeCaller: L2StandardBridgeCaller{contract: contract}, L2StandardBridgeTransactor: L2StandardBridgeTransactor{contract: contract}, L2StandardBridgeFilterer: L2StandardBridgeFilterer{contract: contract}}, nil
}

// NewL2StandardBridgeCaller creates a new read-only instance of L2StandardBridge, bound to a specific deployed contract.
func NewL2StandardBridgeCaller(address common.Address, caller bind.ContractCaller) (*L2StandardBridgeCaller, error) {
	contract, err := bindL2StandardBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2StandardBridgeCaller{contract: contract}, nil
}

// NewL2StandardBridgeTransactor creates a new write-only instance of L2StandardBridge, bound to a specific deployed contract.
func NewL2StandardBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*L2StandardBridgeTransactor, error) {
	contract, err := bindL2StandardBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2StandardBridgeTransactor{contract: contract}, nil
}

// NewL2StandardBridgeFilterer creates a new log filterer instance of L2StandardBridge, bound to a specific deployed contract.
func NewL2StandardBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*L2StandardBridgeFilterer, error) {
	contract, err := bindL2StandardBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2StandardBridgeFilterer{contract: contract}, nil
}

// bindL2StandardBridge binds a generic wrapper to an already deployed contract.
func bindL2StandardBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2StandardBridgeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2StandardBridge *L2StandardBridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2StandardBridge.Contract.L2StandardBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2StandardBridge *L2StandardBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.L2StandardBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2StandardBridge *L2StandardBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.L2StandardBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2StandardBridge *L2StandardBridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2StandardBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2StandardBridge *L2StandardBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2StandardBridge *L2StandardBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.contract.Transact(opts, method, params...)
}

// Deposits is a free data retrieval call binding the contract method 0x8f601f66.
//
// Solidity: function deposits(address , address ) view returns(uint256)
func (_L2StandardBridge *L2StandardBridgeCaller) Deposits(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _L2StandardBridge.contract.Call(opts, &out, "deposits", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Deposits is a free data retrieval call binding the contract method 0x8f601f66.
//
// Solidity: function deposits(address , address ) view returns(uint256)
func (_L2StandardBridge *L2StandardBridgeSession) Deposits(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _L2StandardBridge.Contract.Deposits(&_L2StandardBridge.CallOpts, arg0, arg1)
}

// Deposits is a free data retrieval call binding the contract method 0x8f601f66.
//
// Solidity: function deposits(address , address ) view returns(uint256)
func (_L2StandardBridge *L2StandardBridgeCallerSession) Deposits(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _L2StandardBridge.Contract.Deposits(&_L2StandardBridge.CallOpts, arg0, arg1)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2StandardBridge *L2StandardBridgeCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2StandardBridge.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2StandardBridge *L2StandardBridgeSession) Messenger() (common.Address, error) {
	return _L2StandardBridge.Contract.Messenger(&_L2StandardBridge.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2StandardBridge *L2StandardBridgeCallerSession) Messenger() (common.Address, error) {
	return _L2StandardBridge.Contract.Messenger(&_L2StandardBridge.CallOpts)
}

// OtherBridge is a free data retrieval call binding the contract method 0xc89701a2.
//
// Solidity: function otherBridge() view returns(address)
func (_L2StandardBridge *L2StandardBridgeCaller) OtherBridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2StandardBridge.contract.Call(opts, &out, "otherBridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OtherBridge is a free data retrieval call binding the contract method 0xc89701a2.
//
// Solidity: function otherBridge() view returns(address)
func (_L2StandardBridge *L2StandardBridgeSession) OtherBridge() (common.Address, error) {
	return _L2StandardBridge.Contract.OtherBridge(&_L2StandardBridge.CallOpts)
}

// OtherBridge is a free data retrieval call binding the contract method 0xc89701a2.
//
// Solidity: function otherBridge() view returns(address)
func (_L2StandardBridge *L2StandardBridgeCallerSession) OtherBridge() (common.Address, error) {
	return _L2StandardBridge.Contract.OtherBridge(&_L2StandardBridge.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2StandardBridge *L2StandardBridgeCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L2StandardBridge.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2StandardBridge *L2StandardBridgeSession) Version() (string, error) {
	return _L2StandardBridge.Contract.Version(&_L2StandardBridge.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2StandardBridge *L2StandardBridgeCallerSession) Version() (string, error) {
	return _L2StandardBridge.Contract.Version(&_L2StandardBridge.CallOpts)
}

// BridgeERC20 is a paid mutator transaction binding the contract method 0x87087623.
//
// Solidity: function bridgeERC20(address _localToken, address _remoteToken, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2StandardBridge *L2StandardBridgeTransactor) BridgeERC20(opts *bind.TransactOpts, _localToken common.Address, _remoteToken common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.contract.Transact(opts, "bridgeERC20", _localToken, _remoteToken, _amount, _minGasLimit, _extraData)
}

// BridgeERC20 is a paid mutator transaction binding the contract method 0x87087623.
//
// Solidity: function bridgeERC20(address _localToken, address _remoteToken, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2StandardBridge *L2StandardBridgeSession) BridgeERC20(_localToken common.Address, _remoteToken common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.BridgeERC20(&_L2StandardBridge.TransactOpts, _localToken, _remoteToken, _amount, _minGasLimit, _extraData)
}

// BridgeERC20 is a paid mutator transaction binding the contract method 0x87087623.
//
// Solidity: function bridgeERC20(address _localToken, address _remoteToken, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2StandardBridge *L2StandardBridgeTransactorSession) BridgeERC20(_localToken common.Address, _remoteToken common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.BridgeERC20(&_L2StandardBridge.TransactOpts, _localToken, _remoteToken, _amount, _minGasLimit, _extraData)
}

// BridgeERC20To is a paid mutator transaction binding the contract method 0x540abf73.
//
// Solidity: function bridgeERC20To(address _localToken, address _remoteToken, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2StandardBridge *L2StandardBridgeTransactor) BridgeERC20To(opts *bind.TransactOpts, _localToken common.Address, _remoteToken common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.contract.Transact(opts, "bridgeERC20To", _localToken, _remoteToken, _to, _amount, _minGasLimit, _extraData)
}

// BridgeERC20To is a paid mutator transaction binding the contract method 0x540abf73.
//
// Solidity: function bridgeERC20To(address _localToken, address _remoteToken, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2StandardBridge *L2StandardBridgeSession) BridgeERC20To(_localToken common.Address, _remoteToken common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.BridgeERC20To(&_L2StandardBridge.TransactOpts, _localToken, _remoteToken, _to, _amount, _minGasLimit, _extraData)
}

// BridgeERC20To is a paid mutator transaction binding the contract method 0x540abf73.
//
// Solidity: function bridgeERC20To(address _localToken, address _remoteToken, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2StandardBridge *L2StandardBridgeTransactorSession) BridgeERC20To(_localToken common.Address, _remoteToken common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.BridgeERC20To(&_L2StandardBridge.TransactOpts, _localToken, _remoteToken, _to, _amount, _minGasLimit, _extraData)
}

// BridgeETH is a paid mutator transaction binding the contract method 0x09fc8843.
//
// Solidity: function bridgeETH(uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactor) BridgeETH(opts *bind.TransactOpts, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.contract.Transact(opts, "bridgeETH", _minGasLimit, _extraData)
}

// BridgeETH is a paid mutator transaction binding the contract method 0x09fc8843.
//
// Solidity: function bridgeETH(uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeSession) BridgeETH(_minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.BridgeETH(&_L2StandardBridge.TransactOpts, _minGasLimit, _extraData)
}

// BridgeETH is a paid mutator transaction binding the contract method 0x09fc8843.
//
// Solidity: function bridgeETH(uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactorSession) BridgeETH(_minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.BridgeETH(&_L2StandardBridge.TransactOpts, _minGasLimit, _extraData)
}

// BridgeETHTo is a paid mutator transaction binding the contract method 0xe11013dd.
//
// Solidity: function bridgeETHTo(address _to, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactor) BridgeETHTo(opts *bind.TransactOpts, _to common.Address, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.contract.Transact(opts, "bridgeETHTo", _to, _minGasLimit, _extraData)
}

// BridgeETHTo is a paid mutator transaction binding the contract method 0xe11013dd.
//
// Solidity: function bridgeETHTo(address _to, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeSession) BridgeETHTo(_to common.Address, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.BridgeETHTo(&_L2StandardBridge.TransactOpts, _to, _minGasLimit, _extraData)
}

// BridgeETHTo is a paid mutator transaction binding the contract method 0xe11013dd.
//
// Solidity: function bridgeETHTo(address _to, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactorSession) BridgeETHTo(_to common.Address, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.BridgeETHTo(&_L2StandardBridge.TransactOpts, _to, _minGasLimit, _extraData)
}

// FinalizeBridgeERC20 is a paid mutator transaction binding the contract method 0x0166a07a.
//
// Solidity: function finalizeBridgeERC20(address _localToken, address _remoteToken, address _from, address _to, uint256 _amount, bytes _extraData) returns()
func (_L2StandardBridge *L2StandardBridgeTransactor) FinalizeBridgeERC20(opts *bind.TransactOpts, _localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.contract.Transact(opts, "finalizeBridgeERC20", _localToken, _remoteToken, _from, _to, _amount, _extraData)
}

// FinalizeBridgeERC20 is a paid mutator transaction binding the contract method 0x0166a07a.
//
// Solidity: function finalizeBridgeERC20(address _localToken, address _remoteToken, address _from, address _to, uint256 _amount, bytes _extraData) returns()
func (_L2StandardBridge *L2StandardBridgeSession) FinalizeBridgeERC20(_localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.FinalizeBridgeERC20(&_L2StandardBridge.TransactOpts, _localToken, _remoteToken, _from, _to, _amount, _extraData)
}

// FinalizeBridgeERC20 is a paid mutator transaction binding the contract method 0x0166a07a.
//
// Solidity: function finalizeBridgeERC20(address _localToken, address _remoteToken, address _from, address _to, uint256 _amount, bytes _extraData) returns()
func (_L2StandardBridge *L2StandardBridgeTransactorSession) FinalizeBridgeERC20(_localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.FinalizeBridgeERC20(&_L2StandardBridge.TransactOpts, _localToken, _remoteToken, _from, _to, _amount, _extraData)
}

// FinalizeBridgeETH is a paid mutator transaction binding the contract method 0x1635f5fd.
//
// Solidity: function finalizeBridgeETH(address _from, address _to, uint256 _amount, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactor) FinalizeBridgeETH(opts *bind.TransactOpts, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.contract.Transact(opts, "finalizeBridgeETH", _from, _to, _amount, _extraData)
}

// FinalizeBridgeETH is a paid mutator transaction binding the contract method 0x1635f5fd.
//
// Solidity: function finalizeBridgeETH(address _from, address _to, uint256 _amount, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeSession) FinalizeBridgeETH(_from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.FinalizeBridgeETH(&_L2StandardBridge.TransactOpts, _from, _to, _amount, _extraData)
}

// FinalizeBridgeETH is a paid mutator transaction binding the contract method 0x1635f5fd.
//
// Solidity: function finalizeBridgeETH(address _from, address _to, uint256 _amount, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactorSession) FinalizeBridgeETH(_from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.FinalizeBridgeETH(&_L2StandardBridge.TransactOpts, _from, _to, _amount, _extraData)
}

// FinalizeDeposit is a paid mutator transaction binding the contract method 0x662a633a.
//
// Solidity: function finalizeDeposit(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactor) FinalizeDeposit(opts *bind.TransactOpts, _l1Token common.Address, _l2Token common.Address, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.contract.Transact(opts, "finalizeDeposit", _l1Token, _l2Token, _from, _to, _amount, _extraData)
}

// FinalizeDeposit is a paid mutator transaction binding the contract method 0x662a633a.
//
// Solidity: function finalizeDeposit(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeSession) FinalizeDeposit(_l1Token common.Address, _l2Token common.Address, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.FinalizeDeposit(&_L2StandardBridge.TransactOpts, _l1Token, _l2Token, _from, _to, _amount, _extraData)
}

// FinalizeDeposit is a paid mutator transaction binding the contract method 0x662a633a.
//
// Solidity: function finalizeDeposit(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactorSession) FinalizeDeposit(_l1Token common.Address, _l2Token common.Address, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.FinalizeDeposit(&_L2StandardBridge.TransactOpts, _l1Token, _l2Token, _from, _to, _amount, _extraData)
}

// Withdraw is a paid mutator transaction binding the contract method 0x32b7006d.
//
// Solidity: function withdraw(address _l2Token, uint256 _amount, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactor) Withdraw(opts *bind.TransactOpts, _l2Token common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.contract.Transact(opts, "withdraw", _l2Token, _amount, _minGasLimit, _extraData)
}

// Withdraw is a paid mutator transaction binding the contract method 0x32b7006d.
//
// Solidity: function withdraw(address _l2Token, uint256 _amount, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeSession) Withdraw(_l2Token common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.Withdraw(&_L2StandardBridge.TransactOpts, _l2Token, _amount, _minGasLimit, _extraData)
}

// Withdraw is a paid mutator transaction binding the contract method 0x32b7006d.
//
// Solidity: function withdraw(address _l2Token, uint256 _amount, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactorSession) Withdraw(_l2Token common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.Withdraw(&_L2StandardBridge.TransactOpts, _l2Token, _amount, _minGasLimit, _extraData)
}

// WithdrawTo is a paid mutator transaction binding the contract method 0xa3a79548.
//
// Solidity: function withdrawTo(address _l2Token, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactor) WithdrawTo(opts *bind.TransactOpts, _l2Token common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.contract.Transact(opts, "withdrawTo", _l2Token, _to, _amount, _minGasLimit, _extraData)
}

// WithdrawTo is a paid mutator transaction binding the contract method 0xa3a79548.
//
// Solidity: function withdrawTo(address _l2Token, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeSession) WithdrawTo(_l2Token common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.WithdrawTo(&_L2StandardBridge.TransactOpts, _l2Token, _to, _amount, _minGasLimit, _extraData)
}

// WithdrawTo is a paid mutator transaction binding the contract method 0xa3a79548.
//
// Solidity: function withdrawTo(address _l2Token, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactorSession) WithdrawTo(_l2Token common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2StandardBridge.Contract.WithdrawTo(&_L2StandardBridge.TransactOpts, _l2Token, _to, _amount, _minGasLimit, _extraData)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2StandardBridge.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L2StandardBridge *L2StandardBridgeSession) Receive() (*types.Transaction, error) {
	return _L2StandardBridge.Contract.Receive(&_L2StandardBridge.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L2StandardBridge *L2StandardBridgeTransactorSession) Receive() (*types.Transaction, error) {
	return _L2StandardBridge.Contract.Receive(&_L2StandardBridge.TransactOpts)
}

// L2StandardBridgeDepositFinalizedIterator is returned from FilterDepositFinalized and is used to iterate over the raw logs and unpacked data for DepositFinalized events raised by the L2StandardBridge contract.
type L2StandardBridgeDepositFinalizedIterator struct {
	Event *L2StandardBridgeDepositFinalized // Event containing the contract specifics and raw log

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
func (it *L2StandardBridgeDepositFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2StandardBridgeDepositFinalized)
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
		it.Event = new(L2StandardBridgeDepositFinalized)
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
func (it *L2StandardBridgeDepositFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2StandardBridgeDepositFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2StandardBridgeDepositFinalized represents a DepositFinalized event raised by the L2StandardBridge contract.
type L2StandardBridgeDepositFinalized struct {
	L1Token   common.Address
	L2Token   common.Address
	From      common.Address
	To        common.Address
	Amount    *big.Int
	ExtraData []byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDepositFinalized is a free log retrieval operation binding the contract event 0xb0444523268717a02698be47d0803aa7468c00acbed2f8bd93a0459cde61dd89.
//
// Solidity: event DepositFinalized(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) FilterDepositFinalized(opts *bind.FilterOpts, l1Token []common.Address, l2Token []common.Address, from []common.Address) (*L2StandardBridgeDepositFinalizedIterator, error) {

	var l1TokenRule []interface{}
	for _, l1TokenItem := range l1Token {
		l1TokenRule = append(l1TokenRule, l1TokenItem)
	}
	var l2TokenRule []interface{}
	for _, l2TokenItem := range l2Token {
		l2TokenRule = append(l2TokenRule, l2TokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L2StandardBridge.contract.FilterLogs(opts, "DepositFinalized", l1TokenRule, l2TokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return &L2StandardBridgeDepositFinalizedIterator{contract: _L2StandardBridge.contract, event: "DepositFinalized", logs: logs, sub: sub}, nil
}

// WatchDepositFinalized is a free log subscription operation binding the contract event 0xb0444523268717a02698be47d0803aa7468c00acbed2f8bd93a0459cde61dd89.
//
// Solidity: event DepositFinalized(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) WatchDepositFinalized(opts *bind.WatchOpts, sink chan<- *L2StandardBridgeDepositFinalized, l1Token []common.Address, l2Token []common.Address, from []common.Address) (event.Subscription, error) {

	var l1TokenRule []interface{}
	for _, l1TokenItem := range l1Token {
		l1TokenRule = append(l1TokenRule, l1TokenItem)
	}
	var l2TokenRule []interface{}
	for _, l2TokenItem := range l2Token {
		l2TokenRule = append(l2TokenRule, l2TokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L2StandardBridge.contract.WatchLogs(opts, "DepositFinalized", l1TokenRule, l2TokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2StandardBridgeDepositFinalized)
				if err := _L2StandardBridge.contract.UnpackLog(event, "DepositFinalized", log); err != nil {
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

// ParseDepositFinalized is a log parse operation binding the contract event 0xb0444523268717a02698be47d0803aa7468c00acbed2f8bd93a0459cde61dd89.
//
// Solidity: event DepositFinalized(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) ParseDepositFinalized(log types.Log) (*L2StandardBridgeDepositFinalized, error) {
	event := new(L2StandardBridgeDepositFinalized)
	if err := _L2StandardBridge.contract.UnpackLog(event, "DepositFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2StandardBridgeERC20BridgeFinalizedIterator is returned from FilterERC20BridgeFinalized and is used to iterate over the raw logs and unpacked data for ERC20BridgeFinalized events raised by the L2StandardBridge contract.
type L2StandardBridgeERC20BridgeFinalizedIterator struct {
	Event *L2StandardBridgeERC20BridgeFinalized // Event containing the contract specifics and raw log

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
func (it *L2StandardBridgeERC20BridgeFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2StandardBridgeERC20BridgeFinalized)
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
		it.Event = new(L2StandardBridgeERC20BridgeFinalized)
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
func (it *L2StandardBridgeERC20BridgeFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2StandardBridgeERC20BridgeFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2StandardBridgeERC20BridgeFinalized represents a ERC20BridgeFinalized event raised by the L2StandardBridge contract.
type L2StandardBridgeERC20BridgeFinalized struct {
	LocalToken  common.Address
	RemoteToken common.Address
	From        common.Address
	To          common.Address
	Amount      *big.Int
	ExtraData   []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterERC20BridgeFinalized is a free log retrieval operation binding the contract event 0xd59c65b35445225835c83f50b6ede06a7be047d22e357073e250d9af537518cd.
//
// Solidity: event ERC20BridgeFinalized(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) FilterERC20BridgeFinalized(opts *bind.FilterOpts, localToken []common.Address, remoteToken []common.Address, from []common.Address) (*L2StandardBridgeERC20BridgeFinalizedIterator, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L2StandardBridge.contract.FilterLogs(opts, "ERC20BridgeFinalized", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return &L2StandardBridgeERC20BridgeFinalizedIterator{contract: _L2StandardBridge.contract, event: "ERC20BridgeFinalized", logs: logs, sub: sub}, nil
}

// WatchERC20BridgeFinalized is a free log subscription operation binding the contract event 0xd59c65b35445225835c83f50b6ede06a7be047d22e357073e250d9af537518cd.
//
// Solidity: event ERC20BridgeFinalized(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) WatchERC20BridgeFinalized(opts *bind.WatchOpts, sink chan<- *L2StandardBridgeERC20BridgeFinalized, localToken []common.Address, remoteToken []common.Address, from []common.Address) (event.Subscription, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L2StandardBridge.contract.WatchLogs(opts, "ERC20BridgeFinalized", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2StandardBridgeERC20BridgeFinalized)
				if err := _L2StandardBridge.contract.UnpackLog(event, "ERC20BridgeFinalized", log); err != nil {
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

// ParseERC20BridgeFinalized is a log parse operation binding the contract event 0xd59c65b35445225835c83f50b6ede06a7be047d22e357073e250d9af537518cd.
//
// Solidity: event ERC20BridgeFinalized(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) ParseERC20BridgeFinalized(log types.Log) (*L2StandardBridgeERC20BridgeFinalized, error) {
	event := new(L2StandardBridgeERC20BridgeFinalized)
	if err := _L2StandardBridge.contract.UnpackLog(event, "ERC20BridgeFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2StandardBridgeERC20BridgeInitiatedIterator is returned from FilterERC20BridgeInitiated and is used to iterate over the raw logs and unpacked data for ERC20BridgeInitiated events raised by the L2StandardBridge contract.
type L2StandardBridgeERC20BridgeInitiatedIterator struct {
	Event *L2StandardBridgeERC20BridgeInitiated // Event containing the contract specifics and raw log

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
func (it *L2StandardBridgeERC20BridgeInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2StandardBridgeERC20BridgeInitiated)
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
		it.Event = new(L2StandardBridgeERC20BridgeInitiated)
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
func (it *L2StandardBridgeERC20BridgeInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2StandardBridgeERC20BridgeInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2StandardBridgeERC20BridgeInitiated represents a ERC20BridgeInitiated event raised by the L2StandardBridge contract.
type L2StandardBridgeERC20BridgeInitiated struct {
	LocalToken  common.Address
	RemoteToken common.Address
	From        common.Address
	To          common.Address
	Amount      *big.Int
	ExtraData   []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterERC20BridgeInitiated is a free log retrieval operation binding the contract event 0x7ff126db8024424bbfd9826e8ab82ff59136289ea440b04b39a0df1b03b9cabf.
//
// Solidity: event ERC20BridgeInitiated(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) FilterERC20BridgeInitiated(opts *bind.FilterOpts, localToken []common.Address, remoteToken []common.Address, from []common.Address) (*L2StandardBridgeERC20BridgeInitiatedIterator, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L2StandardBridge.contract.FilterLogs(opts, "ERC20BridgeInitiated", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return &L2StandardBridgeERC20BridgeInitiatedIterator{contract: _L2StandardBridge.contract, event: "ERC20BridgeInitiated", logs: logs, sub: sub}, nil
}

// WatchERC20BridgeInitiated is a free log subscription operation binding the contract event 0x7ff126db8024424bbfd9826e8ab82ff59136289ea440b04b39a0df1b03b9cabf.
//
// Solidity: event ERC20BridgeInitiated(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) WatchERC20BridgeInitiated(opts *bind.WatchOpts, sink chan<- *L2StandardBridgeERC20BridgeInitiated, localToken []common.Address, remoteToken []common.Address, from []common.Address) (event.Subscription, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L2StandardBridge.contract.WatchLogs(opts, "ERC20BridgeInitiated", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2StandardBridgeERC20BridgeInitiated)
				if err := _L2StandardBridge.contract.UnpackLog(event, "ERC20BridgeInitiated", log); err != nil {
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

// ParseERC20BridgeInitiated is a log parse operation binding the contract event 0x7ff126db8024424bbfd9826e8ab82ff59136289ea440b04b39a0df1b03b9cabf.
//
// Solidity: event ERC20BridgeInitiated(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) ParseERC20BridgeInitiated(log types.Log) (*L2StandardBridgeERC20BridgeInitiated, error) {
	event := new(L2StandardBridgeERC20BridgeInitiated)
	if err := _L2StandardBridge.contract.UnpackLog(event, "ERC20BridgeInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2StandardBridgeETHBridgeFinalizedIterator is returned from FilterETHBridgeFinalized and is used to iterate over the raw logs and unpacked data for ETHBridgeFinalized events raised by the L2StandardBridge contract.
type L2StandardBridgeETHBridgeFinalizedIterator struct {
	Event *L2StandardBridgeETHBridgeFinalized // Event containing the contract specifics and raw log

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
func (it *L2StandardBridgeETHBridgeFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2StandardBridgeETHBridgeFinalized)
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
		it.Event = new(L2StandardBridgeETHBridgeFinalized)
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
func (it *L2StandardBridgeETHBridgeFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2StandardBridgeETHBridgeFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2StandardBridgeETHBridgeFinalized represents a ETHBridgeFinalized event raised by the L2StandardBridge contract.
type L2StandardBridgeETHBridgeFinalized struct {
	From      common.Address
	To        common.Address
	Amount    *big.Int
	ExtraData []byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterETHBridgeFinalized is a free log retrieval operation binding the contract event 0x31b2166ff604fc5672ea5df08a78081d2bc6d746cadce880747f3643d819e83d.
//
// Solidity: event ETHBridgeFinalized(address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) FilterETHBridgeFinalized(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*L2StandardBridgeETHBridgeFinalizedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _L2StandardBridge.contract.FilterLogs(opts, "ETHBridgeFinalized", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &L2StandardBridgeETHBridgeFinalizedIterator{contract: _L2StandardBridge.contract, event: "ETHBridgeFinalized", logs: logs, sub: sub}, nil
}

// WatchETHBridgeFinalized is a free log subscription operation binding the contract event 0x31b2166ff604fc5672ea5df08a78081d2bc6d746cadce880747f3643d819e83d.
//
// Solidity: event ETHBridgeFinalized(address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) WatchETHBridgeFinalized(opts *bind.WatchOpts, sink chan<- *L2StandardBridgeETHBridgeFinalized, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _L2StandardBridge.contract.WatchLogs(opts, "ETHBridgeFinalized", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2StandardBridgeETHBridgeFinalized)
				if err := _L2StandardBridge.contract.UnpackLog(event, "ETHBridgeFinalized", log); err != nil {
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

// ParseETHBridgeFinalized is a log parse operation binding the contract event 0x31b2166ff604fc5672ea5df08a78081d2bc6d746cadce880747f3643d819e83d.
//
// Solidity: event ETHBridgeFinalized(address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) ParseETHBridgeFinalized(log types.Log) (*L2StandardBridgeETHBridgeFinalized, error) {
	event := new(L2StandardBridgeETHBridgeFinalized)
	if err := _L2StandardBridge.contract.UnpackLog(event, "ETHBridgeFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2StandardBridgeETHBridgeInitiatedIterator is returned from FilterETHBridgeInitiated and is used to iterate over the raw logs and unpacked data for ETHBridgeInitiated events raised by the L2StandardBridge contract.
type L2StandardBridgeETHBridgeInitiatedIterator struct {
	Event *L2StandardBridgeETHBridgeInitiated // Event containing the contract specifics and raw log

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
func (it *L2StandardBridgeETHBridgeInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2StandardBridgeETHBridgeInitiated)
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
		it.Event = new(L2StandardBridgeETHBridgeInitiated)
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
func (it *L2StandardBridgeETHBridgeInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2StandardBridgeETHBridgeInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2StandardBridgeETHBridgeInitiated represents a ETHBridgeInitiated event raised by the L2StandardBridge contract.
type L2StandardBridgeETHBridgeInitiated struct {
	From      common.Address
	To        common.Address
	Amount    *big.Int
	ExtraData []byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterETHBridgeInitiated is a free log retrieval operation binding the contract event 0x2849b43074093a05396b6f2a937dee8565b15a48a7b3d4bffb732a5017380af5.
//
// Solidity: event ETHBridgeInitiated(address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) FilterETHBridgeInitiated(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*L2StandardBridgeETHBridgeInitiatedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _L2StandardBridge.contract.FilterLogs(opts, "ETHBridgeInitiated", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &L2StandardBridgeETHBridgeInitiatedIterator{contract: _L2StandardBridge.contract, event: "ETHBridgeInitiated", logs: logs, sub: sub}, nil
}

// WatchETHBridgeInitiated is a free log subscription operation binding the contract event 0x2849b43074093a05396b6f2a937dee8565b15a48a7b3d4bffb732a5017380af5.
//
// Solidity: event ETHBridgeInitiated(address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) WatchETHBridgeInitiated(opts *bind.WatchOpts, sink chan<- *L2StandardBridgeETHBridgeInitiated, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _L2StandardBridge.contract.WatchLogs(opts, "ETHBridgeInitiated", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2StandardBridgeETHBridgeInitiated)
				if err := _L2StandardBridge.contract.UnpackLog(event, "ETHBridgeInitiated", log); err != nil {
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

// ParseETHBridgeInitiated is a log parse operation binding the contract event 0x2849b43074093a05396b6f2a937dee8565b15a48a7b3d4bffb732a5017380af5.
//
// Solidity: event ETHBridgeInitiated(address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) ParseETHBridgeInitiated(log types.Log) (*L2StandardBridgeETHBridgeInitiated, error) {
	event := new(L2StandardBridgeETHBridgeInitiated)
	if err := _L2StandardBridge.contract.UnpackLog(event, "ETHBridgeInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2StandardBridgeWithdrawalInitiatedIterator is returned from FilterWithdrawalInitiated and is used to iterate over the raw logs and unpacked data for WithdrawalInitiated events raised by the L2StandardBridge contract.
type L2StandardBridgeWithdrawalInitiatedIterator struct {
	Event *L2StandardBridgeWithdrawalInitiated // Event containing the contract specifics and raw log

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
func (it *L2StandardBridgeWithdrawalInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2StandardBridgeWithdrawalInitiated)
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
		it.Event = new(L2StandardBridgeWithdrawalInitiated)
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
func (it *L2StandardBridgeWithdrawalInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2StandardBridgeWithdrawalInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2StandardBridgeWithdrawalInitiated represents a WithdrawalInitiated event raised by the L2StandardBridge contract.
type L2StandardBridgeWithdrawalInitiated struct {
	L1Token   common.Address
	L2Token   common.Address
	From      common.Address
	To        common.Address
	Amount    *big.Int
	ExtraData []byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalInitiated is a free log retrieval operation binding the contract event 0x73d170910aba9e6d50b102db522b1dbcd796216f5128b445aa2135272886497e.
//
// Solidity: event WithdrawalInitiated(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) FilterWithdrawalInitiated(opts *bind.FilterOpts, l1Token []common.Address, l2Token []common.Address, from []common.Address) (*L2StandardBridgeWithdrawalInitiatedIterator, error) {

	var l1TokenRule []interface{}
	for _, l1TokenItem := range l1Token {
		l1TokenRule = append(l1TokenRule, l1TokenItem)
	}
	var l2TokenRule []interface{}
	for _, l2TokenItem := range l2Token {
		l2TokenRule = append(l2TokenRule, l2TokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L2StandardBridge.contract.FilterLogs(opts, "WithdrawalInitiated", l1TokenRule, l2TokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return &L2StandardBridgeWithdrawalInitiatedIterator{contract: _L2StandardBridge.contract, event: "WithdrawalInitiated", logs: logs, sub: sub}, nil
}

// WatchWithdrawalInitiated is a free log subscription operation binding the contract event 0x73d170910aba9e6d50b102db522b1dbcd796216f5128b445aa2135272886497e.
//
// Solidity: event WithdrawalInitiated(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) WatchWithdrawalInitiated(opts *bind.WatchOpts, sink chan<- *L2StandardBridgeWithdrawalInitiated, l1Token []common.Address, l2Token []common.Address, from []common.Address) (event.Subscription, error) {

	var l1TokenRule []interface{}
	for _, l1TokenItem := range l1Token {
		l1TokenRule = append(l1TokenRule, l1TokenItem)
	}
	var l2TokenRule []interface{}
	for _, l2TokenItem := range l2Token {
		l2TokenRule = append(l2TokenRule, l2TokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L2StandardBridge.contract.WatchLogs(opts, "WithdrawalInitiated", l1TokenRule, l2TokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2StandardBridgeWithdrawalInitiated)
				if err := _L2StandardBridge.contract.UnpackLog(event, "WithdrawalInitiated", log); err != nil {
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

// ParseWithdrawalInitiated is a log parse operation binding the contract event 0x73d170910aba9e6d50b102db522b1dbcd796216f5128b445aa2135272886497e.
//
// Solidity: event WithdrawalInitiated(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes extraData)
func (_L2StandardBridge *L2StandardBridgeFilterer) ParseWithdrawalInitiated(log types.Log) (*L2StandardBridgeWithdrawalInitiated, error) {
	event := new(L2StandardBridgeWithdrawalInitiated)
	if err := _L2StandardBridge.contract.UnpackLog(event, "WithdrawalInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
