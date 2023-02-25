package sequencer_test

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcd/txscript"
	"github.com/ethereum-optimism/optimism/batch-submitter/drivers/sequencer"
	"github.com/stretchr/testify/assert"
)

func TestCreateBitcoinScript(t *testing.T) {

	t.Run("bitcoin script era", func(t *testing.T) {
		data := bytes.Repeat([]byte{0x01}, 520)
		simulatedCalldata := [][]byte{data}
		simulatedCalldata = append(simulatedCalldata, data)

		result, err := sequencer.CreateBitcoinScript(simulatedCalldata)
		if err != nil {
			panic("oh no")
		}

		expectedOutput, err := txscript.NewScriptBuilder().AddOp(txscript.OP_FALSE).AddOp(txscript.OP_IF).AddData(data).AddData(data).AddOp(txscript.OP_ENDIF).Script()
		if err != nil {
			panic("oh no")
		}

		assert.Equal(t, result, expectedOutput)
	})

}
