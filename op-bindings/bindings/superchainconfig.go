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
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"enumSuperchainConfig.UpdateType\",\"name\":\"updateType\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"ConfigUpdate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"duration\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"identifier\",\"type\":\"string\"}],\"name\":\"PauseExtended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"duration\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"identifier\",\"type\":\"string\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DELAY_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"GUARDIAN_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"INITIATOR_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_PAUSE_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PAUSED_TIME_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"VETOER_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"batcherHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"unsafeBlockSigner\",\"type\":\"address\"}],\"internalType\":\"structTypes.SequencerKeyPair\",\"name\":\"_sequencer\",\"type\":\"tuple\"}],\"name\":\"addSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"allowedSequencers\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"delay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"delay_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"guardian\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"guardian_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_initiator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_vetoer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_guardian\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_delay\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_maxPause\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"batcherHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"unsafeBlockSigner\",\"type\":\"address\"}],\"internalType\":\"structTypes.SequencerKeyPair[]\",\"name\":\"_sequencers\",\"type\":\"tuple[]\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initiator\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"initiator_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"batcherHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"unsafeBlockSigner\",\"type\":\"address\"}],\"internalType\":\"structTypes.SequencerKeyPair\",\"name\":\"_sequencer\",\"type\":\"tuple\"}],\"name\":\"isAllowedSequencer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxPause\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxPause_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"duration\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"identifier\",\"type\":\"string\"}],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"paused_\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pausedUntil\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"paused_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"batcherHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"unsafeBlockSigner\",\"type\":\"address\"}],\"internalType\":\"structTypes.SequencerKeyPair\",\"name\":\"_sequencer\",\"type\":\"tuple\"}],\"name\":\"removeSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"systemOwner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"systemOwner_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"vetoer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"vetoer_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b506200006b600080808080806040519080825280602002602001820160405280156200006457816020015b60408051808201909152600080825260208201528152602001906001900390816200003c5790505b5062000071565b62000522565b600054600290610100900460ff1615801562000094575060005460ff8083169116105b620000fc5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b606482015260840160405180910390fd5b6000805461ffff191660ff8316176101001790556200011b87620001e1565b620001268662000277565b6200013185620002b0565b6200013c84620002e9565b620001478362000341565b60005b82518110156200019657620001818382815181106200016d576200016d62000468565b60200260200101516200037a60201b60201c565b806200018d8162000494565b9150506200014a565b506000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a150505050505050565b620002276200021260017f12c56161f16f492fd4016a16e534c3a2bcceceb7f70ec9bb75867affe3370316620004b0565b60001b826200041760201b620007801760201c565b60005b604080516001600160a01b0384166020820152600080516020620019af83398151915291015b60408051601f19818403018152908290526200026c91620004ca565b60405180910390a250565b620002a86200021260017f704ae3ec629461681409737f623e0cebb30122362e8cb04e0a0d3581d958db7d620004b0565b60016200022a565b620002e16200021260017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe69620004b0565b60026200022a565b6200031a6200021260017f0e2f5ebd54326cdea9bf943c0fc37413dccba70cdeb76374557a8f757e898390620004b0565b60035b600080516020620019af833981519152826040516020016200025091815260200190565b620003726200021260017f1399bee5471a817c3420e8d52c99ada34eb0c2eaf753dd2f4555bc879d1cd994620004b0565b60046200031d565b600062000392826200041b60201b62000db51760201c565b6000818152600160208190526040909120805460ff1916909117905590506005600080516020620019af83398151915283604051602001620003ef9190815181526020918201516001600160a01b03169181019190915260400190565b60408051601f19818403018152908290526200040b91620004ca565b60405180910390a25050565b9055565b6000816000015182602001516040516020016200044b9291909182526001600160a01b0316602082015260400190565b604051602081830303815290604052805190602001209050919050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b600060018201620004a957620004a96200047e565b5060010190565b600082821015620004c557620004c56200047e565b500390565b600060208083528351808285015260005b81811015620004f957858101830151858201604001528201620004db565b818111156200050c576000604083870101525b50601f01601f1916929092016040019392505050565b61147d80620005326000396000f3fe608060405234801561001057600080fd5b50600436106101825760003560e01c806376ea31a4116100d8578063ba605d891161008c578063d92a09bc11610066578063d92a09bc146102d6578063da748b10146102f9578063f1e8cf061461030157600080fd5b8063ba605d89146102b3578063c23a451a146102c6578063d8bff440146102ce57600080fd5b8063a0654956116100bd578063a065495614610290578063a2f9c408146102a3578063b5f41ad8146102ab57600080fd5b806376ea31a4146102755780639eb17d4b1461028857600080fd5b80634b5b189f1161013a5780635c975abb116101145780635c975abb146102425780636a42b8f81461025a5780636b2ca1631461026257600080fd5b80634b5b189f146101e957806354fd4d50146101f15780635c39fcc11461023a57600080fd5b80633f4ba83a1161016b5780633f4ba83a146101cf578063452a9320146101d95780634886eb9c146101e157600080fd5b80631cd94ec01461018757806333779254146101a2575b600080fd5b61018f610314565b6040519081526020015b60405180910390f35b6101aa610342565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610199565b6101d7610371565b005b6101aa610497565b61018f6104cb565b61018f6104f6565b61022d6040518060400160405280600581526020017f322e302e3000000000000000000000000000000000000000000000000000000081525081565b6040516101999190611088565b6101aa610521565b61024a610551565b6040519015158152602001610199565b61018f610588565b6101d7610270366004611120565b6105b8565b61024a61028336600461125d565b610859565b61018f61087e565b6101d761029e36600461125d565b6108a9565b61018f610977565b61018f6109a7565b6101d76102c1366004611279565b6109d2565b61018f610b86565b6101aa610bb1565b61024a6102e4366004611371565b60016020526000908152604090205460ff1681565b61018f610be1565b6101d761030f36600461125d565b610c11565b61033f60017f1399bee5471a817c3420e8d52c99ada34eb0c2eaf753dd2f4555bc879d1cd9946113b9565b81565b600061036c7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61035490565b905090565b610379610497565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610438576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f5375706572636861696e436f6e6669673a206f6e6c7920677561726469616e2060448201527f63616e20756e706175736500000000000000000000000000000000000000000060648201526084015b60405180910390fd5b61046c61046660017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b76113b9565b60009055565b6040517fa45f47fdea8a1efdd9029a5691c7f759c32b7c698632b563573e155625d1693390600090a1565b600061036c6104c760017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe696113b9565b5490565b61033f60017f704ae3ec629461681409737f623e0cebb30122362e8cb04e0a0d3581d958db7d6113b9565b61033f60017f12c56161f16f492fd4016a16e534c3a2bcceceb7f70ec9bb75867affe33703166113b9565b600061036c6104c760017f12c56161f16f492fd4016a16e534c3a2bcceceb7f70ec9bb75867affe33703166113b9565b6000426105826104c760017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b76113b9565b11905090565b600061036c6104c760017f0e2f5ebd54326cdea9bf943c0fc37413dccba70cdeb76374557a8f757e8983906113b9565b6105c0610497565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161461067a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602960248201527f5375706572636861696e436f6e6669673a206f6e6c7920677561726469616e2060448201527f63616e2070617573650000000000000000000000000000000000000000000000606482015260840161042f565b6106a86104c760017f1399bee5471a817c3420e8d52c99ada34eb0c2eaf753dd2f4555bc879d1cd9946113b9565b821115610737576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f5375706572636861696e436f6e6669673a206475726174696f6e20657863656560448201527f6473206d61785061757365000000000000000000000000000000000000000000606482015260840161042f565b61073f610551565b15156000036107c15761078461077660017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b76113b9565b61078084426113d0565b9055565b7fefbb713a829fa70ddb05ecac01512a81b393a83dcba75fd9a3f72ebc2dd1a13782826040516107b59291906113e8565b60405180910390a15050565b6108286107ef60017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b76113b9565b8361081e6104c760017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b76113b9565b61078091906113d0565b7f88e8ad654c0f119ace7d7870c65d03eeef4a7bde33d5d78910fce8dba91e055e82826040516107b59291906113e8565b60008061086583610db5565b60009081526001602052604090205460ff169392505050565b61033f60017f0e2f5ebd54326cdea9bf943c0fc37413dccba70cdeb76374557a8f757e8983906113b9565b6108b1610521565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161461096b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603260248201527f5375706572636861696e436f6e6669673a206f6e6c7920696e69746961746f7260448201527f2063616e206164642073657175656e6365720000000000000000000000000000606482015260840161042f565b61097481610e0e565b50565b600061036c6104c760017f1399bee5471a817c3420e8d52c99ada34eb0c2eaf753dd2f4555bc879d1cd9946113b9565b61033f60017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b76113b9565b600054600290610100900460ff161580156109f4575060005460ff8083169116105b610a80576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a6564000000000000000000000000000000000000606482015260840161042f565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff831617610100179055610aba87610e5c565b610ac386610f19565b610acc85610f4e565b610ad584610f83565b610ade83610fe8565b60005b8251811015610b1e57610b0c838281518110610aff57610aff611409565b6020026020010151610e0e565b80610b1681611438565b915050610ae1565b50600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a150505050505050565b61033f60017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe696113b9565b600061036c6104c760017f704ae3ec629461681409737f623e0cebb30122362e8cb04e0a0d3581d958db7d6113b9565b600061036c6104c760017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b76113b9565b610c19610342565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610cd3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603960248201527f5375706572636861696e436f6e6669673a206f6e6c792073797374656d4f776e60448201527f65722063616e2072656d6f766520612073657175656e63657200000000000000606482015260840161042f565b6000610cde82610db5565b600081815260016020526040902080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00169055905060065b7f7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb83604051602001610d7191908151815260209182015173ffffffffffffffffffffffffffffffffffffffff169181019190915260400190565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815290829052610da991611088565b60405180910390a25050565b600081600001518260200151604051602001610df192919091825273ffffffffffffffffffffffffffffffffffffffff16602082015260400190565b604051602081830303815290604052805190602001209050919050565b6000610e1982610db5565b600081815260016020819052604090912080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016909117905590506005610d17565b610e8f610e8a60017f12c56161f16f492fd4016a16e534c3a2bcceceb7f70ec9bb75867affe33703166113b9565b829055565b60005b6040805173ffffffffffffffffffffffffffffffffffffffff841660208201527f7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb91015b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815290829052610f0e91611088565b60405180910390a250565b610f47610e8a60017f704ae3ec629461681409737f623e0cebb30122362e8cb04e0a0d3581d958db7d6113b9565b6001610e92565b610f7c610e8a60017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe696113b9565b6002610e92565b610fb1610e8a60017f0e2f5ebd54326cdea9bf943c0fc37413dccba70cdeb76374557a8f757e8983906113b9565b60035b7f7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb82604051602001610ed691815260200190565b611016610e8a60017f1399bee5471a817c3420e8d52c99ada34eb0c2eaf753dd2f4555bc879d1cd9946113b9565b6004610fb4565b6000815180845260005b8181101561104357602081850181015186830182015201611027565b81811115611055576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061109b602083018461101d565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff81118282101715611118576111186110a2565b604052919050565b6000806040838503121561113357600080fd5b8235915060208084013567ffffffffffffffff8082111561115357600080fd5b818601915086601f83011261116757600080fd5b813581811115611179576111796110a2565b6111a9847fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116016110d1565b915080825287848285010111156111bf57600080fd5b80848401858401376000848284010152508093505050509250929050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461120157600080fd5b919050565b60006040828403121561121857600080fd5b6040516040810181811067ffffffffffffffff8211171561123b5761123b6110a2565b60405282358152905080611251602084016111dd565b60208201525092915050565b60006040828403121561126f57600080fd5b61109b8383611206565b60008060008060008060c0878903121561129257600080fd5b61129b876111dd565b955060206112aa8189016111dd565b955060406112b9818a016111dd565b9550606089013594506080890135935060a089013567ffffffffffffffff808211156112e457600080fd5b818b0191508b601f8301126112f857600080fd5b81358181111561130a5761130a6110a2565b611318858260051b016110d1565b818152858101925060069190911b83018501908d82111561133857600080fd5b928501925b8184101561135e5761134f8e85611206565b8352928401929185019161133d565b8096505050505050509295509295509295565b60006020828403121561138357600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000828210156113cb576113cb61138a565b500390565b600082198211156113e3576113e361138a565b500190565b828152604060208201526000611401604083018461101d565b949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036114695761146961138a565b506001019056fea164736f6c634300080f000a7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb",
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

// PausedUntil is a free data retrieval call binding the contract method 0xda748b10.
//
// Solidity: function pausedUntil() view returns(uint256 paused_)
func (_SuperchainConfig *SuperchainConfigCaller) PausedUntil(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "pausedUntil")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PausedUntil is a free data retrieval call binding the contract method 0xda748b10.
//
// Solidity: function pausedUntil() view returns(uint256 paused_)
func (_SuperchainConfig *SuperchainConfigSession) PausedUntil() (*big.Int, error) {
	return _SuperchainConfig.Contract.PausedUntil(&_SuperchainConfig.CallOpts)
}

// PausedUntil is a free data retrieval call binding the contract method 0xda748b10.
//
// Solidity: function pausedUntil() view returns(uint256 paused_)
func (_SuperchainConfig *SuperchainConfigCallerSession) PausedUntil() (*big.Int, error) {
	return _SuperchainConfig.Contract.PausedUntil(&_SuperchainConfig.CallOpts)
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

// Initialize is a paid mutator transaction binding the contract method 0xba605d89.
//
// Solidity: function initialize(address _initiator, address _vetoer, address _guardian, uint256 _delay, uint256 _maxPause, (bytes32,address)[] _sequencers) returns()
func (_SuperchainConfig *SuperchainConfigTransactor) Initialize(opts *bind.TransactOpts, _initiator common.Address, _vetoer common.Address, _guardian common.Address, _delay *big.Int, _maxPause *big.Int, _sequencers []TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "initialize", _initiator, _vetoer, _guardian, _delay, _maxPause, _sequencers)
}

// Initialize is a paid mutator transaction binding the contract method 0xba605d89.
//
// Solidity: function initialize(address _initiator, address _vetoer, address _guardian, uint256 _delay, uint256 _maxPause, (bytes32,address)[] _sequencers) returns()
func (_SuperchainConfig *SuperchainConfigSession) Initialize(_initiator common.Address, _vetoer common.Address, _guardian common.Address, _delay *big.Int, _maxPause *big.Int, _sequencers []TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Initialize(&_SuperchainConfig.TransactOpts, _initiator, _vetoer, _guardian, _delay, _maxPause, _sequencers)
}

// Initialize is a paid mutator transaction binding the contract method 0xba605d89.
//
// Solidity: function initialize(address _initiator, address _vetoer, address _guardian, uint256 _delay, uint256 _maxPause, (bytes32,address)[] _sequencers) returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) Initialize(_initiator common.Address, _vetoer common.Address, _guardian common.Address, _delay *big.Int, _maxPause *big.Int, _sequencers []TypesSequencerKeyPair) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Initialize(&_SuperchainConfig.TransactOpts, _initiator, _vetoer, _guardian, _delay, _maxPause, _sequencers)
}

// Pause is a paid mutator transaction binding the contract method 0x6b2ca163.
//
// Solidity: function pause(uint256 duration, string identifier) returns()
func (_SuperchainConfig *SuperchainConfigTransactor) Pause(opts *bind.TransactOpts, duration *big.Int, identifier string) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "pause", duration, identifier)
}

// Pause is a paid mutator transaction binding the contract method 0x6b2ca163.
//
// Solidity: function pause(uint256 duration, string identifier) returns()
func (_SuperchainConfig *SuperchainConfigSession) Pause(duration *big.Int, identifier string) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Pause(&_SuperchainConfig.TransactOpts, duration, identifier)
}

// Pause is a paid mutator transaction binding the contract method 0x6b2ca163.
//
// Solidity: function pause(uint256 duration, string identifier) returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) Pause(duration *big.Int, identifier string) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Pause(&_SuperchainConfig.TransactOpts, duration, identifier)
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

// SuperchainConfigPauseExtendedIterator is returned from FilterPauseExtended and is used to iterate over the raw logs and unpacked data for PauseExtended events raised by the SuperchainConfig contract.
type SuperchainConfigPauseExtendedIterator struct {
	Event *SuperchainConfigPauseExtended // Event containing the contract specifics and raw log

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
func (it *SuperchainConfigPauseExtendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SuperchainConfigPauseExtended)
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
		it.Event = new(SuperchainConfigPauseExtended)
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
func (it *SuperchainConfigPauseExtendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SuperchainConfigPauseExtendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SuperchainConfigPauseExtended represents a PauseExtended event raised by the SuperchainConfig contract.
type SuperchainConfigPauseExtended struct {
	Duration   *big.Int
	Identifier string
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterPauseExtended is a free log retrieval operation binding the contract event 0x88e8ad654c0f119ace7d7870c65d03eeef4a7bde33d5d78910fce8dba91e055e.
//
// Solidity: event PauseExtended(uint256 duration, string identifier)
func (_SuperchainConfig *SuperchainConfigFilterer) FilterPauseExtended(opts *bind.FilterOpts) (*SuperchainConfigPauseExtendedIterator, error) {

	logs, sub, err := _SuperchainConfig.contract.FilterLogs(opts, "PauseExtended")
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigPauseExtendedIterator{contract: _SuperchainConfig.contract, event: "PauseExtended", logs: logs, sub: sub}, nil
}

// WatchPauseExtended is a free log subscription operation binding the contract event 0x88e8ad654c0f119ace7d7870c65d03eeef4a7bde33d5d78910fce8dba91e055e.
//
// Solidity: event PauseExtended(uint256 duration, string identifier)
func (_SuperchainConfig *SuperchainConfigFilterer) WatchPauseExtended(opts *bind.WatchOpts, sink chan<- *SuperchainConfigPauseExtended) (event.Subscription, error) {

	logs, sub, err := _SuperchainConfig.contract.WatchLogs(opts, "PauseExtended")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SuperchainConfigPauseExtended)
				if err := _SuperchainConfig.contract.UnpackLog(event, "PauseExtended", log); err != nil {
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

// ParsePauseExtended is a log parse operation binding the contract event 0x88e8ad654c0f119ace7d7870c65d03eeef4a7bde33d5d78910fce8dba91e055e.
//
// Solidity: event PauseExtended(uint256 duration, string identifier)
func (_SuperchainConfig *SuperchainConfigFilterer) ParsePauseExtended(log types.Log) (*SuperchainConfigPauseExtended, error) {
	event := new(SuperchainConfigPauseExtended)
	if err := _SuperchainConfig.contract.UnpackLog(event, "PauseExtended", log); err != nil {
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
	Duration   *big.Int
	Identifier string
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0xefbb713a829fa70ddb05ecac01512a81b393a83dcba75fd9a3f72ebc2dd1a137.
//
// Solidity: event Paused(uint256 duration, string identifier)
func (_SuperchainConfig *SuperchainConfigFilterer) FilterPaused(opts *bind.FilterOpts) (*SuperchainConfigPausedIterator, error) {

	logs, sub, err := _SuperchainConfig.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigPausedIterator{contract: _SuperchainConfig.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0xefbb713a829fa70ddb05ecac01512a81b393a83dcba75fd9a3f72ebc2dd1a137.
//
// Solidity: event Paused(uint256 duration, string identifier)
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

// ParsePaused is a log parse operation binding the contract event 0xefbb713a829fa70ddb05ecac01512a81b393a83dcba75fd9a3f72ebc2dd1a137.
//
// Solidity: event Paused(uint256 duration, string identifier)
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
