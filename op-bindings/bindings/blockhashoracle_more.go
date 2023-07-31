// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const BlockHashOracleStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/dispute/BlockHashOracle.sol:BlockHashOracle\",\"label\":\"blockHashes\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_mapping(t_uint256,t_userDefinedValueType(Hash)1001)\"}],\"types\":{\"t_mapping(t_uint256,t_userDefinedValueType(Hash)1001)\":{\"encoding\":\"mapping\",\"label\":\"mapping(uint256 =\u003e Hash)\",\"numberOfBytes\":\"32\",\"key\":\"t_uint256\",\"value\":\"t_userDefinedValueType(Hash)1001\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_userDefinedValueType(Hash)1001\":{\"encoding\":\"inplace\",\"label\":\"Hash\",\"numberOfBytes\":\"32\"}}}"

var BlockHashOracleStorageLayout = new(solc.StorageLayout)

var BlockHashOracleDeployedBin = "0x608060405234801561001057600080fd5b50600436106100365760003560e01c80636057361d1461003b57806399d548aa14610050575b600080fd5b61004e610049366004610112565b610075565b005b61006361005e366004610112565b6100c4565b60405190815260200160405180910390f35b804060008190036100b2576040517fd82756d800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60009182526020829052604090912055565b6000818152602081905260408120549081900361010d576040517f37cf270500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b919050565b60006020828403121561012457600080fd5b503591905056fea164736f6c634300080f000a"

func init() {
	if err := json.Unmarshal([]byte(BlockHashOracleStorageLayoutJSON), BlockHashOracleStorageLayout); err != nil {
		panic(err)
	}

	layouts["BlockHashOracle"] = BlockHashOracleStorageLayout
	deployedBytecodes["BlockHashOracle"] = BlockHashOracleDeployedBin
}
