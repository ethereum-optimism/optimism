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

// OptimismMintableERC20MetaData contains all meta data concerning the OptimismMintableERC20 contract.
var OptimismMintableERC20MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_bridge\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Burn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Mint\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1Token\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2Bridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"remoteToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"_interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b506040516200164b3803806200164b833981016040819052620000349162000179565b8181600362000044838262000298565b50600462000053828262000298565b5050600580546001600160a01b039586166001600160a01b03199182161790915560068054969095169516949094179092555062000364915050565b80516001600160a01b0381168114620000a757600080fd5b919050565b634e487b7160e01b600052604160045260246000fd5b600082601f830112620000d457600080fd5b81516001600160401b0380821115620000f157620000f1620000ac565b604051601f8301601f19908116603f011681019082821181831017156200011c576200011c620000ac565b816040528381526020925086838588010111156200013957600080fd5b600091505b838210156200015d57858201830151818301840152908201906200013e565b838211156200016f5760008385830101525b9695505050505050565b600080600080608085870312156200019057600080fd5b6200019b856200008f565b9350620001ab602086016200008f565b60408601519093506001600160401b0380821115620001c957600080fd5b620001d788838901620000c2565b93506060870151915080821115620001ee57600080fd5b50620001fd87828801620000c2565b91505092959194509250565b600181811c908216806200021e57607f821691505b6020821081036200023f57634e487b7160e01b600052602260045260246000fd5b50919050565b601f8211156200029357600081815260208120601f850160051c810160208610156200026e5750805b601f850160051c820191505b818110156200028f578281556001016200027a565b5050505b505050565b81516001600160401b03811115620002b457620002b4620000ac565b620002cc81620002c5845462000209565b8462000245565b602080601f831160018114620003045760008415620002eb5750858301515b600019600386901b1c1916600185901b1785556200028f565b600085815260208120601f198616915b82811015620003355788860151825594840194600190910190840162000314565b5085821015620003545787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b6112d780620003746000396000f3fe608060405234801561001057600080fd5b50600436106101365760003560e01c806395d89b41116100b2578063ae1f6aaf11610081578063d6c0b2c411610066578063d6c0b2c4146102bb578063dd62ed3e146102db578063e78cea921461032157600080fd5b8063ae1f6aaf1461025e578063c01e1bd61461029d57600080fd5b806395d89b411461021d5780639dc29fac14610225578063a457c2d714610238578063a9059cbb1461024b57600080fd5b806323b872dd1161010957806339509351116100ee57806339509351146101bf57806340c10f19146101d257806370a08231146101e757600080fd5b806323b872dd1461019d578063313ce567146101b057600080fd5b806301ffc9a71461013b57806306fdde0314610163578063095ea7b31461017857806318160ddd1461018b575b600080fd5b61014e610149366004611080565b610341565b60405190151581526020015b60405180910390f35b61016b610432565b60405161015a91906110c9565b61014e610186366004611165565b6104c4565b6002545b60405190815260200161015a565b61014e6101ab36600461118f565b6104dc565b6040516012815260200161015a565b61014e6101cd366004611165565b610500565b6101e56101e0366004611165565b61054c565b005b61018f6101f53660046111cb565b73ffffffffffffffffffffffffffffffffffffffff1660009081526020819052604090205490565b61016b610656565b6101e5610233366004611165565b610665565b61014e610246366004611165565b61075e565b61014e610259366004611165565b61082f565b60065473ffffffffffffffffffffffffffffffffffffffff165b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200161015a565b60055473ffffffffffffffffffffffffffffffffffffffff16610278565b6005546102789073ffffffffffffffffffffffffffffffffffffffff1681565b61018f6102e93660046111e6565b73ffffffffffffffffffffffffffffffffffffffff918216600090815260016020908152604080832093909416825291909152205490565b6006546102789073ffffffffffffffffffffffffffffffffffffffff1681565b60007f01ffc9a7000000000000000000000000000000000000000000000000000000007f1d1d8b63000000000000000000000000000000000000000000000000000000007fec4fc8e3000000000000000000000000000000000000000000000000000000007fffffffff0000000000000000000000000000000000000000000000000000000085168314806103fa57507fffffffff00000000000000000000000000000000000000000000000000000000858116908316145b8061042957507fffffffff00000000000000000000000000000000000000000000000000000000858116908216145b95945050505050565b60606003805461044190611219565b80601f016020809104026020016040519081016040528092919081815260200182805461046d90611219565b80156104ba5780601f1061048f576101008083540402835291602001916104ba565b820191906000526020600020905b81548152906001019060200180831161049d57829003601f168201915b5050505050905090565b6000336104d281858561083d565b5060019392505050565b6000336104ea8582856109f1565b6104f5858585610ac8565b506001949350505050565b33600081815260016020908152604080832073ffffffffffffffffffffffffffffffffffffffff871684529091528120549091906104d2908290869061054790879061129b565b61083d565b60065473ffffffffffffffffffffffffffffffffffffffff1633146105f8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603460248201527f4f7074696d69736d4d696e7461626c6545524332303a206f6e6c79206272696460448201527f67652063616e206d696e7420616e64206275726e00000000000000000000000060648201526084015b60405180910390fd5b6106028282610d7b565b8173ffffffffffffffffffffffffffffffffffffffff167f0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d41213968858260405161064a91815260200190565b60405180910390a25050565b60606004805461044190611219565b60065473ffffffffffffffffffffffffffffffffffffffff16331461070c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603460248201527f4f7074696d69736d4d696e7461626c6545524332303a206f6e6c79206272696460448201527f67652063616e206d696e7420616e64206275726e00000000000000000000000060648201526084016105ef565b6107168282610e9b565b8173ffffffffffffffffffffffffffffffffffffffff167fcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca58260405161064a91815260200190565b33600081815260016020908152604080832073ffffffffffffffffffffffffffffffffffffffff8716845290915281205490919083811015610822576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760448201527f207a65726f00000000000000000000000000000000000000000000000000000060648201526084016105ef565b6104f5828686840361083d565b6000336104d2818585610ac8565b73ffffffffffffffffffffffffffffffffffffffff83166108df576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526024808201527f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460448201527f726573730000000000000000000000000000000000000000000000000000000060648201526084016105ef565b73ffffffffffffffffffffffffffffffffffffffff8216610982576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a20617070726f766520746f20746865207a65726f20616464726560448201527f737300000000000000000000000000000000000000000000000000000000000060648201526084016105ef565b73ffffffffffffffffffffffffffffffffffffffff83811660008181526001602090815260408083209487168084529482529182902085905590518481527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92591015b60405180910390a3505050565b73ffffffffffffffffffffffffffffffffffffffff8381166000908152600160209081526040808320938616835292905220547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8114610ac25781811015610ab5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f45524332303a20696e73756666696369656e7420616c6c6f77616e636500000060448201526064016105ef565b610ac2848484840361083d565b50505050565b73ffffffffffffffffffffffffffffffffffffffff8316610b6b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a207472616e736665722066726f6d20746865207a65726f20616460448201527f647265737300000000000000000000000000000000000000000000000000000060648201526084016105ef565b73ffffffffffffffffffffffffffffffffffffffff8216610c0e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f45524332303a207472616e7366657220746f20746865207a65726f206164647260448201527f657373000000000000000000000000000000000000000000000000000000000060648201526084016105ef565b73ffffffffffffffffffffffffffffffffffffffff831660009081526020819052604090205481811015610cc4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f45524332303a207472616e7366657220616d6f756e742065786365656473206260448201527f616c616e6365000000000000000000000000000000000000000000000000000060648201526084016105ef565b73ffffffffffffffffffffffffffffffffffffffff808516600090815260208190526040808220858503905591851681529081208054849290610d0890849061129b565b925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef84604051610d6e91815260200190565b60405180910390a3610ac2565b73ffffffffffffffffffffffffffffffffffffffff8216610df8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f20616464726573730060448201526064016105ef565b8060026000828254610e0a919061129b565b909155505073ffffffffffffffffffffffffffffffffffffffff821660009081526020819052604081208054839290610e4490849061129b565b909155505060405181815273ffffffffffffffffffffffffffffffffffffffff8316906000907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9060200160405180910390a35050565b73ffffffffffffffffffffffffffffffffffffffff8216610f3e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602160248201527f45524332303a206275726e2066726f6d20746865207a65726f2061646472657360448201527f730000000000000000000000000000000000000000000000000000000000000060648201526084016105ef565b73ffffffffffffffffffffffffffffffffffffffff821660009081526020819052604090205481811015610ff4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a206275726e20616d6f756e7420657863656564732062616c616e60448201527f636500000000000000000000000000000000000000000000000000000000000060648201526084016105ef565b73ffffffffffffffffffffffffffffffffffffffff831660009081526020819052604081208383039055600280548492906110309084906112b3565b909155505060405182815260009073ffffffffffffffffffffffffffffffffffffffff8516907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef906020016109e4565b60006020828403121561109257600080fd5b81357fffffffff00000000000000000000000000000000000000000000000000000000811681146110c257600080fd5b9392505050565b600060208083528351808285015260005b818110156110f6578581018301518582016040015282016110da565b81811115611108576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461116057600080fd5b919050565b6000806040838503121561117857600080fd5b6111818361113c565b946020939093013593505050565b6000806000606084860312156111a457600080fd5b6111ad8461113c565b92506111bb6020850161113c565b9150604084013590509250925092565b6000602082840312156111dd57600080fd5b6110c28261113c565b600080604083850312156111f957600080fd5b6112028361113c565b91506112106020840161113c565b90509250929050565b600181811c9082168061122d57607f821691505b602082108103611266577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082198211156112ae576112ae61126c565b500190565b6000828210156112c5576112c561126c565b50039056fea164736f6c634300080f000a",
}

// OptimismMintableERC20ABI is the input ABI used to generate the binding from.
// Deprecated: Use OptimismMintableERC20MetaData.ABI instead.
var OptimismMintableERC20ABI = OptimismMintableERC20MetaData.ABI

// OptimismMintableERC20Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OptimismMintableERC20MetaData.Bin instead.
var OptimismMintableERC20Bin = OptimismMintableERC20MetaData.Bin

// DeployOptimismMintableERC20 deploys a new Ethereum contract, binding an instance of OptimismMintableERC20 to it.
func DeployOptimismMintableERC20(auth *bind.TransactOpts, backend bind.ContractBackend, _bridge common.Address, _remoteToken common.Address, _name string, _symbol string) (common.Address, *types.Transaction, *OptimismMintableERC20, error) {
	parsed, err := OptimismMintableERC20MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OptimismMintableERC20Bin), backend, _bridge, _remoteToken, _name, _symbol)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OptimismMintableERC20{OptimismMintableERC20Caller: OptimismMintableERC20Caller{contract: contract}, OptimismMintableERC20Transactor: OptimismMintableERC20Transactor{contract: contract}, OptimismMintableERC20Filterer: OptimismMintableERC20Filterer{contract: contract}}, nil
}

// OptimismMintableERC20 is an auto generated Go binding around an Ethereum contract.
type OptimismMintableERC20 struct {
	OptimismMintableERC20Caller     // Read-only binding to the contract
	OptimismMintableERC20Transactor // Write-only binding to the contract
	OptimismMintableERC20Filterer   // Log filterer for contract events
}

// OptimismMintableERC20Caller is an auto generated read-only Go binding around an Ethereum contract.
type OptimismMintableERC20Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismMintableERC20Transactor is an auto generated write-only Go binding around an Ethereum contract.
type OptimismMintableERC20Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismMintableERC20Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OptimismMintableERC20Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismMintableERC20Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OptimismMintableERC20Session struct {
	Contract     *OptimismMintableERC20 // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// OptimismMintableERC20CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OptimismMintableERC20CallerSession struct {
	Contract *OptimismMintableERC20Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// OptimismMintableERC20TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OptimismMintableERC20TransactorSession struct {
	Contract     *OptimismMintableERC20Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// OptimismMintableERC20Raw is an auto generated low-level Go binding around an Ethereum contract.
type OptimismMintableERC20Raw struct {
	Contract *OptimismMintableERC20 // Generic contract binding to access the raw methods on
}

// OptimismMintableERC20CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OptimismMintableERC20CallerRaw struct {
	Contract *OptimismMintableERC20Caller // Generic read-only contract binding to access the raw methods on
}

// OptimismMintableERC20TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OptimismMintableERC20TransactorRaw struct {
	Contract *OptimismMintableERC20Transactor // Generic write-only contract binding to access the raw methods on
}

// NewOptimismMintableERC20 creates a new instance of OptimismMintableERC20, bound to a specific deployed contract.
func NewOptimismMintableERC20(address common.Address, backend bind.ContractBackend) (*OptimismMintableERC20, error) {
	contract, err := bindOptimismMintableERC20(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20{OptimismMintableERC20Caller: OptimismMintableERC20Caller{contract: contract}, OptimismMintableERC20Transactor: OptimismMintableERC20Transactor{contract: contract}, OptimismMintableERC20Filterer: OptimismMintableERC20Filterer{contract: contract}}, nil
}

// NewOptimismMintableERC20Caller creates a new read-only instance of OptimismMintableERC20, bound to a specific deployed contract.
func NewOptimismMintableERC20Caller(address common.Address, caller bind.ContractCaller) (*OptimismMintableERC20Caller, error) {
	contract, err := bindOptimismMintableERC20(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20Caller{contract: contract}, nil
}

// NewOptimismMintableERC20Transactor creates a new write-only instance of OptimismMintableERC20, bound to a specific deployed contract.
func NewOptimismMintableERC20Transactor(address common.Address, transactor bind.ContractTransactor) (*OptimismMintableERC20Transactor, error) {
	contract, err := bindOptimismMintableERC20(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20Transactor{contract: contract}, nil
}

// NewOptimismMintableERC20Filterer creates a new log filterer instance of OptimismMintableERC20, bound to a specific deployed contract.
func NewOptimismMintableERC20Filterer(address common.Address, filterer bind.ContractFilterer) (*OptimismMintableERC20Filterer, error) {
	contract, err := bindOptimismMintableERC20(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20Filterer{contract: contract}, nil
}

// bindOptimismMintableERC20 binds a generic wrapper to an already deployed contract.
func bindOptimismMintableERC20(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OptimismMintableERC20ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismMintableERC20 *OptimismMintableERC20Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismMintableERC20.Contract.OptimismMintableERC20Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismMintableERC20 *OptimismMintableERC20Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.OptimismMintableERC20Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismMintableERC20 *OptimismMintableERC20Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.OptimismMintableERC20Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismMintableERC20 *OptimismMintableERC20CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismMintableERC20.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismMintableERC20 *OptimismMintableERC20TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismMintableERC20 *OptimismMintableERC20TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _OptimismMintableERC20.Contract.Allowance(&_OptimismMintableERC20.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _OptimismMintableERC20.Contract.Allowance(&_OptimismMintableERC20.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) BalanceOf(account common.Address) (*big.Int, error) {
	return _OptimismMintableERC20.Contract.BalanceOf(&_OptimismMintableERC20.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _OptimismMintableERC20.Contract.BalanceOf(&_OptimismMintableERC20.CallOpts, account)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) Bridge() (common.Address, error) {
	return _OptimismMintableERC20.Contract.Bridge(&_OptimismMintableERC20.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) Bridge() (common.Address, error) {
	return _OptimismMintableERC20.Contract.Bridge(&_OptimismMintableERC20.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) Decimals() (uint8, error) {
	return _OptimismMintableERC20.Contract.Decimals(&_OptimismMintableERC20.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) Decimals() (uint8, error) {
	return _OptimismMintableERC20.Contract.Decimals(&_OptimismMintableERC20.CallOpts)
}

// L1Token is a free data retrieval call binding the contract method 0xc01e1bd6.
//
// Solidity: function l1Token() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) L1Token(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "l1Token")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1Token is a free data retrieval call binding the contract method 0xc01e1bd6.
//
// Solidity: function l1Token() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) L1Token() (common.Address, error) {
	return _OptimismMintableERC20.Contract.L1Token(&_OptimismMintableERC20.CallOpts)
}

// L1Token is a free data retrieval call binding the contract method 0xc01e1bd6.
//
// Solidity: function l1Token() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) L1Token() (common.Address, error) {
	return _OptimismMintableERC20.Contract.L1Token(&_OptimismMintableERC20.CallOpts)
}

// L2Bridge is a free data retrieval call binding the contract method 0xae1f6aaf.
//
// Solidity: function l2Bridge() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) L2Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "l2Bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2Bridge is a free data retrieval call binding the contract method 0xae1f6aaf.
//
// Solidity: function l2Bridge() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) L2Bridge() (common.Address, error) {
	return _OptimismMintableERC20.Contract.L2Bridge(&_OptimismMintableERC20.CallOpts)
}

// L2Bridge is a free data retrieval call binding the contract method 0xae1f6aaf.
//
// Solidity: function l2Bridge() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) L2Bridge() (common.Address, error) {
	return _OptimismMintableERC20.Contract.L2Bridge(&_OptimismMintableERC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) Name() (string, error) {
	return _OptimismMintableERC20.Contract.Name(&_OptimismMintableERC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) Name() (string, error) {
	return _OptimismMintableERC20.Contract.Name(&_OptimismMintableERC20.CallOpts)
}

// RemoteToken is a free data retrieval call binding the contract method 0xd6c0b2c4.
//
// Solidity: function remoteToken() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) RemoteToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "remoteToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RemoteToken is a free data retrieval call binding the contract method 0xd6c0b2c4.
//
// Solidity: function remoteToken() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) RemoteToken() (common.Address, error) {
	return _OptimismMintableERC20.Contract.RemoteToken(&_OptimismMintableERC20.CallOpts)
}

// RemoteToken is a free data retrieval call binding the contract method 0xd6c0b2c4.
//
// Solidity: function remoteToken() view returns(address)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) RemoteToken() (common.Address, error) {
	return _OptimismMintableERC20.Contract.RemoteToken(&_OptimismMintableERC20.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceId) pure returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) SupportsInterface(opts *bind.CallOpts, _interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "supportsInterface", _interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceId) pure returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) SupportsInterface(_interfaceId [4]byte) (bool, error) {
	return _OptimismMintableERC20.Contract.SupportsInterface(&_OptimismMintableERC20.CallOpts, _interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceId) pure returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) SupportsInterface(_interfaceId [4]byte) (bool, error) {
	return _OptimismMintableERC20.Contract.SupportsInterface(&_OptimismMintableERC20.CallOpts, _interfaceId)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) Symbol() (string, error) {
	return _OptimismMintableERC20.Contract.Symbol(&_OptimismMintableERC20.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) Symbol() (string, error) {
	return _OptimismMintableERC20.Contract.Symbol(&_OptimismMintableERC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_OptimismMintableERC20 *OptimismMintableERC20Caller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismMintableERC20.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) TotalSupply() (*big.Int, error) {
	return _OptimismMintableERC20.Contract.TotalSupply(&_OptimismMintableERC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_OptimismMintableERC20 *OptimismMintableERC20CallerSession) TotalSupply() (*big.Int, error) {
	return _OptimismMintableERC20.Contract.TotalSupply(&_OptimismMintableERC20.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Transactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.Approve(&_OptimismMintableERC20.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20TransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.Approve(&_OptimismMintableERC20.TransactOpts, spender, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address _from, uint256 _amount) returns()
func (_OptimismMintableERC20 *OptimismMintableERC20Transactor) Burn(opts *bind.TransactOpts, _from common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.contract.Transact(opts, "burn", _from, _amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address _from, uint256 _amount) returns()
func (_OptimismMintableERC20 *OptimismMintableERC20Session) Burn(_from common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.Burn(&_OptimismMintableERC20.TransactOpts, _from, _amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address _from, uint256 _amount) returns()
func (_OptimismMintableERC20 *OptimismMintableERC20TransactorSession) Burn(_from common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.Burn(&_OptimismMintableERC20.TransactOpts, _from, _amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Transactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.DecreaseAllowance(&_OptimismMintableERC20.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20TransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.DecreaseAllowance(&_OptimismMintableERC20.TransactOpts, spender, subtractedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Transactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.IncreaseAllowance(&_OptimismMintableERC20.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20TransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.IncreaseAllowance(&_OptimismMintableERC20.TransactOpts, spender, addedValue)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _to, uint256 _amount) returns()
func (_OptimismMintableERC20 *OptimismMintableERC20Transactor) Mint(opts *bind.TransactOpts, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.contract.Transact(opts, "mint", _to, _amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _to, uint256 _amount) returns()
func (_OptimismMintableERC20 *OptimismMintableERC20Session) Mint(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.Mint(&_OptimismMintableERC20.TransactOpts, _to, _amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _to, uint256 _amount) returns()
func (_OptimismMintableERC20 *OptimismMintableERC20TransactorSession) Mint(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.Mint(&_OptimismMintableERC20.TransactOpts, _to, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Transactor) Transfer(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.contract.Transact(opts, "transfer", to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.Transfer(&_OptimismMintableERC20.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20TransactorSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.Transfer(&_OptimismMintableERC20.TransactOpts, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Transactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.contract.Transact(opts, "transferFrom", from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20Session) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.TransferFrom(&_OptimismMintableERC20.TransactOpts, from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_OptimismMintableERC20 *OptimismMintableERC20TransactorSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _OptimismMintableERC20.Contract.TransferFrom(&_OptimismMintableERC20.TransactOpts, from, to, amount)
}

// OptimismMintableERC20ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the OptimismMintableERC20 contract.
type OptimismMintableERC20ApprovalIterator struct {
	Event *OptimismMintableERC20Approval // Event containing the contract specifics and raw log

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
func (it *OptimismMintableERC20ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismMintableERC20Approval)
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
		it.Event = new(OptimismMintableERC20Approval)
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
func (it *OptimismMintableERC20ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismMintableERC20ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismMintableERC20Approval represents a Approval event raised by the OptimismMintableERC20 contract.
type OptimismMintableERC20Approval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*OptimismMintableERC20ApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _OptimismMintableERC20.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20ApprovalIterator{contract: _OptimismMintableERC20.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *OptimismMintableERC20Approval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _OptimismMintableERC20.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismMintableERC20Approval)
				if err := _OptimismMintableERC20.contract.UnpackLog(event, "Approval", log); err != nil {
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
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) ParseApproval(log types.Log) (*OptimismMintableERC20Approval, error) {
	event := new(OptimismMintableERC20Approval)
	if err := _OptimismMintableERC20.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismMintableERC20BurnIterator is returned from FilterBurn and is used to iterate over the raw logs and unpacked data for Burn events raised by the OptimismMintableERC20 contract.
type OptimismMintableERC20BurnIterator struct {
	Event *OptimismMintableERC20Burn // Event containing the contract specifics and raw log

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
func (it *OptimismMintableERC20BurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismMintableERC20Burn)
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
		it.Event = new(OptimismMintableERC20Burn)
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
func (it *OptimismMintableERC20BurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismMintableERC20BurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismMintableERC20Burn represents a Burn event raised by the OptimismMintableERC20 contract.
type OptimismMintableERC20Burn struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterBurn is a free log retrieval operation binding the contract event 0xcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca5.
//
// Solidity: event Burn(address indexed account, uint256 amount)
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) FilterBurn(opts *bind.FilterOpts, account []common.Address) (*OptimismMintableERC20BurnIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _OptimismMintableERC20.contract.FilterLogs(opts, "Burn", accountRule)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20BurnIterator{contract: _OptimismMintableERC20.contract, event: "Burn", logs: logs, sub: sub}, nil
}

// WatchBurn is a free log subscription operation binding the contract event 0xcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca5.
//
// Solidity: event Burn(address indexed account, uint256 amount)
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) WatchBurn(opts *bind.WatchOpts, sink chan<- *OptimismMintableERC20Burn, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _OptimismMintableERC20.contract.WatchLogs(opts, "Burn", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismMintableERC20Burn)
				if err := _OptimismMintableERC20.contract.UnpackLog(event, "Burn", log); err != nil {
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
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) ParseBurn(log types.Log) (*OptimismMintableERC20Burn, error) {
	event := new(OptimismMintableERC20Burn)
	if err := _OptimismMintableERC20.contract.UnpackLog(event, "Burn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismMintableERC20MintIterator is returned from FilterMint and is used to iterate over the raw logs and unpacked data for Mint events raised by the OptimismMintableERC20 contract.
type OptimismMintableERC20MintIterator struct {
	Event *OptimismMintableERC20Mint // Event containing the contract specifics and raw log

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
func (it *OptimismMintableERC20MintIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismMintableERC20Mint)
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
		it.Event = new(OptimismMintableERC20Mint)
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
func (it *OptimismMintableERC20MintIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismMintableERC20MintIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismMintableERC20Mint represents a Mint event raised by the OptimismMintableERC20 contract.
type OptimismMintableERC20Mint struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterMint is a free log retrieval operation binding the contract event 0x0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d4121396885.
//
// Solidity: event Mint(address indexed account, uint256 amount)
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) FilterMint(opts *bind.FilterOpts, account []common.Address) (*OptimismMintableERC20MintIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _OptimismMintableERC20.contract.FilterLogs(opts, "Mint", accountRule)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20MintIterator{contract: _OptimismMintableERC20.contract, event: "Mint", logs: logs, sub: sub}, nil
}

// WatchMint is a free log subscription operation binding the contract event 0x0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d4121396885.
//
// Solidity: event Mint(address indexed account, uint256 amount)
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) WatchMint(opts *bind.WatchOpts, sink chan<- *OptimismMintableERC20Mint, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _OptimismMintableERC20.contract.WatchLogs(opts, "Mint", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismMintableERC20Mint)
				if err := _OptimismMintableERC20.contract.UnpackLog(event, "Mint", log); err != nil {
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
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) ParseMint(log types.Log) (*OptimismMintableERC20Mint, error) {
	event := new(OptimismMintableERC20Mint)
	if err := _OptimismMintableERC20.contract.UnpackLog(event, "Mint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismMintableERC20TransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the OptimismMintableERC20 contract.
type OptimismMintableERC20TransferIterator struct {
	Event *OptimismMintableERC20Transfer // Event containing the contract specifics and raw log

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
func (it *OptimismMintableERC20TransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismMintableERC20Transfer)
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
		it.Event = new(OptimismMintableERC20Transfer)
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
func (it *OptimismMintableERC20TransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismMintableERC20TransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismMintableERC20Transfer represents a Transfer event raised by the OptimismMintableERC20 contract.
type OptimismMintableERC20Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*OptimismMintableERC20TransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _OptimismMintableERC20.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20TransferIterator{contract: _OptimismMintableERC20.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *OptimismMintableERC20Transfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _OptimismMintableERC20.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismMintableERC20Transfer)
				if err := _OptimismMintableERC20.contract.UnpackLog(event, "Transfer", log); err != nil {
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
func (_OptimismMintableERC20 *OptimismMintableERC20Filterer) ParseTransfer(log types.Log) (*OptimismMintableERC20Transfer, error) {
	event := new(OptimismMintableERC20Transfer)
	if err := _OptimismMintableERC20.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
