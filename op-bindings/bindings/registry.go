package bindings

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
	"github.com/ethereum/go-ethereum/common"
)

// layouts respresents the set of storage layouts. It is populated in an init function.
var layouts = make(map[string]*solc.StorageLayout)

// deployedBytecodes represents the set of deployed bytecodes. It is populated
// in an init function.
var deployedBytecodes = make(map[string]string)

// GetStorageLayout returns the storage layout of a contract by name.
func GetStorageLayout(name string) (*solc.StorageLayout, error) {
	layout := layouts[name]
	if layout == nil {
		return nil, fmt.Errorf("%s: storage layout not found", name)
	}
	return layout, nil
}

// GetDeployedBytecode returns the deployed bytecode of a contract by name.
func GetDeployedBytecode(name string) ([]byte, error) {
	bc := deployedBytecodes[name]
	if bc == "" {
		return nil, fmt.Errorf("%s: deployed bytecode not found", name)
	}

	if !isHex(bc) {
		return nil, fmt.Errorf("%s: invalid deployed bytecode", name)
	}

	return common.FromHex(bc), nil
}

// isHexCharacter returns bool of c being a valid hexadecimal.
func isHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// isHex validates whether each byte is valid hexadecimal string.
func isHex(str string) bool {
	if len(str)%2 != 0 {
		return false
	}
	str = strings.TrimPrefix(str, "0x")

	for _, c := range []byte(str) {
		if !isHexCharacter(c) {
			return false
		}
	}
	return true
}
