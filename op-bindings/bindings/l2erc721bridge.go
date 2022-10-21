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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_messenger\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_otherBridge\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ERC721BridgeFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ERC721BridgeInitiated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_tokenId\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeERC721\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_tokenId\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeERC721To\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_tokenId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"finalizeBridgeERC721\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"otherBridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101206040523480156200001257600080fd5b50604051620018a0380380620018a0833981016040819052620000359162000162565b600080600184846001600160a01b038216620000ad5760405162461bcd60e51b815260206004820152602c60248201527f4552433732314272696467653a206d657373656e6765722063616e6e6f74206260448201526b65206164647265737328302960a01b60648201526084015b60405180910390fd5b6001600160a01b0381166200011d5760405162461bcd60e51b815260206004820152602f60248201527f4552433732314272696467653a206f74686572206272696467652063616e6e6f60448201526e74206265206164647265737328302960881b6064820152608401620000a4565b6001600160a01b039182166080521660a05260c09290925260e05261010052506200019a9050565b80516001600160a01b03811681146200015d57600080fd5b919050565b600080604083850312156200017657600080fd5b620001818362000145565b9150620001916020840162000145565b90509250929050565b60805160a05160c05160e0516101005161169f6200020160003960006102440152600061021b015260006101f201526000818161011d015281816102d00152610d630152600081816091015281816102a6015281816103070152610d36015261169f6000f3fe608060405234801561001057600080fd5b50600436106100725760003560e01c8063761f449311610050578063761f4493146100f2578063aa55745214610105578063c89701a21461011857600080fd5b80633687011a146100775780633cb747bf1461008c57806354fd4d50146100dd575b600080fd5b61008a61008536600461116a565b61013f565b005b6100b37f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b6100e56101eb565b6040516100d49190611267565b61008a61010036600461127a565b61028e565b61008a610113366004611312565b6107f5565b6100b37f000000000000000000000000000000000000000000000000000000000000000081565b333b156101d3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602d60248201527f4552433732314272696467653a206163636f756e74206973206e6f742065787460448201527f65726e616c6c79206f776e65640000000000000000000000000000000000000060648201526084015b60405180910390fd5b6101e386863333888888886108b1565b505050505050565b60606102167f0000000000000000000000000000000000000000000000000000000000000000610e4f565b61023f7f0000000000000000000000000000000000000000000000000000000000000000610e4f565b6102687f0000000000000000000000000000000000000000000000000000000000000000610e4f565b60405160200161027a93929190611389565b604051602081830303815290604052905090565b3373ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000161480156103ac57507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b8152600401602060405180830381865afa158015610370573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061039491906113ff565b73ffffffffffffffffffffffffffffffffffffffff16145b610438576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603f60248201527f4552433732314272696467653a2066756e6374696f6e2063616e206f6e6c792060448201527f62652063616c6c65642066726f6d20746865206f74686572206272696467650060648201526084016101ca565b3073ffffffffffffffffffffffffffffffffffffffff8816036104dd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f4c324552433732314272696467653a206c6f63616c20746f6b656e2063616e6e60448201527f6f742062652073656c660000000000000000000000000000000000000000000060648201526084016101ca565b610507877fe49bc7f800000000000000000000000000000000000000000000000000000000610f8c565b610593576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f4c324552433732314272696467653a206c6f63616c20746f6b656e20696e746560448201527f7266616365206973206e6f7420636f6d706c69616e740000000000000000000060648201526084016101ca565b8673ffffffffffffffffffffffffffffffffffffffff1663d6c0b2c46040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105de573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061060291906113ff565b73ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff16146106e2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604b60248201527f4c324552433732314272696467653a2077726f6e672072656d6f746520746f6b60448201527f656e20666f72204f7074696d69736d204d696e7461626c65204552433732312060648201527f6c6f63616c20746f6b656e000000000000000000000000000000000000000000608482015260a4016101ca565b6040517fa144819400000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff85811660048301526024820185905288169063a144819490604401600060405180830381600087803b15801561075257600080fd5b505af1158015610766573d6000803e3d6000fd5b505050508473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff167f1f39bf6707b5d608453e0ae4c067b562bcc4c85c0f562ef5d2c774d2e7f131ac878787876040516107e49493929190611465565b60405180910390a450505050505050565b73ffffffffffffffffffffffffffffffffffffffff8516610898576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f4552433732314272696467653a206e667420726563697069656e742063616e6e60448201527f6f7420626520616464726573732830290000000000000000000000000000000060648201526084016101ca565b6108a887873388888888886108b1565b50505050505050565b73ffffffffffffffffffffffffffffffffffffffff8716610954576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f4552433732314272696467653a2072656d6f746520746f6b656e2063616e6e6f60448201527f742062652061646472657373283029000000000000000000000000000000000060648201526084016101ca565b6040517f6352211e0000000000000000000000000000000000000000000000000000000081526004810185905273ffffffffffffffffffffffffffffffffffffffff891690636352211e90602401602060405180830381865afa1580156109bf573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906109e391906113ff565b73ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff1614610a9d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f5769746864726177616c206973206e6f74206265696e6720696e69746961746560448201527f64206279204e4654206f776e657200000000000000000000000000000000000060648201526084016101ca565b60008873ffffffffffffffffffffffffffffffffffffffff1663d6c0b2c46040518163ffffffff1660e01b8152600401602060405180830381865afa158015610aea573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b0e91906113ff565b90508773ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614610bcb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f4c324552433732314272696467653a2072656d6f746520746f6b656e20646f6560448201527f73206e6f74206d6174636820676976656e2076616c756500000000000000000060648201526084016101ca565b6040517f9dc29fac00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8881166004830152602482018790528a1690639dc29fac90604401600060405180830381600087803b158015610c3b57600080fd5b505af1158015610c4f573d6000803e3d6000fd5b50505050600063761f449360e01b828b8a8a8a8989604051602401610c7a97969594939291906114a5565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009094169390931790925290517f3dbb202b00000000000000000000000000000000000000000000000000000000815290915073ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001690633dbb202b90610d8f907f00000000000000000000000000000000000000000000000000000000000000009085908a90600401611502565b600060405180830381600087803b158015610da957600080fd5b505af1158015610dbd573d6000803e3d6000fd5b505050508773ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff168b73ffffffffffffffffffffffffffffffffffffffff167fb7460e2a880f256ebef3406116ff3eee0cee51ebccdc2a40698f87ebb2e9c1a58a8a8989604051610e3b9493929190611465565b60405180910390a450505050505050505050565b606081600003610e9257505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115610ebc5780610ea681611576565b9150610eb59050600a836115dd565b9150610e96565b60008167ffffffffffffffff811115610ed757610ed76115f1565b6040519080825280601f01601f191660200182016040528015610f01576020820181803683370190505b5090505b8415610f8457610f16600183611620565b9150610f23600a86611637565b610f2e90603061164b565b60f81b818381518110610f4357610f43611663565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610f7d600a866115dd565b9450610f05565b949350505050565b6000610f9783610faf565b8015610fa85750610fa88383611014565b9392505050565b6000610fdb827f01ffc9a700000000000000000000000000000000000000000000000000000000611014565b801561100e575061100c827fffffffff00000000000000000000000000000000000000000000000000000000611014565b155b92915050565b604080517fffffffff000000000000000000000000000000000000000000000000000000008316602480830191909152825180830390910181526044909101909152602080820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01ffc9a700000000000000000000000000000000000000000000000000000000178152825160009392849283928392918391908a617530fa92503d915060005190508280156110cc575060208210155b80156110d85750600081115b979650505050505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461110557600080fd5b50565b803563ffffffff8116811461111c57600080fd5b919050565b60008083601f84011261113357600080fd5b50813567ffffffffffffffff81111561114b57600080fd5b60208301915083602082850101111561116357600080fd5b9250929050565b60008060008060008060a0878903121561118357600080fd5b863561118e816110e3565b9550602087013561119e816110e3565b9450604087013593506111b360608801611108565b9250608087013567ffffffffffffffff8111156111cf57600080fd5b6111db89828a01611121565b979a9699509497509295939492505050565b60005b838110156112085781810151838201526020016111f0565b83811115611217576000848401525b50505050565b600081518084526112358160208601602086016111ed565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000610fa8602083018461121d565b600080600080600080600060c0888a03121561129557600080fd5b87356112a0816110e3565b965060208801356112b0816110e3565b955060408801356112c0816110e3565b945060608801356112d0816110e3565b93506080880135925060a088013567ffffffffffffffff8111156112f357600080fd5b6112ff8a828b01611121565b989b979a50959850939692959293505050565b600080600080600080600060c0888a03121561132d57600080fd5b8735611338816110e3565b96506020880135611348816110e3565b95506040880135611358816110e3565b94506060880135935061136d60808901611108565b925060a088013567ffffffffffffffff8111156112f357600080fd5b6000845161139b8184602089016111ed565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516113d7816001850160208a016111ed565b600192019182015283516113f28160028401602088016111ed565b0160020195945050505050565b60006020828403121561141157600080fd5b8151610fa8816110e3565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b73ffffffffffffffffffffffffffffffffffffffff8516815283602082015260606040820152600061149b60608301848661141c565b9695505050505050565b600073ffffffffffffffffffffffffffffffffffffffff808a1683528089166020840152808816604084015280871660608401525084608083015260c060a08301526114f560c08301848661141c565b9998505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff84168152606060208201526000611531606083018561121d565b905063ffffffff83166040830152949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036115a7576115a7611547565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b6000826115ec576115ec6115ae565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60008282101561163257611632611547565b500390565b600082611646576116466115ae565b500690565b6000821982111561165e5761165e611547565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a",
}

// L2ERC721BridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use L2ERC721BridgeMetaData.ABI instead.
var L2ERC721BridgeABI = L2ERC721BridgeMetaData.ABI

// L2ERC721BridgeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2ERC721BridgeMetaData.Bin instead.
var L2ERC721BridgeBin = L2ERC721BridgeMetaData.Bin

// DeployL2ERC721Bridge deploys a new Ethereum contract, binding an instance of L2ERC721Bridge to it.
func DeployL2ERC721Bridge(auth *bind.TransactOpts, backend bind.ContractBackend, _messenger common.Address, _otherBridge common.Address) (common.Address, *types.Transaction, *L2ERC721Bridge, error) {
	parsed, err := L2ERC721BridgeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2ERC721BridgeBin), backend, _messenger, _otherBridge)
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
