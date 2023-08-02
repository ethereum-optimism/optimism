// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const BlockOracleStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/dispute/BlockOracle.sol:BlockOracle\",\"label\":\"blockHashes\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_mapping(t_uint256,t_struct(BlockInfo)1001_storage)\"}],\"types\":{\"t_mapping(t_uint256,t_struct(BlockInfo)1001_storage)\":{\"encoding\":\"mapping\",\"label\":\"mapping(uint256 =\u003e struct BlockOracle.BlockInfo)\",\"numberOfBytes\":\"32\",\"key\":\"t_uint256\",\"value\":\"t_struct(BlockInfo)1001_storage\"},\"t_struct(BlockInfo)1001_storage\":{\"encoding\":\"inplace\",\"label\":\"struct BlockOracle.BlockInfo\",\"numberOfBytes\":\"64\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_userDefinedValueType(Hash)1002\":{\"encoding\":\"inplace\",\"label\":\"Hash\",\"numberOfBytes\":\"32\"},\"t_userDefinedValueType(Timestamp)1003\":{\"encoding\":\"inplace\",\"label\":\"Timestamp\",\"numberOfBytes\":\"8\"}}}"

var BlockOracleStorageLayout = new(solc.StorageLayout)

var BlockOracleDeployedBin = "0x608060405234801561001057600080fd5b50600436106100365760003560e01c80636057361d1461003b57806399d548aa14610050575b600080fd5b61004e6100493660046101d0565b61008c565b005b61006361005e3660046101d0565b610151565b604080518251815260209283015167ffffffffffffffff16928101929092520160405180910390f35b804060008190036100c9576040517fd82756d800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006100d58343610218565b6100e090600d61022f565b6100ea9042610218565b60408051808201825293845267ffffffffffffffff918216602080860191825260009687528690529420925183559251600190920180547fffffffffffffffffffffffffffffffffffffffffffffffff000000000000000016929093169190911790915550565b604080518082018252600080825260209182018190528381528082528281208351808501909452805480855260019091015467ffffffffffffffff1692840192909252036101cb576040517f37cf270500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b919050565b6000602082840312156101e257600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008282101561022a5761022a6101e9565b500390565b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0483118215151615610267576102676101e9565b50029056fea164736f6c634300080f000a"

func init() {
	if err := json.Unmarshal([]byte(BlockOracleStorageLayoutJSON), BlockOracleStorageLayout); err != nil {
		panic(err)
	}

	layouts["BlockOracle"] = BlockOracleStorageLayout
	deployedBytecodes["BlockOracle"] = BlockOracleDeployedBin
}
