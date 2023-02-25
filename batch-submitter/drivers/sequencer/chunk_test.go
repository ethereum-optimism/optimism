package sequencer_test

import (
	"bytes"
	"testing"

	"github.com/ethereum-optimism/optimism/batch-submitter/drivers/sequencer"
	"github.com/stretchr/testify/assert"
)

func TestChunkifier(t *testing.T) {

	t.Run("chunky", func(t *testing.T) {

		// test if CreateChunk works

		// create populated arbitrary byte array of length 2000
		data := bytes.Repeat([]byte{0x01}, 2000)
		result := sequencer.CreateChunks(520, data)

		assert.Equal(t, len(result), 4)

		// assert blockCount is of type int
		//require.IsType(t, int(1), int(blockCount))
	})

}
