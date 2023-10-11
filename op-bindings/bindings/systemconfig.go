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
	_ = abi.ConvertType
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
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"version\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"enumSystemConfig.UpdateType\",\"name\":\"updateType\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"ConfigUpdate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"BATCH_INBOX_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L1_CROSS_DOMAIN_MESSENGER_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L1_ERC_721_BRIDGE_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L1_STANDARD_BRIDGE_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_OUTPUT_ORACLE_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OPTIMISM_MINTABLE_ERC20_FACTORY_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OPTIMISM_PORTAL_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"UNSAFE_BLOCK_SIGNER_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"VERSION\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batchInbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batcherHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gasLimit\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_overhead\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_scalar\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_batcherHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"_gasLimit\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"_unsafeBlockSigner\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint32\",\"name\":\"maxResourceLimit\",\"type\":\"uint32\"},{\"internalType\":\"uint8\",\"name\":\"elasticityMultiplier\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"baseFeeMaxChangeDenominator\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"minimumBaseFee\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"systemTxMaxGas\",\"type\":\"uint32\"},{\"internalType\":\"uint128\",\"name\":\"maximumBaseFee\",\"type\":\"uint128\"}],\"internalType\":\"structResourceMetering.ResourceConfig\",\"name\":\"_config\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"_startBlock\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_batchInbox\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"l1CrossDomainMessenger\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1ERC721Bridge\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1StandardBridge\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l2OutputOracle\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"optimismPortal\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"optimismMintableERC20Factory\",\"type\":\"address\"}],\"internalType\":\"structSystemConfig.Addresses\",\"name\":\"_addresses\",\"type\":\"tuple\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1CrossDomainMessenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1ERC721Bridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1StandardBridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2OutputOracle\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minimumGasLimit\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"optimismMintableERC20Factory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"optimismPortal\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"overhead\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"resourceConfig\",\"outputs\":[{\"components\":[{\"internalType\":\"uint32\",\"name\":\"maxResourceLimit\",\"type\":\"uint32\"},{\"internalType\":\"uint8\",\"name\":\"elasticityMultiplier\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"baseFeeMaxChangeDenominator\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"minimumBaseFee\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"systemTxMaxGas\",\"type\":\"uint32\"},{\"internalType\":\"uint128\",\"name\":\"maximumBaseFee\",\"type\":\"uint128\"}],\"internalType\":\"structResourceMetering.ResourceConfig\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"scalar\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_batcherHash\",\"type\":\"bytes32\"}],\"name\":\"setBatcherHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_overhead\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_scalar\",\"type\":\"uint256\"}],\"name\":\"setGasConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_gasLimit\",\"type\":\"uint64\"}],\"name\":\"setGasLimit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint32\",\"name\":\"maxResourceLimit\",\"type\":\"uint32\"},{\"internalType\":\"uint8\",\"name\":\"elasticityMultiplier\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"baseFeeMaxChangeDenominator\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"minimumBaseFee\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"systemTxMaxGas\",\"type\":\"uint32\"},{\"internalType\":\"uint128\",\"name\":\"maximumBaseFee\",\"type\":\"uint128\"}],\"internalType\":\"structResourceMetering.ResourceConfig\",\"name\":\"_config\",\"type\":\"tuple\"}],\"name\":\"setResourceConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_unsafeBlockSigner\",\"type\":\"address\"}],\"name\":\"setUnsafeBlockSigner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unsafeBlockSigner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b506040805160c080820183526001808352602080840182905260028486015260006060808601829052608080870183905260a08088018490528851968701895283875293860183905296850182905284018190529483018590528201849052620000909361dead93909283928392909183919060001990839062000096565b62000ccc565b600054600390610100900460ff16158015620000b9575060005460ff8083169116105b620001225760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805461ffff191660ff831617610100179055620001406200043a565b6200014b8b620004a2565b620001568862000521565b620001628a8a62000573565b6200016d87620005d7565b620001788662000674565b620001be620001a960017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc59862000ba9565b60001b84620006dc60201b62000d211760201c565b62000208620001ef60017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce958063762000ba9565b60001b8360000151620006dc60201b62000d211760201c565b620002526200023960017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a862000ba9565b60001b8360200151620006dc60201b62000d211760201c565b6200029c6200028360017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad637762000ba9565b60001b8360400151620006dc60201b62000d211760201c565b620002e6620002cd60017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a687181662000ba9565b60001b8360600151620006dc60201b62000d211760201c565b620003306200031760017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad62000ba9565b60001b8360800151620006dc60201b62000d211760201c565b6200037a6200036160017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d62000ba9565b60001b8360a00151620006dc60201b62000d211760201c565b6200038584620006e0565b62000390856200070d565b6200039a62000a51565b6001600160401b0316876001600160401b03161015620003ec5760405162461bcd60e51b815260206004820152601f60248201526000805160206200281f833981519152604482015260640162000119565b6000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050505050505050505050565b600054610100900460ff16620004965760405162461bcd60e51b815260206004820152602b60248201526000805160206200285f83398151915260448201526a6e697469616c697a696e6760a81b606482015260840162000119565b620004a062000a7e565b565b620004ac62000ae5565b6001600160a01b038116620005135760405162461bcd60e51b815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201526564647265737360d01b606482015260840162000119565b6200051e8162000b41565b50565b60678190556040805160208082018490528251808303909101815290820190915260005b60006000805160206200283f8339815191528360405162000567919062000bc3565b60405180910390a35050565b60658290556066819055604080516020810184905290810182905260009060600160408051601f198184030181529190529050600160006000805160206200283f83398151915283604051620005ca919062000bc3565b60405180910390a3505050565b620005e162000a51565b6001600160401b0316816001600160401b03161015620006335760405162461bcd60e51b815260206004820152601f60248201526000805160206200281f833981519152604482015260640162000119565b606880546001600160401b0319166001600160401b038316908117909155604080516020808201939093528151808203909301835281019052600262000545565b620006ab7f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c0882620006dc60201b62000d211760201c565b604080516001600160a01b03831660208201526000910160408051601f198184030181529190529050600362000545565b9055565b8015801590620006f05750606a54155b15620006fb57606a55565b606a546000036200051e5743606a5550565b8060a001516001600160801b0316816060015163ffffffff1611156200079c5760405162461bcd60e51b815260206004820152603560248201527f53797374656d436f6e6669673a206d696e206261736520666565206d7573742060448201527f6265206c657373207468616e206d617820626173650000000000000000000000606482015260840162000119565b6001816040015160ff16116200080d5760405162461bcd60e51b815260206004820152602f60248201527f53797374656d436f6e6669673a2064656e6f6d696e61746f72206d757374206260448201526e65206c6172676572207468616e203160881b606482015260840162000119565b606854608082015182516001600160401b03909216916200082f919062000c1b565b63ffffffff161115620008745760405162461bcd60e51b815260206004820152601f60248201526000805160206200281f833981519152604482015260640162000119565b6000816020015160ff1611620008e55760405162461bcd60e51b815260206004820152602f60248201527f53797374656d436f6e6669673a20656c6173746963697479206d756c7469706c60448201526e06965722063616e6e6f74206265203608c1b606482015260840162000119565b8051602082015163ffffffff82169160ff909116906200090790829062000c46565b62000913919062000c78565b63ffffffff16146200098e5760405162461bcd60e51b815260206004820152603760248201527f53797374656d436f6e6669673a20707265636973696f6e206c6f73732077697460448201527f6820746172676574207265736f75726365206c696d6974000000000000000000606482015260840162000119565b805160698054602084015160408501516060860151608087015160a09097015163ffffffff96871664ffffffffff199095169490941764010000000060ff948516021764ffffffffff60281b191665010000000000939092169290920263ffffffff60301b19161766010000000000009185169190910217600160501b600160f01b0319166a01000000000000000000009390941692909202600160701b600160f01b03191692909217600160701b6001600160801b0390921691909102179055565b60695460009062000a799063ffffffff6a010000000000000000000082048116911662000ca7565b905090565b600054610100900460ff1662000ada5760405162461bcd60e51b815260206004820152602b60248201526000805160206200285f83398151915260448201526a6e697469616c697a696e6760a81b606482015260840162000119565b620004a03362000b41565b6033546001600160a01b03163314620004a05760405162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640162000119565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b634e487b7160e01b600052601160045260246000fd5b60008282101562000bbe5762000bbe62000b93565b500390565b600060208083528351808285015260005b8181101562000bf25785810183015185820160400152820162000bd4565b8181111562000c05576000604083870101525b50601f01601f1916929092016040019392505050565b600063ffffffff80831681851680830382111562000c3d5762000c3d62000b93565b01949350505050565b600063ffffffff8084168062000c6c57634e487b7160e01b600052601260045260246000fd5b92169190910492915050565b600063ffffffff8083168185168183048111821515161562000c9e5762000c9e62000b93565b02949350505050565b60006001600160401b0382811684821680830382111562000c3d5762000c3d62000b93565b611b438062000cdc6000396000f3fe608060405234801561001057600080fd5b50600436106102265760003560e01c8063935f029e1161012a578063cc731b02116100bd578063f45e65d81161008c578063f8c68de011610071578063f8c68de014610575578063fd32aa0f1461057d578063ffa1ad741461058557600080fd5b8063f45e65d814610558578063f68016b71461056157600080fd5b8063cc731b0214610400578063dac6e63a14610534578063e81b2c6d1461053c578063f2fde38b1461054557600080fd5b8063bc49ce5f116100f9578063bc49ce5f146103ca578063c4e8ddfa146103d2578063c71973f6146103da578063c9b26f61146103ed57600080fd5b8063935f029e146103945780639b7d7f0a146103a7578063a7119869146103af578063b40a817c146103b757600080fd5b80634add321d116101bd57806354fd4d501161018c57806361d157681161017157806361d1576814610366578063715018a61461036e5780638da5cb5b1461037657600080fd5b806354fd4d50146103155780635d73369c1461035e57600080fd5b80634add321d146102b25780634d9f1559146102d35780634f16540b146102db5780635228a6ac1461030257600080fd5b806318d13918116101f957806318d139181461028457806319f5cea8146102995780631fd19ee1146102a157806348cd4cb1146102a957600080fd5b806306c926571461022b578063078f29cf146102465780630a49cb03146102735780630c18c1621461027b575b600080fd5b61023361058d565b6040519081526020015b60405180910390f35b61024e6105bb565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200161023d565b61024e6105f4565b61023360655481565b610297610292366004611677565b610624565b005b610233610638565b61024e610663565b610233606a5481565b6102ba61068d565b60405167ffffffffffffffff909116815260200161023d565b61024e6106b3565b6102337f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c0881565b610297610310366004611807565b6106e3565b6103516040518060400160405280600681526020017f312e31302e30000000000000000000000000000000000000000000000000000081525081565b60405161023d91906119aa565b610233610a79565b610233610aa4565b610297610acf565b60335473ffffffffffffffffffffffffffffffffffffffff1661024e565b6102976103a23660046119bd565b610ae3565b61024e610af9565b61024e610b29565b6102976103c53660046119df565b610b59565b610233610b6a565b61024e610b95565b6102976103e83660046119fa565b610bc5565b6102976103fb366004611a16565b610bd6565b6104c46040805160c081018252600080825260208201819052918101829052606081018290526080810182905260a0810191909152506040805160c08101825260695463ffffffff8082168352640100000000820460ff9081166020850152650100000000008304169383019390935266010000000000008104831660608301526a0100000000000000000000810490921660808201526e0100000000000000000000000000009091046fffffffffffffffffffffffffffffffff1660a082015290565b60405161023d9190600060c08201905063ffffffff80845116835260ff602085015116602084015260ff6040850151166040840152806060850151166060840152806080850151166080840152506fffffffffffffffffffffffffffffffff60a08401511660a083015292915050565b61024e610be7565b61023360675481565b610297610553366004611677565b610c17565b61023360665481565b6068546102ba9067ffffffffffffffff1681565b610233610ccb565b610233610cf6565b610233600081565b6105b860017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d611a5e565b81565b60006105ef6105eb60017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad6377611a5e565b5490565b905090565b60006105ef6105eb60017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad611a5e565b61062c610d25565b61063581610da6565b50565b6105b860017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a8611a5e565b60006105ef7f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c085490565b6069546000906105ef9063ffffffff6a0100000000000000000000820481169116611a75565b60006105ef6105eb60017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a6871816611a5e565b600054600390610100900460ff16158015610705575060005460ff8083169116105b610796576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084015b60405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff8316176101001790556107cf610e63565b6107d88b610c17565b6107e188610f02565b6107eb8a8a610f2a565b6107f487610fbb565b6107fd86610da6565b61083061082b60017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc598611a5e565b849055565b61086461085e60017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce9580637611a5e565b83519055565b61089b61089260017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a8611a5e565b60208401519055565b6108d26108c960017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad6377611a5e565b60408401519055565b61090961090060017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a6871816611a5e565b60608401519055565b61094061093760017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad611a5e565b60808401519055565b61097761096e60017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d611a5e565b60a08401519055565b61098084611099565b610989856110c3565b61099161068d565b67ffffffffffffffff168767ffffffffffffffff161015610a0e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f7700604482015260640161078d565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050505050505050505050565b6105b860017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce9580637611a5e565b6105b860017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a6871816611a5e565b610ad7610d25565b610ae16000611537565b565b610aeb610d25565b610af58282610f2a565b5050565b60006105ef6105eb60017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d611a5e565b60006105ef6105eb60017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce9580637611a5e565b610b61610d25565b61063581610fbb565b6105b860017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc598611a5e565b60006105ef6105eb60017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a8611a5e565b610bcd610d25565b610635816110c3565b610bde610d25565b61063581610f02565b60006105ef6105eb60017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc598611a5e565b610c1f610d25565b73ffffffffffffffffffffffffffffffffffffffff8116610cc2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015260840161078d565b61063581611537565b6105b860017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad6377611a5e565b6105b860017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad611a5e565b9055565b60335473ffffffffffffffffffffffffffffffffffffffff163314610ae1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161078d565b610dcf7f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08829055565b6040805173ffffffffffffffffffffffffffffffffffffffff8316602082015260009101604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0818403018152919052905060035b60007f1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be83604051610e5791906119aa565b60405180910390a35050565b600054610100900460ff16610efa576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161078d565b610ae16115ae565b6067819055604080516020808201849052825180830390910181529082019091526000610e26565b606582905560668190556040805160208101849052908101829052600090606001604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190529050600160007f1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be83604051610fae91906119aa565b60405180910390a3505050565b610fc361068d565b67ffffffffffffffff168167ffffffffffffffff161015611040576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f7700604482015260640161078d565b606880547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff83169081179091556040805160208082019390935281518082039093018352810190526002610e26565b80158015906110a85750606a54155b156110b257606a55565b606a546000036106355743606a5550565b8060a001516fffffffffffffffffffffffffffffffff16816060015163ffffffff161115611173576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603560248201527f53797374656d436f6e6669673a206d696e206261736520666565206d7573742060448201527f6265206c657373207468616e206d617820626173650000000000000000000000606482015260840161078d565b6001816040015160ff161161120a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f53797374656d436f6e6669673a2064656e6f6d696e61746f72206d757374206260448201527f65206c6172676572207468616e20310000000000000000000000000000000000606482015260840161078d565b6068546080820151825167ffffffffffffffff9092169161122b9190611aa1565b63ffffffff161115611299576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f7700604482015260640161078d565b6000816020015160ff1611611330576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f53797374656d436f6e6669673a20656c6173746963697479206d756c7469706c60448201527f6965722063616e6e6f7420626520300000000000000000000000000000000000606482015260840161078d565b8051602082015163ffffffff82169160ff90911690611350908290611ac0565b61135a9190611b0a565b63ffffffff16146113ed576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f53797374656d436f6e6669673a20707265636973696f6e206c6f73732077697460448201527f6820746172676574207265736f75726365206c696d6974000000000000000000606482015260840161078d565b805160698054602084015160408501516060860151608087015160a09097015163ffffffff9687167fffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000009095169490941764010000000060ff94851602177fffffffffffffffffffffffffffffffffffffffffffff0000000000ffffffffff166501000000000093909216929092027fffffffffffffffffffffffffffffffffffffffffffff00000000ffffffffffff1617660100000000000091851691909102177fffff0000000000000000000000000000000000000000ffffffffffffffffffff166a010000000000000000000093909416929092027fffff00000000000000000000000000000000ffffffffffffffffffffffffffff16929092176e0100000000000000000000000000006fffffffffffffffffffffffffffffffff90921691909102179055565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600054610100900460ff16611645576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161078d565b610ae133611537565b803573ffffffffffffffffffffffffffffffffffffffff8116811461167257600080fd5b919050565b60006020828403121561168957600080fd5b6116928261164e565b9392505050565b803567ffffffffffffffff8116811461167257600080fd5b60405160c0810167ffffffffffffffff811182821017156116fb577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60405290565b803563ffffffff8116811461167257600080fd5b803560ff8116811461167257600080fd5b600060c0828403121561173857600080fd5b60405160c0810181811067ffffffffffffffff82111715611782577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60405290508061179183611701565b815261179f60208401611715565b60208201526117b060408401611715565b60408201526117c160608401611701565b60608201526117d260808401611701565b608082015260a08301356fffffffffffffffffffffffffffffffff811681146117fa57600080fd5b60a0919091015292915050565b6000806000806000806000806000808a8c0361028081121561182857600080fd5b6118318c61164e565b9a5060208c0135995060408c0135985060608c0135975061185460808d01611699565b965061186260a08d0161164e565b95506118718d60c08e01611726565b94506101808c013593506118886101a08d0161164e565b925060c07ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe40820112156118ba57600080fd5b506118c36116b1565b6118d06101c08d0161164e565b81526118df6101e08d0161164e565b60208201526118f16102008d0161164e565b60408201526119036102208d0161164e565b60608201526119156102408d0161164e565b60808201526119276102608d0161164e565b60a0820152809150509295989b9194979a5092959850565b6000815180845260005b8181101561196557602081850181015186830182015201611949565b81811115611977576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000611692602083018461193f565b600080604083850312156119d057600080fd5b50508035926020909101359150565b6000602082840312156119f157600080fd5b61169282611699565b600060c08284031215611a0c57600080fd5b6116928383611726565b600060208284031215611a2857600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015611a7057611a70611a2f565b500390565b600067ffffffffffffffff808316818516808303821115611a9857611a98611a2f565b01949350505050565b600063ffffffff808316818516808303821115611a9857611a98611a2f565b600063ffffffff80841680611afe577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b92169190910492915050565b600063ffffffff80831681851681830481118215151615611b2d57611b2d611a2f565b0294935050505056fea164736f6c634300080f000a53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f77001d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be496e697469616c697a61626c653a20636f6e7472616374206973206e6f742069",
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
	parsed, err := SystemConfigMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
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
// Solidity: function startBlock() view returns(uint256)
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
// Solidity: function startBlock() view returns(uint256)
func (_SystemConfig *SystemConfigSession) StartBlock() (*big.Int, error) {
	return _SystemConfig.Contract.StartBlock(&_SystemConfig.CallOpts)
}

// StartBlock is a free data retrieval call binding the contract method 0x48cd4cb1.
//
// Solidity: function startBlock() view returns(uint256)
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

// Initialize is a paid mutator transaction binding the contract method 0x5228a6ac.
//
// Solidity: function initialize(address _owner, uint256 _overhead, uint256 _scalar, bytes32 _batcherHash, uint64 _gasLimit, address _unsafeBlockSigner, (uint32,uint8,uint8,uint32,uint32,uint128) _config, uint256 _startBlock, address _batchInbox, (address,address,address,address,address,address) _addresses) returns()
func (_SystemConfig *SystemConfigTransactor) Initialize(opts *bind.TransactOpts, _owner common.Address, _overhead *big.Int, _scalar *big.Int, _batcherHash [32]byte, _gasLimit uint64, _unsafeBlockSigner common.Address, _config ResourceMeteringResourceConfig, _startBlock *big.Int, _batchInbox common.Address, _addresses SystemConfigAddresses) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "initialize", _owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config, _startBlock, _batchInbox, _addresses)
}

// Initialize is a paid mutator transaction binding the contract method 0x5228a6ac.
//
// Solidity: function initialize(address _owner, uint256 _overhead, uint256 _scalar, bytes32 _batcherHash, uint64 _gasLimit, address _unsafeBlockSigner, (uint32,uint8,uint8,uint32,uint32,uint128) _config, uint256 _startBlock, address _batchInbox, (address,address,address,address,address,address) _addresses) returns()
func (_SystemConfig *SystemConfigSession) Initialize(_owner common.Address, _overhead *big.Int, _scalar *big.Int, _batcherHash [32]byte, _gasLimit uint64, _unsafeBlockSigner common.Address, _config ResourceMeteringResourceConfig, _startBlock *big.Int, _batchInbox common.Address, _addresses SystemConfigAddresses) (*types.Transaction, error) {
	return _SystemConfig.Contract.Initialize(&_SystemConfig.TransactOpts, _owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config, _startBlock, _batchInbox, _addresses)
}

// Initialize is a paid mutator transaction binding the contract method 0x5228a6ac.
//
// Solidity: function initialize(address _owner, uint256 _overhead, uint256 _scalar, bytes32 _batcherHash, uint64 _gasLimit, address _unsafeBlockSigner, (uint32,uint8,uint8,uint32,uint32,uint128) _config, uint256 _startBlock, address _batchInbox, (address,address,address,address,address,address) _addresses) returns()
func (_SystemConfig *SystemConfigTransactorSession) Initialize(_owner common.Address, _overhead *big.Int, _scalar *big.Int, _batcherHash [32]byte, _gasLimit uint64, _unsafeBlockSigner common.Address, _config ResourceMeteringResourceConfig, _startBlock *big.Int, _batchInbox common.Address, _addresses SystemConfigAddresses) (*types.Transaction, error) {
	return _SystemConfig.Contract.Initialize(&_SystemConfig.TransactOpts, _owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config, _startBlock, _batchInbox, _addresses)
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
