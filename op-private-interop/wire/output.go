package wire

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// OutputSelector identifies a logless consensus-data record. Its calldata, not
// contract storage or an execution receipt, is the commitment. Only a record in
// an admitted, canonical sequencer block is a checkpoint.
var OutputSelector = crypto.Keccak256([]byte("recordOutput(bytes32)"))[:4]

func EncodeOutput(root common.Hash) []byte {
	out := make([]byte, 4+common.HashLength)
	copy(out, OutputSelector)
	copy(out[4:], root[:])
	return out
}

func DecodeOutput(data []byte) (common.Hash, error) {
	if len(data) != 4+common.HashLength || !bytes.Equal(data[:4], OutputSelector) {
		return common.Hash{}, fmt.Errorf("noncanonical private output record")
	}
	root := common.BytesToHash(data[4:])
	if root == (common.Hash{}) {
		return common.Hash{}, fmt.Errorf("empty private output commitment")
	}
	return root, nil
}
