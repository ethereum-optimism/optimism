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

// ResourceMeteringResourceConfig is an auto generated low-level Go binding around an user-defined struct.
type ResourceMeteringResourceConfig struct {
	MaxResourceLimit            uint32
	ElasticityMultiplier        uint8
	BaseFeeMaxChangeDenominator uint8
	MinimumBaseFee              uint32
	SystemTxMaxGas              uint32
	MaximumBaseFee              *big.Int
}

// SystemConfigAddresses is an auto generated low-level Go binding around an user-defined struct.
type SystemConfigAddresses struct {
	L1CrossDomainMessenger       common.Address
	L1ERC721Bridge               common.Address
	L1StandardBridge             common.Address
	L2OutputOracle               common.Address
	OptimismPortal               common.Address
	OptimismMintableERC20Factory common.Address
}

// SystemConfigMetaData contains all meta data concerning the SystemConfig contract.
var SystemConfigMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"BATCH_INBOX_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"L1_CROSS_DOMAIN_MESSENGER_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"L1_ERC_721_BRIDGE_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"L1_STANDARD_BRIDGE_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"L2_OUTPUT_ORACLE_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"OPTIMISM_MINTABLE_ERC20_FACTORY_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"OPTIMISM_PORTAL_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"START_BLOCK_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"UNSAFE_BLOCK_SIGNER_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"VERSION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"batchInbox\",\"inputs\":[],\"outputs\":[{\"name\":\"addr_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"batcherHash\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gasLimit\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_overhead\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_scalar\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_batcherHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_gasLimit\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"_unsafeBlockSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_config\",\"type\":\"tuple\",\"internalType\":\"structResourceMetering.ResourceConfig\",\"components\":[{\"name\":\"maxResourceLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"elasticityMultiplier\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"baseFeeMaxChangeDenominator\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"minimumBaseFee\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"systemTxMaxGas\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"maximumBaseFee\",\"type\":\"uint128\",\"internalType\":\"uint128\"}]},{\"name\":\"_batchInbox\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_addresses\",\"type\":\"tuple\",\"internalType\":\"structSystemConfig.Addresses\",\"components\":[{\"name\":\"l1CrossDomainMessenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ERC721Bridge\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1StandardBridge\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l2OutputOracle\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismPortal\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismMintableERC20Factory\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"l1CrossDomainMessenger\",\"inputs\":[],\"outputs\":[{\"name\":\"addr_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l1ERC721Bridge\",\"inputs\":[],\"outputs\":[{\"name\":\"addr_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l1StandardBridge\",\"inputs\":[],\"outputs\":[{\"name\":\"addr_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l2OutputOracle\",\"inputs\":[],\"outputs\":[{\"name\":\"addr_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"minimumGasLimit\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"optimismMintableERC20Factory\",\"inputs\":[],\"outputs\":[{\"name\":\"addr_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"optimismPortal\",\"inputs\":[],\"outputs\":[{\"name\":\"addr_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"overhead\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resourceConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structResourceMetering.ResourceConfig\",\"components\":[{\"name\":\"maxResourceLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"elasticityMultiplier\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"baseFeeMaxChangeDenominator\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"minimumBaseFee\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"systemTxMaxGas\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"maximumBaseFee\",\"type\":\"uint128\",\"internalType\":\"uint128\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"scalar\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setBatcherHash\",\"inputs\":[{\"name\":\"_batcherHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setGasConfig\",\"inputs\":[{\"name\":\"_overhead\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_scalar\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setGasLimit\",\"inputs\":[{\"name\":\"_gasLimit\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setResourceConfig\",\"inputs\":[{\"name\":\"_config\",\"type\":\"tuple\",\"internalType\":\"structResourceMetering.ResourceConfig\",\"components\":[{\"name\":\"maxResourceLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"elasticityMultiplier\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"baseFeeMaxChangeDenominator\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"minimumBaseFee\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"systemTxMaxGas\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"maximumBaseFee\",\"type\":\"uint128\",\"internalType\":\"uint128\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setUnsafeBlockSigner\",\"inputs\":[{\"name\":\"_unsafeBlockSigner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"startBlock\",\"inputs\":[],\"outputs\":[{\"name\":\"startBlock_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unsafeBlockSigner\",\"inputs\":[],\"outputs\":[{\"name\":\"addr_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"ConfigUpdate\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"updateType\",\"type\":\"uint8\",\"indexed\":true,\"internalType\":\"enumSystemConfig.UpdateType\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false}]",
	Bin: "0x60806040523480156200001157600080fd5b506200004962000032600160008051602062002a4783398151915262000c39565b60001b600019620000c960201b62000e061760201c565b6040805160c080820183526001808352602080840182905260028486015260006060808601829052608080870183905260a08088018490528851968701895283875293860183905296850182905284018190529483018590528201849052620000c39361dead9390928392839290918391908290620000cd565b62000d5c565b9055565b600054610100900460ff1615808015620000ee5750600054600160ff909116105b806200011e57506200010b30620004df60201b62000e0a1760201c565b1580156200011e575060005460ff166001145b620001875760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805460ff191660011790558015620001ab576000805461ff0019166101001790555b620001b5620004ee565b620001c08a62000556565b620001cb87620005d5565b620001d7898962000627565b620001e2866200068b565b620002197f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c0886620000c960201b62000e061760201c565b6200025f6200024a60017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc59862000c39565b60001b84620000c960201b62000e061760201c565b620002a96200029060017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce958063762000c39565b60001b8360000151620000c960201b62000e061760201c565b620002f3620002da60017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a862000c39565b60001b8360200151620000c960201b62000e061760201c565b6200033d6200032460017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad637762000c39565b60001b8360400151620000c960201b62000e061760201c565b620003876200036e60017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a687181662000c39565b60001b8360600151620000c960201b62000e061760201c565b620003d1620003b860017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad62000c39565b60001b8360800151620000c960201b62000e061760201c565b6200041b6200040260017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d62000c39565b60001b8360a00151620000c960201b62000e061760201c565b6200042562000728565b620004308462000799565b6200043a62000add565b6001600160401b0316866001600160401b031610156200048c5760405162461bcd60e51b815260206004820152601f6024820152600080516020620029e783398151915260448201526064016200017e565b8015620004d3576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b50505050505050505050565b6001600160a01b03163b151590565b600054610100900460ff166200054a5760405162461bcd60e51b815260206004820152602b602482015260008051602062002a2783398151915260448201526a6e697469616c697a696e6760a81b60648201526084016200017e565b6200055462000b0a565b565b6200056062000b71565b6001600160a01b038116620005c75760405162461bcd60e51b815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201526564647265737360d01b60648201526084016200017e565b620005d28162000bcd565b50565b60678190556040805160208082018490528251808303909101815290820190915260005b600060008051602062002a07833981519152836040516200061b919062000c53565b60405180910390a35050565b60658290556066819055604080516020810184905290810182905260009060600160408051601f1981840301815291905290506001600060008051602062002a07833981519152836040516200067e919062000c53565b60405180910390a3505050565b6200069562000add565b6001600160401b0316816001600160401b03161015620006e75760405162461bcd60e51b815260206004820152601f6024820152600080516020620029e783398151915260448201526064016200017e565b606880546001600160401b0319166001600160401b0383169081179091556040805160208082019390935281518082039093018352810190526002620005f9565b6200075c62000748600160008051602062002a4783398151915262000c39565b60001b62000c1f60201b62000e261760201c565b60000362000554576200055462000784600160008051602062002a4783398151915262000c39565b60001b43620000c960201b62000e061760201c565b8060a001516001600160801b0316816060015163ffffffff161115620008285760405162461bcd60e51b815260206004820152603560248201527f53797374656d436f6e6669673a206d696e206261736520666565206d7573742060448201527f6265206c657373207468616e206d61782062617365000000000000000000000060648201526084016200017e565b6001816040015160ff1611620008995760405162461bcd60e51b815260206004820152602f60248201527f53797374656d436f6e6669673a2064656e6f6d696e61746f72206d757374206260448201526e65206c6172676572207468616e203160881b60648201526084016200017e565b606854608082015182516001600160401b0390921691620008bb919062000cab565b63ffffffff161115620009005760405162461bcd60e51b815260206004820152601f6024820152600080516020620029e783398151915260448201526064016200017e565b6000816020015160ff1611620009715760405162461bcd60e51b815260206004820152602f60248201527f53797374656d436f6e6669673a20656c6173746963697479206d756c7469706c60448201526e06965722063616e6e6f74206265203608c1b60648201526084016200017e565b8051602082015163ffffffff82169160ff909116906200099390829062000cd6565b6200099f919062000d08565b63ffffffff161462000a1a5760405162461bcd60e51b815260206004820152603760248201527f53797374656d436f6e6669673a20707265636973696f6e206c6f73732077697460448201527f6820746172676574207265736f75726365206c696d697400000000000000000060648201526084016200017e565b805160698054602084015160408501516060860151608087015160a09097015163ffffffff96871664ffffffffff199095169490941764010000000060ff948516021764ffffffffff60281b191665010000000000939092169290920263ffffffff60301b19161766010000000000009185169190910217600160501b600160f01b0319166a01000000000000000000009390941692909202600160701b600160f01b03191692909217600160701b6001600160801b0390921691909102179055565b60695460009062000b059063ffffffff6a010000000000000000000082048116911662000d37565b905090565b600054610100900460ff1662000b665760405162461bcd60e51b815260206004820152602b602482015260008051602062002a2783398151915260448201526a6e697469616c697a696e6760a81b60648201526084016200017e565b620005543362000bcd565b6033546001600160a01b03163314620005545760405162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016200017e565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b5490565b634e487b7160e01b600052601160045260246000fd5b60008282101562000c4e5762000c4e62000c23565b500390565b600060208083528351808285015260005b8181101562000c825785810183015185820160400152820162000c64565b8181111562000c95576000604083870101525b50601f01601f1916929092016040019392505050565b600063ffffffff80831681851680830382111562000ccd5762000ccd62000c23565b01949350505050565b600063ffffffff8084168062000cfc57634e487b7160e01b600052601260045260246000fd5b92169190910492915050565b600063ffffffff8083168185168183048111821515161562000d2e5762000d2e62000c23565b02949350505050565b60006001600160401b0382811684821680830382111562000ccd5762000ccd62000c23565b611c7b8062000d6c6000396000f3fe608060405234801561001057600080fd5b50600436106102415760003560e01c8063935f029e11610145578063dac6e63a116100bd578063f45e65d81161008c578063f8c68de011610071578063f8c68de014610597578063fd32aa0f1461059f578063ffa1ad74146105a757600080fd5b8063f45e65d81461057a578063f68016b71461058357600080fd5b8063dac6e63a1461054e578063e0e2016d14610556578063e81b2c6d1461055e578063f2fde38b1461056757600080fd5b8063bc49ce5f11610114578063c71973f6116100f9578063c71973f6146103f4578063c9b26f6114610407578063cc731b021461041a57600080fd5b8063bc49ce5f146103e4578063c4e8ddfa146103ec57600080fd5b8063935f029e146103ae5780639b7d7f0a146103c1578063a7119869146103c9578063b40a817c146103d157600080fd5b806348cd4cb1116101d857806354fd4d50116101a757806361d157681161018c57806361d1576814610380578063715018a6146103885780638da5cb5b1461039057600080fd5b806354fd4d501461032f5780635d73369c1461037857600080fd5b806348cd4cb1146102d75780634add321d146102df5780634d9f1559146103005780634f16540b1461030857600080fd5b80630c18c162116102145780630c18c162146102ab57806318d13918146102b457806319f5cea8146102c75780631fd19ee1146102cf57600080fd5b806306c9265714610246578063078f29cf146102615780630a49cb031461028e5780630bbb796814610296575b600080fd5b61024e6105af565b6040519081526020015b60405180910390f35b6102696105dd565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610258565b610269610616565b6102a96102a4366004611928565b610646565b005b61024e60655481565b6102a96102c2366004611a55565b610a44565b61024e610a58565b610269610a83565b61024e610aad565b6102e7610add565b60405167ffffffffffffffff9091168152602001610258565b610269610b03565b61024e7f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c0881565b61036b6040518060400160405280600681526020017f312e31322e30000000000000000000000000000000000000000000000000000081525081565b6040516102589190611ae2565b61024e610b33565b61024e610b5e565b6102a9610b89565b60335473ffffffffffffffffffffffffffffffffffffffff16610269565b6102a96103bc366004611af5565b610b9d565b610269610bb3565b610269610be3565b6102a96103df366004611b17565b610c13565b61024e610c24565b610269610c4f565b6102a9610402366004611b32565b610c7f565b6102a9610415366004611b4e565b610c90565b6104de6040805160c081018252600080825260208201819052918101829052606081018290526080810182905260a0810191909152506040805160c08101825260695463ffffffff8082168352640100000000820460ff9081166020850152650100000000008304169383019390935266010000000000008104831660608301526a0100000000000000000000810490921660808201526e0100000000000000000000000000009091046fffffffffffffffffffffffffffffffff1660a082015290565b6040516102589190600060c08201905063ffffffff80845116835260ff602085015116602084015260ff6040850151166040840152806060850151166060840152806080850151166080840152506fffffffffffffffffffffffffffffffff60a08401511660a083015292915050565b610269610ca1565b61024e610cd1565b61024e60675481565b6102a9610575366004611a55565b610cfc565b61024e60665481565b6068546102e79067ffffffffffffffff1681565b61024e610db0565b61024e610ddb565b61024e600081565b6105da60017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d611b96565b81565b600061061161060d60017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad6377611b96565b5490565b905090565b600061061161060d60017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad611b96565b600054610100900460ff16158080156106665750600054600160ff909116105b806106805750303b158015610680575060005460ff166001145b610711576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084015b60405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001179055801561076f57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b610777610e2a565b6107808a610cfc565b61078987610ec9565b6107938989610f2a565b61079c86610fbb565b6107c57f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08869055565b6107f86107f360017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc598611b96565b849055565b61082c61082660017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce9580637611b96565b83519055565b61086361085a60017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a8611b96565b60208401519055565b61089a61089160017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad6377611b96565b60408401519055565b6108d16108c860017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a6871816611b96565b60608401519055565b6109086108ff60017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad611b96565b60808401519055565b61093f61093660017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d611b96565b60a08401519055565b610947611099565b61095084611101565b610958610add565b67ffffffffffffffff168667ffffffffffffffff1610156109d5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f77006044820152606401610708565b8015610a3857600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b50505050505050505050565b610a4c611575565b610a55816115f6565b50565b6105da60017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a8611b96565b60006106117f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c085490565b600061061161060d60017fa11ee3ab75b40e88a0105e935d17cd36c8faee0138320d776c411291bdbbb1a0611b96565b6069546000906106119063ffffffff6a0100000000000000000000820481169116611bad565b600061061161060d60017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a6871816611b96565b6105da60017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce9580637611b96565b6105da60017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a6871816611b96565b610b91611575565b610b9b600061167a565b565b610ba5611575565b610baf8282610f2a565b5050565b600061061161060d60017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d611b96565b600061061161060d60017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce9580637611b96565b610c1b611575565b610a5581610fbb565b6105da60017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc598611b96565b600061061161060d60017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a8611b96565b610c87611575565b610a5581611101565b610c98611575565b610a5581610ec9565b600061061161060d60017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc598611b96565b6105da60017fa11ee3ab75b40e88a0105e935d17cd36c8faee0138320d776c411291bdbbb1a0611b96565b610d04611575565b73ffffffffffffffffffffffffffffffffffffffff8116610da7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f64647265737300000000000000000000000000000000000000000000000000006064820152608401610708565b610a558161167a565b6105da60017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad6377611b96565b6105da60017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad611b96565b9055565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b5490565b600054610100900460ff16610ec1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e670000000000000000000000000000000000000000006064820152608401610708565b610b9b6116f1565b60678190556040805160208082018490528251808303909101815290820190915260005b60007f1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be83604051610f1e9190611ae2565b60405180910390a35050565b606582905560668190556040805160208101849052908101829052600090606001604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190529050600160007f1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be83604051610fae9190611ae2565b60405180910390a3505050565b610fc3610add565b67ffffffffffffffff168167ffffffffffffffff161015611040576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f77006044820152606401610708565b606880547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff83169081179091556040805160208082019390935281518082039093018352810190526002610eed565b6110c761060d60017fa11ee3ab75b40e88a0105e935d17cd36c8faee0138320d776c411291bdbbb1a0611b96565b600003610b9b57610b9b6110fc60017fa11ee3ab75b40e88a0105e935d17cd36c8faee0138320d776c411291bdbbb1a0611b96565b439055565b8060a001516fffffffffffffffffffffffffffffffff16816060015163ffffffff1611156111b1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603560248201527f53797374656d436f6e6669673a206d696e206261736520666565206d7573742060448201527f6265206c657373207468616e206d6178206261736500000000000000000000006064820152608401610708565b6001816040015160ff1611611248576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f53797374656d436f6e6669673a2064656e6f6d696e61746f72206d757374206260448201527f65206c6172676572207468616e203100000000000000000000000000000000006064820152608401610708565b6068546080820151825167ffffffffffffffff909216916112699190611bd9565b63ffffffff1611156112d7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f77006044820152606401610708565b6000816020015160ff161161136e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f53797374656d436f6e6669673a20656c6173746963697479206d756c7469706c60448201527f6965722063616e6e6f74206265203000000000000000000000000000000000006064820152608401610708565b8051602082015163ffffffff82169160ff9091169061138e908290611bf8565b6113989190611c42565b63ffffffff161461142b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f53797374656d436f6e6669673a20707265636973696f6e206c6f73732077697460448201527f6820746172676574207265736f75726365206c696d69740000000000000000006064820152608401610708565b805160698054602084015160408501516060860151608087015160a09097015163ffffffff9687167fffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000009095169490941764010000000060ff94851602177fffffffffffffffffffffffffffffffffffffffffffff0000000000ffffffffff166501000000000093909216929092027fffffffffffffffffffffffffffffffffffffffffffff00000000ffffffffffff1617660100000000000091851691909102177fffff0000000000000000000000000000000000000000ffffffffffffffffffff166a010000000000000000000093909416929092027fffff00000000000000000000000000000000ffffffffffffffffffffffffffff16929092176e0100000000000000000000000000006fffffffffffffffffffffffffffffffff90921691909102179055565b60335473ffffffffffffffffffffffffffffffffffffffff163314610b9b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401610708565b61161f7f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08829055565b6040805173ffffffffffffffffffffffffffffffffffffffff8316602082015260009101604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815291905290506003610eed565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600054610100900460ff16611788576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e670000000000000000000000000000000000000000006064820152608401610708565b610b9b3361167a565b803573ffffffffffffffffffffffffffffffffffffffff811681146117b557600080fd5b919050565b803567ffffffffffffffff811681146117b557600080fd5b60405160c0810167ffffffffffffffff8111828210171561181c577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60405290565b803563ffffffff811681146117b557600080fd5b803560ff811681146117b557600080fd5b600060c0828403121561185957600080fd5b60405160c0810181811067ffffffffffffffff821117156118a3577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040529050806118b283611822565b81526118c060208401611836565b60208201526118d160408401611836565b60408201526118e260608401611822565b60608201526118f360808401611822565b608082015260a08301356fffffffffffffffffffffffffffffffff8116811461191b57600080fd5b60a0919091015292915050565b6000806000806000806000806000898b0361026081121561194857600080fd5b6119518b611791565b995060208b0135985060408b0135975060608b0135965061197460808c016117ba565b955061198260a08c01611791565b94506119918c60c08d01611847565b93506119a06101808c01611791565b925060c07ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe60820112156119d257600080fd5b506119db6117d2565b6119e86101a08c01611791565b81526119f76101c08c01611791565b6020820152611a096101e08c01611791565b6040820152611a1b6102008c01611791565b6060820152611a2d6102208c01611791565b6080820152611a3f6102408c01611791565b60a0820152809150509295985092959850929598565b600060208284031215611a6757600080fd5b611a7082611791565b9392505050565b6000815180845260005b81811015611a9d57602081850181015186830182015201611a81565b81811115611aaf576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000611a706020830184611a77565b60008060408385031215611b0857600080fd5b50508035926020909101359150565b600060208284031215611b2957600080fd5b611a70826117ba565b600060c08284031215611b4457600080fd5b611a708383611847565b600060208284031215611b6057600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015611ba857611ba8611b67565b500390565b600067ffffffffffffffff808316818516808303821115611bd057611bd0611b67565b01949350505050565b600063ffffffff808316818516808303821115611bd057611bd0611b67565b600063ffffffff80841680611c36577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b92169190910492915050565b600063ffffffff80831681851681830481118215151615611c6557611c65611b67565b0294935050505056fea164736f6c634300080f000a53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f77001d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be496e697469616c697a61626c653a20636f6e7472616374206973206e6f742069a11ee3ab75b40e88a0105e935d17cd36c8faee0138320d776c411291bdbbb1a0",
}

// SystemConfigABI is the input ABI used to generate the binding from.
// Deprecated: Use SystemConfigMetaData.ABI instead.
var SystemConfigABI = SystemConfigMetaData.ABI

// SystemConfigBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SystemConfigMetaData.Bin instead.
var SystemConfigBin = SystemConfigMetaData.Bin

// DeploySystemConfig deploys a new Ethereum contract, binding an instance of SystemConfig to it.
func DeploySystemConfig(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SystemConfig, error) {
	parsed, err := SystemConfigMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SystemConfigBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SystemConfig{SystemConfigCaller: SystemConfigCaller{contract: contract}, SystemConfigTransactor: SystemConfigTransactor{contract: contract}, SystemConfigFilterer: SystemConfigFilterer{contract: contract}}, nil
}

// SystemConfig is an auto generated Go binding around an Ethereum contract.
type SystemConfig struct {
	SystemConfigCaller     // Read-only binding to the contract
	SystemConfigTransactor // Write-only binding to the contract
	SystemConfigFilterer   // Log filterer for contract events
}

// SystemConfigCaller is an auto generated read-only Go binding around an Ethereum contract.
type SystemConfigCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SystemConfigTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SystemConfigTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SystemConfigFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SystemConfigFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SystemConfigSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SystemConfigSession struct {
	Contract     *SystemConfig     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SystemConfigCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SystemConfigCallerSession struct {
	Contract *SystemConfigCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// SystemConfigTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SystemConfigTransactorSession struct {
	Contract     *SystemConfigTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// SystemConfigRaw is an auto generated low-level Go binding around an Ethereum contract.
type SystemConfigRaw struct {
	Contract *SystemConfig // Generic contract binding to access the raw methods on
}

// SystemConfigCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SystemConfigCallerRaw struct {
	Contract *SystemConfigCaller // Generic read-only contract binding to access the raw methods on
}

// SystemConfigTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SystemConfigTransactorRaw struct {
	Contract *SystemConfigTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSystemConfig creates a new instance of SystemConfig, bound to a specific deployed contract.
func NewSystemConfig(address common.Address, backend bind.ContractBackend) (*SystemConfig, error) {
	contract, err := bindSystemConfig(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SystemConfig{SystemConfigCaller: SystemConfigCaller{contract: contract}, SystemConfigTransactor: SystemConfigTransactor{contract: contract}, SystemConfigFilterer: SystemConfigFilterer{contract: contract}}, nil
}

// NewSystemConfigCaller creates a new read-only instance of SystemConfig, bound to a specific deployed contract.
func NewSystemConfigCaller(address common.Address, caller bind.ContractCaller) (*SystemConfigCaller, error) {
	contract, err := bindSystemConfig(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SystemConfigCaller{contract: contract}, nil
}

// NewSystemConfigTransactor creates a new write-only instance of SystemConfig, bound to a specific deployed contract.
func NewSystemConfigTransactor(address common.Address, transactor bind.ContractTransactor) (*SystemConfigTransactor, error) {
	contract, err := bindSystemConfig(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SystemConfigTransactor{contract: contract}, nil
}

// NewSystemConfigFilterer creates a new log filterer instance of SystemConfig, bound to a specific deployed contract.
func NewSystemConfigFilterer(address common.Address, filterer bind.ContractFilterer) (*SystemConfigFilterer, error) {
	contract, err := bindSystemConfig(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SystemConfigFilterer{contract: contract}, nil
}

// bindSystemConfig binds a generic wrapper to an already deployed contract.
func bindSystemConfig(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SystemConfigABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SystemConfig *SystemConfigRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SystemConfig.Contract.SystemConfigCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SystemConfig *SystemConfigRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SystemConfig.Contract.SystemConfigTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SystemConfig *SystemConfigRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SystemConfig.Contract.SystemConfigTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SystemConfig *SystemConfigCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SystemConfig.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SystemConfig *SystemConfigTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SystemConfig.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SystemConfig *SystemConfigTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SystemConfig.Contract.contract.Transact(opts, method, params...)
}

// BATCHINBOXSLOT is a free data retrieval call binding the contract method 0xbc49ce5f.
//
// Solidity: function BATCH_INBOX_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCaller) BATCHINBOXSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "BATCH_INBOX_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// BATCHINBOXSLOT is a free data retrieval call binding the contract method 0xbc49ce5f.
//
// Solidity: function BATCH_INBOX_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigSession) BATCHINBOXSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.BATCHINBOXSLOT(&_SystemConfig.CallOpts)
}

// BATCHINBOXSLOT is a free data retrieval call binding the contract method 0xbc49ce5f.
//
// Solidity: function BATCH_INBOX_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCallerSession) BATCHINBOXSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.BATCHINBOXSLOT(&_SystemConfig.CallOpts)
}

// L1CROSSDOMAINMESSENGERSLOT is a free data retrieval call binding the contract method 0x5d73369c.
//
// Solidity: function L1_CROSS_DOMAIN_MESSENGER_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCaller) L1CROSSDOMAINMESSENGERSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "L1_CROSS_DOMAIN_MESSENGER_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L1CROSSDOMAINMESSENGERSLOT is a free data retrieval call binding the contract method 0x5d73369c.
//
// Solidity: function L1_CROSS_DOMAIN_MESSENGER_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigSession) L1CROSSDOMAINMESSENGERSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.L1CROSSDOMAINMESSENGERSLOT(&_SystemConfig.CallOpts)
}

// L1CROSSDOMAINMESSENGERSLOT is a free data retrieval call binding the contract method 0x5d73369c.
//
// Solidity: function L1_CROSS_DOMAIN_MESSENGER_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCallerSession) L1CROSSDOMAINMESSENGERSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.L1CROSSDOMAINMESSENGERSLOT(&_SystemConfig.CallOpts)
}

// L1ERC721BRIDGESLOT is a free data retrieval call binding the contract method 0x19f5cea8.
//
// Solidity: function L1_ERC_721_BRIDGE_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCaller) L1ERC721BRIDGESLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "L1_ERC_721_BRIDGE_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L1ERC721BRIDGESLOT is a free data retrieval call binding the contract method 0x19f5cea8.
//
// Solidity: function L1_ERC_721_BRIDGE_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigSession) L1ERC721BRIDGESLOT() ([32]byte, error) {
	return _SystemConfig.Contract.L1ERC721BRIDGESLOT(&_SystemConfig.CallOpts)
}

// L1ERC721BRIDGESLOT is a free data retrieval call binding the contract method 0x19f5cea8.
//
// Solidity: function L1_ERC_721_BRIDGE_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCallerSession) L1ERC721BRIDGESLOT() ([32]byte, error) {
	return _SystemConfig.Contract.L1ERC721BRIDGESLOT(&_SystemConfig.CallOpts)
}

// L1STANDARDBRIDGESLOT is a free data retrieval call binding the contract method 0xf8c68de0.
//
// Solidity: function L1_STANDARD_BRIDGE_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCaller) L1STANDARDBRIDGESLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "L1_STANDARD_BRIDGE_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L1STANDARDBRIDGESLOT is a free data retrieval call binding the contract method 0xf8c68de0.
//
// Solidity: function L1_STANDARD_BRIDGE_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigSession) L1STANDARDBRIDGESLOT() ([32]byte, error) {
	return _SystemConfig.Contract.L1STANDARDBRIDGESLOT(&_SystemConfig.CallOpts)
}

// L1STANDARDBRIDGESLOT is a free data retrieval call binding the contract method 0xf8c68de0.
//
// Solidity: function L1_STANDARD_BRIDGE_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCallerSession) L1STANDARDBRIDGESLOT() ([32]byte, error) {
	return _SystemConfig.Contract.L1STANDARDBRIDGESLOT(&_SystemConfig.CallOpts)
}

// L2OUTPUTORACLESLOT is a free data retrieval call binding the contract method 0x61d15768.
//
// Solidity: function L2_OUTPUT_ORACLE_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCaller) L2OUTPUTORACLESLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "L2_OUTPUT_ORACLE_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L2OUTPUTORACLESLOT is a free data retrieval call binding the contract method 0x61d15768.
//
// Solidity: function L2_OUTPUT_ORACLE_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigSession) L2OUTPUTORACLESLOT() ([32]byte, error) {
	return _SystemConfig.Contract.L2OUTPUTORACLESLOT(&_SystemConfig.CallOpts)
}

// L2OUTPUTORACLESLOT is a free data retrieval call binding the contract method 0x61d15768.
//
// Solidity: function L2_OUTPUT_ORACLE_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCallerSession) L2OUTPUTORACLESLOT() ([32]byte, error) {
	return _SystemConfig.Contract.L2OUTPUTORACLESLOT(&_SystemConfig.CallOpts)
}

// OPTIMISMMINTABLEERC20FACTORYSLOT is a free data retrieval call binding the contract method 0x06c92657.
//
// Solidity: function OPTIMISM_MINTABLE_ERC20_FACTORY_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCaller) OPTIMISMMINTABLEERC20FACTORYSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "OPTIMISM_MINTABLE_ERC20_FACTORY_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// OPTIMISMMINTABLEERC20FACTORYSLOT is a free data retrieval call binding the contract method 0x06c92657.
//
// Solidity: function OPTIMISM_MINTABLE_ERC20_FACTORY_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigSession) OPTIMISMMINTABLEERC20FACTORYSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.OPTIMISMMINTABLEERC20FACTORYSLOT(&_SystemConfig.CallOpts)
}

// OPTIMISMMINTABLEERC20FACTORYSLOT is a free data retrieval call binding the contract method 0x06c92657.
//
// Solidity: function OPTIMISM_MINTABLE_ERC20_FACTORY_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCallerSession) OPTIMISMMINTABLEERC20FACTORYSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.OPTIMISMMINTABLEERC20FACTORYSLOT(&_SystemConfig.CallOpts)
}

// OPTIMISMPORTALSLOT is a free data retrieval call binding the contract method 0xfd32aa0f.
//
// Solidity: function OPTIMISM_PORTAL_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCaller) OPTIMISMPORTALSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "OPTIMISM_PORTAL_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// OPTIMISMPORTALSLOT is a free data retrieval call binding the contract method 0xfd32aa0f.
//
// Solidity: function OPTIMISM_PORTAL_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigSession) OPTIMISMPORTALSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.OPTIMISMPORTALSLOT(&_SystemConfig.CallOpts)
}

// OPTIMISMPORTALSLOT is a free data retrieval call binding the contract method 0xfd32aa0f.
//
// Solidity: function OPTIMISM_PORTAL_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCallerSession) OPTIMISMPORTALSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.OPTIMISMPORTALSLOT(&_SystemConfig.CallOpts)
}

// STARTBLOCKSLOT is a free data retrieval call binding the contract method 0xe0e2016d.
//
// Solidity: function START_BLOCK_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCaller) STARTBLOCKSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "START_BLOCK_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// STARTBLOCKSLOT is a free data retrieval call binding the contract method 0xe0e2016d.
//
// Solidity: function START_BLOCK_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigSession) STARTBLOCKSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.STARTBLOCKSLOT(&_SystemConfig.CallOpts)
}

// STARTBLOCKSLOT is a free data retrieval call binding the contract method 0xe0e2016d.
//
// Solidity: function START_BLOCK_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCallerSession) STARTBLOCKSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.STARTBLOCKSLOT(&_SystemConfig.CallOpts)
}

// UNSAFEBLOCKSIGNERSLOT is a free data retrieval call binding the contract method 0x4f16540b.
//
// Solidity: function UNSAFE_BLOCK_SIGNER_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCaller) UNSAFEBLOCKSIGNERSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "UNSAFE_BLOCK_SIGNER_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// UNSAFEBLOCKSIGNERSLOT is a free data retrieval call binding the contract method 0x4f16540b.
//
// Solidity: function UNSAFE_BLOCK_SIGNER_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigSession) UNSAFEBLOCKSIGNERSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.UNSAFEBLOCKSIGNERSLOT(&_SystemConfig.CallOpts)
}

// UNSAFEBLOCKSIGNERSLOT is a free data retrieval call binding the contract method 0x4f16540b.
//
// Solidity: function UNSAFE_BLOCK_SIGNER_SLOT() view returns(bytes32)
func (_SystemConfig *SystemConfigCallerSession) UNSAFEBLOCKSIGNERSLOT() ([32]byte, error) {
	return _SystemConfig.Contract.UNSAFEBLOCKSIGNERSLOT(&_SystemConfig.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_SystemConfig *SystemConfigCaller) VERSION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "VERSION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_SystemConfig *SystemConfigSession) VERSION() (*big.Int, error) {
	return _SystemConfig.Contract.VERSION(&_SystemConfig.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_SystemConfig *SystemConfigCallerSession) VERSION() (*big.Int, error) {
	return _SystemConfig.Contract.VERSION(&_SystemConfig.CallOpts)
}

// BatchInbox is a free data retrieval call binding the contract method 0xdac6e63a.
//
// Solidity: function batchInbox() view returns(address addr_)
func (_SystemConfig *SystemConfigCaller) BatchInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "batchInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BatchInbox is a free data retrieval call binding the contract method 0xdac6e63a.
//
// Solidity: function batchInbox() view returns(address addr_)
func (_SystemConfig *SystemConfigSession) BatchInbox() (common.Address, error) {
	return _SystemConfig.Contract.BatchInbox(&_SystemConfig.CallOpts)
}

// BatchInbox is a free data retrieval call binding the contract method 0xdac6e63a.
//
// Solidity: function batchInbox() view returns(address addr_)
func (_SystemConfig *SystemConfigCallerSession) BatchInbox() (common.Address, error) {
	return _SystemConfig.Contract.BatchInbox(&_SystemConfig.CallOpts)
}

// BatcherHash is a free data retrieval call binding the contract method 0xe81b2c6d.
//
// Solidity: function batcherHash() view returns(bytes32)
func (_SystemConfig *SystemConfigCaller) BatcherHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "batcherHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// BatcherHash is a free data retrieval call binding the contract method 0xe81b2c6d.
//
// Solidity: function batcherHash() view returns(bytes32)
func (_SystemConfig *SystemConfigSession) BatcherHash() ([32]byte, error) {
	return _SystemConfig.Contract.BatcherHash(&_SystemConfig.CallOpts)
}

// BatcherHash is a free data retrieval call binding the contract method 0xe81b2c6d.
//
// Solidity: function batcherHash() view returns(bytes32)
func (_SystemConfig *SystemConfigCallerSession) BatcherHash() ([32]byte, error) {
	return _SystemConfig.Contract.BatcherHash(&_SystemConfig.CallOpts)
}

// GasLimit is a free data retrieval call binding the contract method 0xf68016b7.
//
// Solidity: function gasLimit() view returns(uint64)
func (_SystemConfig *SystemConfigCaller) GasLimit(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "gasLimit")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GasLimit is a free data retrieval call binding the contract method 0xf68016b7.
//
// Solidity: function gasLimit() view returns(uint64)
func (_SystemConfig *SystemConfigSession) GasLimit() (uint64, error) {
	return _SystemConfig.Contract.GasLimit(&_SystemConfig.CallOpts)
}

// GasLimit is a free data retrieval call binding the contract method 0xf68016b7.
//
// Solidity: function gasLimit() view returns(uint64)
func (_SystemConfig *SystemConfigCallerSession) GasLimit() (uint64, error) {
	return _SystemConfig.Contract.GasLimit(&_SystemConfig.CallOpts)
}

// L1CrossDomainMessenger is a free data retrieval call binding the contract method 0xa7119869.
//
// Solidity: function l1CrossDomainMessenger() view returns(address addr_)
func (_SystemConfig *SystemConfigCaller) L1CrossDomainMessenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "l1CrossDomainMessenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1CrossDomainMessenger is a free data retrieval call binding the contract method 0xa7119869.
//
// Solidity: function l1CrossDomainMessenger() view returns(address addr_)
func (_SystemConfig *SystemConfigSession) L1CrossDomainMessenger() (common.Address, error) {
	return _SystemConfig.Contract.L1CrossDomainMessenger(&_SystemConfig.CallOpts)
}

// L1CrossDomainMessenger is a free data retrieval call binding the contract method 0xa7119869.
//
// Solidity: function l1CrossDomainMessenger() view returns(address addr_)
func (_SystemConfig *SystemConfigCallerSession) L1CrossDomainMessenger() (common.Address, error) {
	return _SystemConfig.Contract.L1CrossDomainMessenger(&_SystemConfig.CallOpts)
}

// L1ERC721Bridge is a free data retrieval call binding the contract method 0xc4e8ddfa.
//
// Solidity: function l1ERC721Bridge() view returns(address addr_)
func (_SystemConfig *SystemConfigCaller) L1ERC721Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "l1ERC721Bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1ERC721Bridge is a free data retrieval call binding the contract method 0xc4e8ddfa.
//
// Solidity: function l1ERC721Bridge() view returns(address addr_)
func (_SystemConfig *SystemConfigSession) L1ERC721Bridge() (common.Address, error) {
	return _SystemConfig.Contract.L1ERC721Bridge(&_SystemConfig.CallOpts)
}

// L1ERC721Bridge is a free data retrieval call binding the contract method 0xc4e8ddfa.
//
// Solidity: function l1ERC721Bridge() view returns(address addr_)
func (_SystemConfig *SystemConfigCallerSession) L1ERC721Bridge() (common.Address, error) {
	return _SystemConfig.Contract.L1ERC721Bridge(&_SystemConfig.CallOpts)
}

// L1StandardBridge is a free data retrieval call binding the contract method 0x078f29cf.
//
// Solidity: function l1StandardBridge() view returns(address addr_)
func (_SystemConfig *SystemConfigCaller) L1StandardBridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "l1StandardBridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1StandardBridge is a free data retrieval call binding the contract method 0x078f29cf.
//
// Solidity: function l1StandardBridge() view returns(address addr_)
func (_SystemConfig *SystemConfigSession) L1StandardBridge() (common.Address, error) {
	return _SystemConfig.Contract.L1StandardBridge(&_SystemConfig.CallOpts)
}

// L1StandardBridge is a free data retrieval call binding the contract method 0x078f29cf.
//
// Solidity: function l1StandardBridge() view returns(address addr_)
func (_SystemConfig *SystemConfigCallerSession) L1StandardBridge() (common.Address, error) {
	return _SystemConfig.Contract.L1StandardBridge(&_SystemConfig.CallOpts)
}

// L2OutputOracle is a free data retrieval call binding the contract method 0x4d9f1559.
//
// Solidity: function l2OutputOracle() view returns(address addr_)
func (_SystemConfig *SystemConfigCaller) L2OutputOracle(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "l2OutputOracle")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2OutputOracle is a free data retrieval call binding the contract method 0x4d9f1559.
//
// Solidity: function l2OutputOracle() view returns(address addr_)
func (_SystemConfig *SystemConfigSession) L2OutputOracle() (common.Address, error) {
	return _SystemConfig.Contract.L2OutputOracle(&_SystemConfig.CallOpts)
}

// L2OutputOracle is a free data retrieval call binding the contract method 0x4d9f1559.
//
// Solidity: function l2OutputOracle() view returns(address addr_)
func (_SystemConfig *SystemConfigCallerSession) L2OutputOracle() (common.Address, error) {
	return _SystemConfig.Contract.L2OutputOracle(&_SystemConfig.CallOpts)
}

// MinimumGasLimit is a free data retrieval call binding the contract method 0x4add321d.
//
// Solidity: function minimumGasLimit() view returns(uint64)
func (_SystemConfig *SystemConfigCaller) MinimumGasLimit(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "minimumGasLimit")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MinimumGasLimit is a free data retrieval call binding the contract method 0x4add321d.
//
// Solidity: function minimumGasLimit() view returns(uint64)
func (_SystemConfig *SystemConfigSession) MinimumGasLimit() (uint64, error) {
	return _SystemConfig.Contract.MinimumGasLimit(&_SystemConfig.CallOpts)
}

// MinimumGasLimit is a free data retrieval call binding the contract method 0x4add321d.
//
// Solidity: function minimumGasLimit() view returns(uint64)
func (_SystemConfig *SystemConfigCallerSession) MinimumGasLimit() (uint64, error) {
	return _SystemConfig.Contract.MinimumGasLimit(&_SystemConfig.CallOpts)
}

// OptimismMintableERC20Factory is a free data retrieval call binding the contract method 0x9b7d7f0a.
//
// Solidity: function optimismMintableERC20Factory() view returns(address addr_)
func (_SystemConfig *SystemConfigCaller) OptimismMintableERC20Factory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "optimismMintableERC20Factory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OptimismMintableERC20Factory is a free data retrieval call binding the contract method 0x9b7d7f0a.
//
// Solidity: function optimismMintableERC20Factory() view returns(address addr_)
func (_SystemConfig *SystemConfigSession) OptimismMintableERC20Factory() (common.Address, error) {
	return _SystemConfig.Contract.OptimismMintableERC20Factory(&_SystemConfig.CallOpts)
}

// OptimismMintableERC20Factory is a free data retrieval call binding the contract method 0x9b7d7f0a.
//
// Solidity: function optimismMintableERC20Factory() view returns(address addr_)
func (_SystemConfig *SystemConfigCallerSession) OptimismMintableERC20Factory() (common.Address, error) {
	return _SystemConfig.Contract.OptimismMintableERC20Factory(&_SystemConfig.CallOpts)
}

// OptimismPortal is a free data retrieval call binding the contract method 0x0a49cb03.
//
// Solidity: function optimismPortal() view returns(address addr_)
func (_SystemConfig *SystemConfigCaller) OptimismPortal(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "optimismPortal")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OptimismPortal is a free data retrieval call binding the contract method 0x0a49cb03.
//
// Solidity: function optimismPortal() view returns(address addr_)
func (_SystemConfig *SystemConfigSession) OptimismPortal() (common.Address, error) {
	return _SystemConfig.Contract.OptimismPortal(&_SystemConfig.CallOpts)
}

// OptimismPortal is a free data retrieval call binding the contract method 0x0a49cb03.
//
// Solidity: function optimismPortal() view returns(address addr_)
func (_SystemConfig *SystemConfigCallerSession) OptimismPortal() (common.Address, error) {
	return _SystemConfig.Contract.OptimismPortal(&_SystemConfig.CallOpts)
}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_SystemConfig *SystemConfigCaller) Overhead(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "overhead")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_SystemConfig *SystemConfigSession) Overhead() (*big.Int, error) {
	return _SystemConfig.Contract.Overhead(&_SystemConfig.CallOpts)
}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_SystemConfig *SystemConfigCallerSession) Overhead() (*big.Int, error) {
	return _SystemConfig.Contract.Overhead(&_SystemConfig.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SystemConfig *SystemConfigCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SystemConfig *SystemConfigSession) Owner() (common.Address, error) {
	return _SystemConfig.Contract.Owner(&_SystemConfig.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SystemConfig *SystemConfigCallerSession) Owner() (common.Address, error) {
	return _SystemConfig.Contract.Owner(&_SystemConfig.CallOpts)
}

// ResourceConfig is a free data retrieval call binding the contract method 0xcc731b02.
//
// Solidity: function resourceConfig() view returns((uint32,uint8,uint8,uint32,uint32,uint128))
func (_SystemConfig *SystemConfigCaller) ResourceConfig(opts *bind.CallOpts) (ResourceMeteringResourceConfig, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "resourceConfig")

	if err != nil {
		return *new(ResourceMeteringResourceConfig), err
	}

	out0 := *abi.ConvertType(out[0], new(ResourceMeteringResourceConfig)).(*ResourceMeteringResourceConfig)

	return out0, err

}

// ResourceConfig is a free data retrieval call binding the contract method 0xcc731b02.
//
// Solidity: function resourceConfig() view returns((uint32,uint8,uint8,uint32,uint32,uint128))
func (_SystemConfig *SystemConfigSession) ResourceConfig() (ResourceMeteringResourceConfig, error) {
	return _SystemConfig.Contract.ResourceConfig(&_SystemConfig.CallOpts)
}

// ResourceConfig is a free data retrieval call binding the contract method 0xcc731b02.
//
// Solidity: function resourceConfig() view returns((uint32,uint8,uint8,uint32,uint32,uint128))
func (_SystemConfig *SystemConfigCallerSession) ResourceConfig() (ResourceMeteringResourceConfig, error) {
	return _SystemConfig.Contract.ResourceConfig(&_SystemConfig.CallOpts)
}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_SystemConfig *SystemConfigCaller) Scalar(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "scalar")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_SystemConfig *SystemConfigSession) Scalar() (*big.Int, error) {
	return _SystemConfig.Contract.Scalar(&_SystemConfig.CallOpts)
}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_SystemConfig *SystemConfigCallerSession) Scalar() (*big.Int, error) {
	return _SystemConfig.Contract.Scalar(&_SystemConfig.CallOpts)
}

// StartBlock is a free data retrieval call binding the contract method 0x48cd4cb1.
//
// Solidity: function startBlock() view returns(uint256 startBlock_)
func (_SystemConfig *SystemConfigCaller) StartBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "startBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StartBlock is a free data retrieval call binding the contract method 0x48cd4cb1.
//
// Solidity: function startBlock() view returns(uint256 startBlock_)
func (_SystemConfig *SystemConfigSession) StartBlock() (*big.Int, error) {
	return _SystemConfig.Contract.StartBlock(&_SystemConfig.CallOpts)
}

// StartBlock is a free data retrieval call binding the contract method 0x48cd4cb1.
//
// Solidity: function startBlock() view returns(uint256 startBlock_)
func (_SystemConfig *SystemConfigCallerSession) StartBlock() (*big.Int, error) {
	return _SystemConfig.Contract.StartBlock(&_SystemConfig.CallOpts)
}

// UnsafeBlockSigner is a free data retrieval call binding the contract method 0x1fd19ee1.
//
// Solidity: function unsafeBlockSigner() view returns(address addr_)
func (_SystemConfig *SystemConfigCaller) UnsafeBlockSigner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "unsafeBlockSigner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// UnsafeBlockSigner is a free data retrieval call binding the contract method 0x1fd19ee1.
//
// Solidity: function unsafeBlockSigner() view returns(address addr_)
func (_SystemConfig *SystemConfigSession) UnsafeBlockSigner() (common.Address, error) {
	return _SystemConfig.Contract.UnsafeBlockSigner(&_SystemConfig.CallOpts)
}

// UnsafeBlockSigner is a free data retrieval call binding the contract method 0x1fd19ee1.
//
// Solidity: function unsafeBlockSigner() view returns(address addr_)
func (_SystemConfig *SystemConfigCallerSession) UnsafeBlockSigner() (common.Address, error) {
	return _SystemConfig.Contract.UnsafeBlockSigner(&_SystemConfig.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SystemConfig *SystemConfigCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SystemConfig *SystemConfigSession) Version() (string, error) {
	return _SystemConfig.Contract.Version(&_SystemConfig.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SystemConfig *SystemConfigCallerSession) Version() (string, error) {
	return _SystemConfig.Contract.Version(&_SystemConfig.CallOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x0bbb7968.
//
// Solidity: function initialize(address _owner, uint256 _overhead, uint256 _scalar, bytes32 _batcherHash, uint64 _gasLimit, address _unsafeBlockSigner, (uint32,uint8,uint8,uint32,uint32,uint128) _config, address _batchInbox, (address,address,address,address,address,address) _addresses) returns()
func (_SystemConfig *SystemConfigTransactor) Initialize(opts *bind.TransactOpts, _owner common.Address, _overhead *big.Int, _scalar *big.Int, _batcherHash [32]byte, _gasLimit uint64, _unsafeBlockSigner common.Address, _config ResourceMeteringResourceConfig, _batchInbox common.Address, _addresses SystemConfigAddresses) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "initialize", _owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config, _batchInbox, _addresses)
}

// Initialize is a paid mutator transaction binding the contract method 0x0bbb7968.
//
// Solidity: function initialize(address _owner, uint256 _overhead, uint256 _scalar, bytes32 _batcherHash, uint64 _gasLimit, address _unsafeBlockSigner, (uint32,uint8,uint8,uint32,uint32,uint128) _config, address _batchInbox, (address,address,address,address,address,address) _addresses) returns()
func (_SystemConfig *SystemConfigSession) Initialize(_owner common.Address, _overhead *big.Int, _scalar *big.Int, _batcherHash [32]byte, _gasLimit uint64, _unsafeBlockSigner common.Address, _config ResourceMeteringResourceConfig, _batchInbox common.Address, _addresses SystemConfigAddresses) (*types.Transaction, error) {
	return _SystemConfig.Contract.Initialize(&_SystemConfig.TransactOpts, _owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config, _batchInbox, _addresses)
}

// Initialize is a paid mutator transaction binding the contract method 0x0bbb7968.
//
// Solidity: function initialize(address _owner, uint256 _overhead, uint256 _scalar, bytes32 _batcherHash, uint64 _gasLimit, address _unsafeBlockSigner, (uint32,uint8,uint8,uint32,uint32,uint128) _config, address _batchInbox, (address,address,address,address,address,address) _addresses) returns()
func (_SystemConfig *SystemConfigTransactorSession) Initialize(_owner common.Address, _overhead *big.Int, _scalar *big.Int, _batcherHash [32]byte, _gasLimit uint64, _unsafeBlockSigner common.Address, _config ResourceMeteringResourceConfig, _batchInbox common.Address, _addresses SystemConfigAddresses) (*types.Transaction, error) {
	return _SystemConfig.Contract.Initialize(&_SystemConfig.TransactOpts, _owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config, _batchInbox, _addresses)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_SystemConfig *SystemConfigTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_SystemConfig *SystemConfigSession) RenounceOwnership() (*types.Transaction, error) {
	return _SystemConfig.Contract.RenounceOwnership(&_SystemConfig.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_SystemConfig *SystemConfigTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _SystemConfig.Contract.RenounceOwnership(&_SystemConfig.TransactOpts)
}

// SetBatcherHash is a paid mutator transaction binding the contract method 0xc9b26f61.
//
// Solidity: function setBatcherHash(bytes32 _batcherHash) returns()
func (_SystemConfig *SystemConfigTransactor) SetBatcherHash(opts *bind.TransactOpts, _batcherHash [32]byte) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "setBatcherHash", _batcherHash)
}

// SetBatcherHash is a paid mutator transaction binding the contract method 0xc9b26f61.
//
// Solidity: function setBatcherHash(bytes32 _batcherHash) returns()
func (_SystemConfig *SystemConfigSession) SetBatcherHash(_batcherHash [32]byte) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetBatcherHash(&_SystemConfig.TransactOpts, _batcherHash)
}

// SetBatcherHash is a paid mutator transaction binding the contract method 0xc9b26f61.
//
// Solidity: function setBatcherHash(bytes32 _batcherHash) returns()
func (_SystemConfig *SystemConfigTransactorSession) SetBatcherHash(_batcherHash [32]byte) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetBatcherHash(&_SystemConfig.TransactOpts, _batcherHash)
}

// SetGasConfig is a paid mutator transaction binding the contract method 0x935f029e.
//
// Solidity: function setGasConfig(uint256 _overhead, uint256 _scalar) returns()
func (_SystemConfig *SystemConfigTransactor) SetGasConfig(opts *bind.TransactOpts, _overhead *big.Int, _scalar *big.Int) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "setGasConfig", _overhead, _scalar)
}

// SetGasConfig is a paid mutator transaction binding the contract method 0x935f029e.
//
// Solidity: function setGasConfig(uint256 _overhead, uint256 _scalar) returns()
func (_SystemConfig *SystemConfigSession) SetGasConfig(_overhead *big.Int, _scalar *big.Int) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetGasConfig(&_SystemConfig.TransactOpts, _overhead, _scalar)
}

// SetGasConfig is a paid mutator transaction binding the contract method 0x935f029e.
//
// Solidity: function setGasConfig(uint256 _overhead, uint256 _scalar) returns()
func (_SystemConfig *SystemConfigTransactorSession) SetGasConfig(_overhead *big.Int, _scalar *big.Int) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetGasConfig(&_SystemConfig.TransactOpts, _overhead, _scalar)
}

// SetGasLimit is a paid mutator transaction binding the contract method 0xb40a817c.
//
// Solidity: function setGasLimit(uint64 _gasLimit) returns()
func (_SystemConfig *SystemConfigTransactor) SetGasLimit(opts *bind.TransactOpts, _gasLimit uint64) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "setGasLimit", _gasLimit)
}

// SetGasLimit is a paid mutator transaction binding the contract method 0xb40a817c.
//
// Solidity: function setGasLimit(uint64 _gasLimit) returns()
func (_SystemConfig *SystemConfigSession) SetGasLimit(_gasLimit uint64) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetGasLimit(&_SystemConfig.TransactOpts, _gasLimit)
}

// SetGasLimit is a paid mutator transaction binding the contract method 0xb40a817c.
//
// Solidity: function setGasLimit(uint64 _gasLimit) returns()
func (_SystemConfig *SystemConfigTransactorSession) SetGasLimit(_gasLimit uint64) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetGasLimit(&_SystemConfig.TransactOpts, _gasLimit)
}

// SetResourceConfig is a paid mutator transaction binding the contract method 0xc71973f6.
//
// Solidity: function setResourceConfig((uint32,uint8,uint8,uint32,uint32,uint128) _config) returns()
func (_SystemConfig *SystemConfigTransactor) SetResourceConfig(opts *bind.TransactOpts, _config ResourceMeteringResourceConfig) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "setResourceConfig", _config)
}

// SetResourceConfig is a paid mutator transaction binding the contract method 0xc71973f6.
//
// Solidity: function setResourceConfig((uint32,uint8,uint8,uint32,uint32,uint128) _config) returns()
func (_SystemConfig *SystemConfigSession) SetResourceConfig(_config ResourceMeteringResourceConfig) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetResourceConfig(&_SystemConfig.TransactOpts, _config)
}

// SetResourceConfig is a paid mutator transaction binding the contract method 0xc71973f6.
//
// Solidity: function setResourceConfig((uint32,uint8,uint8,uint32,uint32,uint128) _config) returns()
func (_SystemConfig *SystemConfigTransactorSession) SetResourceConfig(_config ResourceMeteringResourceConfig) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetResourceConfig(&_SystemConfig.TransactOpts, _config)
}

// SetUnsafeBlockSigner is a paid mutator transaction binding the contract method 0x18d13918.
//
// Solidity: function setUnsafeBlockSigner(address _unsafeBlockSigner) returns()
func (_SystemConfig *SystemConfigTransactor) SetUnsafeBlockSigner(opts *bind.TransactOpts, _unsafeBlockSigner common.Address) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "setUnsafeBlockSigner", _unsafeBlockSigner)
}

// SetUnsafeBlockSigner is a paid mutator transaction binding the contract method 0x18d13918.
//
// Solidity: function setUnsafeBlockSigner(address _unsafeBlockSigner) returns()
func (_SystemConfig *SystemConfigSession) SetUnsafeBlockSigner(_unsafeBlockSigner common.Address) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetUnsafeBlockSigner(&_SystemConfig.TransactOpts, _unsafeBlockSigner)
}

// SetUnsafeBlockSigner is a paid mutator transaction binding the contract method 0x18d13918.
//
// Solidity: function setUnsafeBlockSigner(address _unsafeBlockSigner) returns()
func (_SystemConfig *SystemConfigTransactorSession) SetUnsafeBlockSigner(_unsafeBlockSigner common.Address) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetUnsafeBlockSigner(&_SystemConfig.TransactOpts, _unsafeBlockSigner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_SystemConfig *SystemConfigTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_SystemConfig *SystemConfigSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _SystemConfig.Contract.TransferOwnership(&_SystemConfig.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_SystemConfig *SystemConfigTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _SystemConfig.Contract.TransferOwnership(&_SystemConfig.TransactOpts, newOwner)
}

// SystemConfigConfigUpdateIterator is returned from FilterConfigUpdate and is used to iterate over the raw logs and unpacked data for ConfigUpdate events raised by the SystemConfig contract.
type SystemConfigConfigUpdateIterator struct {
	Event *SystemConfigConfigUpdate // Event containing the contract specifics and raw log

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
func (it *SystemConfigConfigUpdateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SystemConfigConfigUpdate)
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
		it.Event = new(SystemConfigConfigUpdate)
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
func (it *SystemConfigConfigUpdateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SystemConfigConfigUpdateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SystemConfigConfigUpdate represents a ConfigUpdate event raised by the SystemConfig contract.
type SystemConfigConfigUpdate struct {
	Version    *big.Int
	UpdateType uint8
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterConfigUpdate is a free log retrieval operation binding the contract event 0x1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be.
//
// Solidity: event ConfigUpdate(uint256 indexed version, uint8 indexed updateType, bytes data)
func (_SystemConfig *SystemConfigFilterer) FilterConfigUpdate(opts *bind.FilterOpts, version []*big.Int, updateType []uint8) (*SystemConfigConfigUpdateIterator, error) {

	var versionRule []interface{}
	for _, versionItem := range version {
		versionRule = append(versionRule, versionItem)
	}
	var updateTypeRule []interface{}
	for _, updateTypeItem := range updateType {
		updateTypeRule = append(updateTypeRule, updateTypeItem)
	}

	logs, sub, err := _SystemConfig.contract.FilterLogs(opts, "ConfigUpdate", versionRule, updateTypeRule)
	if err != nil {
		return nil, err
	}
	return &SystemConfigConfigUpdateIterator{contract: _SystemConfig.contract, event: "ConfigUpdate", logs: logs, sub: sub}, nil
}

// WatchConfigUpdate is a free log subscription operation binding the contract event 0x1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be.
//
// Solidity: event ConfigUpdate(uint256 indexed version, uint8 indexed updateType, bytes data)
func (_SystemConfig *SystemConfigFilterer) WatchConfigUpdate(opts *bind.WatchOpts, sink chan<- *SystemConfigConfigUpdate, version []*big.Int, updateType []uint8) (event.Subscription, error) {

	var versionRule []interface{}
	for _, versionItem := range version {
		versionRule = append(versionRule, versionItem)
	}
	var updateTypeRule []interface{}
	for _, updateTypeItem := range updateType {
		updateTypeRule = append(updateTypeRule, updateTypeItem)
	}

	logs, sub, err := _SystemConfig.contract.WatchLogs(opts, "ConfigUpdate", versionRule, updateTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SystemConfigConfigUpdate)
				if err := _SystemConfig.contract.UnpackLog(event, "ConfigUpdate", log); err != nil {
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

// ParseConfigUpdate is a log parse operation binding the contract event 0x1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be.
//
// Solidity: event ConfigUpdate(uint256 indexed version, uint8 indexed updateType, bytes data)
func (_SystemConfig *SystemConfigFilterer) ParseConfigUpdate(log types.Log) (*SystemConfigConfigUpdate, error) {
	event := new(SystemConfigConfigUpdate)
	if err := _SystemConfig.contract.UnpackLog(event, "ConfigUpdate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SystemConfigInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the SystemConfig contract.
type SystemConfigInitializedIterator struct {
	Event *SystemConfigInitialized // Event containing the contract specifics and raw log

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
func (it *SystemConfigInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SystemConfigInitialized)
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
		it.Event = new(SystemConfigInitialized)
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
func (it *SystemConfigInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SystemConfigInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SystemConfigInitialized represents a Initialized event raised by the SystemConfig contract.
type SystemConfigInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_SystemConfig *SystemConfigFilterer) FilterInitialized(opts *bind.FilterOpts) (*SystemConfigInitializedIterator, error) {

	logs, sub, err := _SystemConfig.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &SystemConfigInitializedIterator{contract: _SystemConfig.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_SystemConfig *SystemConfigFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *SystemConfigInitialized) (event.Subscription, error) {

	logs, sub, err := _SystemConfig.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SystemConfigInitialized)
				if err := _SystemConfig.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_SystemConfig *SystemConfigFilterer) ParseInitialized(log types.Log) (*SystemConfigInitialized, error) {
	event := new(SystemConfigInitialized)
	if err := _SystemConfig.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SystemConfigOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the SystemConfig contract.
type SystemConfigOwnershipTransferredIterator struct {
	Event *SystemConfigOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *SystemConfigOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SystemConfigOwnershipTransferred)
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
		it.Event = new(SystemConfigOwnershipTransferred)
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
func (it *SystemConfigOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SystemConfigOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SystemConfigOwnershipTransferred represents a OwnershipTransferred event raised by the SystemConfig contract.
type SystemConfigOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_SystemConfig *SystemConfigFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*SystemConfigOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _SystemConfig.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &SystemConfigOwnershipTransferredIterator{contract: _SystemConfig.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_SystemConfig *SystemConfigFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *SystemConfigOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _SystemConfig.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SystemConfigOwnershipTransferred)
				if err := _SystemConfig.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_SystemConfig *SystemConfigFilterer) ParseOwnershipTransferred(log types.Log) (*SystemConfigOwnershipTransferred, error) {
	event := new(SystemConfigOwnershipTransferred)
	if err := _SystemConfig.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
