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

// SystemConfigMetaData contains all meta data concerning the SystemConfig contract.
var SystemConfigMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_overhead\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_scalar\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_batcherHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_gasLimit\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"_unsafeBlockSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_config\",\"type\":\"tuple\",\"internalType\":\"structResourceMetering.ResourceConfig\",\"components\":[{\"name\":\"maxResourceLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"elasticityMultiplier\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"baseFeeMaxChangeDenominator\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"minimumBaseFee\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"systemTxMaxGas\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"maximumBaseFee\",\"type\":\"uint128\",\"internalType\":\"uint128\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"UNSAFE_BLOCK_SIGNER_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"VERSION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"basefeeScalar\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"batcherHash\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"blobBasefeeScalar\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gasLimit\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_overhead\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_scalar\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_batcherHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_gasLimit\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"_unsafeBlockSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_config\",\"type\":\"tuple\",\"internalType\":\"structResourceMetering.ResourceConfig\",\"components\":[{\"name\":\"maxResourceLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"elasticityMultiplier\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"baseFeeMaxChangeDenominator\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"minimumBaseFee\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"systemTxMaxGas\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"maximumBaseFee\",\"type\":\"uint128\",\"internalType\":\"uint128\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"minimumGasLimit\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"overhead\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resourceConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structResourceMetering.ResourceConfig\",\"components\":[{\"name\":\"maxResourceLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"elasticityMultiplier\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"baseFeeMaxChangeDenominator\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"minimumBaseFee\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"systemTxMaxGas\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"maximumBaseFee\",\"type\":\"uint128\",\"internalType\":\"uint128\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"scalar\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setBatcherHash\",\"inputs\":[{\"name\":\"_batcherHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setGasConfig\",\"inputs\":[{\"name\":\"_overhead\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_scalar\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setGasConfigEcotone\",\"inputs\":[{\"name\":\"_basefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_blobBasefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setGasLimit\",\"inputs\":[{\"name\":\"_gasLimit\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setResourceConfig\",\"inputs\":[{\"name\":\"_config\",\"type\":\"tuple\",\"internalType\":\"structResourceMetering.ResourceConfig\",\"components\":[{\"name\":\"maxResourceLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"elasticityMultiplier\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"baseFeeMaxChangeDenominator\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"minimumBaseFee\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"systemTxMaxGas\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"maximumBaseFee\",\"type\":\"uint128\",\"internalType\":\"uint128\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setUnsafeBlockSigner\",\"inputs\":[{\"name\":\"_unsafeBlockSigner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unsafeBlockSigner\",\"inputs\":[],\"outputs\":[{\"name\":\"addr_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"ConfigUpdate\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"updateType\",\"type\":\"uint8\",\"indexed\":true,\"internalType\":\"enumSystemConfig.UpdateType\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false}]",
	Bin: "0x60806040523480156200001157600080fd5b50604051620023a5380380620023a5833981016040819052620000349162000a51565b620000458787878787878762000052565b5050505050505062000ca9565b600054610100900460ff1615808015620000735750600054600160ff909116105b80620000a3575062000090306200023760201b620008c51760201c565b158015620000a3575060005460ff166001145b6200010c5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805460ff19166001179055801562000130576000805461ff0019166101001790555b6200013a62000246565b6200014588620002ae565b62000150856200032d565b6200015c87876200037f565b62000169866000620003dd565b620001748462000466565b6200017f8362000503565b6200018a826200056b565b62000194620008af565b6001600160401b0316846001600160401b03161015620001e65760405162461bcd60e51b815260206004820152601f602482015260008051602062002345833981519152604482015260640162000103565b80156200022d576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050505050505050565b6001600160a01b03163b151590565b600054610100900460ff16620002a25760405162461bcd60e51b815260206004820152602b60248201526000805160206200238583398151915260448201526a6e697469616c697a696e6760a81b606482015260840162000103565b620002ac620008dc565b565b620002b862000943565b6001600160a01b0381166200031f5760405162461bcd60e51b815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201526564647265737360d01b606482015260840162000103565b6200032a816200099f565b50565b60678190556040805160208082018490528251808303909101815290820190915260005b6000600080516020620023658339815191528360405162000373919062000b8a565b60405180910390a35050565b606582905560668190556040805160208101849052808201839052815180820383018152606090910190915260015b60006000805160206200236583398151915283604051620003d0919062000b8a565b60405180910390a3505050565b60688054600160401b600160801b0319166801000000000000000063ffffffff8581169190910263ffffffff60601b1916919091176c0100000000000000000000000091841691909102179055604080516001600160e01b031960e085811b8216602084015284901b1660248201528151808203600801815260289091019091526004620003ae565b62000470620008af565b6001600160401b0316816001600160401b03161015620004c25760405162461bcd60e51b815260206004820152601f602482015260008051602062002345833981519152604482015260640162000103565b606880546001600160401b0319166001600160401b038316908117909155604080516020808201939093528151808203909301835281019052600262000351565b6200053a7f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c0882620009f160201b620008e11760201c565b604080516001600160a01b03831660208201526000910160408051601f198184030181529190529050600362000351565b8060a001516001600160801b0316816060015163ffffffff161115620005fa5760405162461bcd60e51b815260206004820152603560248201527f53797374656d436f6e6669673a206d696e206261736520666565206d7573742060448201527f6265206c657373207468616e206d617820626173650000000000000000000000606482015260840162000103565b6001816040015160ff16116200066b5760405162461bcd60e51b815260206004820152602f60248201527f53797374656d436f6e6669673a2064656e6f6d696e61746f72206d757374206260448201526e65206c6172676572207468616e203160881b606482015260840162000103565b606854608082015182516001600160401b03909216916200068d919062000bf8565b63ffffffff161115620006d25760405162461bcd60e51b815260206004820152601f602482015260008051602062002345833981519152604482015260640162000103565b6000816020015160ff1611620007435760405162461bcd60e51b815260206004820152602f60248201527f53797374656d436f6e6669673a20656c6173746963697479206d756c7469706c60448201526e06965722063616e6e6f74206265203608c1b606482015260840162000103565b8051602082015163ffffffff82169160ff909116906200076590829062000c23565b62000771919062000c55565b63ffffffff1614620007ec5760405162461bcd60e51b815260206004820152603760248201527f53797374656d436f6e6669673a20707265636973696f6e206c6f73732077697460448201527f6820746172676574207265736f75726365206c696d6974000000000000000000606482015260840162000103565b805160698054602084015160408501516060860151608087015160a09097015163ffffffff96871664ffffffffff199095169490941764010000000060ff948516021764ffffffffff60281b191665010000000000939092169290920263ffffffff60301b19161766010000000000009185169190910217600160501b600160f01b0319166a01000000000000000000009390941692909202600160701b600160f01b03191692909217600160701b6001600160801b0390921691909102179055565b606954600090620008d79063ffffffff6a010000000000000000000082048116911662000c84565b905090565b600054610100900460ff16620009385760405162461bcd60e51b815260206004820152602b60248201526000805160206200238583398151915260448201526a6e697469616c697a696e6760a81b606482015260840162000103565b620002ac336200099f565b6033546001600160a01b03163314620002ac5760405162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640162000103565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b9055565b80516001600160a01b038116811462000a0d57600080fd5b919050565b805163ffffffff8116811462000a0d57600080fd5b805160ff8116811462000a0d57600080fd5b80516001600160801b038116811462000a0d57600080fd5b600080600080600080600087890361018081121562000a6f57600080fd5b62000a7a89620009f5565b60208a015160408b015160608c015160808d0151939b50919950975095506001600160401b03808216821462000aaf57600080fd5b81955062000ac060a08c01620009f5565b945060c060bf198401121562000ad557600080fd5b604051925060c083019150828210818311171562000b0357634e487b7160e01b600052604160045260246000fd5b5060405262000b1560c08a0162000a12565b815262000b2560e08a0162000a27565b602082015262000b396101008a0162000a27565b604082015262000b4d6101208a0162000a12565b606082015262000b616101408a0162000a12565b608082015262000b756101608a0162000a39565b60a08201528091505092959891949750929550565b600060208083528351808285015260005b8181101562000bb95785810183015185820160400152820162000b9b565b8181111562000bcc576000604083870101525b50601f01601f1916929092016040019392505050565b634e487b7160e01b600052601160045260246000fd5b600063ffffffff80831681851680830382111562000c1a5762000c1a62000be2565b01949350505050565b600063ffffffff8084168062000c4957634e487b7160e01b600052601260045260246000fd5b92169190910492915050565b600063ffffffff8083168185168183048111821515161562000c7b5762000c7b62000be2565b02949350505050565b60006001600160401b0382811684821680830382111562000c1a5762000c1a62000be2565b61168c8062000cb96000396000f3fe608060405234801561001057600080fd5b50600436106101825760003560e01c8063b40a817c116100d8578063e81b2c6d1161008c578063f68016b711610066578063f68016b7146104a5578063f975e925146104b9578063ffa1ad74146104cc57600080fd5b8063e81b2c6d14610480578063f2fde38b14610489578063f45e65d81461049c57600080fd5b8063c71973f6116100bd578063c71973f614610326578063c9b26f6114610339578063cc731b021461034c57600080fd5b8063b40a817c146102f7578063bfb14fb71461030a57600080fd5b80634f16540b1161013a5780637744864d116101145780637744864d146102915780638da5cb5b146102c6578063935f029e146102e457600080fd5b80634f16540b1461021957806354fd4d5014610240578063715018a61461028957600080fd5b80631fd19ee11161016b5780631fd19ee1146101b857806321d7fde5146101e55780634add321d146101f857600080fd5b80630c18c1621461018757806318d13918146101a3575b600080fd5b61019060655481565b6040519081526020015b60405180910390f35b6101b66101b13660046112b1565b6104d4565b005b6101c06104e8565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200161019a565b6101b66101f33660046112e7565b610517565b61020061052d565b60405167ffffffffffffffff909116815260200161019a565b6101907f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c0881565b61027c6040518060400160405280600681526020017f312e31322e30000000000000000000000000000000000000000000000000000081525081565b60405161019a9190611385565b6101b6610553565b6068546102b1906c01000000000000000000000000900463ffffffff1681565b60405163ffffffff909116815260200161019a565b60335473ffffffffffffffffffffffffffffffffffffffff166101c0565b6101b66102f2366004611398565b610567565b6101b66103053660046113d2565b610579565b6068546102b19068010000000000000000900463ffffffff1681565b6101b66103343660046114e7565b61058a565b6101b6610347366004611503565b61059b565b6104106040805160c081018252600080825260208201819052918101829052606081018290526080810182905260a0810191909152506040805160c08101825260695463ffffffff8082168352640100000000820460ff9081166020850152650100000000008304169383019390935266010000000000008104831660608301526a0100000000000000000000810490921660808201526e0100000000000000000000000000009091046fffffffffffffffffffffffffffffffff1660a082015290565b60405161019a9190600060c08201905063ffffffff80845116835260ff602085015116602084015260ff6040850151166040840152806060850151166060840152806080850151166080840152506fffffffffffffffffffffffffffffffff60a08401511660a083015292915050565b61019060675481565b6101b66104973660046112b1565b6105ac565b61019060665481565b6068546102009067ffffffffffffffff1681565b6101b66104c736600461151c565b610665565b610190600081565b6104dc6108e5565b6104e581610966565b50565b60006105127f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c085490565b905090565b61051f6108e5565b6105298282610a23565b5050565b6069546000906105129063ffffffff6a01000000000000000000008204811691166115be565b61055b6108e5565b6105656000610b2a565b565b61056f6108e5565b6105298282610ba1565b6105816108e5565b6104e581610bd4565b6105926108e5565b6104e581610cb2565b6105a36108e5565b6104e581611126565b6105b46108e5565b73ffffffffffffffffffffffffffffffffffffffff811661065c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b6104e581610b2a565b600054610100900460ff16158080156106855750600054600160ff909116105b8061069f5750303b15801561069f575060005460ff166001145b61072b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a65640000000000000000000000000000000000006064820152608401610653565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001179055801561078957600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b61079161114e565b61079a886105ac565b6107a385611126565b6107ad8787610ba1565b6107b8866000610a23565b6107c184610bd4565b6107ca83610966565b6107d382610cb2565b6107db61052d565b67ffffffffffffffff168467ffffffffffffffff161015610858576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f77006044820152606401610653565b80156108bb57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050505050505050565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b9055565b60335473ffffffffffffffffffffffffffffffffffffffff163314610565576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401610653565b61098f7f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08829055565b6040805173ffffffffffffffffffffffffffffffffffffffff8316602082015260009101604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0818403018152919052905060035b60007f1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be83604051610a179190611385565b60405180910390a35050565b606880547fffffffffffffffffffffffffffffffff0000000000000000ffffffffffffffff166801000000000000000063ffffffff858116919091027fffffffffffffffffffffffffffffffff00000000ffffffffffffffffffffffff16919091176c0100000000000000000000000091841691909102179055604080517fffffffff0000000000000000000000000000000000000000000000000000000060e085811b8216602084015284901b16602482015281518082036008018152602890910190915260045b60007f1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be83604051610b1d9190611385565b60405180910390a3505050565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b60658290556066819055604080516020810184905280820183905281518082038301815260609091019091526001610aec565b610bdc61052d565b67ffffffffffffffff168167ffffffffffffffff161015610c59576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f77006044820152606401610653565b606880547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff831690811790915560408051602080820193909352815180820390930183528101905260026109e6565b8060a001516fffffffffffffffffffffffffffffffff16816060015163ffffffff161115610d62576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603560248201527f53797374656d436f6e6669673a206d696e206261736520666565206d7573742060448201527f6265206c657373207468616e206d6178206261736500000000000000000000006064820152608401610653565b6001816040015160ff1611610df9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f53797374656d436f6e6669673a2064656e6f6d696e61746f72206d757374206260448201527f65206c6172676572207468616e203100000000000000000000000000000000006064820152608401610653565b6068546080820151825167ffffffffffffffff90921691610e1a91906115ea565b63ffffffff161115610e88576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f77006044820152606401610653565b6000816020015160ff1611610f1f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f53797374656d436f6e6669673a20656c6173746963697479206d756c7469706c60448201527f6965722063616e6e6f74206265203000000000000000000000000000000000006064820152608401610653565b8051602082015163ffffffff82169160ff90911690610f3f908290611609565b610f499190611653565b63ffffffff1614610fdc576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f53797374656d436f6e6669673a20707265636973696f6e206c6f73732077697460448201527f6820746172676574207265736f75726365206c696d69740000000000000000006064820152608401610653565b805160698054602084015160408501516060860151608087015160a09097015163ffffffff9687167fffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000009095169490941764010000000060ff94851602177fffffffffffffffffffffffffffffffffffffffffffff0000000000ffffffffff166501000000000093909216929092027fffffffffffffffffffffffffffffffffffffffffffff00000000ffffffffffff1617660100000000000091851691909102177fffff0000000000000000000000000000000000000000ffffffffffffffffffff166a010000000000000000000093909416929092027fffff00000000000000000000000000000000ffffffffffffffffffffffffffff16929092176e0100000000000000000000000000006fffffffffffffffffffffffffffffffff90921691909102179055565b60678190556040805160208082018490528251808303909101815290820190915260006109e6565b600054610100900460ff166111e5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e670000000000000000000000000000000000000000006064820152608401610653565b610565600054610100900460ff1661127f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e670000000000000000000000000000000000000000006064820152608401610653565b61056533610b2a565b803573ffffffffffffffffffffffffffffffffffffffff811681146112ac57600080fd5b919050565b6000602082840312156112c357600080fd5b6112cc82611288565b9392505050565b803563ffffffff811681146112ac57600080fd5b600080604083850312156112fa57600080fd5b611303836112d3565b9150611311602084016112d3565b90509250929050565b6000815180845260005b8181101561134057602081850181015186830182015201611324565b81811115611352576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006112cc602083018461131a565b600080604083850312156113ab57600080fd5b50508035926020909101359150565b803567ffffffffffffffff811681146112ac57600080fd5b6000602082840312156113e457600080fd5b6112cc826113ba565b803560ff811681146112ac57600080fd5b80356fffffffffffffffffffffffffffffffff811681146112ac57600080fd5b600060c0828403121561143057600080fd5b60405160c0810181811067ffffffffffffffff8211171561147a577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604052905080611489836112d3565b8152611497602084016113ed565b60208201526114a8604084016113ed565b60408201526114b9606084016112d3565b60608201526114ca608084016112d3565b60808201526114db60a084016113fe565b60a08201525092915050565b600060c082840312156114f957600080fd5b6112cc838361141e565b60006020828403121561151557600080fd5b5035919050565b6000806000806000806000610180888a03121561153857600080fd5b61154188611288565b9650602088013595506040880135945060608801359350611564608089016113ba565b925061157260a08901611288565b91506115818960c08a0161141e565b905092959891949750929550565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600067ffffffffffffffff8083168185168083038211156115e1576115e161158f565b01949350505050565b600063ffffffff8083168185168083038211156115e1576115e161158f565b600063ffffffff80841680611647577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b92169190910492915050565b600063ffffffff808316818516818304811182151516156116765761167661158f565b0294935050505056fea164736f6c634300080f000a53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f77001d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be496e697469616c697a61626c653a20636f6e7472616374206973206e6f742069",
}

// SystemConfigABI is the input ABI used to generate the binding from.
// Deprecated: Use SystemConfigMetaData.ABI instead.
var SystemConfigABI = SystemConfigMetaData.ABI

// SystemConfigBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SystemConfigMetaData.Bin instead.
var SystemConfigBin = SystemConfigMetaData.Bin

// DeploySystemConfig deploys a new Ethereum contract, binding an instance of SystemConfig to it.
func DeploySystemConfig(auth *bind.TransactOpts, backend bind.ContractBackend, _owner common.Address, _overhead *big.Int, _scalar *big.Int, _batcherHash [32]byte, _gasLimit uint64, _unsafeBlockSigner common.Address, _config ResourceMeteringResourceConfig) (common.Address, *types.Transaction, *SystemConfig, error) {
	parsed, err := SystemConfigMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SystemConfigBin), backend, _owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config)
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

// BasefeeScalar is a free data retrieval call binding the contract method 0xbfb14fb7.
//
// Solidity: function basefeeScalar() view returns(uint32)
func (_SystemConfig *SystemConfigCaller) BasefeeScalar(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "basefeeScalar")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// BasefeeScalar is a free data retrieval call binding the contract method 0xbfb14fb7.
//
// Solidity: function basefeeScalar() view returns(uint32)
func (_SystemConfig *SystemConfigSession) BasefeeScalar() (uint32, error) {
	return _SystemConfig.Contract.BasefeeScalar(&_SystemConfig.CallOpts)
}

// BasefeeScalar is a free data retrieval call binding the contract method 0xbfb14fb7.
//
// Solidity: function basefeeScalar() view returns(uint32)
func (_SystemConfig *SystemConfigCallerSession) BasefeeScalar() (uint32, error) {
	return _SystemConfig.Contract.BasefeeScalar(&_SystemConfig.CallOpts)
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

// BlobBasefeeScalar is a free data retrieval call binding the contract method 0x7744864d.
//
// Solidity: function blobBasefeeScalar() view returns(uint32)
func (_SystemConfig *SystemConfigCaller) BlobBasefeeScalar(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _SystemConfig.contract.Call(opts, &out, "blobBasefeeScalar")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// BlobBasefeeScalar is a free data retrieval call binding the contract method 0x7744864d.
//
// Solidity: function blobBasefeeScalar() view returns(uint32)
func (_SystemConfig *SystemConfigSession) BlobBasefeeScalar() (uint32, error) {
	return _SystemConfig.Contract.BlobBasefeeScalar(&_SystemConfig.CallOpts)
}

// BlobBasefeeScalar is a free data retrieval call binding the contract method 0x7744864d.
//
// Solidity: function blobBasefeeScalar() view returns(uint32)
func (_SystemConfig *SystemConfigCallerSession) BlobBasefeeScalar() (uint32, error) {
	return _SystemConfig.Contract.BlobBasefeeScalar(&_SystemConfig.CallOpts)
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

// Initialize is a paid mutator transaction binding the contract method 0xf975e925.
//
// Solidity: function initialize(address _owner, uint256 _overhead, uint256 _scalar, bytes32 _batcherHash, uint64 _gasLimit, address _unsafeBlockSigner, (uint32,uint8,uint8,uint32,uint32,uint128) _config) returns()
func (_SystemConfig *SystemConfigTransactor) Initialize(opts *bind.TransactOpts, _owner common.Address, _overhead *big.Int, _scalar *big.Int, _batcherHash [32]byte, _gasLimit uint64, _unsafeBlockSigner common.Address, _config ResourceMeteringResourceConfig) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "initialize", _owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config)
}

// Initialize is a paid mutator transaction binding the contract method 0xf975e925.
//
// Solidity: function initialize(address _owner, uint256 _overhead, uint256 _scalar, bytes32 _batcherHash, uint64 _gasLimit, address _unsafeBlockSigner, (uint32,uint8,uint8,uint32,uint32,uint128) _config) returns()
func (_SystemConfig *SystemConfigSession) Initialize(_owner common.Address, _overhead *big.Int, _scalar *big.Int, _batcherHash [32]byte, _gasLimit uint64, _unsafeBlockSigner common.Address, _config ResourceMeteringResourceConfig) (*types.Transaction, error) {
	return _SystemConfig.Contract.Initialize(&_SystemConfig.TransactOpts, _owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config)
}

// Initialize is a paid mutator transaction binding the contract method 0xf975e925.
//
// Solidity: function initialize(address _owner, uint256 _overhead, uint256 _scalar, bytes32 _batcherHash, uint64 _gasLimit, address _unsafeBlockSigner, (uint32,uint8,uint8,uint32,uint32,uint128) _config) returns()
func (_SystemConfig *SystemConfigTransactorSession) Initialize(_owner common.Address, _overhead *big.Int, _scalar *big.Int, _batcherHash [32]byte, _gasLimit uint64, _unsafeBlockSigner common.Address, _config ResourceMeteringResourceConfig) (*types.Transaction, error) {
	return _SystemConfig.Contract.Initialize(&_SystemConfig.TransactOpts, _owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner, _config)
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

// SetGasConfigEcotone is a paid mutator transaction binding the contract method 0x21d7fde5.
//
// Solidity: function setGasConfigEcotone(uint32 _basefeeScalar, uint32 _blobBasefeeScalar) returns()
func (_SystemConfig *SystemConfigTransactor) SetGasConfigEcotone(opts *bind.TransactOpts, _basefeeScalar uint32, _blobBasefeeScalar uint32) (*types.Transaction, error) {
	return _SystemConfig.contract.Transact(opts, "setGasConfigEcotone", _basefeeScalar, _blobBasefeeScalar)
}

// SetGasConfigEcotone is a paid mutator transaction binding the contract method 0x21d7fde5.
//
// Solidity: function setGasConfigEcotone(uint32 _basefeeScalar, uint32 _blobBasefeeScalar) returns()
func (_SystemConfig *SystemConfigSession) SetGasConfigEcotone(_basefeeScalar uint32, _blobBasefeeScalar uint32) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetGasConfigEcotone(&_SystemConfig.TransactOpts, _basefeeScalar, _blobBasefeeScalar)
}

// SetGasConfigEcotone is a paid mutator transaction binding the contract method 0x21d7fde5.
//
// Solidity: function setGasConfigEcotone(uint32 _basefeeScalar, uint32 _blobBasefeeScalar) returns()
func (_SystemConfig *SystemConfigTransactorSession) SetGasConfigEcotone(_basefeeScalar uint32, _blobBasefeeScalar uint32) (*types.Transaction, error) {
	return _SystemConfig.Contract.SetGasConfigEcotone(&_SystemConfig.TransactOpts, _basefeeScalar, _blobBasefeeScalar)
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
