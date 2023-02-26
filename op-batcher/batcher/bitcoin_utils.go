package batcher

import (
	"math"

	"github.com/btcsuite/btcd/txscript"
)

func CreateChunks(chunkSize uint64, calldata []byte) [][]byte {
	chunkArray := make([][]byte, 0)
	for i := uint64(0); i < uint64(len(calldata)); i += chunkSize {
		chunk := make([]byte, 0)
		index := math.Min(float64(len(calldata)), float64(i+chunkSize))
		calldataToAppend := calldata[i:int(index)]
		chunk = append(chunk, calldataToAppend...)
		chunkArray = append(chunkArray, chunk)
	}
	return chunkArray
}

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
