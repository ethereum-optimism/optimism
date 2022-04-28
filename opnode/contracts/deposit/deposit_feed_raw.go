// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package deposit

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

// WithdrawalVerifierOutputRootProof is an auto generated low-level Go binding around an user-defined struct.
type WithdrawalVerifierOutputRootProof struct {
	Version               [32]byte
	StateRoot             [32]byte
	WithdrawerStorageRoot [32]byte
	LatestBlockhash       [32]byte
}

// OptimismPortalMetaData contains all meta data concerning the OptimismPortal contract.
var OptimismPortalMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractL2OutputOracle\",\"name\":\"_l2Oracle\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_finalizationPeriod\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"InvalidOutputRootProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidWithdrawalInclusionProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NonZeroCreationTarget\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotYetFinal\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"WithdrawalAlreadyFinalized\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"mint\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"gasLimit\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"isCreation\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"TransactionDeposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"WithdrawalFinalized\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"FINALIZATION_PERIOD\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_ORACLE\",\"outputs\":[{\"internalType\":\"contractL2OutputOracle\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"_gasLimit\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"_isCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositTransaction\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_timestamp\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"version\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"withdrawerStorageRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"latestBlockhash\",\"type\":\"bytes32\"}],\"internalType\":\"structWithdrawalVerifier.OutputRootProof\",\"name\":\"_outputRootProof\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"_withdrawalProof\",\"type\":\"bytes\"}],\"name\":\"finalizeWithdrawalTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"finalizedWithdrawals\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2Sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x60c0604052600080546001600160a01b03191661dead17905534801561002457600080fd5b50604051620022f9380380620022f98339810160408190526100459161005b565b6001600160a01b0390911660a052608052610095565b6000806040838503121561006e57600080fd5b82516001600160a01b038116811461008557600080fd5b6020939093015192949293505050565b60805160a051612231620000c86000396000818160a501526103500152600081816101a301526102c401526122316000f3fe6080604052600436106100685760003560e01c8063e9e05c4211610043578063e9e05c421461015e578063eecf1c3614610171578063ff61cc931461019157600080fd5b80621c2ff6146100935780639bf62d82146100f1578063a14238e71461011e57600080fd5b3661008e5761008c33346175306000604051806020016040528060008152506101d3565b005b600080fd5b34801561009f57600080fd5b506100c77f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b3480156100fd57600080fd5b506000546100c79073ffffffffffffffffffffffffffffffffffffffff1681565b34801561012a57600080fd5b5061014e610139366004611c41565b60016020526000908152604090205460ff1681565b60405190151581526020016100e8565b61008c61016c366004611cb2565b6101d3565b34801561017d57600080fd5b5061008c61018c366004611e18565b6102c2565b34801561019d57600080fd5b506101c57f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020016100e8565b8180156101f5575073ffffffffffffffffffffffffffffffffffffffff851615155b1561022c576040517ff98844ef00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3332811461024d575033731111000000000000000000000000000000001111015b8573ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f78231ae6eb73366f912bb1d64351601fb76344c537bbab635ce14d0f376f019534888888886040516102b2959493929190611f18565b60405180910390a3505050505050565b7f0000000000000000000000000000000000000000000000000000000000000000840142101561031e576040517fe4750a3000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040517fa25ae557000000000000000000000000000000000000000000000000000000008152600481018590526000907f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff169063a25ae55790602401602060405180830381865afa1580156103ac573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906103d09190611fb4565b9050610424846040805182356020828101919091528301358183015290820135606082810191909152820135608082015260009060a001604051602081830303815290604052805190602001209050919050565b811461045c576040517f9cc00b5b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600061046d8d8d8d8d8d8d8d610648565b905061047f818660400135868661068a565b6104b5576040517feb00eb2200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008181526001602081905260409091205460ff1615151415610504576040517fae89945400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600180600083815260200190815260200160002060006101000a81548160ff0219169083151502179055508b6000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060008b73ffffffffffffffffffffffffffffffffffffffff168b8b908b8b60405161059b929190611fcd565b600060405180830381858888f193505050503d80600081146105d9576040519150601f19603f3d011682016040523d82523d6000602084013e6105de565b606091505b5050600080547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead17815560405191925083917f894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f9190a25050505050505050505050505050565b6000878787878787876040516020016106679796959493929190611fdd565b604051602081830303815290604052805190602001209050979650505050505050565b6000808560016040516020016106aa929190918252602082015260400190565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181528282528051602091820120908301819052925061077091016040516020818303038152906040526040518060400160405280600181526020017f010000000000000000000000000000000000000000000000000000000000000081525086868080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152508b925061077a915050565b9695505050505050565b60008061078686610794565b9050610770818686866107c6565b606081805190602001206040516020016107b091815260200190565b6040516020818303038152906040529050919050565b60008060006107d68786866107f7565b915091508180156107ec57506107ec86826108f1565b979650505050505050565b6000606060006108068561090d565b90506000806000610818848a89610a08565b8151929550909350915015808061082c5750815b610897576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601a60248201527f50726f76696465642070726f6f6620697320696e76616c69642e00000000000060448201526064015b60405180910390fd5b6000816108b357604051806020016040528060008152506108df565b6108df866108c2600188612099565b815181106108d2576108d26120b0565b6020026020010151610f25565b919b919a509098505050505050505050565b6000818051906020012083805190602001201490505b92915050565b6060600061091a83610f4f565b90506000815167ffffffffffffffff81111561093857610938611c83565b60405190808252806020026020018201604052801561097d57816020015b60408051808201909152606080825260208201528152602001906001900390816109565790505b50905060005b8251811015610a005760006109b08483815181106109a3576109a36120b0565b6020026020010151610f82565b905060405180604001604052808281526020016109cc83610f4f565b8152508383815181106109e1576109e16120b0565b60200260200101819052505080806109f8906120df565b915050610983565b509392505050565b60006060818080610a188761102c565b90506000869050600080610a3f604051806040016040528060608152602001606081525090565b60005b8c51811015610ee1578c8181518110610a5d57610a5d6120b0565b602002602001015191508284610a739190612118565b9350610a80600188612118565b965083610afe57815180516020909101208514610af9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601160248201527f496e76616c696420726f6f742068617368000000000000000000000000000000604482015260640161088e565b610bef565b815151602011610b7a57815180516020909101208514610af9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601b60248201527f496e76616c6964206c6172676520696e7465726e616c20686173680000000000604482015260640161088e565b84610b8883600001516111af565b14610bef576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601a60248201527f496e76616c696420696e7465726e616c206e6f64652068617368000000000000604482015260640161088e565b610bfb60106001612118565b8260200151511415610c74578551841415610c1557610ee1565b6000868581518110610c2957610c296120b0565b602001015160f81c60f81b60f81c9050600083602001518260ff1681518110610c5457610c546120b0565b60200260200101519050610c67816111d7565b9650600194505050610ecf565b60028260200151511415610e6d576000610c8d83611214565b9050600081600081518110610ca457610ca46120b0565b016020015160f81c90506000610cbb60028361215f565b610cc6906002612181565b90506000610cd7848360ff16611238565b90506000610ce58b8a611238565b90506000610cf3838361126e565b905060ff851660021480610d0a575060ff85166003145b15610d6057808351148015610d1f5750808251145b15610d3157610d2e818b612118565b99505b507f80000000000000000000000000000000000000000000000000000000000000009950610ee1945050505050565b60ff85161580610d73575060ff85166001145b15610de55782518114610daf57507f80000000000000000000000000000000000000000000000000000000000000009950610ee1945050505050565b610dd68860200151600181518110610dc957610dc96120b0565b60200260200101516111d7565b9a509750610ecf945050505050565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f52656365697665642061206e6f6465207769746820616e20756e6b6e6f776e2060448201527f7072656669780000000000000000000000000000000000000000000000000000606482015260840161088e565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f526563656976656420616e20756e706172736561626c65206e6f64652e000000604482015260640161088e565b80610ed9816120df565b915050610a42565b507f8000000000000000000000000000000000000000000000000000000000000000841486610f108786611238565b909e909d50909b509950505050505050505050565b6020810151805160609161090791610f3f90600190612099565b815181106109a3576109a36120b0565b6040805180820182526000808252602091820152815180830190925282518252808301908201526060906109079061131a565b60606000806000610f928561154d565b919450925090506000816001811115610fad57610fad6121a4565b14611014576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601860248201527f496e76616c696420524c502062797465732076616c75652e0000000000000000604482015260640161088e565b61102385602001518484611954565b95945050505050565b606060008251600261103e91906121d3565b67ffffffffffffffff81111561105657611056611c83565b6040519080825280601f01601f191660200182016040528015611080576020820181803683370190505b50905060005b83518110156111a85760048482815181106110a3576110a36120b0565b01602001517fff0000000000000000000000000000000000000000000000000000000000000016901c826110d88360026121d3565b815181106110e8576110e86120b0565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350601084828151811061112b5761112b6120b0565b016020015161113d919060f81c61215f565b60f81b8261114c8360026121d3565b611157906001612118565b81518110611167576111676120b0565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350806111a0816120df565b915050611086565b5092915050565b60006020825110156111c357506020015190565b818060200190518101906109079190611fb4565b600060606020836000015110156111f8576111f183611a33565b9050611204565b61120183610f82565b90505b61120d816111af565b9392505050565b606061090761123383602001516000815181106109a3576109a36120b0565b61102c565b6060825182106112575750604080516020810190915260008152610907565b61120d83838486516112699190612099565b611a3e565b6000805b8084511180156112825750808351115b8015611303575082818151811061129b5761129b6120b0565b602001015160f81c60f81b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168482815181106112da576112da6120b0565b01602001517fff0000000000000000000000000000000000000000000000000000000000000016145b1561120d5780611312816120df565b915050611272565b60606000806113288461154d565b91935090915060019050816001811115611344576113446121a4565b146113ab576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f496e76616c696420524c50206c6973742076616c75652e000000000000000000604482015260640161088e565b6040805160208082526104208201909252600091816020015b60408051808201909152600080825260208201528152602001906001900390816113c45790505090506000835b8651811015611542576020821061148a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f50726f766964656420524c50206c6973742065786365656473206d6178206c6960448201527f7374206c656e6774682e00000000000000000000000000000000000000000000606482015260840161088e565b6000806114c76040518060400160405280858c600001516114ab9190612099565b8152602001858c602001516114c09190612118565b905261154d565b5091509150604051806040016040528083836114e39190612118565b8152602001848b602001516114f89190612118565b81525085858151811061150d5761150d6120b0565b6020908102919091010152611523600185612118565b935061152f8183612118565b6115399084612118565b925050506113f1565b508152949350505050565b6000806000808460000151116115bf576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601860248201527f524c50206974656d2063616e6e6f74206265206e756c6c2e0000000000000000604482015260640161088e565b6020840151805160001a607f81116115e457600060016000945094509450505061194d565b60b7811161167a5760006115f9608083612099565b905080876000015111611668576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601960248201527f496e76616c696420524c502073686f727420737472696e672e00000000000000604482015260640161088e565b6001955093506000925061194d915050565b60bf811161179d57600061168f60b783612099565b9050808760000151116116fe576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f496e76616c696420524c50206c6f6e6720737472696e67206c656e6774682e00604482015260640161088e565b600183015160208290036101000a90046117188183612118565b885111611781576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601860248201527f496e76616c696420524c50206c6f6e6720737472696e672e0000000000000000604482015260640161088e565b61178c826001612118565b965094506000935061194d92505050565b60f781116118325760006117b260c083612099565b905080876000015111611821576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f496e76616c696420524c502073686f7274206c6973742e000000000000000000604482015260640161088e565b60019550935084925061194d915050565b600061183f60f783612099565b9050808760000151116118ae576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f496e76616c696420524c50206c6f6e67206c697374206c656e6774682e000000604482015260640161088e565b600183015160208290036101000a90046118c88183612118565b885111611931576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601660248201527f496e76616c696420524c50206c6f6e67206c6973742e00000000000000000000604482015260640161088e565b61193c826001612118565b965094506001935061194d92505050565b9193909250565b606060008267ffffffffffffffff81111561197157611971611c83565b6040519080825280601f01601f19166020018201604052801561199b576020820181803683370190505b5090508051600014156119af57905061120d565b60006119bb8587612118565b90506020820160005b6119cf602087612210565b811015611a0657825182526119e5602084612118565b92506119f2602083612118565b9150806119fe816120df565b9150506119c4565b5060006001602087066020036101000a039050808251168119845116178252839450505050509392505050565b606061090782611c2b565b606081611a4c81601f612118565b1015611ab4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f736c6963655f6f766572666c6f77000000000000000000000000000000000000604482015260640161088e565b82611abf8382612118565b1015611b27576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f736c6963655f6f766572666c6f77000000000000000000000000000000000000604482015260640161088e565b611b318284612118565b84511015611b9b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601160248201527f736c6963655f6f75744f66426f756e6473000000000000000000000000000000604482015260640161088e565b606082158015611bba5760405191506000825260208201604052611c22565b6040519150601f8416801560200281840101858101878315602002848b0101015b81831015611bf3578051835260209283019201611bdb565b5050858452601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016604052505b50949350505050565b6060610907826020015160008460000151611954565b600060208284031215611c5357600080fd5b5035919050565b803573ffffffffffffffffffffffffffffffffffffffff81168114611c7e57600080fd5b919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600080600080600060a08688031215611cca57600080fd5b611cd386611c5a565b945060208601359350604086013567ffffffffffffffff8082168214611cf857600080fd5b9093506060870135908115158214611d0f57600080fd5b90925060808701359080821115611d2557600080fd5b818801915088601f830112611d3957600080fd5b813581811115611d4b57611d4b611c83565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908382118183101715611d9157611d91611c83565b816040528281528b6020848701011115611daa57600080fd5b8260208601602083013760006020848301015280955050505050509295509295909350565b60008083601f840112611de157600080fd5b50813567ffffffffffffffff811115611df957600080fd5b602083019150836020828501011115611e1157600080fd5b9250929050565b60008060008060008060008060008060006101808c8e031215611e3a57600080fd5b8b359a50611e4a60208d01611c5a565b9950611e5860408d01611c5a565b985060608c0135975060808c0135965067ffffffffffffffff60a08d01351115611e8157600080fd5b611e918d60a08e01358e01611dcf565b909650945060c08c0135935060808c8e037fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff20011215611ecf57600080fd5b60e08c01925067ffffffffffffffff6101608d01351115611eef57600080fd5b611f008d6101608e01358e01611dcf565b81935080925050509295989b509295989b9093969950565b85815260006020868184015267ffffffffffffffff86166040840152841515606084015260a0608084015283518060a085015260005b81811015611f6a5785810183015185820160c001528201611f4e565b81811115611f7c57600060c083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160c001979650505050505050565b600060208284031215611fc657600080fd5b5051919050565b8183823760009101908152919050565b878152600073ffffffffffffffffffffffffffffffffffffffff808916602084015280881660408401525085606083015284608083015260c060a08301528260c0830152828460e0840137600060e0848401015260e07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f850116830101905098975050505050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000828210156120ab576120ab61206a565b500390565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8214156121115761211161206a565b5060010190565b6000821982111561212b5761212b61206a565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600060ff83168061217257612172612130565b8060ff84160691505092915050565b600060ff821660ff84168082101561219b5761219b61206a565b90039392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff048311821515161561220b5761220b61206a565b500290565b60008261221f5761221f612130565b50049056fea164736f6c634300080a000a",
}

// OptimismPortalABI is the input ABI used to generate the binding from.
// Deprecated: Use OptimismPortalMetaData.ABI instead.
var OptimismPortalABI = OptimismPortalMetaData.ABI

// OptimismPortalBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OptimismPortalMetaData.Bin instead.
var OptimismPortalBin = OptimismPortalMetaData.Bin

// DeployOptimismPortal deploys a new Ethereum contract, binding an instance of OptimismPortal to it.
func DeployOptimismPortal(auth *bind.TransactOpts, backend bind.ContractBackend, _l2Oracle common.Address, _finalizationPeriod *big.Int) (common.Address, *types.Transaction, *OptimismPortal, error) {
	parsed, err := OptimismPortalMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OptimismPortalBin), backend, _l2Oracle, _finalizationPeriod)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OptimismPortal{OptimismPortalCaller: OptimismPortalCaller{contract: contract}, OptimismPortalTransactor: OptimismPortalTransactor{contract: contract}, OptimismPortalFilterer: OptimismPortalFilterer{contract: contract}}, nil
}

// OptimismPortal is an auto generated Go binding around an Ethereum contract.
type OptimismPortal struct {
	OptimismPortalCaller     // Read-only binding to the contract
	OptimismPortalTransactor // Write-only binding to the contract
	OptimismPortalFilterer   // Log filterer for contract events
}

// OptimismPortalCaller is an auto generated read-only Go binding around an Ethereum contract.
type OptimismPortalCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortalTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OptimismPortalTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortalFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OptimismPortalFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortalSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OptimismPortalSession struct {
	Contract     *OptimismPortal   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OptimismPortalCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OptimismPortalCallerSession struct {
	Contract *OptimismPortalCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// OptimismPortalTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OptimismPortalTransactorSession struct {
	Contract     *OptimismPortalTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// OptimismPortalRaw is an auto generated low-level Go binding around an Ethereum contract.
type OptimismPortalRaw struct {
	Contract *OptimismPortal // Generic contract binding to access the raw methods on
}

// OptimismPortalCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OptimismPortalCallerRaw struct {
	Contract *OptimismPortalCaller // Generic read-only contract binding to access the raw methods on
}

// OptimismPortalTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OptimismPortalTransactorRaw struct {
	Contract *OptimismPortalTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOptimismPortal creates a new instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortal(address common.Address, backend bind.ContractBackend) (*OptimismPortal, error) {
	contract, err := bindOptimismPortal(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal{OptimismPortalCaller: OptimismPortalCaller{contract: contract}, OptimismPortalTransactor: OptimismPortalTransactor{contract: contract}, OptimismPortalFilterer: OptimismPortalFilterer{contract: contract}}, nil
}

// NewOptimismPortalCaller creates a new read-only instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortalCaller(address common.Address, caller bind.ContractCaller) (*OptimismPortalCaller, error) {
	contract, err := bindOptimismPortal(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalCaller{contract: contract}, nil
}

// NewOptimismPortalTransactor creates a new write-only instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortalTransactor(address common.Address, transactor bind.ContractTransactor) (*OptimismPortalTransactor, error) {
	contract, err := bindOptimismPortal(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalTransactor{contract: contract}, nil
}

// NewOptimismPortalFilterer creates a new log filterer instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortalFilterer(address common.Address, filterer bind.ContractFilterer) (*OptimismPortalFilterer, error) {
	contract, err := bindOptimismPortal(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalFilterer{contract: contract}, nil
}

// bindOptimismPortal binds a generic wrapper to an already deployed contract.
func bindOptimismPortal(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OptimismPortalABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismPortal *OptimismPortalRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismPortal.Contract.OptimismPortalCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismPortal *OptimismPortalRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal.Contract.OptimismPortalTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismPortal *OptimismPortalRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismPortal.Contract.OptimismPortalTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismPortal *OptimismPortalCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismPortal.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismPortal *OptimismPortalTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismPortal *OptimismPortalTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismPortal.Contract.contract.Transact(opts, method, params...)
}

// FINALIZATIONPERIOD is a free data retrieval call binding the contract method 0xff61cc93.
//
// Solidity: function FINALIZATION_PERIOD() view returns(uint256)
func (_OptimismPortal *OptimismPortalCaller) FINALIZATIONPERIOD(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "FINALIZATION_PERIOD")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FINALIZATIONPERIOD is a free data retrieval call binding the contract method 0xff61cc93.
//
// Solidity: function FINALIZATION_PERIOD() view returns(uint256)
func (_OptimismPortal *OptimismPortalSession) FINALIZATIONPERIOD() (*big.Int, error) {
	return _OptimismPortal.Contract.FINALIZATIONPERIOD(&_OptimismPortal.CallOpts)
}

// FINALIZATIONPERIOD is a free data retrieval call binding the contract method 0xff61cc93.
//
// Solidity: function FINALIZATION_PERIOD() view returns(uint256)
func (_OptimismPortal *OptimismPortalCallerSession) FINALIZATIONPERIOD() (*big.Int, error) {
	return _OptimismPortal.Contract.FINALIZATIONPERIOD(&_OptimismPortal.CallOpts)
}

// L2ORACLE is a free data retrieval call binding the contract method 0x001c2ff6.
//
// Solidity: function L2_ORACLE() view returns(address)
func (_OptimismPortal *OptimismPortalCaller) L2ORACLE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "L2_ORACLE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2ORACLE is a free data retrieval call binding the contract method 0x001c2ff6.
//
// Solidity: function L2_ORACLE() view returns(address)
func (_OptimismPortal *OptimismPortalSession) L2ORACLE() (common.Address, error) {
	return _OptimismPortal.Contract.L2ORACLE(&_OptimismPortal.CallOpts)
}

// L2ORACLE is a free data retrieval call binding the contract method 0x001c2ff6.
//
// Solidity: function L2_ORACLE() view returns(address)
func (_OptimismPortal *OptimismPortalCallerSession) L2ORACLE() (common.Address, error) {
	return _OptimismPortal.Contract.L2ORACLE(&_OptimismPortal.CallOpts)
}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal *OptimismPortalCaller) FinalizedWithdrawals(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "finalizedWithdrawals", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal *OptimismPortalSession) FinalizedWithdrawals(arg0 [32]byte) (bool, error) {
	return _OptimismPortal.Contract.FinalizedWithdrawals(&_OptimismPortal.CallOpts, arg0)
}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal *OptimismPortalCallerSession) FinalizedWithdrawals(arg0 [32]byte) (bool, error) {
	return _OptimismPortal.Contract.FinalizedWithdrawals(&_OptimismPortal.CallOpts, arg0)
}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal *OptimismPortalCaller) L2Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "l2Sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal *OptimismPortalSession) L2Sender() (common.Address, error) {
	return _OptimismPortal.Contract.L2Sender(&_OptimismPortal.CallOpts)
}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal *OptimismPortalCallerSession) L2Sender() (common.Address, error) {
	return _OptimismPortal.Contract.L2Sender(&_OptimismPortal.CallOpts)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xe9e05c42.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal *OptimismPortalTransactor) DepositTransaction(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "depositTransaction", _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xe9e05c42.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal *OptimismPortalSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.DepositTransaction(&_OptimismPortal.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xe9e05c42.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal *OptimismPortalTransactorSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.DepositTransaction(&_OptimismPortal.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0xeecf1c36.
//
// Solidity: function finalizeWithdrawalTransaction(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data, uint256 _timestamp, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _withdrawalProof) returns()
func (_OptimismPortal *OptimismPortalTransactor) FinalizeWithdrawalTransaction(opts *bind.TransactOpts, _nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte, _timestamp *big.Int, _outputRootProof WithdrawalVerifierOutputRootProof, _withdrawalProof []byte) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "finalizeWithdrawalTransaction", _nonce, _sender, _target, _value, _gasLimit, _data, _timestamp, _outputRootProof, _withdrawalProof)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0xeecf1c36.
//
// Solidity: function finalizeWithdrawalTransaction(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data, uint256 _timestamp, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _withdrawalProof) returns()
func (_OptimismPortal *OptimismPortalSession) FinalizeWithdrawalTransaction(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte, _timestamp *big.Int, _outputRootProof WithdrawalVerifierOutputRootProof, _withdrawalProof []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.FinalizeWithdrawalTransaction(&_OptimismPortal.TransactOpts, _nonce, _sender, _target, _value, _gasLimit, _data, _timestamp, _outputRootProof, _withdrawalProof)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0xeecf1c36.
//
// Solidity: function finalizeWithdrawalTransaction(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data, uint256 _timestamp, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _withdrawalProof) returns()
func (_OptimismPortal *OptimismPortalTransactorSession) FinalizeWithdrawalTransaction(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte, _timestamp *big.Int, _outputRootProof WithdrawalVerifierOutputRootProof, _withdrawalProof []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.FinalizeWithdrawalTransaction(&_OptimismPortal.TransactOpts, _nonce, _sender, _target, _value, _gasLimit, _data, _timestamp, _outputRootProof, _withdrawalProof)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal *OptimismPortalTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal *OptimismPortalSession) Receive() (*types.Transaction, error) {
	return _OptimismPortal.Contract.Receive(&_OptimismPortal.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal *OptimismPortalTransactorSession) Receive() (*types.Transaction, error) {
	return _OptimismPortal.Contract.Receive(&_OptimismPortal.TransactOpts)
}

// OptimismPortalTransactionDepositedIterator is returned from FilterTransactionDeposited and is used to iterate over the raw logs and unpacked data for TransactionDeposited events raised by the OptimismPortal contract.
type OptimismPortalTransactionDepositedIterator struct {
	Event *OptimismPortalTransactionDeposited // Event containing the contract specifics and raw log

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
func (it *OptimismPortalTransactionDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortalTransactionDeposited)
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
		it.Event = new(OptimismPortalTransactionDeposited)
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
func (it *OptimismPortalTransactionDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortalTransactionDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortalTransactionDeposited represents a TransactionDeposited event raised by the OptimismPortal contract.
type OptimismPortalTransactionDeposited struct {
	From       common.Address
	To         common.Address
	Mint       *big.Int
	Value      *big.Int
	GasLimit   uint64
	IsCreation bool
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTransactionDeposited is a free log retrieval operation binding the contract event 0x78231ae6eb73366f912bb1d64351601fb76344c537bbab635ce14d0f376f0195.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint64 gasLimit, bool isCreation, bytes data)
func (_OptimismPortal *OptimismPortalFilterer) FilterTransactionDeposited(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*OptimismPortalTransactionDepositedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _OptimismPortal.contract.FilterLogs(opts, "TransactionDeposited", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalTransactionDepositedIterator{contract: _OptimismPortal.contract, event: "TransactionDeposited", logs: logs, sub: sub}, nil
}

// WatchTransactionDeposited is a free log subscription operation binding the contract event 0x78231ae6eb73366f912bb1d64351601fb76344c537bbab635ce14d0f376f0195.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint64 gasLimit, bool isCreation, bytes data)
func (_OptimismPortal *OptimismPortalFilterer) WatchTransactionDeposited(opts *bind.WatchOpts, sink chan<- *OptimismPortalTransactionDeposited, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _OptimismPortal.contract.WatchLogs(opts, "TransactionDeposited", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortalTransactionDeposited)
				if err := _OptimismPortal.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
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

// ParseTransactionDeposited is a log parse operation binding the contract event 0x78231ae6eb73366f912bb1d64351601fb76344c537bbab635ce14d0f376f0195.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint64 gasLimit, bool isCreation, bytes data)
func (_OptimismPortal *OptimismPortalFilterer) ParseTransactionDeposited(log types.Log) (*OptimismPortalTransactionDeposited, error) {
	event := new(OptimismPortalTransactionDeposited)
	if err := _OptimismPortal.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortalWithdrawalFinalizedIterator is returned from FilterWithdrawalFinalized and is used to iterate over the raw logs and unpacked data for WithdrawalFinalized events raised by the OptimismPortal contract.
type OptimismPortalWithdrawalFinalizedIterator struct {
	Event *OptimismPortalWithdrawalFinalized // Event containing the contract specifics and raw log

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
func (it *OptimismPortalWithdrawalFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortalWithdrawalFinalized)
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
		it.Event = new(OptimismPortalWithdrawalFinalized)
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
func (it *OptimismPortalWithdrawalFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortalWithdrawalFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortalWithdrawalFinalized represents a WithdrawalFinalized event raised by the OptimismPortal contract.
type OptimismPortalWithdrawalFinalized struct {
	Arg0 [32]byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalFinalized is a free log retrieval operation binding the contract event 0x894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed arg0)
func (_OptimismPortal *OptimismPortalFilterer) FilterWithdrawalFinalized(opts *bind.FilterOpts, arg0 [][32]byte) (*OptimismPortalWithdrawalFinalizedIterator, error) {

	var arg0Rule []interface{}
	for _, arg0Item := range arg0 {
		arg0Rule = append(arg0Rule, arg0Item)
	}

	logs, sub, err := _OptimismPortal.contract.FilterLogs(opts, "WithdrawalFinalized", arg0Rule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalWithdrawalFinalizedIterator{contract: _OptimismPortal.contract, event: "WithdrawalFinalized", logs: logs, sub: sub}, nil
}

// WatchWithdrawalFinalized is a free log subscription operation binding the contract event 0x894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed arg0)
func (_OptimismPortal *OptimismPortalFilterer) WatchWithdrawalFinalized(opts *bind.WatchOpts, sink chan<- *OptimismPortalWithdrawalFinalized, arg0 [][32]byte) (event.Subscription, error) {

	var arg0Rule []interface{}
	for _, arg0Item := range arg0 {
		arg0Rule = append(arg0Rule, arg0Item)
	}

	logs, sub, err := _OptimismPortal.contract.WatchLogs(opts, "WithdrawalFinalized", arg0Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortalWithdrawalFinalized)
				if err := _OptimismPortal.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
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

// ParseWithdrawalFinalized is a log parse operation binding the contract event 0x894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed arg0)
func (_OptimismPortal *OptimismPortalFilterer) ParseWithdrawalFinalized(log types.Log) (*OptimismPortalWithdrawalFinalized, error) {
	event := new(OptimismPortalWithdrawalFinalized)
	if err := _OptimismPortal.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
