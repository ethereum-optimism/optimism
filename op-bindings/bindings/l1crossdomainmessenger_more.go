// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const L1CrossDomainMessengerStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_0_0_20\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_address\"},{\"astId\":1001,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"_initialized\",\"offset\":20,\"slot\":\"0\",\"type\":\"t_uint8\"},{\"astId\":1002,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"_initializing\",\"offset\":21,\"slot\":\"0\",\"type\":\"t_bool\"},{\"astId\":1003,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_1_0_1600\",\"offset\":0,\"slot\":\"1\",\"type\":\"t_array(t_uint256)50_storage\"},{\"astId\":1004,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_51_0_20\",\"offset\":0,\"slot\":\"51\",\"type\":\"t_address\"},{\"astId\":1005,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_52_0_1568\",\"offset\":0,\"slot\":\"52\",\"type\":\"t_array(t_uint256)49_storage\"},{\"astId\":1006,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_101_0_1\",\"offset\":0,\"slot\":\"101\",\"type\":\"t_bool\"},{\"astId\":1007,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_102_0_1568\",\"offset\":0,\"slot\":\"102\",\"type\":\"t_array(t_uint256)49_storage\"},{\"astId\":1008,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_151_0_32\",\"offset\":0,\"slot\":\"151\",\"type\":\"t_uint256\"},{\"astId\":1009,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_152_0_1568\",\"offset\":0,\"slot\":\"152\",\"type\":\"t_array(t_uint256)49_storage\"},{\"astId\":1010,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_201_0_32\",\"offset\":0,\"slot\":\"201\",\"type\":\"t_mapping(t_bytes32,t_bool)\"},{\"astId\":1011,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_202_0_32\",\"offset\":0,\"slot\":\"202\",\"type\":\"t_mapping(t_bytes32,t_bool)\"},{\"astId\":1012,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"successfulMessages\",\"offset\":0,\"slot\":\"203\",\"type\":\"t_mapping(t_bytes32,t_bool)\"},{\"astId\":1013,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"xDomainMsgSender\",\"offset\":0,\"slot\":\"204\",\"type\":\"t_address\"},{\"astId\":1014,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"msgNonce\",\"offset\":0,\"slot\":\"205\",\"type\":\"t_uint240\"},{\"astId\":1015,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"failedMessages\",\"offset\":0,\"slot\":\"206\",\"type\":\"t_mapping(t_bytes32,t_bool)\"},{\"astId\":1016,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"otherMessenger\",\"offset\":0,\"slot\":\"207\",\"type\":\"t_contract(CrossDomainMessenger)1020\"},{\"astId\":1017,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"208\",\"type\":\"t_array(t_uint256)43_storage\"},{\"astId\":1018,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"superchainConfig\",\"offset\":0,\"slot\":\"251\",\"type\":\"t_contract(SuperchainConfig)1022\"},{\"astId\":1019,\"contract\":\"src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"portal\",\"offset\":0,\"slot\":\"252\",\"type\":\"t_contract(OptimismPortal)1021\"}],\"types\":{\"t_address\":{\"encoding\":\"inplace\",\"label\":\"address\",\"numberOfBytes\":\"20\"},\"t_array(t_uint256)43_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[43]\",\"numberOfBytes\":\"1376\",\"base\":\"t_uint256\"},\"t_array(t_uint256)49_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[49]\",\"numberOfBytes\":\"1568\",\"base\":\"t_uint256\"},\"t_array(t_uint256)50_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[50]\",\"numberOfBytes\":\"1600\",\"base\":\"t_uint256\"},\"t_bool\":{\"encoding\":\"inplace\",\"label\":\"bool\",\"numberOfBytes\":\"1\"},\"t_bytes32\":{\"encoding\":\"inplace\",\"label\":\"bytes32\",\"numberOfBytes\":\"32\"},\"t_contract(CrossDomainMessenger)1020\":{\"encoding\":\"inplace\",\"label\":\"contract CrossDomainMessenger\",\"numberOfBytes\":\"20\"},\"t_contract(OptimismPortal)1021\":{\"encoding\":\"inplace\",\"label\":\"contract OptimismPortal\",\"numberOfBytes\":\"20\"},\"t_contract(SuperchainConfig)1022\":{\"encoding\":\"inplace\",\"label\":\"contract SuperchainConfig\",\"numberOfBytes\":\"20\"},\"t_mapping(t_bytes32,t_bool)\":{\"encoding\":\"mapping\",\"label\":\"mapping(bytes32 =\u003e bool)\",\"numberOfBytes\":\"32\",\"key\":\"t_bytes32\",\"value\":\"t_bool\"},\"t_uint240\":{\"encoding\":\"inplace\",\"label\":\"uint240\",\"numberOfBytes\":\"30\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_uint8\":{\"encoding\":\"inplace\",\"label\":\"uint8\",\"numberOfBytes\":\"1\"}}}"

var L1CrossDomainMessengerStorageLayout = new(solc.StorageLayout)

var L1CrossDomainMessengerDeployedBin = "0x6080604052600436106101805760003560e01c80635c975abb116100d6578063a4e7f8bd1161007f578063d764ad0b11610059578063d764ad0b14610463578063db505d8014610476578063ecc70428146104a357600080fd5b8063a4e7f8bd146103e3578063b1b1b20914610413578063b28ade251461044357600080fd5b806383a74074116100b057806383a74074146103a15780638cbeeef2146102b85780639fce812c146103b857600080fd5b80635c975abb1461033a5780636425666b1461035f5780636e296e451461038c57600080fd5b80633dbb202b116101385780634c1d6a69116101125780634c1d6a69146102b857806354fd4d50146102ce5780635644cfdf1461032457600080fd5b80633dbb202b1461025b5780633f827a5a14610270578063485cc9551461029857600080fd5b80630ff754ea116101695780630ff754ea146101cd5780632828d7e81461021957806335e80ab31461022e57600080fd5b8063028f85f7146101855780630c568498146101b8575b600080fd5b34801561019157600080fd5b5061019a601081565b60405167ffffffffffffffff90911681526020015b60405180910390f35b3480156101c457600080fd5b5061019a603f81565b3480156101d957600080fd5b5060fc5473ffffffffffffffffffffffffffffffffffffffff165b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101af565b34801561022557600080fd5b5061019a604081565b34801561023a57600080fd5b5060fb546101f49073ffffffffffffffffffffffffffffffffffffffff1681565b61026e610269366004611a22565b610508565b005b34801561027c57600080fd5b50610285600181565b60405161ffff90911681526020016101af565b3480156102a457600080fd5b5061026e6102b3366004611a89565b610765565b3480156102c457600080fd5b5061019a619c4081565b3480156102da57600080fd5b506103176040518060400160405280600581526020017f322e322e3000000000000000000000000000000000000000000000000000000081525081565b6040516101af9190611b2d565b34801561033057600080fd5b5061019a61138881565b34801561034657600080fd5b5061034f6109d3565b60405190151581526020016101af565b34801561036b57600080fd5b5060fc546101f49073ffffffffffffffffffffffffffffffffffffffff1681565b34801561039857600080fd5b506101f4610a6c565b3480156103ad57600080fd5b5061019a62030d4081565b3480156103c457600080fd5b5060cf5473ffffffffffffffffffffffffffffffffffffffff166101f4565b3480156103ef57600080fd5b5061034f6103fe366004611b47565b60ce6020526000908152604090205460ff1681565b34801561041f57600080fd5b5061034f61042e366004611b47565b60cb6020526000908152604090205460ff1681565b34801561044f57600080fd5b5061019a61045e366004611b60565b610b53565b61026e610471366004611bb4565b610bc1565b34801561048257600080fd5b5060cf546101f49073ffffffffffffffffffffffffffffffffffffffff1681565b3480156104af57600080fd5b506104fa60cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b6040519081526020016101af565b60cf5461063a9073ffffffffffffffffffffffffffffffffffffffff16610530858585610b53565b347fd764ad0b0000000000000000000000000000000000000000000000000000000061059c60cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b338a34898c8c6040516024016105b89796959493929190611c83565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff00000000000000000000000000000000000000000000000000000000909316929092179091526114f2565b8373ffffffffffffffffffffffffffffffffffffffff167fcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a3385856106bf60cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b866040516106d1959493929190611ce2565b60405180910390a260405134815233907f8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d5469060200160405180910390a2505060cd80547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808216600101167fffff0000000000000000000000000000000000000000000000000000000000009091161790555050565b6000547501000000000000000000000000000000000000000000900460ff16158080156107b0575060005460017401000000000000000000000000000000000000000090910460ff16105b806107e25750303b1580156107e2575060005474010000000000000000000000000000000000000000900460ff166001145b610873576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084015b60405180910390fd5b600080547fffffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffff167401000000000000000000000000000000000000000017905580156108f957600080547fffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffffff1675010000000000000000000000000000000000000000001790555b60fb805473ffffffffffffffffffffffffffffffffffffffff8086167fffffffffffffffffffffffff00000000000000000000000000000000000000009283161790925560fc80549285169290911691909117905561096b73420000000000000000000000000000000000000761158b565b80156109ce57600080547fffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffffff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b505050565b60fb54604080517f5c975abb000000000000000000000000000000000000000000000000000000008152905160009273ffffffffffffffffffffffffffffffffffffffff1691635c975abb9160048083019260209291908290030181865afa158015610a43573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610a679190611d30565b905090565b60cc5460009073ffffffffffffffffffffffffffffffffffffffff167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff215301610b36576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603560248201527f43726f7373446f6d61696e4d657373656e6765723a2078446f6d61696e4d657360448201527f7361676553656e646572206973206e6f74207365740000000000000000000000606482015260840161086a565b5060cc5473ffffffffffffffffffffffffffffffffffffffff1690565b6000611388619c4080603f610b6f604063ffffffff8816611d81565b610b799190611db1565b610b84601088611d81565b610b919062030d40611dff565b610b9b9190611dff565b610ba59190611dff565b610baf9190611dff565b610bb99190611dff565b949350505050565b610bc96109d3565b15610c30576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601c60248201527f43726f7373446f6d61696e4d657373656e6765723a2070617573656400000000604482015260640161086a565b60f087901c60028110610ceb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604d60248201527f43726f7373446f6d61696e4d657373656e6765723a206f6e6c7920766572736960448201527f6f6e2030206f722031206d657373616765732061726520737570706f7274656460648201527f20617420746869732074696d6500000000000000000000000000000000000000608482015260a40161086a565b8061ffff16600003610de0576000610d3c878986868080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152508f92506116c7915050565b600081815260cb602052604090205490915060ff1615610dde576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f43726f7373446f6d61696e4d657373656e6765723a206c65676163792077697460448201527f6864726177616c20616c72656164792072656c61796564000000000000000000606482015260840161086a565b505b6000610e26898989898989898080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152506116e692505050565b9050610e30611709565b15610e6857853414610e4457610e44611e2b565b600081815260ce602052604090205460ff1615610e6357610e63611e2b565b610fba565b3415610f1c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605060248201527f43726f7373446f6d61696e4d657373656e6765723a2076616c7565206d75737460448201527f206265207a65726f20756e6c657373206d6573736167652069732066726f6d2060648201527f612073797374656d206164647265737300000000000000000000000000000000608482015260a40161086a565b600081815260ce602052604090205460ff16610fba576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f43726f7373446f6d61696e4d657373656e6765723a206d65737361676520636160448201527f6e6e6f74206265207265706c6179656400000000000000000000000000000000606482015260840161086a565b610fc3876117e5565b15611076576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604360248201527f43726f7373446f6d61696e4d657373656e6765723a2063616e6e6f742073656e60448201527f64206d65737361676520746f20626c6f636b65642073797374656d206164647260648201527f6573730000000000000000000000000000000000000000000000000000000000608482015260a40161086a565b600081815260cb602052604090205460ff1615611115576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f43726f7373446f6d61696e4d657373656e6765723a206d65737361676520686160448201527f7320616c7265616479206265656e2072656c6179656400000000000000000000606482015260840161086a565b61113685611127611388619c40611dff565b67ffffffffffffffff1661182b565b158061115c575060cc5473ffffffffffffffffffffffffffffffffffffffff1661dead14155b1561127557600081815260ce602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555182917f99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f91a27fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff320161126e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602d60248201527f43726f7373446f6d61696e4d657373656e6765723a206661696c656420746f2060448201527f72656c6179206d65737361676500000000000000000000000000000000000000606482015260840161086a565b50506114cd565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff8a16179055600061130688619c405a6112c99190611e5a565b8988888080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061184992505050565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead179055905080156113bc57600082815260cb602052604090205460ff161561135957611359611e2b565b600082815260cb602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183917f4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c91a26114c9565b600082815260ce602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183917f99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f91a27fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff32016114c9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602d60248201527f43726f7373446f6d61696e4d657373656e6765723a206661696c656420746f2060448201527f72656c6179206d65737361676500000000000000000000000000000000000000606482015260840161086a565b5050505b50505050505050565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b60fc546040517fe9e05c4200000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff9091169063e9e05c42908490611553908890839089906000908990600401611e71565b6000604051808303818588803b15801561156c57600080fd5b505af1158015611580573d6000803e3d6000fd5b505050505050505050565b6000547501000000000000000000000000000000000000000000900460ff16611636576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161086a565b60cc5473ffffffffffffffffffffffffffffffffffffffff166116805760cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead1790555b60cf80547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b60006116d585858585611863565b805190602001209050949350505050565b60006116f68787878787876118fc565b8051906020012090509695505050505050565b60fc5460009073ffffffffffffffffffffffffffffffffffffffff1633148015610a67575060cf5460fc54604080517f9bf62d82000000000000000000000000000000000000000000000000000000008152905173ffffffffffffffffffffffffffffffffffffffff9384169390921691639bf62d82916004808201926020929091908290030181865afa1580156117a5573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906117c99190611ec9565b73ffffffffffffffffffffffffffffffffffffffff1614905090565b600073ffffffffffffffffffffffffffffffffffffffff8216301480611825575060fc5473ffffffffffffffffffffffffffffffffffffffff8381169116145b92915050565b600080603f83619c4001026040850201603f5a021015949350505050565b600080600080845160208601878a8af19695505050505050565b60608484848460405160240161187c9493929190611ee6565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fcbd4ece9000000000000000000000000000000000000000000000000000000001790529050949350505050565b606086868686868660405160240161191996959493929190611f30565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fd764ad0b0000000000000000000000000000000000000000000000000000000017905290509695505050505050565b73ffffffffffffffffffffffffffffffffffffffff811681146119bd57600080fd5b50565b60008083601f8401126119d257600080fd5b50813567ffffffffffffffff8111156119ea57600080fd5b602083019150836020828501011115611a0257600080fd5b9250929050565b803563ffffffff81168114611a1d57600080fd5b919050565b60008060008060608587031215611a3857600080fd5b8435611a438161199b565b9350602085013567ffffffffffffffff811115611a5f57600080fd5b611a6b878288016119c0565b9094509250611a7e905060408601611a09565b905092959194509250565b60008060408385031215611a9c57600080fd5b8235611aa78161199b565b91506020830135611ab78161199b565b809150509250929050565b6000815180845260005b81811015611ae857602081850181015186830182015201611acc565b81811115611afa576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000611b406020830184611ac2565b9392505050565b600060208284031215611b5957600080fd5b5035919050565b600080600060408486031215611b7557600080fd5b833567ffffffffffffffff811115611b8c57600080fd5b611b98868287016119c0565b9094509250611bab905060208501611a09565b90509250925092565b600080600080600080600060c0888a031215611bcf57600080fd5b873596506020880135611be18161199b565b95506040880135611bf18161199b565b9450606088013593506080880135925060a088013567ffffffffffffffff811115611c1b57600080fd5b611c278a828b016119c0565b989b979a50959850939692959293505050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b878152600073ffffffffffffffffffffffffffffffffffffffff808916602084015280881660408401525085606083015263ffffffff8516608083015260c060a0830152611cd560c083018486611c3a565b9998505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff86168152608060208201526000611d12608083018688611c3a565b905083604083015263ffffffff831660608301529695505050505050565b600060208284031215611d4257600080fd5b81518015158114611b4057600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600067ffffffffffffffff80831681851681830481118215151615611da857611da8611d52565b02949350505050565b600067ffffffffffffffff80841680611df3577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b92169190910492915050565b600067ffffffffffffffff808316818516808303821115611e2257611e22611d52565b01949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052600160045260246000fd5b600082821015611e6c57611e6c611d52565b500390565b73ffffffffffffffffffffffffffffffffffffffff8616815284602082015267ffffffffffffffff84166040820152821515606082015260a060808201526000611ebe60a0830184611ac2565b979650505050505050565b600060208284031215611edb57600080fd5b8151611b408161199b565b600073ffffffffffffffffffffffffffffffffffffffff808716835280861660208401525060806040830152611f1f6080830185611ac2565b905082606083015295945050505050565b868152600073ffffffffffffffffffffffffffffffffffffffff808816602084015280871660408401525084606083015283608083015260c060a0830152611f7b60c0830184611ac2565b9897505050505050505056fea164736f6c634300080f000a"


func init() {
	if err := json.Unmarshal([]byte(L1CrossDomainMessengerStorageLayoutJSON), L1CrossDomainMessengerStorageLayout); err != nil {
		panic(err)
	}

	layouts["L1CrossDomainMessenger"] = L1CrossDomainMessengerStorageLayout
	deployedBytecodes["L1CrossDomainMessenger"] = L1CrossDomainMessengerDeployedBin
	immutableReferences["L1CrossDomainMessenger"] = false
}
