// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const L1CrossDomainMessengerStorageLayoutJSON = "{\"storage\":[{\"astId\":24543,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_0_0_20\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_bytes20\"},{\"astId\":26989,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"_initialized\",\"offset\":20,\"slot\":\"0\",\"type\":\"t_uint8\"},{\"astId\":26992,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"_initializing\",\"offset\":21,\"slot\":\"0\",\"type\":\"t_bool\"},{\"astId\":27603,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"1\",\"type\":\"t_array(t_uint256)50_storage\"},{\"astId\":26861,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"_owner\",\"offset\":0,\"slot\":\"51\",\"type\":\"t_address\"},{\"astId\":26981,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"52\",\"type\":\"t_array(t_uint256)49_storage\"},{\"astId\":27154,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"_paused\",\"offset\":0,\"slot\":\"101\",\"type\":\"t_bool\"},{\"astId\":27259,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"102\",\"type\":\"t_array(t_uint256)49_storage\"},{\"astId\":27274,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"_status\",\"offset\":0,\"slot\":\"151\",\"type\":\"t_uint256\"},{\"astId\":27318,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"152\",\"type\":\"t_array(t_uint256)49_storage\"},{\"astId\":24593,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_201_0_32\",\"offset\":0,\"slot\":\"201\",\"type\":\"t_bytes32\"},{\"astId\":24596,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"spacer_202_0_32\",\"offset\":0,\"slot\":\"202\",\"type\":\"t_bytes32\"},{\"astId\":24601,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"successfulMessages\",\"offset\":0,\"slot\":\"203\",\"type\":\"t_mapping(t_bytes32,t_bool)\"},{\"astId\":24604,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"xDomainMsgSender\",\"offset\":0,\"slot\":\"204\",\"type\":\"t_address\"},{\"astId\":24607,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"msgNonce\",\"offset\":0,\"slot\":\"205\",\"type\":\"t_uint240\"},{\"astId\":24612,\"contract\":\"contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger\",\"label\":\"receivedMessages\",\"offset\":0,\"slot\":\"206\",\"type\":\"t_mapping(t_bytes32,t_bool)\"}],\"types\":{\"t_address\":{\"encoding\":\"inplace\",\"label\":\"address\",\"numberOfBytes\":\"20\"},\"t_array(t_uint256)49_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[49]\",\"numberOfBytes\":\"1568\"},\"t_array(t_uint256)50_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[50]\",\"numberOfBytes\":\"1600\"},\"t_bool\":{\"encoding\":\"inplace\",\"label\":\"bool\",\"numberOfBytes\":\"1\"},\"t_bytes20\":{\"encoding\":\"inplace\",\"label\":\"bytes20\",\"numberOfBytes\":\"20\"},\"t_bytes32\":{\"encoding\":\"inplace\",\"label\":\"bytes32\",\"numberOfBytes\":\"32\"},\"t_mapping(t_bytes32,t_bool)\":{\"encoding\":\"mapping\",\"label\":\"mapping(bytes32 =\u003e bool)\",\"numberOfBytes\":\"32\",\"key\":\"t_bytes32\",\"value\":\"t_bool\"},\"t_uint240\":{\"encoding\":\"inplace\",\"label\":\"uint240\",\"numberOfBytes\":\"30\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_uint8\":{\"encoding\":\"inplace\",\"label\":\"uint8\",\"numberOfBytes\":\"1\"}}}"

var L1CrossDomainMessengerStorageLayout = new(solc.StorageLayout)

var L1CrossDomainMessengerDeployedBin = "0x6080604052600436106101755760003560e01c80637dea7cc3116100cb578063b28ade251161007f578063ecc7042811610059578063ecc70428146103f3578063f2fde38b14610458578063f69f81511461047857600080fd5b8063b28ade251461038c578063d764ad0b146103ac578063db505d80146103bf57600080fd5b80638456cb59116100b05780638456cb591461031c5780638da5cb5b14610331578063b1b1b2091461035c57600080fd5b80637dea7cc3146102f05780638129fc1c1461030757600080fd5b80633f827a5a1161012d5780636425666b116101075780636425666b1461026d5780636e296e45146102c6578063715018a6146102db57600080fd5b80633f827a5a146101ff57806354fd4d50146102275780635c975abb1461024957600080fd5b80632828d7e81161015e5780632828d7e8146101bf5780633dbb202b146101d55780633f4ba83a146101ea57600080fd5b8063028f85f71461017a5780630c568498146101a9575b600080fd5b34801561018657600080fd5b5061018f601081565b60405163ffffffff90911681526020015b60405180910390f35b3480156101b557600080fd5b5061018f6103e881565b3480156101cb57600080fd5b5061018f6103f881565b6101e86101e3366004611e64565b6104a8565b005b3480156101f657600080fd5b506101e8610712565b34801561020b57600080fd5b50610214600181565b60405161ffff90911681526020016101a0565b34801561023357600080fd5b5061023c610724565b6040516101a09190611f45565b34801561025557600080fd5b5060655460ff165b60405190151581526020016101a0565b34801561027957600080fd5b506102a17f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101a0565b3480156102d257600080fd5b506102a16107c7565b3480156102e757600080fd5b506101e86108b3565b3480156102fc57600080fd5b5061018f62030d4081565b34801561031357600080fd5b506101e86108c5565b34801561032857600080fd5b506101e8610ac2565b34801561033d57600080fd5b5060335473ffffffffffffffffffffffffffffffffffffffff166102a1565b34801561036857600080fd5b5061025d610377366004611f5f565b60cb6020526000908152604090205460ff1681565b34801561039857600080fd5b5061018f6103a7366004611f78565b610ad2565b6101e86103ba366004611fcc565b610b18565b3480156103cb57600080fd5b506102a17f000000000000000000000000000000000000000000000000000000000000000081565b3480156103ff57600080fd5b5061044a60cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b6040519081526020016101a0565b34801561046457600080fd5b506101e8610473366004612052565b611229565b34801561048457600080fd5b5061025d610493366004611f5f565b60ce6020526000908152604090205460ff1681565b6105e77f00000000000000000000000000000000000000000000000000000000000000006104d7858585610ad2565b63ffffffff16347fd764ad0b0000000000000000000000000000000000000000000000000000000061054960cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b338a34898c8c60405160240161056597969594939291906120b8565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff00000000000000000000000000000000000000000000000000000000909316929092179091526112f9565b8373ffffffffffffffffffffffffffffffffffffffff167fcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a33858561066c60cd547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e010000000000000000000000000000000000000000000000000000000000001790565b8660405161067e959493929190612117565b60405180910390a260405134815233907f8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d5469060200160405180910390a2505060cd80547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808216600101167fffff0000000000000000000000000000000000000000000000000000000000009091161790555050565b61071a6113ae565b61072261142f565b565b606061074f7f00000000000000000000000000000000000000000000000000000000000000006114ac565b6107787f00000000000000000000000000000000000000000000000000000000000000006114ac565b6107a17f00000000000000000000000000000000000000000000000000000000000000006114ac565b6040516020016107b393929190612165565b604051602081830303815290604052905090565b60cc5460009073ffffffffffffffffffffffffffffffffffffffff167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff215301610896576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603560248201527f43726f7373446f6d61696e4d657373656e6765723a2078446f6d61696e4d657360448201527f7361676553656e646572206973206e6f7420736574000000000000000000000060648201526084015b60405180910390fd5b5060cc5473ffffffffffffffffffffffffffffffffffffffff1690565b6108bb6113ae565b61072260006115e1565b6000547501000000000000000000000000000000000000000000900460ff1615808015610910575060005460017401000000000000000000000000000000000000000090910460ff16105b806109425750303b158015610942575060005474010000000000000000000000000000000000000000900460ff166001145b6109ce576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a6564000000000000000000000000000000000000606482015260840161088d565b600080547fffffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffff16740100000000000000000000000000000000000000001790558015610a5457600080547fffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffffff1675010000000000000000000000000000000000000000001790555b610a5c611658565b8015610abf57600080547fffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffffff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b50565b610aca6113ae565b61072261174f565b600062030d40610ae360108561220a565b6103e8610af26103f88661220a565b610afc9190612265565b610b069190612288565b610b109190612288565b949350505050565b600260975403610b84576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f5265656e7472616e637947756172643a207265656e7472616e742063616c6c00604482015260640161088d565b6002609755610b916117aa565b60f087901c60018114610c4c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605560248201527f43726f7373446f6d61696e4d657373656e6765723a206f6e6c7920766572736960448201527f6f6e2031206d657373616765732061726520737570706f72746564206166746560648201527f722074686520426564726f636b20757067726164650000000000000000000000608482015260a40161088d565b6000610c92898989898989898080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061181792505050565b9050610c9c61183a565b15610d3557853414610d30576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f43726f7373446f6d61696e4d657373656e6765723a206d69736d61746368656460448201527f206d6573736167652076616c7565000000000000000000000000000000000000606482015260840161088d565b610e87565b3415610de9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605060248201527f43726f7373446f6d61696e4d657373656e6765723a2076616c7565206d75737460448201527f206265207a65726f20756e6c657373206d6573736167652069732066726f6d2060648201527f612073797374656d206164647265737300000000000000000000000000000000608482015260a40161088d565b600081815260ce602052604090205460ff16610e87576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f43726f7373446f6d61696e4d657373656e6765723a206d65737361676520636160448201527f6e6e6f74206265207265706c6179656400000000000000000000000000000000606482015260840161088d565b610e908761195e565b15610f43576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604360248201527f43726f7373446f6d61696e4d657373656e6765723a2063616e6e6f742073656e60448201527f64206d65737361676520746f20626c6f636b65642073797374656d206164647260648201527f6573730000000000000000000000000000000000000000000000000000000000608482015260a40161088d565b600081815260cb602052604090205460ff1615610fe2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f43726f7373446f6d61696e4d657373656e6765723a206d65737361676520686160448201527f7320616c7265616479206265656e2072656c6179656400000000000000000000606482015260840161088d565b610fee61afc8866122b0565b5a101561107d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f43726f7373446f6d61696e4d657373656e6765723a20696e737566666963696560448201527f6e742067617320746f2072656c6179206d657373616765000000000000000000606482015260840161088d565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff8a161790556000611119886110d161138861afc86122c8565b5a6110dc91906122c8565b8988888080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152506119d592505050565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead17905590508015156001036111b457600082815260cb602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183917f4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c91a2611213565b600082815260ce602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183917f99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f91a25b505060016097555050505050505050565b905090565b6112316113ae565b73ffffffffffffffffffffffffffffffffffffffff81166112d4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015260840161088d565b610abf816115e1565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b6040517fe9e05c4200000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000169063e9e05c429084906113769088908390899060009089906004016122df565b6000604051808303818588803b15801561138f57600080fd5b505af11580156113a3573d6000803e3d6000fd5b505050505050505050565b60335473ffffffffffffffffffffffffffffffffffffffff163314610722576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161088d565b6114376119ef565b606580547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001690557f5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa335b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200160405180910390a1565b6060816000036114ef57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115611519578061150381612337565b91506115129050600a8361236f565b91506114f3565b60008167ffffffffffffffff81111561153457611534612383565b6040519080825280601f01601f19166020018201604052801561155e576020820181803683370190505b5090505b8415610b10576115736001836122c8565b9150611580600a866123b2565b61158b9060306122b0565b60f81b8183815181106115a0576115a06123c6565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053506115da600a8661236f565b9450611562565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b6000547501000000000000000000000000000000000000000000900460ff16611703576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161088d565b60cc80547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead179055611737611a5b565b61173f611b06565b611747611bba565b610722611c8f565b6117576117aa565b606580547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790557f62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a2586114823390565b60655460ff1615610722576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601060248201527f5061757361626c653a2070617573656400000000000000000000000000000000604482015260640161088d565b6000611827878787878787611d41565b8051906020012090509695505050505050565b60003373ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614801561122457507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16639bf62d826040518163ffffffff1660e01b8152600401602060405180830381865afa15801561191e573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061194291906123f5565b73ffffffffffffffffffffffffffffffffffffffff1614905090565b600073ffffffffffffffffffffffffffffffffffffffff82163014806119cf57507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16145b92915050565b600080600080845160208601878a8af19695505050505050565b60655460ff16610722576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f5061757361626c653a206e6f7420706175736564000000000000000000000000604482015260640161088d565b6000547501000000000000000000000000000000000000000000900460ff16610722576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161088d565b6000547501000000000000000000000000000000000000000000900460ff16611bb1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161088d565b610722336115e1565b6000547501000000000000000000000000000000000000000000900460ff16611c65576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161088d565b606580547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00169055565b6000547501000000000000000000000000000000000000000000900460ff16611d3a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161088d565b6001609755565b6060868686868686604051602401611d5e96959493929190612412565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fd764ad0b0000000000000000000000000000000000000000000000000000000017905290509695505050505050565b73ffffffffffffffffffffffffffffffffffffffff81168114610abf57600080fd5b60008083601f840112611e1457600080fd5b50813567ffffffffffffffff811115611e2c57600080fd5b602083019150836020828501011115611e4457600080fd5b9250929050565b803563ffffffff81168114611e5f57600080fd5b919050565b60008060008060608587031215611e7a57600080fd5b8435611e8581611de0565b9350602085013567ffffffffffffffff811115611ea157600080fd5b611ead87828801611e02565b9094509250611ec0905060408601611e4b565b905092959194509250565b60005b83811015611ee6578181015183820152602001611ece565b83811115611ef5576000848401525b50505050565b60008151808452611f13816020860160208601611ecb565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000611f586020830184611efb565b9392505050565b600060208284031215611f7157600080fd5b5035919050565b600080600060408486031215611f8d57600080fd5b833567ffffffffffffffff811115611fa457600080fd5b611fb086828701611e02565b9094509250611fc3905060208501611e4b565b90509250925092565b600080600080600080600060c0888a031215611fe757600080fd5b873596506020880135611ff981611de0565b9550604088013561200981611de0565b9450606088013593506080880135925060a088013567ffffffffffffffff81111561203357600080fd5b61203f8a828b01611e02565b989b979a50959850939692959293505050565b60006020828403121561206457600080fd5b8135611f5881611de0565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b878152600073ffffffffffffffffffffffffffffffffffffffff808916602084015280881660408401525085606083015263ffffffff8516608083015260c060a083015261210a60c08301848661206f565b9998505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff8616815260806020820152600061214760808301868861206f565b905083604083015263ffffffff831660608301529695505050505050565b60008451612177818460208901611ecb565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516121b3816001850160208a01611ecb565b600192019182015283516121ce816002840160208801611ecb565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600063ffffffff8083168185168183048111821515161561222d5761222d6121db565b02949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600063ffffffff8084168061227c5761227c612236565b92169190910492915050565b600063ffffffff8083168185168083038211156122a7576122a76121db565b01949350505050565b600082198211156122c3576122c36121db565b500190565b6000828210156122da576122da6121db565b500390565b73ffffffffffffffffffffffffffffffffffffffff8616815284602082015267ffffffffffffffff84166040820152821515606082015260a06080820152600061232c60a0830184611efb565b979650505050505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203612368576123686121db565b5060010190565b60008261237e5761237e612236565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000826123c1576123c1612236565b500690565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60006020828403121561240757600080fd5b8151611f5881611de0565b868152600073ffffffffffffffffffffffffffffffffffffffff808816602084015280871660408401525084606083015283608083015260c060a083015261245d60c0830184611efb565b9897505050505050505056fea164736f6c634300080f000a"

func init() {
	if err := json.Unmarshal([]byte(L1CrossDomainMessengerStorageLayoutJSON), L1CrossDomainMessengerStorageLayout); err != nil {
		panic(err)
	}

	layouts["L1CrossDomainMessenger"] = L1CrossDomainMessengerStorageLayout
	deployedBytecodes["L1CrossDomainMessenger"] = L1CrossDomainMessengerDeployedBin
}
