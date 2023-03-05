package sequencer

import (
	"github.com/btcsuite/btcd/txscript"
)

func CreateBitcoinScript(calldataChunks [][]byte) ([]byte, error) {
	tapscript := txscript.NewScriptBuilder()
	tapscript.AddOp(txscript.OP_TRUE)
	tapscript.AddOp(txscript.OP_FALSE)
	tapscript.AddOp(txscript.OP_IF)
	// for loop appending elements of calldata to tapscript with AddData
	for _, calldata := range calldataChunks {
		tapscript.AddData(calldata)
	}

	tapscript.AddOp(txscript.OP_ENDIF)

	return tapscript.Script()
}
