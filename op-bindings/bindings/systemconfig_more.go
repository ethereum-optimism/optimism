// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const SystemConfigStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"_initialized\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_uint8\"},{\"astId\":1001,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"_initializing\",\"offset\":1,\"slot\":\"0\",\"type\":\"t_bool\"},{\"astId\":1002,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"1\",\"type\":\"t_array(t_uint256)50_storage\"},{\"astId\":1003,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"_owner\",\"offset\":0,\"slot\":\"51\",\"type\":\"t_address\"},{\"astId\":1004,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"52\",\"type\":\"t_array(t_uint256)49_storage\"},{\"astId\":1005,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"overhead\",\"offset\":0,\"slot\":\"101\",\"type\":\"t_uint256\"},{\"astId\":1006,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"scalar\",\"offset\":0,\"slot\":\"102\",\"type\":\"t_uint256\"},{\"astId\":1007,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"batcherHash\",\"offset\":0,\"slot\":\"103\",\"type\":\"t_bytes32\"},{\"astId\":1008,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"gasLimit\",\"offset\":0,\"slot\":\"104\",\"type\":\"t_uint64\"},{\"astId\":1009,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"_resourceConfig\",\"offset\":0,\"slot\":\"105\",\"type\":\"t_struct(ResourceConfig)1011_storage\"},{\"astId\":1010,\"contract\":\"src/L1/SystemConfig.sol:SystemConfig\",\"label\":\"startBlock\",\"offset\":0,\"slot\":\"106\",\"type\":\"t_uint256\"}],\"types\":{\"t_address\":{\"encoding\":\"inplace\",\"label\":\"address\",\"numberOfBytes\":\"20\"},\"t_array(t_uint256)49_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[49]\",\"numberOfBytes\":\"1568\",\"base\":\"t_uint256\"},\"t_array(t_uint256)50_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[50]\",\"numberOfBytes\":\"1600\",\"base\":\"t_uint256\"},\"t_bool\":{\"encoding\":\"inplace\",\"label\":\"bool\",\"numberOfBytes\":\"1\"},\"t_bytes32\":{\"encoding\":\"inplace\",\"label\":\"bytes32\",\"numberOfBytes\":\"32\"},\"t_struct(ResourceConfig)1011_storage\":{\"encoding\":\"inplace\",\"label\":\"struct ResourceMetering.ResourceConfig\",\"numberOfBytes\":\"32\"},\"t_uint128\":{\"encoding\":\"inplace\",\"label\":\"uint128\",\"numberOfBytes\":\"16\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_uint32\":{\"encoding\":\"inplace\",\"label\":\"uint32\",\"numberOfBytes\":\"4\"},\"t_uint64\":{\"encoding\":\"inplace\",\"label\":\"uint64\",\"numberOfBytes\":\"8\"},\"t_uint8\":{\"encoding\":\"inplace\",\"label\":\"uint8\",\"numberOfBytes\":\"1\"}}}"

var SystemConfigStorageLayout = new(solc.StorageLayout)

var SystemConfigDeployedBin = "0x608060405234801561001057600080fd5b50600436106102265760003560e01c8063935f029e1161012a578063cc731b02116100bd578063f45e65d81161008c578063f8c68de011610071578063f8c68de014610575578063fd32aa0f1461057d578063ffa1ad741461058557600080fd5b8063f45e65d814610558578063f68016b71461056157600080fd5b8063cc731b0214610400578063dac6e63a14610534578063e81b2c6d1461053c578063f2fde38b1461054557600080fd5b8063bc49ce5f116100f9578063bc49ce5f146103ca578063c4e8ddfa146103d2578063c71973f6146103da578063c9b26f61146103ed57600080fd5b8063935f029e146103945780639b7d7f0a146103a7578063a7119869146103af578063b40a817c146103b757600080fd5b80634add321d116101bd57806354fd4d501161018c57806361d157681161017157806361d1576814610366578063715018a61461036e5780638da5cb5b1461037657600080fd5b806354fd4d50146103155780635d73369c1461035e57600080fd5b80634add321d146102b25780634d9f1559146102d35780634f16540b146102db5780635228a6ac1461030257600080fd5b806318d13918116101f957806318d139181461028457806319f5cea8146102995780631fd19ee1146102a157806348cd4cb1146102a957600080fd5b806306c926571461022b578063078f29cf146102465780630a49cb03146102735780630c18c1621461027b575b600080fd5b61023361058d565b6040519081526020015b60405180910390f35b61024e6105bb565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200161023d565b61024e6105f4565b61023360655481565b610297610292366004611775565b610624565b005b610233610638565b61024e610663565b610233606a5481565b6102ba61068d565b60405167ffffffffffffffff909116815260200161023d565b61024e6106b3565b6102337f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c0881565b610297610310366004611905565b6106e3565b6103516040518060400160405280600681526020017f312e31322e30000000000000000000000000000000000000000000000000000081525081565b60405161023d9190611aa8565b610233610ae3565b610233610b0e565b610297610b39565b60335473ffffffffffffffffffffffffffffffffffffffff1661024e565b6102976103a2366004611abb565b610b4d565b61024e610b63565b61024e610b93565b6102976103c5366004611add565b610bc3565b610233610bd4565b61024e610bff565b6102976103e8366004611af8565b610c2f565b6102976103fb366004611b14565b610c40565b6104c46040805160c081018252600080825260208201819052918101829052606081018290526080810182905260a0810191909152506040805160c08101825260695463ffffffff8082168352640100000000820460ff9081166020850152650100000000008304169383019390935266010000000000008104831660608301526a0100000000000000000000810490921660808201526e0100000000000000000000000000009091046fffffffffffffffffffffffffffffffff1660a082015290565b60405161023d9190600060c08201905063ffffffff80845116835260ff602085015116602084015260ff6040850151166040840152806060850151166060840152806080850151166080840152506fffffffffffffffffffffffffffffffff60a08401511660a083015292915050565b61024e610c51565b61023360675481565b610297610553366004611775565b610c81565b61023360665481565b6068546102ba9067ffffffffffffffff1681565b610233610d35565b610233610d60565b610233600081565b6105b860017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d611b5c565b81565b60006105ef6105eb60017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad6377611b5c565b5490565b905090565b60006105ef6105eb60017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad611b5c565b61062c610dab565b61063581610e2c565b50565b6105b860017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a8611b5c565b60006105ef7f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c085490565b6069546000906105ef9063ffffffff6a0100000000000000000000820481169116611b73565b60006105ef6105eb60017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a6871816611b5c565b600054610100900460ff16158080156107035750600054600160ff909116105b8061071d5750303b15801561071d575060005460ff166001145b6107ae576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084015b60405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001179055801561080c57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b610814610ee9565b61081d8b610c81565b61082688610f88565b6108308a8a610fb0565b61083987611041565b6108627f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08879055565b61089561089060017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc598611b5c565b849055565b6108c96108c360017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce9580637611b5c565b83519055565b6109006108f760017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a8611b5c565b60208401519055565b61093761092e60017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad6377611b5c565b60408401519055565b61096e61096560017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a6871816611b5c565b60608401519055565b6109a561099c60017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad611b5c565b60808401519055565b6109dc6109d360017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d611b5c565b60a08401519055565b6109e58461111f565b6109ee856111c1565b6109f661068d565b67ffffffffffffffff168767ffffffffffffffff161015610a73576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f770060448201526064016107a5565b8015610ad657600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050505050505050505050565b6105b860017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce9580637611b5c565b6105b860017fe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a6871816611b5c565b610b41610dab565b610b4b6000611635565b565b610b55610dab565b610b5f8282610fb0565b5050565b60006105ef6105eb60017fa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320d611b5c565b60006105ef6105eb60017f383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce9580637611b5c565b610bcb610dab565b61063581611041565b6105b860017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc598611b5c565b60006105ef6105eb60017f46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a8611b5c565b610c37610dab565b610635816111c1565b610c48610dab565b61063581610f88565b60006105ef6105eb60017f71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc598611b5c565b610c89610dab565b73ffffffffffffffffffffffffffffffffffffffff8116610d2c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084016107a5565b61063581611635565b6105b860017f9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad6377611b5c565b6105b860017f4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ad611b5c565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b9055565b60335473ffffffffffffffffffffffffffffffffffffffff163314610b4b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016107a5565b610e557f65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08829055565b6040805173ffffffffffffffffffffffffffffffffffffffff8316602082015260009101604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0818403018152919052905060035b60007f1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be83604051610edd9190611aa8565b60405180910390a35050565b600054610100900460ff16610f80576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016107a5565b610b4b6116ac565b6067819055604080516020808201849052825180830390910181529082019091526000610eac565b606582905560668190556040805160208101849052908101829052600090606001604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190529050600160007f1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be836040516110349190611aa8565b60405180910390a3505050565b61104961068d565b67ffffffffffffffff168167ffffffffffffffff1610156110c6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f770060448201526064016107a5565b606880547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff83169081179091556040805160208082019390935281518082039093018352810190526002610eac565b606a54156111af576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603860248201527f53797374656d436f6e6669673a2063616e6e6f74206f7665727269646520616e60448201527f20616c72656164792073657420737461727420626c6f636b000000000000000060648201526084016107a5565b80156111ba57606a55565b43606a5550565b8060a001516fffffffffffffffffffffffffffffffff16816060015163ffffffff161115611271576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603560248201527f53797374656d436f6e6669673a206d696e206261736520666565206d7573742060448201527f6265206c657373207468616e206d61782062617365000000000000000000000060648201526084016107a5565b6001816040015160ff1611611308576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f53797374656d436f6e6669673a2064656e6f6d696e61746f72206d757374206260448201527f65206c6172676572207468616e2031000000000000000000000000000000000060648201526084016107a5565b6068546080820151825167ffffffffffffffff909216916113299190611b9f565b63ffffffff161115611397576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f53797374656d436f6e6669673a20676173206c696d697420746f6f206c6f770060448201526064016107a5565b6000816020015160ff161161142e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f53797374656d436f6e6669673a20656c6173746963697479206d756c7469706c60448201527f6965722063616e6e6f742062652030000000000000000000000000000000000060648201526084016107a5565b8051602082015163ffffffff82169160ff9091169061144e908290611bbe565b6114589190611c08565b63ffffffff16146114eb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f53797374656d436f6e6669673a20707265636973696f6e206c6f73732077697460448201527f6820746172676574207265736f75726365206c696d697400000000000000000060648201526084016107a5565b805160698054602084015160408501516060860151608087015160a09097015163ffffffff9687167fffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000009095169490941764010000000060ff94851602177fffffffffffffffffffffffffffffffffffffffffffff0000000000ffffffffff166501000000000093909216929092027fffffffffffffffffffffffffffffffffffffffffffff00000000ffffffffffff1617660100000000000091851691909102177fffff0000000000000000000000000000000000000000ffffffffffffffffffff166a010000000000000000000093909416929092027fffff00000000000000000000000000000000ffffffffffffffffffffffffffff16929092176e0100000000000000000000000000006fffffffffffffffffffffffffffffffff90921691909102179055565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600054610100900460ff16611743576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016107a5565b610b4b33611635565b803573ffffffffffffffffffffffffffffffffffffffff8116811461177057600080fd5b919050565b60006020828403121561178757600080fd5b6117908261174c565b9392505050565b803567ffffffffffffffff8116811461177057600080fd5b60405160c0810167ffffffffffffffff811182821017156117f9577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60405290565b803563ffffffff8116811461177057600080fd5b803560ff8116811461177057600080fd5b600060c0828403121561183657600080fd5b60405160c0810181811067ffffffffffffffff82111715611880577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60405290508061188f836117ff565b815261189d60208401611813565b60208201526118ae60408401611813565b60408201526118bf606084016117ff565b60608201526118d0608084016117ff565b608082015260a08301356fffffffffffffffffffffffffffffffff811681146118f857600080fd5b60a0919091015292915050565b6000806000806000806000806000808a8c0361028081121561192657600080fd5b61192f8c61174c565b9a5060208c0135995060408c0135985060608c0135975061195260808d01611797565b965061196060a08d0161174c565b955061196f8d60c08e01611824565b94506101808c013593506119866101a08d0161174c565b925060c07ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe40820112156119b857600080fd5b506119c16117af565b6119ce6101c08d0161174c565b81526119dd6101e08d0161174c565b60208201526119ef6102008d0161174c565b6040820152611a016102208d0161174c565b6060820152611a136102408d0161174c565b6080820152611a256102608d0161174c565b60a0820152809150509295989b9194979a5092959850565b6000815180845260005b81811015611a6357602081850181015186830182015201611a47565b81811115611a75576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006117906020830184611a3d565b60008060408385031215611ace57600080fd5b50508035926020909101359150565b600060208284031215611aef57600080fd5b61179082611797565b600060c08284031215611b0a57600080fd5b6117908383611824565b600060208284031215611b2657600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015611b6e57611b6e611b2d565b500390565b600067ffffffffffffffff808316818516808303821115611b9657611b96611b2d565b01949350505050565b600063ffffffff808316818516808303821115611b9657611b96611b2d565b600063ffffffff80841680611bfc577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b92169190910492915050565b600063ffffffff80831681851681830481118215151615611c2b57611c2b611b2d565b0294935050505056fea164736f6c634300080f000a"


func init() {
	if err := json.Unmarshal([]byte(SystemConfigStorageLayoutJSON), SystemConfigStorageLayout); err != nil {
		panic(err)
	}

	layouts["SystemConfig"] = SystemConfigStorageLayout
	deployedBytecodes["SystemConfig"] = SystemConfigDeployedBin
	immutableReferences["SystemConfig"] = false
}
