// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const BlockOracleStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/dispute/BlockOracle.sol:BlockOracle\",\"label\":\"blocks\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_mapping(t_uint256,t_struct(BlockInfo)1001_storage)\"}],\"types\":{\"t_mapping(t_uint256,t_struct(BlockInfo)1001_storage)\":{\"encoding\":\"mapping\",\"label\":\"mapping(uint256 =\u003e struct BlockOracle.BlockInfo)\",\"numberOfBytes\":\"32\",\"key\":\"t_uint256\",\"value\":\"t_struct(BlockInfo)1001_storage\"},\"t_struct(BlockInfo)1001_storage\":{\"encoding\":\"inplace\",\"label\":\"struct BlockOracle.BlockInfo\",\"numberOfBytes\":\"64\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_userDefinedValueType(Hash)1002\":{\"encoding\":\"inplace\",\"label\":\"Hash\",\"numberOfBytes\":\"32\"},\"t_userDefinedValueType(Timestamp)1003\":{\"encoding\":\"inplace\",\"label\":\"Timestamp\",\"numberOfBytes\":\"8\"}}}"

var BlockOracleStorageLayout = new(solc.StorageLayout)

var BlockOracleDeployedBin = "0x608060405234801561001057600080fd5b50600436106100365760003560e01c806399d548aa1461003b578063c2c4c5c114610078575b600080fd5b61004e610049366004610184565b61008e565b604080518251815260209283015167ffffffffffffffff1692810192909252015b60405180910390f35b61008061010d565b60405190815260200161006f565b604080518082018252600080825260209182018190528381528082528281208351808501909452805480855260019091015467ffffffffffffffff169284019290925203610108576040517f37cf270500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b919050565b600061011a60014361019d565b604080518082018252824081524267ffffffffffffffff908116602080840191825260008681529081905293909320915182559151600190910180547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001691909216179055919050565b60006020828403121561019657600080fd5b5035919050565b6000828210156101d6577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b50039056fea164736f6c634300080f000a"

func init() {
	if err := json.Unmarshal([]byte(BlockOracleStorageLayoutJSON), BlockOracleStorageLayout); err != nil {
		panic(err)
	}

	layouts["BlockOracle"] = BlockOracleStorageLayout
	deployedBytecodes["BlockOracle"] = BlockOracleDeployedBin
}
