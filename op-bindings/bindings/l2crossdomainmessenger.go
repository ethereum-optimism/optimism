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

// L2CrossDomainMessengerMetaData contains all meta data concerning the L2CrossDomainMessenger contract.
var L2CrossDomainMessengerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1CrossDomainMessenger\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"FailedRelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"RelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"}],\"name\":\"SentMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"SentMessageExtension1\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MESSAGE_VERSION\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_GAS_CALLDATA_OVERHEAD\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_GAS_CONSTANT_OVERHEAD\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"}],\"name\":\"baseGas\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1CrossDomainMessenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageNonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"otherMessenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"receivedMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_minGasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"name\":\"relayMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"successfulMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"xDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101006040523480156200001257600080fd5b50604051620027b8380380620027b8833981016040819052620000359162000442565b6001600160a01b038116608052600060a081905260c052600160e0526200005b62000062565b5062000474565b600054600160a81b900460ff16158080156200008b57506000546001600160a01b90910460ff16105b80620000c25750620000a830620001af60201b620012ad1760201c565b158015620000c25750600054600160a01b900460ff166001145b6200012b5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805460ff60a01b1916600160a01b179055801562000159576000805460ff60a81b1916600160a81b1790555b62000163620001be565b8015620001ac576000805460ff60a81b19169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b50565b6001600160a01b03163b151590565b600054600160a81b900460ff166200021c5760405162461bcd60e51b815260206004820152602b60248201526000805160206200279883398151915260448201526a6e697469616c697a696e6760a81b606482015260840162000122565b60cc80546001600160a01b03191661dead1790556200023a6200025a565b62000244620002b8565b6200024e62000321565b620002586200038b565b565b600054600160a81b900460ff16620002585760405162461bcd60e51b815260206004820152602b60248201526000805160206200279883398151915260448201526a6e697469616c697a696e6760a81b606482015260840162000122565b600054600160a81b900460ff16620003165760405162461bcd60e51b815260206004820152602b60248201526000805160206200279883398151915260448201526a6e697469616c697a696e6760a81b606482015260840162000122565b6200025833620003f0565b600054600160a81b900460ff166200037f5760405162461bcd60e51b815260206004820152602b60248201526000805160206200279883398151915260448201526a6e697469616c697a696e6760a81b606482015260840162000122565b6065805460ff19169055565b600054600160a81b900460ff16620003e95760405162461bcd60e51b815260206004820152602b60248201526000805160206200279883398151915260448201526a6e697469616c697a696e6760a81b606482015260840162000122565b6001609755565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b6000602082840312156200045557600080fd5b81516001600160a01b03811681146200046d57600080fd5b9392505050565b60805160a05160c05160e0516122d5620004c3600039600061077c015260006107530152600061072a015260008181610337015281816103d0015281816104ac0152610ccf01526122d56000f3fe6080604052600436106101755760003560e01c80638129fc1c116100cb578063b28ade251161007f578063ecc7042811610059578063ecc70428146103f2578063f2fde38b14610457578063f69f81511461047757600080fd5b8063b28ade251461038b578063d764ad0b146103ab578063db505d80146103be57600080fd5b80638da5cb5b116100b05780638da5cb5b146102fd578063a711986914610328578063b1b1b2091461035b57600080fd5b80638129fc1c146102d35780638456cb59146102e857600080fd5b80633f827a5a1161012d5780636e296e45116101075780636e296e451461026d578063715018a6146102a75780637dea7cc3146102bc57600080fd5b80633f827a5a146101ff57806354fd4d50146102275780635c975abb1461024957600080fd5b80632828d7e81161015e5780632828d7e8146101bf5780633dbb202b146101d55780633f4ba83a146101ea57600080fd5b8063028f85f71461017a5780630c568498146101a9575b600080fd5b34801561018657600080fd5b5061018f601081565b60405163ffffffff90911681526020015b60405180910390f35b3480156101b557600080fd5b5061018f6103e881565b3480156101cb57600080fd5b5061018f6103f881565b6101e86101e3366004611cc9565b6104a7565b005b3480156101f657600080fd5b506101e8610711565b34801561020b57600080fd5b50610214600181565b60405161ffff90911681526020016101a0565b34801561023357600080fd5b5061023c610723565b6040516101a09190611da8565b34801561025557600080fd5b5060655460ff165b60405190151581526020016101a0565b34801561027957600080fd5b506102826107c6565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101a0565b3480156102b357600080fd5b506101e86108b2565b3480156102c857600080fd5b5061018f62030d4081565b3480156102df57600080fd5b506101e86108c4565b3480156102f457600080fd5b506101e8610ac1565b34801561030957600080fd5b5060335473ffffffffffffffffffffffffffffffffffffffff16610282565b34801561033457600080fd5b507f0000000000000000000000000000000000000000000000000000000000000000610282565b34801561036757600080fd5b5061025d610376366004611dc2565b60cb6020526000908152604090205460ff1681565b34801561039757600080fd5b5061018f6103a6366004611ddb565b610ad1565b6101e86103b9366004611e2f565b610b17565b3480156103ca57600080fd5b506102827f000000000000000000000000000000000000000000000000000000000000000081565b3480156103fe57600080fd5b5061044960cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b6040519081526020016101a0565b34801561046357600080fd5b506101e8610472366004611eb1565b6111f9565b34801561048357600080fd5b5061025d610492366004611dc2565b60ce6020526000908152604090205460ff1681565b6105e67f00000000000000000000000000000000000000000000000000000000000000006104d6858585610ad1565b63ffffffff16347fd764ad0b0000000000000000000000000000000000000000000000000000000061054860cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b338a34898c8c6040516024016105649796959493929190611f15565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff00000000000000000000000000000000000000000000000000000000909316929092179091526112c9565b8373ffffffffffffffffffffffffffffffffffffffff167fcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a33858561066b60cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b8660405161067d959493929190611f74565b60405180910390a260405134815233907f8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d5469060200160405180910390a2505060cd80547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808216600101167fffff0000000000000000000000000000000000000000000000000000000000009091161790555050565b610719611357565b6107216113d8565b565b606061074e7f0000000000000000000000000000000000000000000000000000000000000000611455565b6107777f0000000000000000000000000000000000000000000000000000000000000000611455565b6107a07f0000000000000000000000000000000000000000000000000000000000000000611455565b6040516020016107b293929190611fc2565b604051602081830303815290604052905090565b60cc5460009073ffffffffffffffffffffffffffffffffffffffff167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff215301610895576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603560248201527f43726f7373446f6d61696e4d657373656e6765723a2078446f6d61696e4d657360448201527f7361676553656e646572206973206e6f7420736574000000000000000000000060648201526084015b60405180910390fd5b5060cc5473ffffffffffffffffffffffffffffffffffffffff1690565b6108ba611357565b610721600061158a565b6000547501000000000000000000000000000000000000000000900460ff161580801561090f575060005460017401000000000000000000000000000000000000000090910460ff16105b806109415750303b158015610941575060005474010000000000000000000000000000000000000000900460ff166001145b6109cd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a6564000000000000000000000000000000000000606482015260840161088c565b600080547fffffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffff16740100000000000000000000000000000000000000001790558015610a5357600080547fffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffffff1675010000000000000000000000000000000000000000001790555b610a5b611601565b8015610abe57600080547fffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffffff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b50565b610ac9611357565b6107216116f8565b600062030d40610ae2601085612067565b6103e8610af16103f886612067565b610afb91906120c2565b610b0591906120e5565b610b0f91906120e5565b949350505050565b600260975403610b83576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f5265656e7472616e637947756172643a207265656e7472616e742063616c6c00604482015260640161088c565b6002609755610b90611753565b60f087901c60018114610c4b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605560248201527f43726f7373446f6d61696e4d657373656e6765723a206f6e6c7920766572736960448201527f6f6e2031206d657373616765732061726520737570706f72746564206166746560648201527f722074686520426564726f636b20757067726164650000000000000000000000608482015260a40161088c565b6000610c91898989898989898080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152506117c092505050565b905073ffffffffffffffffffffffffffffffffffffffff7fffffffffffffffffffffffffeeeeffffffffffffffffffffffffffffffffeeef330181167f000000000000000000000000000000000000000000000000000000000000000090911603610d0a57853414610d0557610d0561210d565b610e5c565b3415610dbe576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605060248201527f43726f7373446f6d61696e4d657373656e6765723a2076616c7565206d75737460448201527f206265207a65726f20756e6c657373206d6573736167652069732066726f6d2060648201527f612073797374656d206164647265737300000000000000000000000000000000608482015260a40161088c565b600081815260ce602052604090205460ff16610e5c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f43726f7373446f6d61696e4d657373656e6765723a206d65737361676520636160448201527f6e6e6f74206265207265706c6179656400000000000000000000000000000000606482015260840161088c565b610e65876117e3565b15610f18576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604360248201527f43726f7373446f6d61696e4d657373656e6765723a2063616e6e6f742073656e60448201527f64206d65737361676520746f20626c6f636b65642073797374656d206164647260648201527f6573730000000000000000000000000000000000000000000000000000000000608482015260a40161088c565b600081815260cb602052604090205460ff1615610fb7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f43726f7373446f6d61696e4d657373656e6765723a206d65737361676520686160448201527f7320616c7265616479206265656e2072656c6179656400000000000000000000606482015260840161088c565b610fc361afc88661213c565b5a1015611052576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f43726f7373446f6d61696e4d657373656e6765723a20696e737566666963696560448201527f6e742067617320746f2072656c6179206d657373616765000000000000000000606482015260840161088c565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff8a1617905560006110ee886110a661138861afc8612154565b5a6110b19190612154565b8988888080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061183892505050565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead179055905080151560010361118957600082815260cb602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183917f4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c91a26111e8565b600082815260ce602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183917f99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f91a25b505060016097555050505050505050565b611201611357565b73ffffffffffffffffffffffffffffffffffffffff81166112a4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015260840161088c565b610abe8161158a565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b6040517fc2b3e5ac0000000000000000000000000000000000000000000000000000000081527342000000000000000000000000000000000000169063c2b3e5ac90849061131f9088908890879060040161216b565b6000604051808303818588803b15801561133857600080fd5b505af115801561134c573d6000803e3d6000fd5b505050505050505050565b60335473ffffffffffffffffffffffffffffffffffffffff163314610721576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161088c565b6113e0611852565b606580547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001690557f5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa335b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200160405180910390a1565b60608160000361149857505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156114c257806114ac816121b3565b91506114bb9050600a836121eb565b915061149c565b60008167ffffffffffffffff8111156114dd576114dd6121ff565b6040519080825280601f01601f191660200182016040528015611507576020820181803683370190505b5090505b8415610b0f5761151c600183612154565b9150611529600a8661222e565b61153490603061213c565b60f81b81838151811061154957611549612242565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350611583600a866121eb565b945061150b565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b6000547501000000000000000000000000000000000000000000900460ff166116ac576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161088c565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead1790556116e06118be565b6116e8611969565b6116f0611a1d565b610721611af2565b611700611753565b606580547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790557f62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a25861142b3390565b60655460ff1615610721576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601060248201527f5061757361626c653a2070617573656400000000000000000000000000000000604482015260640161088c565b60006117d0878787878787611ba4565b8051906020012090509695505050505050565b600073ffffffffffffffffffffffffffffffffffffffff8216301480611832575073ffffffffffffffffffffffffffffffffffffffff8216734200000000000000000000000000000000000016145b92915050565b600080600080845160208601878a8af19695505050505050565b60655460ff16610721576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f5061757361626c653a206e6f7420706175736564000000000000000000000000604482015260640161088c565b6000547501000000000000000000000000000000000000000000900460ff16610721576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161088c565b6000547501000000000000000000000000000000000000000000900460ff16611a14576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161088c565b6107213361158a565b6000547501000000000000000000000000000000000000000000900460ff16611ac8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161088c565b606580547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00169055565b6000547501000000000000000000000000000000000000000000900460ff16611b9d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161088c565b6001609755565b6060868686868686604051602401611bc196959493929190612271565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fd764ad0b0000000000000000000000000000000000000000000000000000000017905290509695505050505050565b803573ffffffffffffffffffffffffffffffffffffffff81168114611c6757600080fd5b919050565b60008083601f840112611c7e57600080fd5b50813567ffffffffffffffff811115611c9657600080fd5b602083019150836020828501011115611cae57600080fd5b9250929050565b803563ffffffff81168114611c6757600080fd5b60008060008060608587031215611cdf57600080fd5b611ce885611c43565b9350602085013567ffffffffffffffff811115611d0457600080fd5b611d1087828801611c6c565b9094509250611d23905060408601611cb5565b905092959194509250565b60005b83811015611d49578181015183820152602001611d31565b83811115611d58576000848401525b50505050565b60008151808452611d76816020860160208601611d2e565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000611dbb6020830184611d5e565b9392505050565b600060208284031215611dd457600080fd5b5035919050565b600080600060408486031215611df057600080fd5b833567ffffffffffffffff811115611e0757600080fd5b611e1386828701611c6c565b9094509250611e26905060208501611cb5565b90509250925092565b600080600080600080600060c0888a031215611e4a57600080fd5b87359650611e5a60208901611c43565b9550611e6860408901611c43565b9450606088013593506080880135925060a088013567ffffffffffffffff811115611e9257600080fd5b611e9e8a828b01611c6c565b989b979a50959850939692959293505050565b600060208284031215611ec357600080fd5b611dbb82611c43565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b878152600073ffffffffffffffffffffffffffffffffffffffff808916602084015280881660408401525085606083015263ffffffff8516608083015260c060a0830152611f6760c083018486611ecc565b9998505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff86168152608060208201526000611fa4608083018688611ecc565b905083604083015263ffffffff831660608301529695505050505050565b60008451611fd4818460208901611d2e565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551612010816001850160208a01611d2e565b6001920191820152835161202b816002840160208801611d2e565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600063ffffffff8083168185168183048111821515161561208a5761208a612038565b02949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600063ffffffff808416806120d9576120d9612093565b92169190910492915050565b600063ffffffff80831681851680830382111561210457612104612038565b01949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052600160045260246000fd5b6000821982111561214f5761214f612038565b500190565b60008282101561216657612166612038565b500390565b73ffffffffffffffffffffffffffffffffffffffff8416815267ffffffffffffffff831660208201526060604082015260006121aa6060830184611d5e565b95945050505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036121e4576121e4612038565b5060010190565b6000826121fa576121fa612093565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60008261223d5761223d612093565b500690565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b868152600073ffffffffffffffffffffffffffffffffffffffff808816602084015280871660408401525084606083015283608083015260c060a08301526122bc60c0830184611d5e565b9897505050505050505056fea164736f6c634300080f000a496e697469616c697a61626c653a20636f6e7472616374206973206e6f742069",
}

// L2CrossDomainMessengerABI is the input ABI used to generate the binding from.
// Deprecated: Use L2CrossDomainMessengerMetaData.ABI instead.
var L2CrossDomainMessengerABI = L2CrossDomainMessengerMetaData.ABI

// L2CrossDomainMessengerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2CrossDomainMessengerMetaData.Bin instead.
var L2CrossDomainMessengerBin = L2CrossDomainMessengerMetaData.Bin

// DeployL2CrossDomainMessenger deploys a new Ethereum contract, binding an instance of L2CrossDomainMessenger to it.
func DeployL2CrossDomainMessenger(auth *bind.TransactOpts, backend bind.ContractBackend, _l1CrossDomainMessenger common.Address) (common.Address, *types.Transaction, *L2CrossDomainMessenger, error) {
	parsed, err := L2CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2CrossDomainMessengerBin), backend, _l1CrossDomainMessenger)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2CrossDomainMessenger{L2CrossDomainMessengerCaller: L2CrossDomainMessengerCaller{contract: contract}, L2CrossDomainMessengerTransactor: L2CrossDomainMessengerTransactor{contract: contract}, L2CrossDomainMessengerFilterer: L2CrossDomainMessengerFilterer{contract: contract}}, nil
}

// L2CrossDomainMessenger is an auto generated Go binding around an Ethereum contract.
type L2CrossDomainMessenger struct {
	L2CrossDomainMessengerCaller     // Read-only binding to the contract
	L2CrossDomainMessengerTransactor // Write-only binding to the contract
	L2CrossDomainMessengerFilterer   // Log filterer for contract events
}

// L2CrossDomainMessengerCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2CrossDomainMessengerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2CrossDomainMessengerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2CrossDomainMessengerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2CrossDomainMessengerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2CrossDomainMessengerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2CrossDomainMessengerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2CrossDomainMessengerSession struct {
	Contract     *L2CrossDomainMessenger // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// L2CrossDomainMessengerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2CrossDomainMessengerCallerSession struct {
	Contract *L2CrossDomainMessengerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// L2CrossDomainMessengerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2CrossDomainMessengerTransactorSession struct {
	Contract     *L2CrossDomainMessengerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// L2CrossDomainMessengerRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2CrossDomainMessengerRaw struct {
	Contract *L2CrossDomainMessenger // Generic contract binding to access the raw methods on
}

// L2CrossDomainMessengerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2CrossDomainMessengerCallerRaw struct {
	Contract *L2CrossDomainMessengerCaller // Generic read-only contract binding to access the raw methods on
}

// L2CrossDomainMessengerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2CrossDomainMessengerTransactorRaw struct {
	Contract *L2CrossDomainMessengerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2CrossDomainMessenger creates a new instance of L2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2CrossDomainMessenger(address common.Address, backend bind.ContractBackend) (*L2CrossDomainMessenger, error) {
	contract, err := bindL2CrossDomainMessenger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessenger{L2CrossDomainMessengerCaller: L2CrossDomainMessengerCaller{contract: contract}, L2CrossDomainMessengerTransactor: L2CrossDomainMessengerTransactor{contract: contract}, L2CrossDomainMessengerFilterer: L2CrossDomainMessengerFilterer{contract: contract}}, nil
}

// NewL2CrossDomainMessengerCaller creates a new read-only instance of L2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2CrossDomainMessengerCaller(address common.Address, caller bind.ContractCaller) (*L2CrossDomainMessengerCaller, error) {
	contract, err := bindL2CrossDomainMessenger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerCaller{contract: contract}, nil
}

// NewL2CrossDomainMessengerTransactor creates a new write-only instance of L2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2CrossDomainMessengerTransactor(address common.Address, transactor bind.ContractTransactor) (*L2CrossDomainMessengerTransactor, error) {
	contract, err := bindL2CrossDomainMessenger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerTransactor{contract: contract}, nil
}

// NewL2CrossDomainMessengerFilterer creates a new log filterer instance of L2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2CrossDomainMessengerFilterer(address common.Address, filterer bind.ContractFilterer) (*L2CrossDomainMessengerFilterer, error) {
	contract, err := bindL2CrossDomainMessenger(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerFilterer{contract: contract}, nil
}

// bindL2CrossDomainMessenger binds a generic wrapper to an already deployed contract.
func bindL2CrossDomainMessenger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2CrossDomainMessengerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2CrossDomainMessenger *L2CrossDomainMessengerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2CrossDomainMessenger.Contract.L2CrossDomainMessengerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2CrossDomainMessenger *L2CrossDomainMessengerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.L2CrossDomainMessengerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2CrossDomainMessenger *L2CrossDomainMessengerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.L2CrossDomainMessengerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2CrossDomainMessenger.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.contract.Transact(opts, method, params...)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) MESSAGEVERSION(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "MESSAGE_VERSION")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) MESSAGEVERSION() (uint16, error) {
	return _L2CrossDomainMessenger.Contract.MESSAGEVERSION(&_L2CrossDomainMessenger.CallOpts)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) MESSAGEVERSION() (uint16, error) {
	return _L2CrossDomainMessenger.Contract.MESSAGEVERSION(&_L2CrossDomainMessenger.CallOpts)
}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) MINGASCALLDATAOVERHEAD(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_CALLDATA_OVERHEAD")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) MINGASCALLDATAOVERHEAD() (uint32, error) {
	return _L2CrossDomainMessenger.Contract.MINGASCALLDATAOVERHEAD(&_L2CrossDomainMessenger.CallOpts)
}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) MINGASCALLDATAOVERHEAD() (uint32, error) {
	return _L2CrossDomainMessenger.Contract.MINGASCALLDATAOVERHEAD(&_L2CrossDomainMessenger.CallOpts)
}

// MINGASCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x7dea7cc3.
//
// Solidity: function MIN_GAS_CONSTANT_OVERHEAD() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) MINGASCONSTANTOVERHEAD(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_CONSTANT_OVERHEAD")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// MINGASCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x7dea7cc3.
//
// Solidity: function MIN_GAS_CONSTANT_OVERHEAD() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) MINGASCONSTANTOVERHEAD() (uint32, error) {
	return _L2CrossDomainMessenger.Contract.MINGASCONSTANTOVERHEAD(&_L2CrossDomainMessenger.CallOpts)
}

// MINGASCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x7dea7cc3.
//
// Solidity: function MIN_GAS_CONSTANT_OVERHEAD() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) MINGASCONSTANTOVERHEAD() (uint32, error) {
	return _L2CrossDomainMessenger.Contract.MINGASCONSTANTOVERHEAD(&_L2CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) MINGASDYNAMICOVERHEADDENOMINATOR(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) MINGASDYNAMICOVERHEADDENOMINATOR() (uint32, error) {
	return _L2CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADDENOMINATOR(&_L2CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) MINGASDYNAMICOVERHEADDENOMINATOR() (uint32, error) {
	return _L2CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADDENOMINATOR(&_L2CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) MINGASDYNAMICOVERHEADNUMERATOR(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) MINGASDYNAMICOVERHEADNUMERATOR() (uint32, error) {
	return _L2CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADNUMERATOR(&_L2CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) MINGASDYNAMICOVERHEADNUMERATOR() (uint32, error) {
	return _L2CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADNUMERATOR(&_L2CrossDomainMessenger.CallOpts)
}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) BaseGas(opts *bind.CallOpts, _message []byte, _minGasLimit uint32) (uint32, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "baseGas", _message, _minGasLimit)

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) BaseGas(_message []byte, _minGasLimit uint32) (uint32, error) {
	return _L2CrossDomainMessenger.Contract.BaseGas(&_L2CrossDomainMessenger.CallOpts, _message, _minGasLimit)
}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint32)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) BaseGas(_message []byte, _minGasLimit uint32) (uint32, error) {
	return _L2CrossDomainMessenger.Contract.BaseGas(&_L2CrossDomainMessenger.CallOpts, _message, _minGasLimit)
}

// L1CrossDomainMessenger is a free data retrieval call binding the contract method 0xa7119869.
//
// Solidity: function l1CrossDomainMessenger() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) L1CrossDomainMessenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "l1CrossDomainMessenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1CrossDomainMessenger is a free data retrieval call binding the contract method 0xa7119869.
//
// Solidity: function l1CrossDomainMessenger() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) L1CrossDomainMessenger() (common.Address, error) {
	return _L2CrossDomainMessenger.Contract.L1CrossDomainMessenger(&_L2CrossDomainMessenger.CallOpts)
}

// L1CrossDomainMessenger is a free data retrieval call binding the contract method 0xa7119869.
//
// Solidity: function l1CrossDomainMessenger() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) L1CrossDomainMessenger() (common.Address, error) {
	return _L2CrossDomainMessenger.Contract.L1CrossDomainMessenger(&_L2CrossDomainMessenger.CallOpts)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) MessageNonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "messageNonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) MessageNonce() (*big.Int, error) {
	return _L2CrossDomainMessenger.Contract.MessageNonce(&_L2CrossDomainMessenger.CallOpts)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) MessageNonce() (*big.Int, error) {
	return _L2CrossDomainMessenger.Contract.MessageNonce(&_L2CrossDomainMessenger.CallOpts)
}

// OtherMessenger is a free data retrieval call binding the contract method 0xdb505d80.
//
// Solidity: function otherMessenger() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) OtherMessenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "otherMessenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OtherMessenger is a free data retrieval call binding the contract method 0xdb505d80.
//
// Solidity: function otherMessenger() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) OtherMessenger() (common.Address, error) {
	return _L2CrossDomainMessenger.Contract.OtherMessenger(&_L2CrossDomainMessenger.CallOpts)
}

// OtherMessenger is a free data retrieval call binding the contract method 0xdb505d80.
//
// Solidity: function otherMessenger() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) OtherMessenger() (common.Address, error) {
	return _L2CrossDomainMessenger.Contract.OtherMessenger(&_L2CrossDomainMessenger.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) Owner() (common.Address, error) {
	return _L2CrossDomainMessenger.Contract.Owner(&_L2CrossDomainMessenger.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) Owner() (common.Address, error) {
	return _L2CrossDomainMessenger.Contract.Owner(&_L2CrossDomainMessenger.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) Paused() (bool, error) {
	return _L2CrossDomainMessenger.Contract.Paused(&_L2CrossDomainMessenger.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) Paused() (bool, error) {
	return _L2CrossDomainMessenger.Contract.Paused(&_L2CrossDomainMessenger.CallOpts)
}

// ReceivedMessages is a free data retrieval call binding the contract method 0xf69f8151.
//
// Solidity: function receivedMessages(bytes32 ) view returns(bool)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) ReceivedMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "receivedMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ReceivedMessages is a free data retrieval call binding the contract method 0xf69f8151.
//
// Solidity: function receivedMessages(bytes32 ) view returns(bool)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) ReceivedMessages(arg0 [32]byte) (bool, error) {
	return _L2CrossDomainMessenger.Contract.ReceivedMessages(&_L2CrossDomainMessenger.CallOpts, arg0)
}

// ReceivedMessages is a free data retrieval call binding the contract method 0xf69f8151.
//
// Solidity: function receivedMessages(bytes32 ) view returns(bool)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) ReceivedMessages(arg0 [32]byte) (bool, error) {
	return _L2CrossDomainMessenger.Contract.ReceivedMessages(&_L2CrossDomainMessenger.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) SuccessfulMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "successfulMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _L2CrossDomainMessenger.Contract.SuccessfulMessages(&_L2CrossDomainMessenger.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _L2CrossDomainMessenger.Contract.SuccessfulMessages(&_L2CrossDomainMessenger.CallOpts, arg0)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) Version() (string, error) {
	return _L2CrossDomainMessenger.Contract.Version(&_L2CrossDomainMessenger.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) Version() (string, error) {
	return _L2CrossDomainMessenger.Contract.Version(&_L2CrossDomainMessenger.CallOpts)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCaller) XDomainMessageSender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2CrossDomainMessenger.contract.Call(opts, &out, "xDomainMessageSender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) XDomainMessageSender() (common.Address, error) {
	return _L2CrossDomainMessenger.Contract.XDomainMessageSender(&_L2CrossDomainMessenger.CallOpts)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerCallerSession) XDomainMessageSender() (common.Address, error) {
	return _L2CrossDomainMessenger.Contract.XDomainMessageSender(&_L2CrossDomainMessenger.CallOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactor) Initialize(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.contract.Transact(opts, "initialize")
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) Initialize() (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.Initialize(&_L2CrossDomainMessenger.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactorSession) Initialize() (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.Initialize(&_L2CrossDomainMessenger.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) Pause() (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.Pause(&_L2CrossDomainMessenger.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactorSession) Pause() (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.Pause(&_L2CrossDomainMessenger.TransactOpts)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd764ad0b.
//
// Solidity: function relayMessage(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactor) RelayMessage(opts *bind.TransactOpts, _nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.contract.Transact(opts, "relayMessage", _nonce, _sender, _target, _value, _minGasLimit, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd764ad0b.
//
// Solidity: function relayMessage(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) RelayMessage(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.RelayMessage(&_L2CrossDomainMessenger.TransactOpts, _nonce, _sender, _target, _value, _minGasLimit, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd764ad0b.
//
// Solidity: function relayMessage(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactorSession) RelayMessage(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.RelayMessage(&_L2CrossDomainMessenger.TransactOpts, _nonce, _sender, _target, _value, _minGasLimit, _message)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) RenounceOwnership() (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.RenounceOwnership(&_L2CrossDomainMessenger.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.RenounceOwnership(&_L2CrossDomainMessenger.TransactOpts)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactor) SendMessage(opts *bind.TransactOpts, _target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.contract.Transact(opts, "sendMessage", _target, _message, _minGasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) SendMessage(_target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.SendMessage(&_L2CrossDomainMessenger.TransactOpts, _target, _message, _minGasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactorSession) SendMessage(_target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.SendMessage(&_L2CrossDomainMessenger.TransactOpts, _target, _message, _minGasLimit)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.TransferOwnership(&_L2CrossDomainMessenger.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.TransferOwnership(&_L2CrossDomainMessenger.TransactOpts, newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2CrossDomainMessenger.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerSession) Unpause() (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.Unpause(&_L2CrossDomainMessenger.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_L2CrossDomainMessenger *L2CrossDomainMessengerTransactorSession) Unpause() (*types.Transaction, error) {
	return _L2CrossDomainMessenger.Contract.Unpause(&_L2CrossDomainMessenger.TransactOpts)
}

// L2CrossDomainMessengerFailedRelayedMessageIterator is returned from FilterFailedRelayedMessage and is used to iterate over the raw logs and unpacked data for FailedRelayedMessage events raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerFailedRelayedMessageIterator struct {
	Event *L2CrossDomainMessengerFailedRelayedMessage // Event containing the contract specifics and raw log

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
func (it *L2CrossDomainMessengerFailedRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2CrossDomainMessengerFailedRelayedMessage)
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
		it.Event = new(L2CrossDomainMessengerFailedRelayedMessage)
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
func (it *L2CrossDomainMessengerFailedRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2CrossDomainMessengerFailedRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2CrossDomainMessengerFailedRelayedMessage represents a FailedRelayedMessage event raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerFailedRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterFailedRelayedMessage is a free log retrieval operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) FilterFailedRelayedMessage(opts *bind.FilterOpts, msgHash [][32]byte) (*L2CrossDomainMessengerFailedRelayedMessageIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L2CrossDomainMessenger.contract.FilterLogs(opts, "FailedRelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerFailedRelayedMessageIterator{contract: _L2CrossDomainMessenger.contract, event: "FailedRelayedMessage", logs: logs, sub: sub}, nil
}

// WatchFailedRelayedMessage is a free log subscription operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) WatchFailedRelayedMessage(opts *bind.WatchOpts, sink chan<- *L2CrossDomainMessengerFailedRelayedMessage, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L2CrossDomainMessenger.contract.WatchLogs(opts, "FailedRelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2CrossDomainMessengerFailedRelayedMessage)
				if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
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

// ParseFailedRelayedMessage is a log parse operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) ParseFailedRelayedMessage(log types.Log) (*L2CrossDomainMessengerFailedRelayedMessage, error) {
	event := new(L2CrossDomainMessengerFailedRelayedMessage)
	if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2CrossDomainMessengerInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerInitializedIterator struct {
	Event *L2CrossDomainMessengerInitialized // Event containing the contract specifics and raw log

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
func (it *L2CrossDomainMessengerInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2CrossDomainMessengerInitialized)
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
		it.Event = new(L2CrossDomainMessengerInitialized)
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
func (it *L2CrossDomainMessengerInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2CrossDomainMessengerInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2CrossDomainMessengerInitialized represents a Initialized event raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) FilterInitialized(opts *bind.FilterOpts) (*L2CrossDomainMessengerInitializedIterator, error) {

	logs, sub, err := _L2CrossDomainMessenger.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerInitializedIterator{contract: _L2CrossDomainMessenger.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L2CrossDomainMessengerInitialized) (event.Subscription, error) {

	logs, sub, err := _L2CrossDomainMessenger.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2CrossDomainMessengerInitialized)
				if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) ParseInitialized(log types.Log) (*L2CrossDomainMessengerInitialized, error) {
	event := new(L2CrossDomainMessengerInitialized)
	if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2CrossDomainMessengerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerOwnershipTransferredIterator struct {
	Event *L2CrossDomainMessengerOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *L2CrossDomainMessengerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2CrossDomainMessengerOwnershipTransferred)
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
		it.Event = new(L2CrossDomainMessengerOwnershipTransferred)
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
func (it *L2CrossDomainMessengerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2CrossDomainMessengerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2CrossDomainMessengerOwnershipTransferred represents a OwnershipTransferred event raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*L2CrossDomainMessengerOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L2CrossDomainMessenger.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerOwnershipTransferredIterator{contract: _L2CrossDomainMessenger.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *L2CrossDomainMessengerOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L2CrossDomainMessenger.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2CrossDomainMessengerOwnershipTransferred)
				if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) ParseOwnershipTransferred(log types.Log) (*L2CrossDomainMessengerOwnershipTransferred, error) {
	event := new(L2CrossDomainMessengerOwnershipTransferred)
	if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2CrossDomainMessengerPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerPausedIterator struct {
	Event *L2CrossDomainMessengerPaused // Event containing the contract specifics and raw log

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
func (it *L2CrossDomainMessengerPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2CrossDomainMessengerPaused)
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
		it.Event = new(L2CrossDomainMessengerPaused)
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
func (it *L2CrossDomainMessengerPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2CrossDomainMessengerPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2CrossDomainMessengerPaused represents a Paused event raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) FilterPaused(opts *bind.FilterOpts) (*L2CrossDomainMessengerPausedIterator, error) {

	logs, sub, err := _L2CrossDomainMessenger.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerPausedIterator{contract: _L2CrossDomainMessenger.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *L2CrossDomainMessengerPaused) (event.Subscription, error) {

	logs, sub, err := _L2CrossDomainMessenger.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2CrossDomainMessengerPaused)
				if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "Paused", log); err != nil {
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

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) ParsePaused(log types.Log) (*L2CrossDomainMessengerPaused, error) {
	event := new(L2CrossDomainMessengerPaused)
	if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2CrossDomainMessengerRelayedMessageIterator is returned from FilterRelayedMessage and is used to iterate over the raw logs and unpacked data for RelayedMessage events raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerRelayedMessageIterator struct {
	Event *L2CrossDomainMessengerRelayedMessage // Event containing the contract specifics and raw log

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
func (it *L2CrossDomainMessengerRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2CrossDomainMessengerRelayedMessage)
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
		it.Event = new(L2CrossDomainMessengerRelayedMessage)
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
func (it *L2CrossDomainMessengerRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2CrossDomainMessengerRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2CrossDomainMessengerRelayedMessage represents a RelayedMessage event raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRelayedMessage is a free log retrieval operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) FilterRelayedMessage(opts *bind.FilterOpts, msgHash [][32]byte) (*L2CrossDomainMessengerRelayedMessageIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L2CrossDomainMessenger.contract.FilterLogs(opts, "RelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerRelayedMessageIterator{contract: _L2CrossDomainMessenger.contract, event: "RelayedMessage", logs: logs, sub: sub}, nil
}

// WatchRelayedMessage is a free log subscription operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) WatchRelayedMessage(opts *bind.WatchOpts, sink chan<- *L2CrossDomainMessengerRelayedMessage, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L2CrossDomainMessenger.contract.WatchLogs(opts, "RelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2CrossDomainMessengerRelayedMessage)
				if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
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

// ParseRelayedMessage is a log parse operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) ParseRelayedMessage(log types.Log) (*L2CrossDomainMessengerRelayedMessage, error) {
	event := new(L2CrossDomainMessengerRelayedMessage)
	if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2CrossDomainMessengerSentMessageIterator is returned from FilterSentMessage and is used to iterate over the raw logs and unpacked data for SentMessage events raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerSentMessageIterator struct {
	Event *L2CrossDomainMessengerSentMessage // Event containing the contract specifics and raw log

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
func (it *L2CrossDomainMessengerSentMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2CrossDomainMessengerSentMessage)
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
		it.Event = new(L2CrossDomainMessengerSentMessage)
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
func (it *L2CrossDomainMessengerSentMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2CrossDomainMessengerSentMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2CrossDomainMessengerSentMessage represents a SentMessage event raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerSentMessage struct {
	Target       common.Address
	Sender       common.Address
	Message      []byte
	MessageNonce *big.Int
	GasLimit     *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSentMessage is a free log retrieval operation binding the contract event 0xcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a.
//
// Solidity: event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) FilterSentMessage(opts *bind.FilterOpts, target []common.Address) (*L2CrossDomainMessengerSentMessageIterator, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _L2CrossDomainMessenger.contract.FilterLogs(opts, "SentMessage", targetRule)
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerSentMessageIterator{contract: _L2CrossDomainMessenger.contract, event: "SentMessage", logs: logs, sub: sub}, nil
}

// WatchSentMessage is a free log subscription operation binding the contract event 0xcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a.
//
// Solidity: event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) WatchSentMessage(opts *bind.WatchOpts, sink chan<- *L2CrossDomainMessengerSentMessage, target []common.Address) (event.Subscription, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _L2CrossDomainMessenger.contract.WatchLogs(opts, "SentMessage", targetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2CrossDomainMessengerSentMessage)
				if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
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

// ParseSentMessage is a log parse operation binding the contract event 0xcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a.
//
// Solidity: event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) ParseSentMessage(log types.Log) (*L2CrossDomainMessengerSentMessage, error) {
	event := new(L2CrossDomainMessengerSentMessage)
	if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2CrossDomainMessengerSentMessageExtension1Iterator is returned from FilterSentMessageExtension1 and is used to iterate over the raw logs and unpacked data for SentMessageExtension1 events raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerSentMessageExtension1Iterator struct {
	Event *L2CrossDomainMessengerSentMessageExtension1 // Event containing the contract specifics and raw log

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
func (it *L2CrossDomainMessengerSentMessageExtension1Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2CrossDomainMessengerSentMessageExtension1)
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
		it.Event = new(L2CrossDomainMessengerSentMessageExtension1)
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
func (it *L2CrossDomainMessengerSentMessageExtension1Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2CrossDomainMessengerSentMessageExtension1Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2CrossDomainMessengerSentMessageExtension1 represents a SentMessageExtension1 event raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerSentMessageExtension1 struct {
	Sender common.Address
	Value  *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterSentMessageExtension1 is a free log retrieval operation binding the contract event 0x8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d546.
//
// Solidity: event SentMessageExtension1(address indexed sender, uint256 value)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) FilterSentMessageExtension1(opts *bind.FilterOpts, sender []common.Address) (*L2CrossDomainMessengerSentMessageExtension1Iterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _L2CrossDomainMessenger.contract.FilterLogs(opts, "SentMessageExtension1", senderRule)
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerSentMessageExtension1Iterator{contract: _L2CrossDomainMessenger.contract, event: "SentMessageExtension1", logs: logs, sub: sub}, nil
}

// WatchSentMessageExtension1 is a free log subscription operation binding the contract event 0x8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d546.
//
// Solidity: event SentMessageExtension1(address indexed sender, uint256 value)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) WatchSentMessageExtension1(opts *bind.WatchOpts, sink chan<- *L2CrossDomainMessengerSentMessageExtension1, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _L2CrossDomainMessenger.contract.WatchLogs(opts, "SentMessageExtension1", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2CrossDomainMessengerSentMessageExtension1)
				if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "SentMessageExtension1", log); err != nil {
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

// ParseSentMessageExtension1 is a log parse operation binding the contract event 0x8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d546.
//
// Solidity: event SentMessageExtension1(address indexed sender, uint256 value)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) ParseSentMessageExtension1(log types.Log) (*L2CrossDomainMessengerSentMessageExtension1, error) {
	event := new(L2CrossDomainMessengerSentMessageExtension1)
	if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "SentMessageExtension1", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2CrossDomainMessengerUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerUnpausedIterator struct {
	Event *L2CrossDomainMessengerUnpaused // Event containing the contract specifics and raw log

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
func (it *L2CrossDomainMessengerUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2CrossDomainMessengerUnpaused)
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
		it.Event = new(L2CrossDomainMessengerUnpaused)
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
func (it *L2CrossDomainMessengerUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2CrossDomainMessengerUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2CrossDomainMessengerUnpaused represents a Unpaused event raised by the L2CrossDomainMessenger contract.
type L2CrossDomainMessengerUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) FilterUnpaused(opts *bind.FilterOpts) (*L2CrossDomainMessengerUnpausedIterator, error) {

	logs, sub, err := _L2CrossDomainMessenger.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &L2CrossDomainMessengerUnpausedIterator{contract: _L2CrossDomainMessenger.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *L2CrossDomainMessengerUnpaused) (event.Subscription, error) {

	logs, sub, err := _L2CrossDomainMessenger.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2CrossDomainMessengerUnpaused)
				if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "Unpaused", log); err != nil {
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

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_L2CrossDomainMessenger *L2CrossDomainMessengerFilterer) ParseUnpaused(log types.Log) (*L2CrossDomainMessengerUnpaused, error) {
	event := new(L2CrossDomainMessengerUnpaused)
	if err := _L2CrossDomainMessenger.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
