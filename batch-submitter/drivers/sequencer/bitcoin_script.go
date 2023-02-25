package sequencer

import (
	"github.com/btcsuite/btcd/txscript"
)

func CreateBitcoinScript(calldata [][]byte) ([]byte, error) {
	tapscript := txscript.NewScriptBuilder()
	// @DEV should we have a prefix telling what it is?

	tapscript.AddOp(txscript.OP_FALSE)
	tapscript.AddOp(txscript.OP_IF)
	// for loop appending elements of calldata to tapscript with AddData
	for _, calldata := range calldata {
		tapscript.AddData(calldata)
	}

	tapscript.AddOp(txscript.OP_ENDIF)

	return tapscript.Script()
}
