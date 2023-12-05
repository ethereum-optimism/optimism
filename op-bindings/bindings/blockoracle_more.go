// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const BlockOracleStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/dispute/BlockOracle.sol:BlockOracle\",\"label\":\"blocks\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_mapping(t_uint256,t_struct(BlockInfo)1001_storage)\"}],\"types\":{\"t_mapping(t_uint256,t_struct(BlockInfo)1001_storage)\":{\"encoding\":\"mapping\",\"label\":\"mapping(uint256 =\u003e struct BlockOracle.BlockInfo)\",\"numberOfBytes\":\"32\",\"key\":\"t_uint256\",\"value\":\"t_struct(BlockInfo)1001_storage\"},\"t_struct(BlockInfo)1001_storage\":{\"encoding\":\"inplace\",\"label\":\"struct BlockOracle.BlockInfo\",\"numberOfBytes\":\"64\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_userDefinedValueType(Hash)1002\":{\"encoding\":\"inplace\",\"label\":\"Hash\",\"numberOfBytes\":\"32\"},\"t_userDefinedValueType(Timestamp)1003\":{\"encoding\":\"inplace\",\"label\":\"Timestamp\",\"numberOfBytes\":\"8\"}}}"

var BlockOracleStorageLayout = new(solc.StorageLayout)

var BlockOracleDeployedBin = "0x608060405234801561001057600080fd5b50600436106100415760003560e01c806354fd4d501461004657806399d548aa14610098578063c2c4c5c1146100d0575b600080fd5b6100826040518060400160405280600581526020017f302e302e3100000000000000000000000000000000000000000000000000000081525081565b60405161008f9190610210565b60405180910390f35b6100ab6100a6366004610283565b6100e6565b604080518251815260209283015167ffffffffffffffff16928101929092520161008f565b6100d8610165565b60405190815260200161008f565b604080518082018252600080825260209182018190528381528082528281208351808501909452805480855260019091015467ffffffffffffffff169284019290925203610160576040517f37cf270500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b919050565b600061017260014361029c565b60408051808201825282408082524267ffffffffffffffff81811660208086018281526000898152918290528782209651875551600190960180547fffffffffffffffffffffffffffffffffffffffffffffffff000000000000000016969093169590951790915593519495509093909291849186917fb67ff58b33060fd371a35ae2d9f1c3cdaec9b8197969f6efe2594a1ff4ba68c691a4505090565b600060208083528351808285015260005b8181101561023d57858101830151858201604001528201610221565b8181111561024f576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b60006020828403121561029557600080fd5b5035919050565b6000828210156102d5577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b50039056fea164736f6c634300080f000a"


func init() {
	if err := json.Unmarshal([]byte(BlockOracleStorageLayoutJSON), BlockOracleStorageLayout); err != nil {
		panic(err)
	}

	layouts["BlockOracle"] = BlockOracleStorageLayout
	deployedBytecodes["BlockOracle"] = BlockOracleDeployedBin
	immutableReferences["BlockOracle"] = false
}
