// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const StandardBridgeStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/universal/StandardBridge.sol:StandardBridge\",\"label\":\"_initialized\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_uint8\"},{\"astId\":1001,\"contract\":\"src/universal/StandardBridge.sol:StandardBridge\",\"label\":\"_initializing\",\"offset\":1,\"slot\":\"0\",\"type\":\"t_bool\"},{\"astId\":1002,\"contract\":\"src/universal/StandardBridge.sol:StandardBridge\",\"label\":\"spacer_0_2_30\",\"offset\":2,\"slot\":\"0\",\"type\":\"t_bytes30\"},{\"astId\":1003,\"contract\":\"src/universal/StandardBridge.sol:StandardBridge\",\"label\":\"spacer_1_0_20\",\"offset\":0,\"slot\":\"1\",\"type\":\"t_address\"},{\"astId\":1004,\"contract\":\"src/universal/StandardBridge.sol:StandardBridge\",\"label\":\"deposits\",\"offset\":0,\"slot\":\"2\",\"type\":\"t_mapping(t_address,t_mapping(t_address,t_uint256))\"},{\"astId\":1005,\"contract\":\"src/universal/StandardBridge.sol:StandardBridge\",\"label\":\"messenger\",\"offset\":0,\"slot\":\"3\",\"type\":\"t_contract(CrossDomainMessenger)1008\"},{\"astId\":1006,\"contract\":\"src/universal/StandardBridge.sol:StandardBridge\",\"label\":\"otherBridge\",\"offset\":0,\"slot\":\"4\",\"type\":\"t_contract(StandardBridge)1009\"},{\"astId\":1007,\"contract\":\"src/universal/StandardBridge.sol:StandardBridge\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"5\",\"type\":\"t_array(t_uint256)45_storage\"}],\"types\":{\"t_address\":{\"encoding\":\"inplace\",\"label\":\"address\",\"numberOfBytes\":\"20\"},\"t_array(t_uint256)45_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[45]\",\"numberOfBytes\":\"1440\",\"base\":\"t_uint256\"},\"t_bool\":{\"encoding\":\"inplace\",\"label\":\"bool\",\"numberOfBytes\":\"1\"},\"t_bytes30\":{\"encoding\":\"inplace\",\"label\":\"bytes30\",\"numberOfBytes\":\"30\"},\"t_contract(CrossDomainMessenger)1008\":{\"encoding\":\"inplace\",\"label\":\"contract CrossDomainMessenger\",\"numberOfBytes\":\"20\"},\"t_contract(StandardBridge)1009\":{\"encoding\":\"inplace\",\"label\":\"contract StandardBridge\",\"numberOfBytes\":\"20\"},\"t_mapping(t_address,t_mapping(t_address,t_uint256))\":{\"encoding\":\"mapping\",\"label\":\"mapping(address =\u003e mapping(address =\u003e uint256))\",\"numberOfBytes\":\"32\",\"key\":\"t_address\",\"value\":\"t_mapping(t_address,t_uint256)\"},\"t_mapping(t_address,t_uint256)\":{\"encoding\":\"mapping\",\"label\":\"mapping(address =\u003e uint256)\",\"numberOfBytes\":\"32\",\"key\":\"t_address\",\"value\":\"t_uint256\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_uint8\":{\"encoding\":\"inplace\",\"label\":\"uint8\",\"numberOfBytes\":\"1\"}}}"

var StandardBridgeStorageLayout = new(solc.StorageLayout)

var StandardBridgeDeployedBin = "0x"


func init() {
	if err := json.Unmarshal([]byte(StandardBridgeStorageLayoutJSON), StandardBridgeStorageLayout); err != nil {
		panic(err)
	}

	layouts["StandardBridge"] = StandardBridgeStorageLayout
	deployedBytecodes["StandardBridge"] = StandardBridgeDeployedBin
	immutableReferences["StandardBridge"] = false
}
