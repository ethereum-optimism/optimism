package bindings

import (
	"fmt"

	"github.com/bobanetwork/boba/boba-bindings/solc"
	"github.com/ledgerwatch/erigon/common"
)

var layouts = make(map[string]*solc.StorageLayout)

var deployedBytecodes = make(map[string]string)

var specialContractNames = map[string]string{
	"BobaL2": "L2GovernanceERC20",
}

func GetStorageLayout(name string) (*solc.StorageLayout, error) {
	if specialName, ok := specialContractNames[name]; ok {
		name = specialName
	}
	layout := layouts[name]
	if layout == nil {
		return nil, fmt.Errorf("%s: storage layout not found", name)
	}
	return layout, nil
}

func GetDeployedBytecode(name string) ([]byte, error) {
	if specialName, ok := specialContractNames[name]; ok {
		name = specialName
	}
	bc := deployedBytecodes[name]
	if bc == "" {
		return nil, fmt.Errorf("%s: deployed bytecode not found", name)
	}

	return common.FromHex(bc), nil
}
