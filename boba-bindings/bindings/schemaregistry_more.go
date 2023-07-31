// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/solc"
)

const SchemaRegistryStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"contracts/EAS/SchemaRegistry.sol:SchemaRegistry\",\"label\":\"_registry\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_mapping(t_bytes32,t_struct(SchemaRecord)1003_storage)\"},{\"astId\":1001,\"contract\":\"contracts/EAS/SchemaRegistry.sol:SchemaRegistry\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"1\",\"type\":\"t_array(t_uint256)49_storage\"}],\"types\":{\"t_array(t_uint256)49_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[49]\",\"numberOfBytes\":\"1568\",\"base\":\"t_uint256\"},\"t_bool\":{\"encoding\":\"inplace\",\"label\":\"bool\",\"numberOfBytes\":\"1\"},\"t_bytes32\":{\"encoding\":\"inplace\",\"label\":\"bytes32\",\"numberOfBytes\":\"32\"},\"t_contract(ISchemaResolver)1002\":{\"encoding\":\"inplace\",\"label\":\"contract ISchemaResolver\",\"numberOfBytes\":\"20\"},\"t_mapping(t_bytes32,t_struct(SchemaRecord)1003_storage)\":{\"encoding\":\"mapping\",\"label\":\"mapping(bytes32 =\u003e struct SchemaRecord)\",\"numberOfBytes\":\"32\",\"key\":\"t_bytes32\",\"value\":\"t_struct(SchemaRecord)1003_storage\"},\"t_string_storage\":{\"encoding\":\"bytes\",\"label\":\"string\",\"numberOfBytes\":\"32\"},\"t_struct(SchemaRecord)1003_storage\":{\"encoding\":\"inplace\",\"label\":\"struct SchemaRecord\",\"numberOfBytes\":\"96\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"}}}"

var SchemaRegistryStorageLayout = new(solc.StorageLayout)

var SchemaRegistryDeployedBin = "0x608060405234801561001057600080fd5b50600436106100415760003560e01c806354fd4d501461004657806360d7a27814610064578063a2ea7c6e14610085575b600080fd5b61004e6100a5565b60405161005b9190610604565b60405180910390f35b61007761007236600461061e565b610148565b60405190815260200161005b565b6100986100933660046106d0565b6102f1565b60405161005b91906106e9565b60606100d07f0000000000000000000000000000000000000000000000000000000000000000610419565b6100f97f0000000000000000000000000000000000000000000000000000000000000000610419565b6101227f0000000000000000000000000000000000000000000000000000000000000000610419565b6040516020016101349392919061073a565b604051602081830303815290604052905090565b60008060405180608001604052806000801b81526020018573ffffffffffffffffffffffffffffffffffffffff168152602001841515815260200187878080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920182905250939094525092935091506101ca905082610556565b60008181526020819052604090205490915015610213576040517f23369fa600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b80825260008181526020818152604091829020845181559084015160018201805493860151151574010000000000000000000000000000000000000000027fffffffffffffffffffffff00000000000000000000000000000000000000000090941673ffffffffffffffffffffffffffffffffffffffff9092169190911792909217909155606083015183919060028201906102af9082610881565b50506040513381528291507f7d917fcbc9a29a9705ff9936ffa599500e4fd902e4486bae317414fe967b307c9060200160405180910390a29695505050505050565b604080516080810182526000808252602082018190529181019190915260608082015260008281526020818152604091829020825160808101845281548152600182015473ffffffffffffffffffffffffffffffffffffffff8116938201939093527401000000000000000000000000000000000000000090920460ff16151592820192909252600282018054919291606084019190610390906107df565b80601f01602080910402602001604051908101604052809291908181526020018280546103bc906107df565b80156104095780601f106103de57610100808354040283529160200191610409565b820191906000526020600020905b8154815290600101906020018083116103ec57829003601f168201915b5050505050815250509050919050565b60608160000361045c57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156104865780610470816109ca565b915061047f9050600a83610a31565b9150610460565b60008167ffffffffffffffff8111156104a1576104a16107b0565b6040519080825280601f01601f1916602001820160405280156104cb576020820181803683370190505b5090505b841561054e576104e0600183610a45565b91506104ed600a86610a5e565b6104f8906030610a72565b60f81b81838151811061050d5761050d610a85565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610547600a86610a31565b94506104cf565b949350505050565b600081606001518260200151836040015160405160200161057993929190610ab4565b604051602081830303815290604052805190602001209050919050565b60005b838110156105b1578181015183820152602001610599565b50506000910152565b600081518084526105d2816020860160208601610596565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061061760208301846105ba565b9392505050565b6000806000806060858703121561063457600080fd5b843567ffffffffffffffff8082111561064c57600080fd5b818701915087601f83011261066057600080fd5b81358181111561066f57600080fd5b88602082850101111561068157600080fd5b6020928301965094505085013573ffffffffffffffffffffffffffffffffffffffff811681146106b057600080fd5b9150604085013580151581146106c557600080fd5b939692955090935050565b6000602082840312156106e257600080fd5b5035919050565b602081528151602082015273ffffffffffffffffffffffffffffffffffffffff60208301511660408201526040820151151560608201526000606083015160808084015261054e60a08401826105ba565b6000845161074c818460208901610596565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551610788816001850160208a01610596565b600192019182015283516107a3816002840160208801610596565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600181811c908216806107f357607f821691505b60208210810361082c577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b601f82111561087c57600081815260208120601f850160051c810160208610156108595750805b601f850160051c820191505b8181101561087857828155600101610865565b5050505b505050565b815167ffffffffffffffff81111561089b5761089b6107b0565b6108af816108a984546107df565b84610832565b602080601f83116001811461090257600084156108cc5750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555610878565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b8281101561094f57888601518255948401946001909101908401610930565b508582101561098b57878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036109fb576109fb61099b565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082610a4057610a40610a02565b500490565b81810381811115610a5857610a5861099b565b92915050565b600082610a6d57610a6d610a02565b500690565b80820180821115610a5857610a5861099b565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60008451610ac6818460208901610596565b60609490941b7fffffffffffffffffffffffffffffffffffffffff000000000000000000000000169190930190815290151560f81b60148201526015019291505056fea164736f6c6343000813000a"

func init() {
	if err := json.Unmarshal([]byte(SchemaRegistryStorageLayoutJSON), SchemaRegistryStorageLayout); err != nil {
		panic(err)
	}

	layouts["SchemaRegistry"] = SchemaRegistryStorageLayout
	deployedBytecodes["SchemaRegistry"] = SchemaRegistryDeployedBin
}
