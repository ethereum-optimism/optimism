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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnerUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"addressManager\",\"outputs\":[{\"internalType\":\"contractAddressManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_newAdmin\",\"type\":\"address\"}],\"name\":\"changeProxyAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_proxy\",\"type\":\"address\"}],\"name\":\"getProxyAdmin\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_proxy\",\"type\":\"address\"}],\"name\":\"getProxyImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"implementationName\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isUpgrading\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"proxyType\",\"outputs\":[{\"internalType\":\"enumProxyAdmin.ProxyType\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"setAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractAddressManager\",\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"setAddressManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_address\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"setImplementationName\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"setOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_address\",\"type\":\"address\"},{\"internalType\":\"enumProxyAdmin.ProxyType\",\"name\":\"_type\",\"type\":\"uint8\"}],\"name\":\"setProxyType\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_upgrading\",\"type\":\"bool\"}],\"name\":\"setUpgrading\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_implementation\",\"type\":\"address\"}],\"name\":\"upgrade\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_implementation\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"upgradeAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x60806040526003805460ff60a01b1916905534801561001d57600080fd5b50604051611d13380380611d1383398101604081905261003c9161008b565b600080546001600160a01b0319166001600160a01b03831690811782556040518392907f8292fce18fa69edf4db7b94ea2e58241df0ae57f97e0a6c9b29067028bf92d76908290a350506100bb565b60006020828403121561009d57600080fd5b81516001600160a01b03811681146100b457600080fd5b9392505050565b611c49806100ca6000396000f3fe6080604052600436106100f35760003560e01c8063860f7cda1161008a57806399a88ec41161005957806399a88ec4146102db5780639b2ea4bd146102fb578063b79472621461031b578063f3b7dead1461035657600080fd5b8063860f7cda1461025b5780638d52d4a01461027b5780638da5cb5b1461029b5780639623609d146102c857600080fd5b8063238181ae116100c6578063238181ae146101a45780633ab76e9f146101d15780636bd9f516146101fe5780637eff275e1461023b57600080fd5b80630652b57a146100f857806307c8f7b01461011a57806313af40351461013a578063204e1c7a1461015a575b600080fd5b34801561010457600080fd5b506101186101133660046114c6565b610376565b005b34801561012657600080fd5b506101186101353660046114e3565b610443565b34801561014657600080fd5b506101186101553660046114c6565b61050e565b34801561016657600080fd5b5061017a6101753660046114c6565b6105ff565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b3480156101b057600080fd5b506101c46101bf3660046114c6565b610820565b60405161019b919061157b565b3480156101dd57600080fd5b5060035461017a9073ffffffffffffffffffffffffffffffffffffffff1681565b34801561020a57600080fd5b5061022e6102193660046114c6565b60016020526000908152604090205460ff1681565b60405161019b91906115bd565b34801561024757600080fd5b506101186102563660046115fe565b6108ba565b34801561026757600080fd5b50610118610276366004611759565b610ae6565b34801561028757600080fd5b506101186102963660046117a9565b610b96565b3480156102a757600080fd5b5060005461017a9073ffffffffffffffffffffffffffffffffffffffff1681565b6101186102d63660046117db565b610c83565b3480156102e757600080fd5b506101186102f63660046115fe565b610f13565b34801561030757600080fd5b50610118610316366004611851565b61121c565b34801561032757600080fd5b5060035474010000000000000000000000000000000000000000900460ff16604051901515815260200161019b565b34801561036257600080fd5b5061017a6103713660046114c6565b61132b565b60005473ffffffffffffffffffffffffffffffffffffffff1633146103fc576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600c60248201527f554e415554484f52495a4544000000000000000000000000000000000000000060448201526064015b60405180910390fd5b600380547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b60005473ffffffffffffffffffffffffffffffffffffffff1633146104c4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600c60248201527f554e415554484f52495a4544000000000000000000000000000000000000000060448201526064016103f3565b6003805491151574010000000000000000000000000000000000000000027fffffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffff909216919091179055565b60005473ffffffffffffffffffffffffffffffffffffffff16331461058f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600c60248201527f554e415554484f52495a4544000000000000000000000000000000000000000060448201526064016103f3565b600080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff83169081178255604051909133917f8292fce18fa69edf4db7b94ea2e58241df0ae57f97e0a6c9b29067028bf92d769190a350565b73ffffffffffffffffffffffffffffffffffffffff811660009081526001602052604081205460ff168181600281111561063b5761063b61158e565b036106b6578273ffffffffffffffffffffffffffffffffffffffff16635c60da1b6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561068b573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906106af9190611898565b9392505050565b60018160028111156106ca576106ca61158e565b0361071a578273ffffffffffffffffffffffffffffffffffffffff1663aaf10f426040518163ffffffff1660e01b8152600401602060405180830381865afa15801561068b573d6000803e3d6000fd5b600281600281111561072e5761072e61158e565b036107b85760035473ffffffffffffffffffffffffffffffffffffffff8481166000908152600260205260409081902090517fbf40fac1000000000000000000000000000000000000000000000000000000008152919092169163bf40fac19161079b9190600401611902565b602060405180830381865afa15801561068b573d6000803e3d6000fd5b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601e60248201527f50726f787941646d696e3a20756e6b6e6f776e2070726f78792074797065000060448201526064016103f3565b50919050565b60026020526000908152604090208054610839906118b5565b80601f0160208091040260200160405190810160405280929190818152602001828054610865906118b5565b80156108b25780601f10610887576101008083540402835291602001916108b2565b820191906000526020600020905b81548152906001019060200180831161089557829003601f168201915b505050505081565b60005473ffffffffffffffffffffffffffffffffffffffff16331461093b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600c60248201527f554e415554484f52495a4544000000000000000000000000000000000000000060448201526064016103f3565b73ffffffffffffffffffffffffffffffffffffffff821660009081526001602052604081205460ff16908160028111156109775761097761158e565b03610a03576040517f8f28397000000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8381166004830152841690638f283970906024015b600060405180830381600087803b1580156109e657600080fd5b505af11580156109fa573d6000803e3d6000fd5b50505050505050565b6001816002811115610a1757610a1761158e565b03610a70576040517f13af403500000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff83811660048301528416906313af4035906024016109cc565b6002816002811115610a8457610a8461158e565b036107b8576003546040517ff2fde38b00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff84811660048301529091169063f2fde38b906024016109cc565b505050565b60005473ffffffffffffffffffffffffffffffffffffffff163314610b67576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600c60248201527f554e415554484f52495a4544000000000000000000000000000000000000000060448201526064016103f3565b73ffffffffffffffffffffffffffffffffffffffff82166000908152600260205260409020610ae182826119f1565b60005473ffffffffffffffffffffffffffffffffffffffff163314610c17576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600c60248201527f554e415554484f52495a4544000000000000000000000000000000000000000060448201526064016103f3565b73ffffffffffffffffffffffffffffffffffffffff82166000908152600160208190526040909120805483927fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0090911690836002811115610c7a57610c7a61158e565b02179055505050565b60005473ffffffffffffffffffffffffffffffffffffffff163314610d04576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600c60248201527f554e415554484f52495a4544000000000000000000000000000000000000000060448201526064016103f3565b73ffffffffffffffffffffffffffffffffffffffff831660009081526001602052604081205460ff1690816002811115610d4057610d4061158e565b03610e06576040517f4f1ef28600000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff851690634f1ef286903490610d9b9087908790600401611b0b565b60006040518083038185885af1158015610db9573d6000803e3d6000fd5b50505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0168201604052610e009190810190611b42565b50610f0d565b610e108484610f13565b60008473ffffffffffffffffffffffffffffffffffffffff163484604051610e389190611bb9565b60006040518083038185875af1925050503d8060008114610e75576040519150601f19603f3d011682016040523d82523d6000602084013e610e7a565b606091505b5050905080610f0b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f50726f787941646d696e3a2063616c6c20746f2070726f78792061667465722060448201527f75706772616465206661696c656400000000000000000000000000000000000060648201526084016103f3565b505b50505050565b60005473ffffffffffffffffffffffffffffffffffffffff163314610f94576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600c60248201527f554e415554484f52495a4544000000000000000000000000000000000000000060448201526064016103f3565b73ffffffffffffffffffffffffffffffffffffffff821660009081526001602052604081205460ff1690816002811115610fd057610fd061158e565b03611029576040517f3659cfe600000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8381166004830152841690633659cfe6906024016109cc565b600181600281111561103d5761103d61158e565b036110bc576040517f9b0b0fda0000000000000000000000000000000000000000000000000000000081527f360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc600482015273ffffffffffffffffffffffffffffffffffffffff8381166024830152841690639b0b0fda906044016109cc565b60028160028111156110d0576110d061158e565b036112145773ffffffffffffffffffffffffffffffffffffffff831660009081526002602052604081208054611105906118b5565b80601f0160208091040260200160405190810160405280929190818152602001828054611131906118b5565b801561117e5780601f106111535761010080835404028352916020019161117e565b820191906000526020600020905b81548152906001019060200180831161116157829003601f168201915b50506003546040517f9b2ea4bd00000000000000000000000000000000000000000000000000000000815294955073ffffffffffffffffffffffffffffffffffffffff1693639b2ea4bd93506111dc92508591508790600401611bd5565b600060405180830381600087803b1580156111f657600080fd5b505af115801561120a573d6000803e3d6000fd5b5050505050505050565b610ae1611c0d565b60005473ffffffffffffffffffffffffffffffffffffffff16331461129d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600c60248201527f554e415554484f52495a4544000000000000000000000000000000000000000060448201526064016103f3565b6003546040517f9b2ea4bd00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff90911690639b2ea4bd906112f59085908590600401611bd5565b600060405180830381600087803b15801561130f57600080fd5b505af1158015611323573d6000803e3d6000fd5b505050505050565b73ffffffffffffffffffffffffffffffffffffffff811660009081526001602052604081205460ff16818160028111156113675761136761158e565b036113b7578273ffffffffffffffffffffffffffffffffffffffff1663f851a4406040518163ffffffff1660e01b8152600401602060405180830381865afa15801561068b573d6000803e3d6000fd5b60018160028111156113cb576113cb61158e565b0361141b578273ffffffffffffffffffffffffffffffffffffffff1663893d20e86040518163ffffffff1660e01b8152600401602060405180830381865afa15801561068b573d6000803e3d6000fd5b600281600281111561142f5761142f61158e565b036107b857600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561068b573d6000803e3d6000fd5b73ffffffffffffffffffffffffffffffffffffffff811681146114c357600080fd5b50565b6000602082840312156114d857600080fd5b81356106af816114a1565b6000602082840312156114f557600080fd5b813580151581146106af57600080fd5b60005b83811015611520578181015183820152602001611508565b83811115610f0d5750506000910152565b60008151808452611549816020860160208601611505565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006106af6020830184611531565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b60208101600383106115f8577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b91905290565b6000806040838503121561161157600080fd5b823561161c816114a1565b9150602083013561162c816114a1565b809150509250929050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff811182821017156116ad576116ad611637565b604052919050565b600067ffffffffffffffff8211156116cf576116cf611637565b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01660200190565b600061170e611709846116b5565b611666565b905082815283838301111561172257600080fd5b828260208301376000602084830101529392505050565b600082601f83011261174a57600080fd5b6106af838335602085016116fb565b6000806040838503121561176c57600080fd5b8235611777816114a1565b9150602083013567ffffffffffffffff81111561179357600080fd5b61179f85828601611739565b9150509250929050565b600080604083850312156117bc57600080fd5b82356117c7816114a1565b915060208301356003811061162c57600080fd5b6000806000606084860312156117f057600080fd5b83356117fb816114a1565b9250602084013561180b816114a1565b9150604084013567ffffffffffffffff81111561182757600080fd5b8401601f8101861361183857600080fd5b611847868235602084016116fb565b9150509250925092565b6000806040838503121561186457600080fd5b823567ffffffffffffffff81111561187b57600080fd5b61188785828601611739565b925050602083013561162c816114a1565b6000602082840312156118aa57600080fd5b81516106af816114a1565b600181811c908216806118c957607f821691505b60208210810361081a577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b6000602080835260008454611916816118b5565b80848701526040600180841660008114611937576001811461196f5761199d565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff008516838a01528284151560051b8a0101955061199d565b896000528660002060005b858110156119955781548b820186015290830190880161197a565b8a0184019650505b509398975050505050505050565b601f821115610ae157600081815260208120601f850160051c810160208610156119d25750805b601f850160051c820191505b81811015611323578281556001016119de565b815167ffffffffffffffff811115611a0b57611a0b611637565b611a1f81611a1984546118b5565b846119ab565b602080601f831160018114611a725760008415611a3c5750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555611323565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b82811015611abf57888601518255948401946001909101908401611aa0565b5085821015611afb57878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b73ffffffffffffffffffffffffffffffffffffffff83168152604060208201526000611b3a6040830184611531565b949350505050565b600060208284031215611b5457600080fd5b815167ffffffffffffffff811115611b6b57600080fd5b8201601f81018413611b7c57600080fd5b8051611b8a611709826116b5565b818152856020838501011115611b9f57600080fd5b611bb0826020830160208601611505565b95945050505050565b60008251611bcb818460208701611505565b9190910192915050565b604081526000611be86040830185611531565b905073ffffffffffffffffffffffffffffffffffffffff831660208301529392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052600160045260246000fdfea164736f6c634300080f000a",
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

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address newOwner) returns()
func (_ProxyAdmin *ProxyAdminTransactor) SetOwner(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.contract.Transact(opts, "setOwner", newOwner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address newOwner) returns()
func (_ProxyAdmin *ProxyAdminSession) SetOwner(newOwner common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetOwner(&_ProxyAdmin.TransactOpts, newOwner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address newOwner) returns()
func (_ProxyAdmin *ProxyAdminTransactorSession) SetOwner(newOwner common.Address) (*types.Transaction, error) {
	return _ProxyAdmin.Contract.SetOwner(&_ProxyAdmin.TransactOpts, newOwner)
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

// ProxyAdminOwnerUpdatedIterator is returned from FilterOwnerUpdated and is used to iterate over the raw logs and unpacked data for OwnerUpdated events raised by the ProxyAdmin contract.
type ProxyAdminOwnerUpdatedIterator struct {
	Event *ProxyAdminOwnerUpdated // Event containing the contract specifics and raw log

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
func (it *ProxyAdminOwnerUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ProxyAdminOwnerUpdated)
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
		it.Event = new(ProxyAdminOwnerUpdated)
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
func (it *ProxyAdminOwnerUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ProxyAdminOwnerUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ProxyAdminOwnerUpdated represents a OwnerUpdated event raised by the ProxyAdmin contract.
type ProxyAdminOwnerUpdated struct {
	User     common.Address
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOwnerUpdated is a free log retrieval operation binding the contract event 0x8292fce18fa69edf4db7b94ea2e58241df0ae57f97e0a6c9b29067028bf92d76.
//
// Solidity: event OwnerUpdated(address indexed user, address indexed newOwner)
func (_ProxyAdmin *ProxyAdminFilterer) FilterOwnerUpdated(opts *bind.FilterOpts, user []common.Address, newOwner []common.Address) (*ProxyAdminOwnerUpdatedIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ProxyAdmin.contract.FilterLogs(opts, "OwnerUpdated", userRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ProxyAdminOwnerUpdatedIterator{contract: _ProxyAdmin.contract, event: "OwnerUpdated", logs: logs, sub: sub}, nil
}

// WatchOwnerUpdated is a free log subscription operation binding the contract event 0x8292fce18fa69edf4db7b94ea2e58241df0ae57f97e0a6c9b29067028bf92d76.
//
// Solidity: event OwnerUpdated(address indexed user, address indexed newOwner)
func (_ProxyAdmin *ProxyAdminFilterer) WatchOwnerUpdated(opts *bind.WatchOpts, sink chan<- *ProxyAdminOwnerUpdated, user []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ProxyAdmin.contract.WatchLogs(opts, "OwnerUpdated", userRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ProxyAdminOwnerUpdated)
				if err := _ProxyAdmin.contract.UnpackLog(event, "OwnerUpdated", log); err != nil {
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

// ParseOwnerUpdated is a log parse operation binding the contract event 0x8292fce18fa69edf4db7b94ea2e58241df0ae57f97e0a6c9b29067028bf92d76.
//
// Solidity: event OwnerUpdated(address indexed user, address indexed newOwner)
func (_ProxyAdmin *ProxyAdminFilterer) ParseOwnerUpdated(log types.Log) (*ProxyAdminOwnerUpdated, error) {
	event := new(ProxyAdminOwnerUpdated)
	if err := _ProxyAdmin.contract.UnpackLog(event, "OwnerUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
