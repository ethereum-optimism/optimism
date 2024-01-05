// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const L1BlockStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/L2/L1Block.sol:L1Block\",\"label\":\"number\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_uint64\"},{\"astId\":1001,\"contract\":\"src/L2/L1Block.sol:L1Block\",\"label\":\"timestamp\",\"offset\":8,\"slot\":\"0\",\"type\":\"t_uint64\"},{\"astId\":1002,\"contract\":\"src/L2/L1Block.sol:L1Block\",\"label\":\"basefee\",\"offset\":0,\"slot\":\"1\",\"type\":\"t_uint256\"},{\"astId\":1003,\"contract\":\"src/L2/L1Block.sol:L1Block\",\"label\":\"hash\",\"offset\":0,\"slot\":\"2\",\"type\":\"t_bytes32\"},{\"astId\":1004,\"contract\":\"src/L2/L1Block.sol:L1Block\",\"label\":\"sequenceNumber\",\"offset\":0,\"slot\":\"3\",\"type\":\"t_uint64\"},{\"astId\":1005,\"contract\":\"src/L2/L1Block.sol:L1Block\",\"label\":\"batcherHash\",\"offset\":0,\"slot\":\"4\",\"type\":\"t_bytes32\"},{\"astId\":1006,\"contract\":\"src/L2/L1Block.sol:L1Block\",\"label\":\"l1FeeOverhead\",\"offset\":0,\"slot\":\"5\",\"type\":\"t_uint256\"},{\"astId\":1007,\"contract\":\"src/L2/L1Block.sol:L1Block\",\"label\":\"l1FeeScalar\",\"offset\":0,\"slot\":\"6\",\"type\":\"t_uint256\"}],\"types\":{\"t_bytes32\":{\"encoding\":\"inplace\",\"label\":\"bytes32\",\"numberOfBytes\":\"32\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_uint64\":{\"encoding\":\"inplace\",\"label\":\"uint64\",\"numberOfBytes\":\"8\"}}}"

var L1BlockStorageLayout = new(solc.StorageLayout)

var L1BlockDeployedBin = "0x608060405234801561001057600080fd5b50600436106100c95760003560e01c80638381f58a11610081578063b80777ea1161005b578063b80777ea146101a4578063e591b282146101c4578063e81b2c6d1461020457600080fd5b80638381f58a1461017e5780638b239f73146101925780639e8c49661461019b57600080fd5b806354fd4d50116100b257806354fd4d50146100ff5780635cf249691461014857806364ca23ef1461015157600080fd5b8063015d8eb9146100ce57806309bd5a60146100e3575b600080fd5b6100e16100dc366004610369565b61020d565b005b6100ec60025481565b6040519081526020015b60405180910390f35b61013b6040518060400160405280600581526020017f312e312e3000000000000000000000000000000000000000000000000000000081525081565b6040516100f691906103db565b6100ec60015481565b6003546101659067ffffffffffffffff1681565b60405167ffffffffffffffff90911681526020016100f6565b6000546101659067ffffffffffffffff1681565b6100ec60055481565b6100ec60065481565b6000546101659068010000000000000000900467ffffffffffffffff1681565b6101df73deaddeaddeaddeaddeaddeaddeaddeaddead000181565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100f6565b6100ec60045481565b3373deaddeaddeaddeaddeaddeaddeaddeaddead0001146102b4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603b60248201527f4c31426c6f636b3a206f6e6c7920746865206465706f7369746f72206163636f60448201527f756e742063616e20736574204c3120626c6f636b2076616c7565730000000000606482015260840160405180910390fd5b6000805467ffffffffffffffff98891668010000000000000000027fffffffffffffffffffffffffffffffff00000000000000000000000000000000909116998916999099179890981790975560019490945560029290925560038054919094167fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000009190911617909255600491909155600555600655565b803567ffffffffffffffff8116811461036457600080fd5b919050565b600080600080600080600080610100898b03121561038657600080fd5b61038f8961034c565b975061039d60208a0161034c565b965060408901359550606089013594506103b960808a0161034c565b979a969950949793969560a0850135955060c08501359460e001359350915050565b600060208083528351808285015260005b81811015610408578581018301518582016040015282016103ec565b8181111561041a576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01692909201604001939250505056fea164736f6c634300080f000a"


func init() {
	if err := json.Unmarshal([]byte(L1BlockStorageLayoutJSON), L1BlockStorageLayout); err != nil {
		panic(err)
	}

	layouts["L1Block"] = L1BlockStorageLayout
	deployedBytecodes["L1Block"] = L1BlockDeployedBin
	immutableReferences["L1Block"] = false
}
