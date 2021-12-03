package util

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi"
)

var codec abi.ABI

func init() {
	const abidata = `
	[
		{
			"type": "function",
			"name": "Error",
			"constant": true,
			"inputs": [
				{
					"name": "msg",
					"type": "string"
				}
      ],
			"outputs": []
		}
	]
`

	var err error
	codec, err = abi.JSON(strings.NewReader(abidata))
	if err != nil {
		panic(fmt.Errorf("unable to create abi decoder: %v", err))
	}
}

// EncodeSolidityError generates an abi-encoded error message.
func EncodeSolidityError(err error) ([]byte, error) {
	return codec.Pack("Error", err.Error())
}
