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

// ProxyAdminMetaData contains all meta data concerning the ProxyAdmin contract.
var ProxyAdminMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"addressManager\",\"outputs\":[{\"internalType\":\"contractAddressManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_newAdmin\",\"type\":\"address\"}],\"name\":\"changeProxyAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_proxy\",\"type\":\"address\"}],\"name\":\"getProxyAdmin\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_proxy\",\"type\":\"address\"}],\"name\":\"getProxyImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"implementationName\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isUpgrading\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"proxyType\",\"outputs\":[{\"internalType\":\"enumProxyAdmin.ProxyType\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"setAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractAddressManager\",\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"setAddressManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_address\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"setImplementationName\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_address\",\"type\":\"address\"},{\"internalType\":\"enumProxyAdmin.ProxyType\",\"name\":\"_type\",\"type\":\"uint8\"}],\"name\":\"setProxyType\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_upgrading\",\"type\":\"bool\"}],\"name\":\"setUpgrading\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_implementation\",\"type\":\"address\"}],\"name\":\"upgrade\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_implementation\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"upgradeAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b5060405162001a5f38038062001a5f8339810160408190526200003491620000a1565b6200003f3362000051565b6200004a8162000051565b50620000d3565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b600060208284031215620000b457600080fd5b81516001600160a01b0381168114620000cc57600080fd5b9392505050565b61197c80620000e36000396000f3fe60806040526004361061010e5760003560e01c8063860f7cda116100a557806399a88ec411610074578063b794726211610059578063b794726214610329578063f2fde38b14610364578063f3b7dead1461038457600080fd5b806399a88ec4146102e95780639b2ea4bd1461030957600080fd5b8063860f7cda1461026b5780638d52d4a01461028b5780638da5cb5b146102ab5780639623609d146102d657600080fd5b80633ab76e9f116100e15780633ab76e9f146101cc5780636bd9f516146101f9578063715018a6146102365780637eff275e1461024b57600080fd5b80630652b57a1461011357806307c8f7b014610135578063204e1c7a14610155578063238181ae1461019f575b600080fd5b34801561011f57600080fd5b5061013361012e3660046111f9565b6103a4565b005b34801561014157600080fd5b50610133610150366004611216565b6103f3565b34801561016157600080fd5b506101756101703660046111f9565b610445565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b3480156101ab57600080fd5b506101bf6101ba3660046111f9565b61066b565b60405161019691906112ae565b3480156101d857600080fd5b506003546101759073ffffffffffffffffffffffffffffffffffffffff1681565b34801561020557600080fd5b506102296102143660046111f9565b60016020526000908152604090205460ff1681565b60405161019691906112f0565b34801561024257600080fd5b50610133610705565b34801561025757600080fd5b50610133610266366004611331565b610719565b34801561027757600080fd5b5061013361028636600461148c565b6108cc565b34801561029757600080fd5b506101336102a63660046114dc565b610903565b3480156102b757600080fd5b5060005473ffffffffffffffffffffffffffffffffffffffff16610175565b6101336102e436600461150e565b610977565b3480156102f557600080fd5b50610133610304366004611331565b610b8e565b34801561031557600080fd5b50610133610324366004611584565b610e1e565b34801561033557600080fd5b5060035474010000000000000000000000000000000000000000900460ff166040519015158152602001610196565b34801561037057600080fd5b5061013361037f3660046111f9565b610eb4565b34801561039057600080fd5b5061017561039f3660046111f9565b610f6b565b6103ac6110e1565b600380547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b6103fb6110e1565b6003805491151574010000000000000000000000000000000000000000027fffffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffff909216919091179055565b73ffffffffffffffffffffffffffffffffffffffff811660009081526001602052604081205460ff1681816002811115610481576104816112c1565b036104fc578273ffffffffffffffffffffffffffffffffffffffff16635c60da1b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156104d1573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906104f591906115cb565b9392505050565b6001816002811115610510576105106112c1565b03610560578273ffffffffffffffffffffffffffffffffffffffff1663aaf10f426040518163ffffffff1660e01b8152600401602060405180830381865afa1580156104d1573d6000803e3d6000fd5b6002816002811115610574576105746112c1565b036105fe5760035473ffffffffffffffffffffffffffffffffffffffff8481166000908152600260205260409081902090517fbf40fac1000000000000000000000000000000000000000000000000000000008152919092169163bf40fac1916105e19190600401611635565b602060405180830381865afa1580156104d1573d6000803e3d6000fd5b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601e60248201527f50726f787941646d696e3a20756e6b6e6f776e2070726f78792074797065000060448201526064015b60405180910390fd5b50919050565b60026020526000908152604090208054610684906115e8565b80601f01602080910402602001604051908101604052809291908181526020018280546106b0906115e8565b80156106fd5780601f106106d2576101008083540402835291602001916106fd565b820191906000526020600020905b8154815290600101906020018083116106e057829003601f168201915b505050505081565b61070d6110e1565b6107176000611162565b565b6107216110e1565b73ffffffffffffffffffffffffffffffffffffffff821660009081526001602052604081205460ff169081600281111561075d5761075d6112c1565b036107e9576040517f8f28397000000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8381166004830152841690638f283970906024015b600060405180830381600087803b1580156107cc57600080fd5b505af11580156107e0573d6000803e3d6000fd5b50505050505050565b60018160028111156107fd576107fd6112c1565b03610856576040517f13af403500000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff83811660048301528416906313af4035906024016107b2565b600281600281111561086a5761086a6112c1565b036105fe576003546040517ff2fde38b00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff84811660048301529091169063f2fde38b906024016107b2565b505050565b6108d46110e1565b73ffffffffffffffffffffffffffffffffffffffff821660009081526002602052604090206108c78282611724565b61090b6110e1565b73ffffffffffffffffffffffffffffffffffffffff82166000908152600160208190526040909120805483927fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff009091169083600281111561096e5761096e6112c1565b02179055505050565b61097f6110e1565b73ffffffffffffffffffffffffffffffffffffffff831660009081526001602052604081205460ff16908160028111156109bb576109bb6112c1565b03610a81576040517f4f1ef28600000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff851690634f1ef286903490610a16908790879060040161183e565b60006040518083038185885af1158015610a34573d6000803e3d6000fd5b50505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0168201604052610a7b9190810190611875565b50610b88565b610a8b8484610b8e565b60008473ffffffffffffffffffffffffffffffffffffffff163484604051610ab391906118ec565b60006040518083038185875af1925050503d8060008114610af0576040519150601f19603f3d011682016040523d82523d6000602084013e610af5565b606091505b5050905080610b86576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f50726f787941646d696e3a2063616c6c20746f2070726f78792061667465722060448201527f75706772616465206661696c6564000000000000000000000000000000000000606482015260840161065c565b505b50505050565b610b966110e1565b73ffffffffffffffffffffffffffffffffffffffff821660009081526001602052604081205460ff1690816002811115610bd257610bd26112c1565b03610c2b576040517f3659cfe600000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8381166004830152841690633659cfe6906024016107b2565b6001816002811115610c3f57610c3f6112c1565b03610cbe576040517f9b0b0fda0000000000000000000000000000000000000000000000000000000081527f360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc600482015273ffffffffffffffffffffffffffffffffffffffff8381166024830152841690639b0b0fda906044016107b2565b6002816002811115610cd257610cd26112c1565b03610e165773ffffffffffffffffffffffffffffffffffffffff831660009081526002602052604081208054610d07906115e8565b80601f0160208091040260200160405190810160405280929190818152602001828054610d33906115e8565b8015610d805780601f10610d5557610100808354040283529160200191610d80565b820191906000526020600020905b815481529060010190602001808311610d6357829003601f168201915b50506003546040517f9b2ea4bd00000000000000000000000000000000000000000000000000000000815294955073ffffffffffffffffffffffffffffffffffffffff1693639b2ea4bd9350610dde92508591508790600401611908565b600060405180830381600087803b158015610df857600080fd5b505af1158015610e0c573d6000803e3d6000fd5b5050505050505050565b6108c7611940565b610e266110e1565b6003546040517f9b2ea4bd00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff90911690639b2ea4bd90610e7e9085908590600401611908565b600060405180830381600087803b158015610e9857600080fd5b505af1158015610eac573d6000803e3d6000fd5b505050505050565b610ebc6110e1565b73ffffffffffffffffffffffffffffffffffffffff8116610f5f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015260840161065c565b610f6881611162565b50565b73ffffffffffffffffffffffffffffffffffffffff811660009081526001602052604081205460ff1681816002811115610fa757610fa76112c1565b03610ff7578273ffffffffffffffffffffffffffffffffffffffff1663f851a4406040518163ffffffff1660e01b8152600401602060405180830381865afa1580156104d1573d6000803e3d6000fd5b600181600281111561100b5761100b6112c1565b0361105b578273ffffffffffffffffffffffffffffffffffffffff1663893d20e86040518163ffffffff1660e01b8152600401602060405180830381865afa1580156104d1573d6000803e3d6000fd5b600281600281111561106f5761106f6112c1565b036105fe57600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156104d1573d6000803e3d6000fd5b60005473ffffffffffffffffffffffffffffffffffffffff163314610717576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161065c565b6000805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b73ffffffffffffffffffffffffffffffffffffffff81168114610f6857600080fd5b60006020828403121561120b57600080fd5b81356104f5816111d7565b60006020828403121561122857600080fd5b813580151581146104f557600080fd5b60005b8381101561125357818101518382015260200161123b565b83811115610b885750506000910152565b6000815180845261127c816020860160208601611238565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006104f56020830184611264565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b602081016003831061132b577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b91905290565b6000806040838503121561134457600080fd5b823561134f816111d7565b9150602083013561135f816111d7565b809150509250929050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff811182821017156113e0576113e061136a565b604052919050565b600067ffffffffffffffff8211156114025761140261136a565b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01660200190565b600061144161143c846113e8565b611399565b905082815283838301111561145557600080fd5b828260208301376000602084830101529392505050565b600082601f83011261147d57600080fd5b6104f58383356020850161142e565b6000806040838503121561149f57600080fd5b82356114aa816111d7565b9150602083013567ffffffffffffffff8111156114c657600080fd5b6114d28582860161146c565b9150509250929050565b600080604083850312156114ef57600080fd5b82356114fa816111d7565b915060208301356003811061135f57600080fd5b60008060006060848603121561152357600080fd5b833561152e816111d7565b9250602084013561153e816111d7565b9150604084013567ffffffffffffffff81111561155a57600080fd5b8401601f8101861361156b57600080fd5b61157a8682356020840161142e565b9150509250925092565b6000806040838503121561159757600080fd5b823567ffffffffffffffff8111156115ae57600080fd5b6115ba8582860161146c565b925050602083013561135f816111d7565b6000602082840312156115dd57600080fd5b81516104f5816111d7565b600181811c908216806115fc57607f821691505b602082108103610665577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b6000602080835260008454611649816115e8565b8084870152604060018084166000811461166a57600181146116a2576116d0565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff008516838a01528284151560051b8a010195506116d0565b896000528660002060005b858110156116c85781548b82018601529083019088016116ad565b8a0184019650505b509398975050505050505050565b601f8211156108c757600081815260208120601f850160051c810160208610156117055750805b601f850160051c820191505b81811015610eac57828155600101611711565b815167ffffffffffffffff81111561173e5761173e61136a565b6117528161174c84546115e8565b846116de565b602080601f8311600181146117a5576000841561176f5750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555610eac565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b828110156117f2578886015182559484019460019091019084016117d3565b508582101561182e57878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b73ffffffffffffffffffffffffffffffffffffffff8316815260406020820152600061186d6040830184611264565b949350505050565b60006020828403121561188757600080fd5b815167ffffffffffffffff81111561189e57600080fd5b8201601f810184136118af57600080fd5b80516118bd61143c826113e8565b8181528560208385010111156118d257600080fd5b6118e3826020830160208601611238565b95945050505050565b600082516118fe818460208701611238565b9190910192915050565b60408152600061191b6040830185611264565b905073ffffffffffffffffffffffffffffffffffffffff831660208301529392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052600160045260246000fdfea164736f6c634300080f000a",
}

// ProxyAdminABI is the input ABI used to generate the binding from.
// Deprecated: Use ProxyAdminMetaData.ABI instead.
var ProxyAdminABI = ProxyAdminMetaData.ABI

// ProxyAdminBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ProxyAdminMetaData.Bin instead.
var ProxyAdminBin = ProxyAdminMetaData.Bin

// DeployProxyAdmin deploys a new Ethereum contract, binding an instance of ProxyAdmin to it.
func DeployProxyAdmin(auth *bind.TransactOpts, backend bind.ContractBackend, _owner common.Address) (common.Address, *types.Transaction, *ProxyAdmin, error) {
	parsed, err := ProxyAdminMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ProxyAdminBin), backend, _owner)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ProxyAdmin{ProxyAdminCaller: ProxyAdminCaller{contract: contract}, ProxyAdminTransactor: ProxyAdminTransactor{contract: contract}, ProxyAdminFilterer: ProxyAdminFilterer{contract: contract}}, nil
}

// ProxyAdmin is an auto generated Go binding around an Ethereum contract.
type ProxyAdmin struct {
	ProxyAdminCaller     // Read-only binding to the contract
	ProxyAdminTransactor // Write-only binding to the contract
	ProxyAdminFilterer   // Log filterer for contract events
}

// ProxyAdminCaller is an auto generated read-only Go binding around an Ethereum contract.
type ProxyAdminCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProxyAdminTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ProxyAdminTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProxyAdminFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ProxyAdminFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProxyAdminSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ProxyAdminSession struct {
	Contract     *ProxyAdmin       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ProxyAdminCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ProxyAdminCallerSession struct {
	Contract *ProxyAdminCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// ProxyAdminTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ProxyAdminTransactorSession struct {
	Contract     *ProxyAdminTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ProxyAdminRaw is an auto generated low-level Go binding around an Ethereum contract.
type ProxyAdminRaw struct {
	Contract *ProxyAdmin // Generic contract binding to access the raw methods on
}

// ProxyAdminCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ProxyAdminCallerRaw struct {
	Contract *ProxyAdminCaller // Generic read-only contract binding to access the raw methods on
}

// ProxyAdminTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ProxyAdminTransactorRaw struct {
	Contract *ProxyAdminTransactor // Generic write-only contract binding to access the raw methods on
}

// NewProxyAdmin creates a new instance of ProxyAdmin, bound to a specific deployed contract.
func NewProxyAdmin(address common.Address, backend bind.ContractBackend) (*ProxyAdmin, error) {
	contract, err := bindProxyAdmin(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ProxyAdmin{ProxyAdminCaller: ProxyAdminCaller{contract: contract}, ProxyAdminTransactor: ProxyAdminTransactor{contract: contract}, ProxyAdminFilterer: ProxyAdminFilterer{contract: contract}}, nil
}

// NewProxyAdminCaller creates a new read-only instance of ProxyAdmin, bound to a specific deployed contract.
func NewProxyAdminCaller(address common.Address, caller bind.ContractCaller) (*ProxyAdminCaller, error) {
	contract, err := bindProxyAdmin(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ProxyAdminCaller{contract: contract}, nil
}

// NewProxyAdminTransactor creates a new write-only instance of ProxyAdmin, bound to a specific deployed contract.
func NewProxyAdminTransactor(address common.Address, transactor bind.ContractTransactor) (*ProxyAdminTransactor, error) {
	contract, err := bindProxyAdmin(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ProxyAdminTransactor{contract: contract}, nil
}

// NewProxyAdminFilterer creates a new log filterer instance of ProxyAdmin, bound to a specific deployed contract.
func NewProxyAdminFilterer(address common.Address, filterer bind.ContractFilterer) (*ProxyAdminFilterer, error) {
	contract, err := bindProxyAdmin(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ProxyAdminFilterer{contract: contract}, nil
}

// bindProxyAdmin binds a generic wrapper to an already deployed contract.
func bindProxyAdmin(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ProxyAdminABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProxyAdmin *ProxyAdminRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProxyAdmin.Contract.ProxyAdminCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProxyAdmin *ProxyAdminRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.ProxyAdminTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProxyAdmin *ProxyAdminRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.ProxyAdminTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProxyAdmin *ProxyAdminCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProxyAdmin.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProxyAdmin *ProxyAdminTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProxyAdmin *ProxyAdminTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.contract.Transact(opts, method, params...)
}

// AddressManager is a free data retrieval call binding the contract method 0x3ab76e9f.
//
// Solidity: function addressManager() view returns(address)
func (_ProxyAdmin *ProxyAdminCaller) AddressManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ProxyAdmin.contract.Call(opts, &out, "addressManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AddressManager is a free data retrieval call binding the contract method 0x3ab76e9f.
//
// Solidity: function addressManager() view returns(address)
func (_ProxyAdmin *ProxyAdminSession) AddressManager() (common.Address, error) {
	return _ProxyAdmin.Contract.AddressManager(&_ProxyAdmin.CallOpts)
}

// AddressManager is a free data retrieval call binding the contract method 0x3ab76e9f.
//
// Solidity: function addressManager() view returns(address)
func (_ProxyAdmin *ProxyAdminCallerSession) AddressManager() (common.Address, error) {
	return _ProxyAdmin.Contract.AddressManager(&_ProxyAdmin.CallOpts)
}

// GetProxyAdmin is a free data retrieval call binding the contract method 0xf3b7dead.
//
// Solidity: function getProxyAdmin(address _proxy) view returns(address)
func (_ProxyAdmin *ProxyAdminCaller) GetProxyAdmin(opts *bind.CallOpts, _proxy common.Address) (common.Address, error) {
	var out []interface{}
	err := _ProxyAdmin.contract.Call(opts, &out, "getProxyAdmin", _proxy)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetProxyAdmin is a free data retrieval call binding the contract method 0xf3b7dead.
//
// Solidity: function getProxyAdmin(address _proxy) view returns(address)
func (_ProxyAdmin *ProxyAdminSession) GetProxyAdmin(_proxy common.Address) (common.Address, error) {
	return _ProxyAdmin.Contract.GetProxyAdmin(&_ProxyAdmin.CallOpts, _proxy)
}

// GetProxyAdmin is a free data retrieval call binding the contract method 0xf3b7dead.
//
// Solidity: function getProxyAdmin(address _proxy) view returns(address)
func (_ProxyAdmin *ProxyAdminCallerSession) GetProxyAdmin(_proxy common.Address) (common.Address, error) {
	return _ProxyAdmin.Contract.GetProxyAdmin(&_ProxyAdmin.CallOpts, _proxy)
}

// GetProxyImplementation is a free data retrieval call binding the contract method 0x204e1c7a.
//
// Solidity: function getProxyImplementation(address _proxy) view returns(address)
func (_ProxyAdmin *ProxyAdminCaller) GetProxyImplementation(opts *bind.CallOpts, _proxy common.Address) (common.Address, error) {
	var out []interface{}
	err := _ProxyAdmin.contract.Call(opts, &out, "getProxyImplementation", _proxy)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetProxyImplementation is a free data retrieval call binding the contract method 0x204e1c7a.
//
// Solidity: function getProxyImplementation(address _proxy) view returns(address)
func (_ProxyAdmin *ProxyAdminSession) GetProxyImplementation(_proxy common.Address) (common.Address, error) {
	return _ProxyAdmin.Contract.GetProxyImplementation(&_ProxyAdmin.CallOpts, _proxy)
}

// GetProxyImplementation is a free data retrieval call binding the contract method 0x204e1c7a.
//
// Solidity: function getProxyImplementation(address _proxy) view returns(address)
func (_ProxyAdmin *ProxyAdminCallerSession) GetProxyImplementation(_proxy common.Address) (common.Address, error) {
	return _ProxyAdmin.Contract.GetProxyImplementation(&_ProxyAdmin.CallOpts, _proxy)
}

// ImplementationName is a free data retrieval call binding the contract method 0x238181ae.
//
// Solidity: function implementationName(address ) view returns(string)
func (_ProxyAdmin *ProxyAdminCaller) ImplementationName(opts *bind.CallOpts, arg0 common.Address) (string, error) {
	var out []interface{}
	err := _ProxyAdmin.contract.Call(opts, &out, "implementationName", arg0)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// ImplementationName is a free data retrieval call binding the contract method 0x238181ae.
//
// Solidity: function implementationName(address ) view returns(string)
func (_ProxyAdmin *ProxyAdminSession) ImplementationName(arg0 common.Address) (string, error) {
	return _ProxyAdmin.Contract.ImplementationName(&_ProxyAdmin.CallOpts, arg0)
}

// ImplementationName is a free data retrieval call binding the contract method 0x238181ae.
//
// Solidity: function implementationName(address ) view returns(string)
func (_ProxyAdmin *ProxyAdminCallerSession) ImplementationName(arg0 common.Address) (string, error) {
	return _ProxyAdmin.Contract.ImplementationName(&_ProxyAdmin.CallOpts, arg0)
}

// IsUpgrading is a free data retrieval call binding the contract method 0xb7947262.
//
// Solidity: function isUpgrading() view returns(bool)
func (_ProxyAdmin *ProxyAdminCaller) IsUpgrading(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _ProxyAdmin.contract.Call(opts, &out, "isUpgrading")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsUpgrading is a free data retrieval call binding the contract method 0xb7947262.
//
// Solidity: function isUpgrading() view returns(bool)
func (_ProxyAdmin *ProxyAdminSession) IsUpgrading() (bool, error) {
	return _ProxyAdmin.Contract.IsUpgrading(&_ProxyAdmin.CallOpts)
}

// IsUpgrading is a free data retrieval call binding the contract method 0xb7947262.
//
// Solidity: function isUpgrading() view returns(bool)
func (_ProxyAdmin *ProxyAdminCallerSession) IsUpgrading() (bool, error) {
	return _ProxyAdmin.Contract.IsUpgrading(&_ProxyAdmin.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ProxyAdmin *ProxyAdminCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ProxyAdmin.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ProxyAdmin *ProxyAdminSession) Owner() (common.Address, error) {
	return _ProxyAdmin.Contract.Owner(&_ProxyAdmin.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ProxyAdmin *ProxyAdminCallerSession) Owner() (common.Address, error) {
	return _ProxyAdmin.Contract.Owner(&_ProxyAdmin.CallOpts)
}

// ProxyType is a free data retrieval call binding the contract method 0x6bd9f516.
//
// Solidity: function proxyType(address ) view returns(uint8)
func (_ProxyAdmin *ProxyAdminCaller) ProxyType(opts *bind.CallOpts, arg0 common.Address) (uint8, error) {
	var out []interface{}
	err := _ProxyAdmin.contract.Call(opts, &out, "proxyType", arg0)

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// ProxyType is a free data retrieval call binding the contract method 0x6bd9f516.
//
// Solidity: function proxyType(address ) view returns(uint8)
func (_ProxyAdmin *ProxyAdminSession) ProxyType(arg0 common.Address) (uint8, error) {
	return _ProxyAdmin.Contract.ProxyType(&_ProxyAdmin.CallOpts, arg0)
}

// ProxyType is a free data retrieval call binding the contract method 0x6bd9f516.
//
// Solidity: function proxyType(address ) view returns(uint8)
func (_ProxyAdmin *ProxyAdminCallerSession) ProxyType(arg0 common.Address) (uint8, error) {
	return _ProxyAdmin.Contract.ProxyType(&_ProxyAdmin.CallOpts, arg0)
}

// ChangeProxyAdmin is a paid mutator transaction binding the contract method 0x7eff275e.
//
// Solidity: function changeProxyAdmin(address _proxy, address _newAdmin) returns()
func (_ProxyAdmin *ProxyAdminTransactor) ChangeProxyAdmin(opts *bind.TransactOpts, _proxy common.Address, _newAdmin common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "changeProxyAdmin", _proxy, _newAdmin)
}

// ChangeProxyAdmin is a paid mutator transaction binding the contract method 0x7eff275e.
//
// Solidity: function changeProxyAdmin(address _proxy, address _newAdmin) returns()
func (_ProxyAdmin *ProxyAdminSession) ChangeProxyAdmin(_proxy common.Address, _newAdmin common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.ChangeProxyAdmin(&_ProxyAdmin.TransactOpts, _proxy, _newAdmin)
}

// ChangeProxyAdmin is a paid mutator transaction binding the contract method 0x7eff275e.
//
// Solidity: function changeProxyAdmin(address _proxy, address _newAdmin) returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) ChangeProxyAdmin(_proxy common.Address, _newAdmin common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.ChangeProxyAdmin(&_ProxyAdmin.TransactOpts, _proxy, _newAdmin)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ProxyAdmin *ProxyAdminTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ProxyAdmin *ProxyAdminSession) RenounceOwnership() (*types.Transaction, error) {
	return _ProxyAdmin.Contract.RenounceOwnership(&_ProxyAdmin.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _ProxyAdmin.Contract.RenounceOwnership(&_ProxyAdmin.TransactOpts)
}

// SetAddress is a paid mutator transaction binding the contract method 0x9b2ea4bd.
//
// Solidity: function setAddress(string _name, address _address) returns()
func (_ProxyAdmin *ProxyAdminTransactor) SetAddress(opts *bind.TransactOpts, _name string, _address common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "setAddress", _name, _address)
}

// SetAddress is a paid mutator transaction binding the contract method 0x9b2ea4bd.
//
// Solidity: function setAddress(string _name, address _address) returns()
func (_ProxyAdmin *ProxyAdminSession) SetAddress(_name string, _address common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetAddress(&_ProxyAdmin.TransactOpts, _name, _address)
}

// SetAddress is a paid mutator transaction binding the contract method 0x9b2ea4bd.
//
// Solidity: function setAddress(string _name, address _address) returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) SetAddress(_name string, _address common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetAddress(&_ProxyAdmin.TransactOpts, _name, _address)
}

// SetAddressManager is a paid mutator transaction binding the contract method 0x0652b57a.
//
// Solidity: function setAddressManager(address _address) returns()
func (_ProxyAdmin *ProxyAdminTransactor) SetAddressManager(opts *bind.TransactOpts, _address common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "setAddressManager", _address)
}

// SetAddressManager is a paid mutator transaction binding the contract method 0x0652b57a.
//
// Solidity: function setAddressManager(address _address) returns()
func (_ProxyAdmin *ProxyAdminSession) SetAddressManager(_address common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetAddressManager(&_ProxyAdmin.TransactOpts, _address)
}

// SetAddressManager is a paid mutator transaction binding the contract method 0x0652b57a.
//
// Solidity: function setAddressManager(address _address) returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) SetAddressManager(_address common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetAddressManager(&_ProxyAdmin.TransactOpts, _address)
}

// SetImplementationName is a paid mutator transaction binding the contract method 0x860f7cda.
//
// Solidity: function setImplementationName(address _address, string _name) returns()
func (_ProxyAdmin *ProxyAdminTransactor) SetImplementationName(opts *bind.TransactOpts, _address common.Address, _name string) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "setImplementationName", _address, _name)
}

// SetImplementationName is a paid mutator transaction binding the contract method 0x860f7cda.
//
// Solidity: function setImplementationName(address _address, string _name) returns()
func (_ProxyAdmin *ProxyAdminSession) SetImplementationName(_address common.Address, _name string) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetImplementationName(&_ProxyAdmin.TransactOpts, _address, _name)
}

// SetImplementationName is a paid mutator transaction binding the contract method 0x860f7cda.
//
// Solidity: function setImplementationName(address _address, string _name) returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) SetImplementationName(_address common.Address, _name string) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetImplementationName(&_ProxyAdmin.TransactOpts, _address, _name)
}

// SetProxyType is a paid mutator transaction binding the contract method 0x8d52d4a0.
//
// Solidity: function setProxyType(address _address, uint8 _type) returns()
func (_ProxyAdmin *ProxyAdminTransactor) SetProxyType(opts *bind.TransactOpts, _address common.Address, _type uint8) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "setProxyType", _address, _type)
}

// SetProxyType is a paid mutator transaction binding the contract method 0x8d52d4a0.
//
// Solidity: function setProxyType(address _address, uint8 _type) returns()
func (_ProxyAdmin *ProxyAdminSession) SetProxyType(_address common.Address, _type uint8) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetProxyType(&_ProxyAdmin.TransactOpts, _address, _type)
}

// SetProxyType is a paid mutator transaction binding the contract method 0x8d52d4a0.
//
// Solidity: function setProxyType(address _address, uint8 _type) returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) SetProxyType(_address common.Address, _type uint8) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetProxyType(&_ProxyAdmin.TransactOpts, _address, _type)
}

// SetUpgrading is a paid mutator transaction binding the contract method 0x07c8f7b0.
//
// Solidity: function setUpgrading(bool _upgrading) returns()
func (_ProxyAdmin *ProxyAdminTransactor) SetUpgrading(opts *bind.TransactOpts, _upgrading bool) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "setUpgrading", _upgrading)
}

// SetUpgrading is a paid mutator transaction binding the contract method 0x07c8f7b0.
//
// Solidity: function setUpgrading(bool _upgrading) returns()
func (_ProxyAdmin *ProxyAdminSession) SetUpgrading(_upgrading bool) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetUpgrading(&_ProxyAdmin.TransactOpts, _upgrading)
}

// SetUpgrading is a paid mutator transaction binding the contract method 0x07c8f7b0.
//
// Solidity: function setUpgrading(bool _upgrading) returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) SetUpgrading(_upgrading bool) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetUpgrading(&_ProxyAdmin.TransactOpts, _upgrading)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ProxyAdmin *ProxyAdminTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ProxyAdmin *ProxyAdminSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.TransferOwnership(&_ProxyAdmin.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.TransferOwnership(&_ProxyAdmin.TransactOpts, newOwner)
}

// Upgrade is a paid mutator transaction binding the contract method 0x99a88ec4.
//
// Solidity: function upgrade(address _proxy, address _implementation) returns()
func (_ProxyAdmin *ProxyAdminTransactor) Upgrade(opts *bind.TransactOpts, _proxy common.Address, _implementation common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "upgrade", _proxy, _implementation)
}

// Upgrade is a paid mutator transaction binding the contract method 0x99a88ec4.
//
// Solidity: function upgrade(address _proxy, address _implementation) returns()
func (_ProxyAdmin *ProxyAdminSession) Upgrade(_proxy common.Address, _implementation common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.Upgrade(&_ProxyAdmin.TransactOpts, _proxy, _implementation)
}

// Upgrade is a paid mutator transaction binding the contract method 0x99a88ec4.
//
// Solidity: function upgrade(address _proxy, address _implementation) returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) Upgrade(_proxy common.Address, _implementation common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.Upgrade(&_ProxyAdmin.TransactOpts, _proxy, _implementation)
}

// UpgradeAndCall is a paid mutator transaction binding the contract method 0x9623609d.
//
// Solidity: function upgradeAndCall(address _proxy, address _implementation, bytes _data) payable returns()
func (_ProxyAdmin *ProxyAdminTransactor) UpgradeAndCall(opts *bind.TransactOpts, _proxy common.Address, _implementation common.Address, _data []byte) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "upgradeAndCall", _proxy, _implementation, _data)
}

// UpgradeAndCall is a paid mutator transaction binding the contract method 0x9623609d.
//
// Solidity: function upgradeAndCall(address _proxy, address _implementation, bytes _data) payable returns()
func (_ProxyAdmin *ProxyAdminSession) UpgradeAndCall(_proxy common.Address, _implementation common.Address, _data []byte) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.UpgradeAndCall(&_ProxyAdmin.TransactOpts, _proxy, _implementation, _data)
}

// UpgradeAndCall is a paid mutator transaction binding the contract method 0x9623609d.
//
// Solidity: function upgradeAndCall(address _proxy, address _implementation, bytes _data) payable returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) UpgradeAndCall(_proxy common.Address, _implementation common.Address, _data []byte) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.UpgradeAndCall(&_ProxyAdmin.TransactOpts, _proxy, _implementation, _data)
}

// ProxyAdminOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the ProxyAdmin contract.
type ProxyAdminOwnershipTransferredIterator struct {
	Event *ProxyAdminOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *ProxyAdminOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ProxyAdminOwnershipTransferred)
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
		it.Event = new(ProxyAdminOwnershipTransferred)
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
func (it *ProxyAdminOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ProxyAdminOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ProxyAdminOwnershipTransferred represents a OwnershipTransferred event raised by the ProxyAdmin contract.
type ProxyAdminOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ProxyAdmin *ProxyAdminFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ProxyAdminOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ProxyAdmin.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ProxyAdminOwnershipTransferredIterator{contract: _ProxyAdmin.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ProxyAdmin *ProxyAdminFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ProxyAdminOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ProxyAdmin.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ProxyAdminOwnershipTransferred)
				if err := _ProxyAdmin.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_ProxyAdmin *ProxyAdminFilterer) ParseOwnershipTransferred(log types.Log) (*ProxyAdminOwnershipTransferred, error) {
	event := new(ProxyAdminOwnershipTransferred)
	if err := _ProxyAdmin.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
