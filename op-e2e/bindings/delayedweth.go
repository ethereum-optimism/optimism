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

// DelayedWETHMetaData contains all meta data concerning the DelayedWETH contract.
var DelayedWETHMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_delay\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"fallback\",\"stateMutability\":\"payable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"allowance\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"approve\",\"inputs\":[{\"name\":\"guy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"wad\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"config\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractSuperchainConfig\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"decimals\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"delay\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deposit\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"hold\",\"inputs\":[{\"name\":\"_guy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_wad\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_config\",\"type\":\"address\",\"internalType\":\"contractSuperchainConfig\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"name\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"recover\",\"inputs\":[{\"name\":\"_wad\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"symbol\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transfer\",\"inputs\":[{\"name\":\"dst\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"wad\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferFrom\",\"inputs\":[{\"name\":\"src\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"dst\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"wad\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unlock\",\"inputs\":[{\"name\":\"_guy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_wad\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"withdraw\",\"inputs\":[{\"name\":\"_wad\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdraw\",\"inputs\":[{\"name\":\"_guy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_wad\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdrawals\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"timestamp\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"Approval\",\"inputs\":[{\"name\":\"src\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"guy\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"wad\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Deposit\",\"inputs\":[{\"name\":\"dst\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"wad\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Transfer\",\"inputs\":[{\"name\":\"src\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"dst\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"wad\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Unwrap\",\"inputs\":[{\"name\":\"src\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"wad\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Withdrawal\",\"inputs\":[{\"name\":\"src\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"wad\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
	Bin: "0x60a06040523480156200001157600080fd5b50604051620018aa380380620018aa8339810160408190526200003491620002d7565b6080819052620000466000806200004d565b50620002f1565b600054610100900460ff16158080156200006e5750600054600160ff909116105b806200009e57506200008b30620001a760201b6200107f1760201c565b1580156200009e575060005460ff166001145b620001075760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805460ff1916600117905580156200012b576000805461ff0019166101001790555b62000135620001b6565b62000140836200021e565b606880546001600160a01b0319166001600160a01b0384161790558015620001a2576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b505050565b6001600160a01b03163b151590565b600054610100900460ff16620002125760405162461bcd60e51b815260206004820152602b60248201526000805160206200188a83398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000fe565b6200021c62000270565b565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600054610100900460ff16620002cc5760405162461bcd60e51b815260206004820152602b60248201526000805160206200188a83398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000fe565b6200021c336200021e565b600060208284031215620002ea57600080fd5b5051919050565b608051611576620003146000396000818161033f0152610fa001526115766000f3fe6080604052600436106101845760003560e01c8063715018a6116100d6578063a9059cbb1161007f578063dd62ed3e11610059578063dd62ed3e1461051c578063f2fde38b14610554578063f3fef3a31461057457610193565b8063a9059cbb146104a8578063cd47bde1146104c8578063d0e30db01461019357610193565b80638da5cb5b116100b05780638da5cb5b1461041757806395d89b4114610442578063977a5ec51461048857610193565b8063715018a61461039057806379502c55146103a55780637eee288d146103f757610193565b80632e1a7d4d1161013857806354fd4d501161011257806354fd4d50146102e75780636a42b8f81461033057806370a082311461036357610193565b80632e1a7d4d14610280578063313ce567146102a0578063485cc955146102c757610193565b80630ca35682116101695780630ca356821461022357806318160ddd1461024357806323b872dd1461026057610193565b806306fdde031461019b578063095ea7b3146101f357610193565b3661019357610191610594565b005b610191610594565b3480156101a757600080fd5b5060408051808201909152600d81527f577261707065642045746865720000000000000000000000000000000000000060208201525b6040516101ea9190611378565b60405180910390f35b3480156101ff57600080fd5b5061021361020e36600461140d565b6105ef565b60405190151581526020016101ea565b34801561022f57600080fd5b5061019161023e366004611439565b610668565b34801561024f57600080fd5b50475b6040519081526020016101ea565b34801561026c57600080fd5b5061021361027b366004611452565b610734565b34801561028c57600080fd5b5061019161029b366004611439565b61094b565b3480156102ac57600080fd5b506102b5601281565b60405160ff90911681526020016101ea565b3480156102d357600080fd5b506101916102e2366004611493565b610958565b3480156102f357600080fd5b506101dd6040518060400160405280600581526020017f302e322e3000000000000000000000000000000000000000000000000000000081525081565b34801561033c57600080fd5b507f0000000000000000000000000000000000000000000000000000000000000000610252565b34801561036f57600080fd5b5061025261037e3660046114cc565b60656020526000908152604090205481565b34801561039c57600080fd5b50610191610b34565b3480156103b157600080fd5b506068546103d29073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101ea565b34801561040357600080fd5b5061019161041236600461140d565b610b48565b34801561042357600080fd5b5060335473ffffffffffffffffffffffffffffffffffffffff166103d2565b34801561044e57600080fd5b5060408051808201909152600481527f574554480000000000000000000000000000000000000000000000000000000060208201526101dd565b34801561049457600080fd5b506101916104a336600461140d565b610b9c565b3480156104b457600080fd5b506102136104c336600461140d565b610c89565b3480156104d457600080fd5b506105076104e3366004611493565b60676020908152600092835260408084209091529082529020805460019091015482565b604080519283526020830191909152016101ea565b34801561052857600080fd5b50610252610537366004611493565b606660209081526000928352604080842090915290825290205481565b34801561056057600080fd5b5061019161056f3660046114cc565b610c9d565b34801561058057600080fd5b5061019161058f36600461140d565b610d51565b33600090815260656020526040812080543492906105b3908490611518565b909155505060405134815233907fe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c9060200160405180910390a2565b33600081815260666020908152604080832073ffffffffffffffffffffffffffffffffffffffff8716808552925280832085905551919290917f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925906106579086815260200190565b60405180910390a350600192915050565b60335473ffffffffffffffffffffffffffffffffffffffff1633146106ee576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601660248201527f44656c61796564574554483a206e6f74206f776e65720000000000000000000060448201526064015b60405180910390fd5b60004782106106fd57476106ff565b815b604051909150339082156108fc029083906000818181858888f1935050505015801561072f573d6000803e3d6000fd5b505050565b73ffffffffffffffffffffffffffffffffffffffff831660009081526065602052604081205482111561076657600080fd5b73ffffffffffffffffffffffffffffffffffffffff841633148015906107dc575073ffffffffffffffffffffffffffffffffffffffff841660009081526066602090815260408083203384529091529020547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff14155b156108645773ffffffffffffffffffffffffffffffffffffffff8416600090815260666020908152604080832033845290915290205482111561081e57600080fd5b73ffffffffffffffffffffffffffffffffffffffff841660009081526066602090815260408083203384529091528120805484929061085e908490611530565b90915550505b73ffffffffffffffffffffffffffffffffffffffff841660009081526065602052604081208054849290610899908490611530565b909155505073ffffffffffffffffffffffffffffffffffffffff8316600090815260656020526040812080548492906108d3908490611518565b925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef8460405161093991815260200190565b60405180910390a35060019392505050565b6109553382610d51565b50565b600054610100900460ff16158080156109785750600054600160ff909116105b806109925750303b158015610992575060005460ff166001145b610a1e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016106e5565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790558015610a7c57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b610a8461109b565b610a8d8361113a565b606880547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff8416179055801561072f57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a1505050565b610b3c6111b1565b610b46600061113a565b565b33600090815260676020908152604080832073ffffffffffffffffffffffffffffffffffffffff861684529091528120426001820155805490918391839190610b92908490611518565b9091555050505050565b60335473ffffffffffffffffffffffffffffffffffffffff163314610c1d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601660248201527f44656c61796564574554483a206e6f74206f776e65720000000000000000000060448201526064016106e5565b73ffffffffffffffffffffffffffffffffffffffff821660008181526066602090815260408083203380855290835292819020859055518481529192917f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925910160405180910390a35050565b6000610c96338484610734565b9392505050565b610ca56111b1565b73ffffffffffffffffffffffffffffffffffffffff8116610d48576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084016106e5565b6109558161113a565b606860009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16635c975abb6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610dbe573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610de29190611547565b15610e49576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f44656c61796564574554483a20636f6e7472616374206973207061757365640060448201526064016106e5565b33600090815260676020908152604080832073ffffffffffffffffffffffffffffffffffffffff8616845290915290208054821115610f0a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602d60248201527f44656c61796564574554483a20696e73756666696369656e7420756e6c6f636b60448201527f6564207769746864726177616c0000000000000000000000000000000000000060648201526084016106e5565b6000816001015411610f9d576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526024808201527f44656c61796564574554483a207769746864726177616c206e6f7420756e6c6f60448201527f636b65640000000000000000000000000000000000000000000000000000000060648201526084016106e5565b427f00000000000000000000000000000000000000000000000000000000000000008260010154610fce9190611518565b111561105c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f44656c61796564574554483a207769746864726177616c2064656c6179206e6f60448201527f74206d657400000000000000000000000000000000000000000000000000000060648201526084016106e5565b818160000160008282546110709190611530565b9091555061072f905082611232565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b600054610100900460ff16611132576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016106e5565b610b466112d8565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b60335473ffffffffffffffffffffffffffffffffffffffff163314610b46576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016106e5565b3360009081526065602052604090205481111561124e57600080fd5b336000908152606560205260408120805483929061126d908490611530565b9091555050604051339082156108fc029083906000818181858888f1935050505015801561129f573d6000803e3d6000fd5b5060405181815233907f7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b659060200160405180910390a250565b600054610100900460ff1661136f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016106e5565b610b463361113a565b600060208083528351808285015260005b818110156113a557858101830151858201604001528201611389565b818111156113b7576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461095557600080fd5b6000806040838503121561142057600080fd5b823561142b816113eb565b946020939093013593505050565b60006020828403121561144b57600080fd5b5035919050565b60008060006060848603121561146757600080fd5b8335611472816113eb565b92506020840135611482816113eb565b929592945050506040919091013590565b600080604083850312156114a657600080fd5b82356114b1816113eb565b915060208301356114c1816113eb565b809150509250929050565b6000602082840312156114de57600080fd5b8135610c96816113eb565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000821982111561152b5761152b6114e9565b500190565b600082821015611542576115426114e9565b500390565b60006020828403121561155957600080fd5b81518015158114610c9657600080fdfea164736f6c634300080f000a496e697469616c697a61626c653a20636f6e7472616374206973206e6f742069",
}

// DelayedWETHABI is the input ABI used to generate the binding from.
// Deprecated: Use DelayedWETHMetaData.ABI instead.
var DelayedWETHABI = DelayedWETHMetaData.ABI

// DelayedWETHBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DelayedWETHMetaData.Bin instead.
var DelayedWETHBin = DelayedWETHMetaData.Bin

// DeployDelayedWETH deploys a new Ethereum contract, binding an instance of DelayedWETH to it.
func DeployDelayedWETH(auth *bind.TransactOpts, backend bind.ContractBackend, _delay *big.Int) (common.Address, *types.Transaction, *DelayedWETH, error) {
	parsed, err := DelayedWETHMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DelayedWETHBin), backend, _delay)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DelayedWETH{DelayedWETHCaller: DelayedWETHCaller{contract: contract}, DelayedWETHTransactor: DelayedWETHTransactor{contract: contract}, DelayedWETHFilterer: DelayedWETHFilterer{contract: contract}}, nil
}

// DelayedWETH is an auto generated Go binding around an Ethereum contract.
type DelayedWETH struct {
	DelayedWETHCaller     // Read-only binding to the contract
	DelayedWETHTransactor // Write-only binding to the contract
	DelayedWETHFilterer   // Log filterer for contract events
}

// DelayedWETHCaller is an auto generated read-only Go binding around an Ethereum contract.
type DelayedWETHCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelayedWETHTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DelayedWETHTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelayedWETHFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DelayedWETHFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelayedWETHSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DelayedWETHSession struct {
	Contract     *DelayedWETH      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DelayedWETHCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DelayedWETHCallerSession struct {
	Contract *DelayedWETHCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// DelayedWETHTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DelayedWETHTransactorSession struct {
	Contract     *DelayedWETHTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// DelayedWETHRaw is an auto generated low-level Go binding around an Ethereum contract.
type DelayedWETHRaw struct {
	Contract *DelayedWETH // Generic contract binding to access the raw methods on
}

// DelayedWETHCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DelayedWETHCallerRaw struct {
	Contract *DelayedWETHCaller // Generic read-only contract binding to access the raw methods on
}

// DelayedWETHTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DelayedWETHTransactorRaw struct {
	Contract *DelayedWETHTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDelayedWETH creates a new instance of DelayedWETH, bound to a specific deployed contract.
func NewDelayedWETH(address common.Address, backend bind.ContractBackend) (*DelayedWETH, error) {
	contract, err := bindDelayedWETH(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DelayedWETH{DelayedWETHCaller: DelayedWETHCaller{contract: contract}, DelayedWETHTransactor: DelayedWETHTransactor{contract: contract}, DelayedWETHFilterer: DelayedWETHFilterer{contract: contract}}, nil
}

// NewDelayedWETHCaller creates a new read-only instance of DelayedWETH, bound to a specific deployed contract.
func NewDelayedWETHCaller(address common.Address, caller bind.ContractCaller) (*DelayedWETHCaller, error) {
	contract, err := bindDelayedWETH(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DelayedWETHCaller{contract: contract}, nil
}

// NewDelayedWETHTransactor creates a new write-only instance of DelayedWETH, bound to a specific deployed contract.
func NewDelayedWETHTransactor(address common.Address, transactor bind.ContractTransactor) (*DelayedWETHTransactor, error) {
	contract, err := bindDelayedWETH(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DelayedWETHTransactor{contract: contract}, nil
}

// NewDelayedWETHFilterer creates a new log filterer instance of DelayedWETH, bound to a specific deployed contract.
func NewDelayedWETHFilterer(address common.Address, filterer bind.ContractFilterer) (*DelayedWETHFilterer, error) {
	contract, err := bindDelayedWETH(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DelayedWETHFilterer{contract: contract}, nil
}

// bindDelayedWETH binds a generic wrapper to an already deployed contract.
func bindDelayedWETH(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DelayedWETHABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelayedWETH *DelayedWETHRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelayedWETH.Contract.DelayedWETHCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelayedWETH *DelayedWETHRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedWETH.Contract.DelayedWETHTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelayedWETH *DelayedWETHRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelayedWETH.Contract.DelayedWETHTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelayedWETH *DelayedWETHCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelayedWETH.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelayedWETH *DelayedWETHTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedWETH.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelayedWETH *DelayedWETHTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelayedWETH.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_DelayedWETH *DelayedWETHCaller) Allowance(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "allowance", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_DelayedWETH *DelayedWETHSession) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _DelayedWETH.Contract.Allowance(&_DelayedWETH.CallOpts, arg0, arg1)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_DelayedWETH *DelayedWETHCallerSession) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _DelayedWETH.Contract.Allowance(&_DelayedWETH.CallOpts, arg0, arg1)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_DelayedWETH *DelayedWETHCaller) BalanceOf(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "balanceOf", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_DelayedWETH *DelayedWETHSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _DelayedWETH.Contract.BalanceOf(&_DelayedWETH.CallOpts, arg0)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_DelayedWETH *DelayedWETHCallerSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _DelayedWETH.Contract.BalanceOf(&_DelayedWETH.CallOpts, arg0)
}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_DelayedWETH *DelayedWETHCaller) Config(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "config")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_DelayedWETH *DelayedWETHSession) Config() (common.Address, error) {
	return _DelayedWETH.Contract.Config(&_DelayedWETH.CallOpts)
}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_DelayedWETH *DelayedWETHCallerSession) Config() (common.Address, error) {
	return _DelayedWETH.Contract.Config(&_DelayedWETH.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_DelayedWETH *DelayedWETHCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_DelayedWETH *DelayedWETHSession) Decimals() (uint8, error) {
	return _DelayedWETH.Contract.Decimals(&_DelayedWETH.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_DelayedWETH *DelayedWETHCallerSession) Decimals() (uint8, error) {
	return _DelayedWETH.Contract.Decimals(&_DelayedWETH.CallOpts)
}

// Delay is a free data retrieval call binding the contract method 0x6a42b8f8.
//
// Solidity: function delay() view returns(uint256)
func (_DelayedWETH *DelayedWETHCaller) Delay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "delay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Delay is a free data retrieval call binding the contract method 0x6a42b8f8.
//
// Solidity: function delay() view returns(uint256)
func (_DelayedWETH *DelayedWETHSession) Delay() (*big.Int, error) {
	return _DelayedWETH.Contract.Delay(&_DelayedWETH.CallOpts)
}

// Delay is a free data retrieval call binding the contract method 0x6a42b8f8.
//
// Solidity: function delay() view returns(uint256)
func (_DelayedWETH *DelayedWETHCallerSession) Delay() (*big.Int, error) {
	return _DelayedWETH.Contract.Delay(&_DelayedWETH.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_DelayedWETH *DelayedWETHCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_DelayedWETH *DelayedWETHSession) Name() (string, error) {
	return _DelayedWETH.Contract.Name(&_DelayedWETH.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_DelayedWETH *DelayedWETHCallerSession) Name() (string, error) {
	return _DelayedWETH.Contract.Name(&_DelayedWETH.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DelayedWETH *DelayedWETHCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DelayedWETH *DelayedWETHSession) Owner() (common.Address, error) {
	return _DelayedWETH.Contract.Owner(&_DelayedWETH.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DelayedWETH *DelayedWETHCallerSession) Owner() (common.Address, error) {
	return _DelayedWETH.Contract.Owner(&_DelayedWETH.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_DelayedWETH *DelayedWETHCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_DelayedWETH *DelayedWETHSession) Symbol() (string, error) {
	return _DelayedWETH.Contract.Symbol(&_DelayedWETH.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_DelayedWETH *DelayedWETHCallerSession) Symbol() (string, error) {
	return _DelayedWETH.Contract.Symbol(&_DelayedWETH.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_DelayedWETH *DelayedWETHCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_DelayedWETH *DelayedWETHSession) TotalSupply() (*big.Int, error) {
	return _DelayedWETH.Contract.TotalSupply(&_DelayedWETH.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_DelayedWETH *DelayedWETHCallerSession) TotalSupply() (*big.Int, error) {
	return _DelayedWETH.Contract.TotalSupply(&_DelayedWETH.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DelayedWETH *DelayedWETHCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DelayedWETH *DelayedWETHSession) Version() (string, error) {
	return _DelayedWETH.Contract.Version(&_DelayedWETH.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DelayedWETH *DelayedWETHCallerSession) Version() (string, error) {
	return _DelayedWETH.Contract.Version(&_DelayedWETH.CallOpts)
}

// Withdrawals is a free data retrieval call binding the contract method 0xcd47bde1.
//
// Solidity: function withdrawals(address , address ) view returns(uint256 amount, uint256 timestamp)
func (_DelayedWETH *DelayedWETHCaller) Withdrawals(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (struct {
	Amount    *big.Int
	Timestamp *big.Int
}, error) {
	var out []interface{}
	err := _DelayedWETH.contract.Call(opts, &out, "withdrawals", arg0, arg1)

	outstruct := new(struct {
		Amount    *big.Int
		Timestamp *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Amount = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Timestamp = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Withdrawals is a free data retrieval call binding the contract method 0xcd47bde1.
//
// Solidity: function withdrawals(address , address ) view returns(uint256 amount, uint256 timestamp)
func (_DelayedWETH *DelayedWETHSession) Withdrawals(arg0 common.Address, arg1 common.Address) (struct {
	Amount    *big.Int
	Timestamp *big.Int
}, error) {
	return _DelayedWETH.Contract.Withdrawals(&_DelayedWETH.CallOpts, arg0, arg1)
}

// Withdrawals is a free data retrieval call binding the contract method 0xcd47bde1.
//
// Solidity: function withdrawals(address , address ) view returns(uint256 amount, uint256 timestamp)
func (_DelayedWETH *DelayedWETHCallerSession) Withdrawals(arg0 common.Address, arg1 common.Address) (struct {
	Amount    *big.Int
	Timestamp *big.Int
}, error) {
	return _DelayedWETH.Contract.Withdrawals(&_DelayedWETH.CallOpts, arg0, arg1)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address guy, uint256 wad) returns(bool)
func (_DelayedWETH *DelayedWETHTransactor) Approve(opts *bind.TransactOpts, guy common.Address, wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "approve", guy, wad)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address guy, uint256 wad) returns(bool)
func (_DelayedWETH *DelayedWETHSession) Approve(guy common.Address, wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Approve(&_DelayedWETH.TransactOpts, guy, wad)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address guy, uint256 wad) returns(bool)
func (_DelayedWETH *DelayedWETHTransactorSession) Approve(guy common.Address, wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Approve(&_DelayedWETH.TransactOpts, guy, wad)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_DelayedWETH *DelayedWETHTransactor) Deposit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "deposit")
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_DelayedWETH *DelayedWETHSession) Deposit() (*types.Transaction, error) {
	return _DelayedWETH.Contract.Deposit(&_DelayedWETH.TransactOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_DelayedWETH *DelayedWETHTransactorSession) Deposit() (*types.Transaction, error) {
	return _DelayedWETH.Contract.Deposit(&_DelayedWETH.TransactOpts)
}

// Hold is a paid mutator transaction binding the contract method 0x977a5ec5.
//
// Solidity: function hold(address _guy, uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHTransactor) Hold(opts *bind.TransactOpts, _guy common.Address, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "hold", _guy, _wad)
}

// Hold is a paid mutator transaction binding the contract method 0x977a5ec5.
//
// Solidity: function hold(address _guy, uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHSession) Hold(_guy common.Address, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Hold(&_DelayedWETH.TransactOpts, _guy, _wad)
}

// Hold is a paid mutator transaction binding the contract method 0x977a5ec5.
//
// Solidity: function hold(address _guy, uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHTransactorSession) Hold(_guy common.Address, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Hold(&_DelayedWETH.TransactOpts, _guy, _wad)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _owner, address _config) returns()
func (_DelayedWETH *DelayedWETHTransactor) Initialize(opts *bind.TransactOpts, _owner common.Address, _config common.Address) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "initialize", _owner, _config)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _owner, address _config) returns()
func (_DelayedWETH *DelayedWETHSession) Initialize(_owner common.Address, _config common.Address) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Initialize(&_DelayedWETH.TransactOpts, _owner, _config)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _owner, address _config) returns()
func (_DelayedWETH *DelayedWETHTransactorSession) Initialize(_owner common.Address, _config common.Address) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Initialize(&_DelayedWETH.TransactOpts, _owner, _config)
}

// Recover is a paid mutator transaction binding the contract method 0x0ca35682.
//
// Solidity: function recover(uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHTransactor) Recover(opts *bind.TransactOpts, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "recover", _wad)
}

// Recover is a paid mutator transaction binding the contract method 0x0ca35682.
//
// Solidity: function recover(uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHSession) Recover(_wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Recover(&_DelayedWETH.TransactOpts, _wad)
}

// Recover is a paid mutator transaction binding the contract method 0x0ca35682.
//
// Solidity: function recover(uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHTransactorSession) Recover(_wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Recover(&_DelayedWETH.TransactOpts, _wad)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DelayedWETH *DelayedWETHTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DelayedWETH *DelayedWETHSession) RenounceOwnership() (*types.Transaction, error) {
	return _DelayedWETH.Contract.RenounceOwnership(&_DelayedWETH.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DelayedWETH *DelayedWETHTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _DelayedWETH.Contract.RenounceOwnership(&_DelayedWETH.TransactOpts)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address dst, uint256 wad) returns(bool)
func (_DelayedWETH *DelayedWETHTransactor) Transfer(opts *bind.TransactOpts, dst common.Address, wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "transfer", dst, wad)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address dst, uint256 wad) returns(bool)
func (_DelayedWETH *DelayedWETHSession) Transfer(dst common.Address, wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Transfer(&_DelayedWETH.TransactOpts, dst, wad)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address dst, uint256 wad) returns(bool)
func (_DelayedWETH *DelayedWETHTransactorSession) Transfer(dst common.Address, wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Transfer(&_DelayedWETH.TransactOpts, dst, wad)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address src, address dst, uint256 wad) returns(bool)
func (_DelayedWETH *DelayedWETHTransactor) TransferFrom(opts *bind.TransactOpts, src common.Address, dst common.Address, wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "transferFrom", src, dst, wad)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address src, address dst, uint256 wad) returns(bool)
func (_DelayedWETH *DelayedWETHSession) TransferFrom(src common.Address, dst common.Address, wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.TransferFrom(&_DelayedWETH.TransactOpts, src, dst, wad)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address src, address dst, uint256 wad) returns(bool)
func (_DelayedWETH *DelayedWETHTransactorSession) TransferFrom(src common.Address, dst common.Address, wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.TransferFrom(&_DelayedWETH.TransactOpts, src, dst, wad)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DelayedWETH *DelayedWETHTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DelayedWETH *DelayedWETHSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DelayedWETH.Contract.TransferOwnership(&_DelayedWETH.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DelayedWETH *DelayedWETHTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DelayedWETH.Contract.TransferOwnership(&_DelayedWETH.TransactOpts, newOwner)
}

// Unlock is a paid mutator transaction binding the contract method 0x7eee288d.
//
// Solidity: function unlock(address _guy, uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHTransactor) Unlock(opts *bind.TransactOpts, _guy common.Address, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "unlock", _guy, _wad)
}

// Unlock is a paid mutator transaction binding the contract method 0x7eee288d.
//
// Solidity: function unlock(address _guy, uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHSession) Unlock(_guy common.Address, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Unlock(&_DelayedWETH.TransactOpts, _guy, _wad)
}

// Unlock is a paid mutator transaction binding the contract method 0x7eee288d.
//
// Solidity: function unlock(address _guy, uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHTransactorSession) Unlock(_guy common.Address, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Unlock(&_DelayedWETH.TransactOpts, _guy, _wad)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHTransactor) Withdraw(opts *bind.TransactOpts, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "withdraw", _wad)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHSession) Withdraw(_wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Withdraw(&_DelayedWETH.TransactOpts, _wad)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHTransactorSession) Withdraw(_wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Withdraw(&_DelayedWETH.TransactOpts, _wad)
}

// Withdraw0 is a paid mutator transaction binding the contract method 0xf3fef3a3.
//
// Solidity: function withdraw(address _guy, uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHTransactor) Withdraw0(opts *bind.TransactOpts, _guy common.Address, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.contract.Transact(opts, "withdraw0", _guy, _wad)
}

// Withdraw0 is a paid mutator transaction binding the contract method 0xf3fef3a3.
//
// Solidity: function withdraw(address _guy, uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHSession) Withdraw0(_guy common.Address, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Withdraw0(&_DelayedWETH.TransactOpts, _guy, _wad)
}

// Withdraw0 is a paid mutator transaction binding the contract method 0xf3fef3a3.
//
// Solidity: function withdraw(address _guy, uint256 _wad) returns()
func (_DelayedWETH *DelayedWETHTransactorSession) Withdraw0(_guy common.Address, _wad *big.Int) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Withdraw0(&_DelayedWETH.TransactOpts, _guy, _wad)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_DelayedWETH *DelayedWETHTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _DelayedWETH.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_DelayedWETH *DelayedWETHSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Fallback(&_DelayedWETH.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_DelayedWETH *DelayedWETHTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _DelayedWETH.Contract.Fallback(&_DelayedWETH.TransactOpts, calldata)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_DelayedWETH *DelayedWETHTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedWETH.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_DelayedWETH *DelayedWETHSession) Receive() (*types.Transaction, error) {
	return _DelayedWETH.Contract.Receive(&_DelayedWETH.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_DelayedWETH *DelayedWETHTransactorSession) Receive() (*types.Transaction, error) {
	return _DelayedWETH.Contract.Receive(&_DelayedWETH.TransactOpts)
}

// DelayedWETHApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the DelayedWETH contract.
type DelayedWETHApprovalIterator struct {
	Event *DelayedWETHApproval // Event containing the contract specifics and raw log

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
func (it *DelayedWETHApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelayedWETHApproval)
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
		it.Event = new(DelayedWETHApproval)
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
func (it *DelayedWETHApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelayedWETHApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelayedWETHApproval represents a Approval event raised by the DelayedWETH contract.
type DelayedWETHApproval struct {
	Src common.Address
	Guy common.Address
	Wad *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed src, address indexed guy, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) FilterApproval(opts *bind.FilterOpts, src []common.Address, guy []common.Address) (*DelayedWETHApprovalIterator, error) {

	var srcRule []interface{}
	for _, srcItem := range src {
		srcRule = append(srcRule, srcItem)
	}
	var guyRule []interface{}
	for _, guyItem := range guy {
		guyRule = append(guyRule, guyItem)
	}

	logs, sub, err := _DelayedWETH.contract.FilterLogs(opts, "Approval", srcRule, guyRule)
	if err != nil {
		return nil, err
	}
	return &DelayedWETHApprovalIterator{contract: _DelayedWETH.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed src, address indexed guy, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *DelayedWETHApproval, src []common.Address, guy []common.Address) (event.Subscription, error) {

	var srcRule []interface{}
	for _, srcItem := range src {
		srcRule = append(srcRule, srcItem)
	}
	var guyRule []interface{}
	for _, guyItem := range guy {
		guyRule = append(guyRule, guyItem)
	}

	logs, sub, err := _DelayedWETH.contract.WatchLogs(opts, "Approval", srcRule, guyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelayedWETHApproval)
				if err := _DelayedWETH.contract.UnpackLog(event, "Approval", log); err != nil {
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
// Solidity: event Approval(address indexed src, address indexed guy, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) ParseApproval(log types.Log) (*DelayedWETHApproval, error) {
	event := new(DelayedWETHApproval)
	if err := _DelayedWETH.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelayedWETHDepositIterator is returned from FilterDeposit and is used to iterate over the raw logs and unpacked data for Deposit events raised by the DelayedWETH contract.
type DelayedWETHDepositIterator struct {
	Event *DelayedWETHDeposit // Event containing the contract specifics and raw log

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
func (it *DelayedWETHDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelayedWETHDeposit)
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
		it.Event = new(DelayedWETHDeposit)
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
func (it *DelayedWETHDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelayedWETHDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelayedWETHDeposit represents a Deposit event raised by the DelayedWETH contract.
type DelayedWETHDeposit struct {
	Dst common.Address
	Wad *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterDeposit is a free log retrieval operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed dst, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) FilterDeposit(opts *bind.FilterOpts, dst []common.Address) (*DelayedWETHDepositIterator, error) {

	var dstRule []interface{}
	for _, dstItem := range dst {
		dstRule = append(dstRule, dstItem)
	}

	logs, sub, err := _DelayedWETH.contract.FilterLogs(opts, "Deposit", dstRule)
	if err != nil {
		return nil, err
	}
	return &DelayedWETHDepositIterator{contract: _DelayedWETH.contract, event: "Deposit", logs: logs, sub: sub}, nil
}

// WatchDeposit is a free log subscription operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed dst, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) WatchDeposit(opts *bind.WatchOpts, sink chan<- *DelayedWETHDeposit, dst []common.Address) (event.Subscription, error) {

	var dstRule []interface{}
	for _, dstItem := range dst {
		dstRule = append(dstRule, dstItem)
	}

	logs, sub, err := _DelayedWETH.contract.WatchLogs(opts, "Deposit", dstRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelayedWETHDeposit)
				if err := _DelayedWETH.contract.UnpackLog(event, "Deposit", log); err != nil {
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

// ParseDeposit is a log parse operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed dst, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) ParseDeposit(log types.Log) (*DelayedWETHDeposit, error) {
	event := new(DelayedWETHDeposit)
	if err := _DelayedWETH.contract.UnpackLog(event, "Deposit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelayedWETHInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the DelayedWETH contract.
type DelayedWETHInitializedIterator struct {
	Event *DelayedWETHInitialized // Event containing the contract specifics and raw log

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
func (it *DelayedWETHInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelayedWETHInitialized)
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
		it.Event = new(DelayedWETHInitialized)
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
func (it *DelayedWETHInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelayedWETHInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelayedWETHInitialized represents a Initialized event raised by the DelayedWETH contract.
type DelayedWETHInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_DelayedWETH *DelayedWETHFilterer) FilterInitialized(opts *bind.FilterOpts) (*DelayedWETHInitializedIterator, error) {

	logs, sub, err := _DelayedWETH.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &DelayedWETHInitializedIterator{contract: _DelayedWETH.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_DelayedWETH *DelayedWETHFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *DelayedWETHInitialized) (event.Subscription, error) {

	logs, sub, err := _DelayedWETH.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelayedWETHInitialized)
				if err := _DelayedWETH.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_DelayedWETH *DelayedWETHFilterer) ParseInitialized(log types.Log) (*DelayedWETHInitialized, error) {
	event := new(DelayedWETHInitialized)
	if err := _DelayedWETH.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelayedWETHOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the DelayedWETH contract.
type DelayedWETHOwnershipTransferredIterator struct {
	Event *DelayedWETHOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *DelayedWETHOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelayedWETHOwnershipTransferred)
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
		it.Event = new(DelayedWETHOwnershipTransferred)
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
func (it *DelayedWETHOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelayedWETHOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelayedWETHOwnershipTransferred represents a OwnershipTransferred event raised by the DelayedWETH contract.
type DelayedWETHOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DelayedWETH *DelayedWETHFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*DelayedWETHOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DelayedWETH.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &DelayedWETHOwnershipTransferredIterator{contract: _DelayedWETH.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DelayedWETH *DelayedWETHFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *DelayedWETHOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DelayedWETH.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelayedWETHOwnershipTransferred)
				if err := _DelayedWETH.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_DelayedWETH *DelayedWETHFilterer) ParseOwnershipTransferred(log types.Log) (*DelayedWETHOwnershipTransferred, error) {
	event := new(DelayedWETHOwnershipTransferred)
	if err := _DelayedWETH.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelayedWETHTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the DelayedWETH contract.
type DelayedWETHTransferIterator struct {
	Event *DelayedWETHTransfer // Event containing the contract specifics and raw log

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
func (it *DelayedWETHTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelayedWETHTransfer)
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
		it.Event = new(DelayedWETHTransfer)
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
func (it *DelayedWETHTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelayedWETHTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelayedWETHTransfer represents a Transfer event raised by the DelayedWETH contract.
type DelayedWETHTransfer struct {
	Src common.Address
	Dst common.Address
	Wad *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed src, address indexed dst, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) FilterTransfer(opts *bind.FilterOpts, src []common.Address, dst []common.Address) (*DelayedWETHTransferIterator, error) {

	var srcRule []interface{}
	for _, srcItem := range src {
		srcRule = append(srcRule, srcItem)
	}
	var dstRule []interface{}
	for _, dstItem := range dst {
		dstRule = append(dstRule, dstItem)
	}

	logs, sub, err := _DelayedWETH.contract.FilterLogs(opts, "Transfer", srcRule, dstRule)
	if err != nil {
		return nil, err
	}
	return &DelayedWETHTransferIterator{contract: _DelayedWETH.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed src, address indexed dst, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *DelayedWETHTransfer, src []common.Address, dst []common.Address) (event.Subscription, error) {

	var srcRule []interface{}
	for _, srcItem := range src {
		srcRule = append(srcRule, srcItem)
	}
	var dstRule []interface{}
	for _, dstItem := range dst {
		dstRule = append(dstRule, dstItem)
	}

	logs, sub, err := _DelayedWETH.contract.WatchLogs(opts, "Transfer", srcRule, dstRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelayedWETHTransfer)
				if err := _DelayedWETH.contract.UnpackLog(event, "Transfer", log); err != nil {
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
// Solidity: event Transfer(address indexed src, address indexed dst, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) ParseTransfer(log types.Log) (*DelayedWETHTransfer, error) {
	event := new(DelayedWETHTransfer)
	if err := _DelayedWETH.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelayedWETHUnwrapIterator is returned from FilterUnwrap and is used to iterate over the raw logs and unpacked data for Unwrap events raised by the DelayedWETH contract.
type DelayedWETHUnwrapIterator struct {
	Event *DelayedWETHUnwrap // Event containing the contract specifics and raw log

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
func (it *DelayedWETHUnwrapIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelayedWETHUnwrap)
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
		it.Event = new(DelayedWETHUnwrap)
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
func (it *DelayedWETHUnwrapIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelayedWETHUnwrapIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelayedWETHUnwrap represents a Unwrap event raised by the DelayedWETH contract.
type DelayedWETHUnwrap struct {
	Src common.Address
	Wad *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterUnwrap is a free log retrieval operation binding the contract event 0x5dd085b6070b4cae004f84daafd199fd55b0bdfa11c3a802baffe89c2419d8c2.
//
// Solidity: event Unwrap(address indexed src, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) FilterUnwrap(opts *bind.FilterOpts, src []common.Address) (*DelayedWETHUnwrapIterator, error) {

	var srcRule []interface{}
	for _, srcItem := range src {
		srcRule = append(srcRule, srcItem)
	}

	logs, sub, err := _DelayedWETH.contract.FilterLogs(opts, "Unwrap", srcRule)
	if err != nil {
		return nil, err
	}
	return &DelayedWETHUnwrapIterator{contract: _DelayedWETH.contract, event: "Unwrap", logs: logs, sub: sub}, nil
}

// WatchUnwrap is a free log subscription operation binding the contract event 0x5dd085b6070b4cae004f84daafd199fd55b0bdfa11c3a802baffe89c2419d8c2.
//
// Solidity: event Unwrap(address indexed src, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) WatchUnwrap(opts *bind.WatchOpts, sink chan<- *DelayedWETHUnwrap, src []common.Address) (event.Subscription, error) {

	var srcRule []interface{}
	for _, srcItem := range src {
		srcRule = append(srcRule, srcItem)
	}

	logs, sub, err := _DelayedWETH.contract.WatchLogs(opts, "Unwrap", srcRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelayedWETHUnwrap)
				if err := _DelayedWETH.contract.UnpackLog(event, "Unwrap", log); err != nil {
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

// ParseUnwrap is a log parse operation binding the contract event 0x5dd085b6070b4cae004f84daafd199fd55b0bdfa11c3a802baffe89c2419d8c2.
//
// Solidity: event Unwrap(address indexed src, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) ParseUnwrap(log types.Log) (*DelayedWETHUnwrap, error) {
	event := new(DelayedWETHUnwrap)
	if err := _DelayedWETH.contract.UnpackLog(event, "Unwrap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelayedWETHWithdrawalIterator is returned from FilterWithdrawal and is used to iterate over the raw logs and unpacked data for Withdrawal events raised by the DelayedWETH contract.
type DelayedWETHWithdrawalIterator struct {
	Event *DelayedWETHWithdrawal // Event containing the contract specifics and raw log

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
func (it *DelayedWETHWithdrawalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelayedWETHWithdrawal)
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
		it.Event = new(DelayedWETHWithdrawal)
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
func (it *DelayedWETHWithdrawalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelayedWETHWithdrawalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelayedWETHWithdrawal represents a Withdrawal event raised by the DelayedWETH contract.
type DelayedWETHWithdrawal struct {
	Src common.Address
	Wad *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal is a free log retrieval operation binding the contract event 0x7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65.
//
// Solidity: event Withdrawal(address indexed src, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) FilterWithdrawal(opts *bind.FilterOpts, src []common.Address) (*DelayedWETHWithdrawalIterator, error) {

	var srcRule []interface{}
	for _, srcItem := range src {
		srcRule = append(srcRule, srcItem)
	}

	logs, sub, err := _DelayedWETH.contract.FilterLogs(opts, "Withdrawal", srcRule)
	if err != nil {
		return nil, err
	}
	return &DelayedWETHWithdrawalIterator{contract: _DelayedWETH.contract, event: "Withdrawal", logs: logs, sub: sub}, nil
}

// WatchWithdrawal is a free log subscription operation binding the contract event 0x7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65.
//
// Solidity: event Withdrawal(address indexed src, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) WatchWithdrawal(opts *bind.WatchOpts, sink chan<- *DelayedWETHWithdrawal, src []common.Address) (event.Subscription, error) {

	var srcRule []interface{}
	for _, srcItem := range src {
		srcRule = append(srcRule, srcItem)
	}

	logs, sub, err := _DelayedWETH.contract.WatchLogs(opts, "Withdrawal", srcRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelayedWETHWithdrawal)
				if err := _DelayedWETH.contract.UnpackLog(event, "Withdrawal", log); err != nil {
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

// ParseWithdrawal is a log parse operation binding the contract event 0x7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65.
//
// Solidity: event Withdrawal(address indexed src, uint256 wad)
func (_DelayedWETH *DelayedWETHFilterer) ParseWithdrawal(log types.Log) (*DelayedWETHWithdrawal, error) {
	event := new(DelayedWETHWithdrawal)
	if err := _DelayedWETH.contract.UnpackLog(event, "Withdrawal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
