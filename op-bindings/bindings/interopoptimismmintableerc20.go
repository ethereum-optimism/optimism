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

// InteropOptimismMintableERC20MetaData contains all meta data concerning the InteropOptimismMintableERC20 contract.
var InteropOptimismMintableERC20MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"_decimals\",\"type\":\"uint8\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Burn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Mint\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"BRIDGE\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"REMOTE_TOKEN\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1Token\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2Bridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"remoteToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"_interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60e06040523480156200001157600080fd5b506040516200176d3803806200176d833981016040819052620000349162000164565b7342000000000000000000000000000000000000e384848484828260036200005d838262000297565b5060046200006c828262000297565b5050506001600160a01b039384166080529390921660a052505060ff1660c052506200036392505050565b634e487b7160e01b600052604160045260246000fd5b600082601f830112620000bf57600080fd5b81516001600160401b0380821115620000dc57620000dc62000097565b604051601f8301601f19908116603f0116810190828211818310171562000107576200010762000097565b816040528381526020925086838588010111156200012457600080fd5b600091505b8382101562000148578582018301518183018401529082019062000129565b838211156200015a5760008385830101525b9695505050505050565b600080600080608085870312156200017b57600080fd5b84516001600160a01b03811681146200019357600080fd5b60208601519094506001600160401b0380821115620001b157600080fd5b620001bf88838901620000ad565b94506040870151915080821115620001d657600080fd5b50620001e587828801620000ad565b925050606085015160ff81168114620001fd57600080fd5b939692955090935050565b600181811c908216806200021d57607f821691505b6020821081036200023e57634e487b7160e01b600052602260045260246000fd5b50919050565b601f8211156200029257600081815260208120601f850160051c810160208610156200026d5750805b601f850160051c820191505b818110156200028e5782815560010162000279565b5050505b505050565b81516001600160401b03811115620002b357620002b362000097565b620002cb81620002c4845462000208565b8462000244565b602080601f831160018114620003035760008415620002ea5750858301515b600019600386901b1c1916600185901b1785556200028e565b600085815260208120601f198616915b82811015620003345788860151825594840194600190910190840162000313565b5085821015620003535787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b60805160a05160c0516113cc620003a1600039600061024401526000818161034b01526103e00152600081816101a9015261037101526113cc6000f3fe608060405234801561001057600080fd5b50600436106101775760003560e01c806370a08231116100d8578063ae1f6aaf1161008c578063dd62ed3e11610066578063dd62ed3e14610395578063e78cea9214610349578063ee9a31a2146103db57600080fd5b8063ae1f6aaf14610349578063c01e1bd61461036f578063d6c0b2c41461036f57600080fd5b80639dc29fac116100bd5780639dc29fac14610310578063a457c2d714610323578063a9059cbb1461033657600080fd5b806370a08231146102d257806395d89b411461030857600080fd5b806323b872dd1161012f5780633950935111610114578063395093511461026e57806340c10f191461028157806354fd4d501461029657600080fd5b806323b872dd1461022a578063313ce5671461023d57600080fd5b806306fdde031161016057806306fdde03146101f0578063095ea7b31461020557806318160ddd1461021857600080fd5b806301ffc9a71461017c578063033964be146101a4575b600080fd5b61018f61018a366004611175565b610402565b60405190151581526020015b60405180910390f35b6101cb7f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200161019b565b6101f86104f3565b60405161019b91906111be565b61018f61021336600461125a565b610585565b6002545b60405190815260200161019b565b61018f610238366004611284565b61059d565b60405160ff7f000000000000000000000000000000000000000000000000000000000000000016815260200161019b565b61018f61027c36600461125a565b6105c1565b61029461028f36600461125a565b61060d565b005b6101f86040518060400160405280600581526020017f312e332e3000000000000000000000000000000000000000000000000000000081525081565b61021c6102e03660046112c0565b73ffffffffffffffffffffffffffffffffffffffff1660009081526020819052604090205490565b6101f8610731565b61029461031e36600461125a565b610740565b61018f61033136600461125a565b610853565b61018f61034436600461125a565b610924565b7f00000000000000000000000000000000000000000000000000000000000000006101cb565b7f00000000000000000000000000000000000000000000000000000000000000006101cb565b61021c6103a33660046112db565b73ffffffffffffffffffffffffffffffffffffffff918216600090815260016020908152604080832093909416825291909152205490565b6101cb7f000000000000000000000000000000000000000000000000000000000000000081565b60007f01ffc9a7000000000000000000000000000000000000000000000000000000007f1d1d8b63000000000000000000000000000000000000000000000000000000007fec4fc8e3000000000000000000000000000000000000000000000000000000007fffffffff0000000000000000000000000000000000000000000000000000000085168314806104bb57507fffffffff00000000000000000000000000000000000000000000000000000000858116908316145b806104ea57507fffffffff00000000000000000000000000000000000000000000000000000000858116908216145b95945050505050565b6060600380546105029061130e565b80601f016020809104026020016040519081016040528092919081815260200182805461052e9061130e565b801561057b5780601f106105505761010080835404028352916020019161057b565b820191906000526020600020905b81548152906001019060200180831161055e57829003601f168201915b5050505050905090565b600033610593818585610932565b5060019392505050565b6000336105ab858285610ae6565b6105b6858585610bbd565b506001949350505050565b33600081815260016020908152604080832073ffffffffffffffffffffffffffffffffffffffff871684529091528120549091906105939082908690610608908790611390565b610932565b3373420000000000000000000000000000000000001014806106425750337342000000000000000000000000000000000000e3145b6106d3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603b60248201527f496e7465726f704f7074696d69736d4d696e7461626c6545524332303a206f6e60448201527f6c79206272696467652063616e206d696e7420616e64206275726e000000000060648201526084015b60405180910390fd5b6106dd8282610e70565b8173ffffffffffffffffffffffffffffffffffffffff167f0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d41213968858260405161072591815260200190565b60405180910390a25050565b6060600480546105029061130e565b3373420000000000000000000000000000000000001014806107755750337342000000000000000000000000000000000000e3145b610801576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603b60248201527f496e7465726f704f7074696d69736d4d696e7461626c6545524332303a206f6e60448201527f6c79206272696467652063616e206d696e7420616e64206275726e000000000060648201526084016106ca565b61080b8282610f90565b8173ffffffffffffffffffffffffffffffffffffffff167fcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca58260405161072591815260200190565b33600081815260016020908152604080832073ffffffffffffffffffffffffffffffffffffffff8716845290915281205490919083811015610917576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760448201527f207a65726f00000000000000000000000000000000000000000000000000000060648201526084016106ca565b6105b68286868403610932565b600033610593818585610bbd565b73ffffffffffffffffffffffffffffffffffffffff83166109d4576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526024808201527f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460448201527f726573730000000000000000000000000000000000000000000000000000000060648201526084016106ca565b73ffffffffffffffffffffffffffffffffffffffff8216610a77576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a20617070726f766520746f20746865207a65726f20616464726560448201527f737300000000000000000000000000000000000000000000000000000000000060648201526084016106ca565b73ffffffffffffffffffffffffffffffffffffffff83811660008181526001602090815260408083209487168084529482529182902085905590518481527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92591015b60405180910390a3505050565b73ffffffffffffffffffffffffffffffffffffffff8381166000908152600160209081526040808320938616835292905220547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8114610bb75781811015610baa576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f45524332303a20696e73756666696369656e7420616c6c6f77616e636500000060448201526064016106ca565b610bb78484848403610932565b50505050565b73ffffffffffffffffffffffffffffffffffffffff8316610c60576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a207472616e736665722066726f6d20746865207a65726f20616460448201527f647265737300000000000000000000000000000000000000000000000000000060648201526084016106ca565b73ffffffffffffffffffffffffffffffffffffffff8216610d03576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f45524332303a207472616e7366657220746f20746865207a65726f206164647260448201527f657373000000000000000000000000000000000000000000000000000000000060648201526084016106ca565b73ffffffffffffffffffffffffffffffffffffffff831660009081526020819052604090205481811015610db9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f45524332303a207472616e7366657220616d6f756e742065786365656473206260448201527f616c616e6365000000000000000000000000000000000000000000000000000060648201526084016106ca565b73ffffffffffffffffffffffffffffffffffffffff808516600090815260208190526040808220858503905591851681529081208054849290610dfd908490611390565b925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef84604051610e6391815260200190565b60405180910390a3610bb7565b73ffffffffffffffffffffffffffffffffffffffff8216610eed576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f20616464726573730060448201526064016106ca565b8060026000828254610eff9190611390565b909155505073ffffffffffffffffffffffffffffffffffffffff821660009081526020819052604081208054839290610f39908490611390565b909155505060405181815273ffffffffffffffffffffffffffffffffffffffff8316906000907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9060200160405180910390a35050565b73ffffffffffffffffffffffffffffffffffffffff8216611033576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602160248201527f45524332303a206275726e2066726f6d20746865207a65726f2061646472657360448201527f730000000000000000000000000000000000000000000000000000000000000060648201526084016106ca565b73ffffffffffffffffffffffffffffffffffffffff8216600090815260208190526040902054818110156110e9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a206275726e20616d6f756e7420657863656564732062616c616e60448201527f636500000000000000000000000000000000000000000000000000000000000060648201526084016106ca565b73ffffffffffffffffffffffffffffffffffffffff831660009081526020819052604081208383039055600280548492906111259084906113a8565b909155505060405182815260009073ffffffffffffffffffffffffffffffffffffffff8516907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef90602001610ad9565b60006020828403121561118757600080fd5b81357fffffffff00000000000000000000000000000000000000000000000000000000811681146111b757600080fd5b9392505050565b600060208083528351808285015260005b818110156111eb578581018301518582016040015282016111cf565b818111156111fd576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461125557600080fd5b919050565b6000806040838503121561126d57600080fd5b61127683611231565b946020939093013593505050565b60008060006060848603121561129957600080fd5b6112a284611231565b92506112b060208501611231565b9150604084013590509250925092565b6000602082840312156112d257600080fd5b6111b782611231565b600080604083850312156112ee57600080fd5b6112f783611231565b915061130560208401611231565b90509250929050565b600181811c9082168061132257607f821691505b60208210810361135b577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082198211156113a3576113a3611361565b500190565b6000828210156113ba576113ba611361565b50039056fea164736f6c634300080f000a",
}

// InteropOptimismMintableERC20ABI is the input ABI used to generate the binding from.
// Deprecated: Use InteropOptimismMintableERC20MetaData.ABI instead.
var InteropOptimismMintableERC20ABI = InteropOptimismMintableERC20MetaData.ABI

// InteropOptimismMintableERC20Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use InteropOptimismMintableERC20MetaData.Bin instead.
var InteropOptimismMintableERC20Bin = InteropOptimismMintableERC20MetaData.Bin

// DeployInteropOptimismMintableERC20 deploys a new Ethereum contract, binding an instance of InteropOptimismMintableERC20 to it.
func DeployInteropOptimismMintableERC20(auth *bind.TransactOpts, backend bind.ContractBackend, _remoteToken common.Address, _name string, _symbol string, _decimals uint8) (common.Address, *types.Transaction, *InteropOptimismMintableERC20, error) {
	parsed, err := InteropOptimismMintableERC20MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(InteropOptimismMintableERC20Bin), backend, _remoteToken, _name, _symbol, _decimals)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &InteropOptimismMintableERC20{InteropOptimismMintableERC20Caller: InteropOptimismMintableERC20Caller{contract: contract}, InteropOptimismMintableERC20Transactor: InteropOptimismMintableERC20Transactor{contract: contract}, InteropOptimismMintableERC20Filterer: InteropOptimismMintableERC20Filterer{contract: contract}}, nil
}

// InteropOptimismMintableERC20 is an auto generated Go binding around an Ethereum contract.
type InteropOptimismMintableERC20 struct {
	InteropOptimismMintableERC20Caller     // Read-only binding to the contract
	InteropOptimismMintableERC20Transactor // Write-only binding to the contract
	InteropOptimismMintableERC20Filterer   // Log filterer for contract events
}

// InteropOptimismMintableERC20Caller is an auto generated read-only Go binding around an Ethereum contract.
type InteropOptimismMintableERC20Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InteropOptimismMintableERC20Transactor is an auto generated write-only Go binding around an Ethereum contract.
type InteropOptimismMintableERC20Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InteropOptimismMintableERC20Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InteropOptimismMintableERC20Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InteropOptimismMintableERC20Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InteropOptimismMintableERC20Session struct {
	Contract     *InteropOptimismMintableERC20 // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                 // Call options to use throughout this session
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// InteropOptimismMintableERC20CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InteropOptimismMintableERC20CallerSession struct {
	Contract *InteropOptimismMintableERC20Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                       // Call options to use throughout this session
}

// InteropOptimismMintableERC20TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InteropOptimismMintableERC20TransactorSession struct {
	Contract     *InteropOptimismMintableERC20Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                       // Transaction auth options to use throughout this session
}

// InteropOptimismMintableERC20Raw is an auto generated low-level Go binding around an Ethereum contract.
type InteropOptimismMintableERC20Raw struct {
	Contract *InteropOptimismMintableERC20 // Generic contract binding to access the raw methods on
}

// InteropOptimismMintableERC20CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InteropOptimismMintableERC20CallerRaw struct {
	Contract *InteropOptimismMintableERC20Caller // Generic read-only contract binding to access the raw methods on
}

// InteropOptimismMintableERC20TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InteropOptimismMintableERC20TransactorRaw struct {
	Contract *InteropOptimismMintableERC20Transactor // Generic write-only contract binding to access the raw methods on
}

// NewInteropOptimismMintableERC20 creates a new instance of InteropOptimismMintableERC20, bound to a specific deployed contract.
func NewInteropOptimismMintableERC20(address common.Address, backend bind.ContractBackend) (*InteropOptimismMintableERC20, error) {
	contract, err := bindInteropOptimismMintableERC20(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &InteropOptimismMintableERC20{InteropOptimismMintableERC20Caller: InteropOptimismMintableERC20Caller{contract: contract}, InteropOptimismMintableERC20Transactor: InteropOptimismMintableERC20Transactor{contract: contract}, InteropOptimismMintableERC20Filterer: InteropOptimismMintableERC20Filterer{contract: contract}}, nil
}

// NewInteropOptimismMintableERC20Caller creates a new read-only instance of InteropOptimismMintableERC20, bound to a specific deployed contract.
func NewInteropOptimismMintableERC20Caller(address common.Address, caller bind.ContractCaller) (*InteropOptimismMintableERC20Caller, error) {
	contract, err := bindInteropOptimismMintableERC20(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InteropOptimismMintableERC20Caller{contract: contract}, nil
}

// NewInteropOptimismMintableERC20Transactor creates a new write-only instance of InteropOptimismMintableERC20, bound to a specific deployed contract.
func NewInteropOptimismMintableERC20Transactor(address common.Address, transactor bind.ContractTransactor) (*InteropOptimismMintableERC20Transactor, error) {
	contract, err := bindInteropOptimismMintableERC20(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InteropOptimismMintableERC20Transactor{contract: contract}, nil
}

// NewInteropOptimismMintableERC20Filterer creates a new log filterer instance of InteropOptimismMintableERC20, bound to a specific deployed contract.
func NewInteropOptimismMintableERC20Filterer(address common.Address, filterer bind.ContractFilterer) (*InteropOptimismMintableERC20Filterer, error) {
	contract, err := bindInteropOptimismMintableERC20(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InteropOptimismMintableERC20Filterer{contract: contract}, nil
}

// bindInteropOptimismMintableERC20 binds a generic wrapper to an already deployed contract.
func bindInteropOptimismMintableERC20(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(InteropOptimismMintableERC20ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InteropOptimismMintableERC20.Contract.InteropOptimismMintableERC20Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.InteropOptimismMintableERC20Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.InteropOptimismMintableERC20Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InteropOptimismMintableERC20.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.contract.Transact(opts, method, params...)
}

// BRIDGE is a free data retrieval call binding the contract method 0xee9a31a2.
//
// Solidity: function BRIDGE() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) BRIDGE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "BRIDGE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BRIDGE is a free data retrieval call binding the contract method 0xee9a31a2.
//
// Solidity: function BRIDGE() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) BRIDGE() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.BRIDGE(&_InteropOptimismMintableERC20.CallOpts)
}

// BRIDGE is a free data retrieval call binding the contract method 0xee9a31a2.
//
// Solidity: function BRIDGE() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) BRIDGE() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.BRIDGE(&_InteropOptimismMintableERC20.CallOpts)
}

// REMOTETOKEN is a free data retrieval call binding the contract method 0x033964be.
//
// Solidity: function REMOTE_TOKEN() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) REMOTETOKEN(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "REMOTE_TOKEN")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// REMOTETOKEN is a free data retrieval call binding the contract method 0x033964be.
//
// Solidity: function REMOTE_TOKEN() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) REMOTETOKEN() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.REMOTETOKEN(&_InteropOptimismMintableERC20.CallOpts)
}

// REMOTETOKEN is a free data retrieval call binding the contract method 0x033964be.
//
// Solidity: function REMOTE_TOKEN() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) REMOTETOKEN() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.REMOTETOKEN(&_InteropOptimismMintableERC20.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _InteropOptimismMintableERC20.Contract.Allowance(&_InteropOptimismMintableERC20.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _InteropOptimismMintableERC20.Contract.Allowance(&_InteropOptimismMintableERC20.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) BalanceOf(account common.Address) (*big.Int, error) {
	return _InteropOptimismMintableERC20.Contract.BalanceOf(&_InteropOptimismMintableERC20.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _InteropOptimismMintableERC20.Contract.BalanceOf(&_InteropOptimismMintableERC20.CallOpts, account)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) Bridge() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.Bridge(&_InteropOptimismMintableERC20.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) Bridge() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.Bridge(&_InteropOptimismMintableERC20.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) Decimals() (uint8, error) {
	return _InteropOptimismMintableERC20.Contract.Decimals(&_InteropOptimismMintableERC20.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) Decimals() (uint8, error) {
	return _InteropOptimismMintableERC20.Contract.Decimals(&_InteropOptimismMintableERC20.CallOpts)
}

// L1Token is a free data retrieval call binding the contract method 0xc01e1bd6.
//
// Solidity: function l1Token() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) L1Token(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "l1Token")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1Token is a free data retrieval call binding the contract method 0xc01e1bd6.
//
// Solidity: function l1Token() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) L1Token() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.L1Token(&_InteropOptimismMintableERC20.CallOpts)
}

// L1Token is a free data retrieval call binding the contract method 0xc01e1bd6.
//
// Solidity: function l1Token() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) L1Token() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.L1Token(&_InteropOptimismMintableERC20.CallOpts)
}

// L2Bridge is a free data retrieval call binding the contract method 0xae1f6aaf.
//
// Solidity: function l2Bridge() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) L2Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "l2Bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2Bridge is a free data retrieval call binding the contract method 0xae1f6aaf.
//
// Solidity: function l2Bridge() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) L2Bridge() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.L2Bridge(&_InteropOptimismMintableERC20.CallOpts)
}

// L2Bridge is a free data retrieval call binding the contract method 0xae1f6aaf.
//
// Solidity: function l2Bridge() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) L2Bridge() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.L2Bridge(&_InteropOptimismMintableERC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) Name() (string, error) {
	return _InteropOptimismMintableERC20.Contract.Name(&_InteropOptimismMintableERC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) Name() (string, error) {
	return _InteropOptimismMintableERC20.Contract.Name(&_InteropOptimismMintableERC20.CallOpts)
}

// RemoteToken is a free data retrieval call binding the contract method 0xd6c0b2c4.
//
// Solidity: function remoteToken() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) RemoteToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "remoteToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RemoteToken is a free data retrieval call binding the contract method 0xd6c0b2c4.
//
// Solidity: function remoteToken() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) RemoteToken() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.RemoteToken(&_InteropOptimismMintableERC20.CallOpts)
}

// RemoteToken is a free data retrieval call binding the contract method 0xd6c0b2c4.
//
// Solidity: function remoteToken() view returns(address)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) RemoteToken() (common.Address, error) {
	return _InteropOptimismMintableERC20.Contract.RemoteToken(&_InteropOptimismMintableERC20.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceId) pure returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) SupportsInterface(opts *bind.CallOpts, _interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "supportsInterface", _interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceId) pure returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) SupportsInterface(_interfaceId [4]byte) (bool, error) {
	return _InteropOptimismMintableERC20.Contract.SupportsInterface(&_InteropOptimismMintableERC20.CallOpts, _interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceId) pure returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) SupportsInterface(_interfaceId [4]byte) (bool, error) {
	return _InteropOptimismMintableERC20.Contract.SupportsInterface(&_InteropOptimismMintableERC20.CallOpts, _interfaceId)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) Symbol() (string, error) {
	return _InteropOptimismMintableERC20.Contract.Symbol(&_InteropOptimismMintableERC20.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) Symbol() (string, error) {
	return _InteropOptimismMintableERC20.Contract.Symbol(&_InteropOptimismMintableERC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) TotalSupply() (*big.Int, error) {
	return _InteropOptimismMintableERC20.Contract.TotalSupply(&_InteropOptimismMintableERC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) TotalSupply() (*big.Int, error) {
	return _InteropOptimismMintableERC20.Contract.TotalSupply(&_InteropOptimismMintableERC20.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Caller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _InteropOptimismMintableERC20.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) Version() (string, error) {
	return _InteropOptimismMintableERC20.Contract.Version(&_InteropOptimismMintableERC20.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20CallerSession) Version() (string, error) {
	return _InteropOptimismMintableERC20.Contract.Version(&_InteropOptimismMintableERC20.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Transactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.Approve(&_InteropOptimismMintableERC20.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20TransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.Approve(&_InteropOptimismMintableERC20.TransactOpts, spender, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address _from, uint256 _amount) returns()
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Transactor) Burn(opts *bind.TransactOpts, _from common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.contract.Transact(opts, "burn", _from, _amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address _from, uint256 _amount) returns()
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) Burn(_from common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.Burn(&_InteropOptimismMintableERC20.TransactOpts, _from, _amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address _from, uint256 _amount) returns()
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20TransactorSession) Burn(_from common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.Burn(&_InteropOptimismMintableERC20.TransactOpts, _from, _amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Transactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.DecreaseAllowance(&_InteropOptimismMintableERC20.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20TransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.DecreaseAllowance(&_InteropOptimismMintableERC20.TransactOpts, spender, subtractedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Transactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.IncreaseAllowance(&_InteropOptimismMintableERC20.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20TransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.IncreaseAllowance(&_InteropOptimismMintableERC20.TransactOpts, spender, addedValue)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _to, uint256 _amount) returns()
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Transactor) Mint(opts *bind.TransactOpts, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.contract.Transact(opts, "mint", _to, _amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _to, uint256 _amount) returns()
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) Mint(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.Mint(&_InteropOptimismMintableERC20.TransactOpts, _to, _amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _to, uint256 _amount) returns()
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20TransactorSession) Mint(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.Mint(&_InteropOptimismMintableERC20.TransactOpts, _to, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Transactor) Transfer(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.contract.Transact(opts, "transfer", to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.Transfer(&_InteropOptimismMintableERC20.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20TransactorSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.Transfer(&_InteropOptimismMintableERC20.TransactOpts, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Transactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.contract.Transact(opts, "transferFrom", from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Session) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.TransferFrom(&_InteropOptimismMintableERC20.TransactOpts, from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20TransactorSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InteropOptimismMintableERC20.Contract.TransferFrom(&_InteropOptimismMintableERC20.TransactOpts, from, to, amount)
}

// InteropOptimismMintableERC20ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the InteropOptimismMintableERC20 contract.
type InteropOptimismMintableERC20ApprovalIterator struct {
	Event *InteropOptimismMintableERC20Approval // Event containing the contract specifics and raw log

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
func (it *InteropOptimismMintableERC20ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropOptimismMintableERC20Approval)
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
		it.Event = new(InteropOptimismMintableERC20Approval)
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
func (it *InteropOptimismMintableERC20ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropOptimismMintableERC20ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropOptimismMintableERC20Approval represents a Approval event raised by the InteropOptimismMintableERC20 contract.
type InteropOptimismMintableERC20Approval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*InteropOptimismMintableERC20ApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _InteropOptimismMintableERC20.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &InteropOptimismMintableERC20ApprovalIterator{contract: _InteropOptimismMintableERC20.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *InteropOptimismMintableERC20Approval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _InteropOptimismMintableERC20.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropOptimismMintableERC20Approval)
				if err := _InteropOptimismMintableERC20.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) ParseApproval(log types.Log) (*InteropOptimismMintableERC20Approval, error) {
	event := new(InteropOptimismMintableERC20Approval)
	if err := _InteropOptimismMintableERC20.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InteropOptimismMintableERC20BurnIterator is returned from FilterBurn and is used to iterate over the raw logs and unpacked data for Burn events raised by the InteropOptimismMintableERC20 contract.
type InteropOptimismMintableERC20BurnIterator struct {
	Event *InteropOptimismMintableERC20Burn // Event containing the contract specifics and raw log

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
func (it *InteropOptimismMintableERC20BurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropOptimismMintableERC20Burn)
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
		it.Event = new(InteropOptimismMintableERC20Burn)
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
func (it *InteropOptimismMintableERC20BurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropOptimismMintableERC20BurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropOptimismMintableERC20Burn represents a Burn event raised by the InteropOptimismMintableERC20 contract.
type InteropOptimismMintableERC20Burn struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterBurn is a free log retrieval operation binding the contract event 0xcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca5.
//
// Solidity: event Burn(address indexed account, uint256 amount)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) FilterBurn(opts *bind.FilterOpts, account []common.Address) (*InteropOptimismMintableERC20BurnIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _InteropOptimismMintableERC20.contract.FilterLogs(opts, "Burn", accountRule)
	if err != nil {
		return nil, err
	}
	return &InteropOptimismMintableERC20BurnIterator{contract: _InteropOptimismMintableERC20.contract, event: "Burn", logs: logs, sub: sub}, nil
}

// WatchBurn is a free log subscription operation binding the contract event 0xcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca5.
//
// Solidity: event Burn(address indexed account, uint256 amount)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) WatchBurn(opts *bind.WatchOpts, sink chan<- *InteropOptimismMintableERC20Burn, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _InteropOptimismMintableERC20.contract.WatchLogs(opts, "Burn", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropOptimismMintableERC20Burn)
				if err := _InteropOptimismMintableERC20.contract.UnpackLog(event, "Burn", log); err != nil {
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

// ParseBurn is a log parse operation binding the contract event 0xcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca5.
//
// Solidity: event Burn(address indexed account, uint256 amount)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) ParseBurn(log types.Log) (*InteropOptimismMintableERC20Burn, error) {
	event := new(InteropOptimismMintableERC20Burn)
	if err := _InteropOptimismMintableERC20.contract.UnpackLog(event, "Burn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InteropOptimismMintableERC20MintIterator is returned from FilterMint and is used to iterate over the raw logs and unpacked data for Mint events raised by the InteropOptimismMintableERC20 contract.
type InteropOptimismMintableERC20MintIterator struct {
	Event *InteropOptimismMintableERC20Mint // Event containing the contract specifics and raw log

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
func (it *InteropOptimismMintableERC20MintIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropOptimismMintableERC20Mint)
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
		it.Event = new(InteropOptimismMintableERC20Mint)
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
func (it *InteropOptimismMintableERC20MintIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropOptimismMintableERC20MintIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropOptimismMintableERC20Mint represents a Mint event raised by the InteropOptimismMintableERC20 contract.
type InteropOptimismMintableERC20Mint struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterMint is a free log retrieval operation binding the contract event 0x0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d4121396885.
//
// Solidity: event Mint(address indexed account, uint256 amount)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) FilterMint(opts *bind.FilterOpts, account []common.Address) (*InteropOptimismMintableERC20MintIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _InteropOptimismMintableERC20.contract.FilterLogs(opts, "Mint", accountRule)
	if err != nil {
		return nil, err
	}
	return &InteropOptimismMintableERC20MintIterator{contract: _InteropOptimismMintableERC20.contract, event: "Mint", logs: logs, sub: sub}, nil
}

// WatchMint is a free log subscription operation binding the contract event 0x0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d4121396885.
//
// Solidity: event Mint(address indexed account, uint256 amount)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) WatchMint(opts *bind.WatchOpts, sink chan<- *InteropOptimismMintableERC20Mint, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _InteropOptimismMintableERC20.contract.WatchLogs(opts, "Mint", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropOptimismMintableERC20Mint)
				if err := _InteropOptimismMintableERC20.contract.UnpackLog(event, "Mint", log); err != nil {
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

// ParseMint is a log parse operation binding the contract event 0x0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d4121396885.
//
// Solidity: event Mint(address indexed account, uint256 amount)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) ParseMint(log types.Log) (*InteropOptimismMintableERC20Mint, error) {
	event := new(InteropOptimismMintableERC20Mint)
	if err := _InteropOptimismMintableERC20.contract.UnpackLog(event, "Mint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InteropOptimismMintableERC20TransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the InteropOptimismMintableERC20 contract.
type InteropOptimismMintableERC20TransferIterator struct {
	Event *InteropOptimismMintableERC20Transfer // Event containing the contract specifics and raw log

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
func (it *InteropOptimismMintableERC20TransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropOptimismMintableERC20Transfer)
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
		it.Event = new(InteropOptimismMintableERC20Transfer)
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
func (it *InteropOptimismMintableERC20TransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropOptimismMintableERC20TransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropOptimismMintableERC20Transfer represents a Transfer event raised by the InteropOptimismMintableERC20 contract.
type InteropOptimismMintableERC20Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*InteropOptimismMintableERC20TransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _InteropOptimismMintableERC20.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &InteropOptimismMintableERC20TransferIterator{contract: _InteropOptimismMintableERC20.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *InteropOptimismMintableERC20Transfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _InteropOptimismMintableERC20.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropOptimismMintableERC20Transfer)
				if err := _InteropOptimismMintableERC20.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_InteropOptimismMintableERC20 *InteropOptimismMintableERC20Filterer) ParseTransfer(log types.Log) (*InteropOptimismMintableERC20Transfer, error) {
	event := new(InteropOptimismMintableERC20Transfer)
	if err := _InteropOptimismMintableERC20.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
