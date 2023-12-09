// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const LegacyMessagePasserStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/legacy/LegacyMessagePasser.sol:LegacyMessagePasser\",\"label\":\"sentMessages\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_mapping(t_bytes32,t_bool)\"}],\"types\":{\"t_bool\":{\"encoding\":\"inplace\",\"label\":\"bool\",\"numberOfBytes\":\"1\"},\"t_bytes32\":{\"encoding\":\"inplace\",\"label\":\"bytes32\",\"numberOfBytes\":\"32\"},\"t_mapping(t_bytes32,t_bool)\":{\"encoding\":\"mapping\",\"label\":\"mapping(bytes32 =\u003e bool)\",\"numberOfBytes\":\"32\",\"key\":\"t_bytes32\",\"value\":\"t_bool\"}}}"

var LegacyMessagePasserStorageLayout = new(solc.StorageLayout)

var LegacyMessagePasserDeployedBin = "0x608060405234801561001057600080fd5b50600436106100415760003560e01c806354fd4d501461004657806382e3702d14610098578063cafa81dc146100cb575b600080fd5b6100826040518060400160405280600581526020017f312e312e3000000000000000000000000000000000000000000000000000000081525081565b60405161008f919061019b565b60405180910390f35b6100bb6100a63660046101ec565b60006020819052908152604090205460ff1681565b604051901515815260200161008f565b6100de6100d9366004610234565b6100e0565b005b600160008083336040516020016100f8929190610303565b604080518083037fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe001815291815281516020928301208352908201929092520160002080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001691151591909117905550565b60005b8381101561018657818101518382015260200161016e565b83811115610195576000848401525b50505050565b60208152600082518060208401526101ba81604085016020870161016b565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b6000602082840312156101fe57600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60006020828403121561024657600080fd5b813567ffffffffffffffff8082111561025e57600080fd5b818401915084601f83011261027257600080fd5b81358181111561028457610284610205565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f011681019083821181831017156102ca576102ca610205565b816040528281528760208487010111156102e357600080fd5b826020860160208301376000928101602001929092525095945050505050565b6000835161031581846020880161016b565b60609390931b7fffffffffffffffffffffffffffffffffffffffff00000000000000000000000016919092019081526014019291505056fea164736f6c634300080f000a"


func init() {
	if err := json.Unmarshal([]byte(LegacyMessagePasserStorageLayoutJSON), LegacyMessagePasserStorageLayout); err != nil {
		panic(err)
	}

	layouts["LegacyMessagePasser"] = LegacyMessagePasserStorageLayout
	deployedBytecodes["LegacyMessagePasser"] = LegacyMessagePasserDeployedBin
	immutableReferences["LegacyMessagePasser"] = false
}
