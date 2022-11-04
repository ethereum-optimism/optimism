// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const LegacyMessagePasserStorageLayoutJSON = "{\"storage\":[{\"astId\":4864,\"contract\":\"contracts/legacy/LegacyMessagePasser.sol:LegacyMessagePasser\",\"label\":\"sentMessages\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_mapping(t_bytes32,t_bool)\"}],\"types\":{\"t_bool\":{\"encoding\":\"inplace\",\"label\":\"bool\",\"numberOfBytes\":\"1\"},\"t_bytes32\":{\"encoding\":\"inplace\",\"label\":\"bytes32\",\"numberOfBytes\":\"32\"},\"t_mapping(t_bytes32,t_bool)\":{\"encoding\":\"mapping\",\"label\":\"mapping(bytes32 =\u003e bool)\",\"numberOfBytes\":\"32\",\"key\":\"t_bytes32\",\"value\":\"t_bool\"}}}"

var LegacyMessagePasserStorageLayout = new(solc.StorageLayout)

var LegacyMessagePasserDeployedBin = "0x608060405234801561001057600080fd5b50600436106100415760003560e01c806354fd4d501461004657806382e3702d14610064578063cafa81dc14610097575b600080fd5b61004e6100ac565b60405161005b9190610347565b60405180910390f35b610087610072366004610398565b60006020819052908152604090205460ff1681565b604051901515815260200161005b565b6100aa6100a53660046103e0565b61014f565b005b60606100d77f00000000000000000000000000000000000000000000000000000000000000006101da565b6101007f00000000000000000000000000000000000000000000000000000000000000006101da565b6101297f00000000000000000000000000000000000000000000000000000000000000006101da565b60405160200161013b939291906104af565b604051602081830303815290604052905090565b60016000808333604051602001610167929190610525565b604080518083037fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe001815291815281516020928301208352908201929092520160002080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001691151591909117905550565b60608160000361021d57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b811561024757806102318161059e565b91506102409050600a83610605565b9150610221565b60008167ffffffffffffffff811115610262576102626103b1565b6040519080825280601f01601f19166020018201604052801561028c576020820181803683370190505b5090505b841561030f576102a1600183610619565b91506102ae600a86610630565b6102b9906030610644565b60f81b8183815181106102ce576102ce61065c565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610308600a86610605565b9450610290565b949350505050565b60005b8381101561033257818101518382015260200161031a565b83811115610341576000848401525b50505050565b6020815260008251806020840152610366816040850160208701610317565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b6000602082840312156103aa57600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000602082840312156103f257600080fd5b813567ffffffffffffffff8082111561040a57600080fd5b818401915084601f83011261041e57600080fd5b813581811115610430576104306103b1565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908382118183101715610476576104766103b1565b8160405282815287602084870101111561048f57600080fd5b826020860160208301376000928101602001929092525095945050505050565b600084516104c1818460208901610317565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516104fd816001850160208a01610317565b60019201918201528351610518816002840160208801610317565b0160020195945050505050565b60008351610537818460208801610317565b60609390931b7fffffffffffffffffffffffffffffffffffffffff000000000000000000000000169190920190815260140192915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036105cf576105cf61056f565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082610614576106146105d6565b500490565b60008282101561062b5761062b61056f565b500390565b60008261063f5761063f6105d6565b500690565b600082198211156106575761065761056f565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a"

func init() {
	if err := json.Unmarshal([]byte(LegacyMessagePasserStorageLayoutJSON), LegacyMessagePasserStorageLayout); err != nil {
		panic(err)
	}

	layouts["LegacyMessagePasser"] = LegacyMessagePasserStorageLayout
	deployedBytecodes["LegacyMessagePasser"] = LegacyMessagePasserDeployedBin
}
