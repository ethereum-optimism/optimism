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

// L1DomiconCommitmentMetaData contains all meta data concerning the L1DomiconCommitment contract.
var L1DomiconCommitmentMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"broadcaster\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"sign\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"}],\"name\":\"FinalizeSubmitCommitment\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"broadcaster\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"sign\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"}],\"name\":\"SendDACommitment\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOM\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DOMICON_NODE\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MESSENGER\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OTHER_COMMITMENT\",\"outputs\":[{\"internalType\":\"contractDomiconCommitment\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"SetDom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_index\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_length\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_price\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_sign\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_commitment\",\"type\":\"bytes\"}],\"name\":\"SubmitCommitment\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"domiconNode\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_length\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_price\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_broadcaster\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_sign\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_commitment\",\"type\":\"bytes\"}],\"name\":\"finalizeSubmitCommitment\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"indices\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"_messenger\",\"type\":\"address\"},{\"internalType\":\"contractDomiconNode\",\"name\":\"_node\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"otherCommitment\",\"outputs\":[{\"internalType\":\"contractDomiconCommitment\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"submits\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a0604052600280546001600160a01b03191673779877a7b0d9e8603169ddbd7836e478b46247891790553480156200003757600080fd5b507342000000000000000000000000000000000000226080526200005d60008062000063565b620001f0565b600054600390610100900460ff1615801562000086575060005460ff8083169116105b620000ef5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805461ffff191660ff8316176101001790556200010f838362000155565b6000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a1505050565b600054610100900460ff16620001c25760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201526a6e697469616c697a696e6760a81b6064820152608401620000e6565b600580546001600160a01b039384166001600160a01b03199182161790915560068054929093169116179055565b608051611c08620002216000396000818161035c015281816103920152818161056a0152610de10152611c086000f3fe6080604052600436106100dd5760003560e01c80636d8819891161007f578063927ede2d11610059578063927ede2d14610302578063dcf36d571461032d578063e996e9ac1461034d578063fce1c9741461038057600080fd5b80636d8819891461026d578063777109f8146102cf5780638ee1d239146102e257600080fd5b80635063e207116100bb5780635063e2071461018257806354fd4d50146101bd5780635fa4ad36146102135780636a57f6b11461024057600080fd5b80633817ce86146100e25780633cb747bf14610133578063485cc95514610160575b600080fd5b3480156100ee57600080fd5b5060065473ffffffffffffffffffffffffffffffffffffffff165b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b34801561013f57600080fd5b506005546101099073ffffffffffffffffffffffffffffffffffffffff1681565b34801561016c57600080fd5b5061018061017b366004611466565b6103b4565b005b34801561018e57600080fd5b506101af61019d36600461149f565b60046020526000908152604090205481565b60405190815260200161012a565b3480156101c957600080fd5b506102066040518060400160405280600581526020017f312e342e3100000000000000000000000000000000000000000000000000000081525081565b60405161012a9190611532565b34801561021f57600080fd5b506006546101099073ffffffffffffffffffffffffffffffffffffffff1681565b34801561024c57600080fd5b506002546101099073ffffffffffffffffffffffffffffffffffffffff1681565b34801561027957600080fd5b5061018061028836600461149f565b600280547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b6101806102dd36600461158e565b610505565b3480156102ee57600080fd5b506101806102fd36600461165f565b61075e565b34801561030e57600080fd5b5060055473ffffffffffffffffffffffffffffffffffffffff16610109565b34801561033957600080fd5b50610206610348366004611713565b610af7565b34801561035957600080fd5b507f0000000000000000000000000000000000000000000000000000000000000000610109565b34801561038c57600080fd5b506101097f000000000000000000000000000000000000000000000000000000000000000081565b600054600390610100900460ff161580156103d6575060005460ff8083169116105b610467576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084015b60405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff8316176101001790556104a28383610b9c565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a1505050565b60055473ffffffffffffffffffffffffffffffffffffffff16331480156105f45750600554604080517f6e296e45000000000000000000000000000000000000000000000000000000008152905173ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000008116931691636e296e459160048083019260209291908290030181865afa1580156105b8573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105dc919061173f565b73ffffffffffffffffffffffffffffffffffffffff16145b6106a6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f446f6d69636f6e436f6d6d69746d656e743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c65642066726f6d20746865206f7468657220636f60648201527f6d6d69746d656e74000000000000000000000000000000000000000000000000608482015260a40161045e565b73ffffffffffffffffffffffffffffffffffffffff851660009081526003602090815260408083208c845290915290206106e182848361182c565b508473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff167f9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea8b8b8b8989898960405161074b9796959493929190611990565b60405180910390a3505050505050505050565b333b156107ed576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f446f6d69636f6e436f6d6d69746d656e743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c65642066726f6d20616e20454f41000000000000606482015260840161045e565b6006546040517fc0f2acea00000000000000000000000000000000000000000000000000000000815233600482015273ffffffffffffffffffffffffffffffffffffffff9091169063c0f2acea90602401602060405180830381865afa15801561085b573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061087f91906119c9565b61090b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f446f6d69636f6e436f6d6d69746d656e743a2062726f616463617374206e6f6460448201527f652061646472657373206572726f720000000000000000000000000000000000606482015260840161045e565b61091b85878a8a88888888610c86565b6109a7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f4c31446f6d69636f6e436f6d6d69746d656e743a696e76616c6964205369676e60448201527f6174757265000000000000000000000000000000000000000000000000000000606482015260840161045e565b6002546109cd9073ffffffffffffffffffffffffffffffffffffffff16863060c8610d1f565b73ffffffffffffffffffffffffffffffffffffffff8516600090815260036020908152604080832067ffffffffffffffff8c1684529091529020610a1282848361182c565b5073ffffffffffffffffffffffffffffffffffffffff85166000908152600460205260408120805491610a44836119eb565b91905055508473ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fce7c513598cb8cde5f5798356c301e6ed9a2889048d0cb2e36504b5f9c85d90e8a8a8a89898989604051610ab29796959493929190611a4a565b60405180910390a3610aed62030d408967ffffffffffffffff168967ffffffffffffffff168967ffffffffffffffff16338a8a8a8a8a610dba565b5050505050505050565b600360209081526000928352604080842090915290825290208054610b1b9061178b565b80601f0160208091040260200160405190810160405280929190818152602001828054610b479061178b565b8015610b945780601f10610b6957610100808354040283529160200191610b94565b820191906000526020600020905b815481529060010190602001808311610b7757829003601f168201915b505050505081565b600054610100900460ff16610c33576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161045e565b6005805473ffffffffffffffffffffffffffffffffffffffff9384167fffffffffffffffffffffffff00000000000000000000000000000000000000009182161790915560068054929093169116179055565b600080610ccd8a338b8b8b89898080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250610f1292505050565b9050610d118187878080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152508f9250610f7a915050565b9a9950505050505050505050565b6040805173ffffffffffffffffffffffffffffffffffffffff85811660248301528416604482015260648082018490528251808303909101815260849091019091526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f23b872dd00000000000000000000000000000000000000000000000000000000179052610db4908590611130565b50505050565b60055460405173ffffffffffffffffffffffffffffffffffffffff9091169063b8920c14907f0000000000000000000000000000000000000000000000000000000000000000907f777109f80000000000000000000000000000000000000000000000000000000090610e41908e908e908e908e908e908e908e908e908e90602401611a80565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e085901b9092168252610ed492918f90600401611aef565b600060405180830381600087803b158015610eee57600080fd5b505af1158015610f02573d6000803e3d6000fd5b5050505050505050505050505050565b600080469050600081898989898989604051602001610f379796959493929190611b34565b604080518083037fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe001815291905280516020909101209998505050505050505050565b60008251604114610fe7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601860248201527f496e76616c6964207369676e6174757265206c656e6774680000000000000000604482015260640161045e565b602083810151604080860151606080880151835160008082529681018086528b905290861a938101849052908101849052608081018290529193909160019060a0016020604051602081039080840390855afa15801561104b573d6000803e3d6000fd5b50506040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0015191505073ffffffffffffffffffffffffffffffffffffffff81166110f3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601660248201527f61646472657373206973206e6f742061766169626c6500000000000000000000604482015260640161045e565b8573ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16149450505050505b9392505050565b6000611192826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c65648152508573ffffffffffffffffffffffffffffffffffffffff166112419092919063ffffffff16565b80519091501561123c57808060200190518101906111b091906119c9565b61123c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f74207375636365656400000000000000000000000000000000000000000000606482015260840161045e565b505050565b60606112508484600085611258565b949350505050565b6060824710156112ea576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c0000000000000000000000000000000000000000000000000000606482015260840161045e565b73ffffffffffffffffffffffffffffffffffffffff85163b611368576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e7472616374000000604482015260640161045e565b6000808673ffffffffffffffffffffffffffffffffffffffff1685876040516113919190611bdf565b60006040518083038185875af1925050503d80600081146113ce576040519150601f19603f3d011682016040523d82523d6000602084013e6113d3565b606091505b50915091506113e38282866113ee565b979650505050505050565b606083156113fd575081611129565b82511561140d5782518084602001fd5b816040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161045e9190611532565b73ffffffffffffffffffffffffffffffffffffffff8116811461146357600080fd5b50565b6000806040838503121561147957600080fd5b823561148481611441565b9150602083013561149481611441565b809150509250929050565b6000602082840312156114b157600080fd5b813561112981611441565b60005b838110156114d75781810151838201526020016114bf565b83811115610db45750506000910152565b600081518084526115008160208601602086016114bc565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061112960208301846114e8565b60008083601f84011261155757600080fd5b50813567ffffffffffffffff81111561156f57600080fd5b60208301915083602082850101111561158757600080fd5b9250929050565b600080600080600080600080600060e08a8c0312156115ac57600080fd5b8935985060208a0135975060408a0135965060608a01356115cc81611441565b955060808a01356115dc81611441565b945060a08a013567ffffffffffffffff808211156115f957600080fd5b6116058d838e01611545565b909650945060c08c013591508082111561161e57600080fd5b5061162b8c828d01611545565b915080935050809150509295985092959850929598565b803567ffffffffffffffff8116811461165a57600080fd5b919050565b60008060008060008060008060c0898b03121561167b57600080fd5b61168489611642565b975061169260208a01611642565b96506116a060408a01611642565b955060608901356116b081611441565b9450608089013567ffffffffffffffff808211156116cd57600080fd5b6116d98c838d01611545565b909650945060a08b01359150808211156116f257600080fd5b506116ff8b828c01611545565b999c989b5096995094979396929594505050565b6000806040838503121561172657600080fd5b823561173181611441565b946020939093013593505050565b60006020828403121561175157600080fd5b815161112981611441565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600181811c9082168061179f57607f821691505b6020821081036117d8577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b601f82111561123c57600081815260208120601f850160051c810160208610156118055750805b601f850160051c820191505b8181101561182457828155600101611811565b505050505050565b67ffffffffffffffff8311156118445761184461175c565b61185883611852835461178b565b836117de565b6000601f8411600181146118aa57600085156118745750838201355b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600387901b1c1916600186901b178355611940565b6000838152602090207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0861690835b828110156118f957868501358255602094850194600190920191016118d9565b5086821015611934577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60f88860031b161c19848701351681555b505060018560011b0183555b5050505050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b87815286602082015285604082015260a0606082015260006119b660a083018688611947565b8281036080840152610d11818587611947565b6000602082840312156119db57600080fd5b8151801515811461112957600080fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203611a43577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b5060010190565b600067ffffffffffffffff808a168352808916602084015280881660408401525060a060608301526119b660a083018688611947565b898152886020820152876040820152600073ffffffffffffffffffffffffffffffffffffffff808916606084015280881660808401525060e060a0830152611acc60e083018688611947565b82810360c0840152611adf818587611947565b9c9b505050505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff84168152606060208201526000611b1e60608301856114e8565b905063ffffffff83166040830152949350505050565b60007fffffffffffffffff000000000000000000000000000000000000000000000000808a60c01b1683527fffffffffffffffffffffffffffffffffffffffff000000000000000000000000808a60601b166008850152808960601b16601c85015250808760c01b166030840152808660c01b166038840152808560c01b166040840152508251611bcc8160488501602087016114bc565b9190910160480198975050505050505050565b60008251611bf18184602087016114bc565b919091019291505056fea164736f6c634300080f000a",
}

// L1DomiconCommitmentABI is the input ABI used to generate the binding from.
// Deprecated: Use L1DomiconCommitmentMetaData.ABI instead.
var L1DomiconCommitmentABI = L1DomiconCommitmentMetaData.ABI

// L1DomiconCommitmentBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1DomiconCommitmentMetaData.Bin instead.
var L1DomiconCommitmentBin = L1DomiconCommitmentMetaData.Bin

// DeployL1DomiconCommitment deploys a new Ethereum contract, binding an instance of L1DomiconCommitment to it.
func DeployL1DomiconCommitment(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L1DomiconCommitment, error) {
	parsed, err := L1DomiconCommitmentMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1DomiconCommitmentBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1DomiconCommitment{L1DomiconCommitmentCaller: L1DomiconCommitmentCaller{contract: contract}, L1DomiconCommitmentTransactor: L1DomiconCommitmentTransactor{contract: contract}, L1DomiconCommitmentFilterer: L1DomiconCommitmentFilterer{contract: contract}}, nil
}

// L1DomiconCommitment is an auto generated Go binding around an Ethereum contract.
type L1DomiconCommitment struct {
	L1DomiconCommitmentCaller     // Read-only binding to the contract
	L1DomiconCommitmentTransactor // Write-only binding to the contract
	L1DomiconCommitmentFilterer   // Log filterer for contract events
}

// L1DomiconCommitmentCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1DomiconCommitmentCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1DomiconCommitmentTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1DomiconCommitmentTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1DomiconCommitmentFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1DomiconCommitmentFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1DomiconCommitmentSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1DomiconCommitmentSession struct {
	Contract     *L1DomiconCommitment // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// L1DomiconCommitmentCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1DomiconCommitmentCallerSession struct {
	Contract *L1DomiconCommitmentCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// L1DomiconCommitmentTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1DomiconCommitmentTransactorSession struct {
	Contract     *L1DomiconCommitmentTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// L1DomiconCommitmentRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1DomiconCommitmentRaw struct {
	Contract *L1DomiconCommitment // Generic contract binding to access the raw methods on
}

// L1DomiconCommitmentCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1DomiconCommitmentCallerRaw struct {
	Contract *L1DomiconCommitmentCaller // Generic read-only contract binding to access the raw methods on
}

// L1DomiconCommitmentTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1DomiconCommitmentTransactorRaw struct {
	Contract *L1DomiconCommitmentTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1DomiconCommitment creates a new instance of L1DomiconCommitment, bound to a specific deployed contract.
func NewL1DomiconCommitment(address common.Address, backend bind.ContractBackend) (*L1DomiconCommitment, error) {
	contract, err := bindL1DomiconCommitment(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitment{L1DomiconCommitmentCaller: L1DomiconCommitmentCaller{contract: contract}, L1DomiconCommitmentTransactor: L1DomiconCommitmentTransactor{contract: contract}, L1DomiconCommitmentFilterer: L1DomiconCommitmentFilterer{contract: contract}}, nil
}

// NewL1DomiconCommitmentCaller creates a new read-only instance of L1DomiconCommitment, bound to a specific deployed contract.
func NewL1DomiconCommitmentCaller(address common.Address, caller bind.ContractCaller) (*L1DomiconCommitmentCaller, error) {
	contract, err := bindL1DomiconCommitment(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentCaller{contract: contract}, nil
}

// NewL1DomiconCommitmentTransactor creates a new write-only instance of L1DomiconCommitment, bound to a specific deployed contract.
func NewL1DomiconCommitmentTransactor(address common.Address, transactor bind.ContractTransactor) (*L1DomiconCommitmentTransactor, error) {
	contract, err := bindL1DomiconCommitment(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentTransactor{contract: contract}, nil
}

// NewL1DomiconCommitmentFilterer creates a new log filterer instance of L1DomiconCommitment, bound to a specific deployed contract.
func NewL1DomiconCommitmentFilterer(address common.Address, filterer bind.ContractFilterer) (*L1DomiconCommitmentFilterer, error) {
	contract, err := bindL1DomiconCommitment(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentFilterer{contract: contract}, nil
}

// bindL1DomiconCommitment binds a generic wrapper to an already deployed contract.
func bindL1DomiconCommitment(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := L1DomiconCommitmentMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1DomiconCommitment *L1DomiconCommitmentRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1DomiconCommitment.Contract.L1DomiconCommitmentCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1DomiconCommitment *L1DomiconCommitmentRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.L1DomiconCommitmentTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1DomiconCommitment *L1DomiconCommitmentRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.L1DomiconCommitmentTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1DomiconCommitment *L1DomiconCommitmentCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1DomiconCommitment.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.contract.Transact(opts, method, params...)
}

// DOM is a free data retrieval call binding the contract method 0x6a57f6b1.
//
// Solidity: function DOM() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) DOM(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "DOM")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DOM is a free data retrieval call binding the contract method 0x6a57f6b1.
//
// Solidity: function DOM() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) DOM() (common.Address, error) {
	return _L1DomiconCommitment.Contract.DOM(&_L1DomiconCommitment.CallOpts)
}

// DOM is a free data retrieval call binding the contract method 0x6a57f6b1.
//
// Solidity: function DOM() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) DOM() (common.Address, error) {
	return _L1DomiconCommitment.Contract.DOM(&_L1DomiconCommitment.CallOpts)
}

// DOMICONNODE is a free data retrieval call binding the contract method 0x3817ce86.
//
// Solidity: function DOMICON_NODE() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) DOMICONNODE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "DOMICON_NODE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DOMICONNODE is a free data retrieval call binding the contract method 0x3817ce86.
//
// Solidity: function DOMICON_NODE() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) DOMICONNODE() (common.Address, error) {
	return _L1DomiconCommitment.Contract.DOMICONNODE(&_L1DomiconCommitment.CallOpts)
}

// DOMICONNODE is a free data retrieval call binding the contract method 0x3817ce86.
//
// Solidity: function DOMICON_NODE() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) DOMICONNODE() (common.Address, error) {
	return _L1DomiconCommitment.Contract.DOMICONNODE(&_L1DomiconCommitment.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) MESSENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "MESSENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) MESSENGER() (common.Address, error) {
	return _L1DomiconCommitment.Contract.MESSENGER(&_L1DomiconCommitment.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) MESSENGER() (common.Address, error) {
	return _L1DomiconCommitment.Contract.MESSENGER(&_L1DomiconCommitment.CallOpts)
}

// OTHERCOMMITMENT is a free data retrieval call binding the contract method 0xfce1c974.
//
// Solidity: function OTHER_COMMITMENT() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) OTHERCOMMITMENT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "OTHER_COMMITMENT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OTHERCOMMITMENT is a free data retrieval call binding the contract method 0xfce1c974.
//
// Solidity: function OTHER_COMMITMENT() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) OTHERCOMMITMENT() (common.Address, error) {
	return _L1DomiconCommitment.Contract.OTHERCOMMITMENT(&_L1DomiconCommitment.CallOpts)
}

// OTHERCOMMITMENT is a free data retrieval call binding the contract method 0xfce1c974.
//
// Solidity: function OTHER_COMMITMENT() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) OTHERCOMMITMENT() (common.Address, error) {
	return _L1DomiconCommitment.Contract.OTHERCOMMITMENT(&_L1DomiconCommitment.CallOpts)
}

// DomiconNode is a free data retrieval call binding the contract method 0x5fa4ad36.
//
// Solidity: function domiconNode() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) DomiconNode(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "domiconNode")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DomiconNode is a free data retrieval call binding the contract method 0x5fa4ad36.
//
// Solidity: function domiconNode() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) DomiconNode() (common.Address, error) {
	return _L1DomiconCommitment.Contract.DomiconNode(&_L1DomiconCommitment.CallOpts)
}

// DomiconNode is a free data retrieval call binding the contract method 0x5fa4ad36.
//
// Solidity: function domiconNode() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) DomiconNode() (common.Address, error) {
	return _L1DomiconCommitment.Contract.DomiconNode(&_L1DomiconCommitment.CallOpts)
}

// Indices is a free data retrieval call binding the contract method 0x5063e207.
//
// Solidity: function indices(address ) view returns(uint256)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) Indices(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "indices", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Indices is a free data retrieval call binding the contract method 0x5063e207.
//
// Solidity: function indices(address ) view returns(uint256)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) Indices(arg0 common.Address) (*big.Int, error) {
	return _L1DomiconCommitment.Contract.Indices(&_L1DomiconCommitment.CallOpts, arg0)
}

// Indices is a free data retrieval call binding the contract method 0x5063e207.
//
// Solidity: function indices(address ) view returns(uint256)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) Indices(arg0 common.Address) (*big.Int, error) {
	return _L1DomiconCommitment.Contract.Indices(&_L1DomiconCommitment.CallOpts, arg0)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) Messenger() (common.Address, error) {
	return _L1DomiconCommitment.Contract.Messenger(&_L1DomiconCommitment.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) Messenger() (common.Address, error) {
	return _L1DomiconCommitment.Contract.Messenger(&_L1DomiconCommitment.CallOpts)
}

// OtherCommitment is a free data retrieval call binding the contract method 0xe996e9ac.
//
// Solidity: function otherCommitment() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) OtherCommitment(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "otherCommitment")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OtherCommitment is a free data retrieval call binding the contract method 0xe996e9ac.
//
// Solidity: function otherCommitment() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) OtherCommitment() (common.Address, error) {
	return _L1DomiconCommitment.Contract.OtherCommitment(&_L1DomiconCommitment.CallOpts)
}

// OtherCommitment is a free data retrieval call binding the contract method 0xe996e9ac.
//
// Solidity: function otherCommitment() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) OtherCommitment() (common.Address, error) {
	return _L1DomiconCommitment.Contract.OtherCommitment(&_L1DomiconCommitment.CallOpts)
}

// Submits is a free data retrieval call binding the contract method 0xdcf36d57.
//
// Solidity: function submits(address , uint256 ) view returns(bytes)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) Submits(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) ([]byte, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "submits", arg0, arg1)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// Submits is a free data retrieval call binding the contract method 0xdcf36d57.
//
// Solidity: function submits(address , uint256 ) view returns(bytes)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) Submits(arg0 common.Address, arg1 *big.Int) ([]byte, error) {
	return _L1DomiconCommitment.Contract.Submits(&_L1DomiconCommitment.CallOpts, arg0, arg1)
}

// Submits is a free data retrieval call binding the contract method 0xdcf36d57.
//
// Solidity: function submits(address , uint256 ) view returns(bytes)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) Submits(arg0 common.Address, arg1 *big.Int) ([]byte, error) {
	return _L1DomiconCommitment.Contract.Submits(&_L1DomiconCommitment.CallOpts, arg0, arg1)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) Version() (string, error) {
	return _L1DomiconCommitment.Contract.Version(&_L1DomiconCommitment.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) Version() (string, error) {
	return _L1DomiconCommitment.Contract.Version(&_L1DomiconCommitment.CallOpts)
}

// SetDom is a paid mutator transaction binding the contract method 0x6d881989.
//
// Solidity: function SetDom(address addr) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactor) SetDom(opts *bind.TransactOpts, addr common.Address) (*types.Transaction, error) {
	return _L1DomiconCommitment.contract.Transact(opts, "SetDom", addr)
}

// SetDom is a paid mutator transaction binding the contract method 0x6d881989.
//
// Solidity: function SetDom(address addr) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentSession) SetDom(addr common.Address) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.SetDom(&_L1DomiconCommitment.TransactOpts, addr)
}

// SetDom is a paid mutator transaction binding the contract method 0x6d881989.
//
// Solidity: function SetDom(address addr) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorSession) SetDom(addr common.Address) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.SetDom(&_L1DomiconCommitment.TransactOpts, addr)
}

// SubmitCommitment is a paid mutator transaction binding the contract method 0x8ee1d239.
//
// Solidity: function SubmitCommitment(uint64 _index, uint64 _length, uint64 _price, address _user, bytes _sign, bytes _commitment) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactor) SubmitCommitment(opts *bind.TransactOpts, _index uint64, _length uint64, _price uint64, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.contract.Transact(opts, "SubmitCommitment", _index, _length, _price, _user, _sign, _commitment)
}

// SubmitCommitment is a paid mutator transaction binding the contract method 0x8ee1d239.
//
// Solidity: function SubmitCommitment(uint64 _index, uint64 _length, uint64 _price, address _user, bytes _sign, bytes _commitment) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentSession) SubmitCommitment(_index uint64, _length uint64, _price uint64, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.SubmitCommitment(&_L1DomiconCommitment.TransactOpts, _index, _length, _price, _user, _sign, _commitment)
}

// SubmitCommitment is a paid mutator transaction binding the contract method 0x8ee1d239.
//
// Solidity: function SubmitCommitment(uint64 _index, uint64 _length, uint64 _price, address _user, bytes _sign, bytes _commitment) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorSession) SubmitCommitment(_index uint64, _length uint64, _price uint64, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.SubmitCommitment(&_L1DomiconCommitment.TransactOpts, _index, _length, _price, _user, _sign, _commitment)
}

// FinalizeSubmitCommitment is a paid mutator transaction binding the contract method 0x777109f8.
//
// Solidity: function finalizeSubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _broadcaster, address _user, bytes _sign, bytes _commitment) payable returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactor) FinalizeSubmitCommitment(opts *bind.TransactOpts, _index *big.Int, _length *big.Int, _price *big.Int, _broadcaster common.Address, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.contract.Transact(opts, "finalizeSubmitCommitment", _index, _length, _price, _broadcaster, _user, _sign, _commitment)
}

// FinalizeSubmitCommitment is a paid mutator transaction binding the contract method 0x777109f8.
//
// Solidity: function finalizeSubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _broadcaster, address _user, bytes _sign, bytes _commitment) payable returns()
func (_L1DomiconCommitment *L1DomiconCommitmentSession) FinalizeSubmitCommitment(_index *big.Int, _length *big.Int, _price *big.Int, _broadcaster common.Address, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.FinalizeSubmitCommitment(&_L1DomiconCommitment.TransactOpts, _index, _length, _price, _broadcaster, _user, _sign, _commitment)
}

// FinalizeSubmitCommitment is a paid mutator transaction binding the contract method 0x777109f8.
//
// Solidity: function finalizeSubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _broadcaster, address _user, bytes _sign, bytes _commitment) payable returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorSession) FinalizeSubmitCommitment(_index *big.Int, _length *big.Int, _price *big.Int, _broadcaster common.Address, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.FinalizeSubmitCommitment(&_L1DomiconCommitment.TransactOpts, _index, _length, _price, _broadcaster, _user, _sign, _commitment)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _messenger, address _node) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactor) Initialize(opts *bind.TransactOpts, _messenger common.Address, _node common.Address) (*types.Transaction, error) {
	return _L1DomiconCommitment.contract.Transact(opts, "initialize", _messenger, _node)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _messenger, address _node) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentSession) Initialize(_messenger common.Address, _node common.Address) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.Initialize(&_L1DomiconCommitment.TransactOpts, _messenger, _node)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _messenger, address _node) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorSession) Initialize(_messenger common.Address, _node common.Address) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.Initialize(&_L1DomiconCommitment.TransactOpts, _messenger, _node)
}

// L1DomiconCommitmentFinalizeSubmitCommitmentIterator is returned from FilterFinalizeSubmitCommitment and is used to iterate over the raw logs and unpacked data for FinalizeSubmitCommitment events raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentFinalizeSubmitCommitmentIterator struct {
	Event *L1DomiconCommitmentFinalizeSubmitCommitment // Event containing the contract specifics and raw log

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
func (it *L1DomiconCommitmentFinalizeSubmitCommitmentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1DomiconCommitmentFinalizeSubmitCommitment)
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
		it.Event = new(L1DomiconCommitmentFinalizeSubmitCommitment)
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
func (it *L1DomiconCommitmentFinalizeSubmitCommitmentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1DomiconCommitmentFinalizeSubmitCommitmentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1DomiconCommitmentFinalizeSubmitCommitment represents a FinalizeSubmitCommitment event raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentFinalizeSubmitCommitment struct {
	Index       *big.Int
	Length      *big.Int
	Price       *big.Int
	Broadcaster common.Address
	User        common.Address
	Sign        []byte
	Commitment  []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterFinalizeSubmitCommitment is a free log retrieval operation binding the contract event 0x9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea.
//
// Solidity: event FinalizeSubmitCommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) FilterFinalizeSubmitCommitment(opts *bind.FilterOpts, broadcaster []common.Address, user []common.Address) (*L1DomiconCommitmentFinalizeSubmitCommitmentIterator, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L1DomiconCommitment.contract.FilterLogs(opts, "FinalizeSubmitCommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentFinalizeSubmitCommitmentIterator{contract: _L1DomiconCommitment.contract, event: "FinalizeSubmitCommitment", logs: logs, sub: sub}, nil
}

// WatchFinalizeSubmitCommitment is a free log subscription operation binding the contract event 0x9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea.
//
// Solidity: event FinalizeSubmitCommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) WatchFinalizeSubmitCommitment(opts *bind.WatchOpts, sink chan<- *L1DomiconCommitmentFinalizeSubmitCommitment, broadcaster []common.Address, user []common.Address) (event.Subscription, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L1DomiconCommitment.contract.WatchLogs(opts, "FinalizeSubmitCommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1DomiconCommitmentFinalizeSubmitCommitment)
				if err := _L1DomiconCommitment.contract.UnpackLog(event, "FinalizeSubmitCommitment", log); err != nil {
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

// ParseFinalizeSubmitCommitment is a log parse operation binding the contract event 0x9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea.
//
// Solidity: event FinalizeSubmitCommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) ParseFinalizeSubmitCommitment(log types.Log) (*L1DomiconCommitmentFinalizeSubmitCommitment, error) {
	event := new(L1DomiconCommitmentFinalizeSubmitCommitment)
	if err := _L1DomiconCommitment.contract.UnpackLog(event, "FinalizeSubmitCommitment", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1DomiconCommitmentInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentInitializedIterator struct {
	Event *L1DomiconCommitmentInitialized // Event containing the contract specifics and raw log

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
func (it *L1DomiconCommitmentInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1DomiconCommitmentInitialized)
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
		it.Event = new(L1DomiconCommitmentInitialized)
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
func (it *L1DomiconCommitmentInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1DomiconCommitmentInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1DomiconCommitmentInitialized represents a Initialized event raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) FilterInitialized(opts *bind.FilterOpts) (*L1DomiconCommitmentInitializedIterator, error) {

	logs, sub, err := _L1DomiconCommitment.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentInitializedIterator{contract: _L1DomiconCommitment.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L1DomiconCommitmentInitialized) (event.Subscription, error) {

	logs, sub, err := _L1DomiconCommitment.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1DomiconCommitmentInitialized)
				if err := _L1DomiconCommitment.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) ParseInitialized(log types.Log) (*L1DomiconCommitmentInitialized, error) {
	event := new(L1DomiconCommitmentInitialized)
	if err := _L1DomiconCommitment.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1DomiconCommitmentSendDACommitmentIterator is returned from FilterSendDACommitment and is used to iterate over the raw logs and unpacked data for SendDACommitment events raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentSendDACommitmentIterator struct {
	Event *L1DomiconCommitmentSendDACommitment // Event containing the contract specifics and raw log

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
func (it *L1DomiconCommitmentSendDACommitmentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1DomiconCommitmentSendDACommitment)
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
		it.Event = new(L1DomiconCommitmentSendDACommitment)
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
func (it *L1DomiconCommitmentSendDACommitmentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1DomiconCommitmentSendDACommitmentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1DomiconCommitmentSendDACommitment represents a SendDACommitment event raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentSendDACommitment struct {
	Index       *big.Int
	Length      *big.Int
	Price       *big.Int
	Broadcaster common.Address
	User        common.Address
	Sign        []byte
	Commitment  []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSendDACommitment is a free log retrieval operation binding the contract event 0xce7c513598cb8cde5f5798356c301e6ed9a2889048d0cb2e36504b5f9c85d90e.
//
// Solidity: event SendDACommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) FilterSendDACommitment(opts *bind.FilterOpts, broadcaster []common.Address, user []common.Address) (*L1DomiconCommitmentSendDACommitmentIterator, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L1DomiconCommitment.contract.FilterLogs(opts, "SendDACommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentSendDACommitmentIterator{contract: _L1DomiconCommitment.contract, event: "SendDACommitment", logs: logs, sub: sub}, nil
}

// WatchSendDACommitment is a free log subscription operation binding the contract event 0xce7c513598cb8cde5f5798356c301e6ed9a2889048d0cb2e36504b5f9c85d90e.
//
// Solidity: event SendDACommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) WatchSendDACommitment(opts *bind.WatchOpts, sink chan<- *L1DomiconCommitmentSendDACommitment, broadcaster []common.Address, user []common.Address) (event.Subscription, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L1DomiconCommitment.contract.WatchLogs(opts, "SendDACommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1DomiconCommitmentSendDACommitment)
				if err := _L1DomiconCommitment.contract.UnpackLog(event, "SendDACommitment", log); err != nil {
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

// ParseSendDACommitment is a log parse operation binding the contract event 0xce7c513598cb8cde5f5798356c301e6ed9a2889048d0cb2e36504b5f9c85d90e.
//
// Solidity: event SendDACommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) ParseSendDACommitment(log types.Log) (*L1DomiconCommitmentSendDACommitment, error) {
	event := new(L1DomiconCommitmentSendDACommitment)
	if err := _L1DomiconCommitment.contract.UnpackLog(event, "SendDACommitment", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
