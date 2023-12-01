// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const ISemverStorageLayoutJSON = "{\"storage\":null,\"types\":{}}"

var ISemverStorageLayout = new(solc.StorageLayout)

var ISemverDeployedBin = "0x"


func init() {
	if err := json.Unmarshal([]byte(ISemverStorageLayoutJSON), ISemverStorageLayout); err != nil {
		panic(err)
	}

	layouts["ISemver"] = ISemverStorageLayout
	deployedBytecodes["ISemver"] = ISemverDeployedBin
	immutableReferences["ISemver"] = false
}
