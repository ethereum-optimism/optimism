// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const ERC20StorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"node_modules/@rari-capital/solmate/src/tokens/ERC20.sol:ERC20\",\"label\":\"name\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_string_storage\"},{\"astId\":1001,\"contract\":\"node_modules/@rari-capital/solmate/src/tokens/ERC20.sol:ERC20\",\"label\":\"symbol\",\"offset\":0,\"slot\":\"1\",\"type\":\"t_string_storage\"},{\"astId\":1002,\"contract\":\"node_modules/@rari-capital/solmate/src/tokens/ERC20.sol:ERC20\",\"label\":\"totalSupply\",\"offset\":0,\"slot\":\"2\",\"type\":\"t_uint256\"},{\"astId\":1003,\"contract\":\"node_modules/@rari-capital/solmate/src/tokens/ERC20.sol:ERC20\",\"label\":\"balanceOf\",\"offset\":0,\"slot\":\"3\",\"type\":\"t_mapping(t_address,t_uint256)\"},{\"astId\":1004,\"contract\":\"node_modules/@rari-capital/solmate/src/tokens/ERC20.sol:ERC20\",\"label\":\"allowance\",\"offset\":0,\"slot\":\"4\",\"type\":\"t_mapping(t_address,t_mapping(t_address,t_uint256))\"},{\"astId\":1005,\"contract\":\"node_modules/@rari-capital/solmate/src/tokens/ERC20.sol:ERC20\",\"label\":\"nonces\",\"offset\":0,\"slot\":\"5\",\"type\":\"t_mapping(t_address,t_uint256)\"}],\"types\":{\"t_address\":{\"encoding\":\"inplace\",\"label\":\"address\",\"numberOfBytes\":\"20\"},\"t_mapping(t_address,t_mapping(t_address,t_uint256))\":{\"encoding\":\"mapping\",\"label\":\"mapping(address =\u003e mapping(address =\u003e uint256))\",\"numberOfBytes\":\"32\",\"key\":\"t_address\",\"value\":\"t_mapping(t_address,t_uint256)\"},\"t_mapping(t_address,t_uint256)\":{\"encoding\":\"mapping\",\"label\":\"mapping(address =\u003e uint256)\",\"numberOfBytes\":\"32\",\"key\":\"t_address\",\"value\":\"t_uint256\"},\"t_string_storage\":{\"encoding\":\"bytes\",\"label\":\"string\",\"numberOfBytes\":\"32\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"}}}"

var ERC20StorageLayout = new(solc.StorageLayout)

var ERC20DeployedBin = "0x"

func init() {
	if err := json.Unmarshal([]byte(ERC20StorageLayoutJSON), ERC20StorageLayout); err != nil {
		panic(err)
	}

	layouts["ERC20"] = ERC20StorageLayout
	deployedBytecodes["ERC20"] = ERC20DeployedBin
}
