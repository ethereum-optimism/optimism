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

// L2ERC721BridgeMetaData contains all meta data concerning the L2ERC721Bridge contract.
var L2ERC721BridgeMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"MESSENGER\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractCrossDomainMessenger\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"OTHER_BRIDGE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractStandardBridge\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"bridgeERC721\",\"inputs\":[{\"name\":\"_localToken\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_remoteToken\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_minGasLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_extraData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"bridgeERC721To\",\"inputs\":[{\"name\":\"_localToken\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_remoteToken\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_minGasLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_extraData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizeBridgeERC721\",\"inputs\":[{\"name\":\"_localToken\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_remoteToken\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_extraData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_l1ERC721Bridge\",\"type\":\"address\",\"internalType\":\"addresspayable\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"messenger\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractCrossDomainMessenger\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"otherBridge\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractStandardBridge\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"paused\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"ERC721BridgeFinalized\",\"inputs\":[{\"name\":\"localToken\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"remoteToken\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"extraData\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ERC721BridgeInitiated\",\"inputs\":[{\"name\":\"localToken\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"remoteToken\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"extraData\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false}]",
	Bin: "0x60806040523480156200001157600080fd5b506200001e600062000024565b62000217565b600054610100900460ff1615808015620000455750600054600160ff909116105b8062000075575062000062306200016d60201b62000a3e1760201c565b15801562000075575060005460ff166001145b620000de5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805460ff19166001179055801562000102576000805461ff0019166101001790555b62000122734200000000000000000000000000000000000007836200017c565b801562000169576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050565b6001600160a01b03163b151590565b600054610100900460ff16620001e95760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201526a6e697469616c697a696e6760a81b6064820152608401620000d5565b600180546001600160a01b039384166001600160a01b03199182161790915560028054929093169116179055565b61160c80620002276000396000f3fe608060405234801561001057600080fd5b50600436106100be5760003560e01c80637f46ddb211610076578063aa5574521161005b578063aa557452146101c9578063c4d66de8146101dc578063c89701a2146101ef57600080fd5b80637f46ddb21461018d578063927ede2d146101ab57600080fd5b806354fd4d50116100a757806354fd4d50146101225780635c975abb1461016b578063761f44931461017a57600080fd5b80633687011a146100c35780633cb747bf146100d8575b600080fd5b6100d66100d136600461128a565b61020f565b005b6001546100f89073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b61015e6040518060400160405280600581526020017f312e372e3000000000000000000000000000000000000000000000000000000081525081565b6040516101199190611378565b60405160008152602001610119565b6100d661018836600461138b565b6102bb565b60025473ffffffffffffffffffffffffffffffffffffffff166100f8565b60015473ffffffffffffffffffffffffffffffffffffffff166100f8565b6100d66101d7366004611423565b6107d9565b6100d66101ea36600461149a565b610895565b6002546100f89073ffffffffffffffffffffffffffffffffffffffff1681565b333b156102a3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602d60248201527f4552433732314272696467653a206163636f756e74206973206e6f742065787460448201527f65726e616c6c79206f776e65640000000000000000000000000000000000000060648201526084015b60405180910390fd5b6102b38686333388888888610a5a565b505050505050565b60015473ffffffffffffffffffffffffffffffffffffffff16331480156103905750600254600154604080517f6e296e45000000000000000000000000000000000000000000000000000000008152905173ffffffffffffffffffffffffffffffffffffffff9384169390921691636e296e45916004808201926020929091908290030181865afa158015610354573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061037891906114b7565b73ffffffffffffffffffffffffffffffffffffffff16145b61041c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603f60248201527f4552433732314272696467653a2066756e6374696f6e2063616e206f6e6c792060448201527f62652063616c6c65642066726f6d20746865206f746865722062726964676500606482015260840161029a565b3073ffffffffffffffffffffffffffffffffffffffff8816036104c1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f4c324552433732314272696467653a206c6f63616c20746f6b656e2063616e6e60448201527f6f742062652073656c6600000000000000000000000000000000000000000000606482015260840161029a565b6104eb877f74259ebf00000000000000000000000000000000000000000000000000000000610fc2565b610577576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f4c324552433732314272696467653a206c6f63616c20746f6b656e20696e746560448201527f7266616365206973206e6f7420636f6d706c69616e7400000000000000000000606482015260840161029a565b8673ffffffffffffffffffffffffffffffffffffffff1663d6c0b2c46040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105c2573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105e691906114b7565b73ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff16146106c6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604b60248201527f4c324552433732314272696467653a2077726f6e672072656d6f746520746f6b60448201527f656e20666f72204f7074696d69736d204d696e7461626c65204552433732312060648201527f6c6f63616c20746f6b656e000000000000000000000000000000000000000000608482015260a40161029a565b6040517fa144819400000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff85811660048301526024820185905288169063a144819490604401600060405180830381600087803b15801561073657600080fd5b505af115801561074a573d6000803e3d6000fd5b505050508473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff167f1f39bf6707b5d608453e0ae4c067b562bcc4c85c0f562ef5d2c774d2e7f131ac878787876040516107c8949392919061151d565b60405180910390a450505050505050565b73ffffffffffffffffffffffffffffffffffffffff851661087c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f4552433732314272696467653a206e667420726563697069656e742063616e6e60448201527f6f74206265206164647265737328302900000000000000000000000000000000606482015260840161029a565b61088c8787338888888888610a5a565b50505050505050565b600054610100900460ff16158080156108b55750600054600160ff909116105b806108cf5750303b1580156108cf575060005460ff166001145b61095b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a6564000000000000000000000000000000000000606482015260840161029a565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600117905580156109b957600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b6109d773420000000000000000000000000000000000000783610fe5565b8015610a3a57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b73ffffffffffffffffffffffffffffffffffffffff8716610afd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603160248201527f4c324552433732314272696467653a2072656d6f746520746f6b656e2063616e60448201527f6e6f742062652061646472657373283029000000000000000000000000000000606482015260840161029a565b6040517f6352211e0000000000000000000000000000000000000000000000000000000081526004810185905273ffffffffffffffffffffffffffffffffffffffff891690636352211e90602401602060405180830381865afa158015610b68573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b8c91906114b7565b73ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff1614610c46576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603e60248201527f4c324552433732314272696467653a205769746864726177616c206973206e6f60448201527f74206265696e6720696e69746961746564206279204e4654206f776e65720000606482015260840161029a565b60008873ffffffffffffffffffffffffffffffffffffffff1663d6c0b2c46040518163ffffffff1660e01b8152600401602060405180830381865afa158015610c93573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610cb791906114b7565b90508773ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614610d74576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f4c324552433732314272696467653a2072656d6f746520746f6b656e20646f6560448201527f73206e6f74206d6174636820676976656e2076616c7565000000000000000000606482015260840161029a565b6040517f9dc29fac00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8881166004830152602482018790528a1690639dc29fac90604401600060405180830381600087803b158015610de457600080fd5b505af1158015610df8573d6000803e3d6000fd5b50505050600063761f449360e01b828b8a8a8a8989604051602401610e23979695949392919061155d565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009094169390931790925260015460025492517f3dbb202b00000000000000000000000000000000000000000000000000000000815291935073ffffffffffffffffffffffffffffffffffffffff90811692633dbb202b92610f0292919091169085908a906004016115ba565b600060405180830381600087803b158015610f1c57600080fd5b505af1158015610f30573d6000803e3d6000fd5b505050508773ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff168b73ffffffffffffffffffffffffffffffffffffffff167fb7460e2a880f256ebef3406116ff3eee0cee51ebccdc2a40698f87ebb2e9c1a58a8a8989604051610fae949392919061151d565b60405180910390a450505050505050505050565b6000610fcd836110cf565b8015610fde5750610fde8383611134565b9392505050565b600054610100900460ff1661107c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161029a565b6001805473ffffffffffffffffffffffffffffffffffffffff9384167fffffffffffffffffffffffff00000000000000000000000000000000000000009182161790915560028054929093169116179055565b60006110fb827f01ffc9a700000000000000000000000000000000000000000000000000000000611134565b801561112e575061112c827fffffffff00000000000000000000000000000000000000000000000000000000611134565b155b92915050565b604080517fffffffff000000000000000000000000000000000000000000000000000000008316602480830191909152825180830390910181526044909101909152602080820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01ffc9a700000000000000000000000000000000000000000000000000000000178152825160009392849283928392918391908a617530fa92503d915060005190508280156111ec575060208210155b80156111f85750600081115b979650505050505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461122557600080fd5b50565b803563ffffffff8116811461123c57600080fd5b919050565b60008083601f84011261125357600080fd5b50813567ffffffffffffffff81111561126b57600080fd5b60208301915083602082850101111561128357600080fd5b9250929050565b60008060008060008060a087890312156112a357600080fd5b86356112ae81611203565b955060208701356112be81611203565b9450604087013593506112d360608801611228565b9250608087013567ffffffffffffffff8111156112ef57600080fd5b6112fb89828a01611241565b979a9699509497509295939492505050565b6000815180845260005b8181101561133357602081850181015186830182015201611317565b81811115611345576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000610fde602083018461130d565b600080600080600080600060c0888a0312156113a657600080fd5b87356113b181611203565b965060208801356113c181611203565b955060408801356113d181611203565b945060608801356113e181611203565b93506080880135925060a088013567ffffffffffffffff81111561140457600080fd5b6114108a828b01611241565b989b979a50959850939692959293505050565b600080600080600080600060c0888a03121561143e57600080fd5b873561144981611203565b9650602088013561145981611203565b9550604088013561146981611203565b94506060880135935061147e60808901611228565b925060a088013567ffffffffffffffff81111561140457600080fd5b6000602082840312156114ac57600080fd5b8135610fde81611203565b6000602082840312156114c957600080fd5b8151610fde81611203565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b73ffffffffffffffffffffffffffffffffffffffff851681528360208201526060604082015260006115536060830184866114d4565b9695505050505050565b600073ffffffffffffffffffffffffffffffffffffffff808a1683528089166020840152808816604084015280871660608401525084608083015260c060a08301526115ad60c0830184866114d4565b9998505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff841681526060602082015260006115e9606083018561130d565b905063ffffffff8316604083015294935050505056fea164736f6c634300080f000a",
}

// L2ERC721BridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use L2ERC721BridgeMetaData.ABI instead.
var L2ERC721BridgeABI = L2ERC721BridgeMetaData.ABI

// L2ERC721BridgeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2ERC721BridgeMetaData.Bin instead.
var L2ERC721BridgeBin = L2ERC721BridgeMetaData.Bin

// DeployL2ERC721Bridge deploys a new Ethereum contract, binding an instance of L2ERC721Bridge to it.
func DeployL2ERC721Bridge(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L2ERC721Bridge, error) {
	parsed, err := L2ERC721BridgeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2ERC721BridgeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2ERC721Bridge{L2ERC721BridgeCaller: L2ERC721BridgeCaller{contract: contract}, L2ERC721BridgeTransactor: L2ERC721BridgeTransactor{contract: contract}, L2ERC721BridgeFilterer: L2ERC721BridgeFilterer{contract: contract}}, nil
}

// L2ERC721Bridge is an auto generated Go binding around an Ethereum contract.
type L2ERC721Bridge struct {
	L2ERC721BridgeCaller     // Read-only binding to the contract
	L2ERC721BridgeTransactor // Write-only binding to the contract
	L2ERC721BridgeFilterer   // Log filterer for contract events
}

// L2ERC721BridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2ERC721BridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ERC721BridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2ERC721BridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ERC721BridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2ERC721BridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ERC721BridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2ERC721BridgeSession struct {
	Contract     *L2ERC721Bridge   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L2ERC721BridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2ERC721BridgeCallerSession struct {
	Contract *L2ERC721BridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// L2ERC721BridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2ERC721BridgeTransactorSession struct {
	Contract     *L2ERC721BridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// L2ERC721BridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2ERC721BridgeRaw struct {
	Contract *L2ERC721Bridge // Generic contract binding to access the raw methods on
}

// L2ERC721BridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2ERC721BridgeCallerRaw struct {
	Contract *L2ERC721BridgeCaller // Generic read-only contract binding to access the raw methods on
}

// L2ERC721BridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2ERC721BridgeTransactorRaw struct {
	Contract *L2ERC721BridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2ERC721Bridge creates a new instance of L2ERC721Bridge, bound to a specific deployed contract.
func NewL2ERC721Bridge(address common.Address, backend bind.ContractBackend) (*L2ERC721Bridge, error) {
	contract, err := bindL2ERC721Bridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2ERC721Bridge{L2ERC721BridgeCaller: L2ERC721BridgeCaller{contract: contract}, L2ERC721BridgeTransactor: L2ERC721BridgeTransactor{contract: contract}, L2ERC721BridgeFilterer: L2ERC721BridgeFilterer{contract: contract}}, nil
}

// NewL2ERC721BridgeCaller creates a new read-only instance of L2ERC721Bridge, bound to a specific deployed contract.
func NewL2ERC721BridgeCaller(address common.Address, caller bind.ContractCaller) (*L2ERC721BridgeCaller, error) {
	contract, err := bindL2ERC721Bridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2ERC721BridgeCaller{contract: contract}, nil
}

// NewL2ERC721BridgeTransactor creates a new write-only instance of L2ERC721Bridge, bound to a specific deployed contract.
func NewL2ERC721BridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*L2ERC721BridgeTransactor, error) {
	contract, err := bindL2ERC721Bridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2ERC721BridgeTransactor{contract: contract}, nil
}

// NewL2ERC721BridgeFilterer creates a new log filterer instance of L2ERC721Bridge, bound to a specific deployed contract.
func NewL2ERC721BridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*L2ERC721BridgeFilterer, error) {
	contract, err := bindL2ERC721Bridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2ERC721BridgeFilterer{contract: contract}, nil
}

// bindL2ERC721Bridge binds a generic wrapper to an already deployed contract.
func bindL2ERC721Bridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2ERC721BridgeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2ERC721Bridge *L2ERC721BridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2ERC721Bridge.Contract.L2ERC721BridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2ERC721Bridge *L2ERC721BridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.L2ERC721BridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2ERC721Bridge *L2ERC721BridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.L2ERC721BridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2ERC721Bridge *L2ERC721BridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2ERC721Bridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2ERC721Bridge *L2ERC721BridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2ERC721Bridge *L2ERC721BridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.contract.Transact(opts, method, params...)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeCaller) MESSENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ERC721Bridge.contract.Call(opts, &out, "MESSENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeSession) MESSENGER() (common.Address, error) {
	return _L2ERC721Bridge.Contract.MESSENGER(&_L2ERC721Bridge.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeCallerSession) MESSENGER() (common.Address, error) {
	return _L2ERC721Bridge.Contract.MESSENGER(&_L2ERC721Bridge.CallOpts)
}

// OTHERBRIDGE is a free data retrieval call binding the contract method 0x7f46ddb2.
//
// Solidity: function OTHER_BRIDGE() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeCaller) OTHERBRIDGE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ERC721Bridge.contract.Call(opts, &out, "OTHER_BRIDGE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OTHERBRIDGE is a free data retrieval call binding the contract method 0x7f46ddb2.
//
// Solidity: function OTHER_BRIDGE() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeSession) OTHERBRIDGE() (common.Address, error) {
	return _L2ERC721Bridge.Contract.OTHERBRIDGE(&_L2ERC721Bridge.CallOpts)
}

// OTHERBRIDGE is a free data retrieval call binding the contract method 0x7f46ddb2.
//
// Solidity: function OTHER_BRIDGE() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeCallerSession) OTHERBRIDGE() (common.Address, error) {
	return _L2ERC721Bridge.Contract.OTHERBRIDGE(&_L2ERC721Bridge.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ERC721Bridge.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeSession) Messenger() (common.Address, error) {
	return _L2ERC721Bridge.Contract.Messenger(&_L2ERC721Bridge.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeCallerSession) Messenger() (common.Address, error) {
	return _L2ERC721Bridge.Contract.Messenger(&_L2ERC721Bridge.CallOpts)
}

// OtherBridge is a free data retrieval call binding the contract method 0xc89701a2.
//
// Solidity: function otherBridge() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeCaller) OtherBridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ERC721Bridge.contract.Call(opts, &out, "otherBridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OtherBridge is a free data retrieval call binding the contract method 0xc89701a2.
//
// Solidity: function otherBridge() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeSession) OtherBridge() (common.Address, error) {
	return _L2ERC721Bridge.Contract.OtherBridge(&_L2ERC721Bridge.CallOpts)
}

// OtherBridge is a free data retrieval call binding the contract method 0xc89701a2.
//
// Solidity: function otherBridge() view returns(address)
func (_L2ERC721Bridge *L2ERC721BridgeCallerSession) OtherBridge() (common.Address, error) {
	return _L2ERC721Bridge.Contract.OtherBridge(&_L2ERC721Bridge.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L2ERC721Bridge *L2ERC721BridgeCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L2ERC721Bridge.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L2ERC721Bridge *L2ERC721BridgeSession) Paused() (bool, error) {
	return _L2ERC721Bridge.Contract.Paused(&_L2ERC721Bridge.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L2ERC721Bridge *L2ERC721BridgeCallerSession) Paused() (bool, error) {
	return _L2ERC721Bridge.Contract.Paused(&_L2ERC721Bridge.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ERC721Bridge *L2ERC721BridgeCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L2ERC721Bridge.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ERC721Bridge *L2ERC721BridgeSession) Version() (string, error) {
	return _L2ERC721Bridge.Contract.Version(&_L2ERC721Bridge.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ERC721Bridge *L2ERC721BridgeCallerSession) Version() (string, error) {
	return _L2ERC721Bridge.Contract.Version(&_L2ERC721Bridge.CallOpts)
}

// BridgeERC721 is a paid mutator transaction binding the contract method 0x3687011a.
//
// Solidity: function bridgeERC721(address _localToken, address _remoteToken, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2ERC721Bridge *L2ERC721BridgeTransactor) BridgeERC721(opts *bind.TransactOpts, _localToken common.Address, _remoteToken common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2ERC721Bridge.contract.Transact(opts, "bridgeERC721", _localToken, _remoteToken, _tokenId, _minGasLimit, _extraData)
}

// BridgeERC721 is a paid mutator transaction binding the contract method 0x3687011a.
//
// Solidity: function bridgeERC721(address _localToken, address _remoteToken, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2ERC721Bridge *L2ERC721BridgeSession) BridgeERC721(_localToken common.Address, _remoteToken common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.BridgeERC721(&_L2ERC721Bridge.TransactOpts, _localToken, _remoteToken, _tokenId, _minGasLimit, _extraData)
}

// BridgeERC721 is a paid mutator transaction binding the contract method 0x3687011a.
//
// Solidity: function bridgeERC721(address _localToken, address _remoteToken, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2ERC721Bridge *L2ERC721BridgeTransactorSession) BridgeERC721(_localToken common.Address, _remoteToken common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.BridgeERC721(&_L2ERC721Bridge.TransactOpts, _localToken, _remoteToken, _tokenId, _minGasLimit, _extraData)
}

// BridgeERC721To is a paid mutator transaction binding the contract method 0xaa557452.
//
// Solidity: function bridgeERC721To(address _localToken, address _remoteToken, address _to, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2ERC721Bridge *L2ERC721BridgeTransactor) BridgeERC721To(opts *bind.TransactOpts, _localToken common.Address, _remoteToken common.Address, _to common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2ERC721Bridge.contract.Transact(opts, "bridgeERC721To", _localToken, _remoteToken, _to, _tokenId, _minGasLimit, _extraData)
}

// BridgeERC721To is a paid mutator transaction binding the contract method 0xaa557452.
//
// Solidity: function bridgeERC721To(address _localToken, address _remoteToken, address _to, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2ERC721Bridge *L2ERC721BridgeSession) BridgeERC721To(_localToken common.Address, _remoteToken common.Address, _to common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.BridgeERC721To(&_L2ERC721Bridge.TransactOpts, _localToken, _remoteToken, _to, _tokenId, _minGasLimit, _extraData)
}

// BridgeERC721To is a paid mutator transaction binding the contract method 0xaa557452.
//
// Solidity: function bridgeERC721To(address _localToken, address _remoteToken, address _to, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L2ERC721Bridge *L2ERC721BridgeTransactorSession) BridgeERC721To(_localToken common.Address, _remoteToken common.Address, _to common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.BridgeERC721To(&_L2ERC721Bridge.TransactOpts, _localToken, _remoteToken, _to, _tokenId, _minGasLimit, _extraData)
}

// FinalizeBridgeERC721 is a paid mutator transaction binding the contract method 0x761f4493.
//
// Solidity: function finalizeBridgeERC721(address _localToken, address _remoteToken, address _from, address _to, uint256 _tokenId, bytes _extraData) returns()
func (_L2ERC721Bridge *L2ERC721BridgeTransactor) FinalizeBridgeERC721(opts *bind.TransactOpts, _localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _tokenId *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2ERC721Bridge.contract.Transact(opts, "finalizeBridgeERC721", _localToken, _remoteToken, _from, _to, _tokenId, _extraData)
}

// FinalizeBridgeERC721 is a paid mutator transaction binding the contract method 0x761f4493.
//
// Solidity: function finalizeBridgeERC721(address _localToken, address _remoteToken, address _from, address _to, uint256 _tokenId, bytes _extraData) returns()
func (_L2ERC721Bridge *L2ERC721BridgeSession) FinalizeBridgeERC721(_localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _tokenId *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.FinalizeBridgeERC721(&_L2ERC721Bridge.TransactOpts, _localToken, _remoteToken, _from, _to, _tokenId, _extraData)
}

// FinalizeBridgeERC721 is a paid mutator transaction binding the contract method 0x761f4493.
//
// Solidity: function finalizeBridgeERC721(address _localToken, address _remoteToken, address _from, address _to, uint256 _tokenId, bytes _extraData) returns()
func (_L2ERC721Bridge *L2ERC721BridgeTransactorSession) FinalizeBridgeERC721(_localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _tokenId *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.FinalizeBridgeERC721(&_L2ERC721Bridge.TransactOpts, _localToken, _remoteToken, _from, _to, _tokenId, _extraData)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _l1ERC721Bridge) returns()
func (_L2ERC721Bridge *L2ERC721BridgeTransactor) Initialize(opts *bind.TransactOpts, _l1ERC721Bridge common.Address) (*types.Transaction, error) {
	return _L2ERC721Bridge.contract.Transact(opts, "initialize", _l1ERC721Bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _l1ERC721Bridge) returns()
func (_L2ERC721Bridge *L2ERC721BridgeSession) Initialize(_l1ERC721Bridge common.Address) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.Initialize(&_L2ERC721Bridge.TransactOpts, _l1ERC721Bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _l1ERC721Bridge) returns()
func (_L2ERC721Bridge *L2ERC721BridgeTransactorSession) Initialize(_l1ERC721Bridge common.Address) (*types.Transaction, error) {
	return _L2ERC721Bridge.Contract.Initialize(&_L2ERC721Bridge.TransactOpts, _l1ERC721Bridge)
}

// L2ERC721BridgeERC721BridgeFinalizedIterator is returned from FilterERC721BridgeFinalized and is used to iterate over the raw logs and unpacked data for ERC721BridgeFinalized events raised by the L2ERC721Bridge contract.
type L2ERC721BridgeERC721BridgeFinalizedIterator struct {
	Event *L2ERC721BridgeERC721BridgeFinalized // Event containing the contract specifics and raw log

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
func (it *L2ERC721BridgeERC721BridgeFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2ERC721BridgeERC721BridgeFinalized)
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
		it.Event = new(L2ERC721BridgeERC721BridgeFinalized)
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
func (it *L2ERC721BridgeERC721BridgeFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2ERC721BridgeERC721BridgeFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2ERC721BridgeERC721BridgeFinalized represents a ERC721BridgeFinalized event raised by the L2ERC721Bridge contract.
type L2ERC721BridgeERC721BridgeFinalized struct {
	LocalToken  common.Address
	RemoteToken common.Address
	From        common.Address
	To          common.Address
	TokenId     *big.Int
	ExtraData   []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterERC721BridgeFinalized is a free log retrieval operation binding the contract event 0x1f39bf6707b5d608453e0ae4c067b562bcc4c85c0f562ef5d2c774d2e7f131ac.
//
// Solidity: event ERC721BridgeFinalized(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L2ERC721Bridge *L2ERC721BridgeFilterer) FilterERC721BridgeFinalized(opts *bind.FilterOpts, localToken []common.Address, remoteToken []common.Address, from []common.Address) (*L2ERC721BridgeERC721BridgeFinalizedIterator, error) {

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

	logs, sub, err := _L2ERC721Bridge.contract.FilterLogs(opts, "ERC721BridgeFinalized", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return &L2ERC721BridgeERC721BridgeFinalizedIterator{contract: _L2ERC721Bridge.contract, event: "ERC721BridgeFinalized", logs: logs, sub: sub}, nil
}

// WatchERC721BridgeFinalized is a free log subscription operation binding the contract event 0x1f39bf6707b5d608453e0ae4c067b562bcc4c85c0f562ef5d2c774d2e7f131ac.
//
// Solidity: event ERC721BridgeFinalized(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L2ERC721Bridge *L2ERC721BridgeFilterer) WatchERC721BridgeFinalized(opts *bind.WatchOpts, sink chan<- *L2ERC721BridgeERC721BridgeFinalized, localToken []common.Address, remoteToken []common.Address, from []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _L2ERC721Bridge.contract.WatchLogs(opts, "ERC721BridgeFinalized", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2ERC721BridgeERC721BridgeFinalized)
				if err := _L2ERC721Bridge.contract.UnpackLog(event, "ERC721BridgeFinalized", log); err != nil {
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

// ParseERC721BridgeFinalized is a log parse operation binding the contract event 0x1f39bf6707b5d608453e0ae4c067b562bcc4c85c0f562ef5d2c774d2e7f131ac.
//
// Solidity: event ERC721BridgeFinalized(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L2ERC721Bridge *L2ERC721BridgeFilterer) ParseERC721BridgeFinalized(log types.Log) (*L2ERC721BridgeERC721BridgeFinalized, error) {
	event := new(L2ERC721BridgeERC721BridgeFinalized)
	if err := _L2ERC721Bridge.contract.UnpackLog(event, "ERC721BridgeFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2ERC721BridgeERC721BridgeInitiatedIterator is returned from FilterERC721BridgeInitiated and is used to iterate over the raw logs and unpacked data for ERC721BridgeInitiated events raised by the L2ERC721Bridge contract.
type L2ERC721BridgeERC721BridgeInitiatedIterator struct {
	Event *L2ERC721BridgeERC721BridgeInitiated // Event containing the contract specifics and raw log

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
func (it *L2ERC721BridgeERC721BridgeInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2ERC721BridgeERC721BridgeInitiated)
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
		it.Event = new(L2ERC721BridgeERC721BridgeInitiated)
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
func (it *L2ERC721BridgeERC721BridgeInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2ERC721BridgeERC721BridgeInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2ERC721BridgeERC721BridgeInitiated represents a ERC721BridgeInitiated event raised by the L2ERC721Bridge contract.
type L2ERC721BridgeERC721BridgeInitiated struct {
	LocalToken  common.Address
	RemoteToken common.Address
	From        common.Address
	To          common.Address
	TokenId     *big.Int
	ExtraData   []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterERC721BridgeInitiated is a free log retrieval operation binding the contract event 0xb7460e2a880f256ebef3406116ff3eee0cee51ebccdc2a40698f87ebb2e9c1a5.
//
// Solidity: event ERC721BridgeInitiated(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L2ERC721Bridge *L2ERC721BridgeFilterer) FilterERC721BridgeInitiated(opts *bind.FilterOpts, localToken []common.Address, remoteToken []common.Address, from []common.Address) (*L2ERC721BridgeERC721BridgeInitiatedIterator, error) {

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

	logs, sub, err := _L2ERC721Bridge.contract.FilterLogs(opts, "ERC721BridgeInitiated", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return &L2ERC721BridgeERC721BridgeInitiatedIterator{contract: _L2ERC721Bridge.contract, event: "ERC721BridgeInitiated", logs: logs, sub: sub}, nil
}

// WatchERC721BridgeInitiated is a free log subscription operation binding the contract event 0xb7460e2a880f256ebef3406116ff3eee0cee51ebccdc2a40698f87ebb2e9c1a5.
//
// Solidity: event ERC721BridgeInitiated(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L2ERC721Bridge *L2ERC721BridgeFilterer) WatchERC721BridgeInitiated(opts *bind.WatchOpts, sink chan<- *L2ERC721BridgeERC721BridgeInitiated, localToken []common.Address, remoteToken []common.Address, from []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _L2ERC721Bridge.contract.WatchLogs(opts, "ERC721BridgeInitiated", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2ERC721BridgeERC721BridgeInitiated)
				if err := _L2ERC721Bridge.contract.UnpackLog(event, "ERC721BridgeInitiated", log); err != nil {
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

// ParseERC721BridgeInitiated is a log parse operation binding the contract event 0xb7460e2a880f256ebef3406116ff3eee0cee51ebccdc2a40698f87ebb2e9c1a5.
//
// Solidity: event ERC721BridgeInitiated(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L2ERC721Bridge *L2ERC721BridgeFilterer) ParseERC721BridgeInitiated(log types.Log) (*L2ERC721BridgeERC721BridgeInitiated, error) {
	event := new(L2ERC721BridgeERC721BridgeInitiated)
	if err := _L2ERC721Bridge.contract.UnpackLog(event, "ERC721BridgeInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2ERC721BridgeInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L2ERC721Bridge contract.
type L2ERC721BridgeInitializedIterator struct {
	Event *L2ERC721BridgeInitialized // Event containing the contract specifics and raw log

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
func (it *L2ERC721BridgeInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2ERC721BridgeInitialized)
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
		it.Event = new(L2ERC721BridgeInitialized)
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
func (it *L2ERC721BridgeInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2ERC721BridgeInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2ERC721BridgeInitialized represents a Initialized event raised by the L2ERC721Bridge contract.
type L2ERC721BridgeInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2ERC721Bridge *L2ERC721BridgeFilterer) FilterInitialized(opts *bind.FilterOpts) (*L2ERC721BridgeInitializedIterator, error) {

	logs, sub, err := _L2ERC721Bridge.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L2ERC721BridgeInitializedIterator{contract: _L2ERC721Bridge.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2ERC721Bridge *L2ERC721BridgeFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L2ERC721BridgeInitialized) (event.Subscription, error) {

	logs, sub, err := _L2ERC721Bridge.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2ERC721BridgeInitialized)
				if err := _L2ERC721Bridge.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_L2ERC721Bridge *L2ERC721BridgeFilterer) ParseInitialized(log types.Log) (*L2ERC721BridgeInitialized, error) {
	event := new(L2ERC721BridgeInitialized)
	if err := _L2ERC721Bridge.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
