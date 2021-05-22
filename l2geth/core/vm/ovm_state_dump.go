package vm

import (
	"os"

	"github.com/ethereum/go-ethereum/common"
)

// AbiBytesTrue represents the ABI encoding of "true" as a byte slice
var AbiBytesTrue = common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000001")

// AbiBytesFalse represents the ABI encoding of "false" as a byte slice
var AbiBytesFalse = common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000")

// UsingOVM is used to enable or disable functionality necessary for the OVM.
var UsingOVM bool

func init() {
	UsingOVM = os.Getenv("USING_OVM") == "true"
}
