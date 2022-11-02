// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const L2StandardBridgeStorageLayoutJSON = "{\"storage\":[{\"astId\":28410,\"contract\":\"contracts/L2/L2StandardBridge.sol:L2StandardBridge\",\"label\":\"spacer_0_0_20\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_address\"},{\"astId\":28413,\"contract\":\"contracts/L2/L2StandardBridge.sol:L2StandardBridge\",\"label\":\"spacer_1_0_20\",\"offset\":0,\"slot\":\"1\",\"type\":\"t_address\"},{\"astId\":28420,\"contract\":\"contracts/L2/L2StandardBridge.sol:L2StandardBridge\",\"label\":\"deposits\",\"offset\":0,\"slot\":\"2\",\"type\":\"t_mapping(t_address,t_mapping(t_address,t_uint256))\"},{\"astId\":28425,\"contract\":\"contracts/L2/L2StandardBridge.sol:L2StandardBridge\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"3\",\"type\":\"t_array(t_uint256)47_storage\"}],\"types\":{\"t_address\":{\"encoding\":\"inplace\",\"label\":\"address\",\"numberOfBytes\":\"20\"},\"t_array(t_uint256)47_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[47]\",\"numberOfBytes\":\"1504\"},\"t_mapping(t_address,t_mapping(t_address,t_uint256))\":{\"encoding\":\"mapping\",\"label\":\"mapping(address =\u003e mapping(address =\u003e uint256))\",\"numberOfBytes\":\"32\",\"key\":\"t_address\",\"value\":\"t_mapping(t_address,t_uint256)\"},\"t_mapping(t_address,t_uint256)\":{\"encoding\":\"mapping\",\"label\":\"mapping(address =\u003e uint256)\",\"numberOfBytes\":\"32\",\"key\":\"t_address\",\"value\":\"t_uint256\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"}}}"

var L2StandardBridgeStorageLayout = new(solc.StorageLayout)

var L2StandardBridgeDeployedBin = "0x6080604052600436106100d65760003560e01c806354fd4d501161007f5780638f601f66116100595780638f601f66146102c1578063a3a7954814610307578063c89701a21461031a578063e11013dd1461034e57600080fd5b806354fd4d501461026c578063662a633a1461028e57806387087623146102a157600080fd5b806332b7006d116100b057806332b7006d146101db5780633cb747bf146101ee578063540abf731461024c57600080fd5b80630166a07a1461019557806309fc8843146101b55780631635f5fd146101c857600080fd5b3661019057333b1561016f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f4100000000000000000060648201526084015b60405180910390fd5b61018e33333462030d4060405180602001604052806000815250610361565b005b600080fd5b3480156101a157600080fd5b5061018e6101b0366004611ff2565b6105a0565b61018e6101c33660046120a3565b6109d4565b61018e6101d63660046120f6565b610aab565b61018e6101e9366004612169565b610fa7565b3480156101fa57600080fd5b506102227f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b34801561025857600080fd5b5061018e6102673660046121bd565b61104c565b34801561027857600080fd5b50610281611065565b60405161024391906122aa565b61018e61029c366004611ff2565b611108565b3480156102ad57600080fd5b5061018e6102bc3660046122bd565b6111f5565b3480156102cd57600080fd5b506102f96102dc366004612340565b600260209081526000928352604080842090915290825290205481565b604051908152602001610243565b61018e6103153660046122bd565b611294565b34801561032657600080fd5b506102227f000000000000000000000000000000000000000000000000000000000000000081565b61018e61035c366004612379565b6112a3565b8234146103f0576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603e60248201527f5374616e646172644272696467653a206272696467696e6720455448206d757360448201527f7420696e636c7564652073756666696369656e74204554482076616c756500006064820152608401610166565b8373ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff167f2849b43074093a05396b6f2a937dee8565b15a48a7b3d4bffb732a5017380af5858460405161044f9291906123dc565b60405180910390a37f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16633dbb202b847f0000000000000000000000000000000000000000000000000000000000000000631635f5fd60e01b898989886040516024016104d494939291906123f5565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e086901b90921682526105679291889060040161243e565b6000604051808303818588803b15801561058057600080fd5b505af1158015610594573d6000803e3d6000fd5b50505050505050505050565b3373ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000161480156106be57507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b8152600401602060405180830381865afa158015610682573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906106a69190612483565b73ffffffffffffffffffffffffffffffffffffffff16145b610770576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20746865206f7468657220627269646760648201527f6500000000000000000000000000000000000000000000000000000000000000608482015260a401610166565b610779876112ec565b156108c757610788878761134e565b61083a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f5374616e646172644272696467653a2077726f6e672072656d6f746520746f6b60448201527f656e20666f72204f7074696d69736d204d696e7461626c65204552433230206c60648201527f6f63616c20746f6b656e00000000000000000000000000000000000000000000608482015260a401610166565b6040517f40c10f1900000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8581166004830152602482018590528816906340c10f1990604401600060405180830381600087803b1580156108aa57600080fd5b505af11580156108be573d6000803e3d6000fd5b50505050610949565b73ffffffffffffffffffffffffffffffffffffffff8088166000908152600260209081526040808320938a16835292905220546109059084906124cf565b73ffffffffffffffffffffffffffffffffffffffff8089166000818152600260209081526040808320948c16835293905291909120919091556109499085856113f5565b8473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff167fd59c65b35445225835c83f50b6ede06a7be047d22e357073e250d9af537518cd878787876040516109c3949392919061252f565b60405180910390a450505050505050565b333b15610a63576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f410000000000000000006064820152608401610166565b610aa63333348686868080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061036192505050565b505050565b3373ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016148015610bc957507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b8152600401602060405180830381865afa158015610b8d573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610bb19190612483565b73ffffffffffffffffffffffffffffffffffffffff16145b610c7b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20746865206f7468657220627269646760648201527f6500000000000000000000000000000000000000000000000000000000000000608482015260a401610166565b823414610d0a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f5374616e646172644272696467653a20616d6f756e742073656e7420646f657360448201527f206e6f74206d6174636820616d6f756e742072657175697265640000000000006064820152608401610166565b3073ffffffffffffffffffffffffffffffffffffffff851603610daf576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f5374616e646172644272696467653a2063616e6e6f742073656e6420746f207360448201527f656c6600000000000000000000000000000000000000000000000000000000006064820152608401610166565b7f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff1603610e8a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602860248201527f5374616e646172644272696467653a2063616e6e6f742073656e6420746f206d60448201527f657373656e6765720000000000000000000000000000000000000000000000006064820152608401610166565b8373ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff167f31b2166ff604fc5672ea5df08a78081d2bc6d746cadce880747f3643d819e83d858585604051610eeb93929190612565565b60405180910390a36000610f10855a86604051806020016040528060008152506114c9565b905080610f9f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f5374616e646172644272696467653a20455448207472616e736665722066616960448201527f6c656400000000000000000000000000000000000000000000000000000000006064820152608401610166565b505050505050565b333b15611036576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f410000000000000000006064820152608401610166565b611045853333878787876114e3565b5050505050565b61105c8787338888888888611677565b50505050505050565b60606110907f0000000000000000000000000000000000000000000000000000000000000000611a35565b6110b97f0000000000000000000000000000000000000000000000000000000000000000611a35565b6110e27f0000000000000000000000000000000000000000000000000000000000000000611a35565b6040516020016110f493929190612588565b604051602081830303815290604052905090565b73ffffffffffffffffffffffffffffffffffffffff8716158015611155575073ffffffffffffffffffffffffffffffffffffffff861673deaddeaddeaddeaddeaddeaddeaddeaddead0000145b1561116c576111678585858585610aab565b61117b565b61117b868887878787876105a0565b8473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff167fb0444523268717a02698be47d0803aa7468c00acbed2f8bd93a0459cde61dd89878787876040516109c3949392919061252f565b333b15611284576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f410000000000000000006064820152608401610166565b610f9f8686333388888888611677565b610f9f863387878787876114e3565b6112e63385348686868080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061036192505050565b50505050565b6000611318827f1d1d8b6300000000000000000000000000000000000000000000000000000000611b72565b806113485750611348827fec4fc8e300000000000000000000000000000000000000000000000000000000611b72565b92915050565b60008273ffffffffffffffffffffffffffffffffffffffff1663c01e1bd66040518163ffffffff1660e01b8152600401602060405180830381865afa15801561139b573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906113bf9190612483565b73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1614905092915050565b60405173ffffffffffffffffffffffffffffffffffffffff8316602482015260448101829052610aa69084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff0000000000000000000000000000000000000000000000000000000090931692909217909152611b95565b600080600080845160208601878a8af19695505050505050565b60008773ffffffffffffffffffffffffffffffffffffffff1663c01e1bd66040518163ffffffff1660e01b8152600401602060405180830381865afa158015611530573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906115549190612483565b90507fffffffffffffffffffffffff215221522152215221522152215221522153000073ffffffffffffffffffffffffffffffffffffffff8916016115db576115d68787878787878080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061036192505050565b6115eb565b6115eb8882898989898989611677565b8673ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff167f73d170910aba9e6d50b102db522b1dbcd796216f5128b445aa2135272886497e89898888604051611665949392919061252f565b60405180910390a45050505050505050565b611680886112ec565b156117ce5761168f888861134e565b611741576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f5374616e646172644272696467653a2077726f6e672072656d6f746520746f6b60448201527f656e20666f72204f7074696d69736d204d696e7461626c65204552433230206c60648201527f6f63616c20746f6b656e00000000000000000000000000000000000000000000608482015260a401610166565b6040517f9dc29fac00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff878116600483015260248201869052891690639dc29fac90604401600060405180830381600087803b1580156117b157600080fd5b505af11580156117c5573d6000803e3d6000fd5b50505050611862565b6117f073ffffffffffffffffffffffffffffffffffffffff8916873087611ca1565b73ffffffffffffffffffffffffffffffffffffffff8089166000908152600260209081526040808320938b168352929052205461182e9085906125fe565b73ffffffffffffffffffffffffffffffffffffffff808a166000908152600260209081526040808320938c16835292905220555b8573ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff168973ffffffffffffffffffffffffffffffffffffffff167f7ff126db8024424bbfd9826e8ab82ff59136289ea440b04b39a0df1b03b9cabf888887876040516118dc949392919061252f565b60405180910390a47f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16633dbb202b7f0000000000000000000000000000000000000000000000000000000000000000630166a07a60e01b8a8c8b8b8b8a8a6040516024016119669796959493929190612616565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e085901b90921682526119f99291889060040161243e565b600060405180830381600087803b158015611a1357600080fd5b505af1158015611a27573d6000803e3d6000fd5b505050505050505050505050565b606081600003611a7857505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115611aa25780611a8c81612673565b9150611a9b9050600a836126da565b9150611a7c565b60008167ffffffffffffffff811115611abd57611abd6126ee565b6040519080825280601f01601f191660200182016040528015611ae7576020820181803683370190505b5090505b8415611b6a57611afc6001836124cf565b9150611b09600a8661271d565b611b149060306125fe565b60f81b818381518110611b2957611b29612731565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350611b63600a866126da565b9450611aeb565b949350505050565b6000611b7d83611cff565b8015611b8e5750611b8e8383611d63565b9392505050565b6000611bf7826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c65648152508573ffffffffffffffffffffffffffffffffffffffff16611e329092919063ffffffff16565b805190915015610aa65780806020019051810190611c159190612760565b610aa6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f742073756363656564000000000000000000000000000000000000000000006064820152608401610166565b60405173ffffffffffffffffffffffffffffffffffffffff808516602483015283166044820152606481018290526112e69085907f23b872dd0000000000000000000000000000000000000000000000000000000090608401611447565b6000611d2b827f01ffc9a700000000000000000000000000000000000000000000000000000000611d63565b80156113485750611d5c827fffffffff00000000000000000000000000000000000000000000000000000000611d63565b1592915050565b604080517fffffffff000000000000000000000000000000000000000000000000000000008316602480830191909152825180830390910181526044909101909152602080820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01ffc9a700000000000000000000000000000000000000000000000000000000178152825160009392849283928392918391908a617530fa92503d91506000519050828015611e1b575060208210155b8015611e275750600081115b979650505050505050565b6060611b6a84846000858573ffffffffffffffffffffffffffffffffffffffff85163b611ebb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e74726163740000006044820152606401610166565b6000808673ffffffffffffffffffffffffffffffffffffffff168587604051611ee49190612782565b60006040518083038185875af1925050503d8060008114611f21576040519150601f19603f3d011682016040523d82523d6000602084013e611f26565b606091505b5091509150611e2782828660608315611f40575081611b8e565b825115611f505782518084602001fd5b816040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161016691906122aa565b73ffffffffffffffffffffffffffffffffffffffff81168114611fa657600080fd5b50565b60008083601f840112611fbb57600080fd5b50813567ffffffffffffffff811115611fd357600080fd5b602083019150836020828501011115611feb57600080fd5b9250929050565b600080600080600080600060c0888a03121561200d57600080fd5b873561201881611f84565b9650602088013561202881611f84565b9550604088013561203881611f84565b9450606088013561204881611f84565b93506080880135925060a088013567ffffffffffffffff81111561206b57600080fd5b6120778a828b01611fa9565b989b979a50959850939692959293505050565b803563ffffffff8116811461209e57600080fd5b919050565b6000806000604084860312156120b857600080fd5b6120c18461208a565b9250602084013567ffffffffffffffff8111156120dd57600080fd5b6120e986828701611fa9565b9497909650939450505050565b60008060008060006080868803121561210e57600080fd5b853561211981611f84565b9450602086013561212981611f84565b935060408601359250606086013567ffffffffffffffff81111561214c57600080fd5b61215888828901611fa9565b969995985093965092949392505050565b60008060008060006080868803121561218157600080fd5b853561218c81611f84565b9450602086013593506121a16040870161208a565b9250606086013567ffffffffffffffff81111561214c57600080fd5b600080600080600080600060c0888a0312156121d857600080fd5b87356121e381611f84565b965060208801356121f381611f84565b9550604088013561220381611f84565b9450606088013593506122186080890161208a565b925060a088013567ffffffffffffffff81111561206b57600080fd5b60005b8381101561224f578181015183820152602001612237565b838111156112e65750506000910152565b60008151808452612278816020860160208601612234565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000611b8e6020830184612260565b60008060008060008060a087890312156122d657600080fd5b86356122e181611f84565b955060208701356122f181611f84565b9450604087013593506123066060880161208a565b9250608087013567ffffffffffffffff81111561232257600080fd5b61232e89828a01611fa9565b979a9699509497509295939492505050565b6000806040838503121561235357600080fd5b823561235e81611f84565b9150602083013561236e81611f84565b809150509250929050565b6000806000806060858703121561238f57600080fd5b843561239a81611f84565b93506123a86020860161208a565b9250604085013567ffffffffffffffff8111156123c457600080fd5b6123d087828801611fa9565b95989497509550505050565b828152604060208201526000611b6a6040830184612260565b600073ffffffffffffffffffffffffffffffffffffffff8087168352808616602084015250836040830152608060608301526124346080830184612260565b9695505050505050565b73ffffffffffffffffffffffffffffffffffffffff8416815260606020820152600061246d6060830185612260565b905063ffffffff83166040830152949350505050565b60006020828403121561249557600080fd5b8151611b8e81611f84565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000828210156124e1576124e16124a0565b500390565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b73ffffffffffffffffffffffffffffffffffffffff851681528360208201526060604082015260006124346060830184866124e6565b83815260406020820152600061257f6040830184866124e6565b95945050505050565b6000845161259a818460208901612234565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516125d6816001850160208a01612234565b600192019182015283516125f1816002840160208801612234565b0160020195945050505050565b60008219821115612611576126116124a0565b500190565b600073ffffffffffffffffffffffffffffffffffffffff808a1683528089166020840152808816604084015280871660608401525084608083015260c060a083015261266660c0830184866124e6565b9998505050505050505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036126a4576126a46124a0565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b6000826126e9576126e96126ab565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60008261272c5761272c6126ab565b500690565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60006020828403121561277257600080fd5b81518015158114611b8e57600080fd5b60008251612794818460208701612234565b919091019291505056fea164736f6c634300080f000a"

func init() {
	if err := json.Unmarshal([]byte(L2StandardBridgeStorageLayoutJSON), L2StandardBridgeStorageLayout); err != nil {
		panic(err)
	}

	layouts["L2StandardBridge"] = L2StandardBridgeStorageLayout
	deployedBytecodes["L2StandardBridge"] = L2StandardBridgeDeployedBin
}
