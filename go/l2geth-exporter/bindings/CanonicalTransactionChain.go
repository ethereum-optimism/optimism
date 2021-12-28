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

// Lib_OVMCodecQueueElement is an auto generated low-level Go binding around an user-defined struct.
type Lib_OVMCodecQueueElement struct {
	TransactionHash [32]byte
	Timestamp       *big.Int
	BlockNumber     *big.Int
}

// CanonicalTransactionChainMetaData contains all meta data concerning the CanonicalTransactionChain contract.
var CanonicalTransactionChainMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_libAddressManager\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_maxTransactionGasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_l2GasDiscountDivisor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_enqueueGasCost\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"l2GasDiscountDivisor\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"enqueueGasCost\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"enqueueL2GasPrepaid\",\"type\":\"uint256\"}],\"name\":\"L2GasParamsUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_startingQueueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_numQueueElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"name\":\"QueueBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_startingQueueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_numQueueElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"name\":\"SequencerBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_batchSize\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_prevTotalElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"TransactionBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_l1TxOrigin\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_queueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_timestamp\",\"type\":\"uint256\"}],\"name\":\"TransactionEnqueued\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MAX_ROLLUP_TX_SIZE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_ROLLUP_TX_GAS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"appendSequencerBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batches\",\"outputs\":[{\"internalType\":\"contractIChainStorageContainer\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"enqueue\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"enqueueGasCost\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"enqueueL2GasPrepaid\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastBlockNumber\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastTimestamp\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNextQueueIndex\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNumPendingQueueElements\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"getQueueElement\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"transactionHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint40\",\"name\":\"timestamp\",\"type\":\"uint40\"},{\"internalType\":\"uint40\",\"name\":\"blockNumber\",\"type\":\"uint40\"}],\"internalType\":\"structLib_OVMCodec.QueueElement\",\"name\":\"_element\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getQueueLength\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalBatches\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalBatches\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalElements\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2GasDiscountDivisor\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"libAddressManager\",\"outputs\":[{\"internalType\":\"contractLib_AddressManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxTransactionGasLimit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"resolve\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2GasDiscountDivisor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_enqueueGasCost\",\"type\":\"uint256\"}],\"name\":\"setGasParams\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5060405162001a9838038062001a9883398101604081905261003191610072565b600080546001600160a01b0319166001600160a01b03861617905560048390556002829055600181905561006581836100bd565b600355506100ea92505050565b6000806000806080858703121561008857600080fd5b84516001600160a01b038116811461009f57600080fd5b60208601516040870151606090970151919890975090945092505050565b60008160001904831182151516156100e557634e487b7160e01b600052601160045260246000fd5b500290565b61199e80620000fa6000396000f3fe608060405234801561001057600080fd5b506004361061016c5760003560e01c8063876ed5cb116100cd578063d0f8934411610081578063e654b1fb11610066578063e654b1fb146102c0578063edcc4a45146102c9578063f722b41a146102dc57600080fd5b8063d0f89344146102b0578063e561dddc146102b857600080fd5b8063b8f77005116100b2578063b8f7700514610297578063ccf987c81461029f578063cfdf677e146102a857600080fd5b8063876ed5cb146102855780638d38c6c11461028e57600080fd5b80635ae6256d1161012457806378f4b2f21161010957806378f4b2f2146102645780637a167a8a1461026e5780637aa63a861461027d57600080fd5b80635ae6256d146102475780636fee07e01461024f57600080fd5b80632a7f18be116101555780632a7f18be146101d25780633789977014610216578063461a44781461023457600080fd5b80630b3dfa9714610171578063299ca4781461018d575b600080fd5b61017a60035481565b6040519081526020015b60405180910390f35b6000546101ad9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610184565b6101e56101e03660046113e5565b6102e4565b604080518251815260208084015164ffffffffff908116918301919091529282015190921690820152606001610184565b61021e610362565b60405164ffffffffff9091168152602001610184565b6101ad6102423660046114c1565b610376565b61021e610423565b61026261025d366004611537565b610437565b005b61017a620186a081565b60055464ffffffffff1661021e565b61017a610899565b61017a61c35081565b61017a60045481565b60065461021e565b61017a60025481565b6101ad6108b4565b6102626108dc565b61017a610df8565b61017a60015481565b6102626102d73660046115a4565b610e7f565b61021e611016565b604080516060810182526000808252602082018190529181019190915260068281548110610314576103146115c6565b6000918252602091829020604080516060810182526002909302909101805483526001015464ffffffffff808216948401949094526501000000000090049092169181019190915292915050565b60008061036d611032565b50949350505050565b600080546040517fbf40fac100000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff9091169063bf40fac1906103cd908590600401611660565b60206040518083038186803b1580156103e557600080fd5b505afa1580156103f9573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061041d919061167a565b92915050565b60008061042e611032565b95945050505050565b61c350815111156104cf576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603d60248201527f5472616e73616374696f6e20646174612073697a652065786365656473206d6160448201527f78696d756d20666f7220726f6c6c7570207472616e73616374696f6e2e00000060648201526084015b60405180910390fd5b600454821115610561576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603d60248201527f5472616e73616374696f6e20676173206c696d69742065786365656473206d6160448201527f78696d756d20666f7220726f6c6c7570207472616e73616374696f6e2e00000060648201526084016104c6565b620186a08210156105f4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602960248201527f5472616e73616374696f6e20676173206c696d697420746f6f206c6f7720746f60448201527f20656e71756575652e000000000000000000000000000000000000000000000060648201526084016104c6565b6003548211156106dc5760006002546003548461061191906116c6565b61061b91906116dd565b905060005a90508181116106b1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e73756666696369656e742067617320666f72204c322072617465206c696d60448201527f6974696e67206275726e2e00000000000000000000000000000000000000000060648201526084016104c6565b60005b825a6106c090846116c6565b10156106d857806106d081611718565b9150506106b4565b5050505b6000333214156106ed575033610706565b5033731111000000000000000000000000000000001111015b60008185858560405160200161071f9493929190611751565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0818403018152828252805160209182012060608401835280845264ffffffffff42811692850192835243811693850193845260068054600181810183556000838152975160029092027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f81019290925594517ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d4090910180549651841665010000000000027fffffffffffffffffffffffffffffffffffffffffffff0000000000000000000090971691909316179490941790559154919350610825916116c6565b9050808673ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167f4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb58888426040516108899392919061179a565b60405180910390a4505050505050565b6000806108a4611032565b50505064ffffffffff1692915050565b60006108d760405180606001604052806021815260200161194860219139610376565b905090565b60043560d81c60093560e890811c90600c35901c6108f8610899565b8364ffffffffff161461098d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603d60248201527f41637475616c20626174636820737461727420696e64657820646f6573206e6f60448201527f74206d6174636820657870656374656420737461727420696e6465782e00000060648201526084016104c6565b6109cb6040518060400160405280600d81526020017f4f564d5f53657175656e63657200000000000000000000000000000000000000815250610376565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610a85576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602d60248201527f46756e6374696f6e2063616e206f6e6c792062652063616c6c6564206279207460448201527f68652053657175656e6365722e0000000000000000000000000000000000000060648201526084016104c6565b6000610a9762ffffff831660106117c3565b610aa290600f611800565b905064ffffffffff8116361015610b3b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f4e6f7420656e6f756768204261746368436f6e74657874732070726f7669646560448201527f642e00000000000000000000000000000000000000000000000000000000000060648201526084016104c6565b6005546040805160808101825260008082526020820181905291810182905260608101829052909164ffffffffff169060005b8562ffffff168163ffffffff161015610bcc576000610b928263ffffffff166110ed565b8051909350839150610ba49086611818565b9450826020015184610bb69190611840565b9350508080610bc490611860565b915050610b6e565b5060065464ffffffffff83161115610c8c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604260248201527f417474656d7074656420746f20617070656e64206d6f726520656c656d656e7460448201527f73207468616e2061726520617661696c61626c6520696e20746865207175657560648201527f652e000000000000000000000000000000000000000000000000000000000000608482015260a4016104c6565b6000610c9d8462ffffff8916611884565b63ffffffff169050600080836020015160001415610cc657505060408201516060830151610d37565b60006006610cd56001886118a9565b64ffffffffff1681548110610cec57610cec6115c6565b6000918252602091829020604080516060810182526002909302909101805483526001015464ffffffffff808216948401859052650100000000009091041691018190529093509150505b610d5b610d456001436116c6565b408a62ffffff168564ffffffffff168585611174565b7f602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899610d8684876118a9565b84610d8f610899565b6040805164ffffffffff94851681529390921660208401529082015260600160405180910390a15050600580547fffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000001664ffffffffff949094169390931790925550505050505050565b6000610e026108b4565b73ffffffffffffffffffffffffffffffffffffffff16631f7b6d326040518163ffffffff1660e01b815260040160206040518083038186803b158015610e4757600080fd5b505afa158015610e5b573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906108d791906118c7565b60008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b815260040160206040518083038186803b158015610ee557600080fd5b505afa158015610ef9573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610f1d919061167a565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610fb1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f6e6c792063616c6c61626c6520627920746865204275726e2041646d696e2e60448201526064016104c6565b60018190556002829055610fc581836117c3565b60038190556002546001546040805192835260208301919091528101919091527fc6ed75e96b8b18b71edc1a6e82a9d677f8268c774a262c624eeb2cf0a8b3e07e9060600160405180910390a15050565b6005546006546000916108d79164ffffffffff909116906118a9565b60008060008060006110426108b4565b73ffffffffffffffffffffffffffffffffffffffff1663ccf8f9696040518163ffffffff1660e01b815260040160206040518083038186803b15801561108757600080fd5b505afa15801561109b573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906110bf91906118e0565b64ffffffffff602882901c811697605083901c82169750607883901c8216965060a09290921c169350915050565b6111186040518060800160405280600081526020016000815260200160008152602001600081525090565b60006111256010846117c3565b61113090600f611800565b60408051608081018252823560e890811c82526003840135901c6020820152600683013560d890811c92820192909252600b90920135901c60608201529392505050565b600061117e6108b4565b905060008061118b611032565b50509150915060006040518060a001604052808573ffffffffffffffffffffffffffffffffffffffff16631f7b6d326040518163ffffffff1660e01b815260040160206040518083038186803b1580156111e457600080fd5b505afa1580156111f8573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061121c91906118c7565b81526020018a81526020018981526020018464ffffffffff16815260200160405180602001604052806000815250815250905080600001517f127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e82602001518360400151846060015185608001516040516112999493929190611922565b60405180910390a260006112ac8261139f565b905060006112e78360400151866112c39190611840565b6112cd8b87611840565b602890811b9190911760508b901b1760788a901b17901b90565b6040517f2015276c000000000000000000000000000000000000000000000000000000008152600481018490527fffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000008216602482015290915073ffffffffffffffffffffffffffffffffffffffff871690632015276c90604401600060405180830381600087803b15801561137a57600080fd5b505af115801561138e573d6000803e3d6000fd5b505050505050505050505050505050565b600081602001518260400151836060015184608001516040516020016113c89493929190611922565b604051602081830303815290604052805190602001209050919050565b6000602082840312156113f757600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600067ffffffffffffffff80841115611448576114486113fe565b604051601f85017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f0116810190828211818310171561148e5761148e6113fe565b816040528093508581528686860111156114a757600080fd5b858560208301376000602087830101525050509392505050565b6000602082840312156114d357600080fd5b813567ffffffffffffffff8111156114ea57600080fd5b8201601f810184136114fb57600080fd5b61150a8482356020840161142d565b949350505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461153457600080fd5b50565b60008060006060848603121561154c57600080fd5b833561155781611512565b925060208401359150604084013567ffffffffffffffff81111561157a57600080fd5b8401601f8101861361158b57600080fd5b61159a8682356020840161142d565b9150509250925092565b600080604083850312156115b757600080fd5b50508035926020909101359150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b6000815180845260005b8181101561161b576020818501810151868301820152016115ff565b8181111561162d576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061167360208301846115f5565b9392505050565b60006020828403121561168c57600080fd5b815161167381611512565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000828210156116d8576116d8611697565b500390565b600082611713577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b500490565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82141561174a5761174a611697565b5060010190565b600073ffffffffffffffffffffffffffffffffffffffff80871683528086166020840152508360408301526080606083015261179060808301846115f5565b9695505050505050565b8381526060602082015260006117b360608301856115f5565b9050826040830152949350505050565b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff04831182151516156117fb576117fb611697565b500290565b6000821982111561181357611813611697565b500190565b600063ffffffff80831681851680830382111561183757611837611697565b01949350505050565b600064ffffffffff80831681851680830382111561183757611837611697565b600063ffffffff8083168181141561187a5761187a611697565b6001019392505050565b600063ffffffff838116908316818110156118a1576118a1611697565b039392505050565b600064ffffffffff838116908316818110156118a1576118a1611697565b6000602082840312156118d957600080fd5b5051919050565b6000602082840312156118f257600080fd5b81517fffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000008116811461167357600080fd5b84815283602082015282604082015260806060820152600061179060808301846115f556fe436861696e53746f72616765436f6e7461696e65722d4354432d62617463686573a2646970667358221220e14033f9f98984edb3353943a45655d112afab7b0a7aa8401f8826506d85b00164736f6c63430008090033",
}

// CanonicalTransactionChainABI is the input ABI used to generate the binding from.
// Deprecated: Use CanonicalTransactionChainMetaData.ABI instead.
var CanonicalTransactionChainABI = CanonicalTransactionChainMetaData.ABI

// CanonicalTransactionChainBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CanonicalTransactionChainMetaData.Bin instead.
var CanonicalTransactionChainBin = CanonicalTransactionChainMetaData.Bin

// DeployCanonicalTransactionChain deploys a new Ethereum contract, binding an instance of CanonicalTransactionChain to it.
func DeployCanonicalTransactionChain(auth *bind.TransactOpts, backend bind.ContractBackend, _libAddressManager common.Address, _maxTransactionGasLimit *big.Int, _l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (common.Address, *types.Transaction, *CanonicalTransactionChain, error) {
	parsed, err := CanonicalTransactionChainMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CanonicalTransactionChainBin), backend, _libAddressManager, _maxTransactionGasLimit, _l2GasDiscountDivisor, _enqueueGasCost)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &CanonicalTransactionChain{CanonicalTransactionChainCaller: CanonicalTransactionChainCaller{contract: contract}, CanonicalTransactionChainTransactor: CanonicalTransactionChainTransactor{contract: contract}, CanonicalTransactionChainFilterer: CanonicalTransactionChainFilterer{contract: contract}}, nil
}

// CanonicalTransactionChain is an auto generated Go binding around an Ethereum contract.
type CanonicalTransactionChain struct {
	CanonicalTransactionChainCaller     // Read-only binding to the contract
	CanonicalTransactionChainTransactor // Write-only binding to the contract
	CanonicalTransactionChainFilterer   // Log filterer for contract events
}

// CanonicalTransactionChainCaller is an auto generated read-only Go binding around an Ethereum contract.
type CanonicalTransactionChainCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CanonicalTransactionChainTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CanonicalTransactionChainTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CanonicalTransactionChainFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CanonicalTransactionChainFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CanonicalTransactionChainSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CanonicalTransactionChainSession struct {
	Contract     *CanonicalTransactionChain // Generic contract binding to set the session for
	CallOpts     bind.CallOpts              // Call options to use throughout this session
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// CanonicalTransactionChainCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CanonicalTransactionChainCallerSession struct {
	Contract *CanonicalTransactionChainCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                    // Call options to use throughout this session
}

// CanonicalTransactionChainTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CanonicalTransactionChainTransactorSession struct {
	Contract     *CanonicalTransactionChainTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                    // Transaction auth options to use throughout this session
}

// CanonicalTransactionChainRaw is an auto generated low-level Go binding around an Ethereum contract.
type CanonicalTransactionChainRaw struct {
	Contract *CanonicalTransactionChain // Generic contract binding to access the raw methods on
}

// CanonicalTransactionChainCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CanonicalTransactionChainCallerRaw struct {
	Contract *CanonicalTransactionChainCaller // Generic read-only contract binding to access the raw methods on
}

// CanonicalTransactionChainTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CanonicalTransactionChainTransactorRaw struct {
	Contract *CanonicalTransactionChainTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCanonicalTransactionChain creates a new instance of CanonicalTransactionChain, bound to a specific deployed contract.
func NewCanonicalTransactionChain(address common.Address, backend bind.ContractBackend) (*CanonicalTransactionChain, error) {
	contract, err := bindCanonicalTransactionChain(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChain{CanonicalTransactionChainCaller: CanonicalTransactionChainCaller{contract: contract}, CanonicalTransactionChainTransactor: CanonicalTransactionChainTransactor{contract: contract}, CanonicalTransactionChainFilterer: CanonicalTransactionChainFilterer{contract: contract}}, nil
}

// NewCanonicalTransactionChainCaller creates a new read-only instance of CanonicalTransactionChain, bound to a specific deployed contract.
func NewCanonicalTransactionChainCaller(address common.Address, caller bind.ContractCaller) (*CanonicalTransactionChainCaller, error) {
	contract, err := bindCanonicalTransactionChain(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainCaller{contract: contract}, nil
}

// NewCanonicalTransactionChainTransactor creates a new write-only instance of CanonicalTransactionChain, bound to a specific deployed contract.
func NewCanonicalTransactionChainTransactor(address common.Address, transactor bind.ContractTransactor) (*CanonicalTransactionChainTransactor, error) {
	contract, err := bindCanonicalTransactionChain(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainTransactor{contract: contract}, nil
}

// NewCanonicalTransactionChainFilterer creates a new log filterer instance of CanonicalTransactionChain, bound to a specific deployed contract.
func NewCanonicalTransactionChainFilterer(address common.Address, filterer bind.ContractFilterer) (*CanonicalTransactionChainFilterer, error) {
	contract, err := bindCanonicalTransactionChain(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainFilterer{contract: contract}, nil
}

// bindCanonicalTransactionChain binds a generic wrapper to an already deployed contract.
func bindCanonicalTransactionChain(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CanonicalTransactionChainABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CanonicalTransactionChain *CanonicalTransactionChainRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CanonicalTransactionChain.Contract.CanonicalTransactionChainCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CanonicalTransactionChain *CanonicalTransactionChainRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.CanonicalTransactionChainTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CanonicalTransactionChain *CanonicalTransactionChainRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.CanonicalTransactionChainTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CanonicalTransactionChain.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.contract.Transact(opts, method, params...)
}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) MAXROLLUPTXSIZE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "MAX_ROLLUP_TX_SIZE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) MAXROLLUPTXSIZE() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MAXROLLUPTXSIZE(&_CanonicalTransactionChain.CallOpts)
}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) MAXROLLUPTXSIZE() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MAXROLLUPTXSIZE(&_CanonicalTransactionChain.CallOpts)
}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) MINROLLUPTXGAS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "MIN_ROLLUP_TX_GAS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) MINROLLUPTXGAS() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MINROLLUPTXGAS(&_CanonicalTransactionChain.CallOpts)
}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) MINROLLUPTXGAS() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MINROLLUPTXGAS(&_CanonicalTransactionChain.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) Batches(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "batches")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) Batches() (common.Address, error) {
	return _CanonicalTransactionChain.Contract.Batches(&_CanonicalTransactionChain.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) Batches() (common.Address, error) {
	return _CanonicalTransactionChain.Contract.Batches(&_CanonicalTransactionChain.CallOpts)
}

// EnqueueGasCost is a free data retrieval call binding the contract method 0xe654b1fb.
//
// Solidity: function enqueueGasCost() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) EnqueueGasCost(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "enqueueGasCost")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EnqueueGasCost is a free data retrieval call binding the contract method 0xe654b1fb.
//
// Solidity: function enqueueGasCost() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) EnqueueGasCost() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.EnqueueGasCost(&_CanonicalTransactionChain.CallOpts)
}

// EnqueueGasCost is a free data retrieval call binding the contract method 0xe654b1fb.
//
// Solidity: function enqueueGasCost() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) EnqueueGasCost() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.EnqueueGasCost(&_CanonicalTransactionChain.CallOpts)
}

// EnqueueL2GasPrepaid is a free data retrieval call binding the contract method 0x0b3dfa97.
//
// Solidity: function enqueueL2GasPrepaid() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) EnqueueL2GasPrepaid(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "enqueueL2GasPrepaid")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EnqueueL2GasPrepaid is a free data retrieval call binding the contract method 0x0b3dfa97.
//
// Solidity: function enqueueL2GasPrepaid() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) EnqueueL2GasPrepaid() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.EnqueueL2GasPrepaid(&_CanonicalTransactionChain.CallOpts)
}

// EnqueueL2GasPrepaid is a free data retrieval call binding the contract method 0x0b3dfa97.
//
// Solidity: function enqueueL2GasPrepaid() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) EnqueueL2GasPrepaid() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.EnqueueL2GasPrepaid(&_CanonicalTransactionChain.CallOpts)
}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetLastBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getLastBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetLastBlockNumber() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetLastBlockNumber(&_CanonicalTransactionChain.CallOpts)
}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetLastBlockNumber() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetLastBlockNumber(&_CanonicalTransactionChain.CallOpts)
}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetLastTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getLastTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetLastTimestamp() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetLastTimestamp(&_CanonicalTransactionChain.CallOpts)
}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetLastTimestamp() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetLastTimestamp(&_CanonicalTransactionChain.CallOpts)
}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetNextQueueIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getNextQueueIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetNextQueueIndex() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetNextQueueIndex(&_CanonicalTransactionChain.CallOpts)
}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetNextQueueIndex() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetNextQueueIndex(&_CanonicalTransactionChain.CallOpts)
}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetNumPendingQueueElements(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getNumPendingQueueElements")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetNumPendingQueueElements() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetNumPendingQueueElements(&_CanonicalTransactionChain.CallOpts)
}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetNumPendingQueueElements() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetNumPendingQueueElements(&_CanonicalTransactionChain.CallOpts)
}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetQueueElement(opts *bind.CallOpts, _index *big.Int) (Lib_OVMCodecQueueElement, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getQueueElement", _index)

	if err != nil {
		return *new(Lib_OVMCodecQueueElement), err
	}

	out0 := *abi.ConvertType(out[0], new(Lib_OVMCodecQueueElement)).(*Lib_OVMCodecQueueElement)

	return out0, err

}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetQueueElement(_index *big.Int) (Lib_OVMCodecQueueElement, error) {
	return _CanonicalTransactionChain.Contract.GetQueueElement(&_CanonicalTransactionChain.CallOpts, _index)
}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetQueueElement(_index *big.Int) (Lib_OVMCodecQueueElement, error) {
	return _CanonicalTransactionChain.Contract.GetQueueElement(&_CanonicalTransactionChain.CallOpts, _index)
}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetQueueLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getQueueLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetQueueLength() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetQueueLength(&_CanonicalTransactionChain.CallOpts)
}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetQueueLength() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetQueueLength(&_CanonicalTransactionChain.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetTotalBatches(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getTotalBatches")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetTotalBatches() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetTotalBatches(&_CanonicalTransactionChain.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetTotalBatches() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetTotalBatches(&_CanonicalTransactionChain.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetTotalElements(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getTotalElements")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetTotalElements() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetTotalElements(&_CanonicalTransactionChain.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetTotalElements() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetTotalElements(&_CanonicalTransactionChain.CallOpts)
}

// L2GasDiscountDivisor is a free data retrieval call binding the contract method 0xccf987c8.
//
// Solidity: function l2GasDiscountDivisor() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) L2GasDiscountDivisor(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "l2GasDiscountDivisor")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2GasDiscountDivisor is a free data retrieval call binding the contract method 0xccf987c8.
//
// Solidity: function l2GasDiscountDivisor() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) L2GasDiscountDivisor() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.L2GasDiscountDivisor(&_CanonicalTransactionChain.CallOpts)
}

// L2GasDiscountDivisor is a free data retrieval call binding the contract method 0xccf987c8.
//
// Solidity: function l2GasDiscountDivisor() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) L2GasDiscountDivisor() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.L2GasDiscountDivisor(&_CanonicalTransactionChain.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) LibAddressManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "libAddressManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) LibAddressManager() (common.Address, error) {
	return _CanonicalTransactionChain.Contract.LibAddressManager(&_CanonicalTransactionChain.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) LibAddressManager() (common.Address, error) {
	return _CanonicalTransactionChain.Contract.LibAddressManager(&_CanonicalTransactionChain.CallOpts)
}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) MaxTransactionGasLimit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "maxTransactionGasLimit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) MaxTransactionGasLimit() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MaxTransactionGasLimit(&_CanonicalTransactionChain.CallOpts)
}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) MaxTransactionGasLimit() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MaxTransactionGasLimit(&_CanonicalTransactionChain.CallOpts)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) Resolve(opts *bind.CallOpts, _name string) (common.Address, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "resolve", _name)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) Resolve(_name string) (common.Address, error) {
	return _CanonicalTransactionChain.Contract.Resolve(&_CanonicalTransactionChain.CallOpts, _name)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) Resolve(_name string) (common.Address, error) {
	return _CanonicalTransactionChain.Contract.Resolve(&_CanonicalTransactionChain.CallOpts, _name)
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactor) AppendSequencerBatch(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CanonicalTransactionChain.contract.Transact(opts, "appendSequencerBatch")
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) AppendSequencerBatch() (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.AppendSequencerBatch(&_CanonicalTransactionChain.TransactOpts)
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactorSession) AppendSequencerBatch() (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.AppendSequencerBatch(&_CanonicalTransactionChain.TransactOpts)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactor) Enqueue(opts *bind.TransactOpts, _target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _CanonicalTransactionChain.contract.Transact(opts, "enqueue", _target, _gasLimit, _data)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) Enqueue(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.Enqueue(&_CanonicalTransactionChain.TransactOpts, _target, _gasLimit, _data)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactorSession) Enqueue(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.Enqueue(&_CanonicalTransactionChain.TransactOpts, _target, _gasLimit, _data)
}

// SetGasParams is a paid mutator transaction binding the contract method 0xedcc4a45.
//
// Solidity: function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactor) SetGasParams(opts *bind.TransactOpts, _l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (*types.Transaction, error) {
	return _CanonicalTransactionChain.contract.Transact(opts, "setGasParams", _l2GasDiscountDivisor, _enqueueGasCost)
}

// SetGasParams is a paid mutator transaction binding the contract method 0xedcc4a45.
//
// Solidity: function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) SetGasParams(_l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.SetGasParams(&_CanonicalTransactionChain.TransactOpts, _l2GasDiscountDivisor, _enqueueGasCost)
}

// SetGasParams is a paid mutator transaction binding the contract method 0xedcc4a45.
//
// Solidity: function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactorSession) SetGasParams(_l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.SetGasParams(&_CanonicalTransactionChain.TransactOpts, _l2GasDiscountDivisor, _enqueueGasCost)
}

// CanonicalTransactionChainL2GasParamsUpdatedIterator is returned from FilterL2GasParamsUpdated and is used to iterate over the raw logs and unpacked data for L2GasParamsUpdated events raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainL2GasParamsUpdatedIterator struct {
	Event *CanonicalTransactionChainL2GasParamsUpdated // Event containing the contract specifics and raw log

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
func (it *CanonicalTransactionChainL2GasParamsUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CanonicalTransactionChainL2GasParamsUpdated)
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
		it.Event = new(CanonicalTransactionChainL2GasParamsUpdated)
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
func (it *CanonicalTransactionChainL2GasParamsUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CanonicalTransactionChainL2GasParamsUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CanonicalTransactionChainL2GasParamsUpdated represents a L2GasParamsUpdated event raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainL2GasParamsUpdated struct {
	L2GasDiscountDivisor *big.Int
	EnqueueGasCost       *big.Int
	EnqueueL2GasPrepaid  *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterL2GasParamsUpdated is a free log retrieval operation binding the contract event 0xc6ed75e96b8b18b71edc1a6e82a9d677f8268c774a262c624eeb2cf0a8b3e07e.
//
// Solidity: event L2GasParamsUpdated(uint256 l2GasDiscountDivisor, uint256 enqueueGasCost, uint256 enqueueL2GasPrepaid)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) FilterL2GasParamsUpdated(opts *bind.FilterOpts) (*CanonicalTransactionChainL2GasParamsUpdatedIterator, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.FilterLogs(opts, "L2GasParamsUpdated")
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainL2GasParamsUpdatedIterator{contract: _CanonicalTransactionChain.contract, event: "L2GasParamsUpdated", logs: logs, sub: sub}, nil
}

// WatchL2GasParamsUpdated is a free log subscription operation binding the contract event 0xc6ed75e96b8b18b71edc1a6e82a9d677f8268c774a262c624eeb2cf0a8b3e07e.
//
// Solidity: event L2GasParamsUpdated(uint256 l2GasDiscountDivisor, uint256 enqueueGasCost, uint256 enqueueL2GasPrepaid)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) WatchL2GasParamsUpdated(opts *bind.WatchOpts, sink chan<- *CanonicalTransactionChainL2GasParamsUpdated) (event.Subscription, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.WatchLogs(opts, "L2GasParamsUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CanonicalTransactionChainL2GasParamsUpdated)
				if err := _CanonicalTransactionChain.contract.UnpackLog(event, "L2GasParamsUpdated", log); err != nil {
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

// ParseL2GasParamsUpdated is a log parse operation binding the contract event 0xc6ed75e96b8b18b71edc1a6e82a9d677f8268c774a262c624eeb2cf0a8b3e07e.
//
// Solidity: event L2GasParamsUpdated(uint256 l2GasDiscountDivisor, uint256 enqueueGasCost, uint256 enqueueL2GasPrepaid)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) ParseL2GasParamsUpdated(log types.Log) (*CanonicalTransactionChainL2GasParamsUpdated, error) {
	event := new(CanonicalTransactionChainL2GasParamsUpdated)
	if err := _CanonicalTransactionChain.contract.UnpackLog(event, "L2GasParamsUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CanonicalTransactionChainQueueBatchAppendedIterator is returned from FilterQueueBatchAppended and is used to iterate over the raw logs and unpacked data for QueueBatchAppended events raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainQueueBatchAppendedIterator struct {
	Event *CanonicalTransactionChainQueueBatchAppended // Event containing the contract specifics and raw log

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
func (it *CanonicalTransactionChainQueueBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CanonicalTransactionChainQueueBatchAppended)
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
		it.Event = new(CanonicalTransactionChainQueueBatchAppended)
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
func (it *CanonicalTransactionChainQueueBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CanonicalTransactionChainQueueBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CanonicalTransactionChainQueueBatchAppended represents a QueueBatchAppended event raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainQueueBatchAppended struct {
	StartingQueueIndex *big.Int
	NumQueueElements   *big.Int
	TotalElements      *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterQueueBatchAppended is a free log retrieval operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) FilterQueueBatchAppended(opts *bind.FilterOpts) (*CanonicalTransactionChainQueueBatchAppendedIterator, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.FilterLogs(opts, "QueueBatchAppended")
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainQueueBatchAppendedIterator{contract: _CanonicalTransactionChain.contract, event: "QueueBatchAppended", logs: logs, sub: sub}, nil
}

// WatchQueueBatchAppended is a free log subscription operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) WatchQueueBatchAppended(opts *bind.WatchOpts, sink chan<- *CanonicalTransactionChainQueueBatchAppended) (event.Subscription, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.WatchLogs(opts, "QueueBatchAppended")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CanonicalTransactionChainQueueBatchAppended)
				if err := _CanonicalTransactionChain.contract.UnpackLog(event, "QueueBatchAppended", log); err != nil {
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

// ParseQueueBatchAppended is a log parse operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) ParseQueueBatchAppended(log types.Log) (*CanonicalTransactionChainQueueBatchAppended, error) {
	event := new(CanonicalTransactionChainQueueBatchAppended)
	if err := _CanonicalTransactionChain.contract.UnpackLog(event, "QueueBatchAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CanonicalTransactionChainSequencerBatchAppendedIterator is returned from FilterSequencerBatchAppended and is used to iterate over the raw logs and unpacked data for SequencerBatchAppended events raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainSequencerBatchAppendedIterator struct {
	Event *CanonicalTransactionChainSequencerBatchAppended // Event containing the contract specifics and raw log

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
func (it *CanonicalTransactionChainSequencerBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CanonicalTransactionChainSequencerBatchAppended)
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
		it.Event = new(CanonicalTransactionChainSequencerBatchAppended)
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
func (it *CanonicalTransactionChainSequencerBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CanonicalTransactionChainSequencerBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CanonicalTransactionChainSequencerBatchAppended represents a SequencerBatchAppended event raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainSequencerBatchAppended struct {
	StartingQueueIndex *big.Int
	NumQueueElements   *big.Int
	TotalElements      *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterSequencerBatchAppended is a free log retrieval operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) FilterSequencerBatchAppended(opts *bind.FilterOpts) (*CanonicalTransactionChainSequencerBatchAppendedIterator, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.FilterLogs(opts, "SequencerBatchAppended")
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainSequencerBatchAppendedIterator{contract: _CanonicalTransactionChain.contract, event: "SequencerBatchAppended", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchAppended is a free log subscription operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) WatchSequencerBatchAppended(opts *bind.WatchOpts, sink chan<- *CanonicalTransactionChainSequencerBatchAppended) (event.Subscription, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.WatchLogs(opts, "SequencerBatchAppended")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CanonicalTransactionChainSequencerBatchAppended)
				if err := _CanonicalTransactionChain.contract.UnpackLog(event, "SequencerBatchAppended", log); err != nil {
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

// ParseSequencerBatchAppended is a log parse operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) ParseSequencerBatchAppended(log types.Log) (*CanonicalTransactionChainSequencerBatchAppended, error) {
	event := new(CanonicalTransactionChainSequencerBatchAppended)
	if err := _CanonicalTransactionChain.contract.UnpackLog(event, "SequencerBatchAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CanonicalTransactionChainTransactionBatchAppendedIterator is returned from FilterTransactionBatchAppended and is used to iterate over the raw logs and unpacked data for TransactionBatchAppended events raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainTransactionBatchAppendedIterator struct {
	Event *CanonicalTransactionChainTransactionBatchAppended // Event containing the contract specifics and raw log

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
func (it *CanonicalTransactionChainTransactionBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CanonicalTransactionChainTransactionBatchAppended)
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
		it.Event = new(CanonicalTransactionChainTransactionBatchAppended)
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
func (it *CanonicalTransactionChainTransactionBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CanonicalTransactionChainTransactionBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CanonicalTransactionChainTransactionBatchAppended represents a TransactionBatchAppended event raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainTransactionBatchAppended struct {
	BatchIndex        *big.Int
	BatchRoot         [32]byte
	BatchSize         *big.Int
	PrevTotalElements *big.Int
	ExtraData         []byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterTransactionBatchAppended is a free log retrieval operation binding the contract event 0x127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e.
//
// Solidity: event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) FilterTransactionBatchAppended(opts *bind.FilterOpts, _batchIndex []*big.Int) (*CanonicalTransactionChainTransactionBatchAppendedIterator, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _CanonicalTransactionChain.contract.FilterLogs(opts, "TransactionBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainTransactionBatchAppendedIterator{contract: _CanonicalTransactionChain.contract, event: "TransactionBatchAppended", logs: logs, sub: sub}, nil
}

// WatchTransactionBatchAppended is a free log subscription operation binding the contract event 0x127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e.
//
// Solidity: event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) WatchTransactionBatchAppended(opts *bind.WatchOpts, sink chan<- *CanonicalTransactionChainTransactionBatchAppended, _batchIndex []*big.Int) (event.Subscription, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _CanonicalTransactionChain.contract.WatchLogs(opts, "TransactionBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CanonicalTransactionChainTransactionBatchAppended)
				if err := _CanonicalTransactionChain.contract.UnpackLog(event, "TransactionBatchAppended", log); err != nil {
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

// ParseTransactionBatchAppended is a log parse operation binding the contract event 0x127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e.
//
// Solidity: event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) ParseTransactionBatchAppended(log types.Log) (*CanonicalTransactionChainTransactionBatchAppended, error) {
	event := new(CanonicalTransactionChainTransactionBatchAppended)
	if err := _CanonicalTransactionChain.contract.UnpackLog(event, "TransactionBatchAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CanonicalTransactionChainTransactionEnqueuedIterator is returned from FilterTransactionEnqueued and is used to iterate over the raw logs and unpacked data for TransactionEnqueued events raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainTransactionEnqueuedIterator struct {
	Event *CanonicalTransactionChainTransactionEnqueued // Event containing the contract specifics and raw log

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
func (it *CanonicalTransactionChainTransactionEnqueuedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CanonicalTransactionChainTransactionEnqueued)
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
		it.Event = new(CanonicalTransactionChainTransactionEnqueued)
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
func (it *CanonicalTransactionChainTransactionEnqueuedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CanonicalTransactionChainTransactionEnqueuedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CanonicalTransactionChainTransactionEnqueued represents a TransactionEnqueued event raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainTransactionEnqueued struct {
	L1TxOrigin common.Address
	Target     common.Address
	GasLimit   *big.Int
	Data       []byte
	QueueIndex *big.Int
	Timestamp  *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTransactionEnqueued is a free log retrieval operation binding the contract event 0x4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb5.
//
// Solidity: event TransactionEnqueued(address indexed _l1TxOrigin, address indexed _target, uint256 _gasLimit, bytes _data, uint256 indexed _queueIndex, uint256 _timestamp)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) FilterTransactionEnqueued(opts *bind.FilterOpts, _l1TxOrigin []common.Address, _target []common.Address, _queueIndex []*big.Int) (*CanonicalTransactionChainTransactionEnqueuedIterator, error) {

	var _l1TxOriginRule []interface{}
	for _, _l1TxOriginItem := range _l1TxOrigin {
		_l1TxOriginRule = append(_l1TxOriginRule, _l1TxOriginItem)
	}
	var _targetRule []interface{}
	for _, _targetItem := range _target {
		_targetRule = append(_targetRule, _targetItem)
	}

	var _queueIndexRule []interface{}
	for _, _queueIndexItem := range _queueIndex {
		_queueIndexRule = append(_queueIndexRule, _queueIndexItem)
	}

	logs, sub, err := _CanonicalTransactionChain.contract.FilterLogs(opts, "TransactionEnqueued", _l1TxOriginRule, _targetRule, _queueIndexRule)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainTransactionEnqueuedIterator{contract: _CanonicalTransactionChain.contract, event: "TransactionEnqueued", logs: logs, sub: sub}, nil
}

// WatchTransactionEnqueued is a free log subscription operation binding the contract event 0x4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb5.
//
// Solidity: event TransactionEnqueued(address indexed _l1TxOrigin, address indexed _target, uint256 _gasLimit, bytes _data, uint256 indexed _queueIndex, uint256 _timestamp)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) WatchTransactionEnqueued(opts *bind.WatchOpts, sink chan<- *CanonicalTransactionChainTransactionEnqueued, _l1TxOrigin []common.Address, _target []common.Address, _queueIndex []*big.Int) (event.Subscription, error) {

	var _l1TxOriginRule []interface{}
	for _, _l1TxOriginItem := range _l1TxOrigin {
		_l1TxOriginRule = append(_l1TxOriginRule, _l1TxOriginItem)
	}
	var _targetRule []interface{}
	for _, _targetItem := range _target {
		_targetRule = append(_targetRule, _targetItem)
	}

	var _queueIndexRule []interface{}
	for _, _queueIndexItem := range _queueIndex {
		_queueIndexRule = append(_queueIndexRule, _queueIndexItem)
	}

	logs, sub, err := _CanonicalTransactionChain.contract.WatchLogs(opts, "TransactionEnqueued", _l1TxOriginRule, _targetRule, _queueIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CanonicalTransactionChainTransactionEnqueued)
				if err := _CanonicalTransactionChain.contract.UnpackLog(event, "TransactionEnqueued", log); err != nil {
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

// ParseTransactionEnqueued is a log parse operation binding the contract event 0x4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb5.
//
// Solidity: event TransactionEnqueued(address indexed _l1TxOrigin, address indexed _target, uint256 _gasLimit, bytes _data, uint256 indexed _queueIndex, uint256 _timestamp)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) ParseTransactionEnqueued(log types.Log) (*CanonicalTransactionChainTransactionEnqueued, error) {
	event := new(CanonicalTransactionChainTransactionEnqueued)
	if err := _CanonicalTransactionChain.contract.UnpackLog(event, "TransactionEnqueued", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
