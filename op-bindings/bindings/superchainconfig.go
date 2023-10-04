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

// TypesSequencerKeyPair is an auto generated low-level Go binding around an user-defined struct.
type TypesSequencerKeyPair struct {
	BatcherHash       [32]byte
	UnsafeBlockSigner common.Address
}

// SuperchainConfigMetaData contains all meta data concerning the SuperchainConfig contract.
var SuperchainConfigMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"enumSuperchainConfig.UpdateType\",\"name\":\"updateType\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"ConfigUpdate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DELAY_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"GUARDIAN_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"INITIATOR_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_PAUSE_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PAUSED_TIME_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SYSTEM_OWNER_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"VETOER_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"batcherHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"unsafeBlockSigner\",\"type\":\"address\"}],\"internalType\":\"structTypes.SequencerKeyPair\",\"name\":\"_sequencer\",\"type\":\"tuple\"}],\"name\":\"addSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"allowedSequencers\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"delay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"delay_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"guardian\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"guardian_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_systemOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_initiator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_vetoer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_guardian\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_delay\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_maxPause\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"batcherHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"unsafeBlockSigner\",\"type\":\"address\"}],\"internalType\":\"structTypes.SequencerKeyPair[]\",\"name\":\"_sequencers\",\"type\":\"tuple[]\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initiator\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"initiator_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"batcherHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"unsafeBlockSigner\",\"type\":\"address\"}],\"internalType\":\"structTypes.SequencerKeyPair\",\"name\":\"_sequencer\",\"type\":\"tuple\"}],\"name\":\"isAllowedSequencer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxPause\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxPause_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"duration\",\"type\":\"uint256\"}],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"paused_\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"batcherHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"unsafeBlockSigner\",\"type\":\"address\"}],\"internalType\":\"structTypes.SequencerKeyPair\",\"name\":\"_sequencer\",\"type\":\"tuple\"}],\"name\":\"removeSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"systemOwner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"systemOwner_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"vetoer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"vetoer_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b506200006c60008080808080806040519080825280602002602001820160405280156200006557816020015b60408051808201909152600080825260208201528152602001906001900390816200003d5790505b5062000072565b62000568565b600054600290610100900460ff1615801562000095575060005460ff8083169116105b620000fd5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b606482015260840160405180910390fd5b6000805461ffff191660ff8316176101001790556200011c88620001ee565b620001278762000284565b6200013286620002bd565b6200013d85620002f6565b62000148846200032f565b620001538362000387565b60005b8251811015620001a2576200018d838281518110620001795762000179620004ae565b6020026020010151620003c060201b60201c565b806200019981620004da565b91505062000156565b506000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050505050505050565b620002346200021f60017fe5134cb7d217efbc8c357a6644e3c656a6235651a8f25717e410cbf378e57753620004f6565b60001b826200045d60201b62000d061760201c565b60005b604080516001600160a01b0384166020820152600080516020620018b383398151915291015b60408051601f1981840301815290829052620002799162000510565b60405180910390a250565b620002b56200021f60017f12c56161f16f492fd4016a16e534c3a2bcceceb7f70ec9bb75867affe3370316620004f6565b600162000237565b620002ee6200021f60017f704ae3ec629461681409737f623e0cebb30122362e8cb04e0a0d3581d958db7d620004f6565b600262000237565b620003276200021f60017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe69620004f6565b600362000237565b620003606200021f60017f0e2f5ebd54326cdea9bf943c0fc37413dccba70cdeb76374557a8f757e898390620004f6565b60045b600080516020620018b3833981519152826040516020016200025d91815260200190565b620003b86200021f60017f1399bee5471a817c3420e8d52c99ada34eb0c2eaf753dd2f4555bc879d1cd994620004f6565b600562000363565b6000620003d8826200046160201b62000d0a1760201c565b6000818152600160208190526040909120805460ff1916909117905590506006600080516020620018b383398151915283604051602001620004359190815181526020918201516001600160a01b03169181019190915260400190565b60408051601f1981840301815290829052620004519162000510565b60405180910390a25050565b9055565b600081600001518260200151604051602001620004919291909182526001600160a01b0316602082015260400190565b604051602081830303815290604052805190602001209050919050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b600060018201620004ef57620004ef620004c4565b5060010190565b6000828210156200050b576200050b620004c4565b500390565b600060208083528351808285015260005b818110156200053f5785810183015185820160400152820162000521565b8181111562000552576000604083870101525b50601f01601f1916929092016040019392505050565b61133b80620005786000396000f3fe608060405234801561001057600080fd5b50600436106101825760003560e01c80636a42b8f8116100d8578063a2f9c4081161008c578063d8bff44011610066578063d8bff440146102d6578063d92a09bc146102de578063f1e8cf061461030157600080fd5b8063a2f9c408146102be578063b5f41ad8146102c6578063c23a451a146102ce57600080fd5b80638a6fb7a3116100bd5780638a6fb7a3146102905780639eb17d4b146102a3578063a0654956146102ab57600080fd5b80636a42b8f81461027557806376ea31a41461027d57600080fd5b8063452a93201161013a57806354fd4d501161011457806354fd4d501461020c5780635c39fcc1146102555780635c975abb1461025d57600080fd5b8063452a9320146101f45780634886eb9c146101fc5780634b5b189f1461020457600080fd5b8063332392021161016b57806333239202146101b757806333779254146101bf5780633f4ba83a146101ec57600080fd5b8063136439dd146101875780631cd94ec01461019c575b600080fd5b61019a610195366004610fa7565b610314565b005b6101a4610504565b6040519081526020015b60405180910390f35b6101a4610532565b6101c761055d565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101ae565b61019a610592565b6101c76106b3565b6101a46106e3565b6101a461070e565b6102486040518060400160405280600581526020017f312e302e3000000000000000000000000000000000000000000000000000000081525081565b6040516101ae919061102b565b6101c7610739565b610265610769565b60405190151581526020016101ae565b6101a46107a0565b61026561028b366004611143565b6107d0565b61019a61029e36600461115f565b6107f5565b6101a46109b3565b61019a6102b9366004611143565b6109de565b6101a4610aac565b6101a4610adc565b6101a4610b07565b6101c7610b32565b6102656102ec366004610fa7565b60016020526000908152604090205460ff1681565b61019a61030f366004611143565b610b62565b61031c6106b3565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146103db576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602960248201527f5375706572636861696e436f6e6669673a206f6e6c7920677561726469616e2060448201527f63616e207061757365000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b61040d61040960017f1399bee5471a817c3420e8d52c99ada34eb0c2eaf753dd2f4555bc879d1cd994611298565b5490565b81111561049c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f5375706572636861696e436f6e6669673a206475726174696f6e20657863656560448201527f6473206d6178506175736500000000000000000000000000000000000000000060648201526084016103d2565b6104d86104ca60017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b7611298565b6104d483426112af565b9055565b6040517f9e87fac88ff661f02d44f95383c817fece4bce600a3dab7a54406878b965e75290600090a150565b61052f60017f1399bee5471a817c3420e8d52c99ada34eb0c2eaf753dd2f4555bc879d1cd994611298565b81565b61052f60017fe5134cb7d217efbc8c357a6644e3c656a6235651a8f25717e410cbf378e57753611298565b600061058d61040960017fe5134cb7d217efbc8c357a6644e3c656a6235651a8f25717e410cbf378e57753611298565b905090565b61059a6106b3565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610654576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f5375706572636861696e436f6e6669673a206f6e6c7920677561726469616e2060448201527f63616e20756e706175736500000000000000000000000000000000000000000060648201526084016103d2565b61068861068260017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b7611298565b60009055565b6040517fa45f47fdea8a1efdd9029a5691c7f759c32b7c698632b563573e155625d1693390600090a1565b600061058d61040960017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe69611298565b61052f60017f704ae3ec629461681409737f623e0cebb30122362e8cb04e0a0d3581d958db7d611298565b61052f60017f12c56161f16f492fd4016a16e534c3a2bcceceb7f70ec9bb75867affe3370316611298565b600061058d61040960017f12c56161f16f492fd4016a16e534c3a2bcceceb7f70ec9bb75867affe3370316611298565b60004261079a61040960017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b7611298565b11905090565b600061058d61040960017f0e2f5ebd54326cdea9bf943c0fc37413dccba70cdeb76374557a8f757e898390611298565b6000806107dc83610d0a565b60009081526001602052604090205460ff169392505050565b600054600290610100900460ff16158015610817575060005460ff8083169116105b6108a3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016103d2565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff8316176101001790556108dd88610d63565b6108e687610e20565b6108ef86610e55565b6108f885610e8a565b61090184610ebf565b61090a83610f24565b60005b825181101561094a5761093883828151811061092b5761092b6112c7565b6020026020010151610f59565b80610942816112f6565b91505061090d565b50600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050505050505050565b61052f60017f0e2f5ebd54326cdea9bf943c0fc37413dccba70cdeb76374557a8f757e898390611298565b6109e6610739565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610aa0576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603260248201527f5375706572636861696e436f6e6669673a206f6e6c7920696e69746961746f7260448201527f2063616e206164642073657175656e636572000000000000000000000000000060648201526084016103d2565b610aa981610f59565b50565b600061058d61040960017f1399bee5471a817c3420e8d52c99ada34eb0c2eaf753dd2f4555bc879d1cd994611298565b61052f60017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b7611298565b61052f60017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe69611298565b600061058d61040960017f704ae3ec629461681409737f623e0cebb30122362e8cb04e0a0d3581d958db7d611298565b610b6a61055d565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610c24576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603960248201527f5375706572636861696e436f6e6669673a206f6e6c792073797374656d4f776e60448201527f65722063616e2072656d6f766520612073657175656e6365720000000000000060648201526084016103d2565b6000610c2f82610d0a565b600081815260016020526040902080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00169055905060075b7f7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb83604051602001610cc291908151815260209182015173ffffffffffffffffffffffffffffffffffffffff169181019190915260400190565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815290829052610cfa9161102b565b60405180910390a25050565b9055565b600081600001518260200151604051602001610d4692919091825273ffffffffffffffffffffffffffffffffffffffff16602082015260400190565b604051602081830303815290604052805190602001209050919050565b610d96610d9160017fe5134cb7d217efbc8c357a6644e3c656a6235651a8f25717e410cbf378e57753611298565b829055565b60005b6040805173ffffffffffffffffffffffffffffffffffffffff841660208201527f7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb91015b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815290829052610e159161102b565b60405180910390a250565b610e4e610d9160017f12c56161f16f492fd4016a16e534c3a2bcceceb7f70ec9bb75867affe3370316611298565b6001610d99565b610e83610d9160017f704ae3ec629461681409737f623e0cebb30122362e8cb04e0a0d3581d958db7d611298565b6002610d99565b610eb8610d9160017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe69611298565b6003610d99565b610eed610d9160017f0e2f5ebd54326cdea9bf943c0fc37413dccba70cdeb76374557a8f757e898390611298565b60045b7f7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb82604051602001610ddd91815260200190565b610f52610d9160017f1399bee5471a817c3420e8d52c99ada34eb0c2eaf753dd2f4555bc879d1cd994611298565b6005610ef0565b6000610f6482610d0a565b600081815260016020819052604090912080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016909117905590506006610c68565b600060208284031215610fb957600080fd5b5035919050565b6000815180845260005b81811015610fe657602081850181015186830182015201610fca565b81811115610ff8576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061103e6020830184610fc0565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff811182821017156110bb576110bb611045565b604052919050565b803573ffffffffffffffffffffffffffffffffffffffff811681146110e757600080fd5b919050565b6000604082840312156110fe57600080fd5b6040516040810181811067ffffffffffffffff8211171561112157611121611045565b60405282358152905080611137602084016110c3565b60208201525092915050565b60006040828403121561115557600080fd5b61103e83836110ec565b600080600080600080600060e0888a03121561117a57600080fd5b611183886110c3565b96506020611192818a016110c3565b965060406111a1818b016110c3565b96506111af60608b016110c3565b955060808a0135945060a08a0135935060c08a013567ffffffffffffffff808211156111da57600080fd5b818c0191508c601f8301126111ee57600080fd5b81358181111561120057611200611045565b61120e858260051b01611074565b818152858101925060069190911b83018501908e82111561122e57600080fd5b928501925b81841015611254576112458f856110ec565b83529284019291850191611233565b80965050505050505092959891949750929550565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000828210156112aa576112aa611269565b500390565b600082198211156112c2576112c2611269565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361132757611327611269565b506001019056fea164736f6c634300080f000a7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb",
}

// SuperchainConfigABI is the input ABI used to generate the binding from.
// Deprecated: Use SuperchainConfigMetaData.ABI instead.
var SuperchainConfigABI = SuperchainConfigMetaData.ABI

// SuperchainConfigBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SuperchainConfigMetaData.Bin instead.
var SuperchainConfigBin = SuperchainConfigMetaData.Bin

// DeploySuperchainConfig deploys a new Ethereum contract, binding an instance of SuperchainConfig to it.
func DeploySuperchainConfig(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SuperchainConfig, error) {
	parsed, err := SuperchainConfigMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SuperchainConfigBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SuperchainConfig{SuperchainConfigCaller: SuperchainConfigCaller{contract: contract}, SuperchainConfigTransactor: SuperchainConfigTransactor{contract: contract}, SuperchainConfigFilterer: SuperchainConfigFilterer{contract: contract}}, nil
}

// SuperchainConfig is an auto generated Go binding around an Ethereum contract.
type SuperchainConfig struct {
	SuperchainConfigCaller     // Read-only binding to the contract
	SuperchainConfigTransactor // Write-only binding to the contract
	SuperchainConfigFilterer   // Log filterer for contract events
}

// SuperchainConfigCaller is an auto generated read-only Go binding around an Ethereum contract.
type SuperchainConfigCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SuperchainConfigTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SuperchainConfigTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SuperchainConfigFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SuperchainConfigFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SuperchainConfigSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SuperchainConfigSession struct {
	Contract     *SuperchainConfig // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SuperchainConfigCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SuperchainConfigCallerSession struct {
	Contract *SuperchainConfigCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// SuperchainConfigTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SuperchainConfigTransactorSession struct {
	Contract     *SuperchainConfigTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// SuperchainConfigRaw is an auto generated low-level Go binding around an Ethereum contract.
type SuperchainConfigRaw struct {
	Contract *SuperchainConfig // Generic contract binding to access the raw methods on
}

// SuperchainConfigCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SuperchainConfigCallerRaw struct {
	Contract *SuperchainConfigCaller // Generic read-only contract binding to access the raw methods on
}

// SuperchainConfigTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SuperchainConfigTransactorRaw struct {
	Contract *SuperchainConfigTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSuperchainConfig creates a new instance of SuperchainConfig, bound to a specific deployed contract.
func NewSuperchainConfig(address common.Address, backend bind.ContractBackend) (*SuperchainConfig, error) {
	contract, err := bindSuperchainConfig(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SuperchainConfig{SuperchainConfigCaller: SuperchainConfigCaller{contract: contract}, SuperchainConfigTransactor: SuperchainConfigTransactor{contract: contract}, SuperchainConfigFilterer: SuperchainConfigFilterer{contract: contract}}, nil
}

// NewSuperchainConfigCaller creates a new read-only instance of SuperchainConfig, bound to a specific deployed contract.
func NewSuperchainConfigCaller(address common.Address, caller bind.ContractCaller) (*SuperchainConfigCaller, error) {
	contract, err := bindSuperchainConfig(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigCaller{contract: contract}, nil
}

// NewSuperchainConfigTransactor creates a new write-only instance of SuperchainConfig, bound to a specific deployed contract.
func NewSuperchainConfigTransactor(address common.Address, transactor bind.ContractTransactor) (*SuperchainConfigTransactor, error) {
	contract, err := bindSuperchainConfig(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigTransactor{contract: contract}, nil
}

// NewSuperchainConfigFilterer creates a new log filterer instance of SuperchainConfig, bound to a specific deployed contract.
func NewSuperchainConfigFilterer(address common.Address, filterer bind.ContractFilterer) (*SuperchainConfigFilterer, error) {
	contract, err := bindSuperchainConfig(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigFilterer{contract: contract}, nil
}

// bindSuperchainConfig binds a generic wrapper to an already deployed contract.
func bindSuperchainConfig(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SuperchainConfigABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SuperchainConfig *SuperchainConfigRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SuperchainConfig.Contract.SuperchainConfigCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SuperchainConfig *SuperchainConfigRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.SuperchainConfigTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SuperchainConfig *SuperchainConfigRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.SuperchainConfigTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SuperchainConfig *SuperchainConfigCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SuperchainConfig.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SuperchainConfig *SuperchainConfigTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SuperchainConfig *SuperchainConfigTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.contract.Transact(opts, method, params...)
}

// DELAYSLOT is a free data retrieval call binding the contract method 0x9eb17d4b.
//
// Solidity: function DELAY_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCaller) DELAYSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "DELAY_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DELAYSLOT is a free data retrieval call binding the contract method 0x9eb17d4b.
//
// Solidity: function DELAY_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigSession) DELAYSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.DELAYSLOT(&_SuperchainConfig.CallOpts)
}

// DELAYSLOT is a free data retrieval call binding the contract method 0x9eb17d4b.
//
// Solidity: function DELAY_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCallerSession) DELAYSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.DELAYSLOT(&_SuperchainConfig.CallOpts)
}

// GUARDIANSLOT is a free data retrieval call binding the contract method 0xc23a451a.
//
// Solidity: function GUARDIAN_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCaller) GUARDIANSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "GUARDIAN_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GUARDIANSLOT is a free data retrieval call binding the contract method 0xc23a451a.
//
// Solidity: function GUARDIAN_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigSession) GUARDIANSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.GUARDIANSLOT(&_SuperchainConfig.CallOpts)
}

// GUARDIANSLOT is a free data retrieval call binding the contract method 0xc23a451a.
//
// Solidity: function GUARDIAN_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCallerSession) GUARDIANSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.GUARDIANSLOT(&_SuperchainConfig.CallOpts)
}

// INITIATORSLOT is a free data retrieval call binding the contract method 0x4b5b189f.
//
// Solidity: function INITIATOR_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCaller) INITIATORSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "INITIATOR_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// INITIATORSLOT is a free data retrieval call binding the contract method 0x4b5b189f.
//
// Solidity: function INITIATOR_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigSession) INITIATORSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.INITIATORSLOT(&_SuperchainConfig.CallOpts)
}

// INITIATORSLOT is a free data retrieval call binding the contract method 0x4b5b189f.
//
// Solidity: function INITIATOR_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCallerSession) INITIATORSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.INITIATORSLOT(&_SuperchainConfig.CallOpts)
}

// MAXPAUSESLOT is a free data retrieval call binding the contract method 0x1cd94ec0.
//
// Solidity: function MAX_PAUSE_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCaller) MAXPAUSESLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "MAX_PAUSE_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// MAXPAUSESLOT is a free data retrieval call binding the contract method 0x1cd94ec0.
//
// Solidity: function MAX_PAUSE_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigSession) MAXPAUSESLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.MAXPAUSESLOT(&_SuperchainConfig.CallOpts)
}

// MAXPAUSESLOT is a free data retrieval call binding the contract method 0x1cd94ec0.
//
// Solidity: function MAX_PAUSE_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCallerSession) MAXPAUSESLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.MAXPAUSESLOT(&_SuperchainConfig.CallOpts)
}

// PAUSEDTIMESLOT is a free data retrieval call binding the contract method 0xb5f41ad8.
//
// Solidity: function PAUSED_TIME_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCaller) PAUSEDTIMESLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "PAUSED_TIME_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PAUSEDTIMESLOT is a free data retrieval call binding the contract method 0xb5f41ad8.
//
// Solidity: function PAUSED_TIME_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigSession) PAUSEDTIMESLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.PAUSEDTIMESLOT(&_SuperchainConfig.CallOpts)
}

// PAUSEDTIMESLOT is a free data retrieval call binding the contract method 0xb5f41ad8.
//
// Solidity: function PAUSED_TIME_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCallerSession) PAUSEDTIMESLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.PAUSEDTIMESLOT(&_SuperchainConfig.CallOpts)
}

// SYSTEMOWNERSLOT is a free data retrieval call binding the contract method 0x33239202.
//
// Solidity: function SYSTEM_OWNER_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCaller) SYSTEMOWNERSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "SYSTEM_OWNER_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// SYSTEMOWNERSLOT is a free data retrieval call binding the contract method 0x33239202.
//
// Solidity: function SYSTEM_OWNER_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigSession) SYSTEMOWNERSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.SYSTEMOWNERSLOT(&_SuperchainConfig.CallOpts)
}

// SYSTEMOWNERSLOT is a free data retrieval call binding the contract method 0x33239202.
//
// Solidity: function SYSTEM_OWNER_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCallerSession) SYSTEMOWNERSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.SYSTEMOWNERSLOT(&_SuperchainConfig.CallOpts)
}

// VETOERSLOT is a free data retrieval call binding the contract method 0x4886eb9c.
//
// Solidity: function VETOER_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCaller) VETOERSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "VETOER_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// VETOERSLOT is a free data retrieval call binding the contract method 0x4886eb9c.
//
// Solidity: function VETOER_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigSession) VETOERSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.VETOERSLOT(&_SuperchainConfig.CallOpts)
}

// VETOERSLOT is a free data retrieval call binding the contract method 0x4886eb9c.
//
// Solidity: function VETOER_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCallerSession) VETOERSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.VETOERSLOT(&_SuperchainConfig.CallOpts)
}

// AllowedSequencers is a free data retrieval call binding the contract method 0xd92a09bc.
//
// Solidity: function allowedSequencers(bytes32 ) view returns(bool)
func (_SuperchainConfig *SuperchainConfigCaller) AllowedSequencers(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "allowedSequencers", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedSequencers is a free data retrieval call binding the contract method 0xd92a09bc.
//
// Solidity: function allowedSequencers(bytes32 ) view returns(bool)
func (_SuperchainConfig *SuperchainConfigSession) AllowedSequencers(arg0 [32]byte) (bool, error) {
	return _SuperchainConfig.Contract.AllowedSequencers(&_SuperchainConfig.CallOpts, arg0)
}

// AllowedSequencers is a free data retrieval call binding the contract method 0xd92a09bc.
//
// Solidity: function allowedSequencers(bytes32 ) view returns(bool)
func (_SuperchainConfig *SuperchainConfigCallerSession) AllowedSequencers(arg0 [32]byte) (bool, error) {
	return _SuperchainConfig.Contract.AllowedSequencers(&_SuperchainConfig.CallOpts, arg0)
}

// Delay is a free data retrieval call binding the contract method 0x6a42b8f8.
//
// Solidity: function delay() view returns(uint256 delay_)
func (_SuperchainConfig *SuperchainConfigCaller) Delay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "delay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Delay is a free data retrieval call binding the contract method 0x6a42b8f8.
//
// Solidity: function delay() view returns(uint256 delay_)
func (_SuperchainConfig *SuperchainConfigSession) Delay() (*big.Int, error) {
	return _SuperchainConfig.Contract.Delay(&_SuperchainConfig.CallOpts)
}

// Delay is a free data retrieval call binding the contract method 0x6a42b8f8.
//
// Solidity: function delay() view returns(uint256 delay_)
func (_SuperchainConfig *SuperchainConfigCallerSession) Delay() (*big.Int, error) {
	return _SuperchainConfig.Contract.Delay(&_SuperchainConfig.CallOpts)
}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address guardian_)
func (_SuperchainConfig *SuperchainConfigCaller) Guardian(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "guardian")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address guardian_)
func (_SuperchainConfig *SuperchainConfigSession) Guardian() (common.Address, error) {
	return _SuperchainConfig.Contract.Guardian(&_SuperchainConfig.CallOpts)
}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address guardian_)
func (_SuperchainConfig *SuperchainConfigCallerSession) Guardian() (common.Address, error) {
	return _SuperchainConfig.Contract.Guardian(&_SuperchainConfig.CallOpts)
}

// Initiator is a free data retrieval call binding the contract method 0x5c39fcc1.
//
// Solidity: function initiator() view returns(address initiator_)
func (_SuperchainConfig *SuperchainConfigCaller) Initiator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "initiator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Initiator is a free data retrieval call binding the contract method 0x5c39fcc1.
//
// Solidity: function initiator() view returns(address initiator_)
func (_SuperchainConfig *SuperchainConfigSession) Initiator() (common.Address, error) {
	return _SuperchainConfig.Contract.Initiator(&_SuperchainConfig.CallOpts)
}

// Initiator is a free data retrieval call binding the contract method 0x5c39fcc1.
//
// Solidity: function initiator() view returns(address initiator_)
func (_SuperchainConfig *SuperchainConfigCallerSession) Initiator() (common.Address, error) {
	return _SuperchainConfig.Contract.Initiator(&_SuperchainConfig.CallOpts)
}

// IsAllowedSequencer is a free data retrieval call binding the contract method 0x76ea31a4.
//
// Solidity: function isAllowedSequencer((bytes32,address) _sequencer) view returns(bool)
func (_SuperchainConfig *SuperchainConfigCaller) IsAllowedSequencer(opts *bind.CallOpts, _sequencer TypesSequencerKeyPair) (bool, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "isAllowedSequencer", _sequencer)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAllowedSequencer is a free data retrieval call binding the contract method 0x76ea31a4.
//
// Solidity: function isAllowedSequencer((bytes32,address) _sequencer) view returns(bool)
func (_SuperchainConfig *SuperchainConfigSession) IsAllowedSequencer(_sequencer TypesSequencerKeyPair) (bool, error) {
	return _SuperchainConfig.Contract.IsAllowedSequencer(&_SuperchainConfig.CallOpts, _sequencer)
}

// IsAllowedSequencer is a free data retrieval call binding the contract method 0x76ea31a4.
//
// Solidity: function isAllowedSequencer((bytes32,address) _sequencer) view returns(bool)
func (_SuperchainConfig *SuperchainConfigCallerSession) IsAllowedSequencer(_sequencer TypesSequencerKeyPair) (bool, error) {
	return _SuperchainConfig.Contract.IsAllowedSequencer(&_SuperchainConfig.CallOpts, _sequencer)
}

// MaxPause is a free data retrieval call binding the contract method 0xa2f9c408.
//
// Solidity: function maxPause() view returns(uint256 maxPause_)
func (_SuperchainConfig *SuperchainConfigCaller) MaxPause(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "maxPause")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxPause is a free data retrieval call binding the contract method 0xa2f9c408.
//
// Solidity: function maxPause() view returns(uint256 maxPause_)
func (_SuperchainConfig *SuperchainConfigSession) MaxPause() (*big.Int, error) {
	return _SuperchainConfig.Contract.MaxPause(&_SuperchainConfig.CallOpts)
}

// MaxPause is a free data retrieval call binding the contract method 0xa2f9c408.
//
// Solidity: function maxPause() view returns(uint256 maxPause_)
func (_SuperchainConfig *SuperchainConfigCallerSession) MaxPause() (*big.Int, error) {
	return _SuperchainConfig.Contract.MaxPause(&_SuperchainConfig.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool paused_)
func (_SuperchainConfig *SuperchainConfigCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool paused_)
func (_SuperchainConfig *SuperchainConfigSession) Paused() (bool, error) {
	return _SuperchainConfig.Contract.Paused(&_SuperchainConfig.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool paused_)
func (_SuperchainConfig *SuperchainConfigCallerSession) Paused() (bool, error) {
	return _SuperchainConfig.Contract.Paused(&_SuperchainConfig.CallOpts)
}

// SystemOwner is a free data retrieval call binding the contract method 0x33779254.
//
// Solidity: function systemOwner() view returns(address systemOwner_)
func (_SuperchainConfig *SuperchainConfigCaller) SystemOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "systemOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SystemOwner is a free data retrieval call binding the contract method 0x33779254.
//
// Solidity: function systemOwner() view returns(address systemOwner_)
func (_SuperchainConfig *SuperchainConfigSession) SystemOwner() (common.Address, error) {
	return _SuperchainConfig.Contract.SystemOwner(&_SuperchainConfig.CallOpts)
}

// SystemOwner is a free data retrieval call binding the contract method 0x33779254.
//
// Solidity: function systemOwner() view returns(address systemOwner_)
func (_SuperchainConfig *SuperchainConfigCallerSession) SystemOwner() (common.Address, error) {
	return _SuperchainConfig.Contract.SystemOwner(&_SuperchainConfig.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SuperchainConfig *SuperchainConfigCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SuperchainConfig *SuperchainConfigSession) Version() (string, error) {
	return _SuperchainConfig.Contract.Version(&_SuperchainConfig.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SuperchainConfig *SuperchainConfigCallerSession) Version() (string, error) {
	return _SuperchainConfig.Contract.Version(&_SuperchainConfig.CallOpts)
}

// Vetoer is a free data retrieval call binding the contract method 0xd8bff440.
//
// Solidity: function vetoer() view returns(address vetoer_)
func (_SuperchainConfig *SuperchainConfigCaller) Vetoer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "vetoer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Vetoer is a free data retrieval call binding the contract method 0xd8bff440.
//
// Solidity: function vetoer() view returns(address vetoer_)
func (_SuperchainConfig *SuperchainConfigSession) Vetoer() (common.Address, error) {
	return _SuperchainConfig.Contract.Vetoer(&_SuperchainConfig.CallOpts)
}

// Vetoer is a free data retrieval call binding the contract method 0xd8bff440.
//
// Solidity: function vetoer() view returns(address vetoer_)
func (_SuperchainConfig *SuperchainConfigCallerSession) Vetoer() (common.Address, error) {
	return _SuperchainConfig.Contract.Vetoer(&_SuperchainConfig.CallOpts)
}

// AddSequencer is a paid mutator transaction binding the contract method 0xa0654956.
//
// Solidity: function addSequencer((bytes32,address) _sequencer) returns()
func (_SuperchainConfig *SuperchainConfigTransactor) AddSequencer(opts *bind.TransactOpts, _sequencer TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "addSequencer", _sequencer)
}

// AddSequencer is a paid mutator transaction binding the contract method 0xa0654956.
//
// Solidity: function addSequencer((bytes32,address) _sequencer) returns()
func (_SuperchainConfig *SuperchainConfigSession) AddSequencer(_sequencer TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.AddSequencer(&_SuperchainConfig.TransactOpts, _sequencer)
}

// AddSequencer is a paid mutator transaction binding the contract method 0xa0654956.
//
// Solidity: function addSequencer((bytes32,address) _sequencer) returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) AddSequencer(_sequencer TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.AddSequencer(&_SuperchainConfig.TransactOpts, _sequencer)
}

// Initialize is a paid mutator transaction binding the contract method 0x8a6fb7a3.
//
// Solidity: function initialize(address _systemOwner, address _initiator, address _vetoer, address _guardian, uint256 _delay, uint256 _maxPause, (bytes32,address)[] _sequencers) returns()
func (_SuperchainConfig *SuperchainConfigTransactor) Initialize(opts *bind.TransactOpts, _systemOwner common.Address, _initiator common.Address, _vetoer common.Address, _guardian common.Address, _delay *big.Int, _maxPause *big.Int, _sequencers []TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "initialize", _systemOwner, _initiator, _vetoer, _guardian, _delay, _maxPause, _sequencers)
}

// Initialize is a paid mutator transaction binding the contract method 0x8a6fb7a3.
//
// Solidity: function initialize(address _systemOwner, address _initiator, address _vetoer, address _guardian, uint256 _delay, uint256 _maxPause, (bytes32,address)[] _sequencers) returns()
func (_SuperchainConfig *SuperchainConfigSession) Initialize(_systemOwner common.Address, _initiator common.Address, _vetoer common.Address, _guardian common.Address, _delay *big.Int, _maxPause *big.Int, _sequencers []TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Initialize(&_SuperchainConfig.TransactOpts, _systemOwner, _initiator, _vetoer, _guardian, _delay, _maxPause, _sequencers)
}

// Initialize is a paid mutator transaction binding the contract method 0x8a6fb7a3.
//
// Solidity: function initialize(address _systemOwner, address _initiator, address _vetoer, address _guardian, uint256 _delay, uint256 _maxPause, (bytes32,address)[] _sequencers) returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) Initialize(_systemOwner common.Address, _initiator common.Address, _vetoer common.Address, _guardian common.Address, _delay *big.Int, _maxPause *big.Int, _sequencers []TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Initialize(&_SuperchainConfig.TransactOpts, _systemOwner, _initiator, _vetoer, _guardian, _delay, _maxPause, _sequencers)
}

// Pause is a paid mutator transaction binding the contract method 0x136439dd.
//
// Solidity: function pause(uint256 duration) returns()
func (_SuperchainConfig *SuperchainConfigTransactor) Pause(opts *bind.TransactOpts, duration *big.Int) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "pause", duration)
}

// Pause is a paid mutator transaction binding the contract method 0x136439dd.
//
// Solidity: function pause(uint256 duration) returns()
func (_SuperchainConfig *SuperchainConfigSession) Pause(duration *big.Int) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Pause(&_SuperchainConfig.TransactOpts, duration)
}

// Pause is a paid mutator transaction binding the contract method 0x136439dd.
//
// Solidity: function pause(uint256 duration) returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) Pause(duration *big.Int) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Pause(&_SuperchainConfig.TransactOpts, duration)
}

// RemoveSequencer is a paid mutator transaction binding the contract method 0xf1e8cf06.
//
// Solidity: function removeSequencer((bytes32,address) _sequencer) returns()
func (_SuperchainConfig *SuperchainConfigTransactor) RemoveSequencer(opts *bind.TransactOpts, _sequencer TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "removeSequencer", _sequencer)
}

// RemoveSequencer is a paid mutator transaction binding the contract method 0xf1e8cf06.
//
// Solidity: function removeSequencer((bytes32,address) _sequencer) returns()
func (_SuperchainConfig *SuperchainConfigSession) RemoveSequencer(_sequencer TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.RemoveSequencer(&_SuperchainConfig.TransactOpts, _sequencer)
}

// RemoveSequencer is a paid mutator transaction binding the contract method 0xf1e8cf06.
//
// Solidity: function removeSequencer((bytes32,address) _sequencer) returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) RemoveSequencer(_sequencer TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.RemoveSequencer(&_SuperchainConfig.TransactOpts, _sequencer)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SuperchainConfig *SuperchainConfigTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SuperchainConfig *SuperchainConfigSession) Unpause() (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Unpause(&_SuperchainConfig.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) Unpause() (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Unpause(&_SuperchainConfig.TransactOpts)
}

// SuperchainConfigConfigUpdateIterator is returned from FilterConfigUpdate and is used to iterate over the raw logs and unpacked data for ConfigUpdate events raised by the SuperchainConfig contract.
type SuperchainConfigConfigUpdateIterator struct {
	Event *SuperchainConfigConfigUpdate // Event containing the contract specifics and raw log

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
func (it *SuperchainConfigConfigUpdateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SuperchainConfigConfigUpdate)
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
		it.Event = new(SuperchainConfigConfigUpdate)
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
func (it *SuperchainConfigConfigUpdateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SuperchainConfigConfigUpdateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SuperchainConfigConfigUpdate represents a ConfigUpdate event raised by the SuperchainConfig contract.
type SuperchainConfigConfigUpdate struct {
	UpdateType uint8
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterConfigUpdate is a free log retrieval operation binding the contract event 0x7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb.
//
// Solidity: event ConfigUpdate(uint8 indexed updateType, bytes data)
func (_SuperchainConfig *SuperchainConfigFilterer) FilterConfigUpdate(opts *bind.FilterOpts, updateType []uint8) (*SuperchainConfigConfigUpdateIterator, error) {

	var updateTypeRule []interface{}
	for _, updateTypeItem := range updateType {
		updateTypeRule = append(updateTypeRule, updateTypeItem)
	}

	logs, sub, err := _SuperchainConfig.contract.FilterLogs(opts, "ConfigUpdate", updateTypeRule)
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigConfigUpdateIterator{contract: _SuperchainConfig.contract, event: "ConfigUpdate", logs: logs, sub: sub}, nil
}

// WatchConfigUpdate is a free log subscription operation binding the contract event 0x7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb.
//
// Solidity: event ConfigUpdate(uint8 indexed updateType, bytes data)
func (_SuperchainConfig *SuperchainConfigFilterer) WatchConfigUpdate(opts *bind.WatchOpts, sink chan<- *SuperchainConfigConfigUpdate, updateType []uint8) (event.Subscription, error) {

	var updateTypeRule []interface{}
	for _, updateTypeItem := range updateType {
		updateTypeRule = append(updateTypeRule, updateTypeItem)
	}

	logs, sub, err := _SuperchainConfig.contract.WatchLogs(opts, "ConfigUpdate", updateTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SuperchainConfigConfigUpdate)
				if err := _SuperchainConfig.contract.UnpackLog(event, "ConfigUpdate", log); err != nil {
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

// ParseConfigUpdate is a log parse operation binding the contract event 0x7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb.
//
// Solidity: event ConfigUpdate(uint8 indexed updateType, bytes data)
func (_SuperchainConfig *SuperchainConfigFilterer) ParseConfigUpdate(log types.Log) (*SuperchainConfigConfigUpdate, error) {
	event := new(SuperchainConfigConfigUpdate)
	if err := _SuperchainConfig.contract.UnpackLog(event, "ConfigUpdate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SuperchainConfigInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the SuperchainConfig contract.
type SuperchainConfigInitializedIterator struct {
	Event *SuperchainConfigInitialized // Event containing the contract specifics and raw log

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
func (it *SuperchainConfigInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SuperchainConfigInitialized)
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
		it.Event = new(SuperchainConfigInitialized)
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
func (it *SuperchainConfigInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SuperchainConfigInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SuperchainConfigInitialized represents a Initialized event raised by the SuperchainConfig contract.
type SuperchainConfigInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_SuperchainConfig *SuperchainConfigFilterer) FilterInitialized(opts *bind.FilterOpts) (*SuperchainConfigInitializedIterator, error) {

	logs, sub, err := _SuperchainConfig.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigInitializedIterator{contract: _SuperchainConfig.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_SuperchainConfig *SuperchainConfigFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *SuperchainConfigInitialized) (event.Subscription, error) {

	logs, sub, err := _SuperchainConfig.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SuperchainConfigInitialized)
				if err := _SuperchainConfig.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_SuperchainConfig *SuperchainConfigFilterer) ParseInitialized(log types.Log) (*SuperchainConfigInitialized, error) {
	event := new(SuperchainConfigInitialized)
	if err := _SuperchainConfig.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SuperchainConfigPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the SuperchainConfig contract.
type SuperchainConfigPausedIterator struct {
	Event *SuperchainConfigPaused // Event containing the contract specifics and raw log

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
func (it *SuperchainConfigPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SuperchainConfigPaused)
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
		it.Event = new(SuperchainConfigPaused)
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
func (it *SuperchainConfigPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SuperchainConfigPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SuperchainConfigPaused represents a Paused event raised by the SuperchainConfig contract.
type SuperchainConfigPaused struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x9e87fac88ff661f02d44f95383c817fece4bce600a3dab7a54406878b965e752.
//
// Solidity: event Paused()
func (_SuperchainConfig *SuperchainConfigFilterer) FilterPaused(opts *bind.FilterOpts) (*SuperchainConfigPausedIterator, error) {

	logs, sub, err := _SuperchainConfig.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigPausedIterator{contract: _SuperchainConfig.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x9e87fac88ff661f02d44f95383c817fece4bce600a3dab7a54406878b965e752.
//
// Solidity: event Paused()
func (_SuperchainConfig *SuperchainConfigFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *SuperchainConfigPaused) (event.Subscription, error) {

	logs, sub, err := _SuperchainConfig.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SuperchainConfigPaused)
				if err := _SuperchainConfig.contract.UnpackLog(event, "Paused", log); err != nil {
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

// ParsePaused is a log parse operation binding the contract event 0x9e87fac88ff661f02d44f95383c817fece4bce600a3dab7a54406878b965e752.
//
// Solidity: event Paused()
func (_SuperchainConfig *SuperchainConfigFilterer) ParsePaused(log types.Log) (*SuperchainConfigPaused, error) {
	event := new(SuperchainConfigPaused)
	if err := _SuperchainConfig.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SuperchainConfigUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the SuperchainConfig contract.
type SuperchainConfigUnpausedIterator struct {
	Event *SuperchainConfigUnpaused // Event containing the contract specifics and raw log

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
func (it *SuperchainConfigUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SuperchainConfigUnpaused)
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
		it.Event = new(SuperchainConfigUnpaused)
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
func (it *SuperchainConfigUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SuperchainConfigUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SuperchainConfigUnpaused represents a Unpaused event raised by the SuperchainConfig contract.
type SuperchainConfigUnpaused struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0xa45f47fdea8a1efdd9029a5691c7f759c32b7c698632b563573e155625d16933.
//
// Solidity: event Unpaused()
func (_SuperchainConfig *SuperchainConfigFilterer) FilterUnpaused(opts *bind.FilterOpts) (*SuperchainConfigUnpausedIterator, error) {

	logs, sub, err := _SuperchainConfig.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigUnpausedIterator{contract: _SuperchainConfig.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0xa45f47fdea8a1efdd9029a5691c7f759c32b7c698632b563573e155625d16933.
//
// Solidity: event Unpaused()
func (_SuperchainConfig *SuperchainConfigFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *SuperchainConfigUnpaused) (event.Subscription, error) {

	logs, sub, err := _SuperchainConfig.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SuperchainConfigUnpaused)
				if err := _SuperchainConfig.contract.UnpackLog(event, "Unpaused", log); err != nil {
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

// ParseUnpaused is a log parse operation binding the contract event 0xa45f47fdea8a1efdd9029a5691c7f759c32b7c698632b563573e155625d16933.
//
// Solidity: event Unpaused()
func (_SuperchainConfig *SuperchainConfigFilterer) ParseUnpaused(log types.Log) (*SuperchainConfigUnpaused, error) {
	event := new(SuperchainConfigUnpaused)
	if err := _SuperchainConfig.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
