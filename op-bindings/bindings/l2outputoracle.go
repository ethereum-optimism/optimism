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

// TypesOutputProposal is an auto generated low-level Go binding around an user-defined struct.
type TypesOutputProposal struct {
	OutputRoot    [32]byte
	Timestamp     *big.Int
	L2BlockNumber *big.Int
}

// L2OutputOracleMetaData contains all meta data concerning the L2OutputOracle contract.
var L2OutputOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_submissionInterval\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_finalizationPeriodSeconds\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l2OutputIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l2BlockNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"l1Timestamp\",\"type\":\"uint256\"}],\"name\":\"OutputProposed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"prevNextOutputIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"newNextOutputIndex\",\"type\":\"uint256\"}],\"name\":\"OutputsDeleted\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"CHALLENGER\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"FINALIZATION_PERIOD_SECONDS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_BLOCK_TIME\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PROPOSER\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SUBMISSION_INTERVAL\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"challenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"computeL2Timestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2OutputIndex\",\"type\":\"uint256\"}],\"name\":\"deleteL2Outputs\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"finalizationPeriodSeconds\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2OutputIndex\",\"type\":\"uint256\"}],\"name\":\"getL2Output\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint128\",\"name\":\"timestamp\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"l2BlockNumber\",\"type\":\"uint128\"}],\"internalType\":\"structTypes.OutputProposal\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"getL2OutputAfter\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint128\",\"name\":\"timestamp\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"l2BlockNumber\",\"type\":\"uint128\"}],\"internalType\":\"structTypes.OutputProposal\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"getL2OutputIndexAfter\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_startingBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_startingTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_proposer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_challenger\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2BlockTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestOutputIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextOutputIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_l1BlockHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l1BlockNumber\",\"type\":\"uint256\"}],\"name\":\"proposeL2Output\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proposer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startingBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startingTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"submissionInterval\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101406040523480156200001257600080fd5b5060405162001b9038038062001b90833981016040819052620000359162000302565b60016080819052600460a05260c05281620000bd5760405162461bcd60e51b815260206004820152603460248201527f4c324f75747075744f7261636c653a204c3220626c6f636b2074696d65206d7560448201527f73742062652067726561746572207468616e203000000000000000000000000060648201526084015b60405180910390fd5b60008311620001355760405162461bcd60e51b815260206004820152603a60248201527f4c324f75747075744f7261636c653a207375626d697373696f6e20696e74657260448201527f76616c206d7573742062652067726561746572207468616e20300000000000006064820152608401620000b4565b60e08390526101008290526101208190526200015560008080806200015e565b50505062000331565b600054600290610100900460ff1615801562000181575060005460ff8083169116105b620001e65760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b6064820152608401620000b4565b6000805461ffff191660ff83161761010017905542841115620002805760405162461bcd60e51b8152602060048201526044602482018190527f4c324f75747075744f7261636c653a207374617274696e67204c322074696d65908201527f7374616d70206d757374206265206c657373207468616e2063757272656e742060648201526374696d6560e01b608482015260a401620000b4565b60028490556001859055600580546001600160a01b038581166001600160a01b03199283161790925560048054928516929091169190911790556000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050505050565b6000806000606084860312156200031857600080fd5b8351925060208401519150604084015190509250925092565b60805160a05160c05160e05161010051610120516117e5620003ab6000396000818161047f0152818161053d0152610c5a0152600081816101a10152818161038501526112e301526000818161021f0152818161050701526113310152600061088a015260006108610152600061083801526117e56000f3fe60806040526004361061018a5760003560e01c806389c44cbb116100d6578063ce5db8d61161007f578063dcec334811610059578063dcec3348146104e3578063e1a41bcf146104f8578063f4daa2911461052b57600080fd5b8063ce5db8d614610470578063cf8e5cf0146104a3578063d1de856c146104c357600080fd5b8063a25ae557116100b0578063a25ae557146103bc578063a8e4fb9014610418578063bffa7f0f1461044557600080fd5b806389c44cbb1461035657806393991af3146103765780639aaab648146103a957600080fd5b806369f16eec1161013857806370872aa51161011257806370872aa51461030a5780637f00642014610320578063887862721461034057600080fd5b806369f16eec146102b55780636abcf563146102ca5780636b4d98dd146102df57600080fd5b8063529933df11610169578063529933df1461020d578063534db0e21461024157806354fd4d501461029357600080fd5b80622134cc1461018f578063019e2729146101d65780634599c788146101f8575b600080fd5b34801561019b57600080fd5b506101c37f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020015b60405180910390f35b3480156101e257600080fd5b506101f66101f13660046114c8565b61055f565b005b34801561020457600080fd5b506101c36107be565b34801561021957600080fd5b506101c37f000000000000000000000000000000000000000000000000000000000000000081565b34801561024d57600080fd5b5060045461026e9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101cd565b34801561029f57600080fd5b506102a8610831565b6040516101cd919061153e565b3480156102c157600080fd5b506101c36108d4565b3480156102d657600080fd5b506003546101c3565b3480156102eb57600080fd5b5060045473ffffffffffffffffffffffffffffffffffffffff1661026e565b34801561031657600080fd5b506101c360015481565b34801561032c57600080fd5b506101c361033b36600461158f565b6108e6565b34801561034c57600080fd5b506101c360025481565b34801561036257600080fd5b506101f661037136600461158f565b610afa565b34801561038257600080fd5b507f00000000000000000000000000000000000000000000000000000000000000006101c3565b6101f66103b73660046115a8565b610db2565b3480156103c857600080fd5b506103dc6103d736600461158f565b611213565b60408051825181526020808401516fffffffffffffffffffffffffffffffff9081169183019190915292820151909216908201526060016101cd565b34801561042457600080fd5b5060055461026e9073ffffffffffffffffffffffffffffffffffffffff1681565b34801561045157600080fd5b5060055473ffffffffffffffffffffffffffffffffffffffff1661026e565b34801561047c57600080fd5b507f00000000000000000000000000000000000000000000000000000000000000006101c3565b3480156104af57600080fd5b506103dc6104be36600461158f565b6112a7565b3480156104cf57600080fd5b506101c36104de36600461158f565b6112df565b3480156104ef57600080fd5b506101c361132d565b34801561050457600080fd5b507f00000000000000000000000000000000000000000000000000000000000000006101c3565b34801561053757600080fd5b506101c37f000000000000000000000000000000000000000000000000000000000000000081565b600054600290610100900460ff16158015610581575060005460ff8083169116105b610612576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084015b60405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff831617610100179055428411156106fa576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526044602482018190527f4c324f75747075744f7261636c653a207374617274696e67204c322074696d65908201527f7374616d70206d757374206265206c657373207468616e2063757272656e742060648201527f74696d6500000000000000000000000000000000000000000000000000000000608482015260a401610609565b600284905560018590556005805473ffffffffffffffffffffffffffffffffffffffff8581167fffffffffffffffffffffffff0000000000000000000000000000000000000000928316179092556004805492851692909116919091179055600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050505050565b6003546000901561082857600380546107d990600190611609565b815481106107e9576107e9611620565b600091825260209091206002909102016001015470010000000000000000000000000000000090046fffffffffffffffffffffffffffffffff16919050565b6001545b905090565b606061085c7f0000000000000000000000000000000000000000000000000000000000000000611362565b6108857f0000000000000000000000000000000000000000000000000000000000000000611362565b6108ae7f0000000000000000000000000000000000000000000000000000000000000000611362565b6040516020016108c09392919061164f565b604051602081830303815290604052905090565b60035460009061082c90600190611609565b60006108f06107be565b8211156109a5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f4c324f75747075744f7261636c653a2063616e6e6f7420676574206f7574707560448201527f7420666f72206120626c6f636b207468617420686173206e6f74206265656e2060648201527f70726f706f736564000000000000000000000000000000000000000000000000608482015260a401610609565b600354610a5a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604660248201527f4c324f75747075744f7261636c653a2063616e6e6f7420676574206f7574707560448201527f74206173206e6f206f7574707574732068617665206265656e2070726f706f7360648201527f6564207965740000000000000000000000000000000000000000000000000000608482015260a401610609565b6003546000905b80821015610af35760006002610a7783856116c5565b610a81919061170c565b90508460038281548110610a9757610a97611620565b600091825260209091206002909102016001015470010000000000000000000000000000000090046fffffffffffffffffffffffffffffffff161015610ae957610ae28160016116c5565b9250610aed565b8091505b50610a61565b5092915050565b60045473ffffffffffffffffffffffffffffffffffffffff163314610ba1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603e60248201527f4c324f75747075744f7261636c653a206f6e6c7920746865206368616c6c656e60448201527f67657220616464726573732063616e2064656c657465206f75747075747300006064820152608401610609565b6003548110610c58576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604360248201527f4c324f75747075744f7261636c653a2063616e6e6f742064656c657465206f7560448201527f747075747320616674657220746865206c6174657374206f757470757420696e60648201527f6465780000000000000000000000000000000000000000000000000000000000608482015260a401610609565b7f000000000000000000000000000000000000000000000000000000000000000060038281548110610c8c57610c8c611620565b6000918252602090912060016002909202010154610cbc906fffffffffffffffffffffffffffffffff1642611609565b10610d6f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604660248201527f4c324f75747075744f7261636c653a2063616e6e6f742064656c657465206f7560448201527f74707574732074686174206861766520616c7265616479206265656e2066696e60648201527f616c697a65640000000000000000000000000000000000000000000000000000608482015260a401610609565b6000610d7a60035490565b90508160035581817f4ee37ac2c786ec85e87592d3c5c8a1dd66f8496dda3f125d9ea8ca5f657629b660405160405180910390a35050565b60055473ffffffffffffffffffffffffffffffffffffffff163314610e7f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f4c324f75747075744f7261636c653a206f6e6c79207468652070726f706f736560448201527f7220616464726573732063616e2070726f706f7365206e6577206f757470757460648201527f7300000000000000000000000000000000000000000000000000000000000000608482015260a401610609565b610e8761132d565b8314610f3b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f4c324f75747075744f7261636c653a20626c6f636b206e756d626572206d757360448201527f7420626520657175616c20746f206e65787420657870656374656420626c6f6360648201527f6b206e756d626572000000000000000000000000000000000000000000000000608482015260a401610609565b42610f45846112df565b10610fd2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f4c324f75747075744f7261636c653a2063616e6e6f742070726f706f7365204c60448201527f32206f757470757420696e2074686520667574757265000000000000000000006064820152608401610609565b8361105f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f4c324f75747075744f7261636c653a204c32206f75747075742070726f706f7360448201527f616c2063616e6e6f7420626520746865207a65726f20686173680000000000006064820152608401610609565b811561111b578181401461111b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604960248201527f4c324f75747075744f7261636c653a20626c6f636b206861736820646f65732060448201527f6e6f74206d61746368207468652068617368206174207468652065787065637460648201527f6564206865696768740000000000000000000000000000000000000000000000608482015260a401610609565b8261112560035490565b857fa7aaf2512769da4e444e3de247be2564225c2e7a8f74cfe528e46e17d24868e24260405161115791815260200190565b60405180910390a45050604080516060810182529283526fffffffffffffffffffffffffffffffff4281166020850190815292811691840191825260038054600181018255600091909152935160029094027fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b810194909455915190518216700100000000000000000000000000000000029116177fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85c90910155565b60408051606081018252600080825260208201819052918101919091526003828154811061124357611243611620565b600091825260209182902060408051606081018252600290930290910180548352600101546fffffffffffffffffffffffffffffffff8082169484019490945270010000000000000000000000000000000090049092169181019190915292915050565b604080516060810182526000808252602082018190529181019190915260036112cf836108e6565b8154811061124357611243611620565b60007f0000000000000000000000000000000000000000000000000000000000000000600154836113109190611609565b61131a9190611720565b60025461132791906116c5565b92915050565b60007f00000000000000000000000000000000000000000000000000000000000000006113586107be565b61082c91906116c5565b6060816000036113a557505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156113cf57806113b98161175d565b91506113c89050600a8361170c565b91506113a9565b60008167ffffffffffffffff8111156113ea576113ea611795565b6040519080825280601f01601f191660200182016040528015611414576020820181803683370190505b5090505b841561149757611429600183611609565b9150611436600a866117c4565b6114419060306116c5565b60f81b81838151811061145657611456611620565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350611490600a8661170c565b9450611418565b949350505050565b803573ffffffffffffffffffffffffffffffffffffffff811681146114c357600080fd5b919050565b600080600080608085870312156114de57600080fd5b84359350602085013592506114f56040860161149f565b91506115036060860161149f565b905092959194509250565b60005b83811015611529578181015183820152602001611511565b83811115611538576000848401525b50505050565b602081526000825180602084015261155d81604085016020870161150e565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b6000602082840312156115a157600080fd5b5035919050565b600080600080608085870312156115be57600080fd5b5050823594602084013594506040840135936060013592509050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008282101561161b5761161b6115da565b500390565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b6000845161166181846020890161150e565b80830190507f2e00000000000000000000000000000000000000000000000000000000000000808252855161169d816001850160208a0161150e565b600192019182015283516116b881600284016020880161150e565b0160020195945050505050565b600082198211156116d8576116d86115da565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b60008261171b5761171b6116dd565b500490565b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0483118215151615611758576117586115da565b500290565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361178e5761178e6115da565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000826117d3576117d36116dd565b50069056fea164736f6c634300080f000a",
}

// L2OutputOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use L2OutputOracleMetaData.ABI instead.
var L2OutputOracleABI = L2OutputOracleMetaData.ABI

// L2OutputOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2OutputOracleMetaData.Bin instead.
var L2OutputOracleBin = L2OutputOracleMetaData.Bin

// DeployL2OutputOracle deploys a new Ethereum contract, binding an instance of L2OutputOracle to it.
func DeployL2OutputOracle(auth *bind.TransactOpts, backend bind.ContractBackend, _submissionInterval *big.Int, _l2BlockTime *big.Int, _finalizationPeriodSeconds *big.Int) (common.Address, *types.Transaction, *L2OutputOracle, error) {
	parsed, err := L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2OutputOracleBin), backend, _submissionInterval, _l2BlockTime, _finalizationPeriodSeconds)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2OutputOracle{L2OutputOracleCaller: L2OutputOracleCaller{contract: contract}, L2OutputOracleTransactor: L2OutputOracleTransactor{contract: contract}, L2OutputOracleFilterer: L2OutputOracleFilterer{contract: contract}}, nil
}

// L2OutputOracle is an auto generated Go binding around an Ethereum contract.
type L2OutputOracle struct {
	L2OutputOracleCaller     // Read-only binding to the contract
	L2OutputOracleTransactor // Write-only binding to the contract
	L2OutputOracleFilterer   // Log filterer for contract events
}

// L2OutputOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2OutputOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2OutputOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2OutputOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2OutputOracleSession struct {
	Contract     *L2OutputOracle   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L2OutputOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2OutputOracleCallerSession struct {
	Contract *L2OutputOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// L2OutputOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2OutputOracleTransactorSession struct {
	Contract     *L2OutputOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// L2OutputOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2OutputOracleRaw struct {
	Contract *L2OutputOracle // Generic contract binding to access the raw methods on
}

// L2OutputOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2OutputOracleCallerRaw struct {
	Contract *L2OutputOracleCaller // Generic read-only contract binding to access the raw methods on
}

// L2OutputOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2OutputOracleTransactorRaw struct {
	Contract *L2OutputOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2OutputOracle creates a new instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracle(address common.Address, backend bind.ContractBackend) (*L2OutputOracle, error) {
	contract, err := bindL2OutputOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracle{L2OutputOracleCaller: L2OutputOracleCaller{contract: contract}, L2OutputOracleTransactor: L2OutputOracleTransactor{contract: contract}, L2OutputOracleFilterer: L2OutputOracleFilterer{contract: contract}}, nil
}

// NewL2OutputOracleCaller creates a new read-only instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleCaller(address common.Address, caller bind.ContractCaller) (*L2OutputOracleCaller, error) {
	contract, err := bindL2OutputOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleCaller{contract: contract}, nil
}

// NewL2OutputOracleTransactor creates a new write-only instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*L2OutputOracleTransactor, error) {
	contract, err := bindL2OutputOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleTransactor{contract: contract}, nil
}

// NewL2OutputOracleFilterer creates a new log filterer instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*L2OutputOracleFilterer, error) {
	contract, err := bindL2OutputOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleFilterer{contract: contract}, nil
}

// bindL2OutputOracle binds a generic wrapper to an already deployed contract.
func bindL2OutputOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2OutputOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2OutputOracle *L2OutputOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2OutputOracle.Contract.L2OutputOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2OutputOracle *L2OutputOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.L2OutputOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2OutputOracle *L2OutputOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.L2OutputOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2OutputOracle *L2OutputOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2OutputOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2OutputOracle *L2OutputOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2OutputOracle *L2OutputOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.contract.Transact(opts, method, params...)
}

// CHALLENGER is a free data retrieval call binding the contract method 0x6b4d98dd.
//
// Solidity: function CHALLENGER() view returns(address)
func (_L2OutputOracle *L2OutputOracleCaller) CHALLENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "CHALLENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CHALLENGER is a free data retrieval call binding the contract method 0x6b4d98dd.
//
// Solidity: function CHALLENGER() view returns(address)
func (_L2OutputOracle *L2OutputOracleSession) CHALLENGER() (common.Address, error) {
	return _L2OutputOracle.Contract.CHALLENGER(&_L2OutputOracle.CallOpts)
}

// CHALLENGER is a free data retrieval call binding the contract method 0x6b4d98dd.
//
// Solidity: function CHALLENGER() view returns(address)
func (_L2OutputOracle *L2OutputOracleCallerSession) CHALLENGER() (common.Address, error) {
	return _L2OutputOracle.Contract.CHALLENGER(&_L2OutputOracle.CallOpts)
}

// FINALIZATIONPERIODSECONDS is a free data retrieval call binding the contract method 0xf4daa291.
//
// Solidity: function FINALIZATION_PERIOD_SECONDS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) FINALIZATIONPERIODSECONDS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "FINALIZATION_PERIOD_SECONDS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FINALIZATIONPERIODSECONDS is a free data retrieval call binding the contract method 0xf4daa291.
//
// Solidity: function FINALIZATION_PERIOD_SECONDS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) FINALIZATIONPERIODSECONDS() (*big.Int, error) {
	return _L2OutputOracle.Contract.FINALIZATIONPERIODSECONDS(&_L2OutputOracle.CallOpts)
}

// FINALIZATIONPERIODSECONDS is a free data retrieval call binding the contract method 0xf4daa291.
//
// Solidity: function FINALIZATION_PERIOD_SECONDS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) FINALIZATIONPERIODSECONDS() (*big.Int, error) {
	return _L2OutputOracle.Contract.FINALIZATIONPERIODSECONDS(&_L2OutputOracle.CallOpts)
}

// L2BLOCKTIME is a free data retrieval call binding the contract method 0x002134cc.
//
// Solidity: function L2_BLOCK_TIME() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) L2BLOCKTIME(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "L2_BLOCK_TIME")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2BLOCKTIME is a free data retrieval call binding the contract method 0x002134cc.
//
// Solidity: function L2_BLOCK_TIME() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) L2BLOCKTIME() (*big.Int, error) {
	return _L2OutputOracle.Contract.L2BLOCKTIME(&_L2OutputOracle.CallOpts)
}

// L2BLOCKTIME is a free data retrieval call binding the contract method 0x002134cc.
//
// Solidity: function L2_BLOCK_TIME() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) L2BLOCKTIME() (*big.Int, error) {
	return _L2OutputOracle.Contract.L2BLOCKTIME(&_L2OutputOracle.CallOpts)
}

// PROPOSER is a free data retrieval call binding the contract method 0xbffa7f0f.
//
// Solidity: function PROPOSER() view returns(address)
func (_L2OutputOracle *L2OutputOracleCaller) PROPOSER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "PROPOSER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PROPOSER is a free data retrieval call binding the contract method 0xbffa7f0f.
//
// Solidity: function PROPOSER() view returns(address)
func (_L2OutputOracle *L2OutputOracleSession) PROPOSER() (common.Address, error) {
	return _L2OutputOracle.Contract.PROPOSER(&_L2OutputOracle.CallOpts)
}

// PROPOSER is a free data retrieval call binding the contract method 0xbffa7f0f.
//
// Solidity: function PROPOSER() view returns(address)
func (_L2OutputOracle *L2OutputOracleCallerSession) PROPOSER() (common.Address, error) {
	return _L2OutputOracle.Contract.PROPOSER(&_L2OutputOracle.CallOpts)
}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) SUBMISSIONINTERVAL(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "SUBMISSION_INTERVAL")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) SUBMISSIONINTERVAL() (*big.Int, error) {
	return _L2OutputOracle.Contract.SUBMISSIONINTERVAL(&_L2OutputOracle.CallOpts)
}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) SUBMISSIONINTERVAL() (*big.Int, error) {
	return _L2OutputOracle.Contract.SUBMISSIONINTERVAL(&_L2OutputOracle.CallOpts)
}

// Challenger is a free data retrieval call binding the contract method 0x534db0e2.
//
// Solidity: function challenger() view returns(address)
func (_L2OutputOracle *L2OutputOracleCaller) Challenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "challenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Challenger is a free data retrieval call binding the contract method 0x534db0e2.
//
// Solidity: function challenger() view returns(address)
func (_L2OutputOracle *L2OutputOracleSession) Challenger() (common.Address, error) {
	return _L2OutputOracle.Contract.Challenger(&_L2OutputOracle.CallOpts)
}

// Challenger is a free data retrieval call binding the contract method 0x534db0e2.
//
// Solidity: function challenger() view returns(address)
func (_L2OutputOracle *L2OutputOracleCallerSession) Challenger() (common.Address, error) {
	return _L2OutputOracle.Contract.Challenger(&_L2OutputOracle.CallOpts)
}

// ComputeL2Timestamp is a free data retrieval call binding the contract method 0xd1de856c.
//
// Solidity: function computeL2Timestamp(uint256 _l2BlockNumber) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) ComputeL2Timestamp(opts *bind.CallOpts, _l2BlockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "computeL2Timestamp", _l2BlockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ComputeL2Timestamp is a free data retrieval call binding the contract method 0xd1de856c.
//
// Solidity: function computeL2Timestamp(uint256 _l2BlockNumber) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) ComputeL2Timestamp(_l2BlockNumber *big.Int) (*big.Int, error) {
	return _L2OutputOracle.Contract.ComputeL2Timestamp(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// ComputeL2Timestamp is a free data retrieval call binding the contract method 0xd1de856c.
//
// Solidity: function computeL2Timestamp(uint256 _l2BlockNumber) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) ComputeL2Timestamp(_l2BlockNumber *big.Int) (*big.Int, error) {
	return _L2OutputOracle.Contract.ComputeL2Timestamp(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// FinalizationPeriodSeconds is a free data retrieval call binding the contract method 0xce5db8d6.
//
// Solidity: function finalizationPeriodSeconds() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) FinalizationPeriodSeconds(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "finalizationPeriodSeconds")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FinalizationPeriodSeconds is a free data retrieval call binding the contract method 0xce5db8d6.
//
// Solidity: function finalizationPeriodSeconds() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) FinalizationPeriodSeconds() (*big.Int, error) {
	return _L2OutputOracle.Contract.FinalizationPeriodSeconds(&_L2OutputOracle.CallOpts)
}

// FinalizationPeriodSeconds is a free data retrieval call binding the contract method 0xce5db8d6.
//
// Solidity: function finalizationPeriodSeconds() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) FinalizationPeriodSeconds() (*big.Int, error) {
	return _L2OutputOracle.Contract.FinalizationPeriodSeconds(&_L2OutputOracle.CallOpts)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2OutputIndex) view returns((bytes32,uint128,uint128))
func (_L2OutputOracle *L2OutputOracleCaller) GetL2Output(opts *bind.CallOpts, _l2OutputIndex *big.Int) (TypesOutputProposal, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "getL2Output", _l2OutputIndex)

	if err != nil {
		return *new(TypesOutputProposal), err
	}

	out0 := *abi.ConvertType(out[0], new(TypesOutputProposal)).(*TypesOutputProposal)

	return out0, err

}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2OutputIndex) view returns((bytes32,uint128,uint128))
func (_L2OutputOracle *L2OutputOracleSession) GetL2Output(_l2OutputIndex *big.Int) (TypesOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2OutputIndex)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2OutputIndex) view returns((bytes32,uint128,uint128))
func (_L2OutputOracle *L2OutputOracleCallerSession) GetL2Output(_l2OutputIndex *big.Int) (TypesOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2OutputIndex)
}

// GetL2OutputAfter is a free data retrieval call binding the contract method 0xcf8e5cf0.
//
// Solidity: function getL2OutputAfter(uint256 _l2BlockNumber) view returns((bytes32,uint128,uint128))
func (_L2OutputOracle *L2OutputOracleCaller) GetL2OutputAfter(opts *bind.CallOpts, _l2BlockNumber *big.Int) (TypesOutputProposal, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "getL2OutputAfter", _l2BlockNumber)

	if err != nil {
		return *new(TypesOutputProposal), err
	}

	out0 := *abi.ConvertType(out[0], new(TypesOutputProposal)).(*TypesOutputProposal)

	return out0, err

}

// GetL2OutputAfter is a free data retrieval call binding the contract method 0xcf8e5cf0.
//
// Solidity: function getL2OutputAfter(uint256 _l2BlockNumber) view returns((bytes32,uint128,uint128))
func (_L2OutputOracle *L2OutputOracleSession) GetL2OutputAfter(_l2BlockNumber *big.Int) (TypesOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2OutputAfter(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// GetL2OutputAfter is a free data retrieval call binding the contract method 0xcf8e5cf0.
//
// Solidity: function getL2OutputAfter(uint256 _l2BlockNumber) view returns((bytes32,uint128,uint128))
func (_L2OutputOracle *L2OutputOracleCallerSession) GetL2OutputAfter(_l2BlockNumber *big.Int) (TypesOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2OutputAfter(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// GetL2OutputIndexAfter is a free data retrieval call binding the contract method 0x7f006420.
//
// Solidity: function getL2OutputIndexAfter(uint256 _l2BlockNumber) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) GetL2OutputIndexAfter(opts *bind.CallOpts, _l2BlockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "getL2OutputIndexAfter", _l2BlockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL2OutputIndexAfter is a free data retrieval call binding the contract method 0x7f006420.
//
// Solidity: function getL2OutputIndexAfter(uint256 _l2BlockNumber) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) GetL2OutputIndexAfter(_l2BlockNumber *big.Int) (*big.Int, error) {
	return _L2OutputOracle.Contract.GetL2OutputIndexAfter(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// GetL2OutputIndexAfter is a free data retrieval call binding the contract method 0x7f006420.
//
// Solidity: function getL2OutputIndexAfter(uint256 _l2BlockNumber) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) GetL2OutputIndexAfter(_l2BlockNumber *big.Int) (*big.Int, error) {
	return _L2OutputOracle.Contract.GetL2OutputIndexAfter(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// L2BlockTime is a free data retrieval call binding the contract method 0x93991af3.
//
// Solidity: function l2BlockTime() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) L2BlockTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "l2BlockTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2BlockTime is a free data retrieval call binding the contract method 0x93991af3.
//
// Solidity: function l2BlockTime() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) L2BlockTime() (*big.Int, error) {
	return _L2OutputOracle.Contract.L2BlockTime(&_L2OutputOracle.CallOpts)
}

// L2BlockTime is a free data retrieval call binding the contract method 0x93991af3.
//
// Solidity: function l2BlockTime() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) L2BlockTime() (*big.Int, error) {
	return _L2OutputOracle.Contract.L2BlockTime(&_L2OutputOracle.CallOpts)
}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) LatestBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "latestBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) LatestBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestBlockNumber(&_L2OutputOracle.CallOpts)
}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) LatestBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestBlockNumber(&_L2OutputOracle.CallOpts)
}

// LatestOutputIndex is a free data retrieval call binding the contract method 0x69f16eec.
//
// Solidity: function latestOutputIndex() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) LatestOutputIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "latestOutputIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestOutputIndex is a free data retrieval call binding the contract method 0x69f16eec.
//
// Solidity: function latestOutputIndex() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) LatestOutputIndex() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestOutputIndex(&_L2OutputOracle.CallOpts)
}

// LatestOutputIndex is a free data retrieval call binding the contract method 0x69f16eec.
//
// Solidity: function latestOutputIndex() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) LatestOutputIndex() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestOutputIndex(&_L2OutputOracle.CallOpts)
}

// NextBlockNumber is a free data retrieval call binding the contract method 0xdcec3348.
//
// Solidity: function nextBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) NextBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "nextBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextBlockNumber is a free data retrieval call binding the contract method 0xdcec3348.
//
// Solidity: function nextBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) NextBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextBlockNumber(&_L2OutputOracle.CallOpts)
}

// NextBlockNumber is a free data retrieval call binding the contract method 0xdcec3348.
//
// Solidity: function nextBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) NextBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextBlockNumber(&_L2OutputOracle.CallOpts)
}

// NextOutputIndex is a free data retrieval call binding the contract method 0x6abcf563.
//
// Solidity: function nextOutputIndex() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) NextOutputIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "nextOutputIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextOutputIndex is a free data retrieval call binding the contract method 0x6abcf563.
//
// Solidity: function nextOutputIndex() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) NextOutputIndex() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextOutputIndex(&_L2OutputOracle.CallOpts)
}

// NextOutputIndex is a free data retrieval call binding the contract method 0x6abcf563.
//
// Solidity: function nextOutputIndex() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) NextOutputIndex() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextOutputIndex(&_L2OutputOracle.CallOpts)
}

// Proposer is a free data retrieval call binding the contract method 0xa8e4fb90.
//
// Solidity: function proposer() view returns(address)
func (_L2OutputOracle *L2OutputOracleCaller) Proposer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "proposer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Proposer is a free data retrieval call binding the contract method 0xa8e4fb90.
//
// Solidity: function proposer() view returns(address)
func (_L2OutputOracle *L2OutputOracleSession) Proposer() (common.Address, error) {
	return _L2OutputOracle.Contract.Proposer(&_L2OutputOracle.CallOpts)
}

// Proposer is a free data retrieval call binding the contract method 0xa8e4fb90.
//
// Solidity: function proposer() view returns(address)
func (_L2OutputOracle *L2OutputOracleCallerSession) Proposer() (common.Address, error) {
	return _L2OutputOracle.Contract.Proposer(&_L2OutputOracle.CallOpts)
}

// StartingBlockNumber is a free data retrieval call binding the contract method 0x70872aa5.
//
// Solidity: function startingBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) StartingBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "startingBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StartingBlockNumber is a free data retrieval call binding the contract method 0x70872aa5.
//
// Solidity: function startingBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) StartingBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.StartingBlockNumber(&_L2OutputOracle.CallOpts)
}

// StartingBlockNumber is a free data retrieval call binding the contract method 0x70872aa5.
//
// Solidity: function startingBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) StartingBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.StartingBlockNumber(&_L2OutputOracle.CallOpts)
}

// StartingTimestamp is a free data retrieval call binding the contract method 0x88786272.
//
// Solidity: function startingTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) StartingTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "startingTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StartingTimestamp is a free data retrieval call binding the contract method 0x88786272.
//
// Solidity: function startingTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) StartingTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.StartingTimestamp(&_L2OutputOracle.CallOpts)
}

// StartingTimestamp is a free data retrieval call binding the contract method 0x88786272.
//
// Solidity: function startingTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) StartingTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.StartingTimestamp(&_L2OutputOracle.CallOpts)
}

// SubmissionInterval is a free data retrieval call binding the contract method 0xe1a41bcf.
//
// Solidity: function submissionInterval() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) SubmissionInterval(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "submissionInterval")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SubmissionInterval is a free data retrieval call binding the contract method 0xe1a41bcf.
//
// Solidity: function submissionInterval() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) SubmissionInterval() (*big.Int, error) {
	return _L2OutputOracle.Contract.SubmissionInterval(&_L2OutputOracle.CallOpts)
}

// SubmissionInterval is a free data retrieval call binding the contract method 0xe1a41bcf.
//
// Solidity: function submissionInterval() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) SubmissionInterval() (*big.Int, error) {
	return _L2OutputOracle.Contract.SubmissionInterval(&_L2OutputOracle.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2OutputOracle *L2OutputOracleCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2OutputOracle *L2OutputOracleSession) Version() (string, error) {
	return _L2OutputOracle.Contract.Version(&_L2OutputOracle.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2OutputOracle *L2OutputOracleCallerSession) Version() (string, error) {
	return _L2OutputOracle.Contract.Version(&_L2OutputOracle.CallOpts)
}

// DeleteL2Outputs is a paid mutator transaction binding the contract method 0x89c44cbb.
//
// Solidity: function deleteL2Outputs(uint256 _l2OutputIndex) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) DeleteL2Outputs(opts *bind.TransactOpts, _l2OutputIndex *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "deleteL2Outputs", _l2OutputIndex)
}

// DeleteL2Outputs is a paid mutator transaction binding the contract method 0x89c44cbb.
//
// Solidity: function deleteL2Outputs(uint256 _l2OutputIndex) returns()
func (_L2OutputOracle *L2OutputOracleSession) DeleteL2Outputs(_l2OutputIndex *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Outputs(&_L2OutputOracle.TransactOpts, _l2OutputIndex)
}

// DeleteL2Outputs is a paid mutator transaction binding the contract method 0x89c44cbb.
//
// Solidity: function deleteL2Outputs(uint256 _l2OutputIndex) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) DeleteL2Outputs(_l2OutputIndex *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Outputs(&_L2OutputOracle.TransactOpts, _l2OutputIndex)
}

// Initialize is a paid mutator transaction binding the contract method 0x019e2729.
//
// Solidity: function initialize(uint256 _startingBlockNumber, uint256 _startingTimestamp, address _proposer, address _challenger) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) Initialize(opts *bind.TransactOpts, _startingBlockNumber *big.Int, _startingTimestamp *big.Int, _proposer common.Address, _challenger common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "initialize", _startingBlockNumber, _startingTimestamp, _proposer, _challenger)
}

// Initialize is a paid mutator transaction binding the contract method 0x019e2729.
//
// Solidity: function initialize(uint256 _startingBlockNumber, uint256 _startingTimestamp, address _proposer, address _challenger) returns()
func (_L2OutputOracle *L2OutputOracleSession) Initialize(_startingBlockNumber *big.Int, _startingTimestamp *big.Int, _proposer common.Address, _challenger common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.Initialize(&_L2OutputOracle.TransactOpts, _startingBlockNumber, _startingTimestamp, _proposer, _challenger)
}

// Initialize is a paid mutator transaction binding the contract method 0x019e2729.
//
// Solidity: function initialize(uint256 _startingBlockNumber, uint256 _startingTimestamp, address _proposer, address _challenger) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) Initialize(_startingBlockNumber *big.Int, _startingTimestamp *big.Int, _proposer common.Address, _challenger common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.Initialize(&_L2OutputOracle.TransactOpts, _startingBlockNumber, _startingTimestamp, _proposer, _challenger)
}

// ProposeL2Output is a paid mutator transaction binding the contract method 0x9aaab648.
//
// Solidity: function proposeL2Output(bytes32 _outputRoot, uint256 _l2BlockNumber, bytes32 _l1BlockHash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactor) ProposeL2Output(opts *bind.TransactOpts, _outputRoot [32]byte, _l2BlockNumber *big.Int, _l1BlockHash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "proposeL2Output", _outputRoot, _l2BlockNumber, _l1BlockHash, _l1BlockNumber)
}

// ProposeL2Output is a paid mutator transaction binding the contract method 0x9aaab648.
//
// Solidity: function proposeL2Output(bytes32 _outputRoot, uint256 _l2BlockNumber, bytes32 _l1BlockHash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleSession) ProposeL2Output(_outputRoot [32]byte, _l2BlockNumber *big.Int, _l1BlockHash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.ProposeL2Output(&_L2OutputOracle.TransactOpts, _outputRoot, _l2BlockNumber, _l1BlockHash, _l1BlockNumber)
}

// ProposeL2Output is a paid mutator transaction binding the contract method 0x9aaab648.
//
// Solidity: function proposeL2Output(bytes32 _outputRoot, uint256 _l2BlockNumber, bytes32 _l1BlockHash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) ProposeL2Output(_outputRoot [32]byte, _l2BlockNumber *big.Int, _l1BlockHash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.ProposeL2Output(&_L2OutputOracle.TransactOpts, _outputRoot, _l2BlockNumber, _l1BlockHash, _l1BlockNumber)
}

// L2OutputOracleInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L2OutputOracle contract.
type L2OutputOracleInitializedIterator struct {
	Event *L2OutputOracleInitialized // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleInitialized)
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
		it.Event = new(L2OutputOracleInitialized)
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
func (it *L2OutputOracleInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleInitialized represents a Initialized event raised by the L2OutputOracle contract.
type L2OutputOracleInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterInitialized(opts *bind.FilterOpts) (*L2OutputOracleInitializedIterator, error) {

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleInitializedIterator{contract: _L2OutputOracle.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L2OutputOracleInitialized) (event.Subscription, error) {

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleInitialized)
				if err := _L2OutputOracle.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_L2OutputOracle *L2OutputOracleFilterer) ParseInitialized(log types.Log) (*L2OutputOracleInitialized, error) {
	event := new(L2OutputOracleInitialized)
	if err := _L2OutputOracle.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleOutputProposedIterator is returned from FilterOutputProposed and is used to iterate over the raw logs and unpacked data for OutputProposed events raised by the L2OutputOracle contract.
type L2OutputOracleOutputProposedIterator struct {
	Event *L2OutputOracleOutputProposed // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleOutputProposedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleOutputProposed)
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
		it.Event = new(L2OutputOracleOutputProposed)
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
func (it *L2OutputOracleOutputProposedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleOutputProposedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleOutputProposed represents a OutputProposed event raised by the L2OutputOracle contract.
type L2OutputOracleOutputProposed struct {
	OutputRoot    [32]byte
	L2OutputIndex *big.Int
	L2BlockNumber *big.Int
	L1Timestamp   *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOutputProposed is a free log retrieval operation binding the contract event 0xa7aaf2512769da4e444e3de247be2564225c2e7a8f74cfe528e46e17d24868e2.
//
// Solidity: event OutputProposed(bytes32 indexed outputRoot, uint256 indexed l2OutputIndex, uint256 indexed l2BlockNumber, uint256 l1Timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterOutputProposed(opts *bind.FilterOpts, outputRoot [][32]byte, l2OutputIndex []*big.Int, l2BlockNumber []*big.Int) (*L2OutputOracleOutputProposedIterator, error) {

	var outputRootRule []interface{}
	for _, outputRootItem := range outputRoot {
		outputRootRule = append(outputRootRule, outputRootItem)
	}
	var l2OutputIndexRule []interface{}
	for _, l2OutputIndexItem := range l2OutputIndex {
		l2OutputIndexRule = append(l2OutputIndexRule, l2OutputIndexItem)
	}
	var l2BlockNumberRule []interface{}
	for _, l2BlockNumberItem := range l2BlockNumber {
		l2BlockNumberRule = append(l2BlockNumberRule, l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "OutputProposed", outputRootRule, l2OutputIndexRule, l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleOutputProposedIterator{contract: _L2OutputOracle.contract, event: "OutputProposed", logs: logs, sub: sub}, nil
}

// WatchOutputProposed is a free log subscription operation binding the contract event 0xa7aaf2512769da4e444e3de247be2564225c2e7a8f74cfe528e46e17d24868e2.
//
// Solidity: event OutputProposed(bytes32 indexed outputRoot, uint256 indexed l2OutputIndex, uint256 indexed l2BlockNumber, uint256 l1Timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchOutputProposed(opts *bind.WatchOpts, sink chan<- *L2OutputOracleOutputProposed, outputRoot [][32]byte, l2OutputIndex []*big.Int, l2BlockNumber []*big.Int) (event.Subscription, error) {

	var outputRootRule []interface{}
	for _, outputRootItem := range outputRoot {
		outputRootRule = append(outputRootRule, outputRootItem)
	}
	var l2OutputIndexRule []interface{}
	for _, l2OutputIndexItem := range l2OutputIndex {
		l2OutputIndexRule = append(l2OutputIndexRule, l2OutputIndexItem)
	}
	var l2BlockNumberRule []interface{}
	for _, l2BlockNumberItem := range l2BlockNumber {
		l2BlockNumberRule = append(l2BlockNumberRule, l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "OutputProposed", outputRootRule, l2OutputIndexRule, l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleOutputProposed)
				if err := _L2OutputOracle.contract.UnpackLog(event, "OutputProposed", log); err != nil {
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

// ParseOutputProposed is a log parse operation binding the contract event 0xa7aaf2512769da4e444e3de247be2564225c2e7a8f74cfe528e46e17d24868e2.
//
// Solidity: event OutputProposed(bytes32 indexed outputRoot, uint256 indexed l2OutputIndex, uint256 indexed l2BlockNumber, uint256 l1Timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseOutputProposed(log types.Log) (*L2OutputOracleOutputProposed, error) {
	event := new(L2OutputOracleOutputProposed)
	if err := _L2OutputOracle.contract.UnpackLog(event, "OutputProposed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleOutputsDeletedIterator is returned from FilterOutputsDeleted and is used to iterate over the raw logs and unpacked data for OutputsDeleted events raised by the L2OutputOracle contract.
type L2OutputOracleOutputsDeletedIterator struct {
	Event *L2OutputOracleOutputsDeleted // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleOutputsDeletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleOutputsDeleted)
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
		it.Event = new(L2OutputOracleOutputsDeleted)
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
func (it *L2OutputOracleOutputsDeletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleOutputsDeletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleOutputsDeleted represents a OutputsDeleted event raised by the L2OutputOracle contract.
type L2OutputOracleOutputsDeleted struct {
	PrevNextOutputIndex *big.Int
	NewNextOutputIndex  *big.Int
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterOutputsDeleted is a free log retrieval operation binding the contract event 0x4ee37ac2c786ec85e87592d3c5c8a1dd66f8496dda3f125d9ea8ca5f657629b6.
//
// Solidity: event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterOutputsDeleted(opts *bind.FilterOpts, prevNextOutputIndex []*big.Int, newNextOutputIndex []*big.Int) (*L2OutputOracleOutputsDeletedIterator, error) {

	var prevNextOutputIndexRule []interface{}
	for _, prevNextOutputIndexItem := range prevNextOutputIndex {
		prevNextOutputIndexRule = append(prevNextOutputIndexRule, prevNextOutputIndexItem)
	}
	var newNextOutputIndexRule []interface{}
	for _, newNextOutputIndexItem := range newNextOutputIndex {
		newNextOutputIndexRule = append(newNextOutputIndexRule, newNextOutputIndexItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "OutputsDeleted", prevNextOutputIndexRule, newNextOutputIndexRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleOutputsDeletedIterator{contract: _L2OutputOracle.contract, event: "OutputsDeleted", logs: logs, sub: sub}, nil
}

// WatchOutputsDeleted is a free log subscription operation binding the contract event 0x4ee37ac2c786ec85e87592d3c5c8a1dd66f8496dda3f125d9ea8ca5f657629b6.
//
// Solidity: event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchOutputsDeleted(opts *bind.WatchOpts, sink chan<- *L2OutputOracleOutputsDeleted, prevNextOutputIndex []*big.Int, newNextOutputIndex []*big.Int) (event.Subscription, error) {

	var prevNextOutputIndexRule []interface{}
	for _, prevNextOutputIndexItem := range prevNextOutputIndex {
		prevNextOutputIndexRule = append(prevNextOutputIndexRule, prevNextOutputIndexItem)
	}
	var newNextOutputIndexRule []interface{}
	for _, newNextOutputIndexItem := range newNextOutputIndex {
		newNextOutputIndexRule = append(newNextOutputIndexRule, newNextOutputIndexItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "OutputsDeleted", prevNextOutputIndexRule, newNextOutputIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleOutputsDeleted)
				if err := _L2OutputOracle.contract.UnpackLog(event, "OutputsDeleted", log); err != nil {
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

// ParseOutputsDeleted is a log parse operation binding the contract event 0x4ee37ac2c786ec85e87592d3c5c8a1dd66f8496dda3f125d9ea8ca5f657629b6.
//
// Solidity: event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseOutputsDeleted(log types.Log) (*L2OutputOracleOutputsDeleted, error) {
	event := new(L2OutputOracleOutputsDeleted)
	if err := _L2OutputOracle.contract.UnpackLog(event, "OutputsDeleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
