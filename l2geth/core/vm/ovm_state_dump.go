package vm

import (
	"os"
)

// UsingOVM is used to enable or disable functionality necessary for the OVM.
var UsingOVM bool

func init() {
	UsingOVM = os.Getenv("USING_OVM") == "true"
}
