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

// L1CrossDomainMessengerMetaData contains all meta data concerning the L1CrossDomainMessenger contract.
var L1CrossDomainMessengerMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_portal\",\"type\":\"address\",\"internalType\":\"contractOptimismPortal\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"MESSAGE_VERSION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MIN_GAS_CALLDATA_OVERHEAD\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"OTHER_MESSENGER\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"PORTAL\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOptimismPortal\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RELAY_CALL_OVERHEAD\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RELAY_CONSTANT_OVERHEAD\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RELAY_GAS_CHECK_BUFFER\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RELAY_RESERVED_GAS\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"baseGas\",\"inputs\":[{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_minGasLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"failedMessages\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_superchainConfig\",\"type\":\"address\",\"internalType\":\"contractSuperchainConfig\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"messageNonce\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"paused\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"portal\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"relayMessage\",\"inputs\":[{\"name\":\"_nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_minGasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"sendMessage\",\"inputs\":[{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_minGasLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"setSuperchainConfig\",\"inputs\":[{\"name\":\"_superchainConfig\",\"type\":\"address\",\"internalType\":\"contractSuperchainConfig\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"successfulMessages\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"superchainConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractSuperchainConfig\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"xDomainMessageSender\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"FailedRelayedMessage\",\"inputs\":[{\"name\":\"msgHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RelayedMessage\",\"inputs\":[{\"name\":\"msgHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SentMessage\",\"inputs\":[{\"name\":\"target\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"message\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"},{\"name\":\"messageNonce\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"gasLimit\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SentMessageExtension1\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
	Bin: "0x60c06040523480156200001157600080fd5b506040516200232f3803806200232f833981016040819052620000349162000269565b7342000000000000000000000000000000000000076080526001600160a01b03811660a0526200006560006200006c565b506200029b565b600054600160a81b900460ff16158080156200009557506000546001600160a01b90910460ff16105b80620000cc5750620000b230620001d560201b620015691760201c565b158015620000cc5750600054600160a01b900460ff166001145b620001355760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805460ff60a01b1916600160a01b179055801562000163576000805460ff60a81b1916600160a81b1790555b60fb80546001600160a01b0319166001600160a01b03841617905562000188620001e4565b8015620001d1576000805460ff60a81b19169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050565b6001600160a01b03163b151590565b600054600160a81b900460ff16620002535760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201526a6e697469616c697a696e6760a81b60648201526084016200012c565b60cc80546001600160a01b03191661dead179055565b6000602082840312156200027c57600080fd5b81516001600160a01b03811681146200029457600080fd5b9392505050565b60805160a05161203d620002f2600039600081816101df0152818161035b015281816115c20152818161176f015281816117d0015261189c0152600081816103bd0152818161051c0152611799015261203d6000f3fe6080604052600436106101805760003560e01c80636425666b116100d6578063b1b1b2091161007f578063d764ad0b11610059578063d764ad0b1461047f578063e23754da14610492578063ecc70428146104b257600080fd5b8063b1b1b2091461040f578063b28ade251461043f578063c4d66de81461045f57600080fd5b80638cbeeef2116100b05780638cbeeef2146102a55780639fce812c146103ab578063a4e7f8bd146103df57600080fd5b80636425666b1461034c5780636e296e451461037f57806383a740741461039457600080fd5b80633dbb202b1161013857806354fd4d501161011257806354fd4d50146102bb5780635644cfdf146103115780635c975abb1461032757600080fd5b80633dbb202b146102685780633f827a5a1461027d5780634c1d6a69146102a557600080fd5b80630ff754ea116101695780630ff754ea146101cd5780632828d7e81461022657806335e80ab31461023b57600080fd5b8063028f85f7146101855780630c568498146101b8575b600080fd5b34801561019157600080fd5b5061019a601081565b60405167ffffffffffffffff90911681526020015b60405180910390f35b3480156101c457600080fd5b5061019a603f81565b3480156101d957600080fd5b506102017f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101af565b34801561023257600080fd5b5061019a604081565b34801561024757600080fd5b5060fb546102019073ffffffffffffffffffffffffffffffffffffffff1681565b61027b610276366004611ae7565b610517565b005b34801561028957600080fd5b50610292600181565b60405161ffff90911681526020016101af565b3480156102b157600080fd5b5061019a619c4081565b3480156102c757600080fd5b506103046040518060400160405280600581526020017f322e312e3100000000000000000000000000000000000000000000000000000081525081565b6040516101af9190611bb9565b34801561031d57600080fd5b5061019a61138881565b34801561033357600080fd5b5061033c61077b565b60405190151581526020016101af565b34801561035857600080fd5b507f0000000000000000000000000000000000000000000000000000000000000000610201565b34801561038b57600080fd5b50610201610814565b3480156103a057600080fd5b5061019a62030d4081565b3480156103b757600080fd5b506102017f000000000000000000000000000000000000000000000000000000000000000081565b3480156103eb57600080fd5b5061033c6103fa366004611bd3565b60ce6020526000908152604090205460ff1681565b34801561041b57600080fd5b5061033c61042a366004611bd3565b60cb6020526000908152604090205460ff1681565b34801561044b57600080fd5b5061019a61045a366004611bec565b610900565b34801561046b57600080fd5b5061027b61047a366004611c40565b61096e565b61027b61048d366004611c5d565b610bac565b34801561049e57600080fd5b5061027b6104ad366004611c40565b6114a2565b3480156104be57600080fd5b5061050960cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b6040519081526020016101af565b6106507f0000000000000000000000000000000000000000000000000000000000000000610546858585610900565b347fd764ad0b000000000000000000000000000000000000000000000000000000006105b260cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b338a34898c8c6040516024016105ce9796959493929190611d2c565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff0000000000000000000000000000000000000000000000000000000090931692909217909152611585565b8373ffffffffffffffffffffffffffffffffffffffff167fcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a3385856106d560cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b866040516106e7959493929190611d8b565b60405180910390a260405134815233907f8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d5469060200160405180910390a2505060cd80547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808216600101167fffff0000000000000000000000000000000000000000000000000000000000009091161790555050565b60fb54604080517f5c975abb000000000000000000000000000000000000000000000000000000008152905160009273ffffffffffffffffffffffffffffffffffffffff1691635c975abb9160048083019260209291908290030181865afa1580156107eb573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061080f9190611dd9565b905090565b60cc5460009073ffffffffffffffffffffffffffffffffffffffff167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff2153016108e3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603560248201527f43726f7373446f6d61696e4d657373656e6765723a2078446f6d61696e4d657360448201527f7361676553656e646572206973206e6f7420736574000000000000000000000060648201526084015b60405180910390fd5b5060cc5473ffffffffffffffffffffffffffffffffffffffff1690565b6000611388619c4080603f61091c604063ffffffff8816611e2a565b6109269190611e5a565b610931601088611e2a565b61093e9062030d40611ea8565b6109489190611ea8565b6109529190611ea8565b61095c9190611ea8565b6109669190611ea8565b949350505050565b6000547501000000000000000000000000000000000000000000900460ff16158080156109b9575060005460017401000000000000000000000000000000000000000090910460ff16105b806109eb5750303b1580156109eb575060005474010000000000000000000000000000000000000000900460ff166001145b610a77576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016108da565b600080547fffffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffff16740100000000000000000000000000000000000000001790558015610afd57600080547fffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffffff1675010000000000000000000000000000000000000000001790555b60fb80547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff8416179055610b4561163a565b8015610ba857600080547fffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffffff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050565b610bb461077b565b15610c1b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601c60248201527f43726f7373446f6d61696e4d657373656e6765723a207061757365640000000060448201526064016108da565b60f087901c60028110610cd6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604d60248201527f43726f7373446f6d61696e4d657373656e6765723a206f6e6c7920766572736960448201527f6f6e2030206f722031206d657373616765732061726520737570706f7274656460648201527f20617420746869732074696d6500000000000000000000000000000000000000608482015260a4016108da565b8061ffff16600003610dcb576000610d27878986868080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152508f9250611713915050565b600081815260cb602052604090205490915060ff1615610dc9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f43726f7373446f6d61696e4d657373656e6765723a206c65676163792077697460448201527f6864726177616c20616c72656164792072656c6179656400000000000000000060648201526084016108da565b505b6000610e11898989898989898080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061173292505050565b9050610e1b611755565b15610e5357853414610e2f57610e2f611ed4565b600081815260ce602052604090205460ff1615610e4e57610e4e611ed4565b610fa5565b3415610f07576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605060248201527f43726f7373446f6d61696e4d657373656e6765723a2076616c7565206d75737460448201527f206265207a65726f20756e6c657373206d6573736167652069732066726f6d2060648201527f612073797374656d206164647265737300000000000000000000000000000000608482015260a4016108da565b600081815260ce602052604090205460ff16610fa5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f43726f7373446f6d61696e4d657373656e6765723a206d65737361676520636160448201527f6e6e6f74206265207265706c617965640000000000000000000000000000000060648201526084016108da565b610fae87611879565b15611061576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604360248201527f43726f7373446f6d61696e4d657373656e6765723a2063616e6e6f742073656e60448201527f64206d65737361676520746f20626c6f636b65642073797374656d206164647260648201527f6573730000000000000000000000000000000000000000000000000000000000608482015260a4016108da565b600081815260cb602052604090205460ff1615611100576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f43726f7373446f6d61696e4d657373656e6765723a206d65737361676520686160448201527f7320616c7265616479206265656e2072656c617965640000000000000000000060648201526084016108da565b61112185611112611388619c40611ea8565b67ffffffffffffffff166118f0565b1580611147575060cc5473ffffffffffffffffffffffffffffffffffffffff1661dead14155b1561126057600081815260ce602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555182917f99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f91a27fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff3201611259576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602d60248201527f43726f7373446f6d61696e4d657373656e6765723a206661696c656420746f2060448201527f72656c6179206d6573736167650000000000000000000000000000000000000060648201526084016108da565b5050611499565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff8a1617905560006112f188619c405a6112b49190611f03565b8988888080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061190e92505050565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead1790559050801561138857600082815260cb602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183917f4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c91a2611495565b600082815260ce602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183917f99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f91a27fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff3201611495576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602d60248201527f43726f7373446f6d61696e4d657373656e6765723a206661696c656420746f2060448201527f72656c6179206d6573736167650000000000000000000000000000000000000060648201526084016108da565b5050505b50505050505050565b60fb5473ffffffffffffffffffffffffffffffffffffffff1615611522576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601c60248201527f5375706572636861696e436f6e66696720616c7265616479207365740000000060448201526064016108da565b60fb80547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b6040517fe9e05c4200000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000169063e9e05c42908490611602908890839089906000908990600401611f1a565b6000604051808303818588803b15801561161b57600080fd5b505af115801561162f573d6000803e3d6000fd5b505050505050505050565b6000547501000000000000000000000000000000000000000000900460ff166116e5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016108da565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead179055565b600061172185858585611928565b805190602001209050949350505050565b60006117428787878787876119c1565b8051906020012090509695505050505050565b60003373ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614801561080f57507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16639bf62d826040518163ffffffff1660e01b8152600401602060405180830381865afa158015611839573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061185d9190611f72565b73ffffffffffffffffffffffffffffffffffffffff1614905090565b600073ffffffffffffffffffffffffffffffffffffffff82163014806118ea57507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16145b92915050565b600080603f83619c4001026040850201603f5a021015949350505050565b600080600080845160208601878a8af19695505050505050565b6060848484846040516024016119419493929190611f8f565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fcbd4ece9000000000000000000000000000000000000000000000000000000001790529050949350505050565b60608686868686866040516024016119de96959493929190611fd9565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fd764ad0b0000000000000000000000000000000000000000000000000000000017905290509695505050505050565b73ffffffffffffffffffffffffffffffffffffffff81168114611a8257600080fd5b50565b60008083601f840112611a9757600080fd5b50813567ffffffffffffffff811115611aaf57600080fd5b602083019150836020828501011115611ac757600080fd5b9250929050565b803563ffffffff81168114611ae257600080fd5b919050565b60008060008060608587031215611afd57600080fd5b8435611b0881611a60565b9350602085013567ffffffffffffffff811115611b2457600080fd5b611b3087828801611a85565b9094509250611b43905060408601611ace565b905092959194509250565b6000815180845260005b81811015611b7457602081850181015186830182015201611b58565b81811115611b86576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000611bcc6020830184611b4e565b9392505050565b600060208284031215611be557600080fd5b5035919050565b600080600060408486031215611c0157600080fd5b833567ffffffffffffffff811115611c1857600080fd5b611c2486828701611a85565b9094509250611c37905060208501611ace565b90509250925092565b600060208284031215611c5257600080fd5b8135611bcc81611a60565b600080600080600080600060c0888a031215611c7857600080fd5b873596506020880135611c8a81611a60565b95506040880135611c9a81611a60565b9450606088013593506080880135925060a088013567ffffffffffffffff811115611cc457600080fd5b611cd08a828b01611a85565b989b979a50959850939692959293505050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b878152600073ffffffffffffffffffffffffffffffffffffffff808916602084015280881660408401525085606083015263ffffffff8516608083015260c060a0830152611d7e60c083018486611ce3565b9998505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff86168152608060208201526000611dbb608083018688611ce3565b905083604083015263ffffffff831660608301529695505050505050565b600060208284031215611deb57600080fd5b81518015158114611bcc57600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600067ffffffffffffffff80831681851681830481118215151615611e5157611e51611dfb565b02949350505050565b600067ffffffffffffffff80841680611e9c577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b92169190910492915050565b600067ffffffffffffffff808316818516808303821115611ecb57611ecb611dfb565b01949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052600160045260246000fd5b600082821015611f1557611f15611dfb565b500390565b73ffffffffffffffffffffffffffffffffffffffff8616815284602082015267ffffffffffffffff84166040820152821515606082015260a060808201526000611f6760a0830184611b4e565b979650505050505050565b600060208284031215611f8457600080fd5b8151611bcc81611a60565b600073ffffffffffffffffffffffffffffffffffffffff808716835280861660208401525060806040830152611fc86080830185611b4e565b905082606083015295945050505050565b868152600073ffffffffffffffffffffffffffffffffffffffff808816602084015280871660408401525084606083015283608083015260c060a083015261202460c0830184611b4e565b9897505050505050505056fea164736f6c634300080f000a",
}

// L1CrossDomainMessengerABI is the input ABI used to generate the binding from.
// Deprecated: Use L1CrossDomainMessengerMetaData.ABI instead.
var L1CrossDomainMessengerABI = L1CrossDomainMessengerMetaData.ABI

// L1CrossDomainMessengerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1CrossDomainMessengerMetaData.Bin instead.
var L1CrossDomainMessengerBin = L1CrossDomainMessengerMetaData.Bin

// DeployL1CrossDomainMessenger deploys a new Ethereum contract, binding an instance of L1CrossDomainMessenger to it.
func DeployL1CrossDomainMessenger(auth *bind.TransactOpts, backend bind.ContractBackend, _portal common.Address) (common.Address, *types.Transaction, *L1CrossDomainMessenger, error) {
	parsed, err := L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1CrossDomainMessengerBin), backend, _portal)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1CrossDomainMessenger{L1CrossDomainMessengerCaller: L1CrossDomainMessengerCaller{contract: contract}, L1CrossDomainMessengerTransactor: L1CrossDomainMessengerTransactor{contract: contract}, L1CrossDomainMessengerFilterer: L1CrossDomainMessengerFilterer{contract: contract}}, nil
}

// L1CrossDomainMessenger is an auto generated Go binding around an Ethereum contract.
type L1CrossDomainMessenger struct {
	L1CrossDomainMessengerCaller     // Read-only binding to the contract
	L1CrossDomainMessengerTransactor // Write-only binding to the contract
	L1CrossDomainMessengerFilterer   // Log filterer for contract events
}

// L1CrossDomainMessengerCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1CrossDomainMessengerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1CrossDomainMessengerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1CrossDomainMessengerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1CrossDomainMessengerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1CrossDomainMessengerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1CrossDomainMessengerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1CrossDomainMessengerSession struct {
	Contract     *L1CrossDomainMessenger // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// L1CrossDomainMessengerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1CrossDomainMessengerCallerSession struct {
	Contract *L1CrossDomainMessengerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// L1CrossDomainMessengerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1CrossDomainMessengerTransactorSession struct {
	Contract     *L1CrossDomainMessengerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// L1CrossDomainMessengerRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1CrossDomainMessengerRaw struct {
	Contract *L1CrossDomainMessenger // Generic contract binding to access the raw methods on
}

// L1CrossDomainMessengerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1CrossDomainMessengerCallerRaw struct {
	Contract *L1CrossDomainMessengerCaller // Generic read-only contract binding to access the raw methods on
}

// L1CrossDomainMessengerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1CrossDomainMessengerTransactorRaw struct {
	Contract *L1CrossDomainMessengerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1CrossDomainMessenger creates a new instance of L1CrossDomainMessenger, bound to a specific deployed contract.
func NewL1CrossDomainMessenger(address common.Address, backend bind.ContractBackend) (*L1CrossDomainMessenger, error) {
	contract, err := bindL1CrossDomainMessenger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1CrossDomainMessenger{L1CrossDomainMessengerCaller: L1CrossDomainMessengerCaller{contract: contract}, L1CrossDomainMessengerTransactor: L1CrossDomainMessengerTransactor{contract: contract}, L1CrossDomainMessengerFilterer: L1CrossDomainMessengerFilterer{contract: contract}}, nil
}

// NewL1CrossDomainMessengerCaller creates a new read-only instance of L1CrossDomainMessenger, bound to a specific deployed contract.
func NewL1CrossDomainMessengerCaller(address common.Address, caller bind.ContractCaller) (*L1CrossDomainMessengerCaller, error) {
	contract, err := bindL1CrossDomainMessenger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1CrossDomainMessengerCaller{contract: contract}, nil
}

// NewL1CrossDomainMessengerTransactor creates a new write-only instance of L1CrossDomainMessenger, bound to a specific deployed contract.
func NewL1CrossDomainMessengerTransactor(address common.Address, transactor bind.ContractTransactor) (*L1CrossDomainMessengerTransactor, error) {
	contract, err := bindL1CrossDomainMessenger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1CrossDomainMessengerTransactor{contract: contract}, nil
}

// NewL1CrossDomainMessengerFilterer creates a new log filterer instance of L1CrossDomainMessenger, bound to a specific deployed contract.
func NewL1CrossDomainMessengerFilterer(address common.Address, filterer bind.ContractFilterer) (*L1CrossDomainMessengerFilterer, error) {
	contract, err := bindL1CrossDomainMessenger(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1CrossDomainMessengerFilterer{contract: contract}, nil
}

// bindL1CrossDomainMessenger binds a generic wrapper to an already deployed contract.
func bindL1CrossDomainMessenger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1CrossDomainMessengerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1CrossDomainMessenger *L1CrossDomainMessengerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1CrossDomainMessenger.Contract.L1CrossDomainMessengerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1CrossDomainMessenger *L1CrossDomainMessengerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.L1CrossDomainMessengerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1CrossDomainMessenger *L1CrossDomainMessengerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.L1CrossDomainMessengerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1CrossDomainMessenger.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1CrossDomainMessenger *L1CrossDomainMessengerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1CrossDomainMessenger *L1CrossDomainMessengerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.contract.Transact(opts, method, params...)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) MESSAGEVERSION(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "MESSAGE_VERSION")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) MESSAGEVERSION() (uint16, error) {
	return _L1CrossDomainMessenger.Contract.MESSAGEVERSION(&_L1CrossDomainMessenger.CallOpts)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) MESSAGEVERSION() (uint16, error) {
	return _L1CrossDomainMessenger.Contract.MESSAGEVERSION(&_L1CrossDomainMessenger.CallOpts)
}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) MINGASCALLDATAOVERHEAD(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_CALLDATA_OVERHEAD")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) MINGASCALLDATAOVERHEAD() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.MINGASCALLDATAOVERHEAD(&_L1CrossDomainMessenger.CallOpts)
}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) MINGASCALLDATAOVERHEAD() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.MINGASCALLDATAOVERHEAD(&_L1CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) MINGASDYNAMICOVERHEADDENOMINATOR(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) MINGASDYNAMICOVERHEADDENOMINATOR() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADDENOMINATOR(&_L1CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) MINGASDYNAMICOVERHEADDENOMINATOR() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADDENOMINATOR(&_L1CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) MINGASDYNAMICOVERHEADNUMERATOR(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) MINGASDYNAMICOVERHEADNUMERATOR() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADNUMERATOR(&_L1CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) MINGASDYNAMICOVERHEADNUMERATOR() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADNUMERATOR(&_L1CrossDomainMessenger.CallOpts)
}

// OTHERMESSENGER is a free data retrieval call binding the contract method 0x9fce812c.
//
// Solidity: function OTHER_MESSENGER() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) OTHERMESSENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "OTHER_MESSENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OTHERMESSENGER is a free data retrieval call binding the contract method 0x9fce812c.
//
// Solidity: function OTHER_MESSENGER() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) OTHERMESSENGER() (common.Address, error) {
	return _L1CrossDomainMessenger.Contract.OTHERMESSENGER(&_L1CrossDomainMessenger.CallOpts)
}

// OTHERMESSENGER is a free data retrieval call binding the contract method 0x9fce812c.
//
// Solidity: function OTHER_MESSENGER() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) OTHERMESSENGER() (common.Address, error) {
	return _L1CrossDomainMessenger.Contract.OTHERMESSENGER(&_L1CrossDomainMessenger.CallOpts)
}

// PORTAL is a free data retrieval call binding the contract method 0x0ff754ea.
//
// Solidity: function PORTAL() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) PORTAL(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "PORTAL")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PORTAL is a free data retrieval call binding the contract method 0x0ff754ea.
//
// Solidity: function PORTAL() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) PORTAL() (common.Address, error) {
	return _L1CrossDomainMessenger.Contract.PORTAL(&_L1CrossDomainMessenger.CallOpts)
}

// PORTAL is a free data retrieval call binding the contract method 0x0ff754ea.
//
// Solidity: function PORTAL() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) PORTAL() (common.Address, error) {
	return _L1CrossDomainMessenger.Contract.PORTAL(&_L1CrossDomainMessenger.CallOpts)
}

// RELAYCALLOVERHEAD is a free data retrieval call binding the contract method 0x4c1d6a69.
//
// Solidity: function RELAY_CALL_OVERHEAD() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) RELAYCALLOVERHEAD(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "RELAY_CALL_OVERHEAD")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYCALLOVERHEAD is a free data retrieval call binding the contract method 0x4c1d6a69.
//
// Solidity: function RELAY_CALL_OVERHEAD() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) RELAYCALLOVERHEAD() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.RELAYCALLOVERHEAD(&_L1CrossDomainMessenger.CallOpts)
}

// RELAYCALLOVERHEAD is a free data retrieval call binding the contract method 0x4c1d6a69.
//
// Solidity: function RELAY_CALL_OVERHEAD() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) RELAYCALLOVERHEAD() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.RELAYCALLOVERHEAD(&_L1CrossDomainMessenger.CallOpts)
}

// RELAYCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x83a74074.
//
// Solidity: function RELAY_CONSTANT_OVERHEAD() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) RELAYCONSTANTOVERHEAD(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "RELAY_CONSTANT_OVERHEAD")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x83a74074.
//
// Solidity: function RELAY_CONSTANT_OVERHEAD() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) RELAYCONSTANTOVERHEAD() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.RELAYCONSTANTOVERHEAD(&_L1CrossDomainMessenger.CallOpts)
}

// RELAYCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x83a74074.
//
// Solidity: function RELAY_CONSTANT_OVERHEAD() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) RELAYCONSTANTOVERHEAD() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.RELAYCONSTANTOVERHEAD(&_L1CrossDomainMessenger.CallOpts)
}

// RELAYGASCHECKBUFFER is a free data retrieval call binding the contract method 0x5644cfdf.
//
// Solidity: function RELAY_GAS_CHECK_BUFFER() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) RELAYGASCHECKBUFFER(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "RELAY_GAS_CHECK_BUFFER")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYGASCHECKBUFFER is a free data retrieval call binding the contract method 0x5644cfdf.
//
// Solidity: function RELAY_GAS_CHECK_BUFFER() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) RELAYGASCHECKBUFFER() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.RELAYGASCHECKBUFFER(&_L1CrossDomainMessenger.CallOpts)
}

// RELAYGASCHECKBUFFER is a free data retrieval call binding the contract method 0x5644cfdf.
//
// Solidity: function RELAY_GAS_CHECK_BUFFER() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) RELAYGASCHECKBUFFER() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.RELAYGASCHECKBUFFER(&_L1CrossDomainMessenger.CallOpts)
}

// RELAYRESERVEDGAS is a free data retrieval call binding the contract method 0x8cbeeef2.
//
// Solidity: function RELAY_RESERVED_GAS() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) RELAYRESERVEDGAS(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "RELAY_RESERVED_GAS")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYRESERVEDGAS is a free data retrieval call binding the contract method 0x8cbeeef2.
//
// Solidity: function RELAY_RESERVED_GAS() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) RELAYRESERVEDGAS() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.RELAYRESERVEDGAS(&_L1CrossDomainMessenger.CallOpts)
}

// RELAYRESERVEDGAS is a free data retrieval call binding the contract method 0x8cbeeef2.
//
// Solidity: function RELAY_RESERVED_GAS() view returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) RELAYRESERVEDGAS() (uint64, error) {
	return _L1CrossDomainMessenger.Contract.RELAYRESERVEDGAS(&_L1CrossDomainMessenger.CallOpts)
}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) BaseGas(opts *bind.CallOpts, _message []byte, _minGasLimit uint32) (uint64, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "baseGas", _message, _minGasLimit)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) BaseGas(_message []byte, _minGasLimit uint32) (uint64, error) {
	return _L1CrossDomainMessenger.Contract.BaseGas(&_L1CrossDomainMessenger.CallOpts, _message, _minGasLimit)
}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint64)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) BaseGas(_message []byte, _minGasLimit uint32) (uint64, error) {
	return _L1CrossDomainMessenger.Contract.BaseGas(&_L1CrossDomainMessenger.CallOpts, _message, _minGasLimit)
}

// FailedMessages is a free data retrieval call binding the contract method 0xa4e7f8bd.
//
// Solidity: function failedMessages(bytes32 ) view returns(bool)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) FailedMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "failedMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// FailedMessages is a free data retrieval call binding the contract method 0xa4e7f8bd.
//
// Solidity: function failedMessages(bytes32 ) view returns(bool)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) FailedMessages(arg0 [32]byte) (bool, error) {
	return _L1CrossDomainMessenger.Contract.FailedMessages(&_L1CrossDomainMessenger.CallOpts, arg0)
}

// FailedMessages is a free data retrieval call binding the contract method 0xa4e7f8bd.
//
// Solidity: function failedMessages(bytes32 ) view returns(bool)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) FailedMessages(arg0 [32]byte) (bool, error) {
	return _L1CrossDomainMessenger.Contract.FailedMessages(&_L1CrossDomainMessenger.CallOpts, arg0)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) MessageNonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "messageNonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) MessageNonce() (*big.Int, error) {
	return _L1CrossDomainMessenger.Contract.MessageNonce(&_L1CrossDomainMessenger.CallOpts)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) MessageNonce() (*big.Int, error) {
	return _L1CrossDomainMessenger.Contract.MessageNonce(&_L1CrossDomainMessenger.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) Paused() (bool, error) {
	return _L1CrossDomainMessenger.Contract.Paused(&_L1CrossDomainMessenger.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) Paused() (bool, error) {
	return _L1CrossDomainMessenger.Contract.Paused(&_L1CrossDomainMessenger.CallOpts)
}

// Portal is a free data retrieval call binding the contract method 0x6425666b.
//
// Solidity: function portal() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) Portal(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "portal")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Portal is a free data retrieval call binding the contract method 0x6425666b.
//
// Solidity: function portal() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) Portal() (common.Address, error) {
	return _L1CrossDomainMessenger.Contract.Portal(&_L1CrossDomainMessenger.CallOpts)
}

// Portal is a free data retrieval call binding the contract method 0x6425666b.
//
// Solidity: function portal() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) Portal() (common.Address, error) {
	return _L1CrossDomainMessenger.Contract.Portal(&_L1CrossDomainMessenger.CallOpts)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) SuccessfulMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "successfulMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _L1CrossDomainMessenger.Contract.SuccessfulMessages(&_L1CrossDomainMessenger.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _L1CrossDomainMessenger.Contract.SuccessfulMessages(&_L1CrossDomainMessenger.CallOpts, arg0)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) SuperchainConfig(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "superchainConfig")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) SuperchainConfig() (common.Address, error) {
	return _L1CrossDomainMessenger.Contract.SuperchainConfig(&_L1CrossDomainMessenger.CallOpts)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) SuperchainConfig() (common.Address, error) {
	return _L1CrossDomainMessenger.Contract.SuperchainConfig(&_L1CrossDomainMessenger.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) Version() (string, error) {
	return _L1CrossDomainMessenger.Contract.Version(&_L1CrossDomainMessenger.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) Version() (string, error) {
	return _L1CrossDomainMessenger.Contract.Version(&_L1CrossDomainMessenger.CallOpts)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCaller) XDomainMessageSender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1CrossDomainMessenger.contract.Call(opts, &out, "xDomainMessageSender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) XDomainMessageSender() (common.Address, error) {
	return _L1CrossDomainMessenger.Contract.XDomainMessageSender(&_L1CrossDomainMessenger.CallOpts)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerCallerSession) XDomainMessageSender() (common.Address, error) {
	return _L1CrossDomainMessenger.Contract.XDomainMessageSender(&_L1CrossDomainMessenger.CallOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _superchainConfig) returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerTransactor) Initialize(opts *bind.TransactOpts, _superchainConfig common.Address) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.contract.Transact(opts, "initialize", _superchainConfig)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _superchainConfig) returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) Initialize(_superchainConfig common.Address) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.Initialize(&_L1CrossDomainMessenger.TransactOpts, _superchainConfig)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _superchainConfig) returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerTransactorSession) Initialize(_superchainConfig common.Address) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.Initialize(&_L1CrossDomainMessenger.TransactOpts, _superchainConfig)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd764ad0b.
//
// Solidity: function relayMessage(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerTransactor) RelayMessage(opts *bind.TransactOpts, _nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.contract.Transact(opts, "relayMessage", _nonce, _sender, _target, _value, _minGasLimit, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd764ad0b.
//
// Solidity: function relayMessage(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) RelayMessage(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.RelayMessage(&_L1CrossDomainMessenger.TransactOpts, _nonce, _sender, _target, _value, _minGasLimit, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd764ad0b.
//
// Solidity: function relayMessage(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerTransactorSession) RelayMessage(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.RelayMessage(&_L1CrossDomainMessenger.TransactOpts, _nonce, _sender, _target, _value, _minGasLimit, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerTransactor) SendMessage(opts *bind.TransactOpts, _target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.contract.Transact(opts, "sendMessage", _target, _message, _minGasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) SendMessage(_target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.SendMessage(&_L1CrossDomainMessenger.TransactOpts, _target, _message, _minGasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerTransactorSession) SendMessage(_target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.SendMessage(&_L1CrossDomainMessenger.TransactOpts, _target, _message, _minGasLimit)
}

// SetSuperchainConfig is a paid mutator transaction binding the contract method 0xe23754da.
//
// Solidity: function setSuperchainConfig(address _superchainConfig) returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerTransactor) SetSuperchainConfig(opts *bind.TransactOpts, _superchainConfig common.Address) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.contract.Transact(opts, "setSuperchainConfig", _superchainConfig)
}

// SetSuperchainConfig is a paid mutator transaction binding the contract method 0xe23754da.
//
// Solidity: function setSuperchainConfig(address _superchainConfig) returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerSession) SetSuperchainConfig(_superchainConfig common.Address) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.SetSuperchainConfig(&_L1CrossDomainMessenger.TransactOpts, _superchainConfig)
}

// SetSuperchainConfig is a paid mutator transaction binding the contract method 0xe23754da.
//
// Solidity: function setSuperchainConfig(address _superchainConfig) returns()
func (_L1CrossDomainMessenger *L1CrossDomainMessengerTransactorSession) SetSuperchainConfig(_superchainConfig common.Address) (*types.Transaction, error) {
	return _L1CrossDomainMessenger.Contract.SetSuperchainConfig(&_L1CrossDomainMessenger.TransactOpts, _superchainConfig)
}

// L1CrossDomainMessengerFailedRelayedMessageIterator is returned from FilterFailedRelayedMessage and is used to iterate over the raw logs and unpacked data for FailedRelayedMessage events raised by the L1CrossDomainMessenger contract.
type L1CrossDomainMessengerFailedRelayedMessageIterator struct {
	Event *L1CrossDomainMessengerFailedRelayedMessage // Event containing the contract specifics and raw log

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
func (it *L1CrossDomainMessengerFailedRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1CrossDomainMessengerFailedRelayedMessage)
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
		it.Event = new(L1CrossDomainMessengerFailedRelayedMessage)
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
func (it *L1CrossDomainMessengerFailedRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1CrossDomainMessengerFailedRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1CrossDomainMessengerFailedRelayedMessage represents a FailedRelayedMessage event raised by the L1CrossDomainMessenger contract.
type L1CrossDomainMessengerFailedRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterFailedRelayedMessage is a free log retrieval operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) FilterFailedRelayedMessage(opts *bind.FilterOpts, msgHash [][32]byte) (*L1CrossDomainMessengerFailedRelayedMessageIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L1CrossDomainMessenger.contract.FilterLogs(opts, "FailedRelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &L1CrossDomainMessengerFailedRelayedMessageIterator{contract: _L1CrossDomainMessenger.contract, event: "FailedRelayedMessage", logs: logs, sub: sub}, nil
}

// WatchFailedRelayedMessage is a free log subscription operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) WatchFailedRelayedMessage(opts *bind.WatchOpts, sink chan<- *L1CrossDomainMessengerFailedRelayedMessage, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L1CrossDomainMessenger.contract.WatchLogs(opts, "FailedRelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1CrossDomainMessengerFailedRelayedMessage)
				if err := _L1CrossDomainMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
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

// ParseFailedRelayedMessage is a log parse operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) ParseFailedRelayedMessage(log types.Log) (*L1CrossDomainMessengerFailedRelayedMessage, error) {
	event := new(L1CrossDomainMessengerFailedRelayedMessage)
	if err := _L1CrossDomainMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1CrossDomainMessengerInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L1CrossDomainMessenger contract.
type L1CrossDomainMessengerInitializedIterator struct {
	Event *L1CrossDomainMessengerInitialized // Event containing the contract specifics and raw log

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
func (it *L1CrossDomainMessengerInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1CrossDomainMessengerInitialized)
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
		it.Event = new(L1CrossDomainMessengerInitialized)
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
func (it *L1CrossDomainMessengerInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1CrossDomainMessengerInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1CrossDomainMessengerInitialized represents a Initialized event raised by the L1CrossDomainMessenger contract.
type L1CrossDomainMessengerInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) FilterInitialized(opts *bind.FilterOpts) (*L1CrossDomainMessengerInitializedIterator, error) {

	logs, sub, err := _L1CrossDomainMessenger.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L1CrossDomainMessengerInitializedIterator{contract: _L1CrossDomainMessenger.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L1CrossDomainMessengerInitialized) (event.Subscription, error) {

	logs, sub, err := _L1CrossDomainMessenger.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1CrossDomainMessengerInitialized)
				if err := _L1CrossDomainMessenger.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) ParseInitialized(log types.Log) (*L1CrossDomainMessengerInitialized, error) {
	event := new(L1CrossDomainMessengerInitialized)
	if err := _L1CrossDomainMessenger.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1CrossDomainMessengerRelayedMessageIterator is returned from FilterRelayedMessage and is used to iterate over the raw logs and unpacked data for RelayedMessage events raised by the L1CrossDomainMessenger contract.
type L1CrossDomainMessengerRelayedMessageIterator struct {
	Event *L1CrossDomainMessengerRelayedMessage // Event containing the contract specifics and raw log

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
func (it *L1CrossDomainMessengerRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1CrossDomainMessengerRelayedMessage)
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
		it.Event = new(L1CrossDomainMessengerRelayedMessage)
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
func (it *L1CrossDomainMessengerRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1CrossDomainMessengerRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1CrossDomainMessengerRelayedMessage represents a RelayedMessage event raised by the L1CrossDomainMessenger contract.
type L1CrossDomainMessengerRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRelayedMessage is a free log retrieval operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) FilterRelayedMessage(opts *bind.FilterOpts, msgHash [][32]byte) (*L1CrossDomainMessengerRelayedMessageIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L1CrossDomainMessenger.contract.FilterLogs(opts, "RelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &L1CrossDomainMessengerRelayedMessageIterator{contract: _L1CrossDomainMessenger.contract, event: "RelayedMessage", logs: logs, sub: sub}, nil
}

// WatchRelayedMessage is a free log subscription operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) WatchRelayedMessage(opts *bind.WatchOpts, sink chan<- *L1CrossDomainMessengerRelayedMessage, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L1CrossDomainMessenger.contract.WatchLogs(opts, "RelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1CrossDomainMessengerRelayedMessage)
				if err := _L1CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
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

// ParseRelayedMessage is a log parse operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) ParseRelayedMessage(log types.Log) (*L1CrossDomainMessengerRelayedMessage, error) {
	event := new(L1CrossDomainMessengerRelayedMessage)
	if err := _L1CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1CrossDomainMessengerSentMessageIterator is returned from FilterSentMessage and is used to iterate over the raw logs and unpacked data for SentMessage events raised by the L1CrossDomainMessenger contract.
type L1CrossDomainMessengerSentMessageIterator struct {
	Event *L1CrossDomainMessengerSentMessage // Event containing the contract specifics and raw log

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
func (it *L1CrossDomainMessengerSentMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1CrossDomainMessengerSentMessage)
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
		it.Event = new(L1CrossDomainMessengerSentMessage)
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
func (it *L1CrossDomainMessengerSentMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1CrossDomainMessengerSentMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1CrossDomainMessengerSentMessage represents a SentMessage event raised by the L1CrossDomainMessenger contract.
type L1CrossDomainMessengerSentMessage struct {
	Target       common.Address
	Sender       common.Address
	Message      []byte
	MessageNonce *big.Int
	GasLimit     *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSentMessage is a free log retrieval operation binding the contract event 0xcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a.
//
// Solidity: event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) FilterSentMessage(opts *bind.FilterOpts, target []common.Address) (*L1CrossDomainMessengerSentMessageIterator, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _L1CrossDomainMessenger.contract.FilterLogs(opts, "SentMessage", targetRule)
	if err != nil {
		return nil, err
	}
	return &L1CrossDomainMessengerSentMessageIterator{contract: _L1CrossDomainMessenger.contract, event: "SentMessage", logs: logs, sub: sub}, nil
}

// WatchSentMessage is a free log subscription operation binding the contract event 0xcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a.
//
// Solidity: event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) WatchSentMessage(opts *bind.WatchOpts, sink chan<- *L1CrossDomainMessengerSentMessage, target []common.Address) (event.Subscription, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _L1CrossDomainMessenger.contract.WatchLogs(opts, "SentMessage", targetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1CrossDomainMessengerSentMessage)
				if err := _L1CrossDomainMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
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

// ParseSentMessage is a log parse operation binding the contract event 0xcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a.
//
// Solidity: event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) ParseSentMessage(log types.Log) (*L1CrossDomainMessengerSentMessage, error) {
	event := new(L1CrossDomainMessengerSentMessage)
	if err := _L1CrossDomainMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1CrossDomainMessengerSentMessageExtension1Iterator is returned from FilterSentMessageExtension1 and is used to iterate over the raw logs and unpacked data for SentMessageExtension1 events raised by the L1CrossDomainMessenger contract.
type L1CrossDomainMessengerSentMessageExtension1Iterator struct {
	Event *L1CrossDomainMessengerSentMessageExtension1 // Event containing the contract specifics and raw log

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
func (it *L1CrossDomainMessengerSentMessageExtension1Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1CrossDomainMessengerSentMessageExtension1)
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
		it.Event = new(L1CrossDomainMessengerSentMessageExtension1)
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
func (it *L1CrossDomainMessengerSentMessageExtension1Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1CrossDomainMessengerSentMessageExtension1Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1CrossDomainMessengerSentMessageExtension1 represents a SentMessageExtension1 event raised by the L1CrossDomainMessenger contract.
type L1CrossDomainMessengerSentMessageExtension1 struct {
	Sender common.Address
	Value  *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterSentMessageExtension1 is a free log retrieval operation binding the contract event 0x8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d546.
//
// Solidity: event SentMessageExtension1(address indexed sender, uint256 value)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) FilterSentMessageExtension1(opts *bind.FilterOpts, sender []common.Address) (*L1CrossDomainMessengerSentMessageExtension1Iterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _L1CrossDomainMessenger.contract.FilterLogs(opts, "SentMessageExtension1", senderRule)
	if err != nil {
		return nil, err
	}
	return &L1CrossDomainMessengerSentMessageExtension1Iterator{contract: _L1CrossDomainMessenger.contract, event: "SentMessageExtension1", logs: logs, sub: sub}, nil
}

// WatchSentMessageExtension1 is a free log subscription operation binding the contract event 0x8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d546.
//
// Solidity: event SentMessageExtension1(address indexed sender, uint256 value)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) WatchSentMessageExtension1(opts *bind.WatchOpts, sink chan<- *L1CrossDomainMessengerSentMessageExtension1, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _L1CrossDomainMessenger.contract.WatchLogs(opts, "SentMessageExtension1", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1CrossDomainMessengerSentMessageExtension1)
				if err := _L1CrossDomainMessenger.contract.UnpackLog(event, "SentMessageExtension1", log); err != nil {
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

// ParseSentMessageExtension1 is a log parse operation binding the contract event 0x8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d546.
//
// Solidity: event SentMessageExtension1(address indexed sender, uint256 value)
func (_L1CrossDomainMessenger *L1CrossDomainMessengerFilterer) ParseSentMessageExtension1(log types.Log) (*L1CrossDomainMessengerSentMessageExtension1, error) {
	event := new(L1CrossDomainMessengerSentMessageExtension1)
	if err := _L1CrossDomainMessenger.contract.UnpackLog(event, "SentMessageExtension1", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
